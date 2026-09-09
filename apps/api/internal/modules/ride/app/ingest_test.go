package app

import (
	"context"
	"sort"
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

// rowConflictSlotKey 是「同車同個案」衝突的識別鍵，只計入未解決的衝突，
// 重現真實資料庫 partial unique index 的語意。
type rowConflictSlotKey struct {
	vehicleID uuid.UUID
	caseID    uuid.UUID
	date      string
	legSeq    int16
}

// fakeRowConflict 保留一筆暫存衝突及其解決狀態，供測試斷言。
type fakeRowConflict struct {
	id         uuid.UUID
	input      RowConflictInput
	resolvedAt *time.Time
	resolution string
}

// fakeRecordStore 記下寫入的來源列，並在重算時把它們回讀，重現正式流程中
// InsertRideSource → ListRideSourcesForSlot → UpsertRideRecord 的循環。
type fakeRecordStore struct {
	columns          []FormColumn
	sources          map[slotKey][]fakeSource
	records          map[slotKey]*RideRecord
	submissions      map[uuid.UUID]submissionKey
	payloads         map[uuid.UUID]map[string]interface{}
	submission       uuid.UUID
	lastSource       string
	lastAnomalyFlags []string
	importedMonths   []ImportedMonth
	importedErr      error
	conflictResolved bool
	resolveErr       error
	resolveResult    bool
	pendingConflicts []ConflictRide
	importErrors     []ImportErrorSubmission
	getByIDResult    *RideRecord
	getByIDErr       error

	submissionsForForms    []SubmissionFull
	submissionsForFormsErr error
	unmatchedDrivers       []UnmatchedDriverSubmission
	unmatchedDriversErr    error

	updateSubmissionDriverErr error
	updatedSubmissionDrivers  []submissionDriverUpdate

	monthSubmissions    []MonthSubmissionDetail
	monthSubmissionsErr error
	monthRideEntries    []MonthRideEntry
	monthRideEntriesErr error

	rowConflicts     map[uuid.UUID]*fakeRowConflict
	openRowConflicts map[rowConflictSlotKey]uuid.UUID
}

// submissionDriverUpdate 保留一次提交紀錄司機回填的參數，供測試斷言。
type submissionDriverUpdate struct {
	submissionID uuid.UUID
	driverID     uuid.UUID
}

// submissionKey 讓 fake 能像資料庫一樣依 form 與服務日期查詢提交紀錄，並保留司機與
// 上傳時間，供 ListSubmissionAnswersForColumn 重現真實查詢會回傳的欄位。
type submissionKey struct {
	formID      uuid.UUID
	date        string
	driverID    *uuid.UUID
	submittedAt time.Time
}

// fakeSource 保留來源列與其所屬提交。
type fakeSource struct {
	submissionID uuid.UUID
	row          RideSourceRow
}

func newFakeRecordStore(columns []FormColumn) *fakeRecordStore {
	return &fakeRecordStore{
		columns:          columns,
		sources:          map[slotKey][]fakeSource{},
		records:          map[slotKey]*RideRecord{},
		submissions:      map[uuid.UUID]submissionKey{},
		payloads:         map[uuid.UUID]map[string]interface{}{},
		rowConflicts:     map[uuid.UUID]*fakeRowConflict{},
		openRowConflicts: map[rowConflictSlotKey]uuid.UUID{},
	}
}

func (f *fakeRecordStore) GetFormColumns(context.Context, uuid.UUID) ([]FormColumn, error) {
	return f.columns, nil
}

// ListRideSourcesForSlot 依 SubmittedAt 由新到舊排序，重現真實 SQL 的 ORDER BY，
// 讓 reconcileRideSource 依「這台車最新的一筆」比對的邏輯在測試裡也成立。
func (f *fakeRecordStore) ListRideSourcesForSlot(_ context.Context, caseID uuid.UUID, serviceDate time.Time, legSeq int16) ([]RideSourceRow, error) {
	stored := f.sources[slotKey{caseID, serviceDate.Format("2006-01-02"), legSeq}]
	if len(stored) == 0 {
		return nil, nil
	}
	out := make([]RideSourceRow, 0, len(stored))
	for _, src := range stored {
		out = append(out, src.row)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].SubmittedAt.After(out[j].SubmittedAt) })
	return out, nil
}

func (f *fakeRecordStore) ListCalendarCases(context.Context, time.Time, time.Time, string, string) ([]CalendarCase, error) {
	return nil, nil
}

func (f *fakeRecordStore) ListRideRecordsInRange(context.Context, time.Time, time.Time, string, string) ([]RideRecord, error) {
	return nil, nil
}

func (f *fakeRecordStore) SaveFormSubmission(_ context.Context, formID uuid.UUID, serviceDate, submittedAt time.Time, _ string, driverID *uuid.UUID, source string, payload map[string]interface{}, _ string, anomalyFlags []string) (uuid.UUID, error) {
	f.submission = uuid.New()
	f.lastSource = source
	f.lastAnomalyFlags = anomalyFlags
	f.submissions[f.submission] = submissionKey{formID: formID, date: serviceDate.Format("2006-01-02"), driverID: driverID, submittedAt: submittedAt}
	f.payloads[f.submission] = payload
	return f.submission, nil
}

// ListSubmissionAnswersForColumn 重現真實 SQL 的 payload->'answers'->>header 查詢，
// 只回傳這個表單裡、這一欄留有原始儲存格文字的既有提交。
func (f *fakeRecordStore) ListSubmissionAnswersForColumn(_ context.Context, formID uuid.UUID, columnHeader string) ([]SubmissionAnswer, error) {
	var out []SubmissionAnswer
	for id, sub := range f.submissions {
		if sub.formID != formID {
			continue
		}
		answers, _ := f.payloads[id]["answers"].(map[string]string)
		value, ok := answers[columnHeader]
		if !ok {
			continue
		}
		date, err := time.Parse("2006-01-02", sub.date)
		if err != nil {
			return nil, err
		}
		out = append(out, SubmissionAnswer{SubmissionID: id, ServiceDate: date, SubmittedAt: sub.submittedAt, DriverID: sub.driverID, Value: value})
	}
	return out, nil
}

func (f *fakeRecordStore) InsertRideSource(_ context.Context, submissionID, caseID uuid.UUID, serviceDate time.Time, legSeq int16, vehicleID uuid.UUID, driverID *uuid.UUID, reported string, _ int, submittedAt time.Time) error {
	key := slotKey{caseID, serviceDate.Format("2006-01-02"), legSeq}
	f.sources[key] = append(f.sources[key], fakeSource{
		submissionID: submissionID,
		row: RideSourceRow{
			SubmissionID: submissionID,
			VehicleID:    vehicleID,
			DriverID:     driverID,
			Reported:     reported,
			SubmittedAt:  submittedAt,
		},
	})
	return nil
}

func (f *fakeRecordStore) ListImportedMonths(context.Context) ([]ImportedMonth, error) {
	return f.importedMonths, f.importedErr
}

func (f *fakeRecordStore) ListSubmissionsForForms(context.Context, []uuid.UUID) ([]SubmissionFull, error) {
	return f.submissionsForForms, f.submissionsForFormsErr
}

func (f *fakeRecordStore) ListUnmatchedDriverSubmissions(context.Context) ([]UnmatchedDriverSubmission, error) {
	return f.unmatchedDrivers, f.unmatchedDriversErr
}

func (f *fakeRecordStore) UpdateSubmissionDriverID(_ context.Context, submissionID, driverID uuid.UUID) error {
	f.updatedSubmissionDrivers = append(f.updatedSubmissionDrivers, submissionDriverUpdate{submissionID: submissionID, driverID: driverID})
	return f.updateSubmissionDriverErr
}

func (f *fakeRecordStore) ListSubmissionsForFormMonth(context.Context, uuid.UUID, time.Time, time.Time) ([]MonthSubmissionDetail, error) {
	return f.monthSubmissions, f.monthSubmissionsErr
}

func (f *fakeRecordStore) ListRideEntriesForFormMonth(context.Context, uuid.UUID, time.Time, time.Time) ([]MonthRideEntry, error) {
	return f.monthRideEntries, f.monthRideEntriesErr
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

func (f *fakeRecordStore) CorrectRideRecord(context.Context, uuid.UUID, PatchValue[string], PatchValue[uuid.UUID], PatchValue[uuid.UUID], PatchValue[string], PatchValue[int16], PatchValue[bool], PatchValue[string], uuid.UUID) error {
	return nil
}

func (f *fakeRecordStore) GetRideRecordByID(context.Context, uuid.UUID) (*RideRecord, error) {
	return f.getByIDResult, f.getByIDErr
}

func (f *fakeRecordStore) ResolveConflict(context.Context, uuid.UUID, uuid.UUID, *uuid.UUID, *string, uuid.UUID) (bool, error) {
	f.conflictResolved = true
	if f.resolveErr != nil {
		return false, f.resolveErr
	}
	return f.resolveResult, nil
}

func (f *fakeRecordStore) ListPendingConflicts(context.Context, time.Time, time.Time, string, int, int) ([]ConflictRide, int64, error) {
	return f.pendingConflicts, int64(len(f.pendingConflicts)), nil
}

func (f *fakeRecordStore) ListImportErrorSubmissions(context.Context, time.Time, time.Time, string, int, int) ([]ImportErrorSubmission, int64, error) {
	return f.importErrors, int64(len(f.importErrors)), nil
}

// UpsertRideSourceRowConflict 重現 migration 000035 的 partial unique index：同一 slot
// （vehicle+case+date+leg）同時只有一筆未解決的衝突，第二次呼叫直接更新其新值。
func (f *fakeRecordStore) UpsertRideSourceRowConflict(_ context.Context, in RowConflictInput) (uuid.UUID, error) {
	key := rowConflictSlotKey{in.VehicleID, in.CaseID, in.ServiceDate.Format("2006-01-02"), in.LegSeq}
	if id, ok := f.openRowConflicts[key]; ok {
		c := f.rowConflicts[id]
		c.input.NewSubmissionID = in.NewSubmissionID
		c.input.NewReported = in.NewReported
		c.input.NewDriverID = in.NewDriverID
		c.input.NewSubmittedAt = in.NewSubmittedAt
		return id, nil
	}
	id := uuid.New()
	f.rowConflicts[id] = &fakeRowConflict{id: id, input: in}
	f.openRowConflicts[key] = id
	return id, nil
}

func (f *fakeRecordStore) ListPendingRowConflicts(context.Context) ([]RowConflict, error) {
	var out []RowConflict
	for _, c := range f.rowConflicts {
		if c.resolvedAt != nil {
			continue
		}
		in := c.input
		out = append(out, RowConflict{
			ID: c.id, FormID: in.FormID, VehicleID: in.VehicleID, CaseID: in.CaseID,
			ServiceDate: in.ServiceDate, LegSeq: in.LegSeq, SourceColumnIndex: in.SourceColumnIndex,
			PreviousSubmissionID: in.PreviousSubmissionID, PreviousReported: in.PreviousReported,
			PreviousDriverID: in.PreviousDriverID, PreviousSubmittedAt: in.PreviousSubmittedAt,
			NewSubmissionID: in.NewSubmissionID, NewReported: in.NewReported,
			NewDriverID: in.NewDriverID, NewSubmittedAt: in.NewSubmittedAt,
		})
	}
	return out, nil
}

func (f *fakeRecordStore) ResolveRowConflict(_ context.Context, conflictID uuid.UUID, useNew bool, _ uuid.UUID) (*AppliedRowConflict, bool, error) {
	c, ok := f.rowConflicts[conflictID]
	if !ok || c.resolvedAt != nil {
		return nil, false, nil
	}
	now := time.Now().UTC()
	c.resolvedAt = &now
	if useNew {
		c.resolution = "used_new"
	} else {
		c.resolution = "kept_previous"
	}
	key := rowConflictSlotKey{c.input.VehicleID, c.input.CaseID, c.input.ServiceDate.Format("2006-01-02"), c.input.LegSeq}
	delete(f.openRowConflicts, key)
	if !useNew {
		return nil, true, nil
	}
	in := c.input
	return &AppliedRowConflict{
		CaseID: in.CaseID, ServiceDate: in.ServiceDate, LegSeq: in.LegSeq,
		VehicleID: in.VehicleID, SourceColumnIndex: in.SourceColumnIndex,
		NewSubmissionID: in.NewSubmissionID, NewReported: in.NewReported,
		NewDriverID: in.NewDriverID, NewSubmittedAt: in.NewSubmittedAt,
	}, true, nil
}

func (f *fakeRecordStore) DeleteRowConflict(_ context.Context, conflictID uuid.UUID) (int64, error) {
	c, ok := f.rowConflicts[conflictID]
	if !ok || c.resolvedAt != nil {
		return 0, nil
	}
	key := rowConflictSlotKey{c.input.VehicleID, c.input.CaseID, c.input.ServiceDate.Format("2006-01-02"), c.input.LegSeq}
	delete(f.openRowConflicts, key)
	delete(f.rowConflicts, conflictID)
	return 1, nil
}

func (f *fakeRecordStore) ListRideSourceSlotsForSubmission(_ context.Context, submissionID uuid.UUID) ([]RideSourceSlot, error) {
	var out []RideSourceSlot
	for key, srcs := range f.sources {
		for _, src := range srcs {
			if src.submissionID != submissionID {
				continue
			}
			date, err := time.Parse("2006-01-02", key.date)
			if err != nil {
				return nil, err
			}
			out = append(out, RideSourceSlot{
				CaseID: key.caseID, ServiceDate: date, LegSeq: key.legSeq, VehicleID: src.row.VehicleID,
			})
		}
	}
	return out, nil
}

func (f *fakeRecordStore) DeleteSubmission(_ context.Context, submissionID uuid.UUID) (int64, error) {
	if _, ok := f.submissions[submissionID]; !ok {
		return 0, nil
	}
	delete(f.submissions, submissionID)
	// 對齊資料庫的 ON DELETE CASCADE：提交紀錄消失時，其展開出的搭乘來源一併移除。
	for key, srcs := range f.sources {
		var kept []fakeSource
		for _, src := range srcs {
			if src.submissionID != submissionID {
				kept = append(kept, src)
			}
		}
		if len(kept) == 0 {
			delete(f.sources, key)
			continue
		}
		f.sources[key] = kept
	}
	return 1, nil
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
	driverID := uuid.New()
	serviceDate := time.Date(2026, 3, 2, 0, 0, 0, 0, time.UTC)

	store := newFakeRecordStore([]FormColumn{
		mappedColumn(caseID, "1.吳桂 [去程]", 1, 3),
		mappedColumn(caseID, "1.吳桂 [回程]", 2, 4),
	})
	svc := NewRideService(store, fakeDriverResolver{}, fakeScheduleReader{}, nil, nil)

	result, err := svc.IngestSubmission(context.Background(), uuid.New(), vehicleID, ProcessSubmissionRequest{
		ServiceDate: serviceDate,
		DriverID:    &driverID,
		Answers: map[string]string{
			"1.吳桂 [去程]": "有坐",
			"1.吳桂 [回程]": "沒坐",
		},
	})
	require.NoError(t, err)
	assert.Equal(t, 2, result.Written)
	assert.Zero(t, result.Reaffirmed)
	assert.Zero(t, result.Staged)
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
	driverID := uuid.New()
	store := newFakeRecordStore([]FormColumn{
		mappedColumn(caseID, "1.吳桂 [去程]", 1, 3),
		{ID: uuid.New(), ColumnIndex: 4, ColumnHeader: "2.李四 [去程]", MappingStatus: "pending"},
	})
	svc := NewRideService(store, fakeDriverResolver{}, fakeScheduleReader{}, nil, nil)

	result, err := svc.IngestSubmission(context.Background(), uuid.New(), uuid.New(), ProcessSubmissionRequest{
		ServiceDate: time.Date(2026, 3, 2, 0, 0, 0, 0, time.UTC),
		DriverID:    &driverID,
		Answers: map[string]string{
			"1.吳桂 [去程]": "",
			"2.李四 [去程]": "有坐",
		},
	})
	require.NoError(t, err)
	assert.Zero(t, result.Written, "空白值不建立來源紀錄，未對應欄位不處理")
	assert.Empty(t, store.records)
}

func TestIngestSubmission_ExpandsFourTripPattern(t *testing.T) {
	caseID := uuid.New()
	driverID := uuid.New()
	store := newFakeRecordStore([]FormColumn{mappedColumn(caseID, "1.吳桂 [去程]", 1, 3)})
	svc := NewRideService(store, fakeDriverResolver{}, fakeScheduleReader{tripPattern: 4}, nil, nil)

	result, err := svc.IngestSubmission(context.Background(), uuid.New(), uuid.New(), ProcessSubmissionRequest{
		ServiceDate: time.Date(2026, 3, 2, 0, 0, 0, 0, time.UTC),
		DriverID:    &driverID,
		Answers:     map[string]string{"1.吳桂 [去程]": "有坐"},
	})
	require.NoError(t, err)
	assert.Equal(t, 2, result.Written, "四趟制的表單第 1 趟展開為第 1、3 趟")
	assert.NotNil(t, store.records[slotKey{caseID, "2026-03-02", 1}])
	assert.NotNil(t, store.records[slotKey{caseID, "2026-03-02", 3}])
}

func TestIngestSubmission_RequiresServiceDate(t *testing.T) {
	svc := NewRideService(newFakeRecordStore(nil), fakeDriverResolver{}, fakeScheduleReader{}, nil, nil)

	_, err := svc.IngestSubmission(context.Background(), uuid.New(), uuid.New(), ProcessSubmissionRequest{})
	assert.Error(t, err)
}

func TestIngestSubmission_UnresolvedDriverDoesNotWriteRideSource(t *testing.T) {
	caseID := uuid.New()
	store := newFakeRecordStore([]FormColumn{mappedColumn(caseID, "1.吳桂 [去程]", 1, 3)})
	svc := NewRideService(store, fakeDriverResolver{}, fakeScheduleReader{}, nil, nil)

	result, err := svc.IngestSubmission(context.Background(), uuid.New(), uuid.New(), ProcessSubmissionRequest{
		ServiceDate: time.Date(2026, 3, 2, 0, 0, 0, 0, time.UTC),
		DriverRaw:   "查無此人",
		Answers:     map[string]string{"1.吳桂 [去程]": "有坐"},
	})
	require.NoError(t, err)
	assert.Zero(t, result.Written, "駕駛人比對不到司機主檔時，這一列完全不展開成搭乘來源")
	assert.Empty(t, store.records, "資料不完整時不得出現在司機日曆等其他頁面")
	assert.Empty(t, store.sources)
}

func TestIngestSubmission_SecondUploadSameValueIsReaffirmedNotStaged(t *testing.T) {
	caseID := uuid.New()
	vehicleID := uuid.New()
	driverID := uuid.New()
	formID := uuid.New()
	serviceDate := time.Date(2026, 3, 2, 0, 0, 0, 0, time.UTC)
	store := newFakeRecordStore([]FormColumn{mappedColumn(caseID, "1.吳桂 [去程]", 1, 3)})
	svc := NewRideService(store, fakeDriverResolver{}, fakeScheduleReader{}, nil, nil)

	req := ProcessSubmissionRequest{ServiceDate: serviceDate, DriverID: &driverID, Answers: map[string]string{"1.吳桂 [去程]": "有坐"}}
	first, err := svc.IngestSubmission(context.Background(), formID, vehicleID, req)
	require.NoError(t, err)
	assert.Equal(t, 1, first.Written)

	second, err := svc.IngestSubmission(context.Background(), formID, vehicleID, req)
	require.NoError(t, err)
	assert.Zero(t, second.Written, "重複回報不應再新增一筆來源")
	assert.Equal(t, 1, second.Reaffirmed)
	assert.Zero(t, second.Staged)
	assert.Len(t, store.sources[slotKey{caseID, "2026-03-02", 1}], 1, "無變化的重複回報不應疊加出第二筆來源")
	assert.Empty(t, store.rowConflicts, "值沒有變化不應進待維護")
}

func TestIngestSubmission_SecondUploadDifferentValueStagesConflictWithoutOverwriting(t *testing.T) {
	caseID := uuid.New()
	otherCaseID := uuid.New()
	vehicleID := uuid.New()
	driverID := uuid.New()
	formID := uuid.New()
	serviceDate := time.Date(2026, 3, 2, 0, 0, 0, 0, time.UTC)
	store := newFakeRecordStore([]FormColumn{
		mappedColumn(caseID, "1.吳桂 [去程]", 1, 3),
		mappedColumn(otherCaseID, "2.李四 [去程]", 1, 4),
	})
	svc := NewRideService(store, fakeDriverResolver{}, fakeScheduleReader{}, nil, nil)

	first, err := svc.IngestSubmission(context.Background(), formID, vehicleID, ProcessSubmissionRequest{
		ServiceDate: serviceDate, DriverID: &driverID,
		Answers: map[string]string{"1.吳桂 [去程]": "有坐", "2.李四 [去程]": "有坐"},
	})
	require.NoError(t, err)
	assert.Equal(t, 2, first.Written)

	second, err := svc.IngestSubmission(context.Background(), formID, vehicleID, ProcessSubmissionRequest{
		ServiceDate: serviceDate, DriverID: &driverID,
		Answers: map[string]string{"1.吳桂 [去程]": "沒坐", "2.李四 [去程]": "有坐"},
	})
	require.NoError(t, err)
	assert.Zero(t, second.Written)
	assert.Equal(t, 1, second.Reaffirmed, "沒被改動的個案（李四）仍是重複回報")
	assert.Equal(t, 1, second.Staged, "有坐/沒坐不同的個案（吳桂）要進待維護，不能直接覆蓋")

	// 既有來源必須原封不動：新值只暫存在衝突表，不寫入 ride_sources
	sources := store.sources[slotKey{caseID, "2026-03-02", 1}]
	require.Len(t, sources, 1)
	assert.Equal(t, "boarded", sources[0].row.Reported, "衝突解決前既有來源不得被覆蓋")

	conflicts, err := store.ListPendingRowConflicts(context.Background())
	require.NoError(t, err)
	require.Len(t, conflicts, 1)
	assert.Equal(t, caseID, conflicts[0].CaseID)
	assert.Equal(t, "boarded", conflicts[0].PreviousReported)
	assert.Equal(t, "absent", conflicts[0].NewReported)

	// 未被本次上傳觸及的個案（李四）沒有任何變化
	assert.NotNil(t, store.records[slotKey{otherCaseID, "2026-03-02", 1}])
}

func TestIngestSubmission_ThirdUploadUpdatesOpenConflictButKeepsPreviousValue(t *testing.T) {
	caseID := uuid.New()
	vehicleID := uuid.New()
	driverA := uuid.New()
	driverB := uuid.New()
	formID := uuid.New()
	serviceDate := time.Date(2026, 3, 2, 0, 0, 0, 0, time.UTC)
	t0 := time.Date(2026, 3, 2, 8, 0, 0, 0, time.UTC)
	store := newFakeRecordStore([]FormColumn{mappedColumn(caseID, "1.吳桂 [去程]", 1, 3)})
	svc := NewRideService(store, fakeDriverResolver{}, fakeScheduleReader{}, nil, nil)

	// 既有來源固定是「有坐／driverA」，第二、三次上傳都回報「沒坐」、只是司機不同，
	// 兩次都與既有來源不同、持續待維護；重點是第三次要更新同一筆未解決衝突的新值，
	// 不是又新增一筆。
	first := ProcessSubmissionRequest{ServiceDate: serviceDate, SubmittedAt: t0, DriverID: &driverA, Answers: map[string]string{"1.吳桂 [去程]": "有坐"}}
	_, err := svc.IngestSubmission(context.Background(), formID, vehicleID, first)
	require.NoError(t, err)

	second := ProcessSubmissionRequest{ServiceDate: serviceDate, SubmittedAt: t0.Add(time.Hour), DriverID: &driverA, Answers: map[string]string{"1.吳桂 [去程]": "沒坐"}}
	_, err = svc.IngestSubmission(context.Background(), formID, vehicleID, second)
	require.NoError(t, err)

	third := ProcessSubmissionRequest{ServiceDate: serviceDate, SubmittedAt: t0.Add(2 * time.Hour), DriverID: &driverB, Answers: map[string]string{"1.吳桂 [去程]": "沒坐"}}
	result, err := svc.IngestSubmission(context.Background(), formID, vehicleID, third)
	require.NoError(t, err)
	assert.Equal(t, 1, result.Staged, "第三次上傳仍與既有資料不同，繼續待維護")

	conflicts, err := store.ListPendingRowConflicts(context.Background())
	require.NoError(t, err)
	require.Len(t, conflicts, 1, "同一 slot 同時只保留一筆未解決的衝突，不疊加")
	assert.Equal(t, "boarded", conflicts[0].PreviousReported, "既有值仍是第一次上傳的資料，不受後續衝突影響")
	require.NotNil(t, conflicts[0].PreviousDriverID)
	assert.Equal(t, driverA, *conflicts[0].PreviousDriverID)
	require.NotNil(t, conflicts[0].NewDriverID)
	assert.Equal(t, driverB, *conflicts[0].NewDriverID, "衝突的新值更新為最新一次上傳，不是第二次的司機")
}

func TestResolveRowConflict_UseNewAppliesValueAndRecalculates(t *testing.T) {
	caseID := uuid.New()
	vehicleID := uuid.New()
	driverID := uuid.New()
	formID := uuid.New()
	operatorID := uuid.New()
	serviceDate := time.Date(2026, 3, 2, 0, 0, 0, 0, time.UTC)
	t0 := time.Date(2026, 3, 2, 8, 0, 0, 0, time.UTC)
	store := newFakeRecordStore([]FormColumn{mappedColumn(caseID, "1.吳桂 [去程]", 1, 3)})
	svc := NewRideService(store, fakeDriverResolver{}, fakeScheduleReader{}, nil, nil)

	first := ProcessSubmissionRequest{ServiceDate: serviceDate, SubmittedAt: t0, DriverID: &driverID, Answers: map[string]string{"1.吳桂 [去程]": "有坐"}}
	_, err := svc.IngestSubmission(context.Background(), formID, vehicleID, first)
	require.NoError(t, err)
	second := ProcessSubmissionRequest{ServiceDate: serviceDate, SubmittedAt: t0.Add(time.Hour), DriverID: &driverID, Answers: map[string]string{"1.吳桂 [去程]": "沒坐"}}
	_, err = svc.IngestSubmission(context.Background(), formID, vehicleID, second)
	require.NoError(t, err)

	conflicts, err := store.ListPendingRowConflicts(context.Background())
	require.NoError(t, err)
	require.Len(t, conflicts, 1)

	appliedDriverID, appliedDate, err := svc.ResolveRowConflict(context.Background(), conflicts[0].ID, true, operatorID)
	require.NoError(t, err)
	require.NotNil(t, appliedDriverID)
	assert.Equal(t, driverID, *appliedDriverID)
	require.NotNil(t, appliedDate)

	sources := store.sources[slotKey{caseID, "2026-03-02", 1}]
	require.Len(t, sources, 2, "採用新資料會重放寫入一筆新的來源，既有那筆仍保留")
	assert.Equal(t, "absent", store.records[slotKey{caseID, "2026-03-02", 1}].EffectiveStatus,
		"重算後要反映新值：新值的上傳時間較晚，混車合併同車取最新來源的規則會選中它")

	remaining, err := store.ListPendingRowConflicts(context.Background())
	require.NoError(t, err)
	assert.Empty(t, remaining, "裁決後不再出現在待維護清單")
}

func TestResolveRowConflict_KeepPreviousLeavesExistingRecordUnchanged(t *testing.T) {
	caseID := uuid.New()
	vehicleID := uuid.New()
	driverID := uuid.New()
	formID := uuid.New()
	operatorID := uuid.New()
	serviceDate := time.Date(2026, 3, 2, 0, 0, 0, 0, time.UTC)
	t0 := time.Date(2026, 3, 2, 8, 0, 0, 0, time.UTC)
	store := newFakeRecordStore([]FormColumn{mappedColumn(caseID, "1.吳桂 [去程]", 1, 3)})
	svc := NewRideService(store, fakeDriverResolver{}, fakeScheduleReader{}, nil, nil)

	first := ProcessSubmissionRequest{ServiceDate: serviceDate, SubmittedAt: t0, DriverID: &driverID, Answers: map[string]string{"1.吳桂 [去程]": "有坐"}}
	_, err := svc.IngestSubmission(context.Background(), formID, vehicleID, first)
	require.NoError(t, err)
	second := ProcessSubmissionRequest{ServiceDate: serviceDate, SubmittedAt: t0.Add(time.Hour), DriverID: &driverID, Answers: map[string]string{"1.吳桂 [去程]": "沒坐"}}
	_, err = svc.IngestSubmission(context.Background(), formID, vehicleID, second)
	require.NoError(t, err)

	conflicts, err := store.ListPendingRowConflicts(context.Background())
	require.NoError(t, err)
	require.Len(t, conflicts, 1)

	appliedDriverID, appliedDate, err := svc.ResolveRowConflict(context.Background(), conflicts[0].ID, false, operatorID)
	require.NoError(t, err)
	assert.Nil(t, appliedDriverID, "保留原資料不需要同步出勤")
	assert.Nil(t, appliedDate)

	sources := store.sources[slotKey{caseID, "2026-03-02", 1}]
	require.Len(t, sources, 1, "保留原資料不寫入新來源")
	assert.Equal(t, "boarded", store.records[slotKey{caseID, "2026-03-02", 1}].EffectiveStatus, "既有搭乘紀錄維持不變")
}

func TestResolveRowConflict_AlreadyResolvedReturnsSentinelError(t *testing.T) {
	caseID := uuid.New()
	vehicleID := uuid.New()
	driverID := uuid.New()
	formID := uuid.New()
	operatorID := uuid.New()
	serviceDate := time.Date(2026, 3, 2, 0, 0, 0, 0, time.UTC)
	t0 := time.Date(2026, 3, 2, 8, 0, 0, 0, time.UTC)
	store := newFakeRecordStore([]FormColumn{mappedColumn(caseID, "1.吳桂 [去程]", 1, 3)})
	svc := NewRideService(store, fakeDriverResolver{}, fakeScheduleReader{}, nil, nil)

	first := ProcessSubmissionRequest{ServiceDate: serviceDate, SubmittedAt: t0, DriverID: &driverID, Answers: map[string]string{"1.吳桂 [去程]": "有坐"}}
	_, err := svc.IngestSubmission(context.Background(), formID, vehicleID, first)
	require.NoError(t, err)
	second := ProcessSubmissionRequest{ServiceDate: serviceDate, SubmittedAt: t0.Add(time.Hour), DriverID: &driverID, Answers: map[string]string{"1.吳桂 [去程]": "沒坐"}}
	_, err = svc.IngestSubmission(context.Background(), formID, vehicleID, second)
	require.NoError(t, err)

	conflicts, err := store.ListPendingRowConflicts(context.Background())
	require.NoError(t, err)
	require.Len(t, conflicts, 1)

	_, _, err = svc.ResolveRowConflict(context.Background(), conflicts[0].ID, true, operatorID)
	require.NoError(t, err)

	_, _, err = svc.ResolveRowConflict(context.Background(), conflicts[0].ID, true, operatorID)
	require.ErrorIs(t, err, ErrRowConflictAlreadyResolved)
}

func TestBackfillColumn_WritesFromStoredAnswersWithoutOriginalFile(t *testing.T) {
	caseID := uuid.New()
	vehicleID := uuid.New()
	formID := uuid.New()
	driverID := uuid.New()
	store := newFakeRecordStore(nil)
	svc := NewRideService(store, fakeDriverResolver{}, fakeScheduleReader{}, nil, nil)

	// 模擬上一次上傳時這一欄還在待維護，payload 已存但完全沒有寫入搭乘來源。
	_, err := store.SaveFormSubmission(
		context.Background(), formID, time.Date(2026, 3, 2, 0, 0, 0, 0, time.UTC), time.Now().UTC(),
		"林彥衡", &driverID, "import",
		map[string]interface{}{"answers": map[string]string{"1.吳桂 [去程]": "有坐"}}, "", nil,
	)
	require.NoError(t, err)

	written, err := svc.BackfillColumn(context.Background(), formID, vehicleID, "1.吳桂 [去程]", 3, caseID, 1, nil)
	require.NoError(t, err)
	assert.Equal(t, 1, written, "已存的原始回答要能直接補寫，不需要重新上傳檔案")

	rec := store.records[slotKey{caseID, "2026-03-02", 1}]
	require.NotNil(t, rec)
	assert.Equal(t, "boarded", rec.EffectiveStatus)
}

func TestBackfillColumn_SkipsDatesOwnedByTheCaller(t *testing.T) {
	// 匯入路徑會排除本次檔案涵蓋的日期：那些天的權威值是手上這份檔案，
	// payload 可能還是上一次上傳、尚未被這一批覆蓋的舊值
	caseID := uuid.New()
	formID := uuid.New()
	vehicleID := uuid.New()
	driverID := uuid.New()
	serviceDate := time.Date(2026, 3, 2, 0, 0, 0, 0, time.UTC)
	store := newFakeRecordStore(nil)
	svc := NewRideService(store, fakeDriverResolver{}, fakeScheduleReader{}, nil, nil)

	_, err := store.SaveFormSubmission(
		context.Background(), formID, serviceDate, time.Now().UTC(),
		"林彥衡", &driverID, "import",
		map[string]interface{}{"answers": map[string]string{"1.吳桂 [去程]": "有坐"}}, "", nil,
	)
	require.NoError(t, err)

	written, err := svc.BackfillColumn(
		context.Background(), formID, vehicleID, "1.吳桂 [去程]", 3, caseID, 1,
		[]time.Time{serviceDate},
	)
	require.NoError(t, err)
	assert.Zero(t, written, "被排除的日期不得用既有 payload 補寫")
	assert.Empty(t, store.records)
}

func TestBackfillColumn_SkipsSubmissionsWithoutThisColumn(t *testing.T) {
	formID := uuid.New()
	driverID := uuid.New()
	store := newFakeRecordStore(nil)
	svc := NewRideService(store, fakeDriverResolver{}, fakeScheduleReader{}, nil, nil)

	_, err := store.SaveFormSubmission(
		context.Background(), formID, time.Date(2026, 3, 2, 0, 0, 0, 0, time.UTC), time.Now().UTC(),
		"林彥衡", &driverID, "import",
		map[string]interface{}{"answers": map[string]string{"1.吳桂 [去程]": "有坐"}}, "", nil,
	)
	require.NoError(t, err)

	written, err := svc.BackfillColumn(context.Background(), formID, uuid.New(), "2.李四 [去程]", 4, uuid.New(), 1, nil)
	require.NoError(t, err)
	assert.Zero(t, written)
	assert.Empty(t, store.records)
}

func TestBackfillColumn_SkipsAnswersWithUnresolvedDriver(t *testing.T) {
	caseID := uuid.New()
	formID := uuid.New()
	store := newFakeRecordStore(nil)
	svc := NewRideService(store, fakeDriverResolver{}, fakeScheduleReader{}, nil, nil)

	// 司機仍待維護：payload 已存，但這一列還不該展開成搭乘來源。
	_, err := store.SaveFormSubmission(
		context.Background(), formID, time.Date(2026, 3, 2, 0, 0, 0, 0, time.UTC), time.Now().UTC(),
		"查無此人", nil, "import",
		map[string]interface{}{"answers": map[string]string{"1.吳桂 [去程]": "有坐"}}, "", nil,
	)
	require.NoError(t, err)

	written, err := svc.BackfillColumn(context.Background(), formID, uuid.New(), "1.吳桂 [去程]", 3, caseID, 1, nil)
	require.NoError(t, err)
	assert.Zero(t, written, "司機仍待維護時，個案對應完成也不該展開成搭乘來源")
	assert.Empty(t, store.records)
}

func TestBackfillDriver_BackfillsOnlySubmissionsWithMatchingNormalizedName(t *testing.T) {
	caseID := uuid.New()
	vehicleID := uuid.New()
	formID := uuid.New()
	driverID := uuid.New()
	matchingSubmission := uuid.New()
	otherSubmission := uuid.New()
	serviceDate := time.Date(2026, 3, 2, 0, 0, 0, 0, time.UTC)

	store := newFakeRecordStore([]FormColumn{mappedColumn(caseID, "1.吳桂 [去程]", 1, 3)})
	svc := NewRideService(store, fakeDriverResolver{}, fakeScheduleReader{}, nil, nil)

	// 司機比對不到時原本就沒有展開成搭乘來源，回填要從表單已存的原始答案重新比對寫入，
	// 不是更新既有來源。
	store.unmatchedDrivers = []UnmatchedDriverSubmission{
		{
			SubmissionID: matchingSubmission, FormID: formID, VehicleID: vehicleID, ServiceDate: serviceDate,
			DriverNameRaw: "林彥衡", Answers: map[string]string{"1.吳桂 [去程]": "有坐"},
		},
		{
			SubmissionID: otherSubmission, FormID: formID, VehicleID: vehicleID, ServiceDate: serviceDate,
			DriverNameRaw: "陳大明", Answers: map[string]string{"1.吳桂 [去程]": "有坐"},
		},
	}

	affected, dates, err := svc.BackfillDriver(context.Background(), "林彥衡", driverID)

	require.NoError(t, err)
	assert.Equal(t, 1, affected, "只回填正規化姓名相符的那一筆，不影響其他姓名")
	assert.Equal(t, []time.Time{serviceDate}, dates, "回傳涉及的服務日期，供呼叫端同步司機出勤")

	require.Len(t, store.updatedSubmissionDrivers, 1)
	assert.Equal(t, matchingSubmission, store.updatedSubmissionDrivers[0].submissionID)
	assert.Equal(t, driverID, store.updatedSubmissionDrivers[0].driverID)

	rec := store.records[slotKey{caseID, "2026-03-02", 1}]
	require.NotNil(t, rec, "回填後要重算搭乘紀錄，不需要重新上傳檔案")
	assert.Equal(t, "boarded", rec.EffectiveStatus)

	sources := store.sources[slotKey{caseID, "2026-03-02", 1}]
	require.Len(t, sources, 1)
	require.NotNil(t, sources[0].row.DriverID)
	assert.Equal(t, driverID, *sources[0].row.DriverID, "補綁定後寫入的來源要帶司機")
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
