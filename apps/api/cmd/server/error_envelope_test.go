package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"ltc-system/apps/api/internal/platform/httpx"
)

// newEnvelopeProbeEngine 以 newRouter 相同的前置 middleware 組一個最小 engine，
// 讓 panic 與未註冊路徑的回應可以被單獨驗證。
func newEnvelopeProbeEngine() *gin.Engine {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(httpx.RequestIDMiddleware())
	engine.Use(recoveryMiddleware())
	engine.NoRoute(routeNotFoundHandler(http.StatusNotFound))
	engine.NoMethod(routeNotFoundHandler(http.StatusMethodNotAllowed))
	engine.HandleMethodNotAllowed = true
	return engine
}

// TestPanicRespondsWithErrorEnvelope 鎖住 panic 的回應形狀。gin 內建的 Recovery 只寫出空 body，
// 前端解不到 error.code 就只能顯示毫無線索的通用訊息，也拿不到可反查 log 的識別碼。
func TestPanicRespondsWithErrorEnvelope(t *testing.T) {
	engine := newEnvelopeProbeEngine()
	engine.GET("/boom", func(c *gin.Context) {
		panic("unexpected failure")
	})

	w := httptest.NewRecorder()
	engine.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/boom", nil))

	require.Equal(t, http.StatusInternalServerError, w.Code)

	var got httpx.ErrorResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))
	assert.Equal(t, httpx.CodeInternalError, got.Error.Code)
	assert.NotEmpty(t, got.Error.Message)
	assert.NotEmpty(t, got.Error.RequestID, "使用者要能回報錯誤編號")
	assert.NotContains(t, got.Error.Message, "unexpected failure", "panic 內容不得外洩到前端")
}

// TestUnknownRouteRespondsWithErrorEnvelope 確認打到不存在的位址時，前端拿得到可分辨的錯誤碼，
// 而不是 gin 預設的純文字 404。
func TestUnknownRouteRespondsWithErrorEnvelope(t *testing.T) {
	engine := newEnvelopeProbeEngine()
	engine.GET("/api/v1/known", func(c *gin.Context) { c.Status(http.StatusOK) })

	cases := []struct {
		name   string
		method string
		path   string
		status int
	}{
		{"未註冊路徑", http.MethodGet, "/api/v1/does-not-exist", http.StatusNotFound},
		{"方法不符", http.MethodPost, "/api/v1/known", http.StatusMethodNotAllowed},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			engine.ServeHTTP(w, httptest.NewRequest(tc.method, tc.path, nil))

			require.Equal(t, tc.status, w.Code)

			var got httpx.ErrorResponse
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))
			assert.Equal(t, httpx.CodeRouteNotFound, got.Error.Code)
			assert.NotEmpty(t, got.Error.Message)
		})
	}
}
