package httpx

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newUploadRequest(t *testing.T, size int) *http.Request {
	t.Helper()

	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	part, err := w.CreateFormFile("file", "upload.xlsx")
	require.NoError(t, err)
	_, err = part.Write(bytes.Repeat([]byte("a"), size))
	require.NoError(t, err)
	require.NoError(t, w.Close())

	req := httptest.NewRequest(http.MethodPost, "/upload", &body)
	req.Header.Set("Content-Type", w.FormDataContentType())
	return req
}

func TestBindUploadFile(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name       string
		req        func(t *testing.T) *http.Request
		wantOK     bool
		wantStatus int
	}{
		{
			name:       "正常大小的檔案應通過",
			req:        func(t *testing.T) *http.Request { return newUploadRequest(t, 1024) },
			wantOK:     true,
			wantStatus: http.StatusOK,
		},
		{
			name:       "超過上限的檔案應回 413",
			req:        func(t *testing.T) *http.Request { return newUploadRequest(t, MaxUploadBytes+1) },
			wantOK:     false,
			wantStatus: http.StatusRequestEntityTooLarge,
		},
		{
			name: "未提供檔案欄位應回 400",
			req: func(t *testing.T) *http.Request {
				req := httptest.NewRequest(http.MethodPost, "/upload", bytes.NewReader(nil))
				req.Header.Set("Content-Type", "multipart/form-data; boundary=zzz")
				return req
			},
			wantOK:     false,
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(rec)
			c.Request = tt.req(t)

			fileHeader, ok := BindUploadFile(c, "file")

			assert.Equal(t, tt.wantOK, ok)
			if tt.wantOK {
				require.NotNil(t, fileHeader)
				assert.Equal(t, "upload.xlsx", fileHeader.Filename)
				return
			}
			assert.Nil(t, fileHeader)
			assert.Equal(t, tt.wantStatus, rec.Code)
		})
	}
}
