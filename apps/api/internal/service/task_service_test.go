package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"ltc-system/apps/api/internal/repository"
)

type mockTaskRepo struct {
	slots []repository.ReportedRideSlot
	stats repository.MonthEndRideStats
	err   error
}

func (m *mockTaskRepo) GetReportedRideSlots(ctx context.Context, targetDate time.Time) ([]repository.ReportedRideSlot, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.slots, nil
}

func (m *mockTaskRepo) GetMonthEndRideStats(ctx context.Context, start, end time.Time) (repository.MonthEndRideStats, error) {
	if m.err != nil {
		return repository.MonthEndRideStats{}, m.err
	}
	return m.stats, nil
}

func TestTaskService_MonthEndReminder(t *testing.T) {
	mockRepo := &mockTaskRepo{
		stats: repository.MonthEndRideStats{
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
		slots: []repository.ReportedRideSlot{
			{CaseID: caseID, LegSeq: 1},
		},
	}

	// 驗證 TaskService 與 TaskRepositoryPort 介面相容
	var _ TaskRepositoryPort = mockRepo
	svc := NewTaskService(mockRepo, nil, nil, nil)
	assert.NotNil(t, svc)
}
