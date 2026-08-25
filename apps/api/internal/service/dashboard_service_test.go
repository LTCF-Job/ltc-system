package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDashboardService_GetMetrics(t *testing.T) {
	svc := NewDashboardService(nil)
	ctx := context.Background()

	metrics, err := svc.GetMetrics(ctx, "115-07")
	assert.NoError(t, err)
	assert.NotNil(t, metrics)
	assert.Equal(t, "115-07", metrics.CurrentMonth)
	assert.Greater(t, metrics.TotalCasesCount, 0)
	assert.NotEmpty(t, metrics.VehicleTripTrends)
	assert.Greater(t, metrics.AttendanceDistribution.WorkCount, 0)
}
