package infra

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"ltc-system/apps/api/internal/modules/reporting/app"
)

// ExportJobRepository 保存匯出工作、逐案檔案中繼資料與申報列快照。
// 三張表同屬一次匯出的原子單位，因此完成寫入在本 repository 內自行開啟事務。
type ExportJobRepository struct {
	db      *pgxpool.Pool
	storage app.ObjectStorage
}

// NewExportJobRepository 建立 ExportJobRepository 實例。
func NewExportJobRepository(db *pgxpool.Pool, storages ...app.ObjectStorage) *ExportJobRepository {
	var storage app.ObjectStorage
	if len(storages) > 0 {
		storage = storages[0]
	}
	return &ExportJobRepository{db: db, storage: storage}
}

// CreateJob 以 running 狀態建立匯出工作，回傳工作編號。
func (r *ExportJobRepository) CreateJob(ctx context.Context, job app.ExportJobCreate) (uuid.UUID, error) {
	if r.db == nil {
		return uuid.Nil, errNoDatabase
	}

	precheckJSON := "null"
	if job.Precheck != nil {
		encoded, err := json.Marshal(job.Precheck)
		if err != nil {
			return uuid.Nil, fmt.Errorf("marshal precheck: %w", err)
		}
		precheckJSON = string(encoded)
	}

	caseIDs := job.CaseIDs
	if caseIDs == nil {
		caseIDs = []uuid.UUID{}
	}

	var id uuid.UUID
	err := r.db.QueryRow(ctx, `
		INSERT INTO export_jobs (job_type, period_ym, region, format, filter_case_ids, status, precheck, created_by, created_by_name)
		VALUES ($1, $2, $3, $4, $5::uuid[], 'running', $6::jsonb, $7, $8)
		RETURNING id
	`, job.JobType, job.PeriodYM, job.Region, job.Format, caseIDs, precheckJSON, job.CreatedBy, job.CreatedByName).Scan(&id)
	if err != nil {
		return uuid.Nil, fmt.Errorf("insert export job: %w", err)
	}
	return id, nil
}

// CompleteJob 在單一事務內寫入申報列快照與逐案檔案，並把工作標記為成功。
func (r *ExportJobRepository) CompleteJob(ctx context.Context, jobID uuid.UUID, files []app.GovClaimCaseFile, lines []app.ExportLine) (err error) {
	if r.db == nil {
		return errNoDatabase
	}
	uploadedPaths := make([]string, 0, len(files))
	// Storage upload 與 DB transaction 無法跨系統原子化；DB 失敗時清除本次已上傳物件，
	// 避免 private bucket 留下無法從歷史工作索引到的孤兒檔案。
	defer func() {
		if err == nil || r.storage == nil {
			return
		}
		for _, path := range uploadedPaths {
			if cleanupErr := r.storage.Delete(ctx, path); cleanupErr != nil {
				slog.Warn("failed to clean up export object", slog.String("error_type", fmt.Sprintf("%T", cleanupErr)))
			}
		}
	}()

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin export job tx: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
			return
		}
		err = tx.Commit(ctx)
	}()

	batch := &pgx.Batch{}
	for _, line := range lines {
		payload, marshalErr := json.Marshal(line.Payload)
		if marshalErr != nil {
			return fmt.Errorf("marshal export line payload: %w", marshalErr)
		}
		batch.Queue(`
			INSERT INTO export_lines (job_id, line_no, case_id, national_id_masked, service_date_roc, raw_payload)
			VALUES ($1, $2, $3, $4, $5, $6::jsonb)
		`, jobID, line.LineNo, line.CaseID, line.NationalIDMasked, line.ServiceDateROC, string(payload))
	}
	for seq, file := range files {
		storagePath := exportStoragePath(jobID, file.FileName)
		fileContent := any(file.Bytes)
		if r.storage != nil {
			if err := r.storage.Put(ctx, storagePath, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", file.Bytes); err != nil {
				return fmt.Errorf("upload immutable export file: %w", err)
			}
			uploadedPaths = append(uploadedPaths, storagePath)
			fileContent = nil
		}
		batch.Queue(`
			INSERT INTO export_job_files (job_id, case_id, seq, case_name, region, file_name, row_count, file_checksum, storage_path, file_content, file_size)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		`, jobID, file.CaseID, seq+1, file.CaseName, file.Region, file.FileName, file.RowCount, file.Checksum,
			storagePath, fileContent, len(file.Bytes))
	}
	batch.Queue(`UPDATE export_jobs SET status = 'succeeded', finished_at = now() WHERE id = $1`, jobID)

	results := tx.SendBatch(ctx, batch)
	for i := 0; i < batch.Len(); i++ {
		if _, execErr := results.Exec(); execErr != nil {
			_ = results.Close()
			return fmt.Errorf("write export job content: %w", execErr)
		}
	}
	if closeErr := results.Close(); closeErr != nil {
		return fmt.Errorf("close export job batch: %w", closeErr)
	}
	return nil
}

// FailJob 將工作標記為失敗並記錄可直接呈現給使用者的訊息。
func (r *ExportJobRepository) FailJob(ctx context.Context, jobID uuid.UUID, message string) error {
	if r.db == nil {
		return errNoDatabase
	}
	_, err := r.db.Exec(ctx, `
		UPDATE export_jobs SET status = 'failed', error_message = $2, finished_at = now() WHERE id = $1
	`, jobID, message)
	if err != nil {
		return fmt.Errorf("mark export job failed: %w", err)
	}
	return nil
}

// GetJob 取得單筆匯出工作與其逐案檔案清單。
func (r *ExportJobRepository) GetJob(ctx context.Context, jobID uuid.UUID) (app.GovClaimJob, error) {
	if r.db == nil {
		return app.GovClaimJob{}, errNoDatabase
	}

	var row exportJobRow
	err := r.db.QueryRow(ctx, `
		SELECT id, job_type, period_ym, region, format, status, error_message, created_by, created_by_name, created_at, finished_at
		FROM export_jobs WHERE id = $1
	`, jobID).Scan(
		&row.ID, &row.JobType, &row.PeriodYM, &row.Region, &row.Format,
		&row.Status, &row.ErrorMessage, &row.CreatedBy, &row.CreatedByName, &row.CreatedAt, &row.FinishedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return app.GovClaimJob{}, app.ErrExportJobNotFound
	}
	if err != nil {
		return app.GovClaimJob{}, fmt.Errorf("query export job: %w", err)
	}

	files, err := r.listJobFiles(ctx, jobID)
	if err != nil {
		return app.GovClaimJob{}, err
	}

	job := row.toApp()
	job.Files = files
	job.TotalCases = len(files)
	for _, f := range files {
		job.TotalRows += f.RowCount
	}
	return job, nil
}

// ListJobs 依建立時間新到舊分頁列出匯出工作，不含逐案檔案明細。
func (r *ExportJobRepository) ListJobs(ctx context.Context, page, pageSize int) ([]app.GovClaimJob, int64, error) {
	if r.db == nil {
		return nil, 0, errNoDatabase
	}

	var total int64
	if err := r.db.QueryRow(ctx, `SELECT count(*) FROM export_jobs`).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count export jobs: %w", err)
	}

	rows, err := r.db.Query(ctx, `
		SELECT j.id, j.job_type, j.period_ym, j.region, j.format, j.status, j.error_message,
		       j.created_by, j.created_by_name, j.created_at, j.finished_at,
		       COALESCE(f.file_count, 0)::int, COALESCE(f.row_total, 0)::int
		FROM export_jobs j
		LEFT JOIN (
			SELECT job_id, count(*) AS file_count, sum(row_count) AS row_total
			FROM export_job_files GROUP BY job_id
		) f ON f.job_id = j.id
		ORDER BY j.created_at DESC
		LIMIT $1 OFFSET $2
	`, pageSize, (page-1)*pageSize)
	if err != nil {
		return nil, 0, fmt.Errorf("query export jobs: %w", err)
	}
	defer rows.Close()

	result := make([]app.GovClaimJob, 0)
	for rows.Next() {
		var row exportJobRow
		if err := rows.Scan(
			&row.ID, &row.JobType, &row.PeriodYM, &row.Region, &row.Format,
			&row.Status, &row.ErrorMessage, &row.CreatedBy, &row.CreatedByName, &row.CreatedAt, &row.FinishedAt,
			&row.TotalCases, &row.TotalRows,
		); err != nil {
			return nil, 0, fmt.Errorf("scan export job: %w", err)
		}
		result = append(result, row.toApp())
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate export jobs: %w", err)
	}
	return result, total, nil
}

// LoadCaseLines 依 line_no 順序讀回單一個案的申報列快照，重繪時據此還原原始列序。
func (r *ExportJobRepository) LoadCaseLines(ctx context.Context, jobID, caseID uuid.UUID) ([]app.ExportLine, error) {
	if r.db == nil {
		return nil, errNoDatabase
	}

	rows, err := r.db.Query(ctx, `
		SELECT line_no, case_id, national_id_masked, service_date_roc, raw_payload
		FROM export_lines WHERE job_id = $1 AND case_id = $2 ORDER BY line_no
	`, jobID, caseID)
	if err != nil {
		return nil, fmt.Errorf("query export lines: %w", err)
	}
	defer rows.Close()

	result := make([]app.ExportLine, 0)
	for rows.Next() {
		var row exportLineRow
		if err := rows.Scan(&row.LineNo, &row.CaseID, &row.NationalIDMasked, &row.ServiceDateROC, &row.RawPayload); err != nil {
			return nil, fmt.Errorf("scan export line: %w", err)
		}
		line, err := row.toApp()
		if err != nil {
			return nil, err
		}
		result = append(result, line)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate export lines: %w", err)
	}
	return result, nil
}

// LoadExportFile 讀取匯出成功時保存的原始檔案，不重新依目前主檔產生。
func (r *ExportJobRepository) LoadExportFile(ctx context.Context, jobID, caseID uuid.UUID) ([]byte, error) {
	if r.db == nil {
		return nil, errNoDatabase
	}
	var storagePath string
	var content []byte
	err := r.db.QueryRow(ctx, `
		SELECT COALESCE(storage_path, ''), file_content
		FROM export_job_files
		WHERE job_id = $1 AND case_id = $2
	`, jobID, caseID).Scan(&storagePath, &content)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, app.ErrExportFileNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("query immutable export file: %w", err)
	}
	if r.storage != nil && storagePath != "" {
		stored, err := r.storage.Get(ctx, storagePath)
		if err != nil {
			return nil, fmt.Errorf("download immutable export file: %w", err)
		}
		return stored, nil
	}
	return content, nil
}

// LoadNationalIDCiphers 取回重繪快照時補回身分證欄位所需的密文。
func (r *ExportJobRepository) LoadNationalIDCiphers(ctx context.Context, caseID uuid.UUID, driverIDs []uuid.UUID) (app.NationalIDCiphers, error) {
	result := app.NationalIDCiphers{Drivers: make(map[uuid.UUID][]byte, len(driverIDs))}
	if r.db == nil {
		return app.NationalIDCiphers{}, errNoDatabase
	}

	if err := r.db.QueryRow(ctx, `SELECT national_id_cipher FROM cases WHERE id = $1`, caseID).Scan(&result.Case); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return result, app.ErrExportFileNotFound
		}
		return result, fmt.Errorf("query case national id cipher: %w", err)
	}

	if len(driverIDs) == 0 {
		return result, nil
	}

	rows, err := r.db.Query(ctx, `SELECT id, national_id_cipher FROM drivers WHERE id = ANY($1::uuid[])`, driverIDs)
	if err != nil {
		return result, fmt.Errorf("query driver national id ciphers: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var id uuid.UUID
		var cipher []byte
		if err := rows.Scan(&id, &cipher); err != nil {
			return result, fmt.Errorf("scan driver national id cipher: %w", err)
		}
		result.Drivers[id] = cipher
	}
	if err := rows.Err(); err != nil {
		return result, fmt.Errorf("iterate driver national id ciphers: %w", err)
	}
	return result, nil
}

func (r *ExportJobRepository) listJobFiles(ctx context.Context, jobID uuid.UUID) ([]app.GovClaimCaseFile, error) {
	rows, err := r.db.Query(ctx, `
		SELECT case_id, case_name, region, file_name, row_count, COALESCE(file_checksum, '')
		FROM export_job_files WHERE job_id = $1 ORDER BY seq
	`, jobID)
	if err != nil {
		return nil, fmt.Errorf("query export job files: %w", err)
	}
	defer rows.Close()

	result := make([]app.GovClaimCaseFile, 0)
	for rows.Next() {
		var file app.GovClaimCaseFile
		if err := rows.Scan(
			&file.CaseID, &file.CaseName, &file.Region,
			&file.FileName, &file.RowCount, &file.Checksum,
		); err != nil {
			return nil, fmt.Errorf("scan export job file: %w", err)
		}
		result = append(result, file)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate export job files: %w", err)
	}
	return result, nil
}

var errNoDatabase = errors.New("database connection is not configured")

func exportStoragePath(jobID uuid.UUID, fileName string) string {
	return fmt.Sprintf("exports/%s/%s", jobID, fileName)
}
