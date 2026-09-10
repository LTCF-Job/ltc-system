package transport

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/google/uuid"
	"ltc-system/apps/api/internal/domain/rocdate"
	"ltc-system/apps/api/internal/modules/masterdata/app"
)

// 本檔的 response DTO 是 masterdata 對外的 API 契約。json tag 與搬遷前
// repository entity 逐欄一致，搬遷不得改變任何既有回應形狀；轉換函式對 nil
// slice 回傳 nil，維持清單為空時序列化成 null 的既有行為。

// SiteResponse 代表回傳給前端的據點資料。
type SiteResponse struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Address   string    `json:"address"`
	Region    string    `json:"region"`
	Remarks   string    `json:"remarks"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func newSiteResponse(s app.Site) SiteResponse {
	return SiteResponse{
		ID:        s.ID,
		Name:      s.Name,
		Address:   s.Address,
		Region:    s.Region,
		Remarks:   s.Remarks,
		Status:    s.Status,
		CreatedAt: s.CreatedAt,
		UpdatedAt: s.UpdatedAt,
	}
}

func newSiteResponses(list []app.Site) []SiteResponse {
	if list == nil {
		return nil
	}
	out := make([]SiteResponse, 0, len(list))
	for _, s := range list {
		out = append(out, newSiteResponse(s))
	}
	return out
}

// CreateSiteRequest 代表新增據點請求。
type CreateSiteRequest struct {
	Name    string `json:"name" binding:"required"`
	Address string `json:"address"`
	Region  string `json:"region"`
	Remarks string `json:"remarks"`
	Status  string `json:"status"`
}

// UpdateSiteRequest 代表更新據點請求。
type UpdateSiteRequest struct {
	Name    string `json:"name" binding:"required"`
	Address string `json:"address"`
	Region  string `json:"region"`
	Remarks string `json:"remarks"`
	Status  string `json:"status"`
}

// VehicleResponse 代表回傳給前端的車輛資料。
type VehicleResponse struct {
	ID                        uuid.UUID            `json:"id"`
	PlateNo                   string               `json:"plateNo"`
	DisplayName               string               `json:"displayName"`
	SiteName                  string               `json:"siteName"`
	Brand                     string               `json:"brand"`
	Model                     string               `json:"model"`
	ManufactureYM             string               `json:"manufactureYm"`
	CompulsoryInsuranceExpiry *time.Time           `json:"compulsoryInsuranceExpiry"`
	PassengerInsuranceExpiry  *time.Time           `json:"passengerInsuranceExpiry"`
	ThirdPartyInsuranceExpiry *time.Time           `json:"thirdPartyInsuranceExpiry"`
	LastInspectionDate        *time.Time           `json:"lastInspectionDate"`
	WheelchairAccessible      *bool                `json:"wheelchairAccessible"`
	HasVehicleLicense         bool                 `json:"hasVehicleLicense"`
	HasPurchaseContract       bool                 `json:"hasPurchaseContract"`
	HasPlateRegistration      bool                 `json:"hasPlateRegistration"`
	HasTransferRegistration   bool                 `json:"hasTransferRegistration"`
	Remarks                   string               `json:"remarks"`
	Status                    string               `json:"status"`
	Drivers                   []VehicleDriverBrief `json:"drivers"`
	CreatedAt                 time.Time            `json:"createdAt"`
	UpdatedAt                 time.Time            `json:"updatedAt"`
}

// VehicleDriverBrief 代表掛在車輛上的司機摘要。
type VehicleDriverBrief struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

func newVehicleResponse(v app.Vehicle) VehicleResponse {
	drivers := make([]VehicleDriverBrief, 0, len(v.Drivers))
	for _, d := range v.Drivers {
		drivers = append(drivers, VehicleDriverBrief{ID: d.ID, Name: d.Name})
	}
	return VehicleResponse{
		ID:                        v.ID,
		PlateNo:                   v.PlateNo,
		DisplayName:               v.DisplayName,
		SiteName:                  v.SiteName,
		Brand:                     v.Brand,
		Model:                     v.Model,
		ManufactureYM:             v.ManufactureYM,
		CompulsoryInsuranceExpiry: v.CompulsoryInsuranceExpiry,
		PassengerInsuranceExpiry:  v.PassengerInsuranceExpiry,
		ThirdPartyInsuranceExpiry: v.ThirdPartyInsuranceExpiry,
		LastInspectionDate:        v.LastInspectionDate,
		WheelchairAccessible:      v.WheelchairAccessible,
		HasVehicleLicense:         v.HasVehicleLicense,
		HasPurchaseContract:       v.HasPurchaseContract,
		HasPlateRegistration:      v.HasPlateRegistration,
		HasTransferRegistration:   v.HasTransferRegistration,
		Remarks:                   v.Remarks,
		Status:                    v.Status,
		Drivers:                   drivers,
		CreatedAt:                 v.CreatedAt,
		UpdatedAt:                 v.UpdatedAt,
	}
}

func newVehicleResponses(list []app.Vehicle) []VehicleResponse {
	if list == nil {
		return nil
	}
	out := make([]VehicleResponse, 0, len(list))
	for _, v := range list {
		out = append(out, newVehicleResponse(v))
	}
	return out
}

// CreateVehicleRequest 代表新增車輛請求。既有 API 未強制任何必填欄位，維持不變。
type CreateVehicleRequest struct {
	VehicleWriteFields
}

// UpdateVehicleRequest 代表更新車輛請求。
type UpdateVehicleRequest struct {
	VehicleWriteFields
}

// VehicleWriteFields 是新增與更新車輛共用的可寫欄位。據點是車輛自己的自由輸入文字，
// 非必填，不關聯據點主檔。車號與車別為必填。
type VehicleWriteFields struct {
	PlateNo                   string    `json:"plateNo" binding:"required"`
	DisplayName               string    `json:"displayName" binding:"required"`
	SiteName                  string    `json:"siteName"`
	Brand                     string    `json:"brand"`
	Model                     string    `json:"model"`
	ManufactureYM             string    `json:"manufactureYm"`
	CompulsoryInsuranceExpiry *wireDate `json:"compulsoryInsuranceExpiry"`
	PassengerInsuranceExpiry  *wireDate `json:"passengerInsuranceExpiry"`
	ThirdPartyInsuranceExpiry *wireDate `json:"thirdPartyInsuranceExpiry"`
	LastInspectionDate        *wireDate `json:"lastInspectionDate"`
	WheelchairAccessible      *bool     `json:"wheelchairAccessible"`
	// 四項證件持有註記：行照、汽車買賣合約書、領牌登記書、異動登記書。未提供時預設 false。
	HasVehicleLicense       *bool  `json:"hasVehicleLicense"`
	HasPurchaseContract     *bool  `json:"hasPurchaseContract"`
	HasPlateRegistration    *bool  `json:"hasPlateRegistration"`
	HasTransferRegistration *bool  `json:"hasTransferRegistration"`
	Remarks                 string `json:"remarks"`
	Status                  string `json:"status"`
}

// boolOrFalse 讓未提供的證件註記維持 false，避免 nil 與 false 兩種語意混用。
func boolOrFalse(b *bool) bool {
	return b != nil && *b
}

func (f VehicleWriteFields) toInput() app.VehicleInput {
	return app.VehicleInput{
		PlateNo:                   f.PlateNo,
		DisplayName:               f.DisplayName,
		SiteName:                  f.SiteName,
		Brand:                     f.Brand,
		Model:                     f.Model,
		ManufactureYM:             f.ManufactureYM,
		CompulsoryInsuranceExpiry: f.CompulsoryInsuranceExpiry.toTimePtr(),
		PassengerInsuranceExpiry:  f.PassengerInsuranceExpiry.toTimePtr(),
		ThirdPartyInsuranceExpiry: f.ThirdPartyInsuranceExpiry.toTimePtr(),
		LastInspectionDate:        f.LastInspectionDate.toTimePtr(),
		WheelchairAccessible:      f.WheelchairAccessible,
		HasVehicleLicense:         boolOrFalse(f.HasVehicleLicense),
		HasPurchaseContract:       boolOrFalse(f.HasPurchaseContract),
		HasPlateRegistration:      boolOrFalse(f.HasPlateRegistration),
		HasTransferRegistration:   boolOrFalse(f.HasTransferRegistration),
		Remarks:                   f.Remarks,
		Status:                    f.Status,
	}
}

// DriverResponse 代表回傳給前端的司機資料。身分證密文與 HMAC 索引不對外輸出。
type DriverResponse struct {
	ID                     uuid.UUID  `json:"id"`
	Name                   string     `json:"name"`
	NameNormalized         string     `json:"nameNormalized"`
	NationalID             string     `json:"nationalId"`
	Gender                 *string    `json:"gender,omitempty"`
	BirthDate              *time.Time `json:"birthDate,omitempty"`
	HasProfessionalLicense bool       `json:"hasProfessionalLicense"`
	EmploymentDate         *time.Time `json:"employmentDate,omitempty"`
	HasTransferCert        bool       `json:"hasTransferCert"`
	Remarks                *string    `json:"remarks,omitempty"`
	Email                  *string    `json:"email,omitempty"`
	Status                 string     `json:"status"`
	// LicenseClass 為駕照類別代碼（sedan／truck／bus／trailer），未補登時為 null。
	LicenseClass      *string    `json:"licenseClass"`
	LicenseExpiryDate *time.Time `json:"licenseExpiryDate"`
	CreatedAt         time.Time  `json:"createdAt"`
	UpdatedAt         time.Time  `json:"updatedAt"`
}

func newDriverResponse(d app.Driver) DriverResponse {
	return DriverResponse{
		ID:                     d.ID,
		Name:                   d.Name,
		NameNormalized:         d.NameNormalized,
		NationalID:             d.NationalID,
		Gender:                 d.Gender,
		BirthDate:              d.BirthDate,
		HasProfessionalLicense: d.HasProfessionalLicense,
		EmploymentDate:         d.EmploymentDate,
		HasTransferCert:        d.HasTransferCert,
		Remarks:                d.Remarks,
		Email:                  d.Email,
		Status:                 d.Status,
		LicenseClass:           d.LicenseClass,
		LicenseExpiryDate:      d.LicenseExpiryDate,
		CreatedAt:              d.CreatedAt,
		UpdatedAt:              d.UpdatedAt,
	}
}

func newDriverResponses(list []app.Driver) []DriverResponse {
	if list == nil {
		return nil
	}
	out := make([]DriverResponse, 0, len(list))
	for _, d := range list {
		out = append(out, newDriverResponse(d))
	}
	return out
}

// DriverAssignmentResponse 代表司機車輛指派結果。
type DriverAssignmentResponse struct {
	ID             uuid.UUID  `json:"id"`
	DriverID       uuid.UUID  `json:"driverId"`
	DriverName     string     `json:"driverName,omitempty"`
	VehicleID      uuid.UUID  `json:"vehicleId"`
	VehicleName    string     `json:"vehicleName,omitempty"`
	VehiclePlateNo string     `json:"vehiclePlateNo,omitempty"`
	EffectiveFrom  time.Time  `json:"effectiveFrom"`
	EffectiveTo    *time.Time `json:"effectiveTo,omitempty"`
	CreatedAt      time.Time  `json:"createdAt"`
}

func newDriverAssignmentResponse(a app.DriverAssignment) DriverAssignmentResponse {
	return DriverAssignmentResponse{
		ID:             a.ID,
		DriverID:       a.DriverID,
		DriverName:     a.DriverName,
		VehicleID:      a.VehicleID,
		VehicleName:    a.VehicleName,
		VehiclePlateNo: a.VehiclePlateNo,
		EffectiveFrom:  a.EffectiveFrom,
		EffectiveTo:    a.EffectiveTo,
		CreatedAt:      a.CreatedAt,
	}
}

// CreateDriverRequest 代表新增司機請求。
type CreateDriverRequest struct {
	Name                   string     `json:"name" binding:"required"`
	NationalID             string     `json:"nationalId" binding:"required"`
	Email                  *string    `json:"email"`
	LicenseClass           *string    `json:"licenseClass"`
	LicenseExpiryDate      *wireDate  `json:"licenseExpiryDate"`
	Gender                 *string    `json:"gender"`
	BirthDate              *wireDate  `json:"birthDate"`
	HasProfessionalLicense *bool      `json:"hasProfessionalLicense"`
	EmploymentDate         *wireDate  `json:"employmentDate"`
	HasTransferCert        *bool      `json:"hasTransferCert"`
	Remarks                *string    `json:"remarks"`
	VehicleID              *uuid.UUID `json:"vehicleId"`
}

// UpdateDriverRequest 代表更新司機請求，欄位為 nil 表示不變更。
type UpdateDriverRequest struct {
	Name                   *string      `json:"name"`
	NationalID             *string      `json:"nationalId"`
	Email                  *string      `json:"email"`
	Status                 *string      `json:"status"`
	LicenseClass           *string      `json:"licenseClass"`
	LicenseExpiryDate      nullableTime `json:"licenseExpiryDate"`
	Gender                 *string      `json:"gender"`
	BirthDate              nullableTime `json:"birthDate"`
	HasProfessionalLicense *bool        `json:"hasProfessionalLicense"`
	EmploymentDate         nullableTime `json:"employmentDate"`
	HasTransferCert        *bool        `json:"hasTransferCert"`
	Remarks                *string      `json:"remarks"`
}

// nullableTime 用來區分 JSON 欄位「未提供」與「明確給 null」，後者代表要把日期清空。
type nullableTime struct {
	Present bool
	Value   *time.Time
}

// UnmarshalJSON 只在 JSON 帶有該欄位時被呼叫，因此可用來標記欄位存在。
func (n *nullableTime) UnmarshalJSON(data []byte) error {
	n.Present = true
	if string(data) == "null" {
		n.Value = nil
		return nil
	}
	t, err := parseWireDate(data)
	if err != nil {
		return err
	}
	n.Value = t
	return nil
}

// wireDate 解析前端日期選擇器送出的純日期字串（YYYY-MM-DD），並相容完整 RFC3339 時間字串。
type wireDate time.Time

// UnmarshalJSON 見 wireDate 註解。
func (d *wireDate) UnmarshalJSON(data []byte) error {
	t, err := parseWireDate(data)
	if err != nil {
		return err
	}
	if t != nil {
		*d = wireDate(*t)
	} else {
		*d = wireDate{}
	}
	return nil
}

// parseWireDate 依序嘗試純日期與 RFC3339 時間格式解析。支援空字串與 null，此時回傳 (nil, nil)。
func parseWireDate(data []byte) (*time.Time, error) {
	if string(data) == "null" {
		return nil, nil
	}
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, err
	}
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}
	if t, err := rocdate.ParseDate(s); err == nil {
		return &t, nil
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (d *wireDate) toTimePtr() *time.Time {
	if d == nil || time.Time(*d).IsZero() {
		return nil
	}
	t := time.Time(*d)
	return &t
}

func (d wireDate) toTime() time.Time {
	return time.Time(d)
}

// AssignVehicleRequest 代表指派司機車輛請求。指派不再由使用者輸入期間，
// 一律自今日起生效且不設結束日。
type AssignVehicleRequest struct {
	VehicleID uuid.UUID `json:"vehicleId" binding:"required"`
}

// SetVehicleDriversRequest 代表整批設定車輛司機的請求。DriverIDs 為空代表清空該車司機。
type SetVehicleDriversRequest struct {
	DriverIDs     []uuid.UUID `json:"driverIds"`
	EffectiveFrom *wireDate   `json:"effectiveFrom"`
}
