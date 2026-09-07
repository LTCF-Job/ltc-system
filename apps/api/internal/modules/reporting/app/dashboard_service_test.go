package app

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// fakeDashboardRepo is a deterministic DashboardRepositoryPort test double.
type fakeDashboardRepo struct {
	activeCases      int
	reportedTrips    int
	pendingConflicts int
	pendingColumns   int
	tripTrends       []VehicleTripTrend
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

func (f *fakeDashboardRepo) GetVehicleTripTrends(ctx context.Context, start, end time.Time) ([]VehicleTripTrend, error) {
	return f.tripTrends, nil
}

func (f *fakeDashboardRepo) GetAttendanceDistribution(ctx context.Context, start, end time.Time) (map[string]int, error) {
	return f.attendance, nil
}

func TestDashboardService_GetMetrics_NilRepoFailsClosed(t *testing.T) {
	svc := NewDashboardService(nil, nil)
	ctx := context.Background()

	metrics, err := svc.GetMetrics(ctx, "115-07")
	assert.ErrorIs(t, err, errDashboardRepositoryNotConfigured)
	assert.Nil(t, metrics)
}

func TestDashboardService_GetMetrics_QueriesRepoWhenConfigured(t *testing.T) {
	repo := &fakeDashboardRepo{
		activeCases:      12,
		reportedTrips:    34,
		pendingConflicts: 1,
		pendingColumns:   2,
		tripTrends: []VehicleTripTrend{
			{VehicleName: "測試車", PlateNo: "TEST-01", TripCount: 5},
		},
		attendance: map[string]int{"work": 8, "leave": 1, "sick": 1, "off": 0},
	}
	svc := NewDashboardService(repo, nil)
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

// fakeExportJobStore is a deterministic ExportJobStore test double for GetRecentExports.
type fakeExportJobStore struct {
	jobs        []GovClaimJob
	requestedPg int
	requestedSz int
	listJobsErr error
}

func (f *fakeExportJobStore) CreateJob(ctx context.Context, job ExportJobCreate) (uuid.UUID, error) {
	return uuid.Nil, nil
}
func (f *fakeExportJobStore) CompleteJob(ctx context.Context, jobID uuid.UUID, files []GovClaimCaseFile, lines []ExportLine) error {
	return nil
}
func (f *fakeExportJobStore) FailJob(ctx context.Context, jobID uuid.UUID, message string) error {
	return nil
}
func (f *fakeExportJobStore) GetJob(ctx context.Context, jobID uuid.UUID) (GovClaimJob, error) {
	return GovClaimJob{}, nil
}
func (f *fakeExportJobStore) ListJobs(ctx context.Context, page, pageSize int) ([]GovClaimJob, int64, error) {
	f.requestedPg, f.requestedSz = page, pageSize
	if f.listJobsErr != nil {
		return nil, 0, f.listJobsErr
	}
	return f.jobs, int64(len(f.jobs)), nil
}
func (f *fakeExportJobStore) LoadCaseLines(ctx context.Context, jobID, caseID uuid.UUID) ([]ExportLine, error) {
	return nil, nil
}
func (f *fakeExportJobStore) LoadNationalIDCiphers(ctx context.Context, caseID uuid.UUID, driverIDs []uuid.UUID) (NationalIDCiphers, error) {
	return NationalIDCiphers{}, nil
}

func TestDashboardService_GetRecentExports_NilJobStoreFailsClosed(t *testing.T) {
	svc := NewDashboardService(nil, nil)
	jobs, err := svc.GetRecentExports(context.Background())
	assert.ErrorIs(t, err, errExportJobStoreNotConfigured)
	assert.Nil(t, jobs)
}

func TestDashboardService_GetRecentExports_QueriesFirstPageOfFive(t *testing.T) {
	store := &fakeExportJobStore{jobs: []GovClaimJob{{ID: uuid.New()}, {ID: uuid.New()}, {ID: uuid.New()}}}
	svc := NewDashboardService(nil, store)

	jobs, err := svc.GetRecentExports(context.Background())
	assert.NoError(t, err)
	assert.Len(t, jobs, 3)
	assert.Equal(t, 1, store.requestedPg)
	assert.Equal(t, 5, store.requestedSz)
}

func TestDashboardService_GetRecentExports_PropagatesError(t *testing.T) {
	store := &fakeExportJobStore{listJobsErr: assert.AnError}
	svc := NewDashboardService(nil, store)

	_, err := svc.GetRecentExports(context.Background())
	assert.Error(t, err)
}
