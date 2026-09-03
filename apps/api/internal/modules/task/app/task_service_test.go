package app

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

type mockTaskRepo struct {
	slots []ReportedRideSlotOnDate
	stats MonthEndRideStats
	err   error
}

func (m *mockTaskRepo) GetReportedRideSlotsInRange(ctx context.Context, start, end time.Time) ([]ReportedRideSlotOnDate, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.slots, nil
}

func (m *mockTaskRepo) GetMonthEndRideStats(ctx context.Context, start, end time.Time) (MonthEndRideStats, error) {
	if m.err != nil {
		return MonthEndRideStats{}, m.err
	}
	return m.stats, nil
}

// fakeScheduleReader 回傳固定排班清單，供整月未回報比對測試使用。
type fakeScheduleReader struct {
	schedules []ActiveSchedule
}

func (f *fakeScheduleReader) GetActiveSchedulesForMonth(ctx context.Context, year, month int, region string) ([]ActiveSchedule, error) {
	return f.schedules, nil
}

// weekdayScheduleFixture 建立一個每天出車、只有去程一趟的排班，涵蓋整月。
func weekdayScheduleFixture(caseID uuid.UUID, year, month int) ActiveSchedule {
	from := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	return ActiveSchedule{
		CaseID:        caseID,
		CaseName:      "測試個案",
		Region:        "竹北",
		EffectiveFrom: from,
		Weekdays:      []int16{1, 2, 3, 4, 5, 6, 7},
		SiteOpenDays:  []int16{1, 2, 3, 4, 5, 6, 7},
		TripPattern:   2,
		Legs: []ScheduleLeg{
			{LegSeq: 1, Direction: "go", DepartTime: "08:00"},
		},
	}
}

type fakeHolidayMapReader struct{}

func (f *fakeHolidayMapReader) GetHolidayMap(ctx context.Context, year, month int, region string) (map[string]bool, error) {
	return map[string]bool{}, nil
}

type fakeNotifier struct {
	calls int
}

func (f *fakeNotifier) SendNotification(ctx context.Context, topic, subject, body string) error {
	f.calls++
	return nil
}

func TestTaskService_MonthEndReminder(t *testing.T) {
	mockRepo := &mockTaskRepo{
		stats: MonthEndRideStats{
			TotalRides:      100,
			BoardedRides:    90,
			UnreportedRides: 8,
			ConflictCount:   2,
		},
	}

	svc := NewTaskService(mockRepo, nil, nil, nil)
	ctx := context.Background()

	summary, err := svc.MonthEndReminder(ctx, 2026, 7)
	assert.NoError(t, err)
	assert.NotNil(t, summary)
	assert.Equal(t, "115-07", summary.YearMonth)
	assert.Equal(t, 100, summary.TotalRides)
	assert.Equal(t, 90, summary.BoardedRides)
	assert.Equal(t, 8, summary.UnreportedRides)
	assert.Equal(t, 2, summary.ConflictCount)
}

func TestTaskService_ListMissingReportsForMonth(t *testing.T) {
	caseID := uuid.New()
	scheduleReader := &fakeScheduleReader{schedules: []ActiveSchedule{weekdayScheduleFixture(caseID, 2026, 7)}}
	notifier := &fakeNotifier{}
	repo := &mockTaskRepo{}
	svc := NewTaskService(repo, scheduleReader, &fakeHolidayMapReader{}, notifier)

	items, err := svc.ListMissingReportsForMonth(context.Background(), 2026, 7, "")
	assert.NoError(t, err)
	assert.Equal(t, 31, len(items), "七月每天出車一趟，整月都未回報應有 31 筆")
	assert.Zero(t, notifier.calls, "整月查詢不應觸發催報通知")
}

func TestTaskService_ListMissingReportsForMonth_CrossDateCollisionRegression(t *testing.T) {
	// 同一個案、同一趟次在 7/1 已回報，7/2 未回報：差集必須用 {caseID, date, legSeq}
	// 三元組判斷，否則 7/1 的已回報紀錄會被誤判成也涵蓋 7/2。
	caseID := uuid.New()
	scheduleReader := &fakeScheduleReader{schedules: []ActiveSchedule{weekdayScheduleFixture(caseID, 2026, 7)}}
	repo := &mockTaskRepo{
		slots: []ReportedRideSlotOnDate{
			{CaseID: caseID, ServiceDate: time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC), LegSeq: 1},
		},
	}
	svc := NewTaskService(repo, scheduleReader, &fakeHolidayMapReader{}, nil)

	items, err := svc.ListMissingReportsForMonth(context.Background(), 2026, 7, "")
	assert.NoError(t, err)
	assert.Equal(t, 30, len(items), "7/1 已回報，其餘 30 天仍應視為未回報")
	for _, item := range items {
		assert.NotEqual(t, "2026-07-01", item.ServiceDate)
	}
}

func TestTaskService_CheckMissingReports_SingleDayStillNotifies(t *testing.T) {
	caseID := uuid.New()
	scheduleReader := &fakeScheduleReader{schedules: []ActiveSchedule{weekdayScheduleFixture(caseID, 2026, 7)}}
	notifier := &fakeNotifier{}
	repo := &mockTaskRepo{}
	svc := NewTaskService(repo, scheduleReader, &fakeHolidayMapReader{}, notifier)

	items, err := svc.CheckMissingReports(context.Background(), time.Date(2026, 7, 15, 0, 0, 0, 0, time.UTC), "")
	assert.NoError(t, err)
	assert.Len(t, items, 1)
	assert.Equal(t, 1, notifier.calls, "單日模式偵測到未回報時仍應觸發通知")
}
