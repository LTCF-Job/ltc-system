package transport

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"ltc-system/apps/api/internal/modules/caregiver/app"
	"ltc-system/apps/api/internal/platform/httpx"
)

// CaregiverHandler 處理照護人員主檔與批次匯入相關請求。
type CaregiverHandler struct {
	svc *app.CaregiverService
}

// NewCaregiverHandler 建立 CaregiverHandler 實例。
func NewCaregiverHandler(svc *app.CaregiverService) *CaregiverHandler {
	return &CaregiverHandler{svc: svc}
}

// List 查詢照護人員清單。
func (h *CaregiverHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	unresolvedLink, _ := strconv.ParseBool(c.DefaultQuery("unresolvedLink", "false"))
	incomplete, _ := strconv.ParseBool(c.DefaultQuery("incomplete", "false"))
	excludePending, _ := strconv.ParseBool(c.DefaultQuery("excludePending", "false"))

	list, total, err := h.svc.List(c.Request.Context(), c.Query("q"), c.Query("status"), unresolvedLink, incomplete, excludePending, page, pageSize)
	if err != nil {
		httpx.RespondError(c, http.StatusInternalServerError, httpx.CodeInternalError, "查詢照護人員失敗", nil)
		return
	}

	httpx.RespondSuccess(c, http.StatusOK, newCaregiverResponses(list), httpx.PaginationMeta{
		Page:     page,
		PageSize: pageSize,
		Total:    total,
	})
}

// Create 新增照護人員。
func (h *CaregiverHandler) Create(c *gin.Context) {
	var req CreateCaregiverRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.RespondErrorCode(c, http.StatusBadRequest, httpx.CodeValidationFailed, err, nil)
		return
	}

	caregiver, err := h.svc.Create(c.Request.Context(), app.CreateCaregiverInput{
		SiteID:  req.SiteID,
		Name:    req.Name,
		Type:    req.Type,
		Contact: req.Contact,
		Notes:   req.Notes,
		Status:  req.Status,
	})
	if err != nil {
		httpx.RespondErrorCode(c, http.StatusBadRequest, httpx.CodeValidationFailed, err, nil)
		return
	}

	httpx.RespondSuccess(c, http.StatusCreated, newCaregiverResponse(*caregiver), nil)
}

// Update 更新照護人員。
func (h *CaregiverHandler) Update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondInvalidID(c, "無效的照護人員 ID")
		return
	}

	var req UpdateCaregiverRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.RespondErrorCode(c, http.StatusBadRequest, httpx.CodeValidationFailed, err, nil)
		return
	}

	caregiver, err := h.svc.Update(c.Request.Context(), id, app.UpdateCaregiverInput{
		SiteID:  req.SiteID,
		Name:    req.Name,
		Type:    req.Type,
		Contact: req.Contact,
		Notes:   req.Notes,
		Status:  req.Status,
	})
	if err != nil {
		httpx.RespondErrorCode(c, http.StatusBadRequest, httpx.CodeValidationFailed, err, nil)
		return
	}

	httpx.RespondSuccess(c, http.StatusOK, newCaregiverResponse(*caregiver), nil)
}

// Delete 刪除照護人員。
func (h *CaregiverHandler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondInvalidID(c, "無效的照護人員 ID")
		return
	}

	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		httpx.RespondErrorCode(c, http.StatusBadRequest, httpx.CodeValidationFailed, err, nil)
		return
	}

	httpx.RespondSuccess(c, http.StatusNoContent, nil, nil)
}

// LinkSite 將單位待關聯的照護人員連結至既有單位。
func (h *CaregiverHandler) LinkSite(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondInvalidID(c, "無效的照護人員 ID")
		return
	}

	var req LinkCaregiverSiteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.RespondErrorCode(c, http.StatusBadRequest, httpx.CodeValidationFailed, err, nil)
		return
	}

	caregiver, err := h.svc.LinkSite(c.Request.Context(), id, req.SiteID)
	if err != nil {
		httpx.RespondErrorCode(c, http.StatusBadRequest, httpx.CodeValidationFailed, err, nil)
		return
	}

	httpx.RespondSuccess(c, http.StatusOK, newCaregiverResponse(*caregiver), nil)
}

// ImportExcel 批次上傳解析照護人員新增資料 Excel 檔案。
func (h *CaregiverHandler) ImportExcel(c *gin.Context) {
	fileHeader, ok := httpx.BindUploadFile(c, "file")
	if !ok {
		return
	}

	f, err := fileHeader.Open()
	if err != nil {
		httpx.RespondError(c, http.StatusBadRequest, httpx.CodeValidationFailed, "無法開啟檔案", nil)
		return
	}
	defer f.Close()

	preview, err := h.svc.ParseCaregivers(c.Request.Context(), f, fileHeader.Filename)
	if err != nil {
		httpx.RespondErrorCode(c, http.StatusBadRequest, httpx.CodeValidationFailed, err, nil)
		return
	}

	if c.DefaultQuery("dryRun", "true") == "false" {
		includeDuplicateRows, err := parseIncludeDuplicateRows(c.PostForm("includeDuplicateRows"))
		if err != nil {
			httpx.RespondError(c, http.StatusBadRequest, httpx.CodeValidationFailed, "includeDuplicateRows 格式錯誤", nil)
			return
		}

		result, err := h.svc.CommitCaregivers(c.Request.Context(), preview, includeDuplicateRows)
		if err != nil {
			httpx.RespondError(c, http.StatusInternalServerError, httpx.CodeInternalError, "匯入照護人員寫入失敗", nil)
			return
		}
		httpx.RespondSuccess(c, http.StatusOK, result, nil)
		return
	}

	httpx.RespondSuccess(c, http.StatusOK, preview, nil)
}

// parseIncludeDuplicateRows 解析使用者於預覽階段勾選「仍要匯入」的列號 JSON 陣列
// （如 "[3,7]"）；空字串視為未勾選任何列。
func parseIncludeDuplicateRows(raw string) (map[string]bool, error) {
	set := map[string]bool{}
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return set, nil
	}
	var rowIDs []string
	if err := json.Unmarshal([]byte(raw), &rowIDs); err != nil {
		return nil, err
	}
	for _, rowID := range rowIDs {
		if strings.TrimSpace(rowID) == "" {
			return nil, fmt.Errorf("rowId 不可為空")
		}
		set[rowID] = true
	}
	return set, nil
}

// DownloadTemplate 下載照護人員批次匯入範本。
func (h *CaregiverHandler) DownloadTemplate(c *gin.Context) {
	excelBytes, err := h.svc.CaregiverImportTemplateExcel()
	if err != nil {
		httpx.RespondError(c, http.StatusInternalServerError, httpx.CodeInternalError, "產生 Excel 範本失敗", nil)
		return
	}

	asciiName := "caregiver_template.xlsx"
	utf8Name := "照護人員批次匯入範本.xlsx"
	c.Header("Content-Disposition", `attachment; filename="`+asciiName+`"; filename*=UTF-8''`+url.PathEscape(utf8Name))
	c.Data(http.StatusOK, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", excelBytes)
}
