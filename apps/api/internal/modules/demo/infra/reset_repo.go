package infra

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/jackc/pgx/v5/pgxpool"
	"ltc-system/apps/api/internal/domain/crypto"
	"ltc-system/apps/api/internal/platform/config"
	"ltc-system/apps/api/internal/platform/pgxdb"
)

// truncateBusinessTablesSQL 涵蓋所有業務資料表並靠 CASCADE 處理 FK 依賴順序；regions 是 migrations/000002 維護的靜態參考資料，不隨 Demo 重置清空。
const truncateBusinessTablesSQL = `TRUNCATE TABLE
	sites, vehicles, drivers, driver_assignments,
	cases, case_schedules, case_transport_preferences, schedule_legs,
	caregivers, driver_report_forms, form_columns, form_submissions,
	ride_sources, ride_records,
	export_jobs, export_lines, export_job_files,
	audit_log, notification_recipients, notification_log,
	holidays, attendance_records, maintenance_logs, fuel_logs, app_settings
RESTART IDENTITY CASCADE`

// ResetRepository 在單一交易中清空 Demo 業務資料表並套用版本化種子資料。
type ResetRepository struct {
	pool    *pgxpool.Pool
	tx      *pgxdb.TxRunner
	seedSQL string
	cfg     *config.Config
}

// NewResetRepository 讀取版本化的 Demo 種子檔並建立 ResetRepository；找不到檔案時直接回傳 error，交由呼叫端決定是否啟動。
func NewResetRepository(pool *pgxpool.Pool, cfg *config.Config) (*ResetRepository, error) {
	seedSQL, err := loadSeedFile("0001_baseline.up.sql")
	if err != nil {
		return nil, err
	}
	return &ResetRepository{pool: pool, tx: pgxdb.NewTxRunner(pool), seedSQL: seedSQL, cfg: cfg}, nil
}

// loadSeedFile 依序嘗試容器內路徑、本機從 repo 根目錄執行的路徑，找不到則退回相對路徑。
func loadSeedFile(name string) (string, error) {
	_, thisFile, _, _ := runtime.Caller(0)
	candidates := []string{
		filepath.Join("seed", "demo"),
		filepath.Join("apps", "api", "seed", "demo"),
		// go test 的 cwd 是套件目錄，前兩個候選都對不到，需靠這個相對於本檔案的路徑才能在測試裡讀到種子檔。
		filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "..", "seed", "demo"),
	}

	for _, dir := range candidates {
		path := filepath.Join(dir, name)
		if data, err := os.ReadFile(path); err == nil {
			return string(data), nil
		}
	}
	return "", fmt.Errorf("read demo seed file %s: not found in any of %v", name, candidates)
}

// Reset 清空業務資料表並重新套用種子資料；任何一步失敗都會讓交易整筆回滾。
func (r *ResetRepository) Reset(ctx context.Context) error {
	return r.tx.WithTx(ctx, func(ctx context.Context) error {
		q := pgxdb.FromContext(ctx, r.pool)
		if _, err := q.Exec(ctx, truncateBusinessTablesSQL); err != nil {
			return fmt.Errorf("truncate demo business tables: %w", err)
		}
		if _, err := q.Exec(ctx, r.seedSQL); err != nil {
			return fmt.Errorf("apply demo seed dataset: %w", err)
		}
		if err := r.reencryptNationalIDs(ctx, q); err != nil {
			return fmt.Errorf("re-encrypt demo national IDs: %w", err)
		}
		return nil
	})
}

// reencryptNationalIDs 用本服務目前的加密金鑰，把種子資料裡的佔位密文換成可正常解密與遮罩顯示的假身分證字號。
func (r *ResetRepository) reencryptNationalIDs(ctx context.Context, q pgxdb.Querier) error {
	type target struct {
		id    string
		table string
	}
	var targets []target

	driverRows, err := q.Query(ctx, `SELECT id FROM drivers ORDER BY code`)
	if err != nil {
		return fmt.Errorf("list demo drivers: %w", err)
	}
	for driverRows.Next() {
		var id string
		if err := driverRows.Scan(&id); err != nil {
			driverRows.Close()
			return err
		}
		targets = append(targets, target{id: id, table: "drivers"})
	}
	driverRows.Close()
	if err := driverRows.Err(); err != nil {
		return err
	}

	caseRows, err := q.Query(ctx, `SELECT id FROM cases WHERE national_id_masked IS NOT NULL ORDER BY code`)
	if err != nil {
		return fmt.Errorf("list demo cases: %w", err)
	}
	for caseRows.Next() {
		var id string
		if err := caseRows.Scan(&id); err != nil {
			caseRows.Close()
			return err
		}
		targets = append(targets, target{id: id, table: "cases"})
	}
	caseRows.Close()
	if err := caseRows.Err(); err != nil {
		return err
	}

	for seq, t := range targets {
		fakeID := fakeNationalID(seq)
		cipher, err := crypto.Encrypt(fakeID, r.cfg.EncryptionKey)
		if err != nil {
			return fmt.Errorf("encrypt fake national id: %w", err)
		}
		hmacIdx := crypto.Index(fakeID, r.cfg.HMACKey)
		masked := crypto.Mask(fakeID)

		updateSQL := fmt.Sprintf(`UPDATE %s SET national_id_cipher = $1, national_id_hmac = $2, national_id_masked = $3 WHERE id = $4`, t.table)
		if _, err := q.Exec(ctx, updateSQL, cipher, hmacIdx, masked, t.id); err != nil {
			return fmt.Errorf("update %s national id: %w", t.table, err)
		}
	}
	return nil
}

// fakeNationalID 依序號產生檢查碼正確、以 "A1" 開頭的假身分證字號，僅供 Demo 加密/遮罩示範使用。
func fakeNationalID(seq int) string {
	digits := fmt.Sprintf("1%07d", seq%10000000)
	sum := 1 // 對應 letterCodeMap['A']=10 => n1=1, n2=0, sum = n1 + n2*9
	weights := [8]int{8, 7, 6, 5, 4, 3, 2, 1}
	for i, w := range weights {
		sum += int(digits[i]-'0') * w
	}
	check := (10 - sum%10) % 10
	return fmt.Sprintf("A%s%d", digits, check)
}
