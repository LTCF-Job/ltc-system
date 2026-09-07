package app

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type activeDriverListerStub struct {
	drivers []DriverRef
	paged   []DriverRef
	err     error
}

func (s activeDriverListerStub) List(context.Context, string, string, int, int) ([]DriverRef, int64, error) {
	return s.paged, int64(len(s.drivers)), s.err
}

func (s activeDriverListerStub) ListAllActive(context.Context) ([]DriverRef, error) {
	return s.drivers, s.err
}

type failingHolidayReader struct {
	err error
}

func (r failingHolidayReader) GetHolidayMap(context.Context, int, int, string) (map[string]bool, error) {
	return nil, r.err
}

func TestGetMonthAttendance_DriverQueryErrorFails(t *testing.T) {
	wantErr := errors.New("database unavailable")
	svc := NewAttendanceService(stubAttendanceStore{}, activeDriverListerStub{err: wantErr}, discardAuditWriter{}, stubHolidayReader{})

	report, err := svc.GetMonthAttendance(context.Background(), "115-07", nil)

	assert.ErrorIs(t, err, wantErr)
	assert.Nil(t, report)
}

func TestGetMonthAttendance_EmptyDriverListReturnsEmpty(t *testing.T) {
	svc := NewAttendanceService(stubAttendanceStore{}, activeDriverListerStub{}, discardAuditWriter{}, stubHolidayReader{})

	report, err := svc.GetMonthAttendance(context.Background(), "115-07", nil)

	require.NoError(t, err)
	assert.Empty(t, report.Drivers)
}

func TestGetMonthAttendance_IncludesAllActiveDrivers(t *testing.T) {
	drivers := make([]DriverRef, 101)
	for i := range drivers {
		drivers[i] = DriverRef{ID: uuid.New(), Name: "司機", Region: "hsinchu"}
	}

	firstPage := drivers[:100]
	svc := NewAttendanceService(
		stubAttendanceStore{},
		activeDriverListerStub{drivers: drivers, paged: firstPage},
		discardAuditWriter{},
		stubHolidayReader{},
	)

	report, err := svc.GetMonthAttendance(context.Background(), "115-07", nil)

	require.NoError(t, err)
	assert.Len(t, report.Drivers, len(drivers))
}

func TestGetMonthAttendance_HolidayQueryErrorFails(t *testing.T) {
	wantErr := errors.New("holiday database unavailable")
	svc := NewAttendanceService(
		stubAttendanceStore{},
		activeDriverListerStub{drivers: []DriverRef{{ID: uuid.New(), Name: "司機", Region: "hsinchu"}}},
		discardAuditWriter{},
		failingHolidayReader{err: wantErr},
	)

	report, err := svc.GetMonthAttendance(context.Background(), "115-07", nil)

	assert.ErrorIs(t, err, wantErr)
	assert.Nil(t, report)
}
