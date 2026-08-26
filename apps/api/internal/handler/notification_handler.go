package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"ltc-system/apps/api/internal/middleware"
	"ltc-system/apps/api/internal/service"
)

// NotificationHandler 處理通知收件人與日誌之 HTTP 請求。
type NotificationHandler struct {
	svc *service.NotificationService
}

// NewNotificationHandler 建立 NotificationHandler 實例。
func NewNotificationHandler(svc *service.NotificationService) *NotificationHandler {
	return &NotificationHandler{svc: svc}
}

// CreateRecipientRequest 定義新增收件人請求結構。
type CreateRecipientRequest struct {
	Topic       string  `json:"topic" binding:"required"`
	Email       string  `json:"email" binding:"required,email"`
	DisplayName *string `json:"displayName"`
}

// UpdateRecipientRequest 定義修改收件人請求結構。
type UpdateRecipientRequest struct {
	Email       string  `json:"email" binding:"required,email"`
	DisplayName *string `json:"displayName"`
	Active      bool    `json:"active"`
}

// ListRecipients 取得通知收件人清單。
func (h *NotificationHandler) ListRecipients(c *gin.Context) {
	topic := c.Query("topic")
	recipients, err := h.svc.ListRecipients(c.Request.Context(), topic)
	if err != nil {
		middleware.RespondError(c, http.StatusInternalServerError, middleware.CodeInternalError, "查詢收件人失敗", nil)
		return
	}
	middleware.RespondSuccess(c, http.StatusOK, recipients, nil)
}

// CreateRecipient 新增通知收件人（admin 專屬）。
func (h *NotificationHandler) CreateRecipient(c *gin.Context) {
	var req CreateRecipientRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.RespondError(c, http.StatusBadRequest, middleware.CodeValidationFailed, err.Error(), nil)
		return
	}

	actorID := middleware.GetActorID(c)
	actorRole := middleware.GetActorRole(c)

	item, err := h.svc.CreateRecipient(c.Request.Context(), req.Topic, req.Email, req.DisplayName, actorID, actorRole)
	if err != nil {
		middleware.RespondError(c, http.StatusBadRequest, middleware.CodeValidationFailed, err.Error(), nil)
		return
	}

	middleware.RespondSuccess(c, http.StatusCreated, item, nil)
}

// UpdateRecipient 修改通知收件人。
func (h *NotificationHandler) UpdateRecipient(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		middleware.RespondError(c, http.StatusBadRequest, middleware.CodeValidationFailed, "無效的收件人 ID", nil)
		return
	}

	var req UpdateRecipientRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.RespondError(c, http.StatusBadRequest, middleware.CodeValidationFailed, err.Error(), nil)
		return
	}

	actorID := middleware.GetActorID(c)
	actorRole := middleware.GetActorRole(c)

	item, err := h.svc.UpdateRecipient(c.Request.Context(), id, req.Email, req.DisplayName, req.Active, actorID, actorRole)
	if err != nil {
		middleware.RespondError(c, http.StatusBadRequest, middleware.CodeValidationFailed, err.Error(), nil)
		return
	}

	middleware.RespondSuccess(c, http.StatusOK, item, nil)
}

// DeleteRecipient 刪除通知收件人。
func (h *NotificationHandler) DeleteRecipient(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		middleware.RespondError(c, http.StatusBadRequest, middleware.CodeValidationFailed, "無效的收件人 ID", nil)
		return
	}

	actorID := middleware.GetActorID(c)
	actorRole := middleware.GetActorRole(c)

	if err := h.svc.DeleteRecipient(c.Request.Context(), id, actorID, actorRole); err != nil {
		middleware.RespondError(c, http.StatusInternalServerError, middleware.CodeInternalError, "刪除收件人失敗", nil)
		return
	}

	middleware.RespondSuccess(c, http.StatusOK, gin.H{"deleted": true}, nil)
}

// ListLogs 取得通知發送紀錄歷史。
func (h *NotificationHandler) ListLogs(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	if pageSize > 100 {
		pageSize = 100
	}
	topic := c.Query("topic")

	logs, total, err := h.svc.ListLogs(c.Request.Context(), topic, page, pageSize)
	if err != nil {
		middleware.RespondError(c, http.StatusInternalServerError, middleware.CodeInternalError, "查詢通知日誌失敗", nil)
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
