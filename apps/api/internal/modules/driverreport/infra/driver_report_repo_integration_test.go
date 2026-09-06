//go:build integration

package infra

import (
	"context"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	"ltc-system/apps/api/internal/platform/pgxdb"
)

// TestDriverReportRepository_LockSerializesSameFormMonth 驗證不同 API connection
// 對同一表單／月份會在 transaction 結束前互斥，避免兩次覆蓋式匯入互相清除資料。
func TestDriverReportRepository_LockSerializesSameFormMonth(t *testing.T) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:postgrespassword@localhost:5432/ltc_system?sslmode=disable"
	}

	config, err := pgxpool.ParseConfig(dbURL)
	require.NoError(t, err)
	if config.MaxConns < 4 {
		config.MaxConns = 4
	}
	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	require.NoError(t, err)
	t.Cleanup(pool.Close)
	if err := pool.Ping(context.Background()); err != nil {
		t.Skipf("real Postgres not reachable at %s: %v", dbURL, err)
	}

	repo := NewDriverReportRepository(pool)
	runner := pgxdb.NewTxRunner(pool)
	formID := uuid.New()
	yearMonth := "2026-09"

	firstReady := make(chan struct{})
	releaseFirst := make(chan struct{})
	var releaseOnce sync.Once
	release := func() {
		releaseOnce.Do(func() { close(releaseFirst) })
	}
	defer release()
	firstErr := make(chan error, 1)
	go func() {
		firstErr <- runner.WithTx(context.Background(), func(txCtx context.Context) error {
			if err := repo.LockDriverReportImport(txCtx, formID, yearMonth); err != nil {
				return err
			}
			close(firstReady)
			<-releaseFirst
			return nil
		})
	}()

	select {
	case <-firstReady:
	case <-time.After(5 * time.Second):
		t.Fatal("第一個 transaction 未能取得匯報月份鎖")
	}

	secondReady := make(chan struct{})
	secondErr := make(chan error, 1)
	go func() {
		secondErr <- runner.WithTx(context.Background(), func(txCtx context.Context) error {
			if err := repo.LockDriverReportImport(txCtx, formID, yearMonth); err != nil {
				return err
			}
			close(secondReady)
			return nil
		})
	}()

	select {
	case <-secondReady:
		t.Fatal("同一表單／月份的第二個 transaction 不應在第一個提交前取得鎖")
	case <-time.After(300 * time.Millisecond):
	}

	release()
	require.NoError(t, <-firstErr)
	select {
	case <-secondReady:
	case <-time.After(5 * time.Second):
		t.Fatal("第一個 transaction 結束後，第二個 transaction 未能取得匯報月份鎖")
	}
	require.NoError(t, <-secondErr)
}
