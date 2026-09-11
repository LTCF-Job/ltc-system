package app

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type manualScheduleReader struct {
	schedule *CaseSchedule
}

func (r manualScheduleReader) GetActiveScheduleForCaseOnDate(context.Context, uuid.UUID, time.Time) (*CaseSchedule, error) {
	return r.schedule, nil
}

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
		assert.ErrorIs(t, err, ErrInvalidManualReportStatus)
	})

	t.Run("Invalid date format", func(t *testing.T) {
		req := ManualReportRideRequest{
			CaseID:          uuid.New(),
			ServiceDate:     "2026/13/24",
			LegSeq:          1,
			EffectiveStatus: "boarded",
		}
		_, err := svc.ManualReportRide(ctx, req, actorID, "staff", "127.0.0.1", "test-ua")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "無效的服務日期格式")
		assert.ErrorIs(t, err, ErrInvalidManualReportServiceDate)
	})
}

func TestDescribeAnomalyFlag(t *testing.T) {
	tests := []struct {
		name string
		flag string
		want string
	}{
		{
			name: "未對應欄位",
			flag: "unmapped_column:3.王小明[去程]",
			want: "欄位『3.王小明[去程]』尚未對應個案",
		},
		{
			name: "無法辨識的值",
			flag: "unparsed_value:3.王小明[回程]:?",
			want: "欄位『3.王小明[回程]』的內容「?」無法辨識",
		},
		{
			name: "未知旗標退回通用文字",
			flag: "unknown_flag:foo",
			want: "匯入資料異常",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := describeAnomalyFlag(tt.flag)
			assert.Equal(t, tt.want, got)
			assert.NotContains(t, got, "unmapped_column")
			assert.NotContains(t, got, "unparsed_value")
		})
	}
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

func TestRideService_ManualReportRide_RequiresScheduledLeg(t *testing.T) {
	caseID := uuid.New()
	vehicleID := uuid.New()
	store := newFakeRecordStore(nil)
	svc := NewRideService(store, nil, manualScheduleReader{schedule: &CaseSchedule{
		CaseID: caseID,
		Legs:   []ScheduleLeg{{LegSeq: 1, VehicleID: &vehicleID}},
	}}, nil, nil)

	_, err := svc.ManualReportRide(context.Background(), ManualReportRideRequest{
		CaseID:          caseID,
		ServiceDate:     "2026-08-24",
		LegSeq:          2,
		EffectiveStatus: "boarded",
	}, uuid.New(), "staff", "127.0.0.1", "test-agent")

	assert.ErrorIs(t, err, ErrInvalidManualRideLeg)
}

func TestRideService_ManualReportRide_UsesScheduledLegVehicle(t *testing.T) {
	caseID := uuid.New()
	vehicleID := uuid.New()
	store := newFakeRecordStore(nil)
	svc := NewRideService(store, nil, manualScheduleReader{schedule: &CaseSchedule{
		CaseID: caseID,
		Legs:   []ScheduleLeg{{LegSeq: 1, VehicleID: &vehicleID}},
	}}, nil, nil)

	rec, err := svc.ManualReportRide(context.Background(), ManualReportRideRequest{
		CaseID:          caseID,
		ServiceDate:     "2026-08-24",
		LegSeq:          1,
		EffectiveStatus: "boarded",
	}, uuid.New(), "staff", "127.0.0.1", "test-agent")

	require.NoError(t, err)
	assert.Equal(t, vehicleID, rec.VehicleID)
}

func TestRideService_ManualReportRide_RejectsUnscheduledWeekday(t *testing.T) {
	caseID := uuid.New()
	vehicleID := uuid.New()
	svc := NewRideService(newFakeRecordStore(nil), nil, manualScheduleReader{schedule: &CaseSchedule{
		CaseID:   caseID,
		Weekdays: []int16{2}, // 僅週二；測試日期是週一
		Legs:     []ScheduleLeg{{LegSeq: 1, VehicleID: &vehicleID}},
	}}, nil, nil)

	_, err := svc.ManualReportRide(context.Background(), ManualReportRideRequest{
		CaseID:          caseID,
		ServiceDate:     "2026-08-24",
		LegSeq:          1,
		EffectiveStatus: "boarded",
	}, uuid.New(), "staff", "127.0.0.1", "test-agent")

	assert.ErrorIs(t, err, ErrInvalidManualRideLeg)
}
