package app

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// recordingOpsAuditWriter 收集稽核內容；err 用來模擬稽核寫入失敗。
type recordingOpsAuditWriter struct {
	entries []AuditEntry
	err     error
}

func (w *recordingOpsAuditWriter) Write(_ context.Context, e AuditEntry) error {
	w.entries = append(w.entries, e)
	return w.err
}

// countingOpsTxRunner 直接執行 fn 並記錄交易次數。
type countingOpsTxRunner struct{ calls int }

func (r *countingOpsTxRunner) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	r.calls++
	return fn(ctx)
}

// failingConflictStore 讓 GetConflict 回傳錯誤，覆蓋讀取相依失敗的路徑。
type failingConflictStore struct {
	recordingAttendanceStore
	getErr error
}

func (s *failingConflictStore) GetConflict(context.Context, uuid.UUID) (*AttendanceImportConflict, error) {
	return nil, s.getErr
}

func newPendingConflict(id uuid.UUID) *AttendanceImportConflict {
	return &AttendanceImportConflict{
		ID:             id,
		DriverID:       uuid.New(),
		RecordDate:     time.Date(2026, 7, 15, 0, 0, 0, 0, time.UTC),
		ExistingStatus: "leave",
		ImportedStatus: "work",
		Status:         "pending",
	}
}

func TestAttendanceService_IgnoreConflict_DeletesRowAndKeepsManualRecord(t *testing.T) {
	conflictID := uuid.New()
	conflict := newPendingConflict(conflictID)
	store := &recordingAttendanceStore{conflicts: map[uuid.UUID]*AttendanceImportConflict{conflictID: conflict}}
	audit := &recordingOpsAuditWriter{}
	tx := &countingOpsTxRunner{}
	svc := NewAttendanceService(store, emptyDriverLister{}, audit, stubHolidayReader{}, WithAttendanceTxRunner(tx))

	actorID := uuid.New()
	actorRole := "staff"
	ip := "10.0.0.1"
	ua := "chrome"

	err := svc.IgnoreConflict(context.Background(), conflictID, &actorID, &actorRole, AuditContext{IPAddress: &ip, UserAgent: &ua})

	require.NoError(t, err)
	assert.Equal(t, []uuid.UUID{conflictID}, store.deleteCalls)
	assert.Empty(t, store.upsertCalls, "忽略衝突不得改動 attendance_records，人工登記維持原值")
	assert.Empty(t, store.resolveCalls, "忽略走刪除路徑，不得同時標記為已裁決")
	assert.Equal(t, 1, tx.calls, "刪除與稽核必須在同一交易內")

	require.Len(t, audit.entries, 1)
	entry := audit.entries[0]
	assert.Equal(t, "ignore", entry.Action)
	assert.Equal(t, "attendance_import_conflicts", entry.EntityType)
	require.NotNil(t, entry.EntityID)
	assert.Equal(t, conflictID.String(), *entry.EntityID)
	require.NotNil(t, entry.ActorID)
	assert.Equal(t, actorID, *entry.ActorID)
	require.NotNil(t, entry.IPAddress)
	assert.Equal(t, ip, *entry.IPAddress)
	require.NotNil(t, entry.UserAgent)
	assert.Equal(t, ua, *entry.UserAgent)

	// 稽核要保留被刪掉的衝突內容，否則刪除後無從還原當時的差異。
	snapshot, ok := entry.BeforeData.(AttendanceConflictAuditSnapshot)
	require.True(t, ok, "BeforeData 應為 AttendanceConflictAuditSnapshot")
	assert.Equal(t, conflictID, snapshot.ID)
	assert.Equal(t, "leave", snapshot.ExistingStatus)
	assert.Equal(t, "work", snapshot.ImportedStatus)
}

func TestAttendanceService_IgnoreConflict_NotFound(t *testing.T) {
	store := &recordingAttendanceStore{conflicts: map[uuid.UUID]*AttendanceImportConflict{}}
	audit := &recordingOpsAuditWriter{}
	svc := NewAttendanceService(store, emptyDriverLister{}, audit, stubHolidayReader{})

	err := svc.IgnoreConflict(context.Background(), uuid.New(), nil, nil)

	assert.ErrorIs(t, err, ErrAttendanceConflictNotFound)
	assert.Empty(t, store.deleteCalls, "查無衝突時不得嘗試刪除")
	assert.Empty(t, audit.entries)
}

func TestAttendanceService_IgnoreConflict_PropagatesGetError(t *testing.T) {
	store := &failingConflictStore{getErr: errors.New("db down")}
	svc := NewAttendanceService(store, emptyDriverLister{}, discardAuditWriter{}, stubHolidayReader{})

	err := svc.IgnoreConflict(context.Background(), uuid.New(), nil, nil)

	assert.ErrorContains(t, err, "db down")
	assert.NotErrorIs(t, err, ErrAttendanceConflictNotFound, "相依失敗不得被誤判成查無資料")
}

// 稽核寫入失敗要讓整筆忽略失敗，避免資料被刪掉卻沒有任何紀錄可追。
func TestAttendanceService_IgnoreConflict_FailsWhenAuditFails(t *testing.T) {
	conflictID := uuid.New()
	store := &recordingAttendanceStore{conflicts: map[uuid.UUID]*AttendanceImportConflict{
		conflictID: newPendingConflict(conflictID),
	}}
	audit := &recordingOpsAuditWriter{err: errors.New("audit down")}
	tx := &countingOpsTxRunner{}
	svc := NewAttendanceService(store, emptyDriverLister{}, audit, stubHolidayReader{}, WithAttendanceTxRunner(tx))

	err := svc.IgnoreConflict(context.Background(), conflictID, nil, nil)

	require.Error(t, err)
	assert.Equal(t, 1, tx.calls, "稽核失敗要能整筆回滾，代表刪除必須在交易內")
}

// 未注入交易執行器時仍要能運作，這是 NewAttendanceService 把 txRunner 設為選用的既有約定。
func TestAttendanceService_IgnoreConflict_WorksWithoutTxRunner(t *testing.T) {
	conflictID := uuid.New()
	store := &recordingAttendanceStore{conflicts: map[uuid.UUID]*AttendanceImportConflict{
		conflictID: newPendingConflict(conflictID),
	}}
	svc := NewAttendanceService(store, emptyDriverLister{}, discardAuditWriter{}, stubHolidayReader{})

	require.NoError(t, svc.IgnoreConflict(context.Background(), conflictID, nil, nil))
	assert.Equal(t, []uuid.UUID{conflictID}, store.deleteCalls)
}
