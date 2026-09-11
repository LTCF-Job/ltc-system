package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"ltc-system/apps/api/internal/platform/config"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestEmptyTrustedProxiesIgnoreForwardedFor(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()

	trustedProxies, err := config.ParseTrustedProxies("")
	require.NoError(t, err)
	require.NoError(t, engine.SetTrustedProxies(trustedProxies))

	engine.GET("/client-ip", func(c *gin.Context) {
		c.String(http.StatusOK, c.ClientIP())
	})

	request := httptest.NewRequest(http.MethodGet, "/client-ip", nil)
	request.RemoteAddr = "192.0.2.10:1234"
	request.Header.Set("X-Forwarded-For", "198.51.100.20")
	response := httptest.NewRecorder()

	engine.ServeHTTP(response, request)

	require.Equal(t, http.StatusOK, response.Code)
	require.Equal(t, "192.0.2.10", response.Body.String())
}
