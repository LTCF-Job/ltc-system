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
	markedImported bool
}

func (s *recordingStore) MarkImported(context.Context, uuid.UUID, time.Time) error {
	s.markedImported = true
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
	store := &recordingStore{stubStore: &stubStore{
		existing: []ColumnMapping{mappedColumnMapping()},
		form: &ReportForm{
			ID:                 uuid.MustParse(testFormID),
			VehicleID:          uuid.MustParse("33333333-3333-3333-3333-333333333333"),
			VehicleDisplayName: "竹南2車",
		},
	}}
	svc := NewDriverReportService(
		store,
		stubExcel{table: table},
		nil,
		stubCases{list: []CaseRef{{ID: testCaseID, Name: "吳桂", NameNormalized: "吳桂"}}},
		stubDrivers{known: map[string]DriverRef{"林彥衡": {ID: uuid.New(), Name: "林彥衡"}}},
		ingestor,
		nil,
		directTxRunner{},
	)
	return svc, store
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
	svc, _ := newCommitService(sampleTable(), ingestor)

	_, err := commit(svc, "2026-03")

	require.NoError(t, err)
	require.Len(t, ingestor.clears, 1)
	assert.Len(t, ingestor.clears[0].dates, 31, "三月有 31 天，宣告月份時整個月都要被覆蓋")
	assert.Equal(t, "2026-03-01", ingestor.clears[0].dates[0].Format("2006-01-02"))
	assert.Equal(t, "2026-03-31", ingestor.clears[0].dates[30].Format("2006-01-02"))
}

func TestCommitDriverReport_RepeatedImportProducesSameWrites(t *testing.T) {
	first := &fakeIngestor{}
	svcFirst, _ := newCommitService(sampleTable(), first)
	firstResult, err := commit(svcFirst, "2026-03")
	require.NoError(t, err)

	second := &fakeIngestor{}
	svcSecond, _ := newCommitService(sampleTable(), second)
	secondResult, err := commit(svcSecond, "2026-03")
	require.NoError(t, err)

	assert.Equal(t, firstResult.ImportedRows, secondResult.ImportedRows)
	assert.Equal(t, firstResult.RideRecordRows, secondResult.RideRecordRows)
	assert.Len(t, second.clears, 1, "每次匯入都必須先清除，否則重匯會疊加")
	assert.Len(t, second.submissions, len(first.submissions))
}

func TestCommitDriverReport_SkipsRowsOutsideDeclaredMonth(t *testing.T) {
	// 月份不符不再整份拒絕：讓使用者在預覽畫面看得到比對結果並自行決定，
	// 這裡改成比照「單列日期打錯」的規則，逐列略過而不中斷整個匯入。
	ingestor := &fakeIngestor{}
	svc, store := newCommitService(sampleTable(), ingestor)

	result, err := commit(svc, "2026-04")

	require.NoError(t, err)
	assert.Equal(t, 0, result.ImportedRows)
	// sampleTable 三列有日期的資料：兩列可解析但落在三月（月份不符），一列日期格式本身無效。
	require.Len(t, result.SkippedRows, 3)
	assert.Contains(t, result.SkippedRows[0].Reasons[0], "不屬於宣告匯入的 2026-04")
	assert.Contains(t, result.SkippedRows[1].Reasons[0], "不屬於宣告匯入的 2026-04")
	assert.Contains(t, result.SkippedRows[2].Reasons[0], "日期格式無法解析")
	assert.Empty(t, ingestor.clears, "沒有任何可寫入的列時不得清除既有資料")
	assert.Empty(t, ingestor.submissions)
	assert.False(t, store.markedImported, "整份都被跳過時不算成功匯入，不得更新最後匯入時間")
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

func TestCommitDriverReport_EmptyFileDoesNotClearAnything(t *testing.T) {
	// 只有表頭與一列壞掉的日期：沒有任何可寫入的列，代表多半是傳錯檔案
	table := [][]string{
		{"民國日期", "駕駛人", "1.吳桂(去程竹3) [去程]", "備註"},
		{"壞掉的日期", "林彥衡", "有坐", ""},
	}
	ingestor := &fakeIngestor{}
	svc, _ := newCommitService(table, ingestor)

	result, err := commit(svc, "2026-03")

	require.NoError(t, err)
	assert.Zero(t, result.ImportedRows)
	assert.Len(t, result.SkippedRows, 1)
	assert.Empty(t, ingestor.clears, "沒有有效列時不得清空整個月")
}

func TestCommitDriverReport_RequiresTransactionRunner(t *testing.T) {
	svc := NewDriverReportService(&stubStore{}, stubExcel{}, nil, nil, nil, &fakeIngestor{}, nil, nil)

	_, err := commit(svc, "")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "transaction runner not configured")
}
