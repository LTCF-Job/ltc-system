package app

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// 以下 test double 讓 use case 測試不需要資料庫或 Excel 產生器。

type emptyDriverLister struct{}

func (emptyDriverLister) List(context.Context, string, string, int, int) ([]DriverRef, int64, error) {
	return nil, 0, nil
}

type emptyVehicleLister struct{}

func (emptyVehicleLister) List(context.Context, string, string, int, int) ([]VehicleRef, int64, error) {
	return nil, 0, nil
}

type discardAuditWriter struct{}

func (discardAuditWriter) Write(context.Context, AuditEntry) error { return nil }

type stubAttendanceStore struct{}

func (stubAttendanceStore) GetMonthRecords(context.Context, time.Time, time.Time, *uuid.UUID) ([]AttendanceRecord, error) {
	return nil, nil
}

func (stubAttendanceStore) Upsert(_ context.Context, driverID uuid.UUID, recordDate time.Time, status string, note *string) (*AttendanceRecord, error) {
	return &AttendanceRecord{ID: uuid.New(), DriverID: driverID, RecordDate: recordDate, Status: status, Note: note}, nil
}

type stubHolidayReader struct{}

func (stubHolidayReader) GetHolidayMap(context.Context, int, int, string) (map[string]bool, error) {
	return map[string]bool{}, nil
}

type fixedHolidayReader struct {
	dates map[string]bool
}

func (f fixedHolidayReader) GetHolidayMap(context.Context, int, int, string) (map[string]bool, error) {
	return f.dates, nil
}

type stubMaintenanceStore struct{}

func (stubMaintenanceStore) List(context.Context, int, int, *uuid.UUID, *time.Time, *time.Time, string) ([]MaintenanceLog, int, error) {
	return nil, 0, nil
}
func (stubMaintenanceStore) Create(_ context.Context, item *MaintenanceLog) error {
	item.ID = uuid.New()
	item.CreatedAt = time.Now()
	return nil
}
func (stubMaintenanceStore) Update(context.Context, *MaintenanceLog) error { return nil }
func (stubMaintenanceStore) Delete(context.Context, uuid.UUID) error       { return nil }

type stubTemplateRenderer struct{}

func (stubTemplateRenderer) RenderBlankMaintenanceTemplate([]VehicleLabel) ([]byte, error) {
	return []byte{}, nil
}
