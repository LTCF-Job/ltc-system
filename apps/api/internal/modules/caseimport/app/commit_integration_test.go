//go:build integration

package app_test

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/google/uuid"
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

	caseSvc := caseapp.NewCaseService(cfg, caseRepo, auditWriter{auditSvc}, caseinfra.NewExcelRenderer(), caseinfra.NewCaseDuplicateStagingRepository(pool))
	excel := importinfra.NewExcelAdapter()
	importSvc := importapp.NewImportService(
		failingCaseRegistrar{delegate: caseRegistrar{caseSvc}, failName: "受測個案B"},
		caseDuplicateFinder{caseSvc},
		caseDuplicateStager{caseSvc},
		siteAdapter{siteRepo},
		vehicleAdapter{vehicleRepo},
		caregiverAdapter{},
		caseRepo,
		excel,
		excel,
		txRunner,
	)

	site := masterapp.Site{
		Name:    "測試單位-" + uuid.NewString()[:8],
		Address: "測試地址",
		Status:  "active",
	}
	require.NoError(t, siteRepo.Create(ctx, &site))

	// 個案的申報區域已隨地區主檔一併移除，改以本測試自建的據點界定要清掉的資料範圍。
	t.Cleanup(func() {
		cleanupCtx := context.Background()
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM schedule_legs WHERE schedule_id IN (SELECT id FROM case_schedules WHERE case_id IN (SELECT id FROM cases WHERE site_id = $1))`, site.ID)
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM case_schedules WHERE case_id IN (SELECT id FROM cases WHERE site_id = $1)`, site.ID)
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM case_transport_preferences WHERE case_id IN (SELECT id FROM cases WHERE site_id = $1)`, site.ID)
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM audit_log WHERE entity_type = 'cases' AND entity_id IN (SELECT id::text FROM cases WHERE site_id = $1)`, site.ID)
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM cases WHERE site_id = $1`, site.ID)
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM sites WHERE id = $1`, site.ID)
	})

	// Row A：正常成功列。
	rowA := importapp.CaseImportRowResult{
		RowIndex:    1,
		Name:        "受測個案A",
		NationalID:  "A202559750",
		HomeAddress: "苗栗縣測試路1號",
		SiteName:    site.Name,
	}

	// Row B：故意讓 registrar 在個案主檔寫入成功之後失敗，用來驗證同一列的
	// 個案主檔會隨交易回滾，且批次不會因單列失敗而中止其餘列的處理。
	rowB := importapp.CaseImportRowResult{
		RowIndex:    2,
		Name:        "受測個案B",
		HomeAddress: "苗栗縣測試路2號",
		SiteName:    site.Name,
	}

	// Row C：緊接失敗列之後的正常列，用來驗證批次不會因單列失敗而提早中止。
	rowC := importapp.CaseImportRowResult{
		RowIndex:    3,
		Name:        "受測個案C",
		NationalID:  "G121806465",
		HomeAddress: "苗栗縣測試路3號",
		SiteName:    site.Name,
	}

	preview := &importapp.CaseImportPreviewResult{
		Rows: []importapp.CaseImportRowResult{rowA, rowB, rowC},
	}

	result, err := importSvc.CommitCases(ctx, preview, importapp.Actor{
		ActorID: uuid.New(), ActorRole: "admin", IPAddress: "127.0.0.1", UserAgent: "test-agent",
	})
	require.NoError(t, err)
	for _, sr := range result.SkippedRows {
		t.Logf("skipped row %d (%s): %v", sr.RowIndex, sr.CaseName, sr.Reasons)
	}
	for _, fr := range result.FailedRows {
		t.Logf("failed row %d (%s): %v", fr.RowIndex, fr.CaseName, fr.Reasons)
	}

	require.Equal(t, 2, result.ImportedCount, "Row A 與 Row C 應成功匯入")
	require.Len(t, result.FailedRows, 1, "Row B 應因列交易失敗而被記為失敗")
	require.Equal(t, 2, result.FailedRows[0].RowIndex)

	// 驗證 Row B 沒有殘留孤兒個案。
	orphans, orphanCount, err := caseRepo.List(ctx, "", rowB.Name, 1, 10, false, false)
	require.NoError(t, err)
	require.Zero(t, orphanCount, "Row B 的個案主檔必須未寫入")
	require.Empty(t, orphans, "Row B 的個案主檔必須未寫入")

	// 驗證 Row A／Row C 確實成功寫入。
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

func (a siteAdapter) GetByName(ctx context.Context, name string) (*importapp.SiteRef, error) {
	s, err := a.repo.GetByName(ctx, name)
	if err != nil {
		// 與 composition root 一致：查無單位屬於保留原始名稱待人工關聯，不是整列失敗。
		if errors.Is(err, masterapp.ErrSiteNotFound) {
			return nil, importapp.ErrLookupNotFound
		}
		return nil, err
	}
	if s == nil {
		return nil, nil
	}
	return &importapp.SiteRef{ID: s.ID, Name: s.Name}, nil
}

func (a siteAdapter) List(ctx context.Context, page, pageSize int) ([]importapp.SiteRef, error) {
	list, _, err := a.repo.List(ctx, "", "", "", page, pageSize)
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
		// 與 composition root 一致：查無車輛屬於保留原始名稱待人工關聯，不是整列失敗。
		if errors.Is(err, masterapp.ErrVehicleNotFound) {
			return nil, importapp.ErrLookupNotFound
		}
		return nil, err
	}
	if v == nil {
		return nil, nil
	}
	return &importapp.VehicleRef{ID: v.ID}, nil
}

// caregiverAdapter 在本測試中一律回傳查無資料：照護人員關聯不是這個測試的主題，
// 但仍需佔位讓 composition 與 cmd/server 一致。
type caregiverAdapter struct{}

func (caregiverAdapter) FindByName(ctx context.Context, name string) ([]importapp.CaregiverRef, error) {
	return nil, nil
}

// failingCaseRegistrar 在個案主檔寫入成功「之後」才回報失敗，模擬列交易中途出錯：
// 唯有交易確實回滾，該列才不會留下孤兒個案。
type failingCaseRegistrar struct {
	delegate importapp.CaseRegistrar
	failName string
}

func (w failingCaseRegistrar) CreateCase(ctx context.Context, in importapp.NewCase, actor importapp.Actor) (uuid.UUID, error) {
	id, err := w.delegate.CreateCase(ctx, in, actor)
	if err != nil {
		return uuid.Nil, err
	}
	if in.Name == w.failName {
		return uuid.Nil, errors.New("forced post-create failure")
	}
	return id, nil
}

func (w failingCaseRegistrar) RecordSkipped(ctx context.Context, row importapp.CaseImportSkippedRow, actor importapp.Actor) {
	w.delegate.RecordSkipped(ctx, row, actor)
}

type caseRegistrar struct{ svc *caseapp.CaseService }

func (a caseRegistrar) CreateCase(ctx context.Context, in importapp.NewCase, actor importapp.Actor) (uuid.UUID, error) {
	entity, err := a.svc.CreateCase(ctx, caseapp.CreateCaseRequest{
		ID:   in.ID,
		Name: in.Name, NationalID: in.NationalID,
		HouseholdType: in.HouseholdType, Gender: in.Gender, BirthDate: in.BirthDate,
		CareContactRole: in.CareContactRole, CareContactName: in.CareContactName,
		RegisteredAddress: in.RegisteredAddress, HomeAddress: in.HomeAddress,
		ServiceCategory:  intPointerOrNilForTest(in.ServiceCategory),
		ServiceUsageType: intPointerOrNilForTest(in.ServiceUsageType), Status: in.Status,
		SiteID: in.SiteID, SiteNameRaw: nullableStringPtrForTest(in.SiteNameRaw),
		CaregiverID: in.CaregiverID,
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

func nullableStringPtrForTest(v string) *string {
	if v == "" {
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
	return &importapp.DuplicateRef{CaseID: found.ID, CaseName: found.Name}, nil
}

type caseDuplicateStager struct{ svc *caseapp.CaseService }

func (a caseDuplicateStager) StageDuplicateRow(ctx context.Context, fileHash, rowKey string, in importapp.StageDuplicateCandidate) (uuid.UUID, bool, error) {
	return a.svc.StageDuplicateCandidate(ctx, caseapp.StageDuplicateCandidateInput{
		FileHash: fileHash, RowKey: rowKey,
		RowIndex: in.RowIndex, SheetName: in.SheetName,
		Name: in.Name, NationalID: in.NationalID,
		HouseholdType: in.HouseholdType, Gender: in.Gender, BirthDate: in.BirthDate, BirthDateRaw: in.BirthDateRaw,
		CareContactRole: in.CareContactRole, CareContactName: in.CareContactName,
		RegisteredAddress: in.RegisteredAddress, HomeAddress: in.HomeAddress,
		ServiceCategory:  intPointerOrNilForTest(in.ServiceCategory),
		ServiceUsageType: intPointerOrNilForTest(in.ServiceUsageType),
		Remarks:          in.Remarks,
		SiteID:           in.SiteID, SiteNameRaw: in.SiteNameRaw,
		CaregiverID:     in.CaregiverID,
		DuplicateCaseID: in.DuplicateCaseID,
	})
}
