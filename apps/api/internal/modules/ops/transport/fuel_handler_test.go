package transport

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"ltc-system/apps/api/internal/modules/ops/app"
)

func TestFuelHandlerListRejectsInvalidQueryParameters(t *testing.T) {
	tests := []struct {
		name  string
		query string
	}{
		{name: "invalid page", query: "?page=abc"},
		{name: "zero page size", query: "?pageSize=0"},
		{name: "invalid vehicle uuid", query: "?vehicleId=not-a-uuid"},
		{name: "invalid driver uuid", query: "?driverId=not-a-uuid"},
		{name: "invalid start date", query: "?startDate=2026-99-99"},
		{name: "invalid end date", query: "?endDate=2026/01/01"},
		{name: "reversed date range", query: "?startDate=2026-02-01&endDate=2026-01-01"},
	}

	gin.SetMode(gin.TestMode)
	h := NewFuelHandler(app.NewFuelService(nil, nil))
	router := gin.New()
	router.GET("/fuel-logs", h.List)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/fuel-logs"+tt.query, nil)
			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)

			assert.Equal(t, http.StatusBadRequest, resp.Code)
			assert.Contains(t, resp.Body.String(), `"code":"VALIDATION_FAILED"`)
		})
	}
}
