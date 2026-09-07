package httpx

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// serveWithRequestID 以掛上 RequestIDMiddleware 的最小 engine 執行一次請求，
// 回傳 handler 內看到的識別碼與實際寫出的回應。
func serveWithRequestID(inbound string) (string, *httptest.ResponseRecorder) {
	engine := gin.New()
	engine.Use(RequestIDMiddleware())

	var seen string
	engine.GET("/probe", func(c *gin.Context) {
		seen = RequestID(c)
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/probe", nil)
	if inbound != "" {
		req.Header.Set(RequestIDHeader, inbound)
	}
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)
	return seen, w
}

func TestRequestIDMiddlewareGeneratesID(t *testing.T) {
	seen, w := serveWithRequestID("")

	assert.NotEmpty(t, seen)
	assert.Equal(t, seen, w.Header().Get(RequestIDHeader))
}

func TestRequestIDMiddlewareReusesCleanInboundID(t *testing.T) {
	seen, w := serveWithRequestID("trace-abc_123")

	assert.Equal(t, "trace-abc_123", seen)
	assert.Equal(t, "trace-abc_123", w.Header().Get(RequestIDHeader))
}

// TestRequestIDMiddlewareRejectsDirtyInboundID 確認上游傳入的控制字元、換行或過長字串
// 不會被原樣寫進 log 與 response header。
func TestRequestIDMiddlewareRejectsDirtyInboundID(t *testing.T) {
	cases := map[string]string{
		"含空白":  "abc def",
		"含中文":  "識別碼",
		"含分隔符": "abc/def",
		"過長":   strings.Repeat("a", maxInboundRequestIDLength+1),
	}
	for name, inbound := range cases {
		t.Run(name, func(t *testing.T) {
			seen, _ := serveWithRequestID(inbound)

			assert.NotEqual(t, inbound, seen)
			assert.NotEmpty(t, seen)
		})
	}
}

// TestRequestIDWithoutMiddleware 確認未掛 middleware 的呼叫路徑（例如單元測試直接
// 建立 context）不會 panic，只是拿不到識別碼。
func TestRequestIDWithoutMiddleware(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	assert.Empty(t, RequestID(c))
	assert.Empty(t, RequestID(nil))
}
