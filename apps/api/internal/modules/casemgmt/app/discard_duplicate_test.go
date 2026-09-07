package app

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeDuplicateStagingStore 是 DuplicateStagingStore 的確定性測試替身。介面中 discard 路徑
// 用不到的 Insert 與 ListPending 定義於 case_duplicate_discard_test.go。
type fakeDuplicateStagingStore struct {
	candidate   *DuplicateCandidate
	getErr      error
	deleteCalls []uuid.UUID
	deleteRows  int64
	deleteErr   error
}

func (f *fakeDuplicateStagingStore) GetByID(context.Context, uuid.UUID) (*DuplicateCandidate, error) {
	if f.getErr != nil {
		return nil, f.getErr
	}
	return f.candidate, nil
}

func (f *fakeDuplicateStagingStore) Resolve(context.Context, uuid.UUID, string, uuid.UUID, *uuid.UUID) (int64, error) {
	return 0, nil
}

func (f *fakeDuplicateStagingStore) Delete(_ context.Context, id uuid.UUID) (int64, error) {
	f.deleteCalls = append(f.deleteCalls, id)
	if f.deleteErr != nil {
		return 0, f.deleteErr
	}
	return f.deleteRows, nil
}

// pendingCandidate 造一筆帶身分證密文與地址的待裁決暫存列，讓稽核快照的個資檢查有效。
func pendingCandidate(id uuid.UUID) *DuplicateCandidate {
	homeAddress := "苗栗縣竹南鎮某路 1 號"
	return &DuplicateCandidate{
		ID:               id,
		Status:           "pending",
		Name:             "王大明",
		FileHash:         "hash-1",
		RowKey:           "Sheet1:2",
		RowIndex:         2,
		SheetName:        "Sheet1",
		DuplicateCaseID:  uuid.New(),
		NationalIDCipher: []byte("cipher"),
		NationalIDMasked: "A12****789",
		HomeAddress:      &homeAddress,
	}
}

func TestDiscardDuplicateCandidate_DeletesRowAndWritesAudit(t *testing.T) {
	id := uuid.New()
	actorID := uuid.New()
	cand := pendingCandidate(id)
	staging := &fakeDuplicateStagingStore{candidate: cand, deleteRows: 1}
	audit := &fakeCaseAuditWriter{}
	svc := NewCaseService(testConfig(), newFakeCaseStore(), nil, audit, nil, staging)

	err := svc.DiscardDuplicateCandidate(context.Background(), id, actorID, "staff", "10.0.0.1", "chrome")

	require.NoError(t, err)
	require.Len(t, staging.deleteCalls, 1)
	assert.Equal(t, id, staging.deleteCalls[0])

	require.Len(t, audit.entries, 1)
	entry := audit.entries[0]
	assert.Equal(t, "ignore", entry.Action)
	assert.Equal(t, "case_import_duplicate_rows", entry.EntityType)

	// 稽核要能回答「誰在哪一列做的」，缺任一欄位這筆刪除就追不回來。
	require.NotNil(t, entry.ActorID)
	assert.Equal(t, actorID, *entry.ActorID)
	require.NotNil(t, entry.ActorRole)
	assert.Equal(t, "staff", *entry.ActorRole)
	require.NotNil(t, entry.EntityID)
	assert.Equal(t, id.String(), *entry.EntityID)
	require.NotNil(t, entry.IPAddress)
	assert.Equal(t, "10.0.0.1", *entry.IPAddress)
	require.NotNil(t, entry.UserAgent)
	assert.Equal(t, "chrome", *entry.UserAgent)

	snapshot, ok := entry.BeforeData.(map[string]interface{})
	require.True(t, ok, "稽核快照應為明確挑選欄位的 map，而非整個暫存列")
	assert.Equal(t, "王大明", snapshot["name"])
	assert.Equal(t, "hash-1", snapshot["fileHash"])
	assert.Equal(t, cand.RowIndex, snapshot["rowIndex"])
	assert.Equal(t, cand.SheetName, snapshot["sheetName"])
	assert.Equal(t, cand.RowKey, snapshot["rowKey"])
	assert.Equal(t, cand.DuplicateCaseID.String(), snapshot["duplicateCaseId"])
	for _, piiField := range []string{"nationalIdCipher", "nationalIdMasked", "homeAddress", "registeredAddress", "birthDate"} {
		_, exists := snapshot[piiField]
		assert.False(t, exists, "稽核快照不得包含個資欄位 %s", piiField)
	}
}

func TestDiscardDuplicateCandidate_NotFound(t *testing.T) {
	staging := &fakeDuplicateStagingStore{candidate: nil}
	svc := NewCaseService(testConfig(), newFakeCaseStore(), nil, nil, nil, staging)

	err := svc.DiscardDuplicateCandidate(context.Background(), uuid.New(), uuid.New(), "admin", "", "")

	assert.ErrorIs(t, err, ErrDuplicateCandidateNotFound)
	assert.Empty(t, staging.deleteCalls, "查無資料時不應嘗試刪除")
}

func TestDiscardDuplicateCandidate_AlreadyResolved(t *testing.T) {
	id := uuid.New()
	cand := pendingCandidate(id)
	cand.Status = "confirmed_new"
	staging := &fakeDuplicateStagingStore{candidate: cand}
	svc := NewCaseService(testConfig(), newFakeCaseStore(), nil, nil, nil, staging)

	err := svc.DiscardDuplicateCandidate(context.Background(), id, uuid.New(), "admin", "", "")

	assert.ErrorIs(t, err, ErrDuplicateCandidateResolved)
	assert.Empty(t, staging.deleteCalls)
}

// 並發保護：讀到 pending 之後、刪除之前被他人裁決掉，rowsAffected 會是 0。
func TestDiscardDuplicateCandidate_ConflictWhenRowDisappears(t *testing.T) {
	id := uuid.New()
	staging := &fakeDuplicateStagingStore{candidate: pendingCandidate(id), deleteRows: 0}
	audit := &fakeCaseAuditWriter{}
	svc := NewCaseService(testConfig(), newFakeCaseStore(), nil, audit, nil, staging)

	err := svc.DiscardDuplicateCandidate(context.Background(), id, uuid.New(), "admin", "", "")

	assert.ErrorIs(t, err, ErrDuplicateCandidateResolved)
	assert.Empty(t, audit.entries, "沒有實際刪到資料就不得寫稽核")
}

func TestDiscardDuplicateCandidate_RollsBackWhenAuditFails(t *testing.T) {
	id := uuid.New()
	staging := &fakeDuplicateStagingStore{candidate: pendingCandidate(id), deleteRows: 1}
	audit := &fakeCaseAuditWriter{err: assert.AnError}
	txRunner := &fakeCaseTransactionRunner{}
	svc := NewCaseService(testConfig(), newFakeCaseStore(), nil, audit, nil, staging, txRunner)

	err := svc.DiscardDuplicateCandidate(context.Background(), id, uuid.New(), "admin", "", "")

	require.Error(t, err, "稽核寫入失敗必須讓整筆忽略操作失敗，不能只刪資料不留紀錄")
	assert.Equal(t, 1, txRunner.calls, "刪除與稽核必須在同一交易內")
}

// TestDiscardDuplicateCandidate_PropagatesStoreErrors 確認相依失敗會原樣往上傳，
// 且在任一失敗路徑都不會留下「資料已刪但沒有稽核」的狀態。
func TestDiscardDuplicateCandidate_PropagatesStoreErrors(t *testing.T) {
	t.Run("讀取暫存列失敗", func(t *testing.T) {
		staging := &fakeDuplicateStagingStore{getErr: errors.New("db down")}
		svc := NewCaseService(testConfig(), newFakeCaseStore(), nil, nil, nil, staging)

		err := svc.DiscardDuplicateCandidate(context.Background(), uuid.New(), uuid.New(), "admin", "", "")

		assert.ErrorContains(t, err, "db down")
		assert.Empty(t, staging.deleteCalls)
	})

	t.Run("刪除失敗不寫稽核", func(t *testing.T) {
		id := uuid.New()
		staging := &fakeDuplicateStagingStore{candidate: pendingCandidate(id), deleteErr: errors.New("delete failed")}
		audit := &fakeCaseAuditWriter{}
		svc := NewCaseService(testConfig(), newFakeCaseStore(), nil, audit, nil, staging)

		err := svc.DiscardDuplicateCandidate(context.Background(), id, uuid.New(), "admin", "", "")

		assert.ErrorContains(t, err, "delete failed")
		assert.Empty(t, audit.entries)
	})
}

// TestDiscardDuplicateCandidate_WorksWithoutAuditWriter 確認未注入稽核時仍可刪除，
// 這是 NewCaseService 允許 auditRepo 為 nil 的既有約定。
func TestDiscardDuplicateCandidate_WorksWithoutAuditWriter(t *testing.T) {
	id := uuid.New()
	staging := &fakeDuplicateStagingStore{candidate: pendingCandidate(id), deleteRows: 1}
	svc := NewCaseService(testConfig(), newFakeCaseStore(), nil, nil, nil, staging)

	require.NoError(t, svc.DiscardDuplicateCandidate(context.Background(), id, uuid.New(), "admin", "", ""))
	assert.Equal(t, []uuid.UUID{id}, staging.deleteCalls)
}

// TestResolveDuplicateCandidate_RejectsDiscardedDecision 鎖住「忽略只走專屬端點」：
// 裁決路徑不得接受 discarded，否則兩條路徑會各自寫出不同的稽核與資料狀態。
func TestResolveDuplicateCandidate_RejectsDiscardedDecision(t *testing.T) {
	staging := &fakeDuplicateStagingStore{candidate: pendingCandidate(uuid.New())}
	svc := NewCaseService(testConfig(), newFakeCaseStore(), nil, nil, nil, staging)

	_, err := svc.ResolveDuplicateCandidate(context.Background(), uuid.New(), "discarded", nil, false, uuid.New(), "admin", "", "")

	assert.ErrorIs(t, err, ErrInvalidDuplicateDecision, "忽略只走專屬端點，不混用裁決路徑")
}
