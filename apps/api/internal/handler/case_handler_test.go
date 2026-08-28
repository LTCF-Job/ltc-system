package handler

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"unicode"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDownloadTemplate_TableDriven(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := &CaseHandler{}

	tests := []struct {
		name                string
		url                 string
		expectedContentType string
		expectedFilename    string
		checkBody           func(t *testing.T, body *bytes.Buffer)
	}{
		{
			name:                "預設未傳 format 參數應產生 Excel 範本",
			url:                 "/api/v1/cases/template",
			expectedContentType: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
			expectedFilename:    "case_template.xlsx",
			checkBody: func(t *testing.T, body *bytes.Buffer) {
				data := body.Bytes()
				require.GreaterOrEqual(t, len(data), 4, "Excel 檔案大小應至少有 4 位元組")
				// 驗證 ZIP/Excel 檔案 Magic Number (PK\x03\x04)
				assert.Equal(t, byte(0x50), data[0])
				assert.Equal(t, byte(0x4B), data[1])
				assert.Equal(t, byte(0x03), data[2])
				assert.Equal(t, byte(0x04), data[3])
			},
		},
		{
			name:                "指定 format=xlsx 應產生 Excel 範本",
			url:                 "/api/v1/cases/template?format=xlsx",
			expectedContentType: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
			expectedFilename:    "case_template.xlsx",
			checkBody: func(t *testing.T, body *bytes.Buffer) {
				data := body.Bytes()
				require.Greater(t, len(data), 1000)
				assert.Equal(t, byte('P'), data[0])
				assert.Equal(t, byte('K'), data[1])
			},
		},
		{
			name:                "指定 format=csv 應產生含 UTF-8 BOM 之 CSV 範本",
			url:                 "/api/v1/cases/template?format=csv",
			expectedContentType: "text/csv; charset=utf-8",
			expectedFilename:    "case_template.csv",
			checkBody: func(t *testing.T, body *bytes.Buffer) {
				str := body.String()
				assert.True(t, strings.HasPrefix(str, "\uFEFF個案姓名*"), "CSV 開頭必須包含 UTF-8 BOM 與標題欄")
				assert.Contains(t, str, "週一趟數")
				assert.Contains(t, str, "週二趟數")
				assert.Contains(t, str, "單趟里程")
			},
		},
		{
			name:                "不區分大小寫 format=CSV 應正常回傳 CSV 範本",
			url:                 "/api/v1/cases/template?format=CSV",
			expectedContentType: "text/csv; charset=utf-8",
			expectedFilename:    "case_template.csv",
			checkBody: func(t *testing.T, body *bytes.Buffer) {
				str := body.String()
				assert.True(t, strings.HasPrefix(str, "\uFEFF個案姓名*"))
			},
		},
		{
			name:                "未知格式 format=unknown 應安全降級為 Excel 範本",
			url:                 "/api/v1/cases/template?format=unknown",
			expectedContentType: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
			expectedFilename:    "case_template.xlsx",
			checkBody: func(t *testing.T, body *bytes.Buffer) {
				data := body.Bytes()
				assert.Equal(t, byte('P'), data[0])
				assert.Equal(t, byte('K'), data[1])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request, _ = http.NewRequest(http.MethodGet, tt.url, nil)

			h.DownloadTemplate(c)

			assert.Equal(t, http.StatusOK, w.Code)
			assert.Contains(t, w.Header().Get("Content-Type"), tt.expectedContentType)

			disposition := w.Header().Get("Content-Disposition")
			assert.Contains(t, disposition, tt.expectedFilename)
			assert.Contains(t, disposition, "filename*=UTF-8''")

			// 確保 header 僅包含 ASCII 字元，符合 RFC 5987 規範避免 Proxy 拋出 500
			for _, ch := range disposition {
				assert.True(t, ch <= unicode.MaxASCII, "Content-Disposition 不可包含非 ASCII 字元: %s", disposition)
			}

			if tt.checkBody != nil {
				tt.checkBody(t, w.Body)
			}
		})
	}
}
