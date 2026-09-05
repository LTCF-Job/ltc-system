package transport

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
	"ltc-system/apps/api/internal/modules/caseimport/app"
	"ltc-system/apps/api/internal/platform/auth"
	"ltc-system/apps/api/internal/platform/httpx"
)

// ImportHandler 處理個案批次匯入與範本下載請求。
type ImportHandler struct {
	svc *app.ImportService
}

// NewImportHandler 建立 ImportHandler 實例。
func NewImportHandler(svc *app.ImportService) *ImportHandler {
	return &ImportHandler{svc: svc}
}

// ImportExcel 批次上傳解析個案新增資料 Excel 檔案。
func (h *ImportHandler) ImportExcel(c *gin.Context) {
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

	preview, err := h.svc.ParseCases(c.Request.Context(), f, fileHeader.Filename)
	if err != nil {
		httpx.RespondErrorCode(c, http.StatusBadRequest, httpx.CodeValidationFailed, err, nil)
		return
	}

	// 依 dryRun 參數區分預覽或正式寫入
	if c.DefaultQuery("dryRun", "true") == "false" {
		includeDuplicateRows, err := parseIncludeDuplicateRows(c.PostForm("includeDuplicateRows"))
		if err != nil {
			httpx.RespondError(c, http.StatusBadRequest, httpx.CodeValidationFailed, "includeDuplicateRows 格式錯誤", nil)
			return
		}

		result, err := h.svc.CommitCases(c.Request.Context(), preview, includeDuplicateRows, app.Actor{
			ActorID:   auth.GetActorID(c),
			ActorRole: auth.GetActorRole(c),
			IPAddress: c.ClientIP(),
			UserAgent: c.Request.UserAgent(),
		})
		if err != nil {
			httpx.RespondError(c, http.StatusInternalServerError, httpx.CodeInternalError, "匯入個案寫入失敗", nil)
			return
		}
		httpx.RespondSuccess(c, http.StatusOK, result, nil)
		return
	}

	httpx.RespondSuccess(c, http.StatusOK, preview, nil)
}

// DownloadTemplate 下載個案批次匯入範本 (.xlsx)。
func (h *ImportHandler) DownloadTemplate(c *gin.Context) {
	excelBytes, err := h.svc.CaseImportTemplateExcel()
	if err != nil {
		httpx.RespondError(c, http.StatusInternalServerError, httpx.CodeInternalError, "產生 Excel 範本失敗", nil)
		return
	}

	attachAs(c, "case_template.xlsx", "個案批次匯入範本.xlsx")
	c.Data(http.StatusOK, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", excelBytes)
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

// attachAs 同時給出 ASCII 後備檔名與 UTF-8 檔名，讓舊瀏覽器不致收到亂碼。
func attachAs(c *gin.Context, asciiName, utf8Name string) {
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%q; filename*=UTF-8''%s", asciiName, url.PathEscape(utf8Name)))
}
