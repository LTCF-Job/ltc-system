package app

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

type mockTaskRepo struct {
	slots []ReportedRideSlot
	stats MonthEndRideStats
	err   error
}

func (m *mockTaskRepo) GetReportedRideSlots(ctx context.Context, targetDate time.Time) ([]ReportedRideSlot, error) {
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

func TestTaskService_CheckMissingReports_WithMock(t *testing.T) {
	caseID := uuid.New()
	mockRepo := &mockTaskRepo{
		slots: []ReportedRideSlot{
			{CaseID: caseID, LegSeq: 1},
		},
	}

	// 驗證 TaskService 與 TaskStore 介面相容
	var _ TaskStore = mockRepo
	svc := NewTaskService(mockRepo, nil, nil, nil)
	assert.NotNil(t, svc)
}
