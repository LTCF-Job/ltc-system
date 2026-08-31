package app

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type slotKey struct {
	caseID uuid.UUID
	date   string
	legSeq int16
}

// fakeRecordStore 記下寫入的來源列，並在重算時把它們回讀，重現正式流程中
// InsertRideSource → ListRideSourcesForSlot → UpsertRideRecord 的循環。
type fakeRecordStore struct {
	columns        []FormColumn
	sources        map[slotKey][]fakeSource
	records        map[slotKey]*RideRecord
	submissions    map[uuid.UUID]submissionKey
	submission     uuid.UUID
	lastSource     string
	importedMonths []ImportedMonth
	importedErr    error
}

// submissionKey 讓 fake 能像資料庫一樣依 form 與服務日期刪除提交紀錄。
type submissionKey struct {
	formID uuid.UUID
	date   string
}

// fakeSource 保留來源列與其所屬提交，重現 ON DELETE CASCADE 的連帶清除。
type fakeSource struct {
	submissionID uuid.UUID
	row          RideSourceRow
}

func newFakeRecordStore(columns []FormColumn) *fakeRecordStore {
	return &fakeRecordStore{
		columns:     columns,
		sources:     map[slotKey][]fakeSource{},
		records:     map[slotKey]*RideRecord{},
		submissions: map[uuid.UUID]submissionKey{},
	}
}

func (f *fakeRecordStore) GetFormColumns(context.Context, uuid.UUID) ([]FormColumn, error) {
	return f.columns, nil
}

func (f *fakeRecordStore) ListRideSourcesForSlot(_ context.Context, caseID uuid.UUID, serviceDate time.Time, legSeq int16) ([]RideSourceRow, error) {
	stored := f.sources[slotKey{caseID, serviceDate.Format("2006-01-02"), legSeq}]
	if len(stored) == 0 {
		return nil, nil
	}
	out := make([]RideSourceRow, 0, len(stored))
	for _, src := range stored {
		out = append(out, src.row)
	}
	return out, nil
}

func (f *fakeRecordStore) ListCalendarCases(context.Context, time.Time, time.Time, string, string) ([]CalendarCase, error) {
	return nil, nil
}

func (f *fakeRecordStore) ListRideRecordsInRange(context.Context, time.Time, time.Time, string, string) ([]RideRecord, error) {
	return nil, nil
}

func (f *fakeRecordStore) SaveFormSubmission(_ context.Context, formID uuid.UUID, serviceDate, _ time.Time, _ string, _ *uuid.UUID, source string, _ map[string]interface{}, _ string) (uuid.UUID, error) {
	f.submission = uuid.New()
	f.lastSource = source
	f.submissions[f.submission] = submissionKey{formID: formID, date: serviceDate.Format("2006-01-02")}
	return f.submission, nil
}

func (f *fakeRecordStore) InsertRideSource(_ context.Context, submissionID, caseID uuid.UUID, serviceDate time.Time, legSeq int16, vehicleID uuid.UUID, driverID *uuid.UUID, reported string, _ int) error {
	key := slotKey{caseID, serviceDate.Format("2006-01-02"), legSeq}
	f.sources[key] = append(f.sources[key], fakeSource{
		submissionID: submissionID,
		row: RideSourceRow{
			VehicleID:   vehicleID,
			DriverID:    driverID,
			Reported:    reported,
			SubmittedAt: time.Now().UTC(),
		},
	})
	return nil
}

func (f *fakeRecordStore) ListRideSourceSlotsForForm(_ context.Context, formID uuid.UUID, dates []time.Time) ([]RideSlot, error) {
	wanted := map[string]bool{}
	for _, d := range dates {
		wanted[d.Format("2006-01-02")] = true
	}

	var slots []RideSlot
	for key, stored := range f.sources {
		for _, src := range stored {
			sub := f.submissions[src.submissionID]
			if sub.formID != formID || !wanted[sub.date] {
				continue
			}
			parsed, _ := time.Parse("2006-01-02", key.date)
			slots = append(slots, RideSlot{CaseID: key.caseID, ServiceDate: parsed, LegSeq: key.legSeq})
			break
		}
	}
	return slots, nil
}

func (f *fakeRecordStore) ListImportedMonths(context.Context) ([]ImportedMonth, error) {
	return f.importedMonths, f.importedErr
}

func (f *fakeRecordStore) DeleteFormSubmissions(_ context.Context, formID uuid.UUID, dates []time.Time) (int, error) {
	wanted := map[string]bool{}
	for _, d := range dates {
		wanted[d.Format("2006-01-02")] = true
	}

	removed := map[uuid.UUID]bool{}
	for id, sub := range f.submissions {
		if sub.formID == formID && wanted[sub.date] {
			removed[id] = true
			delete(f.submissions, id)
		}
	}

	for key, stored := range f.sources {
		kept := stored[:0]
		for _, src := range stored {
			if !removed[src.submissionID] {
				kept = append(kept, src)
			}
		}
		if len(kept) == 0 {
			delete(f.sources, key)
			continue
		}
		f.sources[key] = kept
	}
	return len(removed), nil
}

func (f *fakeRecordStore) DeleteDerivedRideRecord(_ context.Context, caseID uuid.UUID, serviceDate time.Time, legSeq int16) error {
	key := slotKey{caseID, serviceDate.Format("2006-01-02"), legSeq}
	rec := f.records[key]
	if rec == nil {
		return nil
	}
	if rec.CorrectedAt != nil || rec.ConflictResolvedAt != nil || rec.NotClaimedAA09 {
		return nil
	}
	delete(f.records, key)
	return nil
}

func (f *fakeRecordStore) GetRideRecordForSlot(_ context.Context, caseID uuid.UUID, serviceDate time.Time, legSeq int16) (*RideRecord, error) {
	return f.records[slotKey{caseID, serviceDate.Format("2006-01-02"), legSeq}], nil
}

func (f *fakeRecordStore) UpsertRideRecord(_ context.Context, rec *RideRecord) error {
	copied := *rec
	f.records[slotKey{rec.CaseID, rec.ServiceDate.Format("2006-01-02"), rec.LegSeq}] = &copied
	return nil
}

func (f *fakeRecordStore) CorrectRideRecord(context.Context, uuid.UUID, *string, *uuid.UUID, *uuid.UUID, *string, *int16, *bool, *string, uuid.UUID) error {
	return nil
}

type fakeScheduleReader struct{ tripPattern int16 }

func (f fakeScheduleReader) GetActiveScheduleForCaseOnDate(_ context.Context, caseID uuid.UUID, _ time.Time) (*CaseSchedule, error) {
	if f.tripPattern == 0 {
		return nil, nil
	}
	return &CaseSchedule{CaseID: caseID, TripPattern: f.tripPattern}, nil
}

type fakeDriverResolver struct{}

func (fakeDriverResolver) GetByNameNormalized(context.Context, string) (*DriverRef, error) {
	return nil, nil
}

func (fakeDriverResolver) ListDriversForVehicleOnDate(context.Context, uuid.UUID, time.Time) ([]DriverRef, error) {
	return nil, nil
}

func mappedColumn(caseID uuid.UUID, header string, legSeq int16, colIdx int) FormColumn {
	return FormColumn{
		ID:            uuid.New(),
		ColumnIndex:   colIdx,
		ColumnHeader:  header,
		MappingStatus: "mapped",
		CaseID:        &caseID,
		LegSeq:        &legSeq,
	}
}

func TestIngestSubmission_WritesReportedStatusVerbatim(t *testing.T) {
	caseID := uuid.New()
	vehicleID := uuid.New()
	serviceDate := time.Date(2026, 3, 2, 0, 0, 0, 0, time.UTC)

	store := newFakeRecordStore([]FormColumn{
		mappedColumn(caseID, "1.吳桂 [去程]", 1, 3),
		mappedColumn(caseID, "1.吳桂 [回程]", 2, 4),
	})
	svc := NewRideService(store, fakeDriverResolver{}, fakeScheduleReader{}, nil)

	written, err := svc.IngestSubmission(context.Background(), uuid.New(), vehicleID, ProcessSubmissionRequest{
		ServiceDate: serviceDate,
		DriverRaw:   "林彥衡",
		Answers: map[string]string{
			"1.吳桂 [去程]": "有坐",
			"1.吳桂 [回程]": "沒坐",
		},
	})
	require.NoError(t, err)
	assert.Equal(t, 2, written)
	assert.Equal(t, "import", store.lastSource)

	outbound := store.records[slotKey{caseID, "2026-03-02", 1}]
	inbound := store.records[slotKey{caseID, "2026-03-02", 2}]
	require.NotNil(t, outbound)
	require.NotNil(t, inbound)

	assert.Equal(t, "boarded", outbound.EffectiveStatus)
	// 匯報「沒坐」必須留在 absent；先前重算是以固定的 "boarded" 當唯一來源，會翻成有坐
	assert.Equal(t, "absent", inbound.MergedStatus)
	assert.Equal(t, "absent", inbound.EffectiveStatus)
}

func TestIngestSubmission_SkipsUnmappedAndNonReportValues(t *testing.T) {
	caseID := uuid.New()
	store := newFakeRecordStore([]FormColumn{
		mappedColumn(caseID, "1.吳桂 [去程]", 1, 3),
		{ID: uuid.New(), ColumnIndex: 4, ColumnHeader: "2.李四 [去程]", MappingStatus: "pending"},
	})
	svc := NewRideService(store, fakeDriverResolver{}, fakeScheduleReader{}, nil)

	written, err := svc.IngestSubmission(context.Background(), uuid.New(), uuid.New(), ProcessSubmissionRequest{
		ServiceDate: time.Date(2026, 3, 2, 0, 0, 0, 0, time.UTC),
		Answers: map[string]string{
			"1.吳桂 [去程]": "",
			"2.李四 [去程]": "有坐",
		},
	})
	require.NoError(t, err)
	assert.Equal(t, 0, written, "空白值不建立來源紀錄，未對應欄位不處理")
	assert.Empty(t, store.records)
}

func TestIngestSubmission_ExpandsFourTripPattern(t *testing.T) {
	caseID := uuid.New()
	store := newFakeRecordStore([]FormColumn{mappedColumn(caseID, "1.吳桂 [去程]", 1, 3)})
	svc := NewRideService(store, fakeDriverResolver{}, fakeScheduleReader{tripPattern: 4}, nil)

	written, err := svc.IngestSubmission(context.Background(), uuid.New(), uuid.New(), ProcessSubmissionRequest{
		ServiceDate: time.Date(2026, 3, 2, 0, 0, 0, 0, time.UTC),
		Answers:     map[string]string{"1.吳桂 [去程]": "有坐"},
	})
	require.NoError(t, err)
	assert.Equal(t, 2, written, "四趟制的表單第 1 趟展開為第 1、3 趟")
	assert.NotNil(t, store.records[slotKey{caseID, "2026-03-02", 1}])
	assert.NotNil(t, store.records[slotKey{caseID, "2026-03-02", 3}])
}

func TestIngestSubmission_RequiresServiceDate(t *testing.T) {
	svc := NewRideService(newFakeRecordStore(nil), fakeDriverResolver{}, fakeScheduleReader{}, nil)

	_, err := svc.IngestSubmission(context.Background(), uuid.New(), uuid.New(), ProcessSubmissionRequest{})
	assert.Error(t, err)
}

func TestExpandLegSeqs(t *testing.T) {
	tests := []struct {
		name     string
		baseLeg  int16
		schedule *CaseSchedule
		want     []int16
	}{
		{name: "無排班資料維持原趟次", baseLeg: 1, schedule: nil, want: []int16{1}},
		{name: "兩趟制不展開", baseLeg: 2, schedule: &CaseSchedule{TripPattern: 2}, want: []int16{2}},
		{name: "四趟制第 1 趟展開為 1、3", baseLeg: 1, schedule: &CaseSchedule{TripPattern: 4}, want: []int16{1, 3}},
		{name: "四趟制第 2 趟展開為 2、4", baseLeg: 2, schedule: &CaseSchedule{TripPattern: 4}, want: []int16{2, 4}},
		{name: "四趟制第 3 趟不再展開", baseLeg: 3, schedule: &CaseSchedule{TripPattern: 4}, want: []int16{3}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, expandLegSeqs(tt.baseLeg, tt.schedule))
		})
	}
}
