//go:build integration

package app_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	"ltc-system/apps/api/internal/domain/crypto"
	auditapp "ltc-system/apps/api/internal/modules/audit/app"
	auditinfra "ltc-system/apps/api/internal/modules/audit/infra"
	importapp "ltc-system/apps/api/internal/modules/caseimport/app"
	importinfra "ltc-system/apps/api/internal/modules/caseimport/infra"
	caseapp "ltc-system/apps/api/internal/modules/casemgmt/app"
	caseinfra "ltc-system/apps/api/internal/modules/casemgmt/infra"
	masterapp "ltc-system/apps/api/internal/modules/masterdata/app"
	masterinfra "ltc-system/apps/api/internal/modules/masterdata/infra"
	"ltc-system/apps/api/internal/platform/config"
	"ltc-system/apps/api/internal/platform/pgxdb"
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

	caseRepo := caseinfra.NewCaseRepository(pool)
	siteRepo := masterinfra.NewSiteRepository(pool)
	vehicleRepo := masterinfra.NewVehicleRepository(pool)
	auditSvc := auditapp.NewService(auditinfra.NewAuditRepository(pool))
	txRunner := pgxdb.NewTxRunner(pool)

	caseSvc := caseapp.NewCaseService(cfg, caseRepo, siteAdapter{siteRepo}, auditWriter{auditSvc}, caseinfra.NewExcelRenderer())
	excel := importinfra.NewExcelAdapter()
	importSvc := importapp.NewImportService(
		caseRegistrar{caseSvc},
		caseDuplicateFinder{caseSvc},
		siteAdapter{siteRepo},
		vehicleAdapter{vehicleRepo},
		caseRepo,
		excel,
		excel,
		txRunner,
	)

	region := "hsinchu-" + uuid.NewString()[:8]
	site := masterapp.Site{
		Code:     "T-" + uuid.NewString()[:8],
		Name:     "測試單位-" + uuid.NewString()[:8],
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


	// Row A：正常成功列。
	rowA := importapp.CaseImportRowResult{
		RowIndex:       1,
		Name:           "受測個案A",
		NationalID:     "A202559750",
		HomeAddress:    "苗栗縣測試路1號",
		Region:         region,
		SiteName:       site.Name,
	}

	// Row B：身分證字號檢查碼故意錯誤，觸發 CreateCase 失敗，用來驗證批次不會因
	// 單列失敗而中止其餘列的處理。
	rowB := importapp.CaseImportRowResult{
		RowIndex:       2,
		Name:           "受測個案B",
		NationalID:     "A100000000",
		HomeAddress:    "苗栗縣測試路2號",
		Region:         region,
		SiteName:       site.Name,
	}

	// Row C：緊接失敗列之後的正常列，用來驗證批次不會因單列失敗而提早中止。
	rowC := importapp.CaseImportRowResult{
		RowIndex:       3,
		Name:           "受測個案C",
		NationalID:     "G121806465",
		HomeAddress:    "苗栗縣測試路3號",
		Region:         region,
		SiteName:       site.Name,
	}

	preview := &importapp.CaseImportPreviewResult{
		Rows: []importapp.CaseImportRowResult{rowA, rowB, rowC},
	}

	result, err := importSvc.CommitCases(ctx, preview, nil, importapp.Actor{
		ActorID: uuid.New(), ActorRole: "admin", IPAddress: "127.0.0.1", UserAgent: "test-agent",
	})
	require.NoError(t, err)
	for _, sr := range result.SkippedRows {
		t.Logf("skipped row %d (%s): %v", sr.RowIndex, sr.CaseName, sr.Reasons)
	}

	require.Equal(t, 2, result.ImportedCount, "Row A 與 Row C 應成功匯入")
	require.Len(t, result.SkippedRows, 1, "Row B 應因身分證字號檢查碼錯誤而被記為略過")
	require.Equal(t, 2, result.SkippedRows[0].RowIndex)

	// 驗證 Row B 沒有殘留孤兒個案。
	hmacIdx := crypto.Index(rowB.NationalID, cfg.HMACKey)
	orphan, err := caseRepo.GetByHMAC(ctx, hmacIdx)
	require.ErrorIs(t, err, pgx.ErrNoRows, "Row B 的個案主檔必須未寫入")
	require.Nil(t, orphan)

	// 驗證 Row A／Row C 確實成功落地。
	for _, nid := range []string{rowA.NationalID, rowC.NationalID} {
		hmacIdx := crypto.Index(nid, cfg.HMACKey)
		created, err := caseRepo.GetByHMAC(ctx, hmacIdx)
		require.NoError(t, err)
		require.NotNil(t, created)
	}
}

// 以下 adapter 與 cmd/server 的 composition root 等價，讓整合測試能在不匯入
// package main 的情況下把 caseimport 接到 casemgmt、masterdata 與 audit。

type auditWriter struct{ svc *auditapp.Service }

func (w auditWriter) Write(ctx context.Context, e caseapp.AuditEntry) error {
	return w.svc.Write(ctx, auditapp.Entry{
		ActorID: e.ActorID, ActorRole: e.ActorRole, Action: e.Action, EntityType: e.EntityType,
		EntityID: e.EntityID, BeforeData: e.BeforeData, AfterData: e.AfterData,
		IPAddress: e.IPAddress, UserAgent: e.UserAgent,
	})
}

type siteAdapter struct{ repo *masterinfra.SiteRepository }

func (a siteAdapter) GetByID(ctx context.Context, id uuid.UUID) (*caseapp.SiteRef, error) {
	s, err := a.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return &caseapp.SiteRef{ID: s.ID, Region: s.Region}, nil
}

func (a siteAdapter) GetByName(ctx context.Context, name string) (*importapp.SiteRef, error) {
	s, err := a.repo.GetByName(ctx, name)
	if err != nil {
		return nil, err
	}
	return &importapp.SiteRef{ID: s.ID, Name: s.Name}, nil
}

func (a siteAdapter) List(ctx context.Context, region string, page, pageSize int) ([]importapp.SiteRef, error) {
	list, _, err := a.repo.List(ctx, region, "", "", page, pageSize)
	if err != nil {
		return nil, err
	}
	out := make([]importapp.SiteRef, 0, len(list))
	for _, s := range list {
		out = append(out, importapp.SiteRef{ID: s.ID, Name: s.Name})
	}
	return out, nil
}

type vehicleAdapter struct {
	repo *masterinfra.VehicleRepository
}

func (a vehicleAdapter) GetByDisplayName(ctx context.Context, displayName string) (*importapp.VehicleRef, error) {
	v, err := a.repo.GetByDisplayName(ctx, displayName)
	if err != nil {
		return nil, err
	}
	return &importapp.VehicleRef{ID: v.ID}, nil
}

type caseRegistrar struct{ svc *caseapp.CaseService }

func (a caseRegistrar) CreateCase(ctx context.Context, in importapp.NewCase, actor importapp.Actor) (uuid.UUID, error) {
	entity, err := a.svc.CreateCase(ctx, caseapp.CreateCaseRequest{
		Code: in.Code, Name: in.Name, NationalID: in.NationalID,
		HouseholdType: in.HouseholdType, Gender: in.Gender, BirthDate: in.BirthDate,
		CareContactRole: in.CareContactRole, CareContactName: in.CareContactName,
		RegisteredAddress: in.RegisteredAddress, HomeAddress: in.HomeAddress, Region: in.Region,
		ServiceCategory: intPointerOrNilForTest(in.ServiceCategory),
		ServiceUsageType: intPointerOrNilForTest(in.ServiceUsageType), Status: in.Status,
	}, actor.ActorID, actor.ActorRole, actor.IPAddress, actor.UserAgent)
	if err != nil {
		return uuid.Nil, err
	}
	return entity.ID, nil
}

func intPointerOrNilForTest(v int) *int {
	if v == 0 {
		return nil
	}
	return &v
}

func (a caseRegistrar) RecordSkipped(ctx context.Context, row importapp.CaseImportSkippedRow, actor importapp.Actor) {
	a.svc.RecordSkippedCaseImport(ctx, caseapp.CaseImportSkippedRow{
		RowIndex: row.RowIndex, CaseName: row.CaseName, Reasons: row.Reasons, RawValues: row.RawValues,
	}, actor.ActorID, actor.ActorRole, actor.IPAddress, actor.UserAgent)
}

type caseDuplicateFinder struct{ svc *caseapp.CaseService }

func (a caseDuplicateFinder) FindDuplicate(ctx context.Context, nationalID, name string) (*importapp.DuplicateRef, error) {
	found, err := a.svc.FindPossibleDuplicate(ctx, nationalID, name)
	if err != nil || found == nil {
		return nil, err
	}
	return &importapp.DuplicateRef{CaseID: found.ID, CaseCode: found.Code}, nil
}
