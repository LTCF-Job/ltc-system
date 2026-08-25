package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"ltc-system/apps/api/internal/repository"
)

func TestAttendanceService_GetMonthAttendance(t *testing.T) {
	attendanceRepo := repository.NewAttendanceRepository(nil)
	driverRepo := repository.NewDriverRepository(nil)
	auditRepo := repository.NewAuditRepository(nil)

	svc := NewAttendanceService(attendanceRepo, driverRepo, auditRepo)
	ctx := context.Background()

	report, err := svc.GetMonthAttendance(ctx, "115-07", nil)
	assert.NoError(t, err)
	assert.NotNil(t, report)
	assert.Equal(t, "115-07", report.PeriodYM)
	assert.Equal(t, 31, report.DaysInMonth)
}

func TestAttendanceService_Upsert(t *testing.T) {
	attendanceRepo := repository.NewAttendanceRepository(nil)
	driverRepo := repository.NewDriverRepository(nil)
	auditRepo := repository.NewAuditRepository(nil)

	svc := NewAttendanceService(attendanceRepo, driverRepo, auditRepo)
	ctx := context.Background()

	driverID := uuid.New()
	recDate := time.Date(2026, 7, 15, 0, 0, 0, 0, time.UTC)
	note := "事假半天"

	item, err := svc.Upsert(ctx, driverID, recDate, "leave", &note, nil, nil)
	assert.NoError(t, err)
	assert.NotNil(t, item)
	assert.Equal(t, "leave", item.Status)
}
