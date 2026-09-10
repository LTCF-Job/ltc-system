package app

import (
	"time"

	"github.com/google/uuid"
)

// 本檔的型別是 masterdata 的 application model：不帶任何 struct tag，由 infra 自
// persistence row 轉入、由 transport 轉為 API DTO。

// Site 代表一個服務據點。
type Site struct {
	ID        uuid.UUID
	Name      string
	Address   string
	Region    string
	Remarks   string
	Status    string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// SiteAuditSnapshot 是據點主檔異動的明確快照。
type SiteAuditSnapshot struct {
	ID      uuid.UUID `json:"id"`
	Name    string    `json:"name"`
	Address string    `json:"address"`
	Region  string    `json:"region"`
	Remarks string    `json:"remarks,omitempty"`
	Status  string    `json:"status"`
}

// AuditSnapshot 產生據點主檔的明確稽核快照。
func (s Site) AuditSnapshot() SiteAuditSnapshot {
	return SiteAuditSnapshot{
		ID:      s.ID,
		Name:    s.Name,
		Address: s.Address,
		Region:  s.Region,
		Remarks: s.Remarks,
		Status:  s.Status,
	}
}

// Vehicle 代表一輛接送車輛。Drivers 是該車目前生效的司機，同一台車可以有多位。
type Vehicle struct {
	ID          uuid.UUID
	PlateNo     string
	DisplayName string
	// SiteName 是車輛自己的據點文字註記，非必填，不關聯據點主檔。
	SiteName                  string
	Brand                     string
	Model                     string
	ManufactureYM             string
	CompulsoryInsuranceExpiry *time.Time
	PassengerInsuranceExpiry  *time.Time
	ThirdPartyInsuranceExpiry *time.Time
	LastInspectionDate        *time.Time
	WheelchairAccessible      *bool
	// 四項證件持有註記：行照、汽車買賣合約書、領牌登記書、異動登記書。
	HasVehicleLicense       bool
	HasPurchaseContract     bool
	HasPlateRegistration    bool
	HasTransferRegistration bool
	Remarks                 string
	Status                  string
	Drivers                 []VehicleDriver
	CreatedAt               time.Time
	UpdatedAt               time.Time
}

// VehicleFilter 是車輛清單的查詢條件，零值欄位代表不篩選。
type VehicleFilter struct {
	Q      string
	Status string
}

// VehicleDriver 是掛在車輛上的司機摘要，只帶識別用欄位。
type VehicleDriver struct {
	ID   uuid.UUID
	Name string
}

// VehicleAuditSnapshot 是車輛主檔稽核快照；不直接序列化 Vehicle，避免把查詢組裝
// 的司機清單或其他非 mutation 欄位寫入 audit_log。
type VehicleAuditSnapshot struct {
	ID                        uuid.UUID  `json:"id"`
	PlateNo                   string     `json:"plateNo"`
	DisplayName               string     `json:"displayName"`
	SiteName                  string     `json:"siteName,omitempty"`
	Brand                     string     `json:"brand"`
	Model                     string     `json:"model"`
	ManufactureYM             string     `json:"manufactureYm"`
	CompulsoryInsuranceExpiry *time.Time `json:"compulsoryInsuranceExpiry,omitempty"`
	PassengerInsuranceExpiry  *time.Time `json:"passengerInsuranceExpiry,omitempty"`
	ThirdPartyInsuranceExpiry *time.Time `json:"thirdPartyInsuranceExpiry,omitempty"`
	LastInspectionDate        *time.Time `json:"lastInspectionDate,omitempty"`
	WheelchairAccessible      *bool      `json:"wheelchairAccessible,omitempty"`
	HasVehicleLicense         bool       `json:"hasVehicleLicense"`
	HasPurchaseContract       bool       `json:"hasPurchaseContract"`
	HasPlateRegistration      bool       `json:"hasPlateRegistration"`
	HasTransferRegistration   bool       `json:"hasTransferRegistration"`
	Remarks                   string     `json:"remarks,omitempty"`
	Status                    string     `json:"status"`
}

// AuditSnapshot 產生車輛主檔的明確稽核快照。
func (v Vehicle) AuditSnapshot() VehicleAuditSnapshot {
	return VehicleAuditSnapshot{
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
	}
}

// VehicleDriversAuditSnapshot 是車輛司機集合異動的明確快照。
type VehicleDriversAuditSnapshot struct {
	VehicleID     uuid.UUID   `json:"vehicleId"`
	DriverIDs     []uuid.UUID `json:"driverIds"`
	EffectiveFrom time.Time   `json:"effectiveFrom"`
}

// Driver 代表一位司機。NationalIDCipher 是身分證密文，僅於查詢時解密給 NationalID，
// 密文本身不得離開 application 層。
type Driver struct {
	ID               uuid.UUID
	Name             string
	NameNormalized   string
	NationalIDCipher []byte
	NationalIDHMAC   []byte
	NationalIDMasked string
	NationalID       string
	Email            *string
	Status           string
	// LicenseClass 為駕照類別代碼，LicenseExpiryDate 為駕照有效日期；兩者皆可為空，代表尚未補登。
	LicenseClass           *string
	LicenseExpiryDate      *time.Time
	Gender                 *string
	BirthDate              *time.Time
	HasProfessionalLicense bool
	EmploymentDate         *time.Time
	HasTransferCert        bool
	Remarks                *string
	CreatedAt              time.Time
	UpdatedAt              time.Time
}

// DriverAuditSnapshot 是司機主檔稽核快照；身分證只保留已遮罩值，不保存密文、HMAC
// 或明文。
type DriverAuditSnapshot struct {
	ID                     uuid.UUID  `json:"id"`
	Name                   string     `json:"name"`
	NationalIDMasked       string     `json:"nationalIdMasked,omitempty"`
	Status                 string     `json:"status"`
	LicenseClass           *string    `json:"licenseClass,omitempty"`
	LicenseExpiryDate      *time.Time `json:"licenseExpiryDate,omitempty"`
	Gender                 *string    `json:"gender,omitempty"`
	BirthDate              *time.Time `json:"birthDate,omitempty"`
	HasProfessionalLicense bool       `json:"hasProfessionalLicense"`
	EmploymentDate         *time.Time `json:"employmentDate,omitempty"`
	HasTransferCert        bool       `json:"hasTransferCert"`
	Remarks                *string    `json:"remarks,omitempty"`
}

// AuditSnapshot 產生司機主檔的明確稽核快照。
func (d Driver) AuditSnapshot() DriverAuditSnapshot {
	return DriverAuditSnapshot{
		ID:                     d.ID,
		Name:                   d.Name,
		NationalIDMasked:       d.NationalIDMasked,
		Status:                 d.Status,
		LicenseClass:           d.LicenseClass,
		LicenseExpiryDate:      d.LicenseExpiryDate,
		Gender:                 d.Gender,
		BirthDate:              d.BirthDate,
		HasProfessionalLicense: d.HasProfessionalLicense,
		EmploymentDate:         d.EmploymentDate,
		HasTransferCert:        d.HasTransferCert,
		Remarks:                d.Remarks,
	}
}

// DriverAssignment 代表司機與車輛在一段期間內的指派關係。一位司機同期只會有一台車，
// 因此不再區分主要與備援車輛。
type DriverAssignment struct {
	ID             uuid.UUID
	DriverID       uuid.UUID
	DriverName     string
	VehicleID      uuid.UUID
	VehicleName    string
	VehiclePlateNo string
	EffectiveFrom  time.Time
	EffectiveTo    *time.Time
	CreatedAt      time.Time
}

// DriverAssignmentAuditSnapshot 是司機車輛指派異動的明確稽核快照。
type DriverAssignmentAuditSnapshot struct {
	ID            uuid.UUID  `json:"id"`
	DriverID      uuid.UUID  `json:"driverId"`
	VehicleID     uuid.UUID  `json:"vehicleId"`
	EffectiveFrom time.Time  `json:"effectiveFrom"`
	EffectiveTo   *time.Time `json:"effectiveTo,omitempty"`
}

// AuditSnapshot 產生司機車輛指派的明確稽核快照。
func (a DriverAssignment) AuditSnapshot() DriverAssignmentAuditSnapshot {
	return DriverAssignmentAuditSnapshot{
		ID:            a.ID,
		DriverID:      a.DriverID,
		VehicleID:     a.VehicleID,
		EffectiveFrom: a.EffectiveFrom,
		EffectiveTo:   a.EffectiveTo,
	}
}

// AuditEntry 代表一筆待寫入的稽核紀錄。BeforeData 與 AfterData 是會被序列化進
// audit_log JSONB 欄位的快照，其形狀即為稽核紀錄的資料契約。
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
