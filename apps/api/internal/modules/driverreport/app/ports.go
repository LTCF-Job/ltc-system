package app

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// FormStore 定義匯報表與欄位對應的資料存取邊界。
type FormStore interface {
	ListForms(ctx context.Context) ([]ReportForm, error)
	GetForm(ctx context.Context, formID uuid.UUID) (*ReportForm, error)
	CreateForm(ctx context.Context, id, vehicleID uuid.UUID, title string) (uuid.UUID, error)
	DeleteForm(ctx context.Context, formID uuid.UUID) error
	ListColumnsWithMapping(ctx context.Context, formID, mappingStatus string) ([]ColumnMapping, error)
	UpsertColumns(ctx context.Context, formID uuid.UUID, drafts []ColumnDraft) error
	// UpdateColumnMappingByID 更新欄位對應，並回傳更新前的狀態供呼叫端判斷是否為
	// 「剛從待維護變成已對應」，藉此決定是否要觸發回填搭乘紀錄。
	UpdateColumnMappingByID(ctx context.Context, colID, status string, caseID *string, legSeq *int16) (formID uuid.UUID, columnHeader string, columnIndex int, previousStatus string, err error)
	UpdateColumnMappingByHeader(ctx context.Context, formID uuid.UUID, header, status string, caseID *string, legSeq *int16) error
	MarkImported(ctx context.Context, formID uuid.UUID, importedAt time.Time) error
}

// SpreadsheetReader 將上傳的 .xlsx 位元組解碼為逐工作表的儲存格文字。
type SpreadsheetReader interface {
	ReadTables(data []byte) ([][][]string, []string, error)
}

// TemplateRenderer 產生匯報表空白範本；caseColumns 是已對應個案的趟次欄表頭。
type TemplateRenderer interface {
	RenderDriverReportTemplate(vehicleName string, caseColumns []string) ([]byte, error)
}

// CaseRef 是推薦個案對應時需要的最小個案資訊。
type CaseRef struct {
	ID             string
	Name           string
	NameNormalized string
}

// CaseLookup 提供可被匯報欄位對應到的個案清單，由擁有個案主檔的模組實作。
type CaseLookup interface {
	ListActiveCases(ctx context.Context) ([]CaseRef, error)
}

// DriverRef 是由姓名推導出的司機。
type DriverRef struct {
	ID   uuid.UUID
	Name string
}

// DriverResolver 由姓名比對司機主檔，由擁有司機主檔的模組實作。
type DriverResolver interface {
	GetByNameNormalized(ctx context.Context, nameNorm string) (*DriverRef, error)
}

// SubmissionAnswerRow 是一筆既有回報留在 payload 的完整原始儲存格文字，供彙整待維護
// 清單時比對哪些欄位這一列「有回報」但仍待對應個案。
type SubmissionAnswerRow struct {
	SubmissionID uuid.UUID
	FormID       uuid.UUID
	FormTitle    string
	VehicleName  string
	ServiceDate  time.Time
	Answers      map[string]string
}

// UnmatchedDriverSubmission 是一筆駕駛人姓名比對不到司機主檔的既有回報。
type UnmatchedDriverSubmission struct {
	SubmissionID  uuid.UUID
	FormID        uuid.UUID
	FormTitle     string
	VehicleName   string
	ServiceDate   time.Time
	DriverNameRaw string
}

// Submission 是一列匯報資料；Answers 以欄位表頭為鍵，值為原始儲存格文字。
type Submission struct {
	ServiceDate time.Time
	SubmittedAt time.Time
	DriverRaw   string
	DriverID    *uuid.UUID
	Remark      string
	Answers     map[string]string
}

// ImportedMonth 是某份匯報表在某個月已匯入的提交統計。
type ImportedMonth struct {
	FormID          uuid.UUID
	YearMonth       string
	SubmissionCount int
	LastImportedAt  time.Time
}

// MonthSubmissionDetail 是某份匯報表某個月一天的完整回報內容，供總覽頁鑽取查看逐日
// 原始資料，不需重新開啟原始檔案。
type MonthSubmissionDetail struct {
	ServiceDate   time.Time
	DriverNameRaw string
	Remark        string
	Answers       map[string]string
}

// MonthRideEntry 是某份匯報表某個月展開後的一筆個案搭乘紀錄，供總覽頁鑽取查看這個月
// 實際寫入了哪些個案、哪些趟次。
type MonthRideEntry struct {
	CaseID      uuid.UUID
	CaseName    string
	ServiceDate time.Time
	LegSeq      int16
	Reported    string
	DriverID    *uuid.UUID
	DriverName  string
	VehicleID   uuid.UUID
}

// RideIngestor 是本模組與搭乘紀錄模組之間的匯報資料邊界，由擁有搭乘紀錄的模組實作。
// IngestSubmission 回傳實際寫入的搭乘紀錄筆數；ClearImportedDates 回傳清除的提交筆數。
type RideIngestor interface {
	IngestSubmission(ctx context.Context, formID, vehicleID uuid.UUID, s Submission) (int, error)
	ClearImportedDates(ctx context.Context, formID uuid.UUID, dates []time.Time) (int, error)
	ListImportedMonths(ctx context.Context) ([]ImportedMonth, error)
	// BackfillColumn 用某欄位既有回報中已存的原始儲存格文字補寫搭乘紀錄，回傳補寫筆數。
	// 用於欄位從待維護變成已對應時，不需要使用者重新上傳原始檔案。
	BackfillColumn(ctx context.Context, formID, vehicleID uuid.UUID, columnHeader string, columnIndex int, caseID uuid.UUID, legSeq int16) (int, error)
	// ListSubmissionsForForms 取出指定表單目前存在的所有回報列與其完整原始儲存格文字，
	// 供彙整待維護清單時比對哪些欄位這一列「有回報」但仍待對應個案。
	ListSubmissionsForForms(ctx context.Context, formIDs []uuid.UUID) ([]SubmissionAnswerRow, error)
	// ListUnmatchedDriverSubmissions 取出目前駕駛人姓名比對不到司機主檔的既有回報。
	ListUnmatchedDriverSubmissions(ctx context.Context) ([]UnmatchedDriverSubmission, error)
	// BackfillDriver 把姓名正規化後相符、目前比對不到司機主檔的既有回報一次回填為指定
	// 司機，回傳實際回填的提交筆數與涉及的服務日期（去重）；不需要重新上傳原始檔案。
	BackfillDriver(ctx context.Context, driverNameRaw string, driverID uuid.UUID) (int, []time.Time, error)
	// ListSubmissionsForFormMonth 取出某份匯報表在指定月份區間內的逐日原始回報，供總覽頁鑽取單一月份的完整內容。
	ListSubmissionsForFormMonth(ctx context.Context, formID uuid.UUID, monthStart, monthEnd time.Time) ([]MonthSubmissionDetail, error)
	// ListRideEntriesForFormMonth 取出某份匯報表在指定月份區間內展開後的個案搭乘紀錄，供總覽頁鑽取單一月份實際寫入了哪些個案與趟次。
	ListRideEntriesForFormMonth(ctx context.Context, formID uuid.UUID, monthStart, monthEnd time.Time) ([]MonthRideEntry, error)
}

// AttendanceRegistrar 是本模組與司機出勤模組之間的邊界，由擁有出勤紀錄的模組實作。
// 匯報比對到司機時，用這個邊界同步該司機當天的出勤登記，不需要使用者另外去司機
// 月曆手動登記。
type AttendanceRegistrar interface {
	// SyncFromImport 依比對到的司機與服務日期同步出勤登記；當天已有不同的人工登記時
	// 不覆蓋，改由出勤模組記一筆待維護衝突讓使用者決定。
	SyncFromImport(ctx context.Context, driverID uuid.UUID, serviceDate time.Time) error
}

// TxRunner 讓一次匯入的清除與重寫落在同一個資料庫交易內。
type TxRunner interface {
	WithTx(ctx context.Context, fn func(ctx context.Context) error) error
}

// Actor 代表發動匯入的操作者與來源資訊，供稽核留痕使用。
type Actor struct {
	ActorID   uuid.UUID
	ActorRole string
	IPAddress string
	UserAgent string
}

// AuditEntry 是本模組寫入稽核日誌的內容。
type AuditEntry struct {
	ActorID    *uuid.UUID
	ActorRole  *string
	Action     string
	EntityType string
	EntityID   *string
	AfterData  interface{}
	IPAddress  *string
	UserAgent  *string
}

// AuditWriter 定義匯入留痕的寫入邊界。
type AuditWriter interface {
	Write(ctx context.Context, e AuditEntry) error
}
