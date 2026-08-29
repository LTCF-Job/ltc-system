package app

import (
	"time"

	"github.com/google/uuid"
)

// Case 代表 cases 資料表實體。
type Case struct {
	ID                uuid.UUID
	Code              string
	Name              string
	NameNormalized    string
	NationalIDCipher  []byte
	NationalIDHMAC    []byte
	NationalIDMasked  string
	HouseholdType     *string
	Gender            *string
	BirthDate         *time.Time
	CareContactRole   *string
	CareContactName   *string
	RegisteredAddress *string
	SiteID            *uuid.UUID
	SiteName          string
	OutboundVehicleID *uuid.UUID
	OutboundVehicle   string
	InboundVehicleID  *uuid.UUID
	InboundVehicle    string
	HomeAddress       string
	Region            string
	LTCLevel          *string
	ServiceCategory   int
	ServiceUsageType  int
	ClaimStartDate    time.Time
	ClaimEndDate      *time.Time
	Status            string
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// CaseSchedule 代表 case_schedules 與 schedule_legs 之組合排班實體。
type CaseSchedule struct {
	ID                 uuid.UUID
	CaseID             uuid.UUID
	SiteID             uuid.UUID
	SiteName           string
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

// ScheduleLeg 代表 schedule_legs 實體。
type ScheduleLeg struct {
	ID          uuid.UUID
	ScheduleID  uuid.UUID
	LegSeq      int16
	Direction   string
	Period      string
	DepartTime  string // "09:40"
	ArriveTime  *string
	RunNo       int16
	VehicleID   *uuid.UUID
	VehicleName string
	CreatedAt   time.Time
}

// ActiveCaseScheduleInfo 代表個案於指定月份之有效排班與關聯基本資訊。
type ActiveCaseScheduleInfo struct {
	CaseID         uuid.UUID
	CaseCode       string
	CaseName       string
	Region         string
	ClaimStartDate time.Time
	ClaimEndDate   *time.Time
	SiteID         uuid.UUID
	SiteOpenDays   []int16
	EffectiveFrom  time.Time
	EffectiveTo    *time.Time
	Weekdays       []int16
	TripPattern    int16
	Legs           []ScheduleLeg
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
