package transport

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"ltc-system/apps/api/internal/modules/ops/app"
)

func TestIsValidReceiptURL(t *testing.T) {
	str := func(s string) *string { return &s }

	tests := []struct {
		name string
		url  *string
		want bool
	}{
		{name: "nil is valid", url: nil, want: true},
		{name: "empty is valid", url: str(""), want: true},
		{name: "http is valid", url: str("http://example.com/receipt.jpg"), want: true},
		{name: "https is valid", url: str("https://example.com/receipt.jpg"), want: true},
		{name: "javascript protocol is invalid", url: str("javascript:alert(1)"), want: false},
		{name: "data protocol is invalid", url: str("data:text/html,<script>alert(1)</script>"), want: false},
		{name: "protocol-relative is invalid", url: str("//evil.example.com"), want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, isValidReceiptURL(tt.url))
		})
	}
}

func TestMaintenanceHandlerCreateRejectsInvalidReceiptURL(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewMaintenanceHandler(app.NewMaintenanceService(nil, nil, nil, nil))
	router := gin.New()
	router.POST("/maintenance-logs", h.Create)

	body := `{
		"vehicleId": "11111111-1111-1111-1111-111111111111",
		"serviceDate": "2026-01-01",
		"mileage": 1000,
		"items": "換機油",
		"cost": 500,
		"receiptUrl": "javascript:alert(1)"
	}`
	req := httptest.NewRequest(http.MethodPost, "/maintenance-logs", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusBadRequest, resp.Code)
	assert.Contains(t, resp.Body.String(), `"code":"VALIDATION_FAILED"`)
}

func TestFuelHandlerCreateRejectsInvalidReceiptURL(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewFuelHandler(app.NewFuelService(nil, nil))
	router := gin.New()
	router.POST("/fuel-logs", h.Create)

	body := `{
		"vehicleId": "11111111-1111-1111-1111-111111111111",
		"fuelDate": "2026-01-01",
		"liters": 10,
		"cost": 500,
		"receiptUrl": "javascript:alert(1)"
	}`
	req := httptest.NewRequest(http.MethodPost, "/fuel-logs", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusBadRequest, resp.Code)
	assert.Contains(t, resp.Body.String(), `"code":"VALIDATION_FAILED"`)
}
