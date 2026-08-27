package repository

import (
	"time"

	"github.com/google/uuid"
)

// CaseEntity 代表 cases 資料表實體。
type CaseEntity struct {
	ID                uuid.UUID  `json:"id"`
	Code              string     `json:"code"`
	Name              string     `json:"name"`
	NameNormalized    string     `json:"nameNormalized"`
	NationalIDCipher  []byte     `json:"-"`
	NationalIDHMAC    []byte     `json:"-"`
	NationalIDMasked  string     `json:"nationalIdMasked"`
	HouseholdType     *string    `json:"householdType,omitempty"`
	Gender            *string    `json:"gender,omitempty"`
	BirthDate         *time.Time `json:"birthDate,omitempty"`
	CareContactRole   *string    `json:"careContactRole,omitempty"`
	CareContactName   *string    `json:"careContactName,omitempty"`
	RegisteredAddress *string    `json:"registeredAddress,omitempty"`
	SiteName          string     `json:"siteName,omitempty"`
	OutboundVehicle   string     `json:"outboundVehicle,omitempty"`
	InboundVehicle    string     `json:"inboundVehicle,omitempty"`
	HomeAddress       string     `json:"homeAddress"`
	Region            string     `json:"region"`
	LTCLevel          *string    `json:"ltcLevel,omitempty"`
	ServiceCategory   int        `json:"serviceCategory"`
	ServiceUsageType  int        `json:"serviceUsageType"`
	ClaimStartDate    time.Time  `json:"claimStartDate"`
	ClaimEndDate      *time.Time `json:"claimEndDate,omitempty"`
	Status            string     `json:"status"`
	CreatedAt         time.Time  `json:"createdAt"`
	UpdatedAt         time.Time  `json:"updatedAt"`
}

// SiteEntity 代表 sites 資料表實體。
type SiteEntity struct {
	ID        uuid.UUID `json:"id"`
	Code      string    `json:"code"`
	Name      string    `json:"name"`
	Address   string    `json:"address"`
	Region    string    `json:"region"`
	OpenDays  []int16   `json:"openDays"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// VehicleEntity 代表 vehicles 資料表實體。
type VehicleEntity struct {
	ID          uuid.UUID `json:"id"`
	PlateNo     string    `json:"plateNo"`
	DisplayName string    `json:"displayName"`
	Region      string    `json:"region"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// DriverEntity 代表 drivers 資料表實體。
type DriverEntity struct {
	ID               uuid.UUID `json:"id"`
	Code             string    `json:"code"`
	Name             string    `json:"name"`
	NameNormalized   string    `json:"nameNormalized"`
	NationalIDCipher []byte    `json:"-"`
	NationalIDHMAC   []byte    `json:"-"`
	NationalIDMasked string    `json:"nationalIdMasked"`
	Email            *string   `json:"email,omitempty"`
	Region           string    `json:"region"`
	Status           string    `json:"status"`
	CreatedAt        time.Time `json:"createdAt"`
	UpdatedAt        time.Time `json:"updatedAt"`
}

// DriverAssignmentEntity 代表 driver_assignments 資料表實體。
type DriverAssignmentEntity struct {
	ID             uuid.UUID  `json:"id"`
	DriverID       uuid.UUID  `json:"driverId"`
	DriverName     string     `json:"driverName,omitempty"`
	VehicleID      uuid.UUID  `json:"vehicleId"`
	VehicleName    string     `json:"vehicleName,omitempty"`
	VehiclePlateNo string     `json:"vehiclePlateNo,omitempty"`
	IsPrimary      bool       `json:"isPrimary"`
	EffectiveFrom  time.Time  `json:"effectiveFrom"`
	EffectiveTo    *time.Time `json:"effectiveTo,omitempty"`
	CreatedAt      time.Time  `json:"createdAt"`
}

// CaseScheduleEntity 代表 case_schedules 與 schedule_legs 之組合排班實體。
type CaseScheduleEntity struct {
	ID                 uuid.UUID           `json:"id"`
	CaseID             uuid.UUID           `json:"caseId"`
	SiteID             uuid.UUID           `json:"siteId"`
	SiteName           string              `json:"siteName,omitempty"`
	EffectiveFrom      time.Time           `json:"effectiveFrom"`
	EffectiveTo        *time.Time          `json:"effectiveTo,omitempty"`
	Weekdays           []int16             `json:"weekdays"`
	TripPattern        int16               `json:"tripPattern"`
	UnitPrice          float64             `json:"unitPrice"`
	DistanceKM         float64             `json:"distanceKm"`
	ServiceDurationMin int16               `json:"serviceDurationMin"`
	ServiceCode        string              `json:"serviceCode"`
	Note               *string             `json:"note,omitempty"`
	Legs               []ScheduleLegEntity `json:"legs,omitempty"`
	CreatedAt          time.Time           `json:"createdAt"`
	UpdatedAt          time.Time           `json:"updatedAt"`
}

// ScheduleLegEntity 代表 schedule_legs 實體。
type ScheduleLegEntity struct {
	ID          uuid.UUID  `json:"id"`
	ScheduleID  uuid.UUID  `json:"scheduleId"`
	LegSeq      int16      `json:"legSeq"`
	Direction   string     `json:"direction"`
	Period      string     `json:"period"`
	DepartTime  string     `json:"departTime"` // "09:40"
	ArriveTime  *string    `json:"arriveTime,omitempty"`
	RunNo       int16      `json:"runNo"`
	VehicleID   *uuid.UUID `json:"vehicleId,omitempty"`
	VehicleName string     `json:"vehicleName,omitempty"`
	CreatedAt   time.Time  `json:"createdAt"`
}

// HolidayEntity 代表國定假日與停駛日。
type HolidayEntity struct {
	HolidayDate time.Time `json:"holidayDate"`
	Name        string    `json:"name"`
	Region      *string   `json:"region,omitempty"`
	Source      string    `json:"source"`
	IsDayOff    bool      `json:"isDayOff"`
	CreatedAt   time.Time `json:"createdAt"`
}

// NotificationRecipientEntity 代表通知收件人設定。
type NotificationRecipientEntity struct {
	ID          int64     `json:"id"`
	Topic       string    `json:"topic"`
	Email       string    `json:"email"`
	DisplayName *string   `json:"displayName,omitempty"`
	Active      bool      `json:"active"`
	CreatedBy   uuid.UUID `json:"createdBy"`
	CreatedAt   time.Time `json:"createdAt"`
}

// NotificationLogEntity 代表通知發送歷史留痕。
type NotificationLogEntity struct {
	ID              int64      `json:"id"`
	Topic           string     `json:"topic"`
	Channel         string     `json:"channel"`
	RecipientEmails []string   `json:"recipientEmails"`
	Subject         string     `json:"subject"`
	ContentSummary  *string    `json:"contentSummary,omitempty"`
	Status          string     `json:"status"`
	ErrorMessage    *string    `json:"errorMessage,omitempty"`
	TriggeredBy     *uuid.UUID `json:"triggeredBy,omitempty"`
	TriggeredByName *string    `json:"triggeredByName,omitempty"`
	SentAt          time.Time  `json:"sentAt"`
}

// AuditLogEntity 代表稽核日誌。
type AuditLogEntity struct {
	ID         int64       `json:"id"`
	ActorID    *uuid.UUID  `json:"actorId,omitempty"`
	ActorRole  *string     `json:"actorRole,omitempty"`
	Action     string      `json:"action"`
	EntityType string      `json:"entityType"`
	EntityID   *string     `json:"entityId,omitempty"`
	BeforeData interface{} `json:"beforeData,omitempty"`
	AfterData  interface{} `json:"afterData,omitempty"`
	IPAddress  *string     `json:"ipAddress,omitempty"`
	UserAgent  *string     `json:"userAgent,omitempty"`
	CreatedAt  time.Time   `json:"createdAt"`
}

// TripSummaryRow 代表個案在特定車輛的單月趟數小計。
type TripSummaryRow struct {
	CaseID        uuid.UUID `json:"caseId"`
	CaseCode      string    `json:"caseCode"`
	CaseName      string    `json:"caseName"`
	OutboundCount int       `json:"outboundCount"`
	InboundCount  int       `json:"inboundCount"`
	TotalCount    int       `json:"totalCount"`
}

// VehicleTripSummary 代表單一車輛之趟數彙整。
type VehicleTripSummary struct {
	VehicleID          uuid.UUID        `json:"vehicleId"`
	PlateNo            string           `json:"plateNo"`
	DisplayName        string           `json:"displayName"`
	Rows               []TripSummaryRow `json:"rows"`
	TotalOutboundCount int              `json:"totalOutboundCount"`
	TotalInboundCount  int              `json:"totalInboundCount"`
	GrandTotalCount    int              `json:"grandTotalCount"`
}

// RegionEntity 代表 regions 資料表實體。
type RegionEntity struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Status      string    `json:"status"`
	SortOrder   int       `json:"sortOrder"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}
