package transport

import (
	"time"

	"github.com/google/uuid"
	"ltc-system/apps/api/internal/modules/casemgmt/app"
)

// CreateCaseRequest 代表新增個案主檔請求。僅姓名為必要欄位；其餘欄位皆選填。
type CreateCaseRequest struct {
	Code              string     `json:"code"`
	Name              string     `json:"name" binding:"required"`
	NationalID        string     `json:"nationalId"`
	HouseholdType     *string    `json:"householdType"`
	Gender            *string    `json:"gender"`
	BirthDate         *time.Time `json:"birthDate"`
	CareContactRole   *string    `json:"careContactRole"`
	CareContactName   *string    `json:"careContactName"`
	RegisteredAddress *string    `json:"registeredAddress"`
	HomeAddress       *string    `json:"homeAddress"`
	Region            *string    `json:"region"`
	LTCLevel          *string    `json:"ltcLevel"`
	ServiceCategory   int        `json:"serviceCategory"`
	ServiceUsageType  int        `json:"serviceUsageType"`
	ClaimStartDate    *time.Time `json:"claimStartDate"`
	ClaimEndDate      *time.Time `json:"claimEndDate"`
	Status            string     `json:"status"`
	Remarks           *string    `json:"remarks"`
}

// ToService 轉換為 service 層的建立個案輸入。
func (r CreateCaseRequest) ToService() app.CreateCaseRequest {
	return app.CreateCaseRequest{
		Code:              r.Code,
		Name:              r.Name,
		NationalID:        r.NationalID,
		HouseholdType:     r.HouseholdType,
		Gender:            r.Gender,
		BirthDate:         r.BirthDate,
		CareContactRole:   r.CareContactRole,
		CareContactName:   r.CareContactName,
		RegisteredAddress: r.RegisteredAddress,
		HomeAddress:       r.HomeAddress,
		Region:            r.Region,
		LTCLevel:          r.LTCLevel,
		ServiceCategory:   r.ServiceCategory,
		ServiceUsageType:  r.ServiceUsageType,
		ClaimStartDate:    r.ClaimStartDate,
		ClaimEndDate:      r.ClaimEndDate,
		Status:            r.Status,
		Remarks:           r.Remarks,
	}
}

// CreateScheduleRequest 代表建立個案排班設定請求。
type CreateScheduleRequest struct {
	CaseID             uuid.UUID                      `json:"caseId" binding:"required"`
	SiteID             uuid.UUID                      `json:"siteId" binding:"required"`
	EffectiveFrom      time.Time                      `json:"effectiveFrom" binding:"required"`
	EffectiveTo        *time.Time                     `json:"effectiveTo"`
	Weekdays           []int16                        `json:"weekdays" binding:"required"`
	TripPattern        int16                          `json:"tripPattern" binding:"required"`
	UnitPrice          float64                        `json:"unitPrice" binding:"required"`
	DistanceKM         float64                        `json:"distanceKm" binding:"required"`
	ServiceDurationMin int16                          `json:"serviceDurationMin" binding:"required"`
	ServiceCode        string                         `json:"serviceCode" binding:"required"`
	Note               *string                        `json:"note"`
	Legs               []CreateScheduleLegItemRequest `json:"legs" binding:"required"`
}

// CreateScheduleLegItemRequest 代表排班單趟設定之請求參數。
type CreateScheduleLegItemRequest struct {
	LegSeq     int16      `json:"legSeq" binding:"required"`
	Direction  string     `json:"direction" binding:"required"`
	DepartTime string     `json:"departTime" binding:"required"`
	VehicleID  *uuid.UUID `json:"vehicleId"`
}

// ToService 轉換為 service 層的建立排班輸入。
func (r CreateScheduleRequest) ToService() app.CreateScheduleRequest {
	legs := make([]app.CreateScheduleLegItemRequest, len(r.Legs))
	for i, l := range r.Legs {
		legs[i] = app.CreateScheduleLegItemRequest{
			LegSeq:     l.LegSeq,
			Direction:  l.Direction,
			DepartTime: l.DepartTime,
			VehicleID:  l.VehicleID,
		}
	}
	return app.CreateScheduleRequest{
		CaseID:             r.CaseID,
		SiteID:             r.SiteID,
		EffectiveFrom:      r.EffectiveFrom,
		EffectiveTo:        r.EffectiveTo,
		Weekdays:           r.Weekdays,
		TripPattern:        r.TripPattern,
		UnitPrice:          r.UnitPrice,
		DistanceKM:         r.DistanceKM,
		ServiceDurationMin: r.ServiceDurationMin,
		ServiceCode:        r.ServiceCode,
		Note:               r.Note,
		Legs:               legs,
	}
}
