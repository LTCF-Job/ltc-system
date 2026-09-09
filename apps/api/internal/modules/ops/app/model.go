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

// AttendanceAuditSnapshot 是出勤異動的明確稽核快照，不直接序列化含有顯示姓名與
// 備註的 AttendanceRecord。
type AttendanceAuditSnapshot struct {
	ID         uuid.UUID `json:"id"`
	DriverID   uuid.UUID `json:"driverId"`
	RecordDate time.Time `json:"recordDate"`
	Status     string    `json:"status"`
	Source     string    `json:"source"`
}

// AuditSnapshot 產生出勤紀錄的明確稽核快照。
func (a AttendanceRecord) AuditSnapshot() AttendanceAuditSnapshot {
	return AttendanceAuditSnapshot{ID: a.ID, DriverID: a.DriverID, RecordDate: a.RecordDate, Status: a.Status, Source: a.Source}
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

// AttendanceConflictAuditSnapshot 是出勤匯入衝突的明確稽核快照，不保存司機顯示姓名。
type AttendanceConflictAuditSnapshot struct {
	ID             uuid.UUID `json:"id"`
	DriverID       uuid.UUID `json:"driverId"`
	RecordDate     time.Time `json:"recordDate"`
	ExistingStatus string    `json:"existingStatus"`
	ImportedStatus string    `json:"importedStatus"`
	Status         string    `json:"status"`
	ResolvedChoice *string   `json:"resolvedChoice,omitempty"`
}

// AuditSnapshot 產生出勤匯入衝突的明確稽核快照。
func (c AttendanceImportConflict) AuditSnapshot() AttendanceConflictAuditSnapshot {
	return AttendanceConflictAuditSnapshot{
		ID:             c.ID,
		DriverID:       c.DriverID,
		RecordDate:     c.RecordDate,
		ExistingStatus: c.ExistingStatus,
		ImportedStatus: c.ImportedStatus,
		Status:         c.Status,
		ResolvedChoice: c.ResolvedChoice,
	}
}

// AttendanceConflictResolutionAuditSnapshot 是衝突裁決結果的明確稽核快照。
type AttendanceConflictResolutionAuditSnapshot struct {
	ConflictID uuid.UUID `json:"conflictId"`
	Choice     string    `json:"choice"`
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

// FuelAuditSnapshot 是油資紀錄的明確稽核快照；不直接序列化 FuelLog，避免把查詢
// 組裝出的顯示欄位或收據 URL 帶入 audit_log。
type FuelAuditSnapshot struct {
	ID        uuid.UUID  `json:"id"`
	VehicleID uuid.UUID  `json:"vehicleId"`
	DriverID  *uuid.UUID `json:"driverId,omitempty"`
	FuelDate  time.Time  `json:"fuelDate"`
	Liters    float64    `json:"liters"`
	Cost      float64    `json:"cost"`
	CreatedBy uuid.UUID  `json:"createdBy"`
}

// AuditSnapshot 產生油資紀錄的明確稽核快照。
func (f FuelLog) AuditSnapshot() FuelAuditSnapshot {
	return FuelAuditSnapshot{
		ID:        f.ID,
		VehicleID: f.VehicleID,
		DriverID:  f.DriverID,
		FuelDate:  f.FuelDate,
		Liters:    f.Liters,
		Cost:      f.Cost,
		CreatedBy: f.CreatedBy,
	}
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

// MaintenanceAuditSnapshot 是維修紀錄的明確稽核快照；收據 URL、查詢顯示欄位與
// 備註不直接寫入稽核資料。
type MaintenanceAuditSnapshot struct {
	ID          uuid.UUID `json:"id"`
	VehicleID   uuid.UUID `json:"vehicleId"`
	ServiceDate time.Time `json:"serviceDate"`
	Mileage     float64   `json:"mileage"`
	Items       string    `json:"items"`
	Vendor      *string   `json:"vendor,omitempty"`
	Cost        float64   `json:"cost"`
	CreatedBy   uuid.UUID `json:"createdBy"`
}

// AuditSnapshot 產生維修紀錄的明確稽核快照。
func (m MaintenanceLog) AuditSnapshot() MaintenanceAuditSnapshot {
	return MaintenanceAuditSnapshot{
		ID:          m.ID,
		VehicleID:   m.VehicleID,
		ServiceDate: m.ServiceDate,
		Mileage:     m.Mileage,
		Items:       m.Items,
		Vendor:      m.Vendor,
		Cost:        m.Cost,
		CreatedBy:   m.CreatedBy,
	}
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
	IPAddress  *string
	UserAgent  *string
}

// AuditContext 是 HTTP mutation 的來源資訊；保留為 optional 參數以維持非 HTTP
// 呼叫端與既有測試的相容性。
type AuditContext struct {
	IPAddress *string
	UserAgent *string
}

// AuditWriter 定義營運紀錄異動留痕的寫入邊界。
type AuditWriter interface {
	Write(ctx context.Context, e AuditEntry) error
}

// DriverLister 提供出勤月報所需的司機清單。
type DriverLister interface {
	List(ctx context.Context, region, q string, page, pageSize int) ([]DriverRef, int64, error)
	ListAllActive(ctx context.Context) ([]DriverRef, error)
}

// ActiveDriverQueryLister 提供可在資料庫端依姓名篩選的完整啟用司機清單。
// 未實作此 optional port 的離線 reader 仍由 service 做保守的記憶體篩選。
type ActiveDriverQueryLister interface {
	ListAllActiveByQuery(ctx context.Context, q string) ([]DriverRef, error)
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
	// DeleteConflict 移除一筆待維護衝突；查無資料回傳 ErrAttendanceConflictNotFound。
	DeleteConflict(ctx context.Context, id uuid.UUID) error
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

// TxRunner 封裝需要跨多次資料異動的交易邊界。
type TxRunner interface {
	WithTx(ctx context.Context, fn func(context.Context) error) error
}
