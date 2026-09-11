package app

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// failingAtCallMappingStore 讓 UpdateColumnMappingByID 前幾次呼叫成功、之後固定失敗，
// 供測試「一個名稱底下多欄位逐一綁定、中途失敗」時已成功的部分是否被正確保留。
type failingAtCallMappingStore struct {
	*stubStore
	calls      int
	failAtCall int
}

func (s *failingAtCallMappingStore) UpdateColumnMappingByID(context.Context, string, string, *string, *int16) (uuid.UUID, string, int, string, error) {
	s.calls++
	if s.calls >= s.failAtCall {
		return uuid.Nil, "", 0, "", errors.New("boom")
	}
	return uuid.MustParse(testFormID), "1.吳桂 [去程]", 3, "mapped", nil
}

// multiDriverStub 讓測試能模擬同一正規化姓名命中多筆司機主檔的情境；
// stubDrivers 固定只能回傳 0 或 1 筆，不夠用於消歧測試。
type multiDriverStub struct{ list []DriverRef }

func (s multiDriverStub) GetByNameNormalized(context.Context, string) (*DriverRef, error) {
	return nil, nil
}

func (s multiDriverStub) ListByNameNormalized(context.Context, string) ([]DriverRef, error) {
	return s.list, nil
}

func TestAutoBindDriver_UniqueMatchBinds(t *testing.T) {
	driverID := uuid.New()
	drivers := stubDrivers{known: map[string]DriverRef{"陳大華": {ID: driverID, Name: "陳大華"}}}
	ingestor := &fakeIngestor{backfillDriverResult: 2}
	svc := NewDriverReportService(&stubStore{}, stubExcel{}, nil, nil, drivers, ingestor, nil, nil, directTxRunner{})

	n, err := svc.AutoBindDriver(context.Background(), "陳大華")

	require.NoError(t, err)
	assert.Equal(t, 2, n)
	require.Len(t, ingestor.backfillDriverCalls, 1)
	assert.Equal(t, driverID, ingestor.backfillDriverCalls[0].driverID)
}

func TestAutoBindDriver_AmbiguousMatchSkips(t *testing.T) {
	drivers := multiDriverStub{list: []DriverRef{{ID: uuid.New()}, {ID: uuid.New()}}}
	ingestor := &fakeIngestor{}
	svc := NewDriverReportService(&stubStore{}, stubExcel{}, nil, nil, drivers, ingestor, nil, nil, directTxRunner{})

	n, err := svc.AutoBindDriver(context.Background(), "陳大華")

	require.NoError(t, err)
	assert.Zero(t, n, "同名多筆司機主檔時不可自動猜測，留在待維護")
	assert.Empty(t, ingestor.backfillDriverCalls)
}

func TestAutoBindColumnsForCase_UniqueCaseNameBindsExactColumns(t *testing.T) {
	cases := stubCases{list: []CaseRef{{ID: testCaseID, Name: "陳大華", NameNormalized: "陳大華"}}}
	store := &mappingStore{
		stubStore: &stubStore{existing: []ColumnMapping{
			{ID: "col-1", Kind: "ride", CleanedName: "陳大華", ColumnHeader: "陳大華[去程]", MappingStatus: "pending"},
		}},
		previousStatus: "mapped", // 跳過回填分支，回填邏輯已由 mapping_test.go 覆蓋
	}
	svc := NewDriverReportService(store, stubExcel{}, nil, cases, stubDrivers{}, &fakeIngestor{}, nil, nil, directTxRunner{})

	n, err := svc.AutoBindColumnsForCase(context.Background(), "陳大華")

	require.NoError(t, err)
	assert.Equal(t, 1, n)
}

func TestAutoBindColumnsForCase_AmbiguousCaseNameSkips(t *testing.T) {
	cases := stubCases{list: []CaseRef{
		{ID: "case-1", Name: "陳大華", NameNormalized: "陳大華"},
		{ID: "case-2", Name: "陳大華", NameNormalized: "陳大華"},
	}}
	store := &mappingStore{
		stubStore: &stubStore{existing: []ColumnMapping{
			{ID: "col-1", Kind: "ride", CleanedName: "陳大華", ColumnHeader: "陳大華[去程]", MappingStatus: "pending"},
		}},
		previousStatus: "mapped",
	}
	svc := NewDriverReportService(store, stubExcel{}, nil, cases, stubDrivers{}, &fakeIngestor{}, nil, nil, directTxRunner{})

	n, err := svc.AutoBindColumnsForCase(context.Background(), "陳大華")

	require.NoError(t, err)
	assert.Zero(t, n, "同名多筆有效個案時不可自動猜測，留在待維護")
}

func TestAutoBindColumnsForCase_NoDirectionColumnSkips(t *testing.T) {
	cases := stubCases{list: []CaseRef{{ID: testCaseID, Name: "陳大華", NameNormalized: "陳大華"}}}
	store := &mappingStore{
		stubStore: &stubStore{existing: []ColumnMapping{
			{ID: "col-1", Kind: "ride", CleanedName: "陳大華", ColumnHeader: "陳大華", MappingStatus: "pending"},
		}},
		previousStatus: "mapped",
	}
	svc := NewDriverReportService(store, stubExcel{}, nil, cases, stubDrivers{}, &fakeIngestor{}, nil, nil, directTxRunner{})

	n, err := svc.AutoBindColumnsForCase(context.Background(), "陳大華")

	require.NoError(t, err)
	assert.Zero(t, n, "表頭沒有去程或回程標記時方向不明，不可自動綁定")
}

func TestAutoBindColumnsForCase_PartialFailureKeepsAlreadyBoundCount(t *testing.T) {
	cases := stubCases{list: []CaseRef{{ID: testCaseID, Name: "陳大華", NameNormalized: "陳大華"}}}
	store := &failingAtCallMappingStore{
		stubStore: &stubStore{existing: []ColumnMapping{
			{ID: "col-1", Kind: "ride", CleanedName: "陳大華", ColumnHeader: "陳大華[去程]", MappingStatus: "pending"},
			{ID: "col-2", Kind: "ride", CleanedName: "陳大華", ColumnHeader: "陳大華[回程]", MappingStatus: "pending"},
		}},
		failAtCall: 2,
	}
	svc := NewDriverReportService(store, stubExcel{}, nil, cases, stubDrivers{}, &fakeIngestor{}, nil, nil, directTxRunner{})

	n, err := svc.AutoBindColumnsForCase(context.Background(), "陳大華")

	require.Error(t, err)
	assert.Equal(t, 1, n, "第二欄綁定失敗前，第一欄已經生效，回傳筆數不能連同已成功的部分一起捨棄")
}

func TestRelinkAllPendingCaseColumns_KeepsPartialCountOnFailure(t *testing.T) {
	cases := stubCases{list: []CaseRef{{ID: testCaseID, Name: "陳大華", NameNormalized: "陳大華"}}}
	store := &failingAtCallMappingStore{
		stubStore: &stubStore{existing: []ColumnMapping{
			{ID: "col-1", Kind: "ride", CleanedName: "陳大華", ColumnHeader: "陳大華[去程]", MappingStatus: "pending"},
			{ID: "col-2", Kind: "ride", CleanedName: "陳大華", ColumnHeader: "陳大華[回程]", MappingStatus: "pending"},
		}},
		failAtCall: 2,
	}
	svc := NewDriverReportService(store, stubExcel{}, nil, cases, stubDrivers{}, &fakeIngestor{}, nil, nil, directTxRunner{})

	total, err := svc.RelinkAllPendingCaseColumns(context.Background())

	require.NoError(t, err, "單一姓名的部分失敗只記 log，不能讓整個手動重新比對回報失敗")
	assert.Equal(t, 1, total, "已成功綁定的那一欄要計入總數，不能因為同一姓名下另一欄失敗而整批歸零")
}
