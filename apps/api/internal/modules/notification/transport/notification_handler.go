package transport

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"ltc-system/apps/api/internal/modules/notification/app"
	"ltc-system/apps/api/internal/platform/auth"
	"ltc-system/apps/api/internal/platform/httpx"
)

// NotificationHandler 處理通知收件人與日誌之 HTTP 請求。
type NotificationHandler struct {
	svc *app.NotificationService
}

// NewNotificationHandler 建立 NotificationHandler 實例。
func NewNotificationHandler(svc *app.NotificationService) *NotificationHandler {
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
		httpx.RespondError(c, http.StatusInternalServerError, httpx.CodeInternalError, "查詢收件人失敗", nil)
		return
	}
	httpx.RespondSuccess(c, http.StatusOK, recipients, nil)
}

// CreateRecipient 新增通知收件人（admin 專屬）。
func (h *NotificationHandler) CreateRecipient(c *gin.Context) {
	var req CreateRecipientRequest
	if err := httpx.BindJSONStrict(c, &req); err != nil {
		httpx.RespondErrorCode(c, http.StatusBadRequest, httpx.CodeValidationFailed, err, nil)
		return
	}

	actorID := auth.GetActorID(c)
	actorRole := auth.GetActorRole(c)

	item, err := h.svc.CreateRecipient(c.Request.Context(), req.Topic, req.Email, req.DisplayName, actorID, actorRole)
	if err != nil {
		httpx.RespondErrorCode(c, http.StatusBadRequest, httpx.CodeValidationFailed, err, nil)
		return
	}

	httpx.RespondSuccess(c, http.StatusCreated, item, nil)
}

// UpdateRecipient 修改通知收件人。
func (h *NotificationHandler) UpdateRecipient(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		httpx.RespondError(c, http.StatusBadRequest, httpx.CodeValidationFailed, "無效的收件人 ID", nil)
		return
	}

	var req UpdateRecipientRequest
	if err := httpx.BindJSONStrict(c, &req); err != nil {
		httpx.RespondErrorCode(c, http.StatusBadRequest, httpx.CodeValidationFailed, err, nil)
		return
	}

	actorID := auth.GetActorID(c)
	actorRole := auth.GetActorRole(c)

	item, err := h.svc.UpdateRecipient(c.Request.Context(), id, req.Email, req.DisplayName, req.Active, actorID, actorRole)
	if err != nil {
		if errors.Is(err, app.ErrRecipientNotFound) {
			httpx.RespondError(c, http.StatusNotFound, httpx.CodeNotFound, "查無通知收件人", nil)
			return
		}
		httpx.RespondErrorCode(c, http.StatusBadRequest, httpx.CodeValidationFailed, err, nil)
		return
	}

	httpx.RespondSuccess(c, http.StatusOK, item, nil)
}

// batchCreateRecipientsRequest 定義批次新增收件人請求結構。
type batchCreateRecipientsRequest struct {
	Recipients []struct {
		Topic       string  `json:"topic" binding:"required"`
		Email       string  `json:"email" binding:"required,email"`
		DisplayName *string `json:"displayName"`
	} `json:"recipients" binding:"required,min=1,dive"`
}

// BatchCreateRecipients 批次新增通知收件人。
func (h *NotificationHandler) BatchCreateRecipients(c *gin.Context) {
	var req batchCreateRecipientsRequest
	if err := httpx.BindJSONStrict(c, &req); err != nil {
		httpx.RespondErrorCode(c, http.StatusBadRequest, httpx.CodeValidationFailed, err, nil)
		return
	}

	items := make([]app.BatchRecipientInput, 0, len(req.Recipients))
	for _, r := range req.Recipients {
		items = append(items, app.BatchRecipientInput{Topic: r.Topic, Email: r.Email, DisplayName: r.DisplayName})
	}

	actorID := auth.GetActorID(c)
	actorRole := auth.GetActorRole(c)

	created, err := h.svc.BatchCreateRecipients(c.Request.Context(), items, actorID, actorRole)
	if err != nil {
		httpx.RespondErrorCode(c, http.StatusBadRequest, httpx.CodeValidationFailed, err, nil)
		return
	}

	httpx.RespondSuccess(c, http.StatusCreated, created, nil)
}

// batchDeleteRecipientsRequest 定義批次刪除收件人請求結構。
type batchDeleteRecipientsRequest struct {
	IDs []string `json:"ids" binding:"required,min=1"`
}

// BatchDeleteRecipients 批次刪除通知收件人。
func (h *NotificationHandler) BatchDeleteRecipients(c *gin.Context) {
	var req batchDeleteRecipientsRequest
	if err := httpx.BindJSONStrict(c, &req); err != nil {
		httpx.RespondErrorCode(c, http.StatusBadRequest, httpx.CodeValidationFailed, err, nil)
		return
	}

	ids := make([]int64, 0, len(req.IDs))
	for _, s := range req.IDs {
		id, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			httpx.RespondError(c, http.StatusBadRequest, httpx.CodeValidationFailed, "無效的收件人 ID", nil)
			return
		}
		ids = append(ids, id)
	}

	actorID := auth.GetActorID(c)
	actorRole := auth.GetActorRole(c)

	count, err := h.svc.BatchDeleteRecipients(c.Request.Context(), ids, actorID, actorRole)
	if err != nil {
		httpx.RespondError(c, http.StatusInternalServerError, httpx.CodeInternalError, "批次刪除收件人失敗", nil)
		return
	}

	httpx.RespondSuccess(c, http.StatusOK, gin.H{"count": count}, nil)
}

// DeleteRecipient 刪除通知收件人。
func (h *NotificationHandler) DeleteRecipient(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		httpx.RespondError(c, http.StatusBadRequest, httpx.CodeValidationFailed, "無效的收件人 ID", nil)
		return
	}

	actorID := auth.GetActorID(c)
	actorRole := auth.GetActorRole(c)

	if err := h.svc.DeleteRecipient(c.Request.Context(), id, actorID, actorRole); err != nil {
		if errors.Is(err, app.ErrRecipientNotFound) {
			httpx.RespondError(c, http.StatusNotFound, httpx.CodeNotFound, "查無通知收件人", nil)
			return
		}
		httpx.RespondError(c, http.StatusInternalServerError, httpx.CodeInternalError, "刪除收件人失敗", nil)
		return
	}

	httpx.RespondSuccess(c, http.StatusOK, gin.H{"deleted": true}, nil)
}

// ListLogs 取得通知發送紀錄歷史。
func (h *NotificationHandler) ListLogs(c *gin.Context) {
	page, pageSize, err := httpx.ParsePagination(c)
	if err != nil {
		httpx.RespondError(c, http.StatusBadRequest, httpx.CodeValidationFailed, "分頁參數格式錯誤", nil)
		return
	}
	topic := c.Query("topic")

	logs, total, err := h.svc.ListLogs(c.Request.Context(), topic, page, pageSize)
	if err != nil {
		httpx.RespondError(c, http.StatusInternalServerError, httpx.CodeInternalError, "查詢通知日誌失敗", nil)
		return
	}

	totalPages := int(total) / pageSize
	if int(total)%pageSize != 0 {
		totalPages++
	}

	httpx.RespondSuccess(c, http.StatusOK, logs, httpx.PaginationMeta{
		Page:       page,
		PageSize:   pageSize,
		Total:      total,
		TotalPages: totalPages,
	})
}
