package transport

import (
	"context"
	"net/http"

	"ltc-system/apps/api/internal/modules/formsync/app"
	"ltc-system/apps/api/internal/platform/httpx"

	"github.com/gin-gonic/gin"
)

// FormServiceInterface 定義 FormHandler 所需的業務服務介面。
type FormServiceInterface interface {
	ListForms(ctx context.Context) ([]app.FormListItemDTO, error)
	ListGoogleDriveFiles(ctx context.Context) ([]app.GoogleDriveFileDTO, error)
	InspectGoogleSheet(ctx context.Context, inputURLOrID string, accessToken string) (*app.InspectSheetDTO, error)
	CreateFormAssociation(ctx context.Context, req app.CreateFormAssociationRequest) (*app.FormListItemDTO, error)
	DeleteFormAssociation(ctx context.Context, formID string) error
	SyncForm(ctx context.Context, formID string, opts *app.SyncFormOptions) (map[string]interface{}, error)
	ListColumns(ctx context.Context, mappingStatus string) ([]app.FormColumnDTO, error)
	UpdateColumnMapping(ctx context.Context, colID string, status string, caseID *string, legSeq *int16) error
	BatchMapping(ctx context.Context, mappings []app.ColumnMappingUpdate) (int, error)
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
		httpx.RespondErrorCode(c, http.StatusInternalServerError, httpx.CodeFormSourceFailed, err, nil)
		return
	}
	httpx.RespondSuccess(c, http.StatusOK, files, nil)
}

// InspectGoogleSheet 解析特定試算表的分頁與欄位結構。
func (h *FormHandler) InspectGoogleSheet(c *gin.Context) {
	var req struct {
		SheetURL      string `json:"sheetUrl"`
		SpreadsheetID string `json:"spreadsheetId"`
		AccessToken   string `json:"accessToken"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.RespondError(c, http.StatusBadRequest, httpx.CodeValidationFailed, "請提供有效之試算表連結或 ID", nil)
		return
	}

	target := req.SheetURL
	if target == "" {
		target = req.SpreadsheetID
	}

	result, err := h.svc.InspectGoogleSheet(c.Request.Context(), target, req.AccessToken)
	if err != nil {
		httpx.RespondErrorCode(c, http.StatusBadRequest, httpx.CodeFormSourceFailed, err, nil)
		return
	}
	httpx.RespondSuccess(c, http.StatusOK, result, nil)
}

// CreateFormAssociation 建立表單與 Google 試算表關聯。
func (h *FormHandler) CreateFormAssociation(c *gin.Context) {
	var req app.CreateFormAssociationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.RespondErrorCode(c, http.StatusBadRequest, httpx.CodeValidationFailed, err, nil)
		return
	}

	form, err := h.svc.CreateFormAssociation(c.Request.Context(), req)
	if err != nil {
		httpx.RespondErrorCode(c, http.StatusBadRequest, httpx.CodeFormMappingFailed, err, nil)
		return
	}
	httpx.RespondSuccess(c, http.StatusCreated, form, nil)
}

// DeleteFormAssociation 解除表單關聯。
func (h *FormHandler) DeleteFormAssociation(c *gin.Context) {
	formID := c.Param("id")
	if err := h.svc.DeleteFormAssociation(c.Request.Context(), formID); err != nil {
		httpx.RespondErrorCode(c, http.StatusInternalServerError, httpx.CodeFormMappingFailed, err, nil)
		return
	}
	httpx.RespondSuccess(c, http.StatusOK, gin.H{"success": true}, nil)
}

// ListForms 取得 Google 表單清單。
func (h *FormHandler) ListForms(c *gin.Context) {
	forms, err := h.svc.ListForms(c.Request.Context())
	if err != nil {
		httpx.RespondErrorCode(c, http.StatusInternalServerError, httpx.CodeFormSyncFailed, err, nil)
		return
	}
	httpx.RespondSuccess(c, http.StatusOK, forms, nil)
}

// SyncForm 手動觸發表單同步。
func (h *FormHandler) SyncForm(c *gin.Context) {
	formID := c.Param("id")
	var opts app.SyncFormOptions
	_ = c.ShouldBindJSON(&opts) // 選填

	res, err := h.svc.SyncForm(c.Request.Context(), formID, &opts)
	if err != nil {
		httpx.RespondErrorCode(c, http.StatusInternalServerError, httpx.CodeFormSyncFailed, err, nil)
		return
	}
	httpx.RespondSuccess(c, http.StatusOK, res, nil)
}

// ListColumns 取得表單欄位對應清單。
func (h *FormHandler) ListColumns(c *gin.Context) {
	status := c.Query("mappingStatus")
	cols, err := h.svc.ListColumns(c.Request.Context(), status)
	if err != nil {
		httpx.RespondErrorCode(c, http.StatusInternalServerError, httpx.CodeFormSyncFailed, err, nil)
		return
	}
	httpx.RespondSuccess(c, http.StatusOK, cols, nil)
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
		httpx.RespondErrorCode(c, http.StatusBadRequest, httpx.CodeValidationFailed, err, nil)
		return
	}

	if err := h.svc.UpdateColumnMapping(c.Request.Context(), colID, req.MappingStatus, req.CaseID, req.LegSeq); err != nil {
		httpx.RespondErrorCode(c, http.StatusInternalServerError, httpx.CodeFormMappingFailed, err, nil)
		return
	}

	httpx.RespondSuccess(c, http.StatusOK, gin.H{
		"id":            colID,
		"mappingStatus": req.MappingStatus,
		"caseId":        req.CaseID,
		"legSeq":        req.LegSeq,
	}, nil)
}

// BatchMapping 批次對應多個欄位。
func (h *FormHandler) BatchMapping(c *gin.Context) {
	var req struct {
		Mappings []app.ColumnMappingUpdate `json:"mappings"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.RespondErrorCode(c, http.StatusBadRequest, httpx.CodeValidationFailed, err, nil)
		return
	}

	updatedCount, err := h.svc.BatchMapping(c.Request.Context(), req.Mappings)
	if err != nil {
		httpx.RespondErrorCode(c, http.StatusInternalServerError, httpx.CodeFormMappingFailed, err, nil)
		return
	}

	httpx.RespondSuccess(c, http.StatusOK, gin.H{
		"updatedCount": updatedCount,
	}, nil)
}
