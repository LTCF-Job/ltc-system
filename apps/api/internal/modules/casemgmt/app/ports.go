package app

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// CaseStore 定義個案主檔與排班的讀寫邊界。
type CaseStore interface {
	List(ctx context.Context, status, q, region string, page, pageSize int, unresolvedLink, excludePending bool) ([]Case, int64, error)
	GetByID(ctx context.Context, id uuid.UUID) (*Case, error)
	GetByHMAC(ctx context.Context, hmac []byte) (*Case, error)
	GetByNameNormalized(ctx context.Context, nameNorm string) ([]Case, error)
	Create(ctx context.Context, c *Case) error
	Update(ctx context.Context, c *Case) error
	CreateSchedule(ctx context.Context, s *CaseSchedule) error
	GetActiveScheduleForCaseOnDate(ctx context.Context, caseID uuid.UUID, serviceDate time.Time) (*CaseSchedule, error)
	GetActiveSchedulesForMonth(ctx context.Context, year, month int) ([]ActiveCaseScheduleInfo, error)
	SoftDelete(ctx context.Context, id, actorID uuid.UUID) (bool, error)
	CloseOpenSchedules(ctx context.Context, caseID uuid.UUID) error
	// RelinkSiteByName 依名稱重新比對待維護個案的據點，唯一命中才寫入並回傳受影響的個案 ID。
	RelinkSiteByName(ctx context.Context, name string) ([]uuid.UUID, error)
	// RelinkCaregiverByName 依名稱重新比對待維護個案的照護人員，規則與匯入時的
	// resolveCaregiver 一致，唯一命中才寫入並回傳受影響的個案 ID。
	RelinkCaregiverByName(ctx context.Context, name string) ([]uuid.UUID, error)
	// ListPendingSiteNames 列出目前待維護個案中相異的據點原始名稱，供「重新比對」按鈕使用。
	ListPendingSiteNames(ctx context.Context) ([]string, error)
	// ListPendingCaregiverNames 列出目前待維護個案中相異的照護人員原始姓名，供
	// 「重新比對」按鈕使用。
	ListPendingCaregiverNames(ctx context.Context) ([]string, error)
}

// DuplicateStagingStore 定義疑似重複個案暫存列的讀寫邊界；裁決前不落地到 cases 表。
type DuplicateStagingStore interface {
	// Insert 寫入一筆暫存列；同一 (fileHash, rowKey) 已存在時回傳 alreadyStaged=true 且不重複寫入。
	Insert(ctx context.Context, cand DuplicateCandidate) (id uuid.UUID, alreadyStaged bool, err error)
	ListPending(ctx context.Context) ([]DuplicateCandidate, error)
	GetByID(ctx context.Context, id uuid.UUID) (*DuplicateCandidate, error)
	// Resolve 將暫存列標記為裁決結果；rowsAffected=0 代表該列已被裁決過（並發保護）。
	Resolve(ctx context.Context, id uuid.UUID, status string, resolvedBy uuid.UUID, resultingCaseID *uuid.UUID) (rowsAffected int64, err error)
	// Delete 移除尚未裁決的暫存列；rowsAffected=0 代表該列不存在或已被裁決過。
	Delete(ctx context.Context, id uuid.UUID) (rowsAffected int64, err error)
}

// AuditWriter 定義個案異動留痕的寫入邊界。
type AuditWriter interface {
	Write(ctx context.Context, e AuditEntry) error
}

// TransactionRunner 封裝需要跨多筆資料異動的交易邊界。
type TransactionRunner interface {
	WithTx(ctx context.Context, fn func(context.Context) error) error
}

// CaseProfileRow 是個案彙整表的一列，欄位順序即表格欄位順序。
type CaseProfileRow struct {
	Name              string
	HouseholdType     string
	NationalID        string
	Gender            string
	Birthday          string
	SiteName          string
	CareContactRole   string
	CareContactName   string
	RegisteredAddress string
	HomeAddress       string
	Remarks           string
}

// ProfileRenderer 產生個案彙整表的 Excel 位元組。
type ProfileRenderer interface {
	RenderCaseProfileWorkbook(rows []CaseProfileRow) ([]byte, error)
}
