package transport

import (
	"bytes"
	"context"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

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
	parseErr         error
}

func (s *stubService) ListForms(context.Context) ([]app.ReportForm, error) { return nil, nil }
func (s *stubService) CreateForm(context.Context, string, string) (*app.ReportForm, error) {
	return nil, nil
}
func (s *stubService) DeleteForm(context.Context, string) error { return nil }
func (s *stubService) ListColumns(context.Context, string, string) ([]app.ColumnMapping, error) {
	return nil, nil
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
func (s *stubService) ParseDriverReport(context.Context, uuid.UUID, io.Reader) (*app.PreviewResult, error) {
	if s.parseErr != nil {
		return nil, s.parseErr
	}
	return &app.PreviewResult{TotalRows: 1, ValidRows: 1}, nil
}
func (s *stubService) CommitDriverReport(_ context.Context, _ uuid.UUID, _ io.Reader, decisions []app.ColumnDecision, _ app.Actor) (*app.CommitResult, error) {
	s.commitCalledWith = decisions
	return &app.CommitResult{ImportedRows: 1, RideRecordRows: 2}, nil
}

func newTestRouter(svc *stubService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := NewDriverReportHandler(svc)
	r.GET("/api/v1/driver-reports/:id/template", h.DownloadTemplate)
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

func TestImportExcel_MissingFile(t *testing.T) {
	r := newTestRouter(&stubService{})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/driver-reports/"+uuid.New().String()+"/import", nil)
	req.Header.Set("Content-Type", "multipart/form-data; boundary=x")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
