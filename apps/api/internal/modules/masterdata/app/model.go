package app

import (
	"time"

	"github.com/google/uuid"
)

// 本檔的型別是 masterdata 的 application model：不帶任何 struct tag，由 infra 自
// persistence row 轉入、由 transport 轉為 API DTO。

// Site 代表一個服務單位。
type Site struct {
	ID        uuid.UUID
	Name      string
	Address   string
	Region    string
	OpenDays  []int16
	Status    string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// SiteAuditSnapshot 是單位主檔異動的明確快照。
type SiteAuditSnapshot struct {
	ID       uuid.UUID `json:"id"`
	Name     string    `json:"name"`
	Address  string    `json:"address"`
	Region   string    `json:"region"`
	OpenDays []int16   `json:"openDays"`
	Status   string    `json:"status"`
}

// AuditSnapshot 產生單位主檔的明確稽核快照。
func (s Site) AuditSnapshot() SiteAuditSnapshot {
	return SiteAuditSnapshot{
		ID:       s.ID,
		Name:     s.Name,
		Address:  s.Address,
		Region:   s.Region,
		OpenDays: append([]int16(nil), s.OpenDays...),
		Status:   s.Status,
	}
}

// Vehicle 代表一輛接送車輛。Drivers 是該車目前生效的司機，同一台車可以有多位。
type Vehicle struct {
	ID          uuid.UUID
	PlateNo     string
	DisplayName string
	SiteID      *uuid.UUID
	SiteName    string
	// Region 由所屬單位帶出，車輛本身不自存區域；未指定單位時為空字串。
	Region                    string
	Brand                     string
	Model                     string
	ManufactureYM             string
	CompulsoryInsuranceExpiry *time.Time
	PassengerInsuranceExpiry  *time.Time
	ThirdPartyInsuranceExpiry *time.Time
	LastInspectionDate        *time.Time
	WheelchairAccessible      *bool
	Status                    string
	Drivers                   []VehicleDriver
	CreatedAt                 time.Time
	UpdatedAt                 time.Time
}

// VehicleFilter 是車輛清單的查詢條件，零值欄位代表不篩選。
type VehicleFilter struct {
	SiteID *uuid.UUID
	Region string
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
	SiteID                    *uuid.UUID `json:"siteId,omitempty"`
	Brand                     string     `json:"brand"`
	Model                     string     `json:"model"`
	ManufactureYM             string     `json:"manufactureYm"`
	CompulsoryInsuranceExpiry *time.Time `json:"compulsoryInsuranceExpiry,omitempty"`
	PassengerInsuranceExpiry  *time.Time `json:"passengerInsuranceExpiry,omitempty"`
	ThirdPartyInsuranceExpiry *time.Time `json:"thirdPartyInsuranceExpiry,omitempty"`
	LastInspectionDate        *time.Time `json:"lastInspectionDate,omitempty"`
	WheelchairAccessible      *bool      `json:"wheelchairAccessible,omitempty"`
	Status                    string     `json:"status"`
}

// AuditSnapshot 產生車輛主檔的明確稽核快照。
func (v Vehicle) AuditSnapshot() VehicleAuditSnapshot {
	return VehicleAuditSnapshot{
		ID:                        v.ID,
		PlateNo:                   v.PlateNo,
		DisplayName:               v.DisplayName,
		SiteID:                    v.SiteID,
		Brand:                     v.Brand,
		Model:                     v.Model,
		ManufactureYM:             v.ManufactureYM,
		CompulsoryInsuranceExpiry: v.CompulsoryInsuranceExpiry,
		PassengerInsuranceExpiry:  v.PassengerInsuranceExpiry,
		ThirdPartyInsuranceExpiry: v.ThirdPartyInsuranceExpiry,
		LastInspectionDate:        v.LastInspectionDate,
		WheelchairAccessible:      v.WheelchairAccessible,
		Status:                    v.Status,
	}
}

// VehicleDriversAuditSnapshot 是車輛司機集合異動的明確快照。
type VehicleDriversAuditSnapshot struct {
	VehicleID     uuid.UUID   `json:"vehicleId"`
	DriverIDs     []uuid.UUID `json:"driverIds"`
	EffectiveFrom time.Time   `json:"effectiveFrom"`
}

// Driver 代表一位司機。NationalIDCipher 是身分證密文，只在 Reveal 用例中解密，
// 不得離開 application 層。
type Driver struct {
	ID               uuid.UUID
	Name             string
	NameNormalized   string
	NationalIDCipher []byte
	NationalIDHMAC   []byte
	NationalIDMasked string
	Email            *string
	Region           string
	Status           string
	// LicenseClass 為駕照類別代碼，LicenseExpiryDate 為駕照有效日期；兩者皆可為空，代表尚未補登。
	LicenseClass           *string
	LicenseExpiryDate      *time.Time
	Gender                 *string
	BirthDate              *time.Time
	HasProfessionalLicense bool
	EmploymentDate         *time.Time
	HasTransferCert        bool
	InspectionDate         *time.Time
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
	Region                 string     `json:"region"`
	Status                 string     `json:"status"`
	LicenseClass           *string    `json:"licenseClass,omitempty"`
	LicenseExpiryDate      *time.Time `json:"licenseExpiryDate,omitempty"`
	Gender                 *string    `json:"gender,omitempty"`
	BirthDate              *time.Time `json:"birthDate,omitempty"`
	HasProfessionalLicense bool       `json:"hasProfessionalLicense"`
	EmploymentDate         *time.Time `json:"employmentDate,omitempty"`
	HasTransferCert        bool       `json:"hasTransferCert"`
	InspectionDate         *time.Time `json:"inspectionDate,omitempty"`
	Remarks                *string    `json:"remarks,omitempty"`
}

// AuditSnapshot 產生司機主檔的明確稽核快照。
func (d Driver) AuditSnapshot() DriverAuditSnapshot {
	return DriverAuditSnapshot{
		ID:                     d.ID,
		Name:                   d.Name,
		NationalIDMasked:       d.NationalIDMasked,
		Region:                 d.Region,
		Status:                 d.Status,
		LicenseClass:           d.LicenseClass,
		LicenseExpiryDate:      d.LicenseExpiryDate,
		Gender:                 d.Gender,
		BirthDate:              d.BirthDate,
		HasProfessionalLicense: d.HasProfessionalLicense,
		EmploymentDate:         d.EmploymentDate,
		HasTransferCert:        d.HasTransferCert,
		InspectionDate:         d.InspectionDate,
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

// Region 代表一個服務區域。
type Region struct {
	ID          uuid.UUID
	Name        string
	Description string
	Status      string
	SortOrder   int
	CreatedAt   time.Time
	UpdatedAt   time.Time
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

// RegionSnapshot 是寫入稽核日誌的區域快照。json tag 必須與歷史紀錄一致，否則同
// 一張表會同時存在兩種欄位命名。
type RegionSnapshot struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Status      string    `json:"status"`
	SortOrder   int       `json:"sortOrder"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// Snapshot 產生供稽核日誌保存的區域快照。
func (r Region) Snapshot() RegionSnapshot {
	return RegionSnapshot{
		ID:          r.ID,
		Name:        r.Name,
		Description: r.Description,
		Status:      r.Status,
		SortOrder:   r.SortOrder,
		CreatedAt:   r.CreatedAt,
		UpdatedAt:   r.UpdatedAt,
	}
}
