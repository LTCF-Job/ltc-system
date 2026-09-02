//go:build integration

package infra_test

import (
	"context"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	"ltc-system/apps/api/internal/domain/crypto"
	demoapp "ltc-system/apps/api/internal/modules/demo/app"
	demoinfra "ltc-system/apps/api/internal/modules/demo/infra"
	"ltc-system/apps/api/internal/platform/config"
)

// TestResetRepository_Reset 對真實 Postgres 驗證 Demo 重置的完整交易：清空業務資料表、
// 套用種子資料、重新加密假身分證字號後可被正確解密——不用 fake repository，直接打真的資料庫。
//
// 執行方式（DATABASE_URL 指向的資料庫必須已跑過 apps/api/migrations 且沒有 auth schema，
// 比照 apps/api/cmd/migrate 的用法）：
//
//	DATABASE_URL=postgres://postgres:postgres@localhost:5432/ltc_demo_test?sslmode=disable \
//	  go test -tags=integration ./internal/modules/demo/... -run TestResetRepository_Reset -v
func TestResetRepository_Reset(t *testing.T) {
	pool, cfg := setupIntegrationDB(t)

	repo, err := demoinfra.NewResetRepository(pool, cfg)
	require.NoError(t, err)

	require.NoError(t, repo.Reset(context.Background()))

	var caseCount, driverCount, regionCount int
	require.NoError(t, pool.QueryRow(context.Background(), `SELECT count(*) FROM cases`).Scan(&caseCount))
	require.NoError(t, pool.QueryRow(context.Background(), `SELECT count(*) FROM drivers`).Scan(&driverCount))
	require.NoError(t, pool.QueryRow(context.Background(), `SELECT count(*) FROM regions`).Scan(&regionCount))
	require.Greater(t, caseCount, 0, "重置後應有種子個案資料")
	require.Greater(t, driverCount, 0, "重置後應有種子司機資料")
	require.Equal(t, 22, regionCount, "regions 是靜態參考資料，重置不應清空或重複寫入")

	var cipher []byte
	var driverID string
	require.NoError(t, pool.QueryRow(context.Background(),
		`SELECT id, national_id_cipher FROM drivers ORDER BY code LIMIT 1`,
	).Scan(&driverID, &cipher))

	plain, err := crypto.Decrypt(cipher, cfg.EncryptionKey)
	require.NoError(t, err, "種子司機的身分證密文必須能被本服務目前的金鑰解密")
	require.True(t, crypto.ValidateNationalID(plain), "重新加密產生的假身分證字號必須通過檢查碼驗證")

	// 重跑一次驗證冪等：業務資料列數不應累加。
	require.NoError(t, repo.Reset(context.Background()))
	var caseCountAfterSecondReset int
	require.NoError(t, pool.QueryRow(context.Background(), `SELECT count(*) FROM cases`).Scan(&caseCountAfterSecondReset))
	require.Equal(t, caseCount, caseCountAfterSecondReset, "重複重置不應改變種子資料筆數")
}

// TestResetService_ConcurrentRequestBlocksRealReset 用真實 ResetRepository 驗證
// ConcurrencyGuard 在真實資料庫工作負載下仍能讓重置等待進行中的請求釋放共享鎖。
func TestResetService_ConcurrentRequestBlocksRealReset(t *testing.T) {
	pool, cfg := setupIntegrationDB(t)

	repo, err := demoinfra.NewResetRepository(pool, cfg)
	require.NoError(t, err)
	guard := demoapp.NewConcurrencyGuard()
	svc := demoapp.NewResetService(repo, guard)

	var events []string
	var mu sync.Mutex
	record := func(e string) {
		mu.Lock()
		events = append(events, e)
		mu.Unlock()
	}

	releaseRequest := guard.BeginRequest()
	record("request_started")

	resetDone := make(chan error, 1)
	go func() {
		_, err := svc.Reset(context.Background())
		resetDone <- err
	}()

	select {
	case <-resetDone:
		t.Fatal("Reset 在模擬請求釋放共享鎖前就完成了")
	case <-time.After(200 * time.Millisecond):
	}

	record("request_released")
	releaseRequest()

	select {
	case err := <-resetDone:
		require.NoError(t, err)
		record("reset_completed")
	case <-time.After(10 * time.Second):
		t.Fatal("Reset 在請求釋放共享鎖後逾時仍未完成")
	}

	require.Equal(t, []string{"request_started", "request_released", "reset_completed"}, events)
}

func setupIntegrationDB(t *testing.T) (*pgxpool.Pool, *config.Config) {
	t.Helper()

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:postgres@localhost:5432/ltc_demo_test?sslmode=disable"
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbURL)
	require.NoError(t, err)
	t.Cleanup(pool.Close)

	if err := pool.Ping(ctx); err != nil {
		t.Skipf("real Postgres not reachable at %s: %v", dbURL, err)
	}

	t.Setenv("APP_ENV", "local")
	t.Setenv("ENCRYPTION_KEY", "MDEwMjAzMDQwNTA2MDcwODAxMDIwMzA0MDUwNjA3MDg=")
	t.Setenv("HMAC_KEY", "MDkwODAwMDcwNjA1MDQwMzA5MDgwMDA3MDYwNTA0MDM=")
	cfg, err := config.LoadFromEnv()
	require.NoError(t, err)

	return pool, cfg
}
