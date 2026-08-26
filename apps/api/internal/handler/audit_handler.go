package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"ltc-system/apps/api/internal/middleware"
	"ltc-system/apps/api/internal/repository"
	"ltc-system/apps/api/internal/service"
)

// AuditHandler 處理系統稽核日誌之 HTTP 請求。
type AuditHandler struct {
	svc *service.AuditService
}

// NewAuditHandler 建立 AuditHandler 實例。
func NewAuditHandler(svc *service.AuditService) *AuditHandler {
	return &AuditHandler{svc: svc}
}

// List 查詢稽核紀錄（admin 專屬權限）。
func (h *AuditHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	if pageSize > 100 {
		pageSize = 100
	}

	filter := repository.AuditFilter{
		Action:     c.Query("action"),
		EntityType: c.Query("entityType"),
		EntityID:   c.Query("entityId"),
		Q:          c.Query("q"),
		Page:       page,
		PageSize:   pageSize,
	}

	if actorIDStr := c.Query("actorId"); actorIDStr != "" {
		if id, err := uuid.Parse(actorIDStr); err == nil {
			filter.ActorID = &id
		}
	}

	if startStr := c.Query("startDate"); startStr != "" {
		if t, err := time.Parse("2006-01-02", startStr); err == nil {
			filter.StartDate = &t
		}
	}

	if endStr := c.Query("endDate"); endStr != "" {
		if t, err := time.Parse("2006-01-02", endStr); err == nil {
			endDay := t.AddDate(0, 0, 1) // 包含該日全天
			filter.EndDate = &endDay
		}
	}

	logs, total, err := h.svc.ListAuditLogs(c.Request.Context(), filter)
	if err != nil {
		middleware.RespondError(c, http.StatusInternalServerError, middleware.CodeInternalError, "查詢稽核日誌失敗", nil)
		return
	}

	totalPages := int(total) / pageSize
	if int(total)%pageSize != 0 {
		totalPages++
	}

	middleware.RespondSuccess(c, http.StatusOK, logs, middleware.PaginationMeta{
		Page:       page,
		PageSize:   pageSize,
		Total:      total,
		TotalPages: totalPages,
	})
}
