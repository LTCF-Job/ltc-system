package repository

import (
	"time"

	"github.com/google/uuid"
)

// CaseEntity 代表 cases 資料表實體。
type CaseEntity struct {
	ID                 uuid.UUID  `json:"id"`
	Code               string     `json:"code"`
	Name               string     `json:"name"`
	NameNormalized     string     `json:"nameNormalized"`
	NationalIDCipher   []byte     `json:"-"`
	NationalIDHMAC     []byte     `json:"-"`
	NationalIDMasked   string     `json:"nationalIdMasked"`
	HomeAddress        string     `json:"homeAddress"`
	Region             string     `json:"region"`
	LTCLevel           *string    `json:"ltcLevel,omitempty"`
	ServiceCategory    int        `json:"serviceCategory"`
	ServiceUsageType   int        `json:"serviceUsageType"`
	ClaimStartDate     time.Time  `json:"claimStartDate"`
	ClaimEndDate       *time.Time `json:"claimEndDate,omitempty"`
	Status             string     `json:"status"`
	CreatedAt          time.Time  `json:"createdAt"`
	UpdatedAt          time.Time  `json:"updatedAt"`
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
	ID            uuid.UUID  `json:"id"`
	DriverID      uuid.UUID  `json:"driverId"`
	DriverName    string     `json:"driverName,omitempty"`
	VehicleID     uuid.UUID  `json:"vehicleId"`
	VehicleName   string     `json:"vehicleName,omitempty"`
	IsPrimary     bool       `json:"isPrimary"`
	EffectiveFrom time.Time  `json:"effectiveFrom"`
	EffectiveTo   *time.Time `json:"effectiveTo,omitempty"`
	CreatedAt     time.Time  `json:"createdAt"`
}

// CaseScheduleEntity 代表 case_schedules 與 schedule_legs 之組合排班實體。
type CaseScheduleEntity struct {
	ID                 uuid.UUID            `json:"id"`
	CaseID             uuid.UUID            `json:"caseId"`
	SiteID             uuid.UUID            `json:"siteId"`
	SiteName           string               `json:"siteName,omitempty"`
	EffectiveFrom      time.Time            `json:"effectiveFrom"`
	EffectiveTo        *time.Time           `json:"effectiveTo,omitempty"`
	Weekdays           []int16              `json:"weekdays"`
	TripPattern        int16                `json:"tripPattern"`
	UnitPrice          float64              `json:"unitPrice"`
	DistanceKM         float64              `json:"distanceKm"`
	ServiceDurationMin int16                `json:"serviceDurationMin"`
	ServiceCode        string               `json:"serviceCode"`
	Note               *string              `json:"note,omitempty"`
	Legs               []ScheduleLegEntity  `json:"legs,omitempty"`
	CreatedAt          time.Time            `json:"createdAt"`
	UpdatedAt          time.Time            `json:"updatedAt"`
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
