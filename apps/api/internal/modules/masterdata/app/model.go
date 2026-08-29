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
	Code      string
	Name      string
	Address   string
	Region    string
	OpenDays  []int16
	Status    string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Vehicle 代表一輛接送車輛。
type Vehicle struct {
	ID          uuid.UUID
	PlateNo     string
	DisplayName string
	Region      string
	Status      string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// Driver 代表一位司機。NationalIDCipher 是身分證密文，只在 Reveal 用例中解密，
// 不得離開 application 層。
type Driver struct {
	ID               uuid.UUID
	Code             string
	Name             string
	NameNormalized   string
	NationalIDCipher []byte
	NationalIDHMAC   []byte
	NationalIDMasked string
	Email            *string
	Region           string
	Status           string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// DriverAssignment 代表司機與車輛在一段期間內的指派關係。
type DriverAssignment struct {
	ID             uuid.UUID
	DriverID       uuid.UUID
	DriverName     string
	VehicleID      uuid.UUID
	VehicleName    string
	VehiclePlateNo string
	IsPrimary      bool
	EffectiveFrom  time.Time
	EffectiveTo    *time.Time
	CreatedAt      time.Time
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
