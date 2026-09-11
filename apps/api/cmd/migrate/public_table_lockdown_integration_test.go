//go:build integration

package main

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

// TestPublicTableLockdown 驗證 000054 不只鎖住現有 table，也會攔住 future public object
// 的建立、移入 public、停用 RLS 與重新授權路徑。測試使用隨機名稱，避免污染既有資料。
func TestPublicTableLockdown(t *testing.T) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:postgrespassword@localhost:5432/ltc_system?sslmode=disable"
	}

	poolConfig, err := pgxpool.ParseConfig(dbURL)
	require.NoError(t, err)
	poolConfig.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol
	pool, err := pgxpool.NewWithConfig(context.Background(), poolConfig)
	require.NoError(t, err)
	t.Cleanup(pool.Close)

	ctx := context.Background()
	if err := pool.Ping(ctx); err != nil {
		t.Skipf("real Postgres not reachable at %s: %v", dbURL, err)
	}

	var triggerExists bool
	require.NoError(t, pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM pg_event_trigger WHERE evtname = 'enforce_public_table_lockdown'
		)
	`).Scan(&triggerExists))
	require.True(t, triggerExists, "000054 event trigger must be applied before this integration test")

	createdRoles := make([]string, 0, 2)
	for _, roleName := range []string{"anon", "authenticated"} {
		var exists bool
		require.NoError(t, pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = $1)`, roleName).Scan(&exists))
		if exists {
			continue
		}
		_, err := pool.Exec(ctx, fmt.Sprintf("CREATE ROLE %s NOLOGIN", pgx.Identifier{roleName}.Sanitize()))
		require.NoError(t, err)
		createdRoles = append(createdRoles, roleName)
	}
	t.Cleanup(func() {
		for _, roleName := range createdRoles {
			if _, err := pool.Exec(context.Background(), fmt.Sprintf("DROP ROLE IF EXISTS %s", pgx.Identifier{roleName}.Sanitize())); err != nil {
				t.Logf("drop test role %s: %v", roleName, err)
			}
		}
	})

	suffix := uuid.NewString()[:8]
	publicTableName := "review_lockdown_" + suffix
	movedTableName := "review_moved_" + suffix
	privateSchemaName := "review_private_" + suffix
	sequenceName := "review_sequence_" + suffix
	publicTable := pgx.Identifier{"public", publicTableName}.Sanitize()
	movedTable := pgx.Identifier{"public", movedTableName}.Sanitize()
	privateTable := pgx.Identifier{privateSchemaName, movedTableName}.Sanitize()
	privateSchema := pgx.Identifier{privateSchemaName}.Sanitize()
	sequence := pgx.Identifier{"public", sequenceName}.Sanitize()

	t.Cleanup(func() {
		cleanupCtx := context.Background()
		for _, statement := range []string{
			"DROP TABLE IF EXISTS " + publicTable + " CASCADE",
			"DROP TABLE IF EXISTS " + movedTable + " CASCADE",
			"DROP SEQUENCE IF EXISTS " + sequence + " CASCADE",
			"DROP SCHEMA IF EXISTS " + privateSchema + " CASCADE",
		} {
			if _, err := pool.Exec(cleanupCtx, statement); err != nil {
				t.Logf("cleanup %s: %v", statement, err)
			}
		}
	})

	// Future create：event trigger 應自動開啟 RLS。
	_, err = pool.Exec(ctx, "CREATE TABLE "+publicTable+" (id integer)")
	require.NoError(t, err)

	// 直接 grant bypass：event trigger 應在 grant 結束時立即撤掉權限。
	_, err = pool.Exec(ctx, "GRANT SELECT ON TABLE "+publicTable+" TO anon")
	require.NoError(t, err)
	var hasTableAccessAfterGrant bool
	require.NoError(t, pool.QueryRow(ctx, `SELECT has_table_privilege('anon', $1, 'SELECT')`, "public."+publicTableName).Scan(&hasTableAccessAfterGrant))
	require.False(t, hasTableAccessAfterGrant)

	// Disable bypass：DDL 結束時應重新開啟 RLS。
	_, err = pool.Exec(ctx, "ALTER TABLE "+publicTable+" DISABLE ROW LEVEL SECURITY")
	require.NoError(t, err)

	// Move-to-public bypass：private schema 建立的 table 移入 public 後仍須被鎖住。
	_, err = pool.Exec(ctx, "CREATE SCHEMA "+privateSchema+"; CREATE TABLE "+privateTable+" (id integer); ALTER TABLE "+privateTable+" SET SCHEMA public")
	require.NoError(t, err)

	// Future sequence 與 grant／ALTER sequence：不允許把 sequence usage 留給 API role。
	_, err = pool.Exec(ctx, "CREATE SEQUENCE "+sequence)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, "GRANT USAGE ON SEQUENCE "+sequence+" TO anon")
	require.NoError(t, err)
	var hasSequenceAccessAfterGrant bool
	require.NoError(t, pool.QueryRow(ctx, `SELECT has_sequence_privilege('anon', $1, 'USAGE')`, "public."+sequenceName).Scan(&hasSequenceAccessAfterGrant))
	require.False(t, hasSequenceAccessAfterGrant)
	_, err = pool.Exec(ctx, "ALTER SEQUENCE "+sequence+" SET LOGGED")
	require.NoError(t, err)

	var lockedTableCount int
	require.NoError(t, pool.QueryRow(ctx, `
		SELECT count(*)
		FROM pg_catalog.pg_class c
		JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace
		WHERE n.nspname = 'public'
		  AND c.relkind = 'r'
		  AND c.relrowsecurity
		  AND c.relname = ANY($1)
	`, []string{publicTableName, movedTableName}).Scan(&lockedTableCount))
	require.Equal(t, 2, lockedTableCount)

	var hasTableAccess, hasSequenceAccess bool
	require.NoError(t, pool.QueryRow(ctx, `SELECT has_table_privilege('anon', $1, 'SELECT')`, "public."+publicTableName).Scan(&hasTableAccess))
	require.NoError(t, pool.QueryRow(ctx, `SELECT has_sequence_privilege('anon', $1, 'USAGE')`, "public."+sequenceName).Scan(&hasSequenceAccess))
	require.False(t, hasTableAccess)
	require.False(t, hasSequenceAccess)
}
