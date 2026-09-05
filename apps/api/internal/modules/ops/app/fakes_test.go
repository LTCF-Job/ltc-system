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

func (emptyDriverLister) ListAllActive(context.Context) ([]DriverRef, error) {
	return nil, nil
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

func (stubAttendanceStore) GetOne(context.Context, uuid.UUID, time.Time) (*AttendanceRecord, error) {
	return nil, nil
}

func (stubAttendanceStore) Upsert(_ context.Context, driverID uuid.UUID, recordDate time.Time, status string, note *string, source string) (*AttendanceRecord, error) {
	return &AttendanceRecord{ID: uuid.New(), DriverID: driverID, RecordDate: recordDate, Status: status, Note: note, Source: source}, nil
}

func (stubAttendanceStore) UpsertConflict(context.Context, uuid.UUID, time.Time, string, string) error {
	return nil
}

func (stubAttendanceStore) ListConflicts(context.Context, string) ([]AttendanceImportConflict, error) {
	return nil, nil
}

func (stubAttendanceStore) GetConflict(context.Context, uuid.UUID) (*AttendanceImportConflict, error) {
	return nil, nil
}

func (stubAttendanceStore) ResolveConflict(context.Context, uuid.UUID, string, *uuid.UUID) error {
	return nil
}

// recordingAttendanceStore 是可設定既有紀錄的 fake，供 SyncFromImport／ResolveConflict
// 的分支邏輯測試斷言實際寫入了什麼。
type recordingAttendanceStore struct {
	existing      *AttendanceRecord
	conflicts     map[uuid.UUID]*AttendanceImportConflict
	upsertCalls   []AttendanceRecord
	conflictCalls []AttendanceImportConflict
	resolveCalls  []uuid.UUID
}

func (s *recordingAttendanceStore) GetMonthRecords(context.Context, time.Time, time.Time, *uuid.UUID) ([]AttendanceRecord, error) {
	return nil, nil
}

func (s *recordingAttendanceStore) GetOne(context.Context, uuid.UUID, time.Time) (*AttendanceRecord, error) {
	return s.existing, nil
}

func (s *recordingAttendanceStore) Upsert(_ context.Context, driverID uuid.UUID, recordDate time.Time, status string, note *string, source string) (*AttendanceRecord, error) {
	item := &AttendanceRecord{ID: uuid.New(), DriverID: driverID, RecordDate: recordDate, Status: status, Note: note, Source: source}
	s.upsertCalls = append(s.upsertCalls, *item)
	return item, nil
}

func (s *recordingAttendanceStore) UpsertConflict(_ context.Context, driverID uuid.UUID, recordDate time.Time, existingStatus, importedStatus string) error {
	s.conflictCalls = append(s.conflictCalls, AttendanceImportConflict{
		DriverID: driverID, RecordDate: recordDate, ExistingStatus: existingStatus, ImportedStatus: importedStatus, Status: "pending",
	})
	return nil
}

func (s *recordingAttendanceStore) ListConflicts(context.Context, string) ([]AttendanceImportConflict, error) {
	out := make([]AttendanceImportConflict, 0, len(s.conflicts))
	for _, c := range s.conflicts {
		out = append(out, *c)
	}
	return out, nil
}

func (s *recordingAttendanceStore) GetConflict(_ context.Context, id uuid.UUID) (*AttendanceImportConflict, error) {
	if s.conflicts == nil {
		return nil, nil
	}
	return s.conflicts[id], nil
}

func (s *recordingAttendanceStore) ResolveConflict(_ context.Context, id uuid.UUID, choice string, actorID *uuid.UUID) error {
	s.resolveCalls = append(s.resolveCalls, id)
	if c, ok := s.conflicts[id]; ok {
		c.Status = "resolved"
		c.ResolvedChoice = &choice
	}
	return nil
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
