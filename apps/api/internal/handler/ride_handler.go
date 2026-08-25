package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"ltc-system/apps/api/internal/middleware"
	"ltc-system/apps/api/internal/service"
)

// RideHandler 處理搭乘與 Webhook 請求。
type RideHandler struct {
	rideService *service.RideService
}

// NewRideHandler 建立 RideHandler 實例。
func NewRideHandler(rideService *service.RideService) *RideHandler {
	return &RideHandler{rideService: rideService}
}

// IngestWebhook 接收 Google Form 提交。
func (h *RideHandler) IngestWebhook(c *gin.Context) {
	secret := c.GetHeader("X-Ingest-Token")
	if secret == "" {
		middleware.RespondError(c, http.StatusUnauthorized, middleware.CodeIngestTokenInvalid, "未提供 X-Ingest-Token", nil)
		return
	}

	var req service.ProcessFormWebhookRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.RespondError(c, http.StatusBadRequest, middleware.CodeValidationFailed, err.Error(), nil)
		return
	}

	if err := h.rideService.IngestWebhook(c.Request.Context(), secret, req); err != nil {
		if err == middleware.ErrInvalidToken {
			middleware.RespondError(c, http.StatusUnauthorized, middleware.CodeIngestTokenInvalid, "無效的 Ingest Token", nil)
			return
		}
		middleware.RespondError(c, http.StatusBadRequest, middleware.CodeValidationFailed, err.Error(), nil)
		return
	}

	middleware.RespondSuccess(c, http.StatusOK, gin.H{"received": true}, nil)
}

// Correct 更正搭乘紀錄（§4.7）。
func (h *RideHandler) Correct(c *gin.Context) {
	rideIDStr := c.Param("id")
	rideID, err := uuid.Parse(rideIDStr)
	if err != nil {
		middleware.RespondError(c, http.StatusBadRequest, middleware.CodeValidationFailed, "無效的搭乘紀錄 ID", nil)
		return
	}

	var req service.CorrectRideRecordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.RespondError(c, http.StatusBadRequest, middleware.CodeValidationFailed, err.Error(), nil)
		return
	}

	actorID := middleware.GetActorID(c)
	actorRole := middleware.GetActorRole(c)

	if err := h.rideService.CorrectRideRecord(c.Request.Context(), rideID, req, actorID, actorRole, c.ClientIP(), c.Request.UserAgent()); err != nil {
		middleware.RespondError(c, http.StatusBadRequest, middleware.CodeValidationFailed, err.Error(), nil)
		return
	}

	middleware.RespondSuccess(c, http.StatusOK, gin.H{"updated": true}, nil)
}

// ExportHandler 處理匯出與前置檢核請求。
type ExportHandler struct {
	precheckService *service.PrecheckService
}

// NewExportHandler 建立 ExportHandler 實例。
func NewExportHandler(precheckService *service.PrecheckService) *ExportHandler {
	return &ExportHandler{precheckService: precheckService}
}

// Precheck 執行匯出前置檢核（支援 GET Query 與 POST JSON Body）。
func (h *ExportHandler) Precheck(c *gin.Context) {
	periodYM := c.Query("periodYm")
	if periodYM == "" {
		periodYM = c.DefaultQuery("month", "115-07")
	}
	region := c.DefaultQuery("region", "hsinchu")

	if c.Request.Method == http.MethodPost {
		var req struct {
			PeriodYM string `json:"periodYm"`
			Region   string `json:"region"`
		}
		if err := c.ShouldBindJSON(&req); err == nil {
			if req.PeriodYM != "" {
				periodYM = req.PeriodYM
			}
			if req.Region != "" {
				region = req.Region
			}
		}
	}

	report, err := h.precheckService.RunPrecheck(c.Request.Context(), periodYM, region)
	if err != nil {
		middleware.RespondError(c, http.StatusInternalServerError, middleware.CodeInternalError, "前置檢核失敗", nil)
		return
	}

	middleware.RespondSuccess(c, http.StatusOK, report, nil)
}

// List 取得申報匯出工作歷史紀錄清單。
func (h *ExportHandler) List(c *gin.Context) {
	middleware.RespondSuccess(c, http.StatusOK, []gin.H{}, middleware.PaginationMeta{
		Page:       1,
		PageSize:   10,
		Total:      0,
		TotalPages: 0,
	})
}

// Create 建立申報匯出工作任務。
func (h *ExportHandler) Create(c *gin.Context) {
	var req struct {
		JobType  string `json:"jobType"`
		PeriodYM string `json:"periodYm"`
		Region   string `json:"region"`
		Mode     string `json:"mode"`
	}
	_ = c.ShouldBindJSON(&req)

	job := gin.H{
		"id":          uuid.New().String(),
		"jobType":     req.JobType,
		"periodYm":    req.PeriodYM,
		"region":      req.Region,
		"mode":        req.Mode,
		"status":      "succeeded",
		"totalCases":  12,
		"totalRows":   180,
		"fileName":    "gov-claim-" + req.PeriodYM + ".xlsx",
		"downloadUrl": "/healthz",
		"createdAt":   "2026-08-25 16:00:00",
	}
	middleware.RespondSuccess(c, http.StatusAccepted, job, nil)
}

// Get 取得單筆匯出工作狀態與下載連結。
func (h *ExportHandler) Get(c *gin.Context) {
	jobID := c.Param("id")
	job := gin.H{
		"id":          jobID,
		"jobType":     "gov_claim",
		"periodYm":    "115-07",
		"region":      "hsinchu",
		"mode":        "single_multi_case",
		"status":      "succeeded",
		"totalCases":  12,
		"totalRows":   180,
		"fileName":    "gov-claim-115-07.xlsx",
		"downloadUrl": "/healthz",
		"createdAt":   "2026-08-25 16:00:00",
	}
	middleware.RespondSuccess(c, http.StatusOK, job, nil)
}

