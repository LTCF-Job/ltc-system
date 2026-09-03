package app

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func mustDate(t *testing.T, raw string) time.Time {
	t.Helper()
	parsed, err := time.Parse("2006-01-02", raw)
	require.NoError(t, err)
	return parsed
}

// ingestOneDay 寫入單日單欄的匯報，回傳該 slot 的鍵，供各測試斷言重匯後的狀態。
func ingestOneDay(t *testing.T, svc *RideService, formID, vehicleID, caseID uuid.UUID, date, header, value string) {
	t.Helper()
	_, err := svc.IngestSubmission(context.Background(), formID, vehicleID, ProcessSubmissionRequest{
		ServiceDate: mustDate(t, date),
		Answers:     map[string]string{header: value},
	})
	require.NoError(t, err)
}

func TestClearImportedDates_RemovesRecordWhenNoSourceRemains(t *testing.T) {
	caseID, formID, vehicleID := uuid.New(), uuid.New(), uuid.New()
	store := newFakeRecordStore([]FormColumn{mappedColumn(caseID, "吳桂 [去程]", 1, 3)})
	svc := NewRideService(store, fakeDriverResolver{}, fakeScheduleReader{}, nil, nil)

	ingestOneDay(t, svc, formID, vehicleID, caseID, "2026-03-02", "吳桂 [去程]", "有坐")
	require.NotNil(t, store.records[slotKey{caseID, "2026-03-02", 1}])

	removed, err := svc.ClearImportedDates(context.Background(), formID, []time.Time{mustDate(t, "2026-03-02")})

	require.NoError(t, err)
	assert.Equal(t, 1, removed)
	assert.Nil(t, store.records[slotKey{caseID, "2026-03-02", 1}], "來源清空後不應留下沒有依據的搭乘紀錄")
	assert.Empty(t, store.sources)
}

func TestClearImportedDates_KeepsManuallyCorrectedRecord(t *testing.T) {
	caseID, formID, vehicleID := uuid.New(), uuid.New(), uuid.New()
	store := newFakeRecordStore([]FormColumn{mappedColumn(caseID, "吳桂 [去程]", 1, 3)})
	svc := NewRideService(store, fakeDriverResolver{}, fakeScheduleReader{}, nil, nil)

	ingestOneDay(t, svc, formID, vehicleID, caseID, "2026-03-02", "吳桂 [去程]", "有坐")
	correctedAt := time.Now().UTC()
	store.records[slotKey{caseID, "2026-03-02", 1}].CorrectedAt = &correctedAt

	_, err := svc.ClearImportedDates(context.Background(), formID, []time.Time{mustDate(t, "2026-03-02")})

	require.NoError(t, err)
	assert.NotNil(t, store.records[slotKey{caseID, "2026-03-02", 1}], "人工更正過的紀錄不得被覆蓋式重匯刪除")
}

func TestClearImportedDates_KeepsOtherFormsMixedVehicleSources(t *testing.T) {
	caseID, vehicleID := uuid.New(), uuid.New()
	formA, formB := uuid.New(), uuid.New()
	otherVehicleID := uuid.New()
	store := newFakeRecordStore([]FormColumn{mappedColumn(caseID, "吳桂 [去程]", 1, 3)})
	svc := NewRideService(store, fakeDriverResolver{}, fakeScheduleReader{}, nil, nil)

	ingestOneDay(t, svc, formA, vehicleID, caseID, "2026-03-02", "吳桂 [去程]", "有坐")
	ingestOneDay(t, svc, formB, otherVehicleID, caseID, "2026-03-02", "吳桂 [去程]", "有坐")

	_, err := svc.ClearImportedDates(context.Background(), formA, []time.Time{mustDate(t, "2026-03-02")})

	require.NoError(t, err)
	rec := store.records[slotKey{caseID, "2026-03-02", 1}]
	require.NotNil(t, rec, "另一台車仍有來源時，搭乘紀錄必須保留")
	assert.Equal(t, otherVehicleID, rec.VehicleID, "重算後應只反映剩下那台車的來源")

	remaining, err := store.ListRideSourcesForSlot(context.Background(), caseID, mustDate(t, "2026-03-02"), 1)
	require.NoError(t, err)
	assert.Len(t, remaining, 1)
}

func TestClearImportedDates_IgnoresDatesOutsideTheGivenSet(t *testing.T) {
	caseID, formID, vehicleID := uuid.New(), uuid.New(), uuid.New()
	store := newFakeRecordStore([]FormColumn{mappedColumn(caseID, "吳桂 [去程]", 1, 3)})
	svc := NewRideService(store, fakeDriverResolver{}, fakeScheduleReader{}, nil, nil)

	ingestOneDay(t, svc, formID, vehicleID, caseID, "2026-03-02", "吳桂 [去程]", "有坐")
	ingestOneDay(t, svc, formID, vehicleID, caseID, "2026-04-02", "吳桂 [去程]", "有坐")

	removed, err := svc.ClearImportedDates(context.Background(), formID, []time.Time{mustDate(t, "2026-03-02")})

	require.NoError(t, err)
	assert.Equal(t, 1, removed)
	assert.Nil(t, store.records[slotKey{caseID, "2026-03-02", 1}])
	assert.NotNil(t, store.records[slotKey{caseID, "2026-04-02", 1}], "不在清除範圍的日期不得受影響")
}

func TestClearImportedDates_NoDatesIsANoOp(t *testing.T) {
	store := newFakeRecordStore(nil)
	svc := NewRideService(store, fakeDriverResolver{}, fakeScheduleReader{}, nil, nil)

	removed, err := svc.ClearImportedDates(context.Background(), uuid.New(), nil)

	require.NoError(t, err)
	assert.Zero(t, removed)
}

func TestClearImportedDates_KeepsRecordsWithOtherManualMarkers(t *testing.T) {
	resolvedAt := time.Now().UTC()
	tests := []struct {
		name  string
		apply func(rec *RideRecord)
	}{
		{name: "已裁決衝突", apply: func(rec *RideRecord) { rec.ConflictResolvedAt = &resolvedAt }},
		{name: "標記不申報", apply: func(rec *RideRecord) { rec.NotClaimedAA09 = true }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			caseID, formID, vehicleID := uuid.New(), uuid.New(), uuid.New()
			store := newFakeRecordStore([]FormColumn{mappedColumn(caseID, "吳桂 [去程]", 1, 3)})
			svc := NewRideService(store, fakeDriverResolver{}, fakeScheduleReader{}, nil, nil)

			ingestOneDay(t, svc, formID, vehicleID, caseID, "2026-03-02", "吳桂 [去程]", "有坐")
			tt.apply(store.records[slotKey{caseID, "2026-03-02", 1}])

			_, err := svc.ClearImportedDates(context.Background(), formID, []time.Time{mustDate(t, "2026-03-02")})

			require.NoError(t, err)
			assert.NotNil(t, store.records[slotKey{caseID, "2026-03-02", 1}], "人工介入過的紀錄不得被覆蓋式重匯刪除")
		})
	}
}

func TestClearImportedDates_ClearsBothLegsOfFourTripPattern(t *testing.T) {
	caseID, formID, vehicleID := uuid.New(), uuid.New(), uuid.New()
	store := newFakeRecordStore([]FormColumn{mappedColumn(caseID, "吳桂 [去程]", 1, 3)})
	svc := NewRideService(store, fakeDriverResolver{}, fakeScheduleReader{tripPattern: 4}, nil, nil)

	ingestOneDay(t, svc, formID, vehicleID, caseID, "2026-03-02", "吳桂 [去程]", "有坐")
	require.NotNil(t, store.records[slotKey{caseID, "2026-03-02", 1}])
	require.NotNil(t, store.records[slotKey{caseID, "2026-03-02", 3}], "四趟制的第 1 趟會展開成 1、3 趟")

	_, err := svc.ClearImportedDates(context.Background(), formID, []time.Time{mustDate(t, "2026-03-02")})

	require.NoError(t, err)
	assert.Nil(t, store.records[slotKey{caseID, "2026-03-02", 1}])
	assert.Nil(t, store.records[slotKey{caseID, "2026-03-02", 3}], "展開出來的趟次也必須一併清除")
}

func TestClearImportedDates_DoesNotFallBackToTheRemovedVehicle(t *testing.T) {
	caseID := uuid.New()
	formA, formB := uuid.New(), uuid.New()
	removedVehicleID, remainingVehicleID := uuid.New(), uuid.New()
	store := newFakeRecordStore([]FormColumn{mappedColumn(caseID, "吳桂 [去程]", 1, 3)})
	svc := NewRideService(store, fakeDriverResolver{}, fakeScheduleReader{}, nil, nil)

	// 兩台車都回報「沒坐」：merge 找不到有坐的來源，會退回預設車輛
	ingestOneDay(t, svc, formA, removedVehicleID, caseID, "2026-03-02", "吳桂 [去程]", "沒坐")
	ingestOneDay(t, svc, formB, remainingVehicleID, caseID, "2026-03-02", "吳桂 [去程]", "沒坐")

	_, err := svc.ClearImportedDates(context.Background(), formA, []time.Time{mustDate(t, "2026-03-02")})

	require.NoError(t, err)
	rec := store.records[slotKey{caseID, "2026-03-02", 1}]
	require.NotNil(t, rec)
	assert.Equal(t, remainingVehicleID, rec.VehicleID,
		"重算的預設車輛必須取自剩餘來源，不能是剛被清掉的那台車")
	assert.NotEqual(t, removedVehicleID, rec.VehicleID)
}
