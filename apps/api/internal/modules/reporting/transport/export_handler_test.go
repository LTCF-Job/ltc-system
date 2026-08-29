package transport

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"unicode"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xuri/excelize/v2"
	"ltc-system/apps/api/internal/modules/reporting/app"
	"ltc-system/apps/api/internal/modules/reporting/infra"
)

func TestExportHandler_Download(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := newTestExportHandler()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest(http.MethodGet, "/api/v1/exports/test_job_123/download?periodYm=115-07&region=hsinchu", nil)

	h.Download(c)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "spreadsheetml.sheet")

	disposition := w.Header().Get("Content-Disposition")
	assert.Contains(t, disposition, "gov-claim-hsinchu-11507.xlsx")
	assert.Contains(t, disposition, "filename*=UTF-8''")

	// 驗證標頭無非 ASCII 字元
	for _, ch := range disposition {
		assert.True(t, ch <= unicode.MaxASCII, "Content-Disposition 應僅含 ASCII: %s", disposition)
	}

	data := w.Body.Bytes()
	require.GreaterOrEqual(t, len(data), 4)
	assert.Equal(t, byte('P'), data[0])
	assert.Equal(t, byte('K'), data[1])
}

func TestExportHandler_DownloadJobTypes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tests := []struct {
		name     string
		jobType  string
		filename string
		sheet    string
	}{
		{name: "gov claim", jobType: "gov_claim", filename: "gov-claim-hsinchu-11507.xlsx", sheet: "工作表1"},
		{name: "trip summary", jobType: "trip_summary", filename: "trip-summary-115-07.xlsx", sheet: "Sheet1"},
		{name: "hsinchu schedule", jobType: "hsinchu_schedule", filename: "hsinchu-schedule.xlsx", sheet: "新竹接送時刻表"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newTestExportHandler()
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			path := "/api/v1/exports/job/download?jobType=" + tt.jobType + "&periodYm=115-07&region=hsinchu"
			c.Request, _ = http.NewRequest(http.MethodGet, path, nil)

			h.Download(c)

			require.Equal(t, http.StatusOK, w.Code)
			require.Contains(t, w.Header().Get("Content-Disposition"), tt.filename)
			f, err := excelize.OpenReader(bytes.NewReader(w.Body.Bytes()))
			require.NoError(t, err)
			defer f.Close()
			require.Contains(t, f.GetSheetList(), tt.sheet)
		})
	}
}

func TestExportHandler_Create(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := newTestExportHandler()

	reqBody := map[string]string{
		"jobType":  "gov_claim",
		"periodYm": "115-07",
		"region":   "hsinchu",
		"mode":     "single_multi_case",
	}
	bodyBytes, _ := json.Marshal(reqBody)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest(http.MethodPost, "/api/v1/exports", bytes.NewReader(bodyBytes))
	c.Request.Header.Set("Content-Type", "application/json")

	h.Create(c)

	assert.Equal(t, http.StatusAccepted, w.Code)

	var resp struct {
		Data struct {
			ID          string `json:"id"`
			DownloadURL string `json:"downloadUrl"`
			FileName    string `json:"fileName"`
			Status      string `json:"status"`
		} `json:"data"`
	}

	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.NotEmpty(t, resp.Data.ID)
	assert.Contains(t, resp.Data.DownloadURL, "/api/v1/exports/")
	assert.Contains(t, resp.Data.DownloadURL, "/download")
	assert.NotEqual(t, "/healthz", resp.Data.DownloadURL, "downloadUrl 不可再指向 /healthz")
}

// newTestExportHandler 以真實的 Excel renderer 組出 handler，與 cmd/server 的接線
// 一致；repository 為 nil 時各 repo 皆回傳空結果，可在無資料庫下驗證檔案輸出。
func newTestExportHandler() *ExportHandler {
	renderer := infra.NewExcelRenderer()
	return NewExportHandler(
		app.NewPrecheckService(infra.NewPrecheckRepository(nil)),
		app.NewReportService(infra.NewReportRepository(nil), renderer),
	)
}
