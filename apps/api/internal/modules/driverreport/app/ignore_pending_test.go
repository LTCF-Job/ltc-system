package app

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// countingTxRunner 直接執行 fn，同時記錄交易次數，讓測試能斷言刪除有沒有被包在交易內。
type countingTxRunner struct{ calls int }

func (r *countingTxRunner) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	r.calls++
	return fn(ctx)
}

// recordingAuditWriter 收集稽核內容；err 用來模擬稽核失敗。
type recordingAuditWriter struct {
	entries []AuditEntry
	err     error
}

func (w *recordingAuditWriter) Write(_ context.Context, e AuditEntry) error {
	w.entries = append(w.entries, e)
	return w.err
}

func testActor() Actor {
	return Actor{
		ActorID:   uuid.MustParse("33333333-3333-3333-3333-333333333333"),
		ActorRole: "staff",
		IPAddress: "10.0.0.1",
		UserAgent: "chrome",
	}
}

// assertIgnoreAudit 斷言「忽略此筆」寫出的稽核內容：可追溯到誰、動了哪張表的哪一列。
func assertIgnoreAudit(t *testing.T, entry AuditEntry, entityType, entityID string) {
	t.Helper()
	actor := testActor()

	assert.Equal(t, "ignore", entry.Action)
	assert.Equal(t, entityType, entry.EntityType)
	require.NotNil(t, entry.EntityID)
	assert.Equal(t, entityID, *entry.EntityID)
	require.NotNil(t, entry.ActorID)
	assert.Equal(t, actor.ActorID, *entry.ActorID)
	require.NotNil(t, entry.ActorRole)
	assert.Equal(t, actor.ActorRole, *entry.ActorRole)
	require.NotNil(t, entry.IPAddress)
	assert.Equal(t, actor.IPAddress, *entry.IPAddress)
	require.NotNil(t, entry.UserAgent)
	assert.Equal(t, actor.UserAgent, *entry.UserAgent)
}

func TestDriverReportService_IgnoreColumn(t *testing.T) {
	ctx := context.Background()
	const colID = "44444444-4444-4444-4444-444444444444"

	t.Run("成功刪除並寫稽核", func(t *testing.T) {
		store := &stubStore{deleteColumnRows: 1}
		audit := &recordingAuditWriter{}
		svc := NewDriverReportService(store, nil, nil, nil, nil, &fakeIngestor{}, nil, audit, nil)

		require.NoError(t, svc.IgnoreColumn(ctx, colID, testActor()))

		assert.Equal(t, []string{colID}, store.deleteColumnCalls)
		require.Len(t, audit.entries, 1)
		assertIgnoreAudit(t, audit.entries[0], "form_columns", colID)
	})

	t.Run("編號格式錯誤時不碰資料庫", func(t *testing.T) {
		store := &stubStore{deleteColumnRows: 1}
		svc := NewDriverReportService(store, nil, nil, nil, nil, &fakeIngestor{}, nil, nil, nil)

		err := svc.IgnoreColumn(ctx, "not-a-uuid", testActor())

		assert.ErrorContains(t, err, "欄位編號格式錯誤")
		assert.Empty(t, store.deleteColumnCalls, "格式錯誤必須在進資料庫前擋掉")
	})

	t.Run("查無資料回 ErrPendingItemNotFound 且不寫稽核", func(t *testing.T) {
		store := &stubStore{deleteColumnRows: 0}
		audit := &recordingAuditWriter{}
		svc := NewDriverReportService(store, nil, nil, nil, nil, &fakeIngestor{}, nil, audit, nil)

		err := svc.IgnoreColumn(ctx, colID, testActor())

		assert.ErrorIs(t, err, ErrPendingItemNotFound)
		assert.Empty(t, audit.entries, "沒刪到資料就不得留下忽略紀錄")
	})

	t.Run("刪除失敗時原樣回傳錯誤", func(t *testing.T) {
		store := &stubStore{deleteColumnErr: errors.New("db down")}
		audit := &recordingAuditWriter{}
		svc := NewDriverReportService(store, nil, nil, nil, nil, &fakeIngestor{}, nil, audit, nil)

		err := svc.IgnoreColumn(ctx, colID, testActor())

		assert.ErrorContains(t, err, "db down")
		assert.NotErrorIs(t, err, ErrPendingItemNotFound, "相依失敗不得被誤判成查無資料")
		assert.Empty(t, audit.entries)
	})

	t.Run("稽核失敗不影響刪除結果", func(t *testing.T) {
		store := &stubStore{deleteColumnRows: 1}
		audit := &recordingAuditWriter{err: errors.New("audit down")}
		svc := NewDriverReportService(store, nil, nil, nil, nil, &fakeIngestor{}, nil, audit, nil)

		assert.NoError(t, svc.IgnoreColumn(ctx, colID, testActor()),
			"稽核為非阻斷處理，失敗只降級為警告日誌")
	})

	t.Run("未注入稽核時仍可刪除", func(t *testing.T) {
		store := &stubStore{deleteColumnRows: 1}
		svc := NewDriverReportService(store, nil, nil, nil, nil, &fakeIngestor{}, nil, nil, nil)

		require.NoError(t, svc.IgnoreColumn(ctx, colID, testActor()))
		assert.Equal(t, []string{colID}, store.deleteColumnCalls)
	})
}

func TestDriverReportService_IgnoreRowConflict(t *testing.T) {
	ctx := context.Background()
	const conflictID = "55555555-5555-5555-5555-555555555555"

	t.Run("成功刪除並寫稽核", func(t *testing.T) {
		ingestor := &fakeIngestor{deleteRowConflictRows: 1}
		audit := &recordingAuditWriter{}
		svc := NewDriverReportService(&stubStore{}, nil, nil, nil, nil, ingestor, nil, audit, nil)

		require.NoError(t, svc.IgnoreRowConflict(ctx, conflictID, testActor()))

		assert.Equal(t, []uuid.UUID{uuid.MustParse(conflictID)}, ingestor.deleteRowConflictCalls)
		require.Len(t, audit.entries, 1)
		assertIgnoreAudit(t, audit.entries[0], "ride_source_row_conflicts", conflictID)
	})

	t.Run("編號格式錯誤時不碰資料庫", func(t *testing.T) {
		ingestor := &fakeIngestor{deleteRowConflictRows: 1}
		svc := NewDriverReportService(&stubStore{}, nil, nil, nil, nil, ingestor, nil, nil, nil)

		err := svc.IgnoreRowConflict(ctx, "not-a-uuid", testActor())

		assert.ErrorContains(t, err, "衝突編號格式錯誤")
		assert.Empty(t, ingestor.deleteRowConflictCalls)
	})

	t.Run("已被他人裁決回 ErrPendingItemNotFound", func(t *testing.T) {
		ingestor := &fakeIngestor{deleteRowConflictRows: 0}
		audit := &recordingAuditWriter{}
		svc := NewDriverReportService(&stubStore{}, nil, nil, nil, nil, ingestor, nil, audit, nil)

		err := svc.IgnoreRowConflict(ctx, conflictID, testActor())

		assert.ErrorIs(t, err, ErrPendingItemNotFound)
		assert.Empty(t, audit.entries)
	})

	t.Run("刪除失敗時原樣回傳錯誤", func(t *testing.T) {
		ingestor := &fakeIngestor{deleteRowConflictErr: errors.New("db down")}
		svc := NewDriverReportService(&stubStore{}, nil, nil, nil, nil, ingestor, nil, nil, nil)

		err := svc.IgnoreRowConflict(ctx, conflictID, testActor())

		assert.ErrorContains(t, err, "db down")
		assert.NotErrorIs(t, err, ErrPendingItemNotFound)
	})
}

func TestDriverReportService_IgnoreSubmission(t *testing.T) {
	ctx := context.Background()
	const submissionID = "66666666-6666-6666-6666-666666666666"

	t.Run("在交易內刪除並寫稽核", func(t *testing.T) {
		ingestor := &fakeIngestor{deleteSubmissionRows: 1}
		audit := &recordingAuditWriter{}
		tx := &countingTxRunner{}
		svc := NewDriverReportService(&stubStore{}, nil, nil, nil, nil, ingestor, nil, audit, tx)

		require.NoError(t, svc.IgnoreSubmission(ctx, submissionID, testActor()))

		assert.Equal(t, []uuid.UUID{uuid.MustParse(submissionID)}, ingestor.deleteSubmissionCalls)
		assert.Equal(t, 1, tx.calls, "刪除提交會連帶影響搭乘來源，必須在同一交易內重算")
		require.Len(t, audit.entries, 1)
		assertIgnoreAudit(t, audit.entries[0], "form_submissions", submissionID)
	})

	t.Run("未注入交易時直接拒絕", func(t *testing.T) {
		ingestor := &fakeIngestor{deleteSubmissionRows: 1}
		svc := NewDriverReportService(&stubStore{}, nil, nil, nil, nil, ingestor, nil, nil, nil)

		err := svc.IgnoreSubmission(ctx, submissionID, testActor())

		require.Error(t, err)
		assert.Empty(t, ingestor.deleteSubmissionCalls, "沒有交易邊界就不得動資料")
	})

	t.Run("編號格式錯誤時不開交易", func(t *testing.T) {
		ingestor := &fakeIngestor{deleteSubmissionRows: 1}
		tx := &countingTxRunner{}
		svc := NewDriverReportService(&stubStore{}, nil, nil, nil, nil, ingestor, nil, nil, tx)

		err := svc.IgnoreSubmission(ctx, "not-a-uuid", testActor())

		assert.ErrorContains(t, err, "匯報列編號格式錯誤")
		assert.Zero(t, tx.calls)
		assert.Empty(t, ingestor.deleteSubmissionCalls)
	})

	t.Run("查無資料回 ErrPendingItemNotFound 且不寫稽核", func(t *testing.T) {
		ingestor := &fakeIngestor{deleteSubmissionRows: 0}
		audit := &recordingAuditWriter{}
		tx := &countingTxRunner{}
		svc := NewDriverReportService(&stubStore{}, nil, nil, nil, nil, ingestor, nil, audit, tx)

		err := svc.IgnoreSubmission(ctx, submissionID, testActor())

		assert.ErrorIs(t, err, ErrPendingItemNotFound)
		assert.Empty(t, audit.entries)
	})

	t.Run("刪除失敗時原樣回傳錯誤", func(t *testing.T) {
		ingestor := &fakeIngestor{deleteSubmissionErr: errors.New("db down")}
		tx := &countingTxRunner{}
		svc := NewDriverReportService(&stubStore{}, nil, nil, nil, nil, ingestor, nil, nil, tx)

		err := svc.IgnoreSubmission(ctx, submissionID, testActor())

		assert.ErrorContains(t, err, "db down")
		assert.NotErrorIs(t, err, ErrPendingItemNotFound)
	})
}
