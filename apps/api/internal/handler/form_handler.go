package handler

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"ltc-system/apps/api/internal/middleware"
	"ltc-system/apps/api/internal/service"
)

// FormServiceInterface 定義 FormHandler 所需的業務服務介面。
type FormServiceInterface interface {
	ListForms(ctx context.Context) ([]service.FormListItemDTO, error)
	SyncForm(ctx context.Context, formID string) (map[string]interface{}, error)
	ListColumns(ctx context.Context, mappingStatus string) ([]service.FormColumnDTO, error)
	UpdateColumnMapping(ctx context.Context, colID string, status string, caseID *string, legSeq *int16) error
	BatchMapping(ctx context.Context, mappings []service.ColumnMappingUpdate) (int, error)
}

// FormHandler 處理 Google 表單同步與欄位對應之 HTTP 請求。
type FormHandler struct {
	svc FormServiceInterface
}

// NewFormHandler 建立 FormHandler 實例。
func NewFormHandler(svc FormServiceInterface) *FormHandler {
	return &FormHandler{svc: svc}
}

// ListForms 取得 Google 表單清單。
func (h *FormHandler) ListForms(c *gin.Context) {
	forms, err := h.svc.ListForms(c.Request.Context())
	if err != nil {
		middleware.RespondError(c, http.StatusInternalServerError, "FAILED_TO_LIST_FORMS", err.Error(), nil)
		return
	}
	middleware.RespondSuccess(c, http.StatusOK, forms, nil)
}

// SyncForm 手動觸發表單同步。
func (h *FormHandler) SyncForm(c *gin.Context) {
	formID := c.Param("id")
	res, err := h.svc.SyncForm(c.Request.Context(), formID)
	if err != nil {
		middleware.RespondError(c, http.StatusInternalServerError, "SYNC_FAILED", err.Error(), nil)
		return
	}
	middleware.RespondSuccess(c, http.StatusOK, res, nil)
}

// ListColumns 取得表單欄位對應清單。
func (h *FormHandler) ListColumns(c *gin.Context) {
	status := c.Query("mappingStatus")
	cols, err := h.svc.ListColumns(c.Request.Context(), status)
	if err != nil {
		middleware.RespondError(c, http.StatusInternalServerError, "FAILED_TO_LIST_COLUMNS", err.Error(), nil)
		return
	}
	middleware.RespondSuccess(c, http.StatusOK, cols, nil)
}

// UpdateColumnMapping 綁定或略過欄位對應。
func (h *FormHandler) UpdateColumnMapping(c *gin.Context) {
	colID := c.Param("id")
	var req struct {
		MappingStatus string  `json:"mappingStatus"`
		CaseID        *string `json:"caseId"`
		LegSeq        *int16  `json:"legSeq"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.RespondError(c, http.StatusBadRequest, "INVALID_PAYLOAD", err.Error(), nil)
		return
	}

	if err := h.svc.UpdateColumnMapping(c.Request.Context(), colID, req.MappingStatus, req.CaseID, req.LegSeq); err != nil {
		middleware.RespondError(c, http.StatusInternalServerError, "FAILED_TO_UPDATE_MAPPING", err.Error(), nil)
		return
	}

	middleware.RespondSuccess(c, http.StatusOK, gin.H{
		"id":            colID,
		"mappingStatus": req.MappingStatus,
		"caseId":        req.CaseID,
		"legSeq":        req.LegSeq,
	}, nil)
}

// BatchMapping 批次對應多個欄位。
func (h *FormHandler) BatchMapping(c *gin.Context) {
	var req struct {
		Mappings []service.ColumnMappingUpdate `json:"mappings"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.RespondError(c, http.StatusBadRequest, "INVALID_PAYLOAD", err.Error(), nil)
		return
	}

	updatedCount, err := h.svc.BatchMapping(c.Request.Context(), req.Mappings)
	if err != nil {
		middleware.RespondError(c, http.StatusInternalServerError, "FAILED_TO_BATCH_MAPPING", err.Error(), nil)
		return
	}

	middleware.RespondSuccess(c, http.StatusOK, gin.H{
		"updatedCount": updatedCount,
	}, nil)
}
