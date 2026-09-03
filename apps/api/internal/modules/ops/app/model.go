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
	CreatedAt  time.Time
	UpdatedAt  time.Time
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
	Upsert(ctx context.Context, driverID uuid.UUID, recordDate time.Time, status string, note *string) (*AttendanceRecord, error)
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
