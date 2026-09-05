package app

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// recordingStore 在 stubStore 之上記錄寫入行為，讓測試能斷言交易失敗時未留下痕跡。
type recordingStore struct {
	*stubStore
	markedImported  bool
	upsertedColumns []ColumnDraft
}

func (s *recordingStore) MarkImported(context.Context, uuid.UUID, time.Time) error {
	s.markedImported = true
	return nil
}

func (s *recordingStore) UpsertColumns(_ context.Context, _ uuid.UUID, drafts []ColumnDraft) error {
	s.upsertedColumns = append([]ColumnDraft(nil), drafts...)
	return nil
}

// clearCall 保留一次覆蓋清除的參數，供斷言清除範圍是否正確。
type clearCall struct {
	formID uuid.UUID
	dates  []time.Time
}

// fakeIngestor 依序記錄清除與寫入的呼叫，讓測試能斷言「先清後寫」的實際順序。
type fakeIngestor struct {
	events         []string
	clears         []clearCall
	submissions    []Submission
	ingestErr      error
	importedMonths []ImportedMonth
	importedErr    error
	backfillCalls  []backfillCall
	backfillResult int
	backfillErr    error

	submissionsForForms  []SubmissionAnswerRow
	unmatchedDrivers     []UnmatchedDriverSubmission
	backfillDriverCalls  []backfillDriverCall
	backfillDriverResult int
	backfillDriverDates  []time.Time
	backfillDriverErr    error

	monthSubmissions []MonthSubmissionDetail
	monthRideEntries []MonthRideEntry
	monthDetailErr   error
}

// backfillDriverCall 保留一次司機回填的參數，供測試斷言傳入值。
type backfillDriverCall struct {
	driverNameRaw string
	driverID      uuid.UUID
}

// backfillCall 保留一次欄位回填的參數，供測試斷言觸發條件與傳入值。
type backfillCall struct {
	formID       uuid.UUID
	vehicleID    uuid.UUID
	columnHeader string
	columnIndex  int
	caseID       uuid.UUID
	legSeq       int16
}

func (f *fakeIngestor) ListImportedMonths(context.Context) ([]ImportedMonth, error) {
	return f.importedMonths, f.importedErr
}

func (f *fakeIngestor) ClearImportedDates(_ context.Context, formID uuid.UUID, dates []time.Time) (int, error) {
	f.events = append(f.events, "clear")
	f.clears = append(f.clears, clearCall{formID: formID, dates: dates})
	return len(dates), nil
}

func (f *fakeIngestor) IngestSubmission(_ context.Context, _, _ uuid.UUID, s Submission) (int, error) {
	if f.ingestErr != nil {
		return 0, f.ingestErr
	}
	f.events = append(f.events, "ingest")
	f.submissions = append(f.submissions, s)
	return 1, nil
}

func (f *fakeIngestor) BackfillColumn(_ context.Context, formID, vehicleID uuid.UUID, columnHeader string, columnIndex int, caseID uuid.UUID, legSeq int16) (int, error) {
	f.backfillCalls = append(f.backfillCalls, backfillCall{
		formID: formID, vehicleID: vehicleID, columnHeader: columnHeader, columnIndex: columnIndex, caseID: caseID, legSeq: legSeq,
	})
	if f.backfillErr != nil {
		return 0, f.backfillErr
	}
	return f.backfillResult, nil
}

func (f *fakeIngestor) ListSubmissionsForForms(context.Context, []uuid.UUID) ([]SubmissionAnswerRow, error) {
	return f.submissionsForForms, nil
}

func (f *fakeIngestor) ListUnmatchedDriverSubmissions(context.Context) ([]UnmatchedDriverSubmission, error) {
	return f.unmatchedDrivers, nil
}

func (f *fakeIngestor) BackfillDriver(_ context.Context, driverNameRaw string, driverID uuid.UUID) (int, []time.Time, error) {
	f.backfillDriverCalls = append(f.backfillDriverCalls, backfillDriverCall{driverNameRaw: driverNameRaw, driverID: driverID})
	if f.backfillDriverErr != nil {
		return 0, nil, f.backfillDriverErr
	}
	return f.backfillDriverResult, f.backfillDriverDates, nil
}

func (f *fakeIngestor) ListSubmissionsForFormMonth(context.Context, uuid.UUID, time.Time, time.Time) ([]MonthSubmissionDetail, error) {
	return f.monthSubmissions, f.monthDetailErr
}

func (f *fakeIngestor) ListRideEntriesForFormMonth(context.Context, uuid.UUID, time.Time, time.Time) ([]MonthRideEntry, error) {
	return f.monthRideEntries, f.monthDetailErr
}

// attendanceSyncCall 保留一次出勤同步呼叫的參數，供測試斷言傳入值。
type attendanceSyncCall struct {
	driverID    uuid.UUID
	serviceDate time.Time
}

// fakeAttendanceRegistrar 記錄每次出勤同步呼叫，讓測試能斷言比對到司機的列有觸發同步。
type fakeAttendanceRegistrar struct {
	calls []attendanceSyncCall
	err   error
}

func (f *fakeAttendanceRegistrar) SyncFromImport(_ context.Context, driverID uuid.UUID, serviceDate time.Time) error {
	if f.err != nil {
		return f.err
	}
	f.calls = append(f.calls, attendanceSyncCall{driverID: driverID, serviceDate: serviceDate})
	return nil
}

// directTxRunner 直接執行 fn，讓 commit 測試不依賴真實資料庫。
type directTxRunner struct{}

func (directTxRunner) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	return fn(ctx)
}

func mappedColumnMapping() ColumnMapping {
	caseID := testCaseID
	legSeq := int16(1)
	return ColumnMapping{
		ID:            uuid.New().String(),
		ColumnIndex:   3,
		ColumnHeader:  "1.吳桂(去程竹3) [去程]",
		MappingStatus: "mapped",
		CaseID:        &caseID,
		LegSeq:        &legSeq,
	}
}

func newCommitService(table [][]string, ingestor *fakeIngestor) (*DriverReportService, *recordingStore) {
	svc, store, _ := newCommitServiceWithAttendance(table, ingestor)
	return svc, store
}

func newCommitServiceWithAttendance(table [][]string, ingestor *fakeIngestor) (*DriverReportService, *recordingStore, *fakeAttendanceRegistrar) {
	store := &recordingStore{stubStore: &stubStore{
		existing: []ColumnMapping{mappedColumnMapping()},
		form: &ReportForm{
			ID:                 uuid.MustParse(testFormID),
			VehicleID:          uuid.MustParse("33333333-3333-3333-3333-333333333333"),
			VehicleDisplayName: "竹南2車",
		},
	}}
	attendance := &fakeAttendanceRegistrar{}
	svc := NewDriverReportService(
		store,
		stubExcel{table: table},
		nil,
		stubCases{list: []CaseRef{{ID: testCaseID, Name: "吳桂", NameNormalized: "吳桂"}}},
		stubDrivers{known: map[string]DriverRef{"林彥衡": {ID: uuid.New(), Name: "林彥衡"}}},
		ingestor,
		attendance,
		nil,
		directTxRunner{},
	)
	return svc, store, attendance
}

func commit(svc *DriverReportService, yearMonth string) (*CommitResult, error) {
	return svc.CommitDriverReport(
		context.Background(),
		uuid.MustParse(testFormID),
		strings.NewReader("x"),
		nil,
		yearMonth,
		Actor{},
	)
}

func validSampleTable() [][]string {
	table := sampleTable()
	return append([][]string(nil), table[:len(table)-1]...)
}

func TestCommitDriverReport_ClearsCoveredDatesBeforeWriting(t *testing.T) {
	ingestor := &fakeIngestor{}
	svc, _ := newCommitService(sampleTable(), ingestor)

	result, err := commit(svc, "")

	require.NoError(t, err)
	assert.Equal(t, 2, result.ImportedRows)
	require.Len(t, ingestor.clears, 1)
	assert.Equal(t, uuid.MustParse(testFormID), ingestor.clears[0].formID)
	assert.Equal(t, []string{"clear", "ingest", "ingest"}, ingestor.events,
		"清除必須發生在任何寫入之前，否則新資料會連同舊資料一起被刪掉")

	var cleared []string
	for _, d := range ingestor.clears[0].dates {
		cleared = append(cleared, d.Format("2006-01-02"))
	}
	assert.ElementsMatch(t, []string{"2026-03-02", "2026-03-03"}, cleared,
		"未宣告月份時只清除檔案實際涵蓋的日期")
}

func TestCommitDriverReport_DeclaredMonthClearsWholeMonth(t *testing.T) {
	ingestor := &fakeIngestor{}
	svc, _ := newCommitService(validSampleTable(), ingestor)

	_, err := commit(svc, "2026-03")

	require.NoError(t, err)
	require.Len(t, ingestor.clears, 1)
	assert.Len(t, ingestor.clears[0].dates, 31, "三月有 31 天，宣告月份時整個月都要被覆蓋")
	assert.Equal(t, "2026-03-01", ingestor.clears[0].dates[0].Format("2006-01-02"))
	assert.Equal(t, "2026-03-31", ingestor.clears[0].dates[30].Format("2006-01-02"))
}

func TestCommitDriverReport_DeclaredMonthBlocksOnMalformedRow(t *testing.T) {
	ingestor := &fakeIngestor{}
	svc, store := newCommitService(sampleTable(), ingestor)

	result, err := commit(svc, "2026-03")

	require.ErrorIs(t, err, ErrImportHasBlockingErrors)
	assert.Nil(t, result)
	assert.Empty(t, ingestor.clears, "阻斷性錯誤時不得先清除整月資料")
	assert.Empty(t, ingestor.submissions, "阻斷性錯誤時不得寫入有效列")
	assert.False(t, store.markedImported, "阻斷性錯誤時不得更新最後匯入時間")
}

func TestCommitDriverReport_RepeatedImportProducesSameWrites(t *testing.T) {
	first := &fakeIngestor{}
	svcFirst, _ := newCommitService(validSampleTable(), first)
	firstResult, err := commit(svcFirst, "2026-03")
	require.NoError(t, err)

	second := &fakeIngestor{}
	svcSecond, _ := newCommitService(validSampleTable(), second)
	secondResult, err := commit(svcSecond, "2026-03")
	require.NoError(t, err)

	assert.Equal(t, firstResult.ImportedRows, secondResult.ImportedRows)
	assert.Equal(t, firstResult.RideRecordRows, secondResult.RideRecordRows)
	assert.Len(t, second.clears, 1, "每次匯入都必須先清除，否則重匯會疊加")
	assert.Len(t, second.submissions, len(first.submissions))
}

func TestCommitDriverReport_SkipsRowsOutsideDeclaredMonth(t *testing.T) {
	// 同一檔案依月份拆分匯入時，其他月份的有效列應只在這一輪略過。
	ingestor := &fakeIngestor{}
	table := [][]string{
		{"民國日期", "駕駛人", "1.吳桂(去程竹3) [去程]", "1.吳桂(去程竹3) [回程]", "備註"},
		{"1150302", "林彥衡", "有坐", "沒坐", "無"},
		{"1150303", "林彥衡", "有坐", "有坐", ""},
		{"1150402", "林彥衡", "有坐", "有坐", ""},
	}
	svc, store := newCommitService(table, ingestor)

	result, err := commit(svc, "2026-04")

	require.NoError(t, err)
	assert.Equal(t, 1, result.ImportedRows)
	// 兩列落在三月，這一輪只略過；四月的有效列仍可完整覆蓋四月。
	require.Len(t, result.SkippedRows, 2)
	assert.Contains(t, result.SkippedRows[0].Reasons[0], "不屬於本次宣告匯入的 2026-04")
	assert.Contains(t, result.SkippedRows[1].Reasons[0], "不屬於本次宣告匯入的 2026-04")
	require.Len(t, ingestor.clears, 1)
	assert.Len(t, ingestor.clears[0].dates, 30)
	assert.Len(t, ingestor.submissions, 1)
	assert.True(t, store.markedImported)
}

func TestCommitDriverReport_RejectsMalformedYearMonth(t *testing.T) {
	ingestor := &fakeIngestor{}
	svc, _ := newCommitService(sampleTable(), ingestor)

	_, err := commit(svc, "2026/03")

	require.ErrorIs(t, err, ErrInvalidYearMonth)
	assert.Empty(t, ingestor.clears)
}

func TestCommitDriverReport_IngestFailureAbortsWholeImport(t *testing.T) {
	ingestor := &fakeIngestor{ingestErr: errors.New("boom")}
	svc, store := newCommitService(sampleTable(), ingestor)

	result, err := commit(svc, "2026-03")

	require.Error(t, err)
	assert.Nil(t, result)
	assert.False(t, store.markedImported, "交易中止時不得更新最後匯入時間")
}

func TestCommitDriverReport_BlockingErrorDoesNotClearAnything(t *testing.T) {
	// 只有表頭與一列壞掉的日期：沒有任何可寫入的列，代表多半是傳錯檔案
	table := [][]string{
		{"民國日期", "駕駛人", "1.吳桂(去程竹3) [去程]", "備註"},
		{"壞掉的日期", "林彥衡", "有坐", ""},
	}
	ingestor := &fakeIngestor{}
	svc, store := newCommitService(table, ingestor)

	result, err := commit(svc, "2026-03")

	require.ErrorIs(t, err, ErrImportHasBlockingErrors)
	assert.Nil(t, result)
	assert.Empty(t, ingestor.clears, "阻斷性錯誤時不得清空整個月")
	assert.False(t, store.markedImported)
}

func TestCommitDriverReport_PersistsPendingColumnAnswersForLaterBackfill(t *testing.T) {
	table := [][]string{
		{"民國日期", "駕駛人", "1.未知個案 [去程]", "備註"},
		{"1150302", "林彥衡", "有坐", ""},
	}
	ingestor := &fakeIngestor{}
	svc, store := newCommitService(table, ingestor)

	result, err := commit(svc, "2026-03")

	require.NoError(t, err)
	assert.Len(t, store.upsertedColumns, 1, "無法對應的欄位仍須保存，供待維護流程處理")
	assert.Equal(t, 1, result.ImportedRows, "原始資料仍要保存，供之後補綁定回填搭乘紀錄")
	require.Len(t, ingestor.submissions, 1, "尚未對應個案的欄位也要保留原始回答，供之後補綁定回填")
	assert.Equal(t, "有坐", ingestor.submissions[0].Answers["1.未知個案 [去程]"],
		"待維護欄位的原始儲存格文字要進 payload，回填時才有資料可用；是否真的略過搭乘來源寫入由 ride 模組的 mapped 篩選負責，見 ride/app 的 TestIngestSubmission_SkipsUnmappedAndNonReportValues")
	assert.True(t, store.markedImported, "已保存這一列的原始回報，視為完成本次匯入")
}

func TestCommitDriverReport_SyncsAttendanceForMatchedDriverRows(t *testing.T) {
	ingestor := &fakeIngestor{}
	svc, _, attendance := newCommitServiceWithAttendance(validSampleTable(), ingestor)

	result, err := commit(svc, "2026-03")

	require.NoError(t, err)
	assert.Equal(t, 2, result.ImportedRows)
	require.Len(t, attendance.calls, 2, "兩列都比對到司機「林彥衡」，各自的服務日期都要同步出勤")
	assert.Equal(t, "2026-03-02", attendance.calls[0].serviceDate.Format("2006-01-02"))
	assert.Equal(t, "2026-03-03", attendance.calls[1].serviceDate.Format("2006-01-02"))
	assert.Equal(t, attendance.calls[0].driverID, attendance.calls[1].driverID, "同一個司機姓名應比對到同一個司機編號")
}

func TestCommitDriverReport_SkipsAttendanceSyncForUnmatchedDriver(t *testing.T) {
	table := [][]string{
		{"民國日期", "駕駛人", "1.吳桂(去程竹3) [去程]", "備註"},
		{"1150302", "查無此人", "有坐", ""},
	}
	ingestor := &fakeIngestor{}
	svc, _, attendance := newCommitServiceWithAttendance(table, ingestor)

	result, err := commit(svc, "2026-03")

	require.NoError(t, err)
	assert.Equal(t, 1, result.ImportedRows)
	assert.Empty(t, attendance.calls, "駕駛人比對不到司機主檔時不寫出勤，留給既有的司機待維護流程處理")
}

func TestCommitDriverReport_AttendanceSyncFailureAbortsWholeImport(t *testing.T) {
	ingestor := &fakeIngestor{}
	svc, store, attendance := newCommitServiceWithAttendance(sampleTable(), ingestor)
	attendance.err = errors.New("boom")

	result, err := commit(svc, "2026-03")

	require.Error(t, err)
	assert.Nil(t, result)
	assert.False(t, store.markedImported, "出勤同步失敗時整筆匯入要回滾，不得標記為已完成")
}

func TestCommitDriverReport_RequiresTransactionRunner(t *testing.T) {
	svc := NewDriverReportService(&stubStore{}, stubExcel{}, nil, nil, nil, &fakeIngestor{}, nil, nil, nil)

	_, err := commit(svc, "")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "transaction runner not configured")
}
