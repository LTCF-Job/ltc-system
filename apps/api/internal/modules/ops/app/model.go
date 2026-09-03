package app

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// AttendanceRecord 代表司機在某一日的出勤狀態。
type AttendanceRecord struct {
	ID         uuid.UUID
	DriverID   uuid.UUID
	DriverName string
	RecordDate time.Time
	Status     string // work, leave, sick, off
	Note       *string
	Source     string // manual, import
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// AttendanceImportConflict 代表匯報匯入比對到司機出勤，但當天已有人工登記且狀態不同，
// 需要使用者決定要保留人工登記還是改採匯入結果。
type AttendanceImportConflict struct {
	ID             uuid.UUID
	DriverID       uuid.UUID
	DriverName     string
	RecordDate     time.Time
	ExistingStatus string
	ImportedStatus string
	Status         string // pending, resolved
	ResolvedChoice *string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// FuelLog 代表一筆油資紀錄。
type FuelLog struct {
	ID          uuid.UUID
	VehicleID   uuid.UUID
	VehicleName string
	PlateNo     string
	DriverID    *uuid.UUID
	DriverName  *string
	FuelDate    time.Time
	Liters      float64
	Cost        float64
	ReceiptURL  *string
	CreatedBy   uuid.UUID
	CreatedAt   time.Time
}

// MaintenanceLog 代表一筆車輛維修保養紀錄。
type MaintenanceLog struct {
	ID          uuid.UUID
	VehicleID   uuid.UUID
	VehicleName string
	PlateNo     string
	ServiceDate time.Time
	Mileage     float64
	Items       string
	Vendor      *string
	Cost        float64
	ReceiptURL  *string
	Note        *string
	CreatedBy   uuid.UUID
	CreatedAt   time.Time
}

// DriverRef 是出勤月報與維修紀錄需要的最小司機／車輛資訊。
type DriverRef struct {
	ID     uuid.UUID
	Name   string
	Region string
}

// VehicleRef 是維修紀錄組裝顯示名稱所需的最小車輛資訊。
type VehicleRef struct {
	ID          uuid.UUID
	DisplayName string
	PlateNo     string
}

// AuditEntry 是本模組寫入稽核日誌的內容。
type AuditEntry struct {
	ActorID    *uuid.UUID
	ActorRole  *string
	Action     string
	EntityType string
	EntityID   *string
	BeforeData interface{}
	AfterData  interface{}
}

// AuditWriter 定義營運紀錄異動留痕的寫入邊界。
type AuditWriter interface {
	Write(ctx context.Context, e AuditEntry) error
}

// DriverLister 提供出勤月報所需的司機清單。
type DriverLister interface {
	List(ctx context.Context, region, q string, page, pageSize int) ([]DriverRef, int64, error)
}

// VehicleLister 提供維修紀錄組裝車輛顯示名稱所需的車輛清單。
type VehicleLister interface {
	List(ctx context.Context, region, q string, page, pageSize int) ([]VehicleRef, int64, error)
}

// AttendanceStore 定義司機出勤紀錄的讀寫邊界。
type AttendanceStore interface {
	GetMonthRecords(ctx context.Context, startDate, endDate time.Time, driverID *uuid.UUID) ([]AttendanceRecord, error)
	// GetOne 查詢單一司機單日的出勤紀錄；不存在時回傳 nil, nil。
	GetOne(ctx context.Context, driverID uuid.UUID, recordDate time.Time) (*AttendanceRecord, error)
	// Upsert 寫入或更新一筆出勤紀錄，source 標示是人工登記(manual)還是匯入同步(import)。
	Upsert(ctx context.Context, driverID uuid.UUID, recordDate time.Time, status string, note *string, source string) (*AttendanceRecord, error)
	// UpsertConflict 記錄一筆匯入與人工登記不一致的待維護衝突；同一司機同一天已有未解決的
	// 衝突時更新內容，已解決但人工狀態未再變動時維持已解決，不重複打擾使用者。
	UpsertConflict(ctx context.Context, driverID uuid.UUID, recordDate time.Time, existingStatus, importedStatus string) error
	ListConflicts(ctx context.Context, status string) ([]AttendanceImportConflict, error)
	GetConflict(ctx context.Context, id uuid.UUID) (*AttendanceImportConflict, error)
	// ResolveConflict 把一筆待維護衝突標記為已解決；choice 為 keep_manual 或 use_import。
	ResolveConflict(ctx context.Context, id uuid.UUID, choice string, actorID *uuid.UUID) error
}

// HolidayReader 提供出勤月曆判斷休假日所需之最小介面。
type HolidayReader interface {
	GetHolidayMap(ctx context.Context, year, month int, region string) (map[string]bool, error)
}

// FuelStore 定義油資紀錄的讀寫邊界。
type FuelStore interface {
	List(ctx context.Context, page, pageSize int, vehicleID, driverID *uuid.UUID, startDate, endDate *time.Time, q string) ([]FuelLog, int, error)
	Create(ctx context.Context, item *FuelLog) error
	Update(ctx context.Context, item *FuelLog) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// MaintenanceStore 定義車輛維修紀錄的讀寫邊界。
type MaintenanceStore interface {
	List(ctx context.Context, page, pageSize int, vehicleID *uuid.UUID, startDate, endDate *time.Time, q string) ([]MaintenanceLog, int, error)
	Create(ctx context.Context, item *MaintenanceLog) error
	Update(ctx context.Context, item *MaintenanceLog) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// MaintenanceTemplateRenderer 產生維修紀錄空白範本的 Excel 位元組。
type MaintenanceTemplateRenderer interface {
	RenderBlankMaintenanceTemplate(labels []VehicleLabel) ([]byte, error)
}

// VehicleLabel 是空白範本上的車輛標示。
type VehicleLabel struct {
	DisplayName string
	PlateNo     string
}
