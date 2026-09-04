package transport

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
	"unicode"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xuri/excelize/v2"
	"ltc-system/apps/api/internal/domain/crypto"
	"ltc-system/apps/api/internal/domain/govform"
	"ltc-system/apps/api/internal/modules/reporting/app"
	"ltc-system/apps/api/internal/modules/reporting/infra"
	"ltc-system/apps/api/internal/platform/config"
)

var (
	testKey      = mustDecodeKey("MDEwMjAzMDQwNTA2MDcwODAxMDIwMzA0MDUwNjA3MDg=")
	testCaseID   = uuid.MustParse("11111111-1111-4111-8111-111111111111")
	testCaseID2  = uuid.MustParse("22222222-2222-4222-8222-222222222222")
	testDriverID = uuid.MustParse("33333333-3333-4333-8333-333333333333")
)

func TestExportHandler_CreateReturnsPerCaseDownloadLinks(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h, _ := newTestExportHandler(t, app.GovClaimModeDirect)

	w := performRequest(h, http.MethodPost, "/api/v1/exports", map[string]interface{}{
		"jobType":  "gov_claim",
		"periodYm": "11507",
		"region":   "hsinchu",
		"mode":     "direct",
		"caseIds":  []string{testCaseID.String(), testCaseID2.String()},
	})

	require.Equal(t, http.StatusAccepted, w.Code)

	var resp struct {
		Data exportJobResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))

	assert.Equal(t, "direct", resp.Data.Mode)
	assert.Equal(t, "succeeded", resp.Data.Status)
	assert.Equal(t, 2, resp.Data.TotalCases)
	assert.Equal(t, 3, resp.Data.TotalRows)
	assert.Empty(t, resp.Data.DownloadURL, "逐案下載模式不提供整包下載連結")

	require.Len(t, resp.Data.Files, 2)
	assert.Equal(t, "林大明11507.xlsx", resp.Data.Files[0].FileName)
	assert.Equal(t, 1, resp.Data.Files[0].RowCount)
	assert.Equal(t,
		fmt.Sprintf("/api/v1/exports/%s/files/%s/download", resp.Data.ID, testCaseID2.String()),
		resp.Data.Files[0].DownloadURL,
	)
}

func TestExportHandler_CreateRejectsMissingCaseIDs(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h, _ := newTestExportHandler(t, app.GovClaimModeDirect)

	w := performRequest(h, http.MethodPost, "/api/v1/exports", map[string]interface{}{
		"jobType":  "gov_claim",
		"periodYm": "11507",
		"mode":     "direct",
		"caseIds":  []string{},
	})

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "VALIDATION_FAILED")
}

func TestExportHandler_CreateRejectsUnknownMode(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h, _ := newTestExportHandler(t, app.GovClaimModeDirect)

	w := performRequest(h, http.MethodPost, "/api/v1/exports", map[string]interface{}{
		"periodYm": "11507",
		"mode":     "single_multi_case",
		"caseIds":  []string{testCaseID.String()},
	})

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestExportHandler_CreateBlockedByPrecheck(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := newExportHandlerWithPrecheckFailure(t)

	w := performRequest(h, http.MethodPost, "/api/v1/exports", map[string]interface{}{
		"periodYm": "11507",
		"region":   "hsinchu",
		"mode":     "direct",
		"caseIds":  []string{testCaseID.String()},
	})

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	assert.Contains(t, w.Body.String(), "PRECHECK_FAILED")
}

func TestExportHandler_DownloadCaseFileServesOpenableWorkbook(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h, jobID := newTestExportHandler(t, app.GovClaimModeDirect)

	path := fmt.Sprintf("/api/v1/exports/%s/files/%s/download", jobID, testCaseID)
	w := performRequest(h, http.MethodGet, path, nil)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "spreadsheetml.sheet")

	disposition := w.Header().Get("Content-Disposition")
	assert.Contains(t, disposition, fmt.Sprintf("gov-claim-%s.xlsx", testCaseID.String()[:8]))
	assert.Contains(t, disposition, "filename*=UTF-8''")
	for _, ch := range disposition {
		assert.True(t, ch <= unicode.MaxASCII, "Content-Disposition 應僅含 ASCII: %s", disposition)
	}

	f, err := excelize.OpenReader(bytes.NewReader(w.Body.Bytes()))
	require.NoError(t, err)
	defer f.Close()

	rows, err := f.GetRows(govform.GovClaimSheetName)
	require.NoError(t, err)
	require.Len(t, rows, 3, "1 列標題加 2 列申報資料")
	assert.Equal(t, govform.Headers33[:], rows[0])
	assert.Equal(t, "A202559750", rows[1][0])
	assert.Equal(t, "K120098177", rows[1][6])
}

func TestExportHandler_DownloadCaseFileUnknownCase(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h, jobID := newTestExportHandler(t, app.GovClaimModeDirect)

	path := fmt.Sprintf("/api/v1/exports/%s/files/%s/download", jobID, uuid.New())
	w := performRequest(h, http.MethodGet, path, nil)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), "NOT_FOUND")
}

func TestExportHandler_DownloadZip(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h, jobID := newTestExportHandler(t, app.GovClaimModeZip)

	w := performRequest(h, http.MethodGet, fmt.Sprintf("/api/v1/exports/%s/download", jobID), nil)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/zip", w.Header().Get("Content-Type"))
	assert.Contains(t, w.Header().Get("Content-Disposition"), "gov-claim-hsinchu-11507.zip")

	archive := w.Body.Bytes()
	reader, err := zip.NewReader(bytes.NewReader(archive), int64(len(archive)))
	require.NoError(t, err)
	require.Len(t, reader.File, 2)
	assert.Equal(t, "林大明11507.xlsx", reader.File[0].Name)
}

func TestExportHandler_DownloadRejectsDirectModeJob(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h, jobID := newTestExportHandler(t, app.GovClaimModeDirect)

	w := performRequest(h, http.MethodGet, fmt.Sprintf("/api/v1/exports/%s/download", jobID), nil)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "VALIDATION_FAILED")
}

func TestExportHandler_GetReturnsCaseListForHistory(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h, jobID := newTestExportHandler(t, app.GovClaimModeDirect)

	w := performRequest(h, http.MethodGet, fmt.Sprintf("/api/v1/exports/%s", jobID), nil)

	require.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		Data exportJobResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Len(t, resp.Data.Files, 2)
	assert.Equal(t, "林大明", resp.Data.Files[0].CaseName)
}

func TestExportHandler_ListReturnsPagination(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h, _ := newTestExportHandler(t, app.GovClaimModeDirect)

	w := performRequest(h, http.MethodGet, "/api/v1/exports?page=1&pageSize=10", nil)

	require.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		Data []exportJobResponse `json:"data"`
		Meta struct {
			Page       int   `json:"page"`
			PageSize   int   `json:"pageSize"`
			Total      int64 `json:"total"`
			TotalPages int   `json:"totalPages"`
		} `json:"meta"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 1, resp.Meta.Page)
	assert.Equal(t, int64(1), resp.Meta.Total)
	assert.Equal(t, 1, resp.Meta.TotalPages)
	require.Len(t, resp.Data, 1)
	assert.Empty(t, resp.Data[0].Files, "歷史列表不帶檔案明細")
	assert.Empty(t, resp.Data[0].DownloadURL, "歷史列表不提供下載連結")
}

func TestExportHandler_PrecheckSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h, _ := newTestExportHandler(t, app.GovClaimModeDirect)

	w := performRequest(h, http.MethodPost, "/api/v1/exports/precheck", map[string]interface{}{
		"periodYm": "11507",
		"region":   "hsinchu",
	})

	require.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		Data precheckResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))

	assert.True(t, resp.Data.Passed)
	assert.False(t, resp.Data.HasErrors)
	assert.False(t, resp.Data.HasWarnings)
	assert.Equal(t, 0, resp.Data.Summary.TotalErrors)
	assert.Equal(t, 0, resp.Data.Summary.TotalWarnings)
	assert.Equal(t, 1, resp.Data.Summary.TotalInfos)
	require.Len(t, resp.Data.Items, 1)
	assert.Equal(t, "QUOTA_CHECK_SKIPPED", resp.Data.Items[0].Code)
	assert.Equal(t, "info", resp.Data.Items[0].Level)
}

func TestExportHandler_PrecheckWithIssues(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svcDate := time.Date(2026, 7, 10, 0, 0, 0, 0, time.UTC)
	precheck := app.NewPrecheckService(stubPrecheckRepo{
		incomplete: []app.IncompleteCase{{ID: testCaseID, Name: "張三"}},
		conflicts: []app.UnresolvedConflict{
			{RideID: uuid.New(), CaseName: "張三", ServiceDate: svcDate},
		},
	})
	h := NewExportHandler(precheck, nil)

	w := performRequest(h, http.MethodGet, "/api/v1/exports/precheck?periodYm=11507&region=hsinchu", nil)

	require.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		Data precheckResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))

	assert.False(t, resp.Data.Passed)
	assert.True(t, resp.Data.HasErrors)
	assert.True(t, resp.Data.HasWarnings)
	assert.Equal(t, 1, resp.Data.Summary.TotalErrors)
	assert.Equal(t, 1, resp.Data.Summary.TotalWarnings)
	assert.Equal(t, 1, resp.Data.Summary.TotalInfos)
	require.Len(t, resp.Data.Items, 3)

	// Items 依序包含 Info, Error, Warning
	assert.Equal(t, "QUOTA_CHECK_SKIPPED", resp.Data.Items[0].Code)
	assert.Equal(t, "MISSING_CASE_PROFILE", resp.Data.Items[1].Code)
	assert.Equal(t, "error", resp.Data.Items[1].Level)
	require.Len(t, resp.Data.Items[1].Details, 1)
	assert.Equal(t, "張三", resp.Data.Items[1].Details[0].CaseName)
	assert.Equal(t, testCaseID.String(), resp.Data.Items[1].Details[0].CaseID)

	assert.Equal(t, "UNRESOLVED_CONFLICT", resp.Data.Items[2].Code)
	assert.Equal(t, "warning", resp.Data.Items[2].Level)
	require.Len(t, resp.Data.Items[2].Details, 1)
	assert.Equal(t, "張三", resp.Data.Items[2].Details[0].CaseName)
	assert.Equal(t, "2026-07-10", resp.Data.Items[2].Details[0].ServiceDate)
}

// --- 測試組裝 ---

// newTestExportHandler 用真實的 Excel renderer 與 zip archiver 組出 handler，只把資料來源與
// 儲存層換成記憶體替身，讓 HTTP 回應內容仍是實際會送給使用者的位元組。
func newTestExportHandler(t *testing.T, mode app.GovClaimMode) (*ExportHandler, uuid.UUID) {
	t.Helper()

	store := newMemoryExportStore()
	svc := app.NewGovClaimService(
		&config.Config{EncryptionKey: testKey},
		&stubSourceReader{sources: testSources(t)},
		store,
		infra.NewExcelRenderer(),
		infra.NewZipArchiver(),
		app.NewPrecheckService(stubPrecheckRepo{}),
		discardAuditWriter{},
	)
	h := NewExportHandler(app.NewPrecheckService(stubPrecheckRepo{}), svc)

	job, err := svc.CreateGovClaimJob(context.Background(), app.CreateGovClaimInput{
		PeriodYM:      "11507",
		Region:        "hsinchu",
		CaseIDs:       []uuid.UUID{testCaseID, testCaseID2},
		Mode:          mode,
		CreatedBy:     uuid.New(),
		CreatedByName: "測試操作員",
	})
	require.NoError(t, err)
	return h, job.ID
}

func newExportHandlerWithPrecheckFailure(t *testing.T) *ExportHandler {
	t.Helper()
	precheck := app.NewPrecheckService(stubPrecheckRepo{
		incomplete: []app.IncompleteCase{{ID: testCaseID, Name: "蔡曾切"}},
	})
	svc := app.NewGovClaimService(
		&config.Config{EncryptionKey: testKey},
		&stubSourceReader{},
		newMemoryExportStore(),
		infra.NewExcelRenderer(),
		infra.NewZipArchiver(),
		precheck,
		discardAuditWriter{},
	)
	return NewExportHandler(precheck, svc)
}

func performRequest(h *ExportHandler, method, path string, body interface{}) *httptest.ResponseRecorder {
	r := gin.New()
	r.POST("/api/v1/exports", h.Create)
	r.GET("/api/v1/exports", h.List)
	r.GET("/api/v1/exports/:id", h.Get)
	r.GET("/api/v1/exports/:id/download", h.Download)
	r.GET("/api/v1/exports/:id/files/:caseId/download", h.DownloadCaseFile)
	r.GET("/api/v1/exports/precheck", h.Precheck)
	r.POST("/api/v1/exports/precheck", h.Precheck)

	var reader *bytes.Reader
	if body != nil {
		encoded, _ := json.Marshal(body)
		reader = bytes.NewReader(encoded)
	} else {
		reader = bytes.NewReader(nil)
	}

	req, _ := http.NewRequest(method, path, reader)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// --- 測試替身 ---

type discardAuditWriter struct{}

func (discardAuditWriter) Write(context.Context, app.AuditEntry) error { return nil }

type stubSourceReader struct {
	sources []app.GovClaimSource
}

func (s *stubSourceReader) QueryGovClaimSources(context.Context, time.Time, time.Time, string, []uuid.UUID) ([]app.GovClaimSource, error) {
	return s.sources, nil
}

type stubPrecheckRepo struct {
	incomplete []app.IncompleteCase
	conflicts  []app.UnresolvedConflict
}

func (s stubPrecheckRepo) FindIncompleteActiveCases(context.Context, string) ([]app.IncompleteCase, error) {
	return s.incomplete, nil
}

func (s stubPrecheckRepo) FindUnresolvedConflicts(context.Context, string) ([]app.UnresolvedConflict, error) {
	return s.conflicts, nil
}

// memoryExportStore 模擬 export_jobs／export_lines／export_job_files 的往返，
// 讓下載路徑真的走「讀回快照再重繪」而不是回傳產生當下留在記憶體的位元組。
type memoryExportStore struct {
	job   app.GovClaimJob
	lines map[uuid.UUID][]app.ExportLine
}

func newMemoryExportStore() *memoryExportStore {
	return &memoryExportStore{lines: make(map[uuid.UUID][]app.ExportLine)}
}

func (m *memoryExportStore) CreateJob(_ context.Context, job app.ExportJobCreate) (uuid.UUID, error) {
	mode := app.GovClaimModeDirect
	if job.Format == "zip" {
		mode = app.GovClaimModeZip
	}
	m.job = app.GovClaimJob{
		ID:            uuid.New(),
		JobType:       job.JobType,
		PeriodYM:      job.PeriodYM,
		Region:        job.Region,
		Mode:          mode,
		Status:        app.ExportStatusRunning,
		CreatedBy:     job.CreatedBy,
		CreatedByName: job.CreatedByName,
		CreatedAt:     time.Date(2026, 9, 1, 10, 0, 0, 0, time.UTC),
	}
	return m.job.ID, nil
}

func (m *memoryExportStore) CompleteJob(_ context.Context, _ uuid.UUID, files []app.GovClaimCaseFile, lines []app.ExportLine) error {
	finished := time.Date(2026, 9, 1, 10, 0, 3, 0, time.UTC)
	m.job.Status = app.ExportStatusSucceeded
	m.job.FinishedAt = &finished
	m.job.Files = files
	m.job.TotalCases = len(files)
	m.job.TotalRows = 0
	for _, file := range files {
		m.job.TotalRows += file.RowCount
	}
	for _, line := range lines {
		m.lines[line.CaseID] = append(m.lines[line.CaseID], line)
	}
	return nil
}

func (m *memoryExportStore) FailJob(_ context.Context, _ uuid.UUID, message string) error {
	m.job.Status = app.ExportStatusFailed
	m.job.ErrorMessage = message
	return nil
}

func (m *memoryExportStore) GetJob(_ context.Context, jobID uuid.UUID) (app.GovClaimJob, error) {
	if m.job.ID != jobID {
		return app.GovClaimJob{}, app.ErrExportJobNotFound
	}
	return m.job, nil
}

func (m *memoryExportStore) ListJobs(context.Context, int, int) ([]app.GovClaimJob, int64, error) {
	summary := m.job
	summary.Files = nil
	return []app.GovClaimJob{summary}, 1, nil
}

func (m *memoryExportStore) LoadCaseLines(_ context.Context, _, caseID uuid.UUID) ([]app.ExportLine, error) {
	return m.lines[caseID], nil
}

func (m *memoryExportStore) LoadNationalIDCiphers(context.Context, uuid.UUID, []uuid.UUID) (app.NationalIDCiphers, error) {
	return app.NationalIDCiphers{
		Case:    mustEncrypt("A202559750"),
		Drivers: map[uuid.UUID][]byte{testDriverID: mustEncrypt("K120098177")},
	}, nil
}

func testSources(t *testing.T) []app.GovClaimSource {
	t.Helper()
	return []app.GovClaimSource{
		newTestSource(testCaseID, "蔡曾切", 1, 1, "outbound", "09:40"),
		newTestSource(testCaseID, "蔡曾切", 1, 2, "inbound", "12:20"),
		newTestSource(testCaseID2, "林大明", 1, 1, "outbound", "09:40"),
	}
}

func newTestSource(caseID uuid.UUID, name string, day int, legSeq int16, direction, departTime string) app.GovClaimSource {
	return app.GovClaimSource{
		CaseID:                 caseID,
		CaseName:               name,
		Region:                 "hsinchu",
		CaseNationalIDCipher:   mustEncrypt("A202559750"),
		CaseNationalIDMasked:   "A2****9750",
		HomeAddress:            "新竹縣竹北市光明六路264號",
		ServiceCategory:        intPtr(1),
		ServiceUsageType:       intPtr(2),
		ServiceDate:            time.Date(2026, 7, day, 0, 0, 0, 0, time.UTC),
		LegSeq:                 legSeq,
		Direction:              &direction,
		DepartTime:             &departTime,
		DurationMin:            intPtr(10),
		ServiceCode:            "BD03",
		UnitPrice:              115,
		DistanceKM:             5,
		SiteAddress:            "新竹縣竹北市中正西路100號",
		PlateNo:                "BZG-7915",
		DriverID:               &testDriverID,
		DriverNationalIDCipher: mustEncrypt("K120098177"),
	}
}

func intPtr(v int) *int { return &v }

func mustEncrypt(plain string) []byte {
	cipher, err := crypto.Encrypt(plain, testKey)
	if err != nil {
		panic(err)
	}
	return cipher
}

func mustDecodeKey(b64 string) []byte {
	key, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		panic(err)
	}
	return key
}
