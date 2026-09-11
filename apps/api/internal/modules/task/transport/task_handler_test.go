package transport

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"ltc-system/apps/api/internal/modules/task/app"
)

// fakeTaskStore 提供無已回報紀錄的固定回應，讓比對邏輯必定算出「未回報」項目。
type fakeTaskStore struct{}

func (fakeTaskStore) GetReportedRideSlotsInRange(context.Context, time.Time, time.Time) ([]app.ReportedRideSlotOnDate, error) {
	return nil, nil
}
func (fakeTaskStore) GetMonthEndRideStats(context.Context, time.Time, time.Time) (app.MonthEndRideStats, error) {
	return app.MonthEndRideStats{}, nil
}

// fakeScheduleReader 回傳一個每天出車的排班，確保測試日一定有未回報項目。
type fakeScheduleReader struct{}

func (fakeScheduleReader) GetActiveSchedulesForMonth(context.Context, int, int) ([]app.ActiveSchedule, error) {
	return []app.ActiveSchedule{
		{
			CaseID:        uuid.New(),
			CaseName:      "測試個案",
			EffectiveFrom: time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
			Weekdays:      []int16{1, 2, 3, 4, 5, 6, 7},
			TripPattern:   2,
			Legs: []app.ScheduleLeg{
				{LegSeq: 1, Direction: "go", DepartTime: "08:00"},
			},
		},
	}, nil
}

type fakeHolidayMapReader struct{}

func (fakeHolidayMapReader) GetHolidayMap(context.Context, int, int) (map[string]bool, error) {
	return map[string]bool{}, nil
}

// controlledNotifier 讓測試指定通知服務是否成功、以及是否完全未設定通知服務。
type controlledNotifier struct {
	err error
}

func (n *controlledNotifier) SendNotification(context.Context, string, string, string) error {
	return n.err
}

func newTaskHandler(notifier app.Notifier) *TaskHandler {
	svc := app.NewTaskService(fakeTaskStore{}, fakeScheduleReader{}, fakeHolidayMapReader{}, notifier)
	return NewTaskHandler(svc)
}

func decodeMissingReportsResponse(t *testing.T, body []byte) map[string]any {
	t.Helper()
	var parsed struct {
		Data map[string]any `json:"data"`
	}
	require.NoError(t, json.Unmarshal(body, &parsed))
	return parsed.Data
}

func TestCheckMissingReports_MessageReflectsNotificationSent(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := newTaskHandler(&controlledNotifier{})
	router := gin.New()
	router.POST("/tasks/check-missing-reports", h.CheckMissingReports)

	req := httptest.NewRequest(http.MethodPost, "/tasks/check-missing-reports?date=2026-07-15", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code)
	data := decodeMissingReportsResponse(t, resp.Body.Bytes())
	assert.Equal(t, true, data["notificationSent"])
	assert.Contains(t, data["message"], "已成功執行未回報檢核")
}

func TestCheckMissingReports_MessageReflectsNotificationFailure(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := newTaskHandler(&controlledNotifier{err: assert.AnError})
	router := gin.New()
	router.POST("/tasks/check-missing-reports", h.CheckMissingReports)

	req := httptest.NewRequest(http.MethodPost, "/tasks/check-missing-reports?date=2026-07-15", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code)
	data := decodeMissingReportsResponse(t, resp.Body.Bytes())
	assert.Equal(t, false, data["notificationSent"])
	assert.Contains(t, data["message"], "通知寄送失敗")
}

func TestCheckMissingReports_MessageReflectsNoNotifierConfigured(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := newTaskHandler(nil)
	router := gin.New()
	router.POST("/tasks/check-missing-reports", h.CheckMissingReports)

	req := httptest.NewRequest(http.MethodPost, "/tasks/check-missing-reports?date=2026-07-15", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code)
	data := decodeMissingReportsResponse(t, resp.Body.Bytes())
	assert.Equal(t, false, data["notificationSent"])
	assert.Contains(t, data["message"], "尚未設定通知收件人")
}
