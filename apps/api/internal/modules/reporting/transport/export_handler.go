package transport

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"unicode"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"ltc-system/apps/api/internal/modules/reporting/app"
	"ltc-system/apps/api/internal/platform/auth"
	"ltc-system/apps/api/internal/platform/clock"
	"ltc-system/apps/api/internal/platform/httpx"
)

// currentPeriodYM 以系統目前日期推導民國 5 碼申報月份（RRRMM），
// 取代先前寫死的過期月份字串（過去固定回傳 "11507"，時間一久就與實際月份脫節，
// 使用者若忘記帶 periodYm 會被導去查一個早已結束的月份）。
func currentPeriodYM() string {
	today := clock.Today()
	rocYear := today.Year() - 1911
	return fmt.Sprintf("%03d%02d", rocYear, int(today.Month()))
}

const xlsxContentType = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"

// ExportHandler 處理政府申報匯出與前置檢核請求。
type ExportHandler struct {
	precheckService  *app.PrecheckService
	govClaimService  *app.GovClaimService
	regionClaimSvc   *app.RegionClaimService
	siteTripSummary  *app.SiteTripSummaryService
	claimCaseResolve app.ClaimCaseResolver
}

// NewExportHandler 建立 ExportHandler 實例。
func NewExportHandler(
	precheckService *app.PrecheckService,
	govClaimService *app.GovClaimService,
	regionClaimSvc *app.RegionClaimService,
	siteTripSummary *app.SiteTripSummaryService,
	claimCaseResolve app.ClaimCaseResolver,
) *ExportHandler {
	return &ExportHandler{
		precheckService:  precheckService,
		govClaimService:  govClaimService,
		regionClaimSvc:   regionClaimSvc,
		siteTripSummary:  siteTripSummary,
		claimCaseResolve: claimCaseResolve,
	}
}

// Precheck 執行匯出前置檢核（支援 GET Query 與 POST JSON Body）。
//
// 三組輸入互為相容的擴充：periodYms 未給時退回單一 periodYm；regions 未給時沿用
// caseIds。這樣三個分頁籤共用同一支端點，逐案勾選的既有請求形狀完全不變。
func (h *ExportHandler) Precheck(c *gin.Context) {
	periodYM := c.Query("periodYm")
	if periodYM == "" {
		periodYM = c.DefaultQuery("month", currentPeriodYM())
	}
	periodYMValues := queryList(c, "periodYms")
	caseIDValues := queryList(c, "caseIds")
	regionValues := queryList(c, "regions")

	if c.Request.Method == http.MethodPost {
		var req struct {
			PeriodYM  string   `json:"periodYm"`
			PeriodYMs []string `json:"periodYms"`
			CaseIDs   []string `json:"caseIds"`
			Regions   []string `json:"regions"`
		}
		if err := httpx.BindJSONStrict(c, &req); err != nil {
			httpx.RespondErrorCode(c, http.StatusBadRequest, httpx.CodeValidationFailed, err, httpx.ExtractValidationDetails(err))
			return
		}
		if req.PeriodYM != "" {
			periodYM = req.PeriodYM
		}
		periodYMValues = req.PeriodYMs
		caseIDValues = req.CaseIDs
		regionValues = req.Regions
	}

	if len(periodYMValues) == 0 {
		periodYMValues = []string{periodYM}
	}
	months, err := app.ParseClaimMonths(periodYMValues)
	if err != nil {
		respondExportError(c, err)
		return
	}

	caseIDs, ok := parseUUIDList(c, caseIDValues)
	if !ok {
		return
	}

	// 依區域檢核時，先把區域展開成個案再送進檢核，讓檢核範圍與實際匯出範圍一致。
	if len(regionValues) > 0 {
		cases, err := h.claimCaseResolve.ListCasesByRegions(c.Request.Context(), regionValues)
		if err != nil {
			httpx.RespondErrorCode(c, http.StatusInternalServerError, httpx.CodeInternalError, err, nil)
			return
		}
		// 該區域底下查無個案時，caseIDs 若維持空陣列，SQL 語意上等同「不限個案」，
		// 會變成對全機構跑檢核、顯示一堆與選定區域無關的混車衝突；此處必須直接
		// 視為「沒有可申報的資料」，不能讓空範圍被解讀成不限範圍。
		if len(cases) == 0 {
			httpx.RespondErrorCode(c, http.StatusUnprocessableEntity, httpx.CodeNoExportData, app.ErrNoExportData, nil)
			return
		}
		for _, item := range cases {
			caseIDs = append(caseIDs, item.ID)
		}
	}

	report, err := h.precheckService.RunPrecheckMulti(c.Request.Context(), months, caseIDs)
	if err != nil {
		httpx.RespondErrorCode(c, http.StatusInternalServerError, httpx.CodeInternalError, err, nil)
		return
	}

	httpx.RespondSuccess(c, http.StatusOK, toPrecheckResultResponse(report), nil)
}

// CreateByRegion 依區域批次建立逐月申報匯出工作。
func (h *ExportHandler) CreateByRegion(c *gin.Context) {
	var req createRegionExportRequest
	if err := httpx.BindJSONStrict(c, &req); err != nil {
		httpx.RespondErrorCode(c, http.StatusBadRequest, httpx.CodeValidationFailed, err, httpx.ExtractValidationDetails(err))
		return
	}

	result, err := h.regionClaimSvc.CreateRegionClaimJobs(c.Request.Context(), app.RegionClaimInput{
		Regions:       req.Regions,
		PeriodYMs:     req.PeriodYMs,
		CreatedBy:     auth.GetActorID(c),
		CreatedByName: auth.GetActorName(c),
		ActorRole:     auth.GetActorRole(c),
	})
	if err != nil {
		respondExportError(c, err)
		return
	}

	httpx.RespondSuccess(c, http.StatusAccepted, toRegionExportResultResponse(result), nil)
}

// DownloadBatch 把多筆匯出工作的檔案合併成單一壓縮檔下載。
func (h *ExportHandler) DownloadBatch(c *gin.Context) {
	jobIDs, ok := parseUUIDList(c, queryList(c, "jobIds"))
	if !ok {
		return
	}
	if len(jobIDs) == 0 {
		httpx.RespondErrorCode(c, http.StatusBadRequest, httpx.CodeValidationFailed, app.ErrExportJobNotFound, nil)
		return
	}

	fileName, archive, skippedPeriodYMs, err := h.govClaimService.RenderBatchZip(c.Request.Context(), jobIDs)
	if err != nil {
		respondExportError(c, err)
		return
	}
	// 被略過的月份（尚未完成或失敗）透過自訂標頭揭露，避免使用者收到一包壓縮檔
	// 卻不知道少了哪幾個月——瀏覽器下載回應無法帶 JSON body 說明。
	if len(skippedPeriodYMs) > 0 {
		c.Header("X-Export-Skipped-Periods", strings.Join(skippedPeriodYMs, ","))
	}

	writeAttachment(c, "application/zip", fileName, fileName, archive)
}

// DownloadSiteTripSummary 產生並下載據點趟數彙總表。
//
// 這支端點不建立匯出工作：趟數彙總表是管理用統計，不是報給政府的申報檔，
// 不進 export_jobs 的不可變快照與稽核軌跡（見 SiteTripSummaryService 的說明）。
func (h *ExportHandler) DownloadSiteTripSummary(c *gin.Context) {
	var req siteTripSummaryRequest
	if err := httpx.BindJSONStrict(c, &req); err != nil {
		httpx.RespondErrorCode(c, http.StatusBadRequest, httpx.CodeValidationFailed, err, httpx.ExtractValidationDetails(err))
		return
	}

	siteIDs, ok := parseUUIDList(c, req.SiteIDs)
	if !ok {
		return
	}

	fileName, content, err := h.siteTripSummary.Generate(c.Request.Context(), siteIDs, req.PeriodYMs)
	if err != nil {
		respondExportError(c, err)
		return
	}

	writeAttachment(c, xlsxContentType, fileName, fileName, content)
}

// queryList 讀取重複出現的 query 參數，並相容逗號分隔的單一參數寫法。
func queryList(c *gin.Context, name string) []string {
	values := c.QueryArray(name)
	if len(values) == 1 && strings.Contains(values[0], ",") {
		values = strings.Split(values[0], ",")
	}
	return values
}

// parseUUIDList 解析字串清單為 UUID；空白項目略過，格式錯誤即整批拒絕。
func parseUUIDList(c *gin.Context, raw []string) ([]uuid.UUID, bool) {
	ids := make([]uuid.UUID, 0, len(raw))
	for _, item := range raw {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		id, err := uuid.Parse(item)
		if err != nil {
			httpx.RespondErrorCode(c, http.StatusBadRequest, httpx.CodeValidationFailed, err, nil)
			return nil, false
		}
		ids = append(ids, id)
	}
	return ids, true
}

// List 取得申報匯出工作歷史紀錄清單。
func (h *ExportHandler) List(c *gin.Context) {
	page, pageSize, err := httpx.ParsePagination(c)
	if err != nil {
		httpx.RespondError(c, http.StatusBadRequest, httpx.CodeValidationFailed, "分頁參數格式錯誤", nil)
		return
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
	if err := httpx.BindJSONStrict(c, &req); err != nil {
		httpx.RespondErrorCode(c, http.StatusBadRequest, httpx.CodeValidationFailed, err, httpx.ExtractValidationDetails(err))
		return
	}

	caseIDs, ok := parseUUIDList(c, req.CaseIDs)
	if !ok {
		return
	}

	job, err := h.govClaimService.CreateGovClaimJob(c.Request.Context(), app.CreateGovClaimInput{
		PeriodYM:      req.PeriodYM,
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
