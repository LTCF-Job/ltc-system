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

// Precheck 執行匯出前置檢核。
func (h *ExportHandler) Precheck(c *gin.Context) {
	periodYM := c.DefaultQuery("month", "115-07")
	region := c.DefaultQuery("region", "hsinchu")

	report, err := h.precheckService.RunPrecheck(c.Request.Context(), periodYM, region)
	if err != nil {
		middleware.RespondError(c, http.StatusInternalServerError, middleware.CodeInternalError, "前置檢核失敗", nil)
		return
	}

	middleware.RespondSuccess(c, http.StatusOK, report, nil)
}
