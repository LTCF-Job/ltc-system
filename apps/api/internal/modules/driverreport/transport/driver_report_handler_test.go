package transport

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xuri/excelize/v2"
	"ltc-system/apps/api/internal/modules/driverreport/app"
	"ltc-system/apps/api/internal/modules/driverreport/infra"
)

// stubService 只實作 handler 用到的方法；範本產生刻意走真正的 infra renderer，
// 讓這支測試涵蓋「渲染 → handler → HTTP 回應主體」的整條位元組路徑。
type stubService struct {
	commitCalledWith []app.ColumnDecision
	parseYearMonth   string
	commitYearMonth  string
	parseErr         error
	importedMonths   []app.ImportedMonth
	columns          []app.ColumnMapping
}

func (s *stubService) ListForms(context.Context) ([]app.ReportForm, error) { return nil, nil }
func (s *stubService) ListImportedMonths(context.Context) ([]app.ImportedMonth, error) {
	return s.importedMonths, nil
}
func (s *stubService) CreateForm(context.Context, string, string) (*app.ReportForm, error) {
	return nil, nil
}
func (s *stubService) DeleteForm(context.Context, string) error { return nil }
func (s *stubService) ListColumns(_ context.Context, formID, mappingStatus string) ([]app.ColumnMapping, error) {
	var matched []app.ColumnMapping
	for _, c := range s.columns {
		if (formID == "" || c.FormID == formID) && (mappingStatus == "" || c.MappingStatus == mappingStatus) {
			matched = append(matched, c)
		}
	}
	return matched, nil
}
func (s *stubService) UpdateColumnMapping(context.Context, string, string, *string, *int16) error {
	return nil
}
func (s *stubService) BatchMapping(context.Context, []app.ColumnMappingUpdate) (int, error) {
	return 0, nil
}
func (s *stubService) TemplateExcel(context.Context, uuid.UUID) ([]byte, string, error) {
	data, err := infra.NewExcelAdapter().RenderDriverReportTemplate("竹南2車", []string{"1.吳桂 [去程]"})
	return data, "竹南2車", err
}
func (s *stubService) ParseDriverReport(_ context.Context, _ uuid.UUID, _ io.Reader, yearMonth string) (*app.PreviewResult, error) {
	s.parseYearMonth = yearMonth
	if s.parseErr != nil {
		return nil, s.parseErr
	}
	return &app.PreviewResult{TotalRows: 1, ValidRows: 1}, nil
}
func (s *stubService) CommitDriverReport(_ context.Context, _ uuid.UUID, _ io.Reader, decisions []app.ColumnDecision, yearMonth string, _ app.Actor) (*app.CommitResult, error) {
	s.commitCalledWith = decisions
	s.commitYearMonth = yearMonth
	return &app.CommitResult{ImportedRows: 1, RideRecordRows: 2}, nil
}

func newTestRouter(svc *stubService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := NewDriverReportHandler(svc)
	// 靜態路徑與 :id 萬用路徑必須同時註冊：routes.go 就是這樣掛的，
	// 只註冊其中一支會漏掉 gin 路由樹衝突（衝突會在服務啟動時 panic）
	r.GET("/api/v1/driver-reports/imported-months", h.ListImportedMonths)
	r.GET("/api/v1/driver-reports/columns", h.ListColumns)
	r.GET("/api/v1/driver-reports/:id/template", h.DownloadTemplate)
	r.DELETE("/api/v1/driver-reports/:id", h.DeleteForm)
	r.POST("/api/v1/driver-reports/:id/import", h.ImportExcel)
	return r
}

func TestDownloadTemplate_ResponseBodyIsAReadableWorkbook(t *testing.T) {
	r := newTestRouter(&stubService{})
	formID := uuid.New().String()

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/driver-reports/"+formID+"/template", nil))

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t,
		"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		w.Header().Get("Content-Type"),
	)
	assert.Contains(t, w.Header().Get("Content-Disposition"), "driver_report_template.xlsx")

	f, err := excelize.OpenReader(bytes.NewReader(w.Body.Bytes()))
	require.NoError(t, err, "HTTP 回應主體必須仍是可開啟的 .xlsx")
	defer f.Close()

	rows, err := f.GetRows("司機接送匯報")
	require.NoError(t, err)
	assert.Equal(t, []string{"民國日期", "駕駛人", "1.吳桂 [去程]", "備註"}, rows[0])
}

func TestDownloadTemplate_RejectsMalformedFormID(t *testing.T) {
	r := newTestRouter(&stubService{})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/driver-reports/not-a-uuid/template", nil))

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestImportExcel_DryRunAndCommit(t *testing.T) {
	svc := &stubService{}
	r := newTestRouter(svc)
	formID := uuid.New().String()

	tests := []struct {
		name             string
		query            string
		decisions        string
		expectCommitCall bool
	}{
		{name: "預設為預覽不寫入", query: "", expectCommitCall: false},
		{name: "dryRun=true 為預覽不寫入", query: "?dryRun=true", expectCommitCall: false},
		{
			name:             "dryRun=false 帶入就地確認的欄位對應",
			query:            "?dryRun=false",
			decisions:        `[{"columnHeader":"1.吳桂 [去程]","mappingStatus":"mapped","caseId":"11111111-1111-1111-1111-111111111111","legSeq":1}]`,
			expectCommitCall: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc.commitCalledWith = nil

			body := &bytes.Buffer{}
			writer := multipart.NewWriter(body)
			part, err := writer.CreateFormFile("file", "report.xlsx")
			require.NoError(t, err)
			_, _ = part.Write([]byte("dummy"))
			if tt.decisions != "" {
				require.NoError(t, writer.WriteField("columnDecisions", tt.decisions))
			}
			require.NoError(t, writer.Close())

			req := httptest.NewRequest(http.MethodPost, "/api/v1/driver-reports/"+formID+"/import"+tt.query, body)
			req.Header.Set("Content-Type", writer.FormDataContentType())

			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			require.Equal(t, http.StatusOK, w.Code)
			if tt.expectCommitCall {
				require.Len(t, svc.commitCalledWith, 1)
				assert.Equal(t, "mapped", svc.commitCalledWith[0].MappingStatus)
				require.NotNil(t, svc.commitCalledWith[0].LegSeq)
				assert.Equal(t, int16(1), *svc.commitCalledWith[0].LegSeq)
			} else {
				assert.Nil(t, svc.commitCalledWith)
			}
		})
	}
}

func TestImportExcel_ForwardsDeclaredYearMonth(t *testing.T) {
	svc := &stubService{}
	r := newTestRouter(svc)
	formID := uuid.New().String()

	for _, tt := range []struct {
		name  string
		query string
	}{
		{name: "預覽帶入宣告月份", query: "?dryRun=true&yearMonth=2026-03"},
		{name: "正式寫入帶入宣告月份", query: "?dryRun=false&yearMonth=2026-03"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			svc.parseYearMonth, svc.commitYearMonth = "", ""

			body := &bytes.Buffer{}
			writer := multipart.NewWriter(body)
			part, err := writer.CreateFormFile("file", "report.xlsx")
			require.NoError(t, err)
			_, _ = part.Write([]byte("dummy"))
			require.NoError(t, writer.Close())

			req := httptest.NewRequest(http.MethodPost, "/api/v1/driver-reports/"+formID+"/import"+tt.query, body)
			req.Header.Set("Content-Type", writer.FormDataContentType())

			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			require.Equal(t, http.StatusOK, w.Code)
			assert.Equal(t, "2026-03", svc.parseYearMonth+svc.commitYearMonth,
				"宣告月份必須原樣傳到服務層，否則覆蓋範圍會落在錯誤的月份")
		})
	}
}

func TestImportExcel_MissingFile(t *testing.T) {
	r := newTestRouter(&stubService{})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/driver-reports/"+uuid.New().String()+"/import", nil)
	req.Header.Set("Content-Type", "multipart/form-data; boundary=x")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var body struct {
		Error struct {
			Code    string `json:"code"`
			Details []struct {
				Field  string `json:"field"`
				Reason string `json:"reason"`
			} `json:"details"`
		} `json:"error"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "VALIDATION_FAILED", body.Error.Code)
	require.Len(t, body.Error.Details, 1)
	assert.Equal(t, "file", body.Error.Details[0].Field)
	assert.Equal(t, "未提供上傳檔案", body.Error.Details[0].Reason)
}

func TestListImportedMonths_ReturnsCountAndFormattedLastImportedAt(t *testing.T) {
	formID := uuid.New()
	svc := &stubService{importedMonths: []app.ImportedMonth{
		{
			FormID:          formID,
			YearMonth:       "2026-03",
			SubmissionCount: 21,
			LastImportedAt:  time.Date(2026, 3, 31, 10, 20, 30, 0, time.UTC),
		},
	}}
	r := newTestRouter(svc)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/driver-reports/imported-months", nil))

	require.Equal(t, http.StatusOK, w.Code)
	var body struct {
		Data []ImportedMonthDTO `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Len(t, body.Data, 1)
	assert.Equal(t, formID.String(), body.Data[0].FormID)
	assert.Equal(t, "2026-03", body.Data[0].YearMonth)
	assert.Equal(t, 21, body.Data[0].SubmissionCount)
	assert.Equal(t, "2026-03-31 10:20:30", body.Data[0].LastImportedAt,
		"時間一律格式化至秒數，不得輸出 raw ISO 8601")
}

func TestListImportedMonths_NoDataIsAnEmptyArray(t *testing.T) {
	r := newTestRouter(&stubService{})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/driver-reports/imported-months", nil))

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"data":[]`)
}

func TestListColumns_FilterByStatusWithoutFormID(t *testing.T) {
	svc := &stubService{columns: []app.ColumnMapping{
		{
			ID:            "col-1",
			FormID:        uuid.New().String(),
			FormTitle:     "竹北一車接送匯報",
			VehicleName:   "竹北一車",
			ColumnIndex:   3,
			ColumnHeader:  "1.王小明 [去程]",
			CleanedName:   "王小明",
			Kind:          "ride",
			MappingStatus: "pending",
		},
		{
			ID:            "col-2",
			FormID:        uuid.New().String(),
			FormTitle:     "竹北二車接送匯報",
			VehicleName:   "竹北二車",
			ColumnIndex:   4,
			ColumnHeader:  "1.張小華 [去程]",
			CleanedName:   "張小華",
			Kind:          "ride",
			MappingStatus: "mapped",
		},
	}}
	r := newTestRouter(svc)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/driver-reports/columns?mappingStatus=pending", nil))

	require.Equal(t, http.StatusOK, w.Code)
	var body struct {
		Data []FormColumnDTO `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Len(t, body.Data, 1)
	assert.Equal(t, "col-1", body.Data[0].ID)
	assert.Equal(t, "1.王小明 [去程]", body.Data[0].ColumnHeader)
	assert.Equal(t, "pending", body.Data[0].MappingStatus)
	assert.Nil(t, body.Data[0].CaseID)
	assert.Nil(t, body.Data[0].CaseName)
}
