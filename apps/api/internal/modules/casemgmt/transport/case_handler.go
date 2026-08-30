package transport

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"ltc-system/apps/api/internal/modules/casemgmt/app"
	"ltc-system/apps/api/internal/platform/auth"
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
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	if pageSize > 100 {
		pageSize = 100
	}
	region := c.Query("region")
	status := c.Query("status")
	q := c.Query("q")
	unresolvedLink := c.Query("unresolvedLink") == "true"

	cases, total, err := h.masterService.ListCases(c.Request.Context(), region, status, q, page, pageSize, unresolvedLink)
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
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.RespondErrorCode(c, http.StatusBadRequest, httpx.CodeValidationFailed, err, nil)
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
		httpx.RespondError(c, http.StatusNotFound, httpx.CodeNotFound, "個案不存在或解密失敗", nil)
		return
	}

	httpx.RespondSuccess(c, http.StatusOK, gin.H{"nationalId": plainID}, nil)
}

// CreateSchedule 建立個案排班設定與時段明細。
func (h *CaseHandler) CreateSchedule(c *gin.Context) {
	var req CreateScheduleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.RespondErrorCode(c, http.StatusBadRequest, httpx.CodeValidationFailed, err, nil)
		return
	}

	sched, err := h.masterService.CreateCaseSchedule(c.Request.Context(), req.ToService())
	if err != nil {
		httpx.RespondErrorCode(c, http.StatusBadRequest, httpx.CodeValidationFailed, err, nil)
		return
	}

	httpx.RespondSuccess(c, http.StatusCreated, newCaseScheduleResponse(*sched), nil)
}

// ExportProfileWorkbook 下載與個案彙整表相同格式的主檔資料。
func (h *CaseHandler) ExportProfileWorkbook(c *gin.Context) {
	excelBytes, err := h.masterService.GenerateCaseProfileWorkbook(c.Request.Context())
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
		httpx.RespondError(c, http.StatusNotFound, httpx.CodeNotFound, "找不到此個案", nil)
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
		Name              *string `json:"name"`
		HomeAddress       *string `json:"homeAddress"`
		Region            *string `json:"region"`
		LTCLevel          *string `json:"ltcLevel"`
		ServiceCategory   *int    `json:"serviceCategory"`
		ServiceUsageType  *int    `json:"serviceUsageType"`
		ClaimEndDate      *string `json:"claimEndDate"`
		Status            *string `json:"status"`
		HouseholdType     *string `json:"householdType"`
		Gender            *string `json:"gender"`
		BirthDate         *string `json:"birthDate"`
		CareContactRole   *string `json:"careContactRole"`
		CareContactName   *string `json:"careContactName"`
		RegisteredAddress *string `json:"registeredAddress"`
		Remarks           *string `json:"remarks"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.RespondErrorCode(c, http.StatusBadRequest, httpx.CodeValidationFailed, err, nil)
		return
	}

	in := app.UpdateCaseInput{
		Name:              req.Name,
		HomeAddress:       req.HomeAddress,
		Region:            req.Region,
		LTCLevel:          req.LTCLevel,
		ServiceCategory:   req.ServiceCategory,
		ServiceUsageType:  req.ServiceUsageType,
		Status:            req.Status,
		HouseholdType:     req.HouseholdType,
		Gender:            req.Gender,
		CareContactRole:   req.CareContactRole,
		CareContactName:   req.CareContactName,
		RegisteredAddress: req.RegisteredAddress,
		Remarks:           req.Remarks,
	}
	if req.BirthDate != nil {
		if t, err := time.Parse("2006-01-02", *req.BirthDate); err == nil {
			in.BirthDate = &t
		}
	}

	entity, err := h.masterService.UpdateCase(c.Request.Context(), id, in)
	if err != nil {
		httpx.RespondError(c, http.StatusNotFound, httpx.CodeNotFound, "找不到此個案", nil)
		return
	}

	httpx.RespondSuccess(c, http.StatusOK, newCaseResponse(*entity), nil)
}

// UpdateTransportPreference 更新個案的交通偏好（所屬據點與去回程車輛）。
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
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.RespondErrorCode(c, http.StatusBadRequest, httpx.CodeValidationFailed, err, nil)
		return
	}

	entity, err := h.masterService.UpdateCaseTransportPreference(
		c.Request.Context(), id, req.SiteID, req.OutboundVehicleID, req.InboundVehicleID,
		req.SiteNameRaw, req.OutboundVehicleNameRaw, req.InboundVehicleNameRaw,
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

	// TODO: 尚無「個案排班」在無現行排班時的產品規格確認，先誠實回傳查無資料，
	// 不再回傳假造的竹北日照中心／竹北一車預設排班（原本無論真實查詢成功與否，
	// 只要查無排班或查詢出錯都會回傳同一組寫死的假資料，兩種情況也未區分）。
	sched, err := h.masterService.GetActiveScheduleForCaseOnDate(c.Request.Context(), id, time.Now())
	if err != nil {
		httpx.RespondError(c, http.StatusInternalServerError, httpx.CodeInternalError, "查詢個案排班失敗", nil)
		return
	}
	if sched == nil {
		httpx.RespondError(c, http.StatusNotFound, httpx.CodeNotFound, "查無現行排班", nil)
		return
	}

	httpx.RespondSuccess(c, http.StatusOK, newCaseScheduleResponse(*sched), nil)
}

// SaveSchedule 儲存/更新個案排班。
func (h *CaseHandler) SaveSchedule(c *gin.Context) {
	var req CreateScheduleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.RespondErrorCode(c, http.StatusBadRequest, httpx.CodeValidationFailed, err, nil)
		return
	}

	sched, err := h.masterService.CreateCaseSchedule(c.Request.Context(), req.ToService())
	if err != nil {
		httpx.RespondErrorCode(c, http.StatusBadRequest, httpx.CodeValidationFailed, err, nil)
		return
	}

	httpx.RespondSuccess(c, http.StatusOK, newCaseScheduleResponse(*sched), nil)
}
