package transport

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"ltc-system/apps/api/internal/modules/casemgmt/app"
	"ltc-system/apps/api/internal/platform/auth"
	"ltc-system/apps/api/internal/platform/clock"
	"ltc-system/apps/api/internal/platform/httpx"
)

// PendingRelinker 讓個案新增或改名後，重新比對名稱相符的待維護司機匯報欄位；
// 未接線時傳 nil，Create／Update 會略過重新比對。
type PendingRelinker interface {
	RelinkByName(ctx context.Context, name string, actorID uuid.UUID, actorRole, ip, ua string) (int, error)
}

// CaseHandler 處理個案相關之 HTTP 請求。
type CaseHandler struct {
	masterService *app.CaseService
	relinker      PendingRelinker
}

// NewCaseHandler 建立 CaseHandler 實例。
func NewCaseHandler(
	masterService *app.CaseService,
	relinkers ...PendingRelinker,
) *CaseHandler {
	h := &CaseHandler{masterService: masterService}
	if len(relinkers) > 0 {
		h.relinker = relinkers[0]
	}
	return h
}

// relinkPendingMeta 呼叫 relinker 重新比對待維護資料；筆數為 0 時回傳 nil，
// 讓 RespondSuccess 的 meta 維持既有形狀，不多長一個恆為 0 的欄位。
func (h *CaseHandler) relinkPendingMeta(c *gin.Context, name string) any {
	if h.relinker == nil || name == "" {
		return nil
	}
	n, err := h.relinker.RelinkByName(c.Request.Context(), name, auth.GetActorID(c), auth.GetActorRole(c), c.ClientIP(), c.Request.UserAgent())
	if err != nil {
		slog.Error("pending_relink_failed", slog.String("name", name), slog.Any("error", err))
		return nil
	}
	if n == 0 {
		return nil
	}
	return gin.H{"pendingRelinked": n}
}

// List 查詢個案清單（回傳遮罩身分證）。
func (h *CaseHandler) List(c *gin.Context) {
	page, pageSize, err := httpx.ParsePagination(c)
	if err != nil {
		httpx.RespondError(c, http.StatusBadRequest, httpx.CodeValidationFailed, "分頁參數格式錯誤", nil)
		return
	}
	status := c.Query("status")
	q := c.Query("q")
	region := c.Query("region")
	// 待維護個案預設不出現在任何清單，呼叫端要明確表態才拿得到：unresolvedLink 只取待維護，
	// includePending 取全部。預設排除，避免新增呼叫端忘記帶參數就把待維護資料洩漏出去。
	unresolvedLink := httpx.QueryBool(c, "unresolvedLink")
	includePending := httpx.QueryBool(c, "includePending")
	excludePending := !unresolvedLink && !includePending

	cases, total, err := h.masterService.ListCases(c.Request.Context(), status, q, region, page, pageSize, unresolvedLink, excludePending)
	if err != nil {
		httpx.RespondError(c, http.StatusInternalServerError, httpx.CodeInternalError, "查詢個案失敗", nil)
		return
	}

	totalPages := int(total) / pageSize
	if int(total)%pageSize != 0 {
		totalPages++
	}

	httpx.RespondSuccess(c, http.StatusOK, newCaseResponses(cases), httpx.PaginationMeta{
		Page:       page,
		PageSize:   pageSize,
		Total:      total,
		TotalPages: totalPages,
	})
}

// Create 新增個案主檔。
func (h *CaseHandler) Create(c *gin.Context) {
	var req CreateCaseRequest
	if err := httpx.BindJSONStrict(c, &req); err != nil {
		httpx.RespondErrorCode(c, http.StatusBadRequest, httpx.CodeValidationFailed, err, httpx.ExtractValidationDetails(err))
		return
	}

	actorID := auth.GetActorID(c)
	actorRole := auth.GetActorRole(c)

	entity, err := h.masterService.CreateCase(
		c.Request.Context(), req.ToService(), actorID, actorRole, c.ClientIP(), c.Request.UserAgent(),
	)
	if err != nil {
		respondCreateCaseError(c, err)
		return
	}

	meta := h.relinkPendingMeta(c, entity.Name)
	httpx.RespondSuccess(c, http.StatusCreated, newCaseResponse(*entity), meta)
}

// respondCreateCaseError 依 CreateCase 已知的 sentinel 錯誤分流具體中文原因，姓名必填／
// 身分證格式錯誤回 400，身分證重複回 409；其餘未預期的加密或資料庫錯誤一律回 500，
// 不再讓所有失敗都退回同一句通用驗證訊息。
func respondCreateCaseError(c *gin.Context, err error) {
	if errors.Is(err, app.ErrCaseNameRequired) {
		httpx.RespondErrorCode(c, http.StatusBadRequest, httpx.CodeValidationFailed, err, []httpx.ErrorDetail{
			{Field: "name", Reason: "請輸入個案姓名"},
		})
		return
	}
	if errors.Is(err, app.ErrInvalidNationalIDFormat) {
		httpx.RespondErrorCode(c, http.StatusBadRequest, httpx.CodeValidationFailed, err, []httpx.ErrorDetail{
			{Field: "nationalId", Reason: "身分證字號格式錯誤"},
		})
		return
	}
	if errors.Is(err, app.ErrDuplicateNationalID) {
		httpx.RespondErrorCode(c, http.StatusConflict, httpx.CodeValidationFailed, err, []httpx.ErrorDetail{
			{Field: "nationalId", Reason: "此身分證字號已存在於其他個案"},
		})
		return
	}
	httpx.RespondErrorCode(c, http.StatusInternalServerError, httpx.CodeInternalError, err, nil)
}

// Delete 軟刪除個案並收斂其生效中排班。
func (h *CaseHandler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		httpx.RespondError(c, http.StatusBadRequest, httpx.CodeValidationFailed, "無效的個案 ID", nil)
		return
	}

	actorID := auth.GetActorID(c)
	actorRole := auth.GetActorRole(c)

	if err := h.masterService.Delete(c.Request.Context(), id, actorID, actorRole, c.ClientIP(), c.Request.UserAgent()); err != nil {
		if errors.Is(err, app.ErrCaseNotFound) {
			httpx.RespondErrorCode(c, http.StatusNotFound, httpx.CodeNotFound, err, nil)
			return
		}
		httpx.RespondErrorCode(c, http.StatusInternalServerError, httpx.CodeInternalError, err, nil)
		return
	}

	c.Status(http.StatusNoContent)
}

// CreateSchedule 建立個案排班設定與時段明細。
func (h *CaseHandler) CreateSchedule(c *gin.Context) {
	var req CreateScheduleRequest
	if err := httpx.BindJSONStrict(c, &req); err != nil {
		httpx.RespondErrorCode(c, http.StatusBadRequest, httpx.CodeValidationFailed, err, httpx.ExtractValidationDetails(err))
		return
	}

	sched, err := h.masterService.CreateCaseSchedule(c.Request.Context(), req.ToService())
	if err != nil {
		respondScheduleError(c, err)
		return
	}

	httpx.RespondSuccess(c, http.StatusCreated, newCaseScheduleResponse(*sched), nil)
}

// scheduleErrorReason 讓已知的排班驗證 sentinel 錯誤對應到欄位與可讀中文原因。
type scheduleErrorReason struct {
	err    error
	field  string
	reason string
}

// scheduleErrorReasons 列出 CreateCaseSchedule／SaveSchedule 會回傳的已知輸入錯誤，
// 逐一寫死中文說明，避免全部退回「輸入資料不符合規則」這種看不出原因的通用句。
var scheduleErrorReasons = []scheduleErrorReason{
	{app.ErrInvalidTripPattern, "tripPattern", "趟次數與時段筆數不一致，請確認出車趟數與時段設定"},
	{app.ErrLegTimesNotOrdered, "legs", "各時段出發時間須依序由早到晚遞增，請確認時間順序"},
	{app.ErrInvalidScheduleWeekday, "weekdays", "星期設定不正確，請選擇 1 到 7 之間且不重複的星期"},
	{app.ErrInvalidScheduleLegSeq, "legSeq", "時段序號設定不正確，請確認序號未重複且未超出趟次數"},
	{app.ErrInvalidScheduleDirection, "direction", "時段方向設定不正確，請選擇去程或回程"},
	{app.ErrInvalidScheduleTime, "departTime", "出發時間格式不正確，請使用 HH:MM 格式（例如 08:30）"},
	{app.ErrInvalidSchedulePrice, "unitPrice", "單價必須大於 0，請重新輸入"},
	{app.ErrInvalidScheduleDistance, "distanceKm", "距離必須大於 0，請重新輸入"},
	{app.ErrInvalidScheduleDuration, "serviceDurationMin", "服務時長必須介於 1 至 240 分鐘，請重新輸入"},
	{app.ErrInvalidScheduleDateRange, "effectiveTo", "結束日期不可早於生效日期，請確認日期區間"},
}

// respondScheduleError 依錯誤種類分流：個案不存在回 404；已知的輸入驗證錯誤回 400 並附上
// 具體中文原因；時段生效期間重疊（no_overlapping_case_schedule）回 409，避免跟一般驗證錯誤
// 混淆；外鍵失效（選到不存在的車輛/個案）回 400 並說明原因；其餘未預期的資料庫或系統錯誤
// 一律回 500，不再讓期間重疊、FK 失效、DB 故障全部偽裝成使用者輸入錯誤。
func respondScheduleError(c *gin.Context, err error) {
	if errors.Is(err, app.ErrCaseNotFound) {
		httpx.RespondErrorCode(c, http.StatusNotFound, httpx.CodeNotFound, err, nil)
		return
	}
	for _, sr := range scheduleErrorReasons {
		if errors.Is(err, sr.err) {
			httpx.RespondErrorCode(c, http.StatusBadRequest, httpx.CodeValidationFailed, err, []httpx.ErrorDetail{
				{Field: sr.field, Reason: sr.reason},
			})
			return
		}
	}
	if errors.Is(err, app.ErrScheduleOverlap) {
		httpx.RespondErrorCode(c, http.StatusConflict, httpx.CodeAssignmentOverlap, err, nil)
		return
	}
	if errors.Is(err, app.ErrScheduleInvalidReference) {
		httpx.RespondErrorCode(c, http.StatusBadRequest, httpx.CodeValidationFailed, err, []httpx.ErrorDetail{
			{Reason: "所選擇的車輛或個案資料不存在，請重新整理後再選擇"},
		})
		return
	}
	httpx.RespondErrorCode(c, http.StatusInternalServerError, httpx.CodeInternalError, err, nil)
}

// ExportProfileWorkbook 下載與個案彙整表相同格式的主檔資料；caseIds 為逗號分隔的個案 ID，省略則匯出全部個案。
func (h *CaseHandler) ExportProfileWorkbook(c *gin.Context) {
	var caseIDs []uuid.UUID
	if raw := c.Query("caseIds"); raw != "" {
		for _, idStr := range strings.Split(raw, ",") {
			id, err := uuid.Parse(strings.TrimSpace(idStr))
			if err != nil {
				httpx.RespondError(c, http.StatusBadRequest, httpx.CodeValidationFailed, "caseIds 含有無效的個案 ID", nil)
				return
			}
			caseIDs = append(caseIDs, id)
		}
	}

	excelBytes, err := h.masterService.GenerateCaseProfileWorkbook(c.Request.Context(), caseIDs)
	if err != nil {
		httpx.RespondError(c, http.StatusInternalServerError, httpx.CodeInternalError, "產生個案主檔 Excel 失敗", nil)
		return
	}
	fileName := "個案資料彙整.xlsx"
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"case_profile.xlsx\"; filename*=UTF-8''%s", url.PathEscape(fileName)))
	c.Data(http.StatusOK, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", excelBytes)
}

// Get 取得單筆個案主檔明細。
func (h *CaseHandler) Get(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		httpx.RespondError(c, http.StatusBadRequest, httpx.CodeValidationFailed, "無效的個案 ID", nil)
		return
	}

	entity, err := h.masterService.GetCaseByID(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, app.ErrCaseNotFound) {
			httpx.RespondErrorCode(c, http.StatusNotFound, httpx.CodeNotFound, err, nil)
			return
		}
		httpx.RespondErrorCode(c, http.StatusInternalServerError, httpx.CodeInternalError, err, nil)
		return
	}

	httpx.RespondSuccess(c, http.StatusOK, newCaseResponse(*entity), nil)
}

// Update 更新個案資料。
func (h *CaseHandler) Update(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		httpx.RespondError(c, http.StatusBadRequest, httpx.CodeValidationFailed, "無效的個案 ID", nil)
		return
	}

	var req struct {
		Name              *string      `json:"name"`
		SiteID            *uuid.UUID   `json:"siteId"`
		CaregiverID       *uuid.UUID   `json:"caregiverId"`
		HomeAddress       *string      `json:"homeAddress"`
		LTCLevel          *string      `json:"ltcLevel"`
		ServiceCategory   *int         `json:"serviceCategory"`
		ServiceUsageType  *int         `json:"serviceUsageType"`
		ClaimEndDate      optionalDate `json:"claimEndDate"`
		Status            *string      `json:"status"`
		HouseholdType     *string      `json:"householdType"`
		Gender            *string      `json:"gender"`
		BirthDate         optionalDate `json:"birthDate"`
		NationalID        *string      `json:"nationalId"`
		CareContactRole   *string      `json:"careContactRole"`
		CareContactName   *string      `json:"careContactName"`
		RegisteredAddress *string      `json:"registeredAddress"`
		Remarks           *string      `json:"remarks"`
	}
	if err := httpx.BindJSONStrict(c, &req); err != nil {
		httpx.RespondErrorCode(c, http.StatusBadRequest, httpx.CodeValidationFailed, err, httpx.ExtractValidationDetails(err))
		return
	}

	in := app.UpdateCaseInput{
		Name:                req.Name,
		SiteID:              req.SiteID,
		CaregiverID:         req.CaregiverID,
		HomeAddress:         req.HomeAddress,
		LTCLevel:            req.LTCLevel,
		ServiceCategory:     req.ServiceCategory,
		ServiceUsageType:    req.ServiceUsageType,
		ClaimEndDate:        req.ClaimEndDate.Value,
		ClaimEndDatePresent: req.ClaimEndDate.Present,
		Status:              req.Status,
		HouseholdType:       req.HouseholdType,
		Gender:              req.Gender,
		NationalID:          req.NationalID,
		CareContactRole:     req.CareContactRole,
		CareContactName:     req.CareContactName,
		RegisteredAddress:   req.RegisteredAddress,
		Remarks:             req.Remarks,
	}
	in.BirthDate = req.BirthDate.Value
	in.BirthDatePresent = req.BirthDate.Present

	actorID := auth.GetActorID(c)
	actorRole := auth.GetActorRole(c)
	entity, err := h.masterService.UpdateCase(c.Request.Context(), id, in, actorID, actorRole, c.ClientIP(), c.Request.UserAgent())
	if err != nil {
		if errors.Is(err, app.ErrCaseNotFound) {
			httpx.RespondErrorCode(c, http.StatusNotFound, httpx.CodeNotFound, err, nil)
			return
		}
		if errors.Is(err, app.ErrInvalidNationalIDFormat) {
			httpx.RespondErrorCode(c, http.StatusBadRequest, httpx.CodeValidationFailed, err, []httpx.ErrorDetail{
				{Field: "nationalId", Reason: "身分證字號格式錯誤"},
			})
			return
		}
		if errors.Is(err, app.ErrDuplicateNationalID) {
			httpx.RespondErrorCode(c, http.StatusConflict, httpx.CodeValidationFailed, err, []httpx.ErrorDetail{
				{Field: "nationalId", Reason: "此身分證字號已存在於其他個案"},
			})
			return
		}
		httpx.RespondErrorCode(c, http.StatusInternalServerError, httpx.CodeInternalError, err, nil)
		return
	}

	var meta any
	if req.Name != nil {
		meta = h.relinkPendingMeta(c, entity.Name)
	}
	httpx.RespondSuccess(c, http.StatusOK, newCaseResponse(*entity), meta)
}

// ListDuplicateCandidates 列出所有待裁決的疑似重複個案。
func (h *CaseHandler) ListDuplicateCandidates(c *gin.Context) {
	list, err := h.masterService.ListDuplicateCandidates(c.Request.Context())
	if err != nil {
		httpx.RespondError(c, http.StatusInternalServerError, httpx.CodeInternalError, "查詢待裁決疑似重複個案失敗", nil)
		return
	}
	httpx.RespondSuccess(c, http.StatusOK, newDuplicateCandidateResponses(list), nil)
}

// ResolveDuplicateCandidate 裁決一筆疑似重複個案。
func (h *CaseHandler) ResolveDuplicateCandidate(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		httpx.RespondError(c, http.StatusBadRequest, httpx.CodeValidationFailed, "無效的暫存列 ID", nil)
		return
	}

	var req ResolveDuplicateCandidateRequest
	if err := httpx.BindJSONStrict(c, &req); err != nil {
		httpx.RespondErrorCode(c, http.StatusBadRequest, httpx.CodeValidationFailed, err, httpx.ExtractValidationDetails(err))
		return
	}

	actorID := auth.GetActorID(c)
	actorRole := auth.GetActorRole(c)
	entity, err := h.masterService.ResolveDuplicateCandidate(
		c.Request.Context(), id, req.Decision, req.TargetCaseID, req.MergeRemarks, actorID, actorRole, c.ClientIP(), c.Request.UserAgent(),
	)
	if err != nil {
		if errors.Is(err, app.ErrInvalidDuplicateDecision) {
			httpx.RespondErrorCode(c, http.StatusBadRequest, httpx.CodeValidationFailed, err, []httpx.ErrorDetail{
				{Field: "decision", Reason: "請選擇處理方式：新增為新個案，或合併至既有個案"},
			})
			return
		}
		if errors.Is(err, app.ErrDuplicateCandidateNotFound) {
			httpx.RespondErrorCode(c, http.StatusNotFound, httpx.CodeNotFound, err, nil)
			return
		}
		if errors.Is(err, app.ErrDuplicateCandidateResolved) {
			httpx.RespondErrorCode(c, http.StatusConflict, httpx.CodeValidationFailed, err, []httpx.ErrorDetail{
				{Reason: "此筆已被其他人裁決"},
			})
			return
		}
		if errors.Is(err, app.ErrDuplicateNationalID) {
			httpx.RespondErrorCode(c, http.StatusConflict, httpx.CodeValidationFailed, err, []httpx.ErrorDetail{
				{Field: "nationalId", Reason: "此身分證字號已存在於其他個案"},
			})
			return
		}
		httpx.RespondErrorCode(c, http.StatusInternalServerError, httpx.CodeInternalError, err, nil)
		return
	}

	httpx.RespondSuccess(c, http.StatusOK, newCaseResponse(*entity), nil)
}

// DiscardDuplicateCandidate 忽略一筆疑似重複個案，直接把暫存列從系統刪除。
func (h *CaseHandler) DiscardDuplicateCandidate(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		httpx.RespondError(c, http.StatusBadRequest, httpx.CodeValidationFailed, "無效的暫存列 ID", nil)
		return
	}

	actorID := auth.GetActorID(c)
	actorRole := auth.GetActorRole(c)
	if err := h.masterService.DiscardDuplicateCandidate(
		c.Request.Context(), id, actorID, actorRole, c.ClientIP(), c.Request.UserAgent(),
	); err != nil {
		if errors.Is(err, app.ErrDuplicateCandidateNotFound) {
			httpx.RespondErrorCode(c, http.StatusNotFound, httpx.CodeNotFound, err, nil)
			return
		}
		if errors.Is(err, app.ErrDuplicateCandidateResolved) {
			httpx.RespondErrorCode(c, http.StatusConflict, httpx.CodeValidationFailed, err, []httpx.ErrorDetail{
				{Reason: "此筆已被其他人裁決"},
			})
			return
		}
		httpx.RespondErrorCode(c, http.StatusInternalServerError, httpx.CodeInternalError, err, nil)
		return
	}

	httpx.RespondSuccess(c, http.StatusNoContent, nil, nil)
}

// GetSchedule 取得個案現行排班。
func (h *CaseHandler) GetSchedule(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		httpx.RespondError(c, http.StatusBadRequest, httpx.CodeValidationFailed, "無效的個案 ID", nil)
		return
	}

	sched, err := h.masterService.GetActiveScheduleForCaseOnDate(c.Request.Context(), id, clock.Today())
	if err != nil {
		httpx.RespondError(c, http.StatusInternalServerError, httpx.CodeInternalError, "查詢個案排班失敗", nil)
		return
	}
	// 個案尚未排班是正常狀態而非錯誤，以 200 搭配 null data 呈現，交由前端顯示「尚無現行排班」
	if sched == nil {
		httpx.RespondSuccess(c, http.StatusOK, nil, nil)
		return
	}

	httpx.RespondSuccess(c, http.StatusOK, newCaseScheduleResponse(*sched), nil)
}

// SaveSchedule 儲存/更新個案排班。
func (h *CaseHandler) SaveSchedule(c *gin.Context) {
	caseID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		httpx.RespondError(c, http.StatusBadRequest, httpx.CodeValidationFailed, "無效的個案 ID", nil)
		return
	}

	var req SaveScheduleRequest
	if err := httpx.BindJSONStrict(c, &req); err != nil {
		httpx.RespondErrorCode(c, http.StatusBadRequest, httpx.CodeValidationFailed, err, httpx.ExtractValidationDetails(err))
		return
	}

	sched, err := h.masterService.CreateCaseSchedule(c.Request.Context(), req.ToService(caseID))
	if err != nil {
		respondScheduleError(c, err)
		return
	}

	httpx.RespondSuccess(c, http.StatusOK, newCaseScheduleResponse(*sched), nil)
}
