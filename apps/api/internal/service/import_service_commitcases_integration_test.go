//go:build integration

package service

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	"ltc-system/apps/api/internal/config"
	"ltc-system/apps/api/internal/domain/crypto"
	"ltc-system/apps/api/internal/platform/pgxdb"
	"ltc-system/apps/api/internal/repository"
)

// TestCommitCases_TransactionRollback 針對真實 Postgres 驗證 CommitCases 的逐列事務語意：
// 一列中途失敗時，該列已寫入的個案主檔會被回滾（不留孤兒資料），
// 但不影響同一批次中已成功或後續待處理的其他列。
//
// 執行方式（需要本機 docker-compose.local.yml 啟動的 Postgres）：
//
//	docker compose -f docker-compose.local.yml up -d postgres
//	DATABASE_URL=postgres://postgres:postgrespassword@localhost:5432/ltc_system?sslmode=disable \
//	  go test -tags=integration ./internal/service/... -run TestCommitCases_TransactionRollback -v
func TestCommitCases_TransactionRollback(t *testing.T) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:postgrespassword@localhost:5432/ltc_system?sslmode=disable"
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbURL)
	require.NoError(t, err)
	t.Cleanup(pool.Close) // 註冊順序須早於資料清理，Cleanup 以 LIFO 執行，確保清理查詢先於連線關閉。

	if err := pool.Ping(ctx); err != nil {
		t.Skipf("real Postgres not reachable at %s: %v", dbURL, err)
	}

	t.Setenv("APP_ENV", "local")
	t.Setenv("ENCRYPTION_KEY", "MDEwMjAzMDQwNTA2MDcwODAxMDIwMzA0MDUwNjA3MDg=")
	t.Setenv("HMAC_KEY", "MDkwODAwMDcwNjA1MDQwMzA5MDgwMDA3MDYwNTA0MDM=")
	cfg, err := config.LoadFromEnv()
	require.NoError(t, err)

	caseRepo := repository.NewCaseRepository(pool)
	siteRepo := repository.NewSiteRepository(pool)
	vehicleRepo := repository.NewVehicleRepository(pool)
	driverRepo := repository.NewDriverRepository(pool)
	auditRepo := repository.NewAuditRepository(pool)
	txRunner := pgxdb.NewTxRunner(pool)

	masterSvc := NewMasterService(cfg, caseRepo, siteRepo, vehicleRepo, driverRepo, auditRepo)
	importSvc := NewImportService(masterSvc, siteRepo, vehicleRepo, driverRepo, caseRepo, txRunner)

	region := "hsinchu-" + uuid.NewString()[:8]
	site := repository.SiteEntity{
		Code:     "T-" + uuid.NewString()[:8],
		Name:     "測試據點-" + uuid.NewString()[:8],
		Address:  "測試地址",
		Region:   region,
		OpenDays: []int16{1, 2, 3, 4, 5},
		Status:   "active",
	}
	require.NoError(t, siteRepo.Create(ctx, &site))

	t.Cleanup(func() {
		cleanupCtx := context.Background()
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM schedule_legs WHERE schedule_id IN (SELECT id FROM case_schedules WHERE site_id = $1)`, site.ID)
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM case_schedules WHERE site_id = $1`, site.ID)
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM case_transport_preferences WHERE case_id IN (SELECT id FROM cases WHERE region = $1)`, region)
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM audit_log WHERE entity_type = 'cases' AND entity_id IN (SELECT id::text FROM cases WHERE region = $1)`, region)
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM cases WHERE region = $1`, region)
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM sites WHERE id = $1`, site.ID)
	})

	claimStart := time.Now().Format("2006-01-02")

	// Row A：正常成功列。
	rowA := CaseImportRowResult{
		RowIndex:           1,
		Name:               "受測個案A",
		NationalID:         "A202559750",
		HomeAddress:        "苗栗縣測試路1號",
		Region:             region,
		ClaimStartDate:     claimStart,
		SiteName:           site.Name,
		Weekdays:           []int16{1, 2, 3},
		OutboundTime:       "09:00",
		InboundTime:        "16:00",
		TripPattern:        2,
		DistanceKM:         5,
		UnitPrice:          115,
		ServiceDurationMin: 10,
	}

	// Row B：個案主檔會先成功寫入，但排班設定因去回程時間未遞增而失敗（ErrLegTimesNotOrdered），
	// 用來驗證同一列內兩個 repository 寫入是否共用同一事務並正確回滾。
	rowB := CaseImportRowResult{
		RowIndex:           2,
		Name:               "受測個案B",
		NationalID:         "K120098177",
		HomeAddress:        "苗栗縣測試路2號",
		Region:             region,
		ClaimStartDate:     claimStart,
		SiteName:           site.Name,
		Weekdays:           []int16{1, 2, 3},
		OutboundTime:       "16:00", // 刻意早於回程時間，觸發 ErrLegTimesNotOrdered
		InboundTime:        "09:00",
		TripPattern:        2,
		DistanceKM:         5,
		UnitPrice:          115,
		ServiceDurationMin: 10,
	}

	// Row C：緊接失敗列之後的正常列，用來驗證批次不會因單列失敗而提早中止。
	rowC := CaseImportRowResult{
		RowIndex:           3,
		Name:               "受測個案C",
		NationalID:         "G121806465",
		HomeAddress:        "苗栗縣測試路3號",
		Region:             region,
		ClaimStartDate:     claimStart,
		SiteName:           site.Name,
		Weekdays:           []int16{1, 2, 3},
		OutboundTime:       "09:00",
		InboundTime:        "16:00",
		TripPattern:        2,
		DistanceKM:         5,
		UnitPrice:          115,
		ServiceDurationMin: 10,
	}

	preview := &CaseImportPreviewResult{
		Rows: []CaseImportRowResult{rowA, rowB, rowC},
	}

	result, err := importSvc.CommitCases(ctx, preview, uuid.New(), "admin", "127.0.0.1", "test-agent")
	require.NoError(t, err)
	for _, sr := range result.SkippedRows {
		t.Logf("skipped row %d (%s): %v", sr.RowIndex, sr.CaseName, sr.Reasons)
	}

	require.Equal(t, 2, result.ImportedCount, "Row A 與 Row C 應成功匯入")
	require.Len(t, result.SkippedRows, 1, "Row B 應因排班設定失敗而被記為略過")
	require.Equal(t, 2, result.SkippedRows[0].RowIndex)

	// 驗證 Row B 沒有殘留孤兒個案：CreateCaseSchedule 失敗必須回滾同一列的 CreateCase。
	hmacIdx := crypto.Index(rowB.NationalID, cfg.HMACKey)
	orphan, err := caseRepo.GetByHMAC(ctx, hmacIdx)
	require.ErrorIs(t, err, pgx.ErrNoRows, "Row B 的個案主檔必須已隨事務回滾，不應留下孤兒資料")
	require.Nil(t, orphan)

	// 驗證 Row A／Row C 確實成功落地。
	for _, nid := range []string{rowA.NationalID, rowC.NationalID} {
		hmacIdx := crypto.Index(nid, cfg.HMACKey)
		created, err := caseRepo.GetByHMAC(ctx, hmacIdx)
		require.NoError(t, err)
		require.NotNil(t, created)
	}
}
