package app

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

var ErrLookupNotFound = errors.New("lookup not found")

// SiteRef 是匯入比對據點時需要的最小資訊。
type SiteRef struct {
	ID   uuid.UUID
	Name string
}

// VehicleRef 是匯入比對車輛時需要的最小資訊。
type VehicleRef struct {
	ID uuid.UUID
}

// CaregiverRef 是匯入比對照護人員時需要的最小資訊；Type 供同名多筆時消歧。
type CaregiverRef struct {
	ID   uuid.UUID
	Name string
	Type string
}

// SiteLookup 提供以名稱或區域比對據點的查詢。
type SiteLookup interface {
	GetByName(ctx context.Context, name string) (*SiteRef, error)
	List(ctx context.Context, page, pageSize int) ([]SiteRef, error)
}

// VehicleLookup 提供以顯示名稱比對車輛的查詢。
// 接送車輛欄位已自匯入範本與匯出移除，commit 路徑不再呼叫這個查詢。
type VehicleLookup interface {
	GetByDisplayName(ctx context.Context, displayName string) (*VehicleRef, error)
}

// CaregiverLookup 以姓名取回同名的照護人員清單；同名多筆時的消歧規則屬於匯入政策，留在 app 層。
type CaregiverLookup interface {
	FindByName(ctx context.Context, name string) ([]CaregiverRef, error)
}

// TransportPreferenceWriter 以 PUT 完整替換個案的去回程車輛偏好。據點已改由個案本身
// 持有，隨 NewCase 一併寫入。接送車輛欄位已自匯入範本與匯出移除，commit 路徑不再呼叫。
type TransportPreferenceWriter interface {
	UpsertTransportPreference(ctx context.Context, caseID uuid.UUID, outboundVehicleID, inboundVehicleID *uuid.UUID, outboundVehicleNameRaw, inboundVehicleNameRaw string) error
}

// NewCase 是建立個案所需的輸入，僅 Name 為必要欄位。AllowInvalidNationalID 讓身分證字號
// 格式錯誤時不擋列，改由 casemgmt 標記待維護；BirthDateRaw 是生日解析失敗時的原始字串。
// SiteID 為 nil 且 SiteNameRaw 有值時，表示據點名稱未比對到主檔，待人工於待維護畫面補齊；
// CaregiverID 為 nil 而 CareContactName 有值時同理，代表照護人員未比對到主檔。
type NewCase struct {
	ID                     uuid.UUID
	Name                   string
	NationalID             string
	AllowInvalidNationalID bool
	HouseholdType          *string
	Gender                 *string
	BirthDate              *time.Time
	BirthDateRaw           *string
	CareContactRole        *string
	CareContactName        *string
	RegisteredAddress      *string
	HomeAddress            *string
	ServiceCategory        int
	ServiceUsageType       int
	Status                 string
	Remarks                *string
	SiteID                 *uuid.UUID
	SiteNameRaw            string
	CaregiverID            *uuid.UUID
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

// CaseImportIdempotencyStore 以檔案雜湊與來源列識別碼抑制重試造成的重複建立。
// 實作必須在目前列的 transaction 內以唯一鍵 claim，避免併發匯入穿透。
type CaseImportIdempotencyStore interface {
	ClaimCaseImportRow(ctx context.Context, fileHash, rowKey string, caseID uuid.UUID) (bool, error)
	// 唯讀查詢，供重複判斷之前先短路掉已建立的列；claim 仍負責併發保護。
	IsCaseImportRowCommitted(ctx context.Context, fileHash, rowKey string) (bool, error)
}

// StageDuplicateCandidate 是疑似重複個案暫存所需的完整列輸入。
type StageDuplicateCandidate struct {
	RowIndex               int
	SheetName              string
	Name                   string
	NationalID             string
	HouseholdType          *string
	Gender                 *string
	BirthDate              *time.Time
	BirthDateRaw           *string
	CareContactRole        *string
	CareContactName        *string
	RegisteredAddress      *string
	HomeAddress            *string
	ServiceCategory        int
	ServiceUsageType       int
	Remarks                *string
	SiteID                 *uuid.UUID
	SiteNameRaw            string
	CaregiverID            *uuid.UUID
	OutboundVehicleID      *uuid.UUID
	OutboundVehicleNameRaw string
	InboundVehicleID       *uuid.UUID
	InboundVehicleNameRaw  string
	DuplicateCaseID        uuid.UUID
}

// DuplicateCandidateStager 讓匯入在偵測到疑似重複個案時，把整列資料交給擁有加密
// 金鑰與個案能力的模組存入待裁決暫存，不直接建立個案。
type DuplicateCandidateStager interface {
	// StageDuplicateRow 寫入一筆暫存列；同一 (fileHash, rowKey) 已存在時回傳 alreadyStaged=true。
	StageDuplicateRow(ctx context.Context, fileHash, rowKey string, in StageDuplicateCandidate) (id uuid.UUID, alreadyStaged bool, err error)
}
