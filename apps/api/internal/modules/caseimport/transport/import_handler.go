package transport

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"

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
		httpx.RespondErrorCode(c, http.StatusBadRequest, httpx.CodeFileUnreadable, err, []httpx.ErrorDetail{
			{Field: "file", Reason: "檔案無法開啟，請重新選擇檔案後再上傳"},
		})
		return
	}
	defer f.Close()

	preview, err := h.svc.ParseCases(c.Request.Context(), f, fileHeader.Filename)
	if err != nil {
		respondParseError(c, err)
		return
	}

	// 依 dryRun 參數區分預覽或正式寫入
	if c.DefaultQuery("dryRun", "true") == "false" {
		result, err := h.svc.CommitCases(c.Request.Context(), preview, app.Actor{
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

// respondParseError 把解析前置失敗分流到各自的錯誤碼並附上可行動的原因。
// 全部壓成 VALIDATION_FAILED 只會顯示「輸入資料不符合規則」，使用者無從判斷是檔案格式、
// 檔案損毀，還是套用了錯誤的範本。
func respondParseError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, app.ErrUnsupportedFileType):
		httpx.RespondErrorCode(c, http.StatusBadRequest, httpx.CodeUnsupportedFileType, err, []httpx.ErrorDetail{
			{Field: "file", Reason: "僅支援 .xlsx 檔案，請另存為 Excel 活頁簿後再上傳"},
		})
	case errors.Is(err, app.ErrTemplateMismatch):
		httpx.RespondErrorCode(c, http.StatusBadRequest, httpx.CodeImportTemplateMismatch, err, []httpx.ErrorDetail{
			{Field: "file", Reason: "找不到「姓名」欄，請確認是否使用個案批次匯入範本，且標題列在前三列內"},
		})
	case errors.Is(err, app.ErrFileUnreadable):
		httpx.RespondErrorCode(c, http.StatusBadRequest, httpx.CodeFileUnreadable, err, []httpx.ErrorDetail{
			{Field: "file", Reason: "檔案內容無法讀取，可能已損毀或不是有效的 Excel 檔"},
		})
	default:
		httpx.RespondErrorCode(c, http.StatusBadRequest, httpx.CodeValidationFailed, err, nil)
	}
}

// attachAs 同時給出 ASCII 後備檔名與 UTF-8 檔名，讓舊瀏覽器不致收到亂碼。
func attachAs(c *gin.Context, asciiName, utf8Name string) {
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%q; filename*=UTF-8''%s", asciiName, url.PathEscape(utf8Name)))
}
