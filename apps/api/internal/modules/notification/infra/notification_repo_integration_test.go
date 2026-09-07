//go:build integration

package infra_test

import (
	"context"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	"ltc-system/apps/api/internal/modules/notification/app"
	"ltc-system/apps/api/internal/modules/notification/infra"
)

// TestBatchCreateRecipients_SimpleProtocol 以 simple protocol 連線驗證批次新增收件人。
// 正式環境的 pgxpool 走 QueryExecModeSimpleProtocol（見 cmd/server/main.go connectDatabase），
// 該模式下所有參數都在 client 端轉成字面值，Go 端必須傳可編碼的型別，
// 否則 []uuid.UUID 之類的參數會在執行期回傳 "cannot find encode plan"。
//
// 執行方式（需要本機 docker-compose.local.yml 啟動的 Postgres）：
//
//	docker compose -f docker-compose.local.yml up -d postgres
//	DATABASE_URL=postgres://postgres:postgrespassword@localhost:5432/ltc_system?sslmode=disable \
//	  go test -tags=integration ./internal/modules/notification/... -v
func TestBatchCreateRecipients_SimpleProtocol(t *testing.T) {
	pool := newSimpleProtocolPool(t)
	repo := infra.NewNotificationRepository(pool)
	ctx := context.Background()

	creator := uuid.New()
	displayName := "值班窗口"
	items := []app.Recipient{
		{Topic: "missing_report", Email: "batch-" + uuid.NewString() + "@example.com", DisplayName: &displayName, CreatedBy: creator},
		{Topic: "missing_report", Email: "batch-" + uuid.NewString() + "@example.com", CreatedBy: creator},
	}
	t.Cleanup(func() {
		for _, item := range items {
			_, _ = pool.Exec(context.Background(), `DELETE FROM notification_recipients WHERE email = $1`, item.Email)
		}
	})

	created, err := repo.BatchCreateRecipients(ctx, items)
	require.NoError(t, err)
	require.Len(t, created, 2)

	byEmail := map[string]app.Recipient{}
	for _, item := range created {
		byEmail[item.Email] = item
	}

	first := byEmail[items[0].Email]
	require.Equal(t, creator, first.CreatedBy)
	require.Equal(t, "email", first.RecipientType)
	require.True(t, first.Active)
	require.NotNil(t, first.DisplayName)
	require.Equal(t, displayName, *first.DisplayName)

	second := byEmail[items[1].Email]
	require.Equal(t, creator, second.CreatedBy)
	require.Nil(t, second.DisplayName, "未提供顯示名稱時應寫入 NULL")

	// 重複的 topic+email 應被 ON CONFLICT 靜默略過，不回傳新列也不報錯。
	again, err := repo.BatchCreateRecipients(ctx, items)
	require.NoError(t, err)
	require.Empty(t, again)
}

func newSimpleProtocolPool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:postgrespassword@localhost:5432/ltc_system?sslmode=disable"
	}
	poolCfg, err := pgxpool.ParseConfig(dbURL)
	require.NoError(t, err)
	poolCfg.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol

	ctx := context.Background()
	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	require.NoError(t, err)
	t.Cleanup(pool.Close)

	if err := pool.Ping(ctx); err != nil {
		t.Skipf("real Postgres not reachable at %s: %v", dbURL, err)
	}
	return pool
}
