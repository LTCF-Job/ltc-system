package app

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAttendanceService_GetMonthAttendance(t *testing.T) {
	attendanceRepo := stubAttendanceStore{}
	driverRepo := activeDriverListerStub{drivers: []DriverRef{{ID: uuid.New(), Name: "測試司機", Region: "hsinchu"}}}
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
	driverRepo := activeDriverListerStub{drivers: []DriverRef{{ID: uuid.New(), Name: "測試司機", Region: "hsinchu"}}}
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
	driverRepo := activeDriverListerStub{drivers: []DriverRef{{ID: uuid.New(), Name: "測試司機", Region: "hsinchu"}}}
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
	assert.Equal(t, "manual", item.Source, "使用者在司機月曆手動登記一律視為人工來源")
}

func TestAttendanceService_SyncFromImport_NoExistingRecord_AutoRegistersWork(t *testing.T) {
	store := &recordingAttendanceStore{}
	svc := NewAttendanceService(store, emptyDriverLister{}, discardAuditWriter{}, stubHolidayReader{})

	driverID := uuid.New()
	serviceDate := time.Date(2026, 7, 15, 0, 0, 0, 0, time.UTC)

	err := svc.SyncFromImport(context.Background(), driverID, serviceDate)
	require.NoError(t, err)
	require.Len(t, store.upsertCalls, 1, "當天沒有任何出勤紀錄時應直接自動登記出勤")
	assert.Equal(t, "work", store.upsertCalls[0].Status)
	assert.Equal(t, "import", store.upsertCalls[0].Source)
	assert.Empty(t, store.conflictCalls)
}

func TestAttendanceService_SyncFromImport_ExistingImportSource_Refreshes(t *testing.T) {
	store := &recordingAttendanceStore{existing: &AttendanceRecord{Status: "work", Source: "import"}}
	svc := NewAttendanceService(store, emptyDriverLister{}, discardAuditWriter{}, stubHolidayReader{})

	err := svc.SyncFromImport(context.Background(), uuid.New(), time.Now())
	require.NoError(t, err)
	assert.Len(t, store.upsertCalls, 1, "既有紀錄本身就是上次匯入寫入的，直接刷新不算衝突")
	assert.Empty(t, store.conflictCalls)
}

func TestAttendanceService_SyncFromImport_ManualRecordMatchesImport_NoOp(t *testing.T) {
	store := &recordingAttendanceStore{existing: &AttendanceRecord{Status: "work", Source: "manual"}}
	svc := NewAttendanceService(store, emptyDriverLister{}, discardAuditWriter{}, stubHolidayReader{})

	err := svc.SyncFromImport(context.Background(), uuid.New(), time.Now())
	require.NoError(t, err)
	assert.Empty(t, store.upsertCalls, "人工登記剛好也是出勤時不需要覆蓋")
	assert.Empty(t, store.conflictCalls)
}

func TestAttendanceService_SyncFromImport_ManualRecordDiffersFromImport_RaisesConflict(t *testing.T) {
	store := &recordingAttendanceStore{existing: &AttendanceRecord{Status: "leave", Source: "manual"}}
	svc := NewAttendanceService(store, emptyDriverLister{}, discardAuditWriter{}, stubHolidayReader{})

	driverID := uuid.New()
	serviceDate := time.Date(2026, 7, 15, 0, 0, 0, 0, time.UTC)

	err := svc.SyncFromImport(context.Background(), driverID, serviceDate)
	require.NoError(t, err)
	assert.Empty(t, store.upsertCalls, "人工登記跟匯入結果不同時不得覆蓋人工判斷")
	require.Len(t, store.conflictCalls, 1)
	assert.Equal(t, "leave", store.conflictCalls[0].ExistingStatus)
	assert.Equal(t, "work", store.conflictCalls[0].ImportedStatus)
}

func TestAttendanceService_ResolveConflict_KeepManual_DoesNotTouchAttendanceRecord(t *testing.T) {
	conflictID := uuid.New()
	store := &recordingAttendanceStore{conflicts: map[uuid.UUID]*AttendanceImportConflict{
		conflictID: {ID: conflictID, DriverID: uuid.New(), RecordDate: time.Now(), ExistingStatus: "leave", ImportedStatus: "work", Status: "pending"},
	}}
	svc := NewAttendanceService(store, emptyDriverLister{}, discardAuditWriter{}, stubHolidayReader{})

	dto, err := svc.ResolveConflict(context.Background(), conflictID, "keep_manual", nil, nil)
	require.NoError(t, err)
	assert.Equal(t, "resolved", dto.Status)
	assert.Empty(t, store.upsertCalls, "保留人工登記不應改動 attendance_records")
	assert.Len(t, store.resolveCalls, 1)
}

func TestAttendanceService_ResolveConflict_UseImport_OverwritesAttendanceRecord(t *testing.T) {
	conflictID := uuid.New()
	driverID := uuid.New()
	recordDate := time.Date(2026, 7, 15, 0, 0, 0, 0, time.UTC)
	store := &recordingAttendanceStore{conflicts: map[uuid.UUID]*AttendanceImportConflict{
		conflictID: {ID: conflictID, DriverID: driverID, RecordDate: recordDate, ExistingStatus: "leave", ImportedStatus: "work", Status: "pending"},
	}}
	svc := NewAttendanceService(store, emptyDriverLister{}, discardAuditWriter{}, stubHolidayReader{})

	dto, err := svc.ResolveConflict(context.Background(), conflictID, "use_import", nil, nil)
	require.NoError(t, err)
	assert.Equal(t, "resolved", dto.Status)
	require.Len(t, store.upsertCalls, 1, "改採匯入結果時要覆蓋人工登記")
	assert.Equal(t, "work", store.upsertCalls[0].Status)
	assert.Equal(t, "import", store.upsertCalls[0].Source)
}

func TestAttendanceService_ResolveConflict_NotFound(t *testing.T) {
	store := &recordingAttendanceStore{conflicts: map[uuid.UUID]*AttendanceImportConflict{}}
	svc := NewAttendanceService(store, emptyDriverLister{}, discardAuditWriter{}, stubHolidayReader{})

	_, err := svc.ResolveConflict(context.Background(), uuid.New(), "keep_manual", nil, nil)
	require.ErrorIs(t, err, ErrAttendanceConflictNotFound)
}
