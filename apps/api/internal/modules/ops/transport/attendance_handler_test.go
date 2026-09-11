package transport

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"ltc-system/apps/api/internal/modules/ops/app"
)

// erroringAttendanceStore 讓 Upsert 回傳指定錯誤，供 handler 層錯誤碼映射測試使用；
// 其餘方法皆回傳空值，測試案例不會用到。
type erroringAttendanceStore struct {
	upsertErr error
}

func (erroringAttendanceStore) GetMonthRecords(context.Context, time.Time, time.Time, *uuid.UUID) ([]app.AttendanceRecord, error) {
	return nil, nil
}
func (erroringAttendanceStore) GetOne(context.Context, uuid.UUID, time.Time) (*app.AttendanceRecord, error) {
	return nil, nil
}
func (s erroringAttendanceStore) Upsert(context.Context, uuid.UUID, time.Time, string, *string, string) (*app.AttendanceRecord, error) {
	return nil, s.upsertErr
}
func (erroringAttendanceStore) UpsertConflict(context.Context, uuid.UUID, time.Time, string, string) error {
	return nil
}
func (erroringAttendanceStore) ListConflicts(context.Context, string) ([]app.AttendanceImportConflict, error) {
	return nil, nil
}
func (erroringAttendanceStore) GetConflict(context.Context, uuid.UUID) (*app.AttendanceImportConflict, error) {
	return nil, nil
}
func (erroringAttendanceStore) ResolveConflict(context.Context, uuid.UUID, string, *uuid.UUID) error {
	return nil
}
func (erroringAttendanceStore) DeleteConflict(context.Context, uuid.UUID) error { return nil }

type emptyDriverListerStub struct{}

func (emptyDriverListerStub) List(context.Context, string, int, int) ([]app.DriverRef, int64, error) {
	return nil, 0, nil
}
func (emptyDriverListerStub) ListAllActive(context.Context) ([]app.DriverRef, error) {
	return nil, nil
}

type discardAuditWriterStub struct{}

func (discardAuditWriterStub) Write(context.Context, app.AuditEntry) error { return nil }

type emptyHolidayReaderStub struct{}

func (emptyHolidayReaderStub) GetHolidayMap(context.Context, int, int) (map[string]bool, error) {
	return map[string]bool{}, nil
}

func newAttendanceHandlerWithUpsertErr(err error) *AttendanceHandler {
	svc := app.NewAttendanceService(
		erroringAttendanceStore{upsertErr: err},
		emptyDriverListerStub{},
		discardAuditWriterStub{},
		emptyHolidayReaderStub{},
	)
	return NewAttendanceHandler(svc)
}

func TestAttendanceHandlerUpsert_MapsDriverNotFoundTo404(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := newAttendanceHandlerWithUpsertErr(app.ErrAttendanceDriverNotFound)
	router := gin.New()
	router.POST("/attendance", h.Upsert)

	body, _ := json.Marshal(map[string]any{
		"driverId":   uuid.New().String(),
		"recordDate": "2026-07-15",
		"status":     "work",
	})
	req := httptest.NewRequest(http.MethodPost, "/attendance", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusNotFound, resp.Code)
	assert.Contains(t, resp.Body.String(), `"code":"NOT_FOUND"`)
}

func TestAttendanceHandlerUpsert_MapsInvalidStatusTo400(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := newAttendanceHandlerWithUpsertErr(app.ErrInvalidAttendanceStatus)
	router := gin.New()
	router.POST("/attendance", h.Upsert)

	body, _ := json.Marshal(map[string]any{
		"driverId":   uuid.New().String(),
		"recordDate": "2026-07-15",
		"status":     "work",
	})
	req := httptest.NewRequest(http.MethodPost, "/attendance", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusBadRequest, resp.Code)
	assert.Contains(t, resp.Body.String(), `"code":"VALIDATION_FAILED"`)
}

func TestAttendanceHandlerGetMonthAttendance_InvalidMonthMessage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := app.NewAttendanceService(
		erroringAttendanceStore{},
		emptyDriverListerStub{},
		discardAuditWriterStub{},
		emptyHolidayReaderStub{},
	)
	h := NewAttendanceHandler(svc)
	router := gin.New()
	router.GET("/attendance/month", h.GetMonthAttendance)

	req := httptest.NewRequest(http.MethodGet, "/attendance/month?month=not-a-month", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusBadRequest, resp.Code)
	assert.Contains(t, resp.Body.String(), "RRR-MM")
}
