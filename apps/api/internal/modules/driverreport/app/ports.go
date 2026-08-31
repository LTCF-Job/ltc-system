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
	UpdateColumnMappingByID(ctx context.Context, colID, status string, caseID *string, legSeq *int16) error
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

// RideIngestor 是本模組與搭乘紀錄模組之間的匯報資料邊界，由擁有搭乘紀錄的模組實作。
// IngestSubmission 回傳實際寫入的搭乘紀錄筆數；ClearImportedDates 回傳清除的提交筆數。
type RideIngestor interface {
	IngestSubmission(ctx context.Context, formID, vehicleID uuid.UUID, s Submission) (int, error)
	ClearImportedDates(ctx context.Context, formID uuid.UUID, dates []time.Time) (int, error)
	ListImportedMonths(ctx context.Context) ([]ImportedMonth, error)
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
