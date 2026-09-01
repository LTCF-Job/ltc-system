package app

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// SiteRef 是匯入比對單位時需要的最小資訊。
type SiteRef struct {
	ID   uuid.UUID
	Name string
}

// VehicleRef 是匯入比對車輛時需要的最小資訊。
type VehicleRef struct {
	ID uuid.UUID
}

// SiteLookup 提供以名稱或區域比對單位的查詢。
type SiteLookup interface {
	GetByName(ctx context.Context, name string) (*SiteRef, error)
	List(ctx context.Context, region string, page, pageSize int) ([]SiteRef, error)
}

// VehicleLookup 提供以顯示名稱比對車輛的查詢。
type VehicleLookup interface {
	GetByDisplayName(ctx context.Context, displayName string) (*VehicleRef, error)
}

// TransportPreferenceWriter 寫入個案的單位與去回程車輛偏好。nil 的 ID 表示該欄位
// 維持現況，raw name 於對應 ID 為 nil 時保留原始名稱待人工關聯。
type TransportPreferenceWriter interface {
	UpsertTransportPreference(ctx context.Context, caseID uuid.UUID, siteID, outboundVehicleID, inboundVehicleID *uuid.UUID, siteNameRaw, outboundVehicleNameRaw, inboundVehicleNameRaw string) error
}

// NewCase 是建立個案所需的輸入，僅 Name 為必要欄位。
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
	HomeAddress       *string
	Region            *string
	ServiceCategory   int
	ServiceUsageType  int
	Status            string
	Remarks           *string
}

// Actor 代表發動匯入的操作者與來源資訊，供稽核留痕使用。
type Actor struct {
	ActorID   uuid.UUID
	ActorRole string
	IPAddress string
	UserAgent string
}

// DuplicateRef 是查重比對到的既有個案基本資訊，供匯入預覽提示使用。
type DuplicateRef struct {
	CaseID   uuid.UUID
	CaseCode string
	CaseName string
}

// CaseDuplicateFinder 是匯入 dry-run 階段查重的邊界，由擁有個案能力的模組實作。
type CaseDuplicateFinder interface {
	FindDuplicate(ctx context.Context, nationalID, name string) (*DuplicateRef, error)
}

// CaseRegistrar 是匯入寫入個案主檔的邊界，由擁有個案能力的模組實作。
type CaseRegistrar interface {
	CreateCase(ctx context.Context, in NewCase, actor Actor) (uuid.UUID, error)
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
