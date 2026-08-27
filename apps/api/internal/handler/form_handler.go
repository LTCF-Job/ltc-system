package handler

import (
	"context"
	"net/http"

	"ltc-system/apps/api/internal/middleware"
	"ltc-system/apps/api/internal/service"

	"github.com/gin-gonic/gin"
)

// FormServiceInterface 定義 FormHandler 所需的業務服務介面。
type FormServiceInterface interface {
	ListForms(ctx context.Context) ([]service.FormListItemDTO, error)
	ListGoogleDriveFiles(ctx context.Context) ([]service.GoogleDriveFileDTO, error)
	InspectGoogleSheet(ctx context.Context, inputURLOrID string) (*service.InspectSheetDTO, error)
	CreateFormAssociation(ctx context.Context, req service.CreateFormAssociationRequest) (*service.FormListItemDTO, error)
	DeleteFormAssociation(ctx context.Context, formID string) error
	SyncForm(ctx context.Context, formID string, opts *service.SyncFormOptions) (map[string]interface{}, error)
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

// ListGoogleDriveFiles 取得 Google 雲端硬碟中的試算表清單。
func (h *FormHandler) ListGoogleDriveFiles(c *gin.Context) {
	files, err := h.svc.ListGoogleDriveFiles(c.Request.Context())
	if err != nil {
		middleware.RespondError(c, http.StatusInternalServerError, "FAILED_TO_LIST_DRIVE_FILES", err.Error(), nil)
		return
	}
	middleware.RespondSuccess(c, http.StatusOK, files, nil)
}

// InspectGoogleSheet 解析特定試算表的分頁與欄位結構。
func (h *FormHandler) InspectGoogleSheet(c *gin.Context) {
	var req struct {
		SheetURL      string `json:"sheetUrl"`
		SpreadsheetID string `json:"spreadsheetId"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.RespondError(c, http.StatusBadRequest, "INVALID_PAYLOAD", "請提供有效之試算表連結或 ID", nil)
		return
	}

	target := req.SheetURL
	if target == "" {
		target = req.SpreadsheetID
	}

	result, err := h.svc.InspectGoogleSheet(c.Request.Context(), target)
	if err != nil {
		middleware.RespondError(c, http.StatusBadRequest, "INSPECT_SHEET_FAILED", err.Error(), nil)
		return
	}
	middleware.RespondSuccess(c, http.StatusOK, result, nil)
}

// CreateFormAssociation 建立表單與 Google 試算表關聯。
func (h *FormHandler) CreateFormAssociation(c *gin.Context) {
	var req service.CreateFormAssociationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.RespondError(c, http.StatusBadRequest, "INVALID_PAYLOAD", err.Error(), nil)
		return
	}

	form, err := h.svc.CreateFormAssociation(c.Request.Context(), req)
	if err != nil {
		middleware.RespondError(c, http.StatusBadRequest, "FAILED_TO_CREATE_ASSOCIATION", err.Error(), nil)
		return
	}
	middleware.RespondSuccess(c, http.StatusCreated, form, nil)
}

// DeleteFormAssociation 解除表單關聯。
func (h *FormHandler) DeleteFormAssociation(c *gin.Context) {
	formID := c.Param("id")
	if err := h.svc.DeleteFormAssociation(c.Request.Context(), formID); err != nil {
		middleware.RespondError(c, http.StatusInternalServerError, "FAILED_TO_DELETE_ASSOCIATION", err.Error(), nil)
		return
	}
	middleware.RespondSuccess(c, http.StatusOK, gin.H{"success": true}, nil)
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
	var opts service.SyncFormOptions
	_ = c.ShouldBindJSON(&opts) // 選填

	res, err := h.svc.SyncForm(c.Request.Context(), formID, &opts)
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
