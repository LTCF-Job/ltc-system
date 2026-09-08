package transport

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/google/uuid"
	"ltc-system/apps/api/internal/domain/rocdate"
	"ltc-system/apps/api/internal/modules/casemgmt/app"
)

// parseFlexibleDate 依序嘗試西元／民國純日期字串與完整 RFC3339 時間字串，相容前端日期選擇器與既有 API 呼叫者。
func parseFlexibleDate(raw string) (time.Time, error) {
	if value, err := rocdate.ParseDate(raw); err == nil {
		return value, nil
	}
	return time.Parse(time.RFC3339, raw)
}

type optionalDate struct {
	Present bool
	Value   *time.Time
}

func (d *optionalDate) UnmarshalJSON(data []byte) error {
	d.Present = true
	if string(data) == "null" {
		d.Value = nil
		return nil
	}
	var raw string
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if strings.TrimSpace(raw) == "" {
		d.Value = nil
		return nil
	}
	value, err := parseFlexibleDate(raw)
	if err != nil {
		return err
	}
	d.Value = &value
	return nil
}

// toTimePtr 將可能為 nil 的 optionalDate 轉為服務層使用的 *time.Time；欄位未提供或明確傳入 null 皆視為無結束日。
func (d *optionalDate) toTimePtr() *time.Time {
	if d == nil {
		return nil
	}
	return d.Value
}

// requiredDate 解析必填日期欄位（如排班生效起始日），接受前端日期選擇器送出的純日期字串或既有呼叫者送出的 RFC3339 時間字串。
type requiredDate time.Time

func (d *requiredDate) UnmarshalJSON(data []byte) error {
	var raw string
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	value, err := parseFlexibleDate(raw)
	if err != nil {
		return err
	}
	*d = requiredDate(value)
	return nil
}

func (d requiredDate) toTime() time.Time {
	return time.Time(d)
}

// CreateCaseRequest 代表新增個案主檔請求。姓名與所屬據點為必要欄位；其餘欄位皆選填。
type CreateCaseRequest struct {
	Name              string     `json:"name" binding:"required"`
	SiteID            uuid.UUID  `json:"siteId" binding:"required"`
	NationalID        string     `json:"nationalId"`
	HouseholdType     *string    `json:"householdType"`
	Gender            *string    `json:"gender"`
	BirthDate         *time.Time `json:"birthDate"`
	CareContactRole   *string    `json:"careContactRole"`
	CareContactName   *string    `json:"careContactName"`
	RegisteredAddress *string    `json:"registeredAddress"`
	HomeAddress       *string    `json:"homeAddress"`
	LTCLevel          *string    `json:"ltcLevel"`
	ServiceCategory   *int       `json:"serviceCategory"`
	ServiceUsageType  *int       `json:"serviceUsageType"`
	ClaimEndDate      *time.Time `json:"claimEndDate"`
	Status            string     `json:"status"`
	Remarks           *string    `json:"remarks"`
}

// ToService 轉換為 service 層的建立個案輸入。
func (r CreateCaseRequest) ToService() app.CreateCaseRequest {
	siteID := r.SiteID
	return app.CreateCaseRequest{
		Name:              r.Name,
		SiteID:            &siteID,
		NationalID:        r.NationalID,
		HouseholdType:     r.HouseholdType,
		Gender:            r.Gender,
		BirthDate:         r.BirthDate,
		CareContactRole:   r.CareContactRole,
		CareContactName:   r.CareContactName,
		RegisteredAddress: r.RegisteredAddress,
		HomeAddress:       r.HomeAddress,
		LTCLevel:          r.LTCLevel,
		ServiceCategory:   r.ServiceCategory,
		ServiceUsageType:  r.ServiceUsageType,
		ClaimEndDate:      r.ClaimEndDate,
		Status:            r.Status,
		Remarks:           r.Remarks,
	}
}

// CreateScheduleRequest 代表建立個案排班設定請求。據點已改由個案本身持有，不在排班中設定。
type CreateScheduleRequest struct {
	CaseID             uuid.UUID                      `json:"caseId" binding:"required"`
	EffectiveFrom      requiredDate                   `json:"effectiveFrom" binding:"required"`
	EffectiveTo        *optionalDate                  `json:"effectiveTo"`
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

// SaveScheduleRequest 代表依個案路徑參數儲存排班設定之請求參數。
type SaveScheduleRequest struct {
	EffectiveFrom      requiredDate                   `json:"effectiveFrom" binding:"required"`
	EffectiveTo        *optionalDate                  `json:"effectiveTo"`
	Weekdays           []int16                        `json:"weekdays" binding:"required"`
	TripPattern        int16                          `json:"tripPattern" binding:"required"`
	UnitPrice          float64                        `json:"unitPrice" binding:"required"`
	DistanceKM         float64                        `json:"distanceKm" binding:"required"`
	ServiceDurationMin int16                          `json:"serviceDurationMin" binding:"required"`
	ServiceCode        string                         `json:"serviceCode" binding:"required"`
	Note               *string                        `json:"note"`
	Legs               []CreateScheduleLegItemRequest `json:"legs" binding:"required"`
}

// ToService 以 URL 的個案 ID 組成 service 層的儲存排班輸入。
func (r SaveScheduleRequest) ToService(caseID uuid.UUID) app.CreateScheduleRequest {
	return app.CreateScheduleRequest{
		CaseID:             caseID,
		EffectiveFrom:      r.EffectiveFrom.toTime(),
		EffectiveTo:        r.EffectiveTo.toTimePtr(),
		Weekdays:           r.Weekdays,
		TripPattern:        r.TripPattern,
		UnitPrice:          r.UnitPrice,
		DistanceKM:         r.DistanceKM,
		ServiceDurationMin: r.ServiceDurationMin,
		ServiceCode:        r.ServiceCode,
		Note:               r.Note,
		Legs:               toServiceScheduleLegs(r.Legs),
	}
}

func toServiceScheduleLegs(legs []CreateScheduleLegItemRequest) []app.CreateScheduleLegItemRequest {
	result := make([]app.CreateScheduleLegItemRequest, len(legs))
	for i, l := range legs {
		result[i] = app.CreateScheduleLegItemRequest{
			LegSeq:     l.LegSeq,
			Direction:  l.Direction,
			DepartTime: l.DepartTime,
			VehicleID:  l.VehicleID,
		}
	}
	return result
}

// ToService 轉換為 service 層的建立排班輸入。
func (r CreateScheduleRequest) ToService() app.CreateScheduleRequest {
	return app.CreateScheduleRequest{
		CaseID:             r.CaseID,
		EffectiveFrom:      r.EffectiveFrom.toTime(),
		EffectiveTo:        r.EffectiveTo.toTimePtr(),
		Weekdays:           r.Weekdays,
		TripPattern:        r.TripPattern,
		UnitPrice:          r.UnitPrice,
		DistanceKM:         r.DistanceKM,
		ServiceDurationMin: r.ServiceDurationMin,
		ServiceCode:        r.ServiceCode,
		Note:               r.Note,
		Legs:               toServiceScheduleLegs(r.Legs),
	}
}

// CaseResponse 代表回傳給前端的個案主檔資料。身分證密文與 HMAC 索引不對外輸出。
type CaseResponse struct {
	ID                     uuid.UUID  `json:"id"`
	Name                   string     `json:"name"`
	NameNormalized         string     `json:"nameNormalized"`
	NationalIDMasked       string     `json:"nationalIdMasked"`
	NationalIDInvalid      bool       `json:"nationalIdInvalid"`
	HouseholdType          *string    `json:"householdType"`
	Gender                 *string    `json:"gender"`
	BirthDate              *time.Time `json:"birthDate"`
	BirthDateRaw           *string    `json:"birthDateRaw"`
	CareContactRole        *string    `json:"careContactRole"`
	CareContactName        *string    `json:"careContactName"`
	RegisteredAddress      *string    `json:"registeredAddress"`
	SiteID                 *uuid.UUID `json:"siteId"`
	SiteName               string     `json:"siteName"`
	SiteNameRaw            *string    `json:"siteNameRaw"`
	OutboundVehicleID      *uuid.UUID `json:"outboundVehicleId"`
	OutboundVehicle        string     `json:"outboundVehicle"`
	OutboundVehicleNameRaw *string    `json:"outboundVehicleNameRaw"`
	InboundVehicleID       *uuid.UUID `json:"inboundVehicleId"`
	InboundVehicle         string     `json:"inboundVehicle"`
	InboundVehicleNameRaw  *string    `json:"inboundVehicleNameRaw"`
	HomeAddress            *string    `json:"homeAddress"`
	LTCLevel               *string    `json:"ltcLevel"`
	ServiceCategory        *int       `json:"serviceCategory"`
	ServiceUsageType       *int       `json:"serviceUsageType"`
	ClaimEndDate           *time.Time `json:"claimEndDate"`
	Status                 string     `json:"status"`
	Remarks                *string    `json:"remarks"`
	CreatedAt              time.Time  `json:"createdAt"`
	UpdatedAt              time.Time  `json:"updatedAt"`
}

func newCaseResponse(c app.Case) CaseResponse {
	return CaseResponse{
		ID:                     c.ID,
		Name:                   c.Name,
		NameNormalized:         c.NameNormalized,
		NationalIDMasked:       c.NationalIDMasked,
		NationalIDInvalid:      c.NationalIDInvalid,
		HouseholdType:          c.HouseholdType,
		Gender:                 c.Gender,
		BirthDate:              c.BirthDate,
		BirthDateRaw:           c.BirthDateRaw,
		CareContactRole:        c.CareContactRole,
		CareContactName:        c.CareContactName,
		RegisteredAddress:      c.RegisteredAddress,
		SiteID:                 c.SiteID,
		SiteName:               c.SiteName,
		SiteNameRaw:            c.SiteNameRaw,
		OutboundVehicleID:      c.OutboundVehicleID,
		OutboundVehicle:        c.OutboundVehicle,
		OutboundVehicleNameRaw: c.OutboundVehicleNameRaw,
		InboundVehicleID:       c.InboundVehicleID,
		InboundVehicle:         c.InboundVehicle,
		InboundVehicleNameRaw:  c.InboundVehicleNameRaw,
		HomeAddress:            c.HomeAddress,
		LTCLevel:               c.LTCLevel,
		ServiceCategory:        c.ServiceCategory,
		ServiceUsageType:       c.ServiceUsageType,
		ClaimEndDate:           c.ClaimEndDate,
		Status:                 c.Status,
		Remarks:                c.Remarks,
		CreatedAt:              c.CreatedAt,
		UpdatedAt:              c.UpdatedAt,
	}
}

// DuplicateCandidateResponse 代表回傳給前端的待裁決疑似重複個案暫存列。身分證字號
// 只提供遮罩值，明文需另外呼叫 reveal API 並留下稽核紀錄後才會回傳。
type DuplicateCandidateResponse struct {
	ID                     uuid.UUID  `json:"id"`
	RowIndex               int        `json:"rowIndex"`
	SheetName              string     `json:"sheetName"`
	Name                   string     `json:"name"`
	NationalIDMasked       string     `json:"nationalIdMasked"`
	NationalIDInvalid      bool       `json:"nationalIdInvalid"`
	HouseholdType          *string    `json:"householdType"`
	Gender                 *string    `json:"gender"`
	BirthDate              *time.Time `json:"birthDate"`
	BirthDateRaw           *string    `json:"birthDateRaw"`
	CareContactRole        *string    `json:"careContactRole"`
	CareContactName        *string    `json:"careContactName"`
	RegisteredAddress      *string    `json:"registeredAddress"`
	HomeAddress            *string    `json:"homeAddress"`
	SiteID                 *uuid.UUID `json:"siteId"`
	SiteName               string     `json:"siteName"`
	SiteNameRaw            *string    `json:"siteNameRaw"`
	OutboundVehicleID      *uuid.UUID `json:"outboundVehicleId"`
	OutboundVehicle        string     `json:"outboundVehicle"`
	OutboundVehicleNameRaw *string    `json:"outboundVehicleNameRaw"`
	InboundVehicleID       *uuid.UUID `json:"inboundVehicleId"`
	InboundVehicle         string     `json:"inboundVehicle"`
	InboundVehicleNameRaw  *string    `json:"inboundVehicleNameRaw"`
	Remarks                *string    `json:"remarks"`
	DuplicateCaseID        uuid.UUID  `json:"duplicateCaseId"`
	DuplicateCaseName      string     `json:"duplicateCaseName"`
	Status                 string     `json:"status"`
	CreatedAt              time.Time  `json:"createdAt"`
}

func newDuplicateCandidateResponse(c app.DuplicateCandidate) DuplicateCandidateResponse {
	return DuplicateCandidateResponse{
		ID:                     c.ID,
		RowIndex:               c.RowIndex,
		SheetName:              c.SheetName,
		Name:                   c.Name,
		NationalIDMasked:       c.NationalIDMasked,
		NationalIDInvalid:      c.NationalIDInvalid,
		HouseholdType:          c.HouseholdType,
		Gender:                 c.Gender,
		BirthDate:              c.BirthDate,
		BirthDateRaw:           c.BirthDateRaw,
		CareContactRole:        c.CareContactRole,
		CareContactName:        c.CareContactName,
		RegisteredAddress:      c.RegisteredAddress,
		HomeAddress:            c.HomeAddress,
		SiteID:                 c.SiteID,
		SiteName:               c.SiteName,
		SiteNameRaw:            c.SiteNameRaw,
		OutboundVehicleID:      c.OutboundVehicleID,
		OutboundVehicle:        c.OutboundVehicle,
		OutboundVehicleNameRaw: c.OutboundVehicleNameRaw,
		InboundVehicleID:       c.InboundVehicleID,
		InboundVehicle:         c.InboundVehicle,
		InboundVehicleNameRaw:  c.InboundVehicleNameRaw,
		Remarks:                c.Remarks,
		DuplicateCaseID:        c.DuplicateCaseID,
		DuplicateCaseName:      c.DuplicateCaseName,
		Status:                 c.Status,
		CreatedAt:              c.CreatedAt,
	}
}

func newDuplicateCandidateResponses(list []app.DuplicateCandidate) []DuplicateCandidateResponse {
	out := make([]DuplicateCandidateResponse, 0, len(list))
	for _, c := range list {
		out = append(out, newDuplicateCandidateResponse(c))
	}
	return out
}

// ResolveDuplicateCandidateRequest 代表裁決一筆疑似重複個案的請求。
type ResolveDuplicateCandidateRequest struct {
	Decision     string     `json:"decision" binding:"required"`
	TargetCaseID *uuid.UUID `json:"targetCaseId"`
	MergeRemarks bool       `json:"mergeRemarks"`
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
