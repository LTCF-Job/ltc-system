package app

import (
	"time"

	"github.com/google/uuid"
)

// CaseNameRef 是個案的姓名索引，供跨模組以姓名比對個案時使用。
type CaseNameRef struct {
	ID             uuid.UUID
	Name           string
	NameNormalized string
}

// Case 代表 cases 資料表實體。
type Case struct {
	ID                     uuid.UUID
	Name                   string
	NameNormalized         string
	NationalIDCipher       []byte
	NationalIDHMAC         []byte
	NationalIDMasked       string
	NationalIDInvalid      bool
	HouseholdType          *string
	Gender                 *string
	BirthDate              *time.Time
	BirthDateRaw           *string
	CareContactRole        *string
	CareContactName        *string
	RegisteredAddress      *string
	// SiteID 為 nil 且 SiteNameRaw 有值時，表示匯入時的據點名稱未比對到主檔，
	// 待人工於「待維護」畫面補齊。這是個案直接關聯的據點，排班與交通偏好不再各自持有。
	SiteID                 *uuid.UUID
	SiteName               string
	SiteNameRaw            *string
	OutboundVehicleID      *uuid.UUID
	OutboundVehicle        string
	OutboundVehicleNameRaw *string
	InboundVehicleID       *uuid.UUID
	InboundVehicle         string
	InboundVehicleNameRaw  *string
	HomeAddress            *string
	Region                 *string
	LTCLevel               *string
	ServiceCategory        *int
	ServiceUsageType       *int
	ClaimEndDate           *time.Time
	Status                 string
	Remarks                *string
	CreatedAt              time.Time
	UpdatedAt              time.Time
}

// DuplicateCandidate 代表批次匯入偵測到疑似重複個案、尚未裁決的一列暫存資料；
// 裁決前不會出現在 cases 表，避免半確認的個案流入排班、匯出等下游流程。
type DuplicateCandidate struct {
	ID                     uuid.UUID
	FileHash               string
	RowKey                 string
	RowIndex               int
	SheetName              string
	Name                   string
	NameNormalized         string
	NationalIDCipher       []byte
	NationalIDHMAC         []byte
	NationalIDMasked       string
	HouseholdType          *string
	Gender                 *string
	BirthDate              *time.Time
	BirthDateRaw           *string
	NationalIDInvalid      bool
	CareContactRole        *string
	CareContactName        *string
	RegisteredAddress      *string
	HomeAddress            *string
	Region                 *string
	ServiceCategory        *int
	ServiceUsageType       *int
	SiteID                 *uuid.UUID
	SiteName               string
	SiteNameRaw            *string
	OutboundVehicleID      *uuid.UUID
	OutboundVehicle        string
	OutboundVehicleNameRaw *string
	InboundVehicleID       *uuid.UUID
	InboundVehicle         string
	InboundVehicleNameRaw  *string
	Remarks                *string
	DuplicateCaseID        uuid.UUID
	DuplicateCaseName      string
	Status                 string
	ResultingCaseID        *uuid.UUID
	ResolvedAt             *time.Time
	ResolvedBy             *uuid.UUID
	CreatedAt              time.Time
}

// CaseSchedule 代表 case_schedules 與 schedule_legs 之組合排班實體。據點已改由個案
// 本身提供（Case.SiteID），排班不再各自持有據點。
type CaseSchedule struct {
	ID                 uuid.UUID
	CaseID             uuid.UUID
	EffectiveFrom      time.Time
	EffectiveTo        *time.Time
	Weekdays           []int16
	TripPattern        int16
	UnitPrice          float64
	DistanceKM         float64
	ServiceDurationMin int16
	ServiceCode        string
	Note               *string
	Legs               []ScheduleLeg
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

// ScheduleLeg 代表單一趟次之時段與車輛指派。
type ScheduleLeg struct {
	ID          uuid.UUID
	ScheduleID  uuid.UUID
	LegSeq      int16 // 1..4
	Direction   string
	Period      string
	DepartTime  string // "09:40"
	ArriveTime  *string
	RunNo       int16
	VehicleID   *uuid.UUID
	VehicleName string
	CreatedAt   time.Time
}

// ActiveCaseScheduleInfo 代表個案於指定月份之有效排班與關聯基本資訊。SiteOpenDays
// 現由個案的據點帶出（見 GetActiveSchedulesForMonth 的 JOIN 路徑）。
type ActiveCaseScheduleInfo struct {
	CaseID        uuid.UUID
	CaseName      string
	Region        string
	ClaimEndDate  *time.Time
	SiteOpenDays  []int16
	EffectiveFrom time.Time
	EffectiveTo   *time.Time
	Weekdays      []int16
	TripPattern   int16
	Legs          []ScheduleLeg
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
