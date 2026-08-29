package transport

import (
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

// ImportExcel 批次上傳解析個案新增資料 Excel 或 CSV 檔案。
func (h *ImportHandler) ImportExcel(c *gin.Context) {
	fileHeader, err := c.FormFile("file")
	if err != nil {
		httpx.RespondError(c, http.StatusBadRequest, httpx.CodeValidationFailed, "未提供上傳檔案", nil)
		return
	}

	f, err := fileHeader.Open()
	if err != nil {
		httpx.RespondError(c, http.StatusBadRequest, httpx.CodeValidationFailed, "無法開啟檔案", nil)
		return
	}
	defer f.Close()

	preview, err := h.svc.ParseCases(f, fileHeader.Filename)
	if err != nil {
		httpx.RespondErrorCode(c, http.StatusBadRequest, httpx.CodeValidationFailed, err, nil)
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

// DownloadTemplate 下載個案批次匯入範本 (支援 .xlsx 與 .csv)。
func (h *ImportHandler) DownloadTemplate(c *gin.Context) {
	if strings.ToLower(c.DefaultQuery("format", "xlsx")) == "csv" {
		attachAs(c, "case_template.csv", "個案批次匯入範本.csv")
		c.Data(http.StatusOK, "text/csv; charset=utf-8", []byte(app.GenerateCaseImportTemplateCSV()))
		return
	}

	excelBytes, err := h.svc.CaseImportTemplateExcel()
	if err != nil {
		httpx.RespondError(c, http.StatusInternalServerError, httpx.CodeInternalError, "產生 Excel 範本失敗", nil)
		return
	}

	attachAs(c, "case_template.xlsx", "個案批次匯入範本.xlsx")
	c.Data(http.StatusOK, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", excelBytes)
}

// attachAs 同時給出 ASCII 後備檔名與 UTF-8 檔名，讓舊瀏覽器不致收到亂碼。
func attachAs(c *gin.Context, asciiName, utf8Name string) {
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%q; filename*=UTF-8''%s", asciiName, url.PathEscape(utf8Name)))
}
