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

// fakeIngestor 記錄每次寫入的呼叫，讓測試能斷言傳入值與彙整結果。
type fakeIngestor struct {
	events         []string
	submissions    []Submission
	ingestOutcome  IngestOutcome
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

	rowConflicts         []RowConflictView
	rowConflictsErr      error
	resolveRowConflictID uuid.UUID
	resolveUseNew        bool
	resolveAppliedDriver *uuid.UUID
	resolveAppliedDate   *time.Time
	resolveErr           error

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
	skipDates    []time.Time
}

func (f *fakeIngestor) ListImportedMonths(context.Context) ([]ImportedMonth, error) {
	return f.importedMonths, f.importedErr
}

func (f *fakeIngestor) IngestSubmission(_ context.Context, _, _ uuid.UUID, s Submission) (IngestOutcome, error) {
	if f.ingestErr != nil {
		return IngestOutcome{}, f.ingestErr
	}
	f.events = append(f.events, "ingest")
	f.submissions = append(f.submissions, s)
	if f.ingestOutcome == (IngestOutcome{}) {
		return IngestOutcome{Written: 1}, nil
	}
	return f.ingestOutcome, nil
}

func (f *fakeIngestor) ListRowConflicts(context.Context) ([]RowConflictView, error) {
	return f.rowConflicts, f.rowConflictsErr
}

func (f *fakeIngestor) ResolveRowConflict(_ context.Context, conflictID uuid.UUID, useNew bool, operatorID uuid.UUID) (*uuid.UUID, *time.Time, error) {
	f.resolveRowConflictID = conflictID
	f.resolveUseNew = useNew
	if f.resolveErr != nil {
		return nil, nil, f.resolveErr
	}
	return f.resolveAppliedDriver, f.resolveAppliedDate, nil
}

func (f *fakeIngestor) BackfillColumn(_ context.Context, formID, vehicleID uuid.UUID, columnHeader string, columnIndex int, caseID uuid.UUID, legSeq int16, skipDates []time.Time) (int, error) {
	f.events = append(f.events, "backfill")
	f.backfillCalls = append(f.backfillCalls, backfillCall{
		formID: formID, vehicleID: vehicleID, columnHeader: columnHeader, columnIndex: columnIndex, caseID: caseID, legSeq: legSeq,
		skipDates: append([]time.Time(nil), skipDates...),
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
	return newCommitServiceWithColumns(table, ingestor, []ColumnMapping{mappedColumnMapping()})
}

// newCommitServiceWithColumns 讓測試自訂表單既有的欄位對應狀態，用來區分「本次才從
// 待維護變成已對應」與「原本就已對應」兩種回填觸發條件。
func newCommitServiceWithColumns(table [][]string, ingestor *fakeIngestor, existing []ColumnMapping) (*DriverReportService, *recordingStore, *fakeAttendanceRegistrar) {
	store := &recordingStore{stubStore: &stubStore{
		existing: existing,
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
	return commitWithDecisions(svc, yearMonth, nil)
}

func commitWithDecisions(svc *DriverReportService, yearMonth string, decisions []ColumnDecision) (*CommitResult, error) {
	return svc.CommitDriverReport(
		context.Background(),
		uuid.MustParse(testFormID),
		strings.NewReader("x"),
		decisions,
		yearMonth,
		Actor{},
	)
}

// mapDecisionForSampleColumn 把 sampleTable 的個案欄位標成已對應，模擬前端把系統推薦
// 的欄位自動送出 mapped。
func mapDecisionForSampleColumn() []ColumnDecision {
	caseID := testCaseID
	legSeq := int16(1)
	return []ColumnDecision{{
		ColumnHeader:  "1.吳桂(去程竹3) [去程]",
		MappingStatus: "mapped",
		CaseID:        &caseID,
		LegSeq:        &legSeq,
	}}
}

// pendingColumnMapping 是同一個欄位尚未對應個案時的狀態。
func pendingColumnMapping() ColumnMapping {
	return ColumnMapping{
		ID:            uuid.New().String(),
		ColumnIndex:   3,
		ColumnHeader:  "1.吳桂(去程竹3) [去程]",
		MappingStatus: "pending",
	}
}

func validSampleTable() [][]string {
	table := sampleTable()
	return append([][]string(nil), table[:len(table)-1]...)
}

func TestCommitDriverReport_WritesEachImportableRow(t *testing.T) {
	ingestor := &fakeIngestor{}
	svc, _ := newCommitService(sampleTable(), ingestor)

	result, err := commit(svc, "")

	require.NoError(t, err)
	assert.Equal(t, 2, result.ImportedRows)
	assert.Equal(t, []string{"ingest", "ingest"}, ingestor.events)
}

func TestCommitDriverReport_DeclaredMonthBlocksOnMalformedRow(t *testing.T) {
	ingestor := &fakeIngestor{}
	svc, store := newCommitService(sampleTable(), ingestor)

	result, err := commit(svc, "2026-03")

	require.ErrorIs(t, err, ErrImportHasBlockingErrors)
	assert.Nil(t, result)
	assert.Empty(t, ingestor.submissions, "阻斷性錯誤時不得寫入有效列")
	assert.False(t, store.markedImported, "阻斷性錯誤時不得更新最後匯入時間")
}

func TestCommitDriverReport_AggregatesReaffirmedAndPendingConflictCounts(t *testing.T) {
	ingestor := &fakeIngestor{ingestOutcome: IngestOutcome{Written: 0, Reaffirmed: 1, Staged: 1}}
	svc, _ := newCommitService(sampleTable(), ingestor)

	result, err := commit(svc, "")

	require.NoError(t, err)
	assert.Equal(t, 2, result.ImportedRows, "逐日提交紀錄本身仍算已處理，即使沒有新增任何搭乘來源")
	assert.Zero(t, result.RideRecordRows)
	assert.Equal(t, 2, result.ReaffirmedRows, "彙整每一列的無變化重複回報筆數")
	assert.Equal(t, 2, result.PendingConflictRows, "彙整每一列的待維護衝突筆數")
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
	// 兩列落在三月，這一輪只略過；四月的有效列仍可正常匯入。
	require.Len(t, result.SkippedRows, 2)
	assert.Contains(t, result.SkippedRows[0].Reasons[0], "不屬於本次宣告匯入的 2026-04")
	assert.Contains(t, result.SkippedRows[1].Reasons[0], "不屬於本次宣告匯入的 2026-04")
	assert.Len(t, ingestor.submissions, 1)
	assert.True(t, store.markedImported)
}

func TestCommitDriverReport_RejectsMalformedYearMonth(t *testing.T) {
	ingestor := &fakeIngestor{}
	svc, _ := newCommitService(sampleTable(), ingestor)

	_, err := commit(svc, "2026/03")

	require.ErrorIs(t, err, ErrInvalidYearMonth)
	assert.Empty(t, ingestor.submissions)
}

func TestCommitDriverReport_IngestFailureAbortsWholeImport(t *testing.T) {
	ingestor := &fakeIngestor{ingestErr: errors.New("boom")}
	svc, store := newCommitService(sampleTable(), ingestor)

	result, err := commit(svc, "2026-03")

	require.Error(t, err)
	assert.Nil(t, result)
	assert.False(t, store.markedImported, "交易中止時不得更新最後匯入時間")
}

func TestCommitDriverReport_BlockingErrorWritesNothing(t *testing.T) {
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
	assert.Empty(t, ingestor.submissions, "阻斷性錯誤時不得寫入任何資料")
	assert.False(t, store.markedImported)
}

func TestCommitDriverReport_SameFileTwiceStillReconciles(t *testing.T) {
	// 冪等改由逐列比對保證：重傳同一份檔案要照樣逐列走一次，不再整份短路成「已匯入」
	ingestor := &fakeIngestor{ingestOutcome: IngestOutcome{Reaffirmed: 1}}
	svc, store := newCommitService(validSampleTable(), ingestor)

	first, err := commit(svc, "")
	require.NoError(t, err)
	second, err := commit(svc, "")
	require.NoError(t, err)

	assert.Equal(t, "succeeded", second.Status)
	assert.Equal(t, first.ImportedRows, second.ImportedRows, "重傳同一份檔案仍逐列處理")
	assert.Zero(t, second.RideRecordRows, "值與既有相同時不新增任何搭乘來源")
	assert.Equal(t, second.ImportedRows, second.ReaffirmedRows)
	assert.Len(t, ingestor.events, 2*first.ImportedRows, "兩次上傳各自逐列呼叫，沒有被冪等鍵短路")
	assert.True(t, store.markedImported)
}

func TestCommitDriverReport_NewlyMappedColumnTriggersBackfill(t *testing.T) {
	// 欄位這次才從待維護變成已對應時，先前月份留在 form_submissions 的原始值要一併補寫，
	// 否則待維護項目會消失但資料永遠寫不進去
	ingestor := &fakeIngestor{backfillResult: 4}
	svc, _, _ := newCommitServiceWithColumns(validSampleTable(), ingestor, []ColumnMapping{pendingColumnMapping()})

	result, err := commitWithDecisions(svc, "", mapDecisionForSampleColumn())

	require.NoError(t, err)
	require.Len(t, ingestor.backfillCalls, 1)
	call := ingestor.backfillCalls[0]
	assert.Equal(t, uuid.MustParse(testFormID), call.formID)
	assert.Equal(t, uuid.MustParse("33333333-3333-3333-3333-333333333333"), call.vehicleID)
	assert.Equal(t, "1.吳桂(去程竹3) [去程]", call.columnHeader)
	assert.Equal(t, 3, call.columnIndex)
	assert.Equal(t, uuid.MustParse(testCaseID), call.caseID)
	assert.Equal(t, int16(1), call.legSeq)
	assert.Equal(t, 4, result.BackfilledRows)
}

func TestCommitDriverReport_AlreadyMappedColumnSkipsBackfill(t *testing.T) {
	// 欄位原本就已對應時重複送出同樣的決定不該再回填，避免疊加重複的搭乘來源
	ingestor := &fakeIngestor{backfillResult: 4}
	svc, _, _ := newCommitServiceWithColumns(validSampleTable(), ingestor, []ColumnMapping{mappedColumnMapping()})

	result, err := commitWithDecisions(svc, "", mapDecisionForSampleColumn())

	require.NoError(t, err)
	assert.Empty(t, ingestor.backfillCalls)
	assert.Zero(t, result.BackfilledRows)
}

func TestCommitDriverReport_BackfillSkipsEveryDateInTheFile(t *testing.T) {
	// 跨月檔案是逐月各送一次 commit：先 commit 的那個月若拿舊 payload 補寫尚未輪到的月份，
	// 下一輪就會用本次的新值比出一批系統自己製造的衝突
	ingestor := &fakeIngestor{}
	table := [][]string{
		{"民國日期", "駕駛人", "1.吳桂(去程竹3) [去程]", "備註"},
		{"1150302", "林彥衡", "有坐", ""},
		{"1150402", "林彥衡", "有坐", ""},
	}
	svc, _, _ := newCommitServiceWithColumns(table, ingestor, []ColumnMapping{pendingColumnMapping()})

	_, err := commitWithDecisions(svc, "2026-03", mapDecisionForSampleColumn())

	require.NoError(t, err)
	require.Len(t, ingestor.backfillCalls, 1)
	skipped := make([]string, 0, len(ingestor.backfillCalls[0].skipDates))
	for _, d := range ingestor.backfillCalls[0].skipDates {
		skipped = append(skipped, d.Format("2006-01-02"))
	}
	assert.ElementsMatch(t, []string{"2026-03-02", "2026-04-02"}, skipped,
		"宣告月份以外的日期也要排除，那些月份還沒輪到 commit，payload 仍是上一次上傳的值")
}

func TestCommitDriverReport_BackfillRunsAfterIngest(t *testing.T) {
	// form_submissions 是一車一天一筆原地更新：補寫必須排在逐列寫入之後，否則會讀到
	// 這幾天更新前的舊答案，再被本次新值比出一筆並不存在的衝突
	ingestor := &fakeIngestor{}
	svc, _, _ := newCommitServiceWithColumns(validSampleTable(), ingestor, []ColumnMapping{pendingColumnMapping()})

	result, err := commitWithDecisions(svc, "", mapDecisionForSampleColumn())

	require.NoError(t, err)
	require.NotEmpty(t, ingestor.events)
	assert.Equal(t, "backfill", ingestor.events[len(ingestor.events)-1])
	assert.Equal(t, result.ImportedRows, len(ingestor.events)-1, "補寫只在所有列寫入後跑一次")
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
