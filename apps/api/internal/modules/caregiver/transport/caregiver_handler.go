package transport

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"ltc-system/apps/api/internal/modules/caregiver/app"
	"ltc-system/apps/api/internal/platform/auth"
	"ltc-system/apps/api/internal/platform/httpx"
)

// PendingRelinker 讓照護人員新增或改名後，重新比對名稱相符的待維護個案；
// 未接線時傳 nil，Create／Update 會略過重新比對。
type PendingRelinker interface {
	RelinkByName(ctx context.Context, name string, actorID uuid.UUID, actorRole, ip, ua string) (int, error)
}

// CaregiverHandler 處理照護人員主檔與批次匯入相關請求。
type CaregiverHandler struct {
	svc      *app.CaregiverService
	relinker PendingRelinker
}

// NewCaregiverHandler 建立 CaregiverHandler 實例。
func NewCaregiverHandler(svc *app.CaregiverService, relinkers ...PendingRelinker) *CaregiverHandler {
	h := &CaregiverHandler{svc: svc}
	if len(relinkers) > 0 {
		h.relinker = relinkers[0]
	}
	return h
}

// relinkPendingMeta 呼叫 relinker 重新比對待維護資料；筆數為 0 時回傳 nil，
// 讓 RespondSuccess 的 meta 維持既有形狀，不多長一個恆為 0 的欄位。
func (h *CaregiverHandler) relinkPendingMeta(c *gin.Context, name string) any {
	if h.relinker == nil || name == "" {
		return nil
	}
	actor := actorOf(c)
	n, err := h.relinker.RelinkByName(c.Request.Context(), name, actor.ActorID, actor.ActorRole, actor.IPAddress, actor.UserAgent)
	if err != nil {
		slog.Error("pending_relink_failed", slog.String("name", name), slog.Any("error", err))
		return nil
	}
	if n == 0 {
		return nil
	}
	return gin.H{"pendingRelinked": n}
}

func actorOf(c *gin.Context) app.ActorContext {
	return app.ActorContext{
		ActorID:   auth.GetActorID(c),
		ActorRole: auth.GetActorRole(c),
		IPAddress: c.ClientIP(),
		UserAgent: c.Request.UserAgent(),
	}
}

// List 查詢照護人員清單。
func (h *CaregiverHandler) List(c *gin.Context) {
	page, pageSize, err := httpx.ParsePagination(c)
	if err != nil {
		httpx.RespondErrorCode(c, http.StatusBadRequest, httpx.CodeValidationFailed, err, nil)
		return
	}
	// 待維護資料預設不出現在任何清單，呼叫端要明確表態才拿得到：pending 只取待維護，
	// includePending 取全部。預設排除，避免新增呼叫端忘記帶參數就把待維護資料洩漏出去。
	pending := httpx.QueryBool(c, "pending")
	includePending := httpx.QueryBool(c, "includePending")
	excludePending := !pending && !includePending

	list, total, err := h.svc.List(c.Request.Context(), c.Query("q"), c.Query("status"), pending, excludePending, page, pageSize)
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
		httpx.RespondErrorCode(c, http.StatusBadRequest, httpx.CodeValidationFailed, err, httpx.ExtractValidationDetails(err))
		return
	}

	caregiver, err := h.svc.Create(c.Request.Context(), app.CreateCaregiverInput{
		SiteName: req.SiteName,
		Name:     req.Name,
		Type:     req.Type,
		Contact:  req.Contact,
		Notes:    req.Notes,
		Status:   req.Status,
	}, actorOf(c))
	if err != nil {
		httpx.RespondErrorCode(c, http.StatusBadRequest, httpx.CodeValidationFailed, err, nil)
		return
	}

	meta := h.relinkPendingMeta(c, caregiver.Name)
	httpx.RespondSuccess(c, http.StatusCreated, newCaregiverResponse(*caregiver), meta)
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
		httpx.RespondErrorCode(c, http.StatusBadRequest, httpx.CodeValidationFailed, err, httpx.ExtractValidationDetails(err))
		return
	}

	caregiver, err := h.svc.Update(c.Request.Context(), id, app.UpdateCaregiverInput{
		SiteName: req.SiteName,
		Name:     req.Name,
		Type:     req.Type,
		Contact:  req.Contact,
		Notes:    req.Notes,
		Status:   req.Status,
	}, actorOf(c))
	if err != nil {
		httpx.RespondErrorCode(c, http.StatusBadRequest, httpx.CodeValidationFailed, err, nil)
		return
	}

	var meta any
	if req.Name != nil {
		meta = h.relinkPendingMeta(c, caregiver.Name)
	}
	httpx.RespondSuccess(c, http.StatusOK, newCaregiverResponse(*caregiver), meta)
}

// Delete 刪除照護人員。
func (h *CaregiverHandler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondInvalidID(c, "無效的照護人員 ID")
		return
	}

	if err := h.svc.Delete(c.Request.Context(), id, actorOf(c)); err != nil {
		httpx.RespondErrorCode(c, http.StatusBadRequest, httpx.CodeValidationFailed, err, nil)
		return
	}

	httpx.RespondSuccess(c, http.StatusNoContent, nil, nil)
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

		result, err := h.svc.CommitCaregivers(c.Request.Context(), preview, includeDuplicateRows, actorOf(c))
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
