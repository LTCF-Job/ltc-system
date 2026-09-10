package app

import (
	"time"

	"github.com/google/uuid"
)

// 本檔是政府申報匯出的應用層模型。與 readmodel.go 的純查詢投影不同，這些型別會寫入
// export_jobs／export_lines／export_job_files，形狀由業務規格決定而非單一查詢決定。

// GovClaimMode 代表匯出檔案模式：逐案下載或打包成單一壓縮檔。
type GovClaimMode string

const (
	GovClaimModeDirect GovClaimMode = "direct"
	GovClaimModeZip    GovClaimMode = "zip"
)

// 匯出工作狀態，對應 export_jobs.status 的 CHECK 約束。
const (
	ExportStatusRunning   = "running"
	ExportStatusSucceeded = "succeeded"
	ExportStatusFailed    = "failed"
)

// 申報列缺漏資料的代碼。這些欄位在申報檔留白，該列與其餘資料照樣匯出，
// 只有 GapReasonBuildRowFailed 是真的組不出列（如日期無法換算民國年）才會少一列。
const (
	GapReasonNoScheduleLeg     = "NO_SCHEDULE_LEG"
	GapReasonNoDepartTime      = "NO_DEPART_TIME"
	GapReasonNoDriver          = "NO_DRIVER"
	GapReasonNoServiceCategory = "NO_SERVICE_CATEGORY"
	GapReasonNoUsageType       = "NO_SERVICE_USAGE_TYPE"
	GapReasonNoUnitPrice       = "NO_UNIT_PRICE"
	GapReasonNoNationalID      = "NO_NATIONAL_ID"
	GapReasonNoAddress         = "NO_ADDRESS"
	GapReasonNoDistance        = "NO_DISTANCE"
	GapReasonNoPlateNo         = "NO_PLATE_NO"
	GapReasonNoServiceCode     = "NO_SERVICE_CODE"
	GapReasonBuildRowFailed    = "BUILD_ROW_FAILED"
)

// GovClaimSource 代表組出單列申報資料所需的原始查詢結果。
// 指標欄位代表來源可能缺漏，由 service 決定該欄在申報檔留白並計入資料缺漏清單。
type GovClaimSource struct {
	CaseID               uuid.UUID
	CaseName             string
	CaseNationalIDCipher []byte
	CaseNationalIDMasked string
	HomeAddress          string
	ServiceCategory      *int
	ServiceUsageType     *int

	ServiceDate    time.Time
	LegSeq         int16
	NotClaimedAA09 bool
	Direction      *string
	DepartTime     *string
	DurationMin    *int

	ServiceCode string
	UnitPrice   float64
	DistanceKM  float64
	SiteAddress string
	PlateNo     string

	DriverID               *uuid.UUID
	DriverNationalIDCipher []byte
}

// ClaimLinePayload 是 export_lines.raw_payload 的形狀。
// Cells[0]（個案身分證）與 Cells[6]（服務人員身分證）一律留空，重繪時才由 cipher 解密補回，
// 避免明文身分證隨快照落到資料庫。
type ClaimLinePayload struct {
	Cells     [33]interface{} `json:"cells"`
	DriverID  *uuid.UUID      `json:"driverId,omitempty"`
	Direction string          `json:"direction"`
	LegSeq    int16           `json:"legSeq"`
}

// ExportLine 代表 export_lines 的一列申報快照。
type ExportLine struct {
	LineNo           int
	CaseID           uuid.UUID
	NationalIDMasked string
	ServiceDateROC   int
	Payload          ClaimLinePayload
}

// GovClaimCaseFile 代表單一個案單一月份的申報工作簿。
// Bytes 只在產生當下有值；讀取歷史工作時為 nil，需由快照重繪。
type GovClaimCaseFile struct {
	CaseID   uuid.UUID
	CaseName string
	FileName string
	RowCount int
	Checksum string
	Bytes    []byte
}

// ClaimDataGap 代表某個案有幾列的某個欄位因來源缺漏而在申報檔留白。
type ClaimDataGap struct {
	CaseID   uuid.UUID
	CaseName string
	Reason   string
	Count    int
}

// GovClaimJob 代表一次政府申報匯出工作。
// DataGaps 只在建立當下有值，不會寫入資料庫，因此讀取歷史工作時為空。
type GovClaimJob struct {
	ID            uuid.UUID
	JobType       string
	PeriodYM      string
	Mode          GovClaimMode
	Status        string
	TotalCases    int
	TotalRows     int
	Files         []GovClaimCaseFile
	DataGaps      []ClaimDataGap
	ErrorMessage  string
	CreatedBy     uuid.UUID
	CreatedByName string
	CreatedAt     time.Time
	FinishedAt    *time.Time
}

// ExportJobCreate 代表建立匯出工作時要寫入 export_jobs 的欄位。
type ExportJobCreate struct {
	JobType       string
	PeriodYM      string
	Format        string
	CaseIDs       []uuid.UUID
	Precheck      *PrecheckReport
	CreatedBy     uuid.UUID
	CreatedByName string
}

// AuditEntry 代表一筆待寫入的稽核紀錄。AfterData 是會被序列化進 audit_log JSONB 欄位的快照，
// 其形狀即為稽核紀錄的資料契約。
type AuditEntry struct {
	ActorID    *uuid.UUID
	ActorRole  *string
	Action     string
	EntityType string
	EntityID   *string
	BeforeData interface{}
	AfterData  interface{}
}

// ExportJobAuditSnapshot 是寫入稽核日誌的匯出工作快照，json tag 需與既有稽核紀錄慣例一致。
// Cases 逐案列出這次實際匯出了哪些個案的哪些檔案，讓稽核紀錄看得出「匯出的內容」而不只是統計數字。
type ExportJobAuditSnapshot struct {
	Status     string                   `json:"status"`
	PeriodYM   string                   `json:"periodYm"`
	Mode       string                   `json:"mode"`
	TotalCases int                      `json:"totalCases"`
	TotalRows  int                      `json:"totalRows"`
	Scope      ExportScopeSnapshot      `json:"scope"`
	Cases      []ExportJobAuditCaseFile `json:"cases"`
}

// ExportScopeSnapshot 記錄這次匯出是用什麼條件選出個案的，只寫入稽核日誌。
// 沒有它就分不出「使用者逐案勾了 62 個人」與「使用者選了一個區域剛好有 62 個人」，
// 事後追查匯錯範圍時差很多。
type ExportScopeSnapshot struct {
	// Type 為 cases（逐案勾選，預設）或 region（依區域批次展開）。
	Type    string   `json:"type"`
	Regions []string `json:"regions,omitempty"`
}

// ExportScopeTypeCases 代表逐案勾選；零值即為此模式。
const ExportScopeTypeCases = "cases"

// ExportScopeTypeRegion 代表依區域批次展開。
const ExportScopeTypeRegion = "region"

// ExportJobAuditCaseFile 是稽核快照中單一個案的匯出檔案摘要。
type ExportJobAuditCaseFile struct {
	CaseName string `json:"caseName"`
	FileName string `json:"fileName"`
	RowCount int    `json:"rowCount"`
}

// NationalIDCiphers 保存重繪快照時補回身分證欄所需的密文。
type NationalIDCiphers struct {
	Case    []byte
	Drivers map[uuid.UUID][]byte
}

// ZipEntry 代表壓縮檔中的一個檔案。
type ZipEntry struct {
	Name    string
	Content []byte
}
