package app

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// SiteRef 是匯入比對據點時需要的最小資訊。
type SiteRef struct {
	ID   uuid.UUID
	Name string
}

// VehicleRef 是匯入比對車輛時需要的最小資訊。
type VehicleRef struct {
	ID uuid.UUID
}

// SiteLookup 提供以名稱或區域比對據點的查詢。
type SiteLookup interface {
	GetByName(ctx context.Context, name string) (*SiteRef, error)
	List(ctx context.Context, region string, page, pageSize int) ([]SiteRef, error)
}

// VehicleLookup 提供以顯示名稱比對車輛的查詢。
type VehicleLookup interface {
	GetByDisplayName(ctx context.Context, displayName string) (*VehicleRef, error)
}

// TransportPreferenceWriter 寫入個案的據點與去回程車輛偏好。
type TransportPreferenceWriter interface {
	UpsertTransportPreference(ctx context.Context, caseID, siteID, outboundVehicleID, inboundVehicleID uuid.UUID) error
}

// NewCase 是建立個案所需的輸入，欄位對應個案主檔的必要與選填資料。
type NewCase struct {
	Code              string
	Name              string
	NationalID        string
	HouseholdType     *string
	Gender            *string
	BirthDate         *time.Time
	CareContactRole   *string
	CareContactName   *string
	RegisteredAddress *string
	HomeAddress       string
	Region            string
	ClaimStartDate    time.Time
	ServiceCategory   int
	ServiceUsageType  int
	Status            string
}

// NewScheduleLeg 是單一趟次的排班設定。
type NewScheduleLeg struct {
	LegSeq     int16
	Direction  string
	DepartTime string
}

// NewSchedule 是建立個案排班所需的輸入。
type NewSchedule struct {
	CaseID             uuid.UUID
	SiteID             uuid.UUID
	EffectiveFrom      time.Time
	Weekdays           []int16
	TripPattern        int16
	UnitPrice          float64
	DistanceKM         float64
	ServiceDurationMin int16
	ServiceCode        string
	Note               *string
	Legs               []NewScheduleLeg
}

// Actor 代表發動匯入的操作者與來源資訊，供稽核留痕使用。
type Actor struct {
	ActorID   uuid.UUID
	ActorRole string
	IPAddress string
	UserAgent string
}

// CaseRegistrar 是匯入寫入個案主檔與排班的邊界，由擁有個案能力的模組實作。
type CaseRegistrar interface {
	CreateCase(ctx context.Context, in NewCase, actor Actor) (uuid.UUID, error)
	CreateSchedule(ctx context.Context, in NewSchedule, actor Actor) error
	RecordSkipped(ctx context.Context, row CaseImportSkippedRow, actor Actor)
}

// SpreadsheetReader 將上傳的 Excel 位元組解碼為逐工作表的儲存格文字。
type SpreadsheetReader interface {
	ReadTables(data []byte) (tables [][][]string, sheetNames []string, err error)
}

// TemplateRenderer 產生批次匯入的 Excel 範本位元組。
type TemplateRenderer interface {
	RenderCaseImportTemplate() ([]byte, error)
}

// TxRunner 讓單列匯入的多次寫入落在同一個資料庫交易內。
type TxRunner interface {
	WithTx(ctx context.Context, fn func(ctx context.Context) error) error
}
