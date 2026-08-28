package handler

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"ltc-system/apps/api/internal/middleware"
	"ltc-system/apps/api/internal/service"
)

// CaseHandler 處理個案相關之 HTTP 請求。
type CaseHandler struct {
	masterService *service.MasterService
	importService *service.ImportService
}

// NewCaseHandler 建立 CaseHandler 實例。
func NewCaseHandler(
	masterService *service.MasterService,
	importService *service.ImportService,
) *CaseHandler {
	return &CaseHandler{
		masterService: masterService,
		importService: importService,
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

	cases, total, err := h.masterService.ListCases(c.Request.Context(), region, status, q, page, pageSize)
	if err != nil {
		middleware.RespondError(c, http.StatusInternalServerError, middleware.CodeInternalError, "查詢個案失敗", nil)
		return
	}

	totalPages := int(total) / pageSize
	if int(total)%pageSize != 0 {
		totalPages++
	}

	middleware.RespondSuccess(c, http.StatusOK, cases, middleware.PaginationMeta{
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
		middleware.RespondErrorCode(c, http.StatusBadRequest, middleware.CodeValidationFailed, err, nil)
		return
	}

	actorID := middleware.GetActorID(c)
	actorRole := middleware.GetActorRole(c)

	entity, err := h.masterService.CreateCase(
		c.Request.Context(), req.ToService(), actorID, actorRole, c.ClientIP(), c.Request.UserAgent(),
	)
	if err != nil {
		if errors.Is(err, service.ErrDuplicateNationalID) {
			middleware.RespondError(c, http.StatusConflict, middleware.CodeDuplicateNationalID, "身分證字號重複", nil)
			return
		}
		middleware.RespondErrorCode(c, http.StatusBadRequest, middleware.CodeValidationFailed, err, nil)
		return
	}

	middleware.RespondSuccess(c, http.StatusCreated, entity, nil)
}

// Reveal 解密個案身分證號。
func (h *CaseHandler) Reveal(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		middleware.RespondError(c, http.StatusBadRequest, middleware.CodeValidationFailed, "無效的個案 ID", nil)
		return
	}

	actorID := middleware.GetActorID(c)
	actorRole := middleware.GetActorRole(c)

	plainID, err := h.masterService.RevealCaseNationalID(
		c.Request.Context(), id, actorID, actorRole, c.ClientIP(), c.Request.UserAgent(),
	)
	if err != nil {
		middleware.RespondError(c, http.StatusNotFound, middleware.CodeNotFound, "個案不存在或解密失敗", nil)
		return
	}

	middleware.RespondSuccess(c, http.StatusOK, gin.H{"nationalId": plainID}, nil)
}

// CreateSchedule 建立個案排班設定與時段明細。
func (h *CaseHandler) CreateSchedule(c *gin.Context) {
	var req CreateScheduleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.RespondErrorCode(c, http.StatusBadRequest, middleware.CodeValidationFailed, err, nil)
		return
	}

	sched, err := h.masterService.CreateCaseSchedule(c.Request.Context(), req.ToService())
	if err != nil {
		middleware.RespondErrorCode(c, http.StatusBadRequest, middleware.CodeValidationFailed, err, nil)
		return
	}

	middleware.RespondSuccess(c, http.StatusCreated, sched, nil)
}

// ImportExcel 批次上傳解析個案新增資料 Excel 或 CSV 檔案。
func (h *CaseHandler) ImportExcel(c *gin.Context) {
	fileHeader, err := c.FormFile("file")
	if err != nil {
		middleware.RespondError(c, http.StatusBadRequest, middleware.CodeValidationFailed, "未提供上傳檔案", nil)
		return
	}

	f, err := fileHeader.Open()
	if err != nil {
		middleware.RespondError(c, http.StatusBadRequest, middleware.CodeValidationFailed, "無法開啟檔案", nil)
		return
	}
	defer f.Close()

	preview, err := h.importService.ParseCases(f, fileHeader.Filename)
	if err != nil {
		middleware.RespondErrorCode(c, http.StatusBadRequest, middleware.CodeValidationFailed, err, nil)
		return
	}

	// 依 dryRun 參數區分預覽或正式寫入
	dryRun := c.DefaultQuery("dryRun", "true")
	if dryRun == "false" {
		actorID := middleware.GetActorID(c)
		actorRole := middleware.GetActorRole(c)
		result, err := h.importService.CommitCases(
			c.Request.Context(), preview, actorID, actorRole, c.ClientIP(), c.Request.UserAgent(),
		)
		if err != nil {
			middleware.RespondError(c, http.StatusInternalServerError, middleware.CodeInternalError, "匯入個案寫入失敗", nil)
			return
		}
		middleware.RespondSuccess(c, http.StatusOK, result, nil)
		return
	}

	middleware.RespondSuccess(c, http.StatusOK, preview, nil)
}

// DownloadTemplate 下載個案批次匯入範本 (支援 .xlsx 與 .csv)。
func (h *CaseHandler) DownloadTemplate(c *gin.Context) {
	format := strings.ToLower(c.DefaultQuery("format", "xlsx"))

	if format == "csv" {
		csvContent := service.GenerateCaseImportTemplateCSV()
		fileName := "個案批次匯入範本.csv"
		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"case_template.csv\"; filename*=UTF-8''%s", url.PathEscape(fileName)))
		c.Data(http.StatusOK, "text/csv; charset=utf-8", []byte(csvContent))
		return
	}

	excelBytes, err := service.GenerateCaseImportTemplateExcel()
	if err != nil {
		middleware.RespondError(c, http.StatusInternalServerError, middleware.CodeInternalError, "產生 Excel 範本失敗", nil)
		return
	}

	fileName := "個案批次匯入範本.xlsx"
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"case_template.xlsx\"; filename*=UTF-8''%s", url.PathEscape(fileName)))
	c.Data(http.StatusOK, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", excelBytes)
}

// ExportProfileWorkbook 下載與個案彙整表相同格式的主檔資料。
func (h *CaseHandler) ExportProfileWorkbook(c *gin.Context) {
	excelBytes, err := h.masterService.GenerateCaseProfileWorkbook(c.Request.Context())
	if err != nil {
		middleware.RespondError(c, http.StatusInternalServerError, middleware.CodeInternalError, "產生個案主檔 Excel 失敗", nil)
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
		middleware.RespondError(c, http.StatusBadRequest, middleware.CodeValidationFailed, "無效的個案 ID", nil)
		return
	}

	entity, err := h.masterService.GetCaseByID(c.Request.Context(), id)
	if err != nil {
		middleware.RespondError(c, http.StatusNotFound, middleware.CodeNotFound, "找不到此個案", nil)
		return
	}

	middleware.RespondSuccess(c, http.StatusOK, entity, nil)
}

// Update 更新個案資料。
func (h *CaseHandler) Update(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		middleware.RespondError(c, http.StatusBadRequest, middleware.CodeValidationFailed, "無效的個案 ID", nil)
		return
	}

	var req struct {
		Name              *string `json:"name"`
		HomeAddress       *string `json:"homeAddress"`
		Region            *string `json:"region"`
		LTCLevel          *string `json:"ltcLevel"`
		ServiceCategory   *int    `json:"serviceCategory"`
		ServiceUsageType  *int    `json:"serviceUsageType"`
		ClaimStartDate    *string `json:"claimStartDate"`
		ClaimEndDate      *string `json:"claimEndDate"`
		Status            *string `json:"status"`
		HouseholdType     *string `json:"householdType"`
		Gender            *string `json:"gender"`
		BirthDate         *string `json:"birthDate"`
		CareContactRole   *string `json:"careContactRole"`
		CareContactName   *string `json:"careContactName"`
		RegisteredAddress *string `json:"registeredAddress"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.RespondErrorCode(c, http.StatusBadRequest, middleware.CodeValidationFailed, err, nil)
		return
	}

	in := service.UpdateCaseInput{
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
	}
	if req.BirthDate != nil {
		if t, err := time.Parse("2006-01-02", *req.BirthDate); err == nil {
			in.BirthDate = &t
		}
	}

	entity, err := h.masterService.UpdateCase(c.Request.Context(), id, in)
	if err != nil {
		middleware.RespondError(c, http.StatusNotFound, middleware.CodeNotFound, "找不到此個案", nil)
		return
	}

	middleware.RespondSuccess(c, http.StatusOK, entity, nil)
}

// UpdateTransportPreference 更新個案的交通偏好（所屬據點與去回程車輛）。
func (h *CaseHandler) UpdateTransportPreference(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		middleware.RespondError(c, http.StatusBadRequest, middleware.CodeValidationFailed, "無效的個案 ID", nil)
		return
	}

	var req struct {
		SiteID            uuid.UUID `json:"siteId" binding:"required"`
		OutboundVehicleID uuid.UUID `json:"outboundVehicleId" binding:"required"`
		InboundVehicleID  uuid.UUID `json:"inboundVehicleId" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.RespondErrorCode(c, http.StatusBadRequest, middleware.CodeValidationFailed, err, nil)
		return
	}

	entity, err := h.masterService.UpdateCaseTransportPreference(c.Request.Context(), id, req.SiteID, req.OutboundVehicleID, req.InboundVehicleID)
	if err != nil {
		middleware.RespondError(c, http.StatusInternalServerError, middleware.CodeInternalError, "更新交通偏好失敗", nil)
		return
	}

	middleware.RespondSuccess(c, http.StatusOK, entity, nil)
}

// GetSchedule 取得個案現行排班。
func (h *CaseHandler) GetSchedule(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		middleware.RespondError(c, http.StatusBadRequest, middleware.CodeValidationFailed, "無效的個案 ID", nil)
		return
	}

	// TODO: 尚無「個案排班」在無現行排班時的產品規格確認，先誠實回傳查無資料，
	// 不再回傳假造的竹北日照中心／竹北一車預設排班（原本無論真實查詢成功與否，
	// 只要查無排班或查詢出錯都會回傳同一組寫死的假資料，兩種情況也未區分）。
	sched, err := h.masterService.GetActiveScheduleForCaseOnDate(c.Request.Context(), id, time.Now())
	if err != nil {
		middleware.RespondError(c, http.StatusInternalServerError, middleware.CodeInternalError, "查詢個案排班失敗", nil)
		return
	}
	if sched == nil {
		middleware.RespondError(c, http.StatusNotFound, middleware.CodeNotFound, "查無現行排班", nil)
		return
	}

	middleware.RespondSuccess(c, http.StatusOK, sched, nil)
}

// SaveSchedule 儲存/更新個案排班。
func (h *CaseHandler) SaveSchedule(c *gin.Context) {
	var req CreateScheduleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.RespondErrorCode(c, http.StatusBadRequest, middleware.CodeValidationFailed, err, nil)
		return
	}

	sched, err := h.masterService.CreateCaseSchedule(c.Request.Context(), req.ToService())
	if err != nil {
		middleware.RespondErrorCode(c, http.StatusBadRequest, middleware.CodeValidationFailed, err, nil)
		return
	}

	middleware.RespondSuccess(c, http.StatusOK, sched, nil)
}
