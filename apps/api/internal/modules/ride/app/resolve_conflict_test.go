package app

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeAuditWriter struct {
	entries []AuditEntry
	err     error
}

func (f *fakeAuditWriter) Write(_ context.Context, e AuditEntry) error {
	if f.err != nil {
		return f.err
	}
	f.entries = append(f.entries, e)
	return nil
}

type fakeMissingProvider struct {
	rides []MissingRide
	err   error
}

func (f *fakeMissingProvider) ListMissingForMonth(context.Context, int, int, string) ([]MissingRide, error) {
	return f.rides, f.err
}

func TestRideService_GetRecord_NotFound(t *testing.T) {
	store := newFakeRecordStore(nil)
	svc := NewRideService(store, fakeDriverResolver{}, fakeScheduleReader{}, nil, nil)

	_, err := svc.GetRecord(context.Background(), uuid.New())
	assert.ErrorIs(t, err, ErrRideNotFound)
}

func TestRideService_GetRecord_Success(t *testing.T) {
	rideID := uuid.New()
	store := newFakeRecordStore(nil)
	store.getByIDResult = &RideRecord{ID: rideID, CaseName: "測試個案"}
	svc := NewRideService(store, fakeDriverResolver{}, fakeScheduleReader{}, nil, nil)

	rec, err := svc.GetRecord(context.Background(), rideID)
	require.NoError(t, err)
	assert.Equal(t, "測試個案", rec.CaseName)
}

func TestRideService_ResolveConflict_NotFound(t *testing.T) {
	store := newFakeRecordStore(nil)
	audit := &fakeAuditWriter{}
	svc := NewRideService(store, fakeDriverResolver{}, fakeScheduleReader{}, audit, nil)

	err := svc.ResolveConflict(context.Background(), uuid.New(), ResolveConflictInput{VehicleID: uuid.New()}, uuid.New(), "admin")
	assert.ErrorIs(t, err, ErrRideNotFound)
	assert.Empty(t, audit.entries)
}

func TestRideService_ResolveConflict_AlreadyResolved(t *testing.T) {
	rideID := uuid.New()
	store := newFakeRecordStore(nil)
	store.getByIDResult = &RideRecord{ID: rideID}
	store.resolveResult = false
	audit := &fakeAuditWriter{}
	svc := NewRideService(store, fakeDriverResolver{}, fakeScheduleReader{}, audit, nil)

	err := svc.ResolveConflict(context.Background(), rideID, ResolveConflictInput{VehicleID: uuid.New()}, uuid.New(), "admin")
	assert.ErrorIs(t, err, ErrConflictAlreadyResolved)
	assert.Empty(t, audit.entries)
}

func TestRideService_ResolveConflict_Success(t *testing.T) {
	rideID := uuid.New()
	vehicleID := uuid.New()
	store := newFakeRecordStore(nil)
	store.getByIDResult = &RideRecord{ID: rideID, CaseName: "不應寫入稽核的個案姓名", VehicleID: uuid.New()}
	store.resolveResult = true
	audit := &fakeAuditWriter{}
	svc := NewRideService(store, fakeDriverResolver{}, fakeScheduleReader{}, audit, nil)

	reason := "與司機確認後由竹北一車承載"
	err := svc.ResolveConflict(context.Background(), rideID, ResolveConflictInput{VehicleID: vehicleID, Reason: &reason}, uuid.New(), "admin")
	require.NoError(t, err)
	require.Len(t, audit.entries, 1)
	assert.Equal(t, "resolve_conflict", audit.entries[0].Action)
	snapshot, ok := audit.entries[0].BeforeData.(rideAuditSnapshot)
	require.True(t, ok)
	assert.Equal(t, rideID, snapshot.ID)
	assert.Equal(t, store.getByIDResult.VehicleID, snapshot.VehicleID)
}

func TestRideService_ResolveConflict_AuditFailureNoRollback(t *testing.T) {
	// 稽核寫入失敗時，裁決結果已經寫入資料庫，不做交易回滾（符合現行慣例）；
	// 呼叫端只會收到稽核失敗的錯誤，資料層面的裁決已生效。
	rideID := uuid.New()
	store := newFakeRecordStore(nil)
	store.getByIDResult = &RideRecord{ID: rideID}
	store.resolveResult = true
	audit := &fakeAuditWriter{err: assert.AnError}
	svc := NewRideService(store, fakeDriverResolver{}, fakeScheduleReader{}, audit, nil)

	err := svc.ResolveConflict(context.Background(), rideID, ResolveConflictInput{VehicleID: uuid.New()}, uuid.New(), "admin")
	assert.NoError(t, err, "稽核寫入失敗只記 log，不讓已成功的裁決回報成失敗")
	assert.True(t, store.conflictResolved)
}

func TestRideService_ListIssues_UnknownType(t *testing.T) {
	store := newFakeRecordStore(nil)
	svc := NewRideService(store, fakeDriverResolver{}, fakeScheduleReader{}, nil, nil)

	_, _, err := svc.ListIssues(context.Background(), "not_a_type", 2026, 7, "", "", 1, 20)
	assert.Error(t, err)
}

func TestRideService_ListIssues_Conflict(t *testing.T) {
	caseID := uuid.New()
	store := newFakeRecordStore(nil)
	store.pendingConflicts = []ConflictRide{
		{ID: uuid.New(), CaseID: caseID, CaseName: "測試個案", ServiceDate: time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC), LegSeq: 1, Vehicles: []string{"竹北一車", "竹北二車"}},
	}
	svc := NewRideService(store, fakeDriverResolver{}, fakeScheduleReader{}, nil, nil)

	items, total, err := svc.ListIssues(context.Background(), "conflict", 2026, 7, "", "", 1, 20)
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	require.Len(t, items, 1)
	assert.Contains(t, items[0].Description, "2")
	assert.Equal(t, []string{"竹北一車", "竹北二車"}, items[0].Vehicles)
}

func TestRideService_ListIssues_ImportError(t *testing.T) {
	store := newFakeRecordStore(nil)
	store.importErrors = []ImportErrorSubmission{
		{ID: uuid.New(), ServiceDate: time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC), DriverNameRaw: "林彥衡", AnomalyFlags: []string{"unparsed_value:1.吳桂 [去程]:半坐"}, RawPayload: `{"a":1}`},
	}
	svc := NewRideService(store, fakeDriverResolver{}, fakeScheduleReader{}, nil, nil)

	items, total, err := svc.ListIssues(context.Background(), "import_error", 2026, 7, "", "", 1, 20)
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	require.Len(t, items, 1)
	assert.Equal(t, "林彥衡", items[0].CaseName)
	assert.Contains(t, items[0].Description, "unparsed_value")
}

func TestRideService_ListIssues_Unreported(t *testing.T) {
	caseID := uuid.New()
	provider := &fakeMissingProvider{rides: []MissingRide{
		{CaseID: caseID, CaseName: "測試個案", ServiceDate: time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC), LegSeq: 1},
	}}
	store := newFakeRecordStore(nil)
	svc := NewRideService(store, fakeDriverResolver{}, fakeScheduleReader{}, nil, provider)

	items, total, err := svc.ListIssues(context.Background(), "unreported", 2026, 7, "", "", 1, 20)
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	require.Len(t, items, 1)
	assert.Equal(t, "測試個案", items[0].CaseName)
}

func TestDetectSubmissionAnomalies(t *testing.T) {
	caseID := uuid.New()
	legSeq := int16(1)
	mapped := FormColumn{ColumnHeader: "1.吳桂 [去程]", MappingStatus: "mapped", CaseID: &caseID, LegSeq: &legSeq}
	unmapped := FormColumn{ColumnHeader: "2.李四 [去程]", MappingStatus: "pending"}

	tests := []struct {
		name    string
		columns []FormColumn
		answers map[string]string
		want    []string
	}{
		{
			name:    "空白值不算異常",
			columns: []FormColumn{mapped},
			answers: map[string]string{"1.吳桂 [去程]": ""},
			want:    nil,
		},
		{
			name:    "已對應欄位值無法辨識",
			columns: []FormColumn{mapped},
			answers: map[string]string{"1.吳桂 [去程]": "半坐"},
			want:    []string{"unparsed_value:1.吳桂 [去程]:半坐"},
		},
		{
			name:    "未完成對應欄位有值",
			columns: []FormColumn{unmapped},
			answers: map[string]string{"2.李四 [去程]": "有坐"},
			want:    []string{"unmapped_column:2.李四 [去程]"},
		},
		{
			name:    "正常值不算異常",
			columns: []FormColumn{mapped},
			answers: map[string]string{"1.吳桂 [去程]": "有坐"},
			want:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, detectSubmissionAnomalies(tt.columns, tt.answers))
		})
	}
}
