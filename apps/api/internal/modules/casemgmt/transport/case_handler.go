package transport

import (
	"errors"
	"fmt"
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

// CaseHandler 處理個案相關之 HTTP 請求。
type CaseHandler struct {
	masterService *app.CaseService
}

// NewCaseHandler 建立 CaseHandler 實例。
func NewCaseHandler(
	masterService *app.CaseService,
) *CaseHandler {
	return &CaseHandler{
		masterService: masterService,
	}
}

// List 查詢個案清單（回傳遮罩身分證）。
func (h *CaseHandler) List(c *gin.Context) {
	page, pageSize, err := httpx.ParsePagination(c)
	if err != nil {
		httpx.RespondError(c, http.StatusBadRequest, httpx.CodeValidationFailed, "分頁參數格式錯誤", nil)
		return
	}
	region := c.Query("region")
	status := c.Query("status")
	q := c.Query("q")
	// 待維護個案預設不出現在任何清單，呼叫端要明確表態才拿得到：unresolvedLink 只取待維護，
	// includePending 取全部。預設排除，避免新增呼叫端忘記帶參數就把待維護資料洩漏出去。
	unresolvedLink := httpx.QueryBool(c, "unresolvedLink")
	includePending := httpx.QueryBool(c, "includePending")
	excludePending := !unresolvedLink && !includePending

	cases, total, err := h.masterService.ListCases(c.Request.Context(), region, status, q, page, pageSize, unresolvedLink, excludePending)
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
		httpx.RespondErrorCode(c, http.StatusBadRequest, httpx.CodeValidationFailed, err, nil)
		return
	}

	httpx.RespondSuccess(c, http.StatusCreated, newCaseResponse(*entity), nil)
}

// Reveal 解密個案身分證號。
func (h *CaseHandler) Reveal(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		httpx.RespondError(c, http.StatusBadRequest, httpx.CodeValidationFailed, "無效的個案 ID", nil)
		return
	}

	actorID := auth.GetActorID(c)
	actorRole := auth.GetActorRole(c)

	plainID, err := h.masterService.RevealCaseNationalID(
		c.Request.Context(), id, actorID, actorRole, c.ClientIP(), c.Request.UserAgent(),
	)
	if err != nil {
		if errors.Is(err, app.ErrNationalIDNotConfigured) {
			httpx.RespondError(c, http.StatusUnprocessableEntity, httpx.CodeValidationFailed, "個案尚未設定身分證資料", nil)
			return
		}
		if errors.Is(err, app.ErrRevealAuditUnavailable) {
			httpx.RespondErrorCode(c, http.StatusServiceUnavailable, httpx.CodeServiceUnavailable, err, nil)
			return
		}
		httpx.RespondError(c, http.StatusNotFound, httpx.CodeNotFound, "個案不存在或解密失敗", nil)
		return
	}

	httpx.RespondSuccess(c, http.StatusOK, gin.H{"nationalId": plainID}, nil)
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
		httpx.RespondErrorCode(c, http.StatusBadRequest, httpx.CodeValidationFailed, err, nil)
		return
	}

	httpx.RespondSuccess(c, http.StatusCreated, newCaseScheduleResponse(*sched), nil)
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
		HomeAddress       *string      `json:"homeAddress"`
		Region            *string      `json:"region"`
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
		HomeAddress:         req.HomeAddress,
		Region:              req.Region,
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

	httpx.RespondSuccess(c, http.StatusOK, newCaseResponse(*entity), nil)
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

// RevealDuplicateCandidateNationalID 解密單筆疑似重複個案暫存列的身分證字號，供裁決頁比對。
func (h *CaseHandler) RevealDuplicateCandidateNationalID(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		httpx.RespondError(c, http.StatusBadRequest, httpx.CodeValidationFailed, "無效的暫存列 ID", nil)
		return
	}

	actorID := auth.GetActorID(c)
	actorRole := auth.GetActorRole(c)
	plainID, err := h.masterService.RevealDuplicateCandidateNationalID(c.Request.Context(), id, actorID, actorRole, c.ClientIP(), c.Request.UserAgent())
	if err != nil {
		if errors.Is(err, app.ErrDuplicateCandidateNotFound) {
			httpx.RespondErrorCode(c, http.StatusNotFound, httpx.CodeNotFound, err, nil)
			return
		}
		if errors.Is(err, app.ErrNationalIDNotConfigured) {
			httpx.RespondError(c, http.StatusUnprocessableEntity, httpx.CodeValidationFailed, "此列尚未設定身分證資料", nil)
			return
		}
		if errors.Is(err, app.ErrRevealAuditUnavailable) {
			httpx.RespondErrorCode(c, http.StatusServiceUnavailable, httpx.CodeServiceUnavailable, err, nil)
			return
		}
		httpx.RespondError(c, http.StatusInternalServerError, httpx.CodeInternalError, "解密失敗", nil)
		return
	}
	httpx.RespondSuccess(c, http.StatusOK, gin.H{"nationalId": plainID}, nil)
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
				{Field: "decision", Reason: "decision 必須為 confirmed_new 或 merged_existing"},
			})
			return
		}
		if errors.Is(err, app.ErrDuplicateCandidateNotFound) {
			httpx.RespondErrorCode(c, http.StatusNotFound, httpx.CodeNotFound, err, nil)
			return
		}
		if errors.Is(err, app.ErrDuplicateCandidateResolved) {
			httpx.RespondErrorCode(c, http.StatusConflict, httpx.CodeValidationFailed, err, nil)
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
			httpx.RespondErrorCode(c, http.StatusConflict, httpx.CodeValidationFailed, err, nil)
			return
		}
		httpx.RespondErrorCode(c, http.StatusInternalServerError, httpx.CodeInternalError, err, nil)
		return
	}

	httpx.RespondSuccess(c, http.StatusNoContent, nil, nil)
}

// UpdateTransportPreference 更新個案的交通偏好（所屬單位與去回程車輛）。
func (h *CaseHandler) UpdateTransportPreference(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		httpx.RespondError(c, http.StatusBadRequest, httpx.CodeValidationFailed, "無效的個案 ID", nil)
		return
	}

	var req struct {
		SiteID                 *uuid.UUID `json:"siteId"`
		OutboundVehicleID      *uuid.UUID `json:"outboundVehicleId"`
		InboundVehicleID       *uuid.UUID `json:"inboundVehicleId"`
		SiteNameRaw            string     `json:"siteNameRaw"`
		OutboundVehicleNameRaw string     `json:"outboundVehicleNameRaw"`
		InboundVehicleNameRaw  string     `json:"inboundVehicleNameRaw"`
	}
	if err := httpx.BindJSONStrict(c, &req); err != nil {
		httpx.RespondErrorCode(c, http.StatusBadRequest, httpx.CodeValidationFailed, err, httpx.ExtractValidationDetails(err))
		return
	}

	entity, err := h.masterService.UpdateCaseTransportPreference(
		c.Request.Context(), id, req.SiteID, req.OutboundVehicleID, req.InboundVehicleID,
		req.SiteNameRaw, req.OutboundVehicleNameRaw, req.InboundVehicleNameRaw,
		app.AuditContext{
			ActorID:   auth.GetActorID(c),
			ActorRole: auth.GetActorRole(c),
			IPAddress: c.ClientIP(),
			UserAgent: c.Request.UserAgent(),
		},
	)
	if err != nil {
		httpx.RespondError(c, http.StatusInternalServerError, httpx.CodeInternalError, "更新交通偏好失敗", nil)
		return
	}

	httpx.RespondSuccess(c, http.StatusOK, newCaseResponse(*entity), nil)
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
		httpx.RespondErrorCode(c, http.StatusBadRequest, httpx.CodeValidationFailed, err, nil)
		return
	}

	httpx.RespondSuccess(c, http.StatusOK, newCaseScheduleResponse(*sched), nil)
}
