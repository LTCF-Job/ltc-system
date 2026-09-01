package app

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// CaseStore 定義個案主檔與排班的讀寫邊界。
type CaseStore interface {
	List(ctx context.Context, region, status, q string, page, pageSize int, unresolvedLink, excludePending bool) ([]Case, int64, error)
	GetByID(ctx context.Context, id uuid.UUID) (*Case, error)
	GetByHMAC(ctx context.Context, hmac []byte) (*Case, error)
	GetByNameNormalized(ctx context.Context, nameNorm string) ([]Case, error)
	Create(ctx context.Context, c *Case) error
	Update(ctx context.Context, c *Case) error
	CreateSchedule(ctx context.Context, s *CaseSchedule) error
	GetActiveScheduleForCaseOnDate(ctx context.Context, caseID uuid.UUID, serviceDate time.Time) (*CaseSchedule, error)
	GetActiveSchedulesForMonth(ctx context.Context, year, month int, region string) ([]ActiveCaseScheduleInfo, error)
	// UpsertTransportPreference 寫入個案的單位與去回程車輛偏好。nil 的 ID 表示該欄位
	// 維持現況，僅有非 nil 的 ID 會覆寫對應欄位；raw name 字串隨對應 ID 是否為 nil
	// 一併寫入或清空，供比對不到主檔時保留原始名稱待人工關聯。
	UpsertTransportPreference(ctx context.Context, caseID uuid.UUID, siteID, outboundVehicleID, inboundVehicleID *uuid.UUID, siteNameRaw, outboundVehicleNameRaw, inboundVehicleNameRaw string) error
}

// SiteRef 是驗證個案交通偏好所需的最小單位資訊。
type SiteRef struct {
	ID     uuid.UUID
	Region string
}

// SiteFinder 提供個案交通偏好驗證所需的單筆單位查詢。
type SiteFinder interface {
	GetByID(ctx context.Context, id uuid.UUID) (*SiteRef, error)
}

// AuditWriter 定義個案異動留痕的寫入邊界。
type AuditWriter interface {
	Write(ctx context.Context, e AuditEntry) error
}

// CaseProfileRow 是個案彙整表的一列，欄位順序即表格欄位順序。
type CaseProfileRow struct {
	Name              string
	HouseholdType     string
	NationalID        string
	Gender            string
	Birthday          string
	Age               string
	SiteName          string
	OutboundVehicle   string
	InboundVehicle    string
	CareContactRole   string
	CareContactName   string
	RegisteredAddress string
	HomeAddress       string
}

// ProfileRenderer 產生個案彙整表的 Excel 位元組。
type ProfileRenderer interface {
	RenderCaseProfileWorkbook(rows []CaseProfileRow) ([]byte, error)
}
