package app

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestAttendanceService_GetMonthAttendance(t *testing.T) {
	attendanceRepo := stubAttendanceStore{}
	driverRepo := emptyDriverLister{}
	auditRepo := discardAuditWriter{}

	svc := NewAttendanceService(attendanceRepo, driverRepo, auditRepo, stubHolidayReader{})
	ctx := context.Background()

	report, err := svc.GetMonthAttendance(ctx, "115-07", nil)
	assert.NoError(t, err)
	assert.NotNil(t, report)
	assert.Equal(t, "115-07", report.PeriodYM)
	assert.Equal(t, 31, report.DaysInMonth)
}

func TestAttendanceService_GetMonthAttendance_FlagsAbsentForPastWeekdayWithoutRecord(t *testing.T) {
	attendanceRepo := stubAttendanceStore{}
	driverRepo := emptyDriverLister{}
	auditRepo := discardAuditWriter{}

	svc := NewAttendanceService(attendanceRepo, driverRepo, auditRepo, stubHolidayReader{})
	ctx := context.Background()

	past := time.Now().UTC().AddDate(0, -2, 0)
	periodYm := fmt.Sprintf("%d-%02d", past.Year()-1911, int(past.Month()))

	report, err := svc.GetMonthAttendance(ctx, periodYm, nil)
	assert.NoError(t, err)
	assert.NotEmpty(t, report.Drivers)

	driver := report.Drivers[0]
	var foundAbsent, foundOff bool
	for _, day := range driver.Days {
		switch day.Status {
		case "absent":
			foundAbsent = true
		case "off":
			foundOff = true
		}
	}
	assert.True(t, foundAbsent, "已過去且無紀錄的平日應標示為 absent（漏報）")
	assert.True(t, foundOff, "週末無紀錄應標示為 off（休）")
	assert.Equal(t, report.DaysInMonth, driver.WorkDays+driver.LeaveDays+driver.SickDays+driver.OffDays+driver.AbsentDays)
}

func TestAttendanceService_GetMonthAttendance_HolidayWeekdayMarkedOff(t *testing.T) {
	attendanceRepo := stubAttendanceStore{}
	driverRepo := emptyDriverLister{}
	auditRepo := discardAuditWriter{}

	past := time.Now().UTC().AddDate(0, -2, 0)
	holidayDate := time.Date(past.Year(), past.Month(), 1, 0, 0, 0, 0, time.UTC)
	for holidayDate.Weekday() == time.Saturday || holidayDate.Weekday() == time.Sunday {
		holidayDate = holidayDate.AddDate(0, 0, 1)
	}
	dateKey := holidayDate.Format("2006-01-02")
	holidayRepo := fixedHolidayReader{dates: map[string]bool{dateKey: true}}

	svc := NewAttendanceService(attendanceRepo, driverRepo, auditRepo, holidayRepo)
	periodYm := fmt.Sprintf("%d-%02d", past.Year()-1911, int(past.Month()))

	report, err := svc.GetMonthAttendance(context.Background(), periodYm, nil)
	assert.NoError(t, err)
	assert.NotEmpty(t, report.Drivers)
	assert.Equal(t, "off", report.Drivers[0].Days[dateKey].Status, "國定假日的平日應標示為 off（休），而非 absent")
}

func TestAttendanceService_Upsert(t *testing.T) {
	attendanceRepo := stubAttendanceStore{}
	driverRepo := emptyDriverLister{}
	auditRepo := discardAuditWriter{}

	svc := NewAttendanceService(attendanceRepo, driverRepo, auditRepo, stubHolidayReader{})
	ctx := context.Background()

	driverID := uuid.New()
	recDate := time.Date(2026, 7, 15, 0, 0, 0, 0, time.UTC)
	note := "事假半天"

	item, err := svc.Upsert(ctx, driverID, recDate, "leave", &note, nil, nil)
	assert.NoError(t, err)
	assert.NotNil(t, item)
	assert.Equal(t, "leave", item.Status)
}
