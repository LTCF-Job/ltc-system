package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"ltc-system/apps/api/internal/repository"
)

// fakeDashboardRepo is a deterministic DashboardRepositoryPort test double.
type fakeDashboardRepo struct {
	activeCases      int
	reportedTrips    int
	pendingConflicts int
	pendingColumns   int
	tripTrends       []repository.VehicleTripTrendData
	attendance       map[string]int
}

func (f *fakeDashboardRepo) GetActiveCasesCount(ctx context.Context) (int, error) {
	return f.activeCases, nil
}

func (f *fakeDashboardRepo) GetReportedTripsCount(ctx context.Context, start, end time.Time) (int, error) {
	return f.reportedTrips, nil
}

func (f *fakeDashboardRepo) GetPendingConflictsCount(ctx context.Context) (int, error) {
	return f.pendingConflicts, nil
}

func (f *fakeDashboardRepo) GetPendingFormColumnsCount(ctx context.Context) (int, error) {
	return f.pendingColumns, nil
}

func (f *fakeDashboardRepo) GetVehicleTripTrends(ctx context.Context, start, end time.Time) ([]repository.VehicleTripTrendData, error) {
	return f.tripTrends, nil
}

func (f *fakeDashboardRepo) GetAttendanceDistribution(ctx context.Context, start, end time.Time) (map[string]int, error) {
	return f.attendance, nil
}

func TestDashboardService_GetMetrics_NilRepoReturnsHonestZeroValues(t *testing.T) {
	svc := NewDashboardService(nil)
	ctx := context.Background()

	metrics, err := svc.GetMetrics(ctx, "115-07")
	assert.NoError(t, err)
	assert.NotNil(t, metrics)
	assert.Equal(t, "115-07", metrics.CurrentMonth)
	assert.Equal(t, 0, metrics.TotalCasesCount)
	assert.Equal(t, 0, metrics.ReportedTripsCount)
	assert.Equal(t, 0, metrics.PendingConflictsCount)
	assert.Empty(t, metrics.VehicleTripTrends)
	assert.Equal(t, 0, metrics.AttendanceDistribution.WorkCount)
}

func TestDashboardService_GetMetrics_QueriesRepoWhenConfigured(t *testing.T) {
	repo := &fakeDashboardRepo{
		activeCases:      12,
		reportedTrips:    34,
		pendingConflicts: 1,
		pendingColumns:   2,
		tripTrends: []repository.VehicleTripTrendData{
			{VehicleName: "測試車", PlateNo: "TEST-01", TripCount: 5},
		},
		attendance: map[string]int{"work": 8, "leave": 1, "sick": 1, "off": 0},
	}
	svc := NewDashboardService(repo)
	ctx := context.Background()

	metrics, err := svc.GetMetrics(ctx, "115-07")
	assert.NoError(t, err)
	assert.Equal(t, 12, metrics.TotalCasesCount)
	assert.Equal(t, 34, metrics.ReportedTripsCount)
	assert.Equal(t, 1, metrics.PendingConflictsCount)
	assert.Equal(t, 2, metrics.PendingFormColumnsCount)
	assert.Len(t, metrics.VehicleTripTrends, 1)
	assert.Equal(t, "測試車", metrics.VehicleTripTrends[0].VehicleName)
	assert.Equal(t, 8, metrics.AttendanceDistribution.WorkCount)
}
