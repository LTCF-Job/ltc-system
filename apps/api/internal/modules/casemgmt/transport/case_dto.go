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
	ServiceCategory   *int       `json:"serviceCategory"`
	ServiceUsageType  *int       `json:"serviceUsageType"`
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

// CaseResponse 代表回傳給前端的個案主檔資料。身分證密文與 HMAC 索引不對外輸出。
type CaseResponse struct {
	ID                uuid.UUID  `json:"id"`
	Code              string     `json:"code"`
	Name              string     `json:"name"`
	NameNormalized    string     `json:"nameNormalized"`
	NationalIDMasked  string     `json:"nationalIdMasked"`
	HouseholdType     *string    `json:"householdType"`
	Gender            *string    `json:"gender"`
	BirthDate         *time.Time `json:"birthDate"`
	CareContactRole   *string    `json:"careContactRole"`
	CareContactName   *string    `json:"careContactName"`
	RegisteredAddress *string    `json:"registeredAddress"`
	SiteID            *uuid.UUID `json:"siteId"`
	SiteName          string     `json:"siteName"`
	OutboundVehicleID *uuid.UUID `json:"outboundVehicleId"`
	OutboundVehicle   string     `json:"outboundVehicle"`
	InboundVehicleID  *uuid.UUID `json:"inboundVehicleId"`
	InboundVehicle    string     `json:"inboundVehicle"`
	HomeAddress       *string    `json:"homeAddress"`
	Region            *string    `json:"region"`
	LTCLevel          *string    `json:"ltcLevel"`
	ServiceCategory   *int       `json:"serviceCategory"`
	ServiceUsageType  *int       `json:"serviceUsageType"`
	ClaimEndDate      *time.Time `json:"claimEndDate"`
	Status            string     `json:"status"`
	Remarks           *string    `json:"remarks"`
	CreatedAt         time.Time  `json:"createdAt"`
	UpdatedAt         time.Time  `json:"updatedAt"`
}

func newCaseResponse(c app.Case) CaseResponse {
	return CaseResponse{
		ID:                c.ID,
		Code:              c.Code,
		Name:              c.Name,
		NameNormalized:    c.NameNormalized,
		NationalIDMasked:  c.NationalIDMasked,
		HouseholdType:     c.HouseholdType,
		Gender:            c.Gender,
		BirthDate:         c.BirthDate,
		CareContactRole:   c.CareContactRole,
		CareContactName:   c.CareContactName,
		RegisteredAddress: c.RegisteredAddress,
		SiteID:            c.SiteID,
		SiteName:          c.SiteName,
		OutboundVehicleID: c.OutboundVehicleID,
		OutboundVehicle:   c.OutboundVehicle,
		InboundVehicleID:  c.InboundVehicleID,
		InboundVehicle:    c.InboundVehicle,
		HomeAddress:       c.HomeAddress,
		Region:            c.Region,
		LTCLevel:          c.LTCLevel,
		ServiceCategory:   c.ServiceCategory,
		ServiceUsageType:  c.ServiceUsageType,
		ClaimEndDate:      c.ClaimEndDate,
		Status:            c.Status,
		Remarks:           c.Remarks,
		CreatedAt:         c.CreatedAt,
		UpdatedAt:         c.UpdatedAt,
	}
}

func newCaseResponses(list []app.Case) []CaseResponse {
	if list == nil {
		return nil
	}
	out := make([]CaseResponse, 0, len(list))
	for _, c := range list {
		out = append(out, newCaseResponse(c))
	}
	return out
}

// ScheduleLegResponse 代表回傳給前端的排班單趟資料。
type ScheduleLegResponse struct {
	ID          uuid.UUID  `json:"id"`
	ScheduleID  uuid.UUID  `json:"scheduleId"`
	LegSeq      int16      `json:"legSeq"`
	Direction   string     `json:"direction"`
	Period      string     `json:"period"`
	DepartTime  string     `json:"departTime"`
	ArriveTime  *string    `json:"arriveTime"`
	RunNo       int16      `json:"runNo"`
	VehicleID   *uuid.UUID `json:"vehicleId"`
	VehicleName string     `json:"vehicleName"`
	CreatedAt   time.Time  `json:"createdAt"`
}

func newScheduleLegResponse(l app.ScheduleLeg) ScheduleLegResponse {
	return ScheduleLegResponse{
		ID:          l.ID,
		ScheduleID:  l.ScheduleID,
		LegSeq:      l.LegSeq,
		Direction:   l.Direction,
		Period:      l.Period,
		DepartTime:  l.DepartTime,
		ArriveTime:  l.ArriveTime,
		RunNo:       l.RunNo,
		VehicleID:   l.VehicleID,
		VehicleName: l.VehicleName,
		CreatedAt:   l.CreatedAt,
	}
}

// CaseScheduleResponse 代表回傳給前端的個案排班資料。
type CaseScheduleResponse struct {
	ID                 uuid.UUID             `json:"id"`
	CaseID             uuid.UUID             `json:"caseId"`
	SiteID             uuid.UUID             `json:"siteId"`
	SiteName           string                `json:"siteName"`
	EffectiveFrom      time.Time             `json:"effectiveFrom"`
	EffectiveTo        *time.Time            `json:"effectiveTo"`
	Weekdays           []int16               `json:"weekdays"`
	TripPattern        int16                 `json:"tripPattern"`
	UnitPrice          float64               `json:"unitPrice"`
	DistanceKM         float64               `json:"distanceKm"`
	ServiceDurationMin int16                 `json:"serviceDurationMin"`
	ServiceCode        string                `json:"serviceCode"`
	Note               *string               `json:"note"`
	Legs               []ScheduleLegResponse `json:"legs"`
	CreatedAt          time.Time             `json:"createdAt"`
	UpdatedAt          time.Time             `json:"updatedAt"`
}

func newCaseScheduleResponse(s app.CaseSchedule) CaseScheduleResponse {
	legs := make([]ScheduleLegResponse, 0, len(s.Legs))
	for _, l := range s.Legs {
		legs = append(legs, newScheduleLegResponse(l))
	}
	return CaseScheduleResponse{
		ID:                 s.ID,
		CaseID:             s.CaseID,
		SiteID:             s.SiteID,
		SiteName:           s.SiteName,
		EffectiveFrom:      s.EffectiveFrom,
		EffectiveTo:        s.EffectiveTo,
		Weekdays:           s.Weekdays,
		TripPattern:        s.TripPattern,
		UnitPrice:          s.UnitPrice,
		DistanceKM:         s.DistanceKM,
		ServiceDurationMin: s.ServiceDurationMin,
		ServiceCode:        s.ServiceCode,
		Note:               s.Note,
		Legs:               legs,
		CreatedAt:          s.CreatedAt,
		UpdatedAt:          s.UpdatedAt,
	}
}
