package transport

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"unicode"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"ltc-system/apps/api/internal/modules/reporting/app"
	"ltc-system/apps/api/internal/platform/auth"
	"ltc-system/apps/api/internal/platform/httpx"
)

const xlsxContentType = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"

// ExportHandler 處理政府申報匯出與前置檢核請求。
type ExportHandler struct {
	precheckService *app.PrecheckService
	govClaimService *app.GovClaimService
}

// NewExportHandler 建立 ExportHandler 實例。
func NewExportHandler(precheckService *app.PrecheckService, govClaimService *app.GovClaimService) *ExportHandler {
	return &ExportHandler{precheckService: precheckService, govClaimService: govClaimService}
}

// Precheck 執行匯出前置檢核（支援 GET Query 與 POST JSON Body）。
func (h *ExportHandler) Precheck(c *gin.Context) {
	periodYM := c.Query("periodYm")
	if periodYM == "" {
		periodYM = c.DefaultQuery("month", "11507")
	}
	region := c.Query("region")

	if c.Request.Method == http.MethodPost {
		var req struct {
			PeriodYM string `json:"periodYm"`
			Region   string `json:"region"`
		}
		if err := c.ShouldBindJSON(&req); err == nil {
			if req.PeriodYM != "" {
				periodYM = req.PeriodYM
			}
			region = req.Region
		}
	}

	report, err := h.precheckService.RunPrecheck(c.Request.Context(), periodYM, region)
	if err != nil {
		httpx.RespondErrorCode(c, http.StatusInternalServerError, httpx.CodeInternalError, err, nil)
		return
	}

	httpx.RespondSuccess(c, http.StatusOK, report, nil)
}

// List 取得申報匯出工作歷史紀錄清單。
func (h *ExportHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "10"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}

	jobs, total, err := h.govClaimService.ListExportJobs(c.Request.Context(), page, pageSize)
	if err != nil {
		respondExportError(c, err)
		return
	}

	totalPages := 0
	if pageSize > 0 {
		totalPages = int((total + int64(pageSize) - 1) / int64(pageSize))
	}

	httpx.RespondSuccess(c, http.StatusOK, toExportJobListResponse(jobs), httpx.PaginationMeta{
		Page:       page,
		PageSize:   pageSize,
		Total:      total,
		TotalPages: totalPages,
	})
}

// Create 建立政府申報匯出工作並同步產生逐案工作簿。
func (h *ExportHandler) Create(c *gin.Context) {
	var req createExportJobRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.RespondErrorCode(c, http.StatusBadRequest, httpx.CodeValidationFailed, err, nil)
		return
	}

	caseIDs := make([]uuid.UUID, 0, len(req.CaseIDs))
	for _, raw := range req.CaseIDs {
		id, err := uuid.Parse(raw)
		if err != nil {
			httpx.RespondErrorCode(c, http.StatusBadRequest, httpx.CodeValidationFailed, err, nil)
			return
		}
		caseIDs = append(caseIDs, id)
	}

	job, err := h.govClaimService.CreateGovClaimJob(c.Request.Context(), app.CreateGovClaimInput{
		PeriodYM:      req.PeriodYM,
		Region:        req.Region,
		CaseIDs:       caseIDs,
		Mode:          app.GovClaimMode(req.Mode),
		CreatedBy:     auth.GetActorID(c),
		CreatedByName: auth.GetActorName(c),
		ActorRole:     auth.GetActorRole(c),
	})
	if err != nil {
		respondExportError(c, err)
		return
	}

	httpx.RespondSuccess(c, http.StatusAccepted, toExportJobResponse(job), nil)
}

// Get 取得單筆匯出工作狀態、逐案檔案清單與下載連結。
func (h *ExportHandler) Get(c *gin.Context) {
	jobID, ok := parseUUIDParam(c, "id")
	if !ok {
		return
	}

	job, err := h.govClaimService.GetGovClaimJob(c.Request.Context(), jobID)
	if err != nil {
		respondExportError(c, err)
		return
	}

	httpx.RespondSuccess(c, http.StatusOK, toExportJobResponse(job), nil)
}

// DownloadCaseFile 下載單一個案的申報工作簿，內容由申報列快照重繪。
func (h *ExportHandler) DownloadCaseFile(c *gin.Context) {
	jobID, ok := parseUUIDParam(c, "id")
	if !ok {
		return
	}
	caseID, ok := parseUUIDParam(c, "caseId")
	if !ok {
		return
	}

	file, err := h.govClaimService.RenderCaseFile(c.Request.Context(), jobID, caseID)
	if err != nil {
		respondExportError(c, err)
		return
	}

	writeAttachment(c, xlsxContentType, asciiFileName("gov-claim", file.CaseID.String()[:8], "xlsx"), file.FileName, file.Bytes)
}

// Download 下載整包壓縮檔；逐案下載模式的工作沒有整包檔案，回傳 ErrNotZipJob。
func (h *ExportHandler) Download(c *gin.Context) {
	jobID, ok := parseUUIDParam(c, "id")
	if !ok {
		return
	}

	fileName, archive, err := h.govClaimService.RenderZip(c.Request.Context(), jobID)
	if err != nil {
		respondExportError(c, err)
		return
	}

	writeAttachment(c, "application/zip", fileName, fileName, archive)
}

// writeAttachment 同時提供 ASCII 檔名與 RFC 5987 的 UTF-8 檔名。
// 個案姓名含中文，只給 filename 會讓標頭出現非 ASCII 位元組而被部分代理伺服器截斷。
func writeAttachment(c *gin.Context, contentType, fallbackName, utf8Name string, content []byte) {
	c.Header("Content-Disposition", fmt.Sprintf(
		"attachment; filename=\"%s\"; filename*=UTF-8''%s",
		fallbackName, url.PathEscape(utf8Name),
	))
	c.Data(http.StatusOK, contentType, content)
}

// asciiFileName 以個案編號組出純 ASCII 的備援檔名，供不支援 RFC 5987 的用戶端使用。
// 個案編號理論上是英數字，仍逐字過濾，避免任何非 ASCII 字元寫進標頭。
func asciiFileName(prefix, code, ext string) string {
	safe := make([]rune, 0, len(code))
	for _, ch := range code {
		if ch <= unicode.MaxASCII && (unicode.IsLetter(ch) || unicode.IsDigit(ch) || ch == '-' || ch == '_') {
			safe = append(safe, ch)
		}
	}
	if len(safe) == 0 {
		return fmt.Sprintf("%s.%s", prefix, ext)
	}
	return fmt.Sprintf("%s-%s.%s", prefix, string(safe), ext)
}

func parseUUIDParam(c *gin.Context, name string) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param(name))
	if err != nil {
		httpx.RespondErrorCode(c, http.StatusBadRequest, httpx.CodeValidationFailed, err, nil)
		return uuid.Nil, false
	}
	return id, true
}
