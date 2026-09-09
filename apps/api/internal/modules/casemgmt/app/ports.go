package app

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// CaseStore 定義個案主檔與排班的讀寫邊界。
type CaseStore interface {
	List(ctx context.Context, status, q string, page, pageSize int, unresolvedLink, excludePending bool) ([]Case, int64, error)
	GetByID(ctx context.Context, id uuid.UUID) (*Case, error)
	GetByHMAC(ctx context.Context, hmac []byte) (*Case, error)
	GetByNameNormalized(ctx context.Context, nameNorm string) ([]Case, error)
	Create(ctx context.Context, c *Case) error
	Update(ctx context.Context, c *Case) error
	CreateSchedule(ctx context.Context, s *CaseSchedule) error
	GetActiveScheduleForCaseOnDate(ctx context.Context, caseID uuid.UUID, serviceDate time.Time) (*CaseSchedule, error)
	GetActiveSchedulesForMonth(ctx context.Context, year, month int) ([]ActiveCaseScheduleInfo, error)
	// UpsertTransportPreference 以 PUT 完整替換個案的去回程車輛偏好。nil 的 ID
	// 代表清除欄位；raw name 僅在沒有對應 ID 時保留來源名稱供人工關聯。據點已改由
	// 個案本身持有（見 Update），不在此處理。
	UpsertTransportPreference(ctx context.Context, caseID uuid.UUID, outboundVehicleID, inboundVehicleID *uuid.UUID, outboundVehicleNameRaw, inboundVehicleNameRaw string) error
	SoftDelete(ctx context.Context, id, actorID uuid.UUID) (bool, error)
	CloseOpenSchedules(ctx context.Context, caseID uuid.UUID) error
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

// AuditContext 是交通偏好等跨層呼叫所需的操作者與來源資訊。
type AuditContext struct {
	ActorID   uuid.UUID
	ActorRole string
	IPAddress string
	UserAgent string
}

// TransactionRunner 封裝需要跨多筆資料異動的交易邊界。
type TransactionRunner interface {
	WithTx(ctx context.Context, fn func(context.Context) error) error
}

// CaseProfileRow 是個案彙整表的一列，欄位順序即表格欄位順序。
// OutboundVehicle / InboundVehicle 目前恆為空字串：來源工作表保留這兩欄的版面，但資料暫不流動。
type CaseProfileRow struct {
	Name              string
	HouseholdType     string
	NationalID        string
	Gender            string
	Birthday          string
	SiteName          string
	OutboundVehicle   string
	InboundVehicle    string
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
