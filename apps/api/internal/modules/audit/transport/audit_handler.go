package transport

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"ltc-system/apps/api/internal/domain/rocdate"
	"ltc-system/apps/api/internal/modules/audit/app"
	"ltc-system/apps/api/internal/platform/httpx"
)

// AuditHandler 處理系統稽核日誌之 HTTP 請求。
type AuditHandler struct {
	svc *app.Service
}

// NewAuditHandler 建立 AuditHandler 實例。
func NewAuditHandler(svc *app.Service) *AuditHandler {
	return &AuditHandler{svc: svc}
}

// List 查詢稽核紀錄（admin 專屬權限）。
func (h *AuditHandler) List(c *gin.Context) {
	page, pageSize, err := httpx.ParsePagination(c)
	if err != nil {
		httpx.RespondError(c, http.StatusBadRequest, httpx.CodeValidationFailed, "分頁參數格式錯誤", nil)
		return
	}

	filter := app.Filter{
		Action:     c.Query("action"),
		EntityType: c.Query("entityType"),
		EntityID:   c.Query("entityId"),
		Q:          c.Query("q"),
		Page:       page,
		PageSize:   pageSize,
	}

	if actorIDStr := c.Query("actorId"); actorIDStr != "" {
		id, err := uuid.Parse(actorIDStr)
		if err != nil {
			httpx.RespondError(c, http.StatusBadRequest, httpx.CodeValidationFailed, "actorId 必須為有效 UUID", nil)
			return
		}
		filter.ActorID = &id
	}

	if startStr := c.Query("startDate"); startStr != "" {
		t, err := rocdate.ParseDate(startStr)
		if err != nil {
			httpx.RespondError(c, http.StatusBadRequest, httpx.CodeValidationFailed, "startDate 必須為有效日期", nil)
			return
		}
		filter.StartDate = &t
	}

	if endStr := c.Query("endDate"); endStr != "" {
		t, err := rocdate.ParseDate(endStr)
		if err != nil {
			httpx.RespondError(c, http.StatusBadRequest, httpx.CodeValidationFailed, "endDate 必須為有效日期", nil)
			return
		}
		endDay := t.AddDate(0, 0, 1) // 包含該日全天
		filter.EndDate = &endDay
	}

	logs, total, err := h.svc.List(c.Request.Context(), filter)
	if err != nil {
		httpx.RespondError(c, http.StatusInternalServerError, httpx.CodeInternalError, "查詢稽核日誌失敗", nil)
		return
	}

	totalPages := int(total) / pageSize
	if int(total)%pageSize != 0 {
		totalPages++
	}

	httpx.RespondSuccess(c, http.StatusOK, newRecordResponses(logs), httpx.PaginationMeta{
		Page:       page,
		PageSize:   pageSize,
		Total:      total,
		TotalPages: totalPages,
	})
}
