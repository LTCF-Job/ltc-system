package transport

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"ltc-system/apps/api/internal/modules/holiday/app"
)

// discardHolidayStoreStub 讓所有寫入直接成功，Delete 一律視為找不到資料（GetByDate 回傳 nil），
// 供測試 Delete 的 404 映射使用。
type discardHolidayStoreStub struct {
	existing map[string]*app.Holiday
}

func (s *discardHolidayStoreStub) List(context.Context, time.Time, time.Time) ([]app.Holiday, error) {
	return nil, nil
}
func (s *discardHolidayStoreStub) Upsert(context.Context, *app.Holiday) error       { return nil }
func (s *discardHolidayStoreStub) BatchUpsert(context.Context, []app.Holiday) error { return nil }
func (s *discardHolidayStoreStub) Delete(context.Context, time.Time) error          { return nil }
func (s *discardHolidayStoreStub) GetByDate(_ context.Context, date time.Time) (*app.Holiday, error) {
	if s.existing == nil {
		return nil, nil
	}
	return s.existing[date.Format("2006-01-02")], nil
}

type failingProviderStub struct{}

func (failingProviderStub) Fetch(context.Context, int) ([]app.HolidayRecord, error) {
	return nil, errors.New("upstream timeout")
}

func decodeErrorResponse(t *testing.T, body []byte) map[string]any {
	t.Helper()
	var parsed struct {
		Error map[string]any `json:"error"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		t.Fatalf("failed to decode error response: %v", err)
	}
	return parsed.Error
}

func TestHolidayHandlerImport_YearOutOfRangeReturns400(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := app.NewHolidaySyncService(&discardHolidayStoreStub{}, nil, failingProviderStub{})
	h := NewHolidayHandler(svc)
	router := gin.New()
	router.POST("/holidays/import", h.Import)

	body, _ := json.Marshal(map[string]any{"year": 1999})
	req := httptest.NewRequest(http.MethodPost, "/holidays/import", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusBadRequest, resp.Code)
	errBody := decodeErrorResponse(t, resp.Body.Bytes())
	assert.Equal(t, "VALIDATION_FAILED", errBody["code"])
}

func TestHolidayHandlerImport_ExternalAPIFailureReturns503(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := app.NewHolidaySyncService(&discardHolidayStoreStub{}, nil, failingProviderStub{})
	h := NewHolidayHandler(svc)
	router := gin.New()
	router.POST("/holidays/import", h.Import)

	body, _ := json.Marshal(map[string]any{"year": 2026})
	req := httptest.NewRequest(http.MethodPost, "/holidays/import", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusServiceUnavailable, resp.Code)
	errBody := decodeErrorResponse(t, resp.Body.Bytes())
	assert.Equal(t, "SERVICE_UNAVAILABLE", errBody["code"])
}

func TestHolidayHandlerDelete_NotFoundReturns404(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := app.NewHolidayService(&discardHolidayStoreStub{existing: map[string]*app.Holiday{}}, nil)
	h := NewHolidayHandler(svc)
	router := gin.New()
	router.DELETE("/holidays/:date", h.Delete)

	req := httptest.NewRequest(http.MethodDelete, "/holidays/2026-10-10", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusNotFound, resp.Code)
	errBody := decodeErrorResponse(t, resp.Body.Bytes())
	assert.Equal(t, "NOT_FOUND", errBody["code"])
}
