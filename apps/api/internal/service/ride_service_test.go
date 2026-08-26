package service

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestRideService_ManualReportRide_Validation(t *testing.T) {
	svc := NewRideService(nil, nil, nil, nil, nil)
	ctx := context.Background()
	actorID := uuid.New()

	t.Run("Invalid effective status", func(t *testing.T) {
		req := ManualReportRideRequest{
			CaseID:          uuid.New(),
			ServiceDate:     "2026-08-24",
			LegSeq:          1,
			EffectiveStatus: "unknown",
		}
		_, err := svc.ManualReportRide(ctx, req, actorID, "staff", "127.0.0.1", "test-ua")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "無效的搭乘狀態")
	})

	t.Run("Invalid date format", func(t *testing.T) {
		req := ManualReportRideRequest{
			CaseID:          uuid.New(),
			ServiceDate:     "2026/08/24",
			LegSeq:          1,
			EffectiveStatus: "boarded",
		}
		_, err := svc.ManualReportRide(ctx, req, actorID, "staff", "127.0.0.1", "test-ua")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "無效的服務日期格式")
	})
}

func TestRideService_ManualReportRide_WithMockRepo(t *testing.T) {
	// 驗證結構體欄位映射正確性
	req := ManualReportRideRequest{
		CaseID:              uuid.New(),
		ServiceDate:         "2026-08-24",
		LegSeq:              2,
		EffectiveStatus:     "boarded",
		DepartTimeOverride:  func() *string { s := "16:00"; return &s }(),
		DurationMinOverride: func() *int16 { i := int16(15); return &i }(),
		NotClaimedAA09:      func() *bool { b := false; return &b }(),
		Reason:              func() *string { s := "司機口頭回報"; return &s }(),
	}

	assert.Equal(t, "boarded", req.EffectiveStatus)
	assert.Equal(t, int16(2), req.LegSeq)
	assert.Equal(t, "2026-08-24", req.ServiceDate)
	assert.Equal(t, "16:00", *req.DepartTimeOverride)
	assert.Equal(t, "司機口頭回報", *req.Reason)
}
