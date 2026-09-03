package app

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mappingStore 讓測試能控制 UpdateColumnMappingByID 回傳的更新前狀態，
// 藉此驗證只有「剛從待維護變成已對應」才會觸發回填。
type mappingStore struct {
	*stubStore
	previousStatus string
	returnErr      error
}

func (s *mappingStore) UpdateColumnMappingByID(context.Context, string, string, *string, *int16) (uuid.UUID, string, int, string, error) {
	if s.returnErr != nil {
		return uuid.Nil, "", 0, "", s.returnErr
	}
	return uuid.MustParse(testFormID), "1.吳桂 [去程]", 3, s.previousStatus, nil
}

func newMappingService(previousStatus string, ingestor *fakeIngestor) (*DriverReportService, *mappingStore) {
	store := &mappingStore{
		stubStore: &stubStore{
			form: &ReportForm{
				ID:        uuid.MustParse(testFormID),
				VehicleID: uuid.MustParse("33333333-3333-3333-3333-333333333333"),
			},
		},
		previousStatus: previousStatus,
	}
	svc := NewDriverReportService(store, stubExcel{}, nil, nil, nil, ingestor, nil, nil, directTxRunner{})
	return svc, store
}

func TestUpdateColumnMapping_BindingPendingColumnTriggersBackfill(t *testing.T) {
	ingestor := &fakeIngestor{backfillResult: 2}
	svc, _ := newMappingService("pending", ingestor)
	caseID := testCaseID
	legSeq := int16(1)

	written, err := svc.UpdateColumnMapping(context.Background(), "col-1", "mapped", &caseID, &legSeq)

	require.NoError(t, err)
	assert.Equal(t, 2, written, "回填筆數要回傳給呼叫端，讓使用者看到綁定後實際補寫了多少筆")
	require.Len(t, ingestor.backfillCalls, 1)
	call := ingestor.backfillCalls[0]
	assert.Equal(t, uuid.MustParse(testFormID), call.formID)
	assert.Equal(t, "1.吳桂 [去程]", call.columnHeader)
	assert.Equal(t, 3, call.columnIndex)
	assert.Equal(t, uuid.MustParse(testCaseID), call.caseID)
	assert.Equal(t, int16(1), call.legSeq)
}

func TestUpdateColumnMapping_AlreadyMappedColumnDoesNotRetriggerBackfill(t *testing.T) {
	ingestor := &fakeIngestor{backfillResult: 5}
	svc, _ := newMappingService("mapped", ingestor)
	caseID := testCaseID
	legSeq := int16(1)

	written, err := svc.UpdateColumnMapping(context.Background(), "col-1", "mapped", &caseID, &legSeq)

	require.NoError(t, err)
	assert.Zero(t, written)
	assert.Empty(t, ingestor.backfillCalls, "重複綁定已對應的欄位不能再補寫一次，否則會產生重複的搭乘來源")
}

func TestUpdateColumnMapping_IgnoringColumnDoesNotTriggerBackfill(t *testing.T) {
	ingestor := &fakeIngestor{}
	svc, _ := newMappingService("pending", ingestor)

	written, err := svc.UpdateColumnMapping(context.Background(), "col-1", "ignored", nil, nil)

	require.NoError(t, err)
	assert.Zero(t, written)
	assert.Empty(t, ingestor.backfillCalls)
}

func TestUpdateColumnMapping_RequiresCaseAndLegSeqWhenMapped(t *testing.T) {
	ingestor := &fakeIngestor{}
	svc, _ := newMappingService("pending", ingestor)

	_, err := svc.UpdateColumnMapping(context.Background(), "col-1", "mapped", nil, nil)

	require.Error(t, err)
	assert.Empty(t, ingestor.backfillCalls)
}
