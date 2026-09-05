package app

import (
	"context"

	"github.com/google/uuid"
)

// GovClaimSourceReader 查詢指定期間、地區與個案範圍內可申報的趟次原始資料。
type GovClaimSourceReader interface {
	QueryGovClaimSources(ctx context.Context, scope ClaimScope) ([]GovClaimSource, error)
}

// ExportJobStore 保存匯出工作、逐案檔案中繼資料與申報列快照。
type ExportJobStore interface {
	CreateJob(ctx context.Context, job ExportJobCreate) (uuid.UUID, error)
	CompleteJob(ctx context.Context, jobID uuid.UUID, files []GovClaimCaseFile, lines []ExportLine) error
	FailJob(ctx context.Context, jobID uuid.UUID, message string) error
	GetJob(ctx context.Context, jobID uuid.UUID) (GovClaimJob, error)
	ListJobs(ctx context.Context, page, pageSize int) ([]GovClaimJob, int64, error)
	LoadCaseLines(ctx context.Context, jobID, caseID uuid.UUID) ([]ExportLine, error)
	LoadNationalIDCiphers(ctx context.Context, caseID uuid.UUID, driverIDs []uuid.UUID) (NationalIDCiphers, error)
}

// Archiver 將多個檔案打包成單一壓縮檔位元組。
type Archiver interface {
	BuildZip(entries []ZipEntry) ([]byte, error)
}

// AuditWriter 定義匯出留痕的寫入邊界。
type AuditWriter interface {
	Write(ctx context.Context, e AuditEntry) error
}
