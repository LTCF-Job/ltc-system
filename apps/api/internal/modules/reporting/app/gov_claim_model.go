package app

import (
	"time"

	"github.com/google/uuid"
)

// 本檔是政府申報匯出的應用層模型。與 readmodel.go 的純查詢投影不同，這些型別會落地到
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

// 申報列被跳過的原因代碼。缺資料的趟次寧可跳過並回報，也不讓 govform.BuildClaimRow
// 的預設值把它補成一列看似正常的申報資料。
const (
	SkipReasonNoScheduleLeg  = "NO_SCHEDULE_LEG"
	SkipReasonNoDepartTime   = "NO_DEPART_TIME"
	SkipReasonNoDriver       = "NO_DRIVER"
	SkipReasonNoUsageType    = "NO_SERVICE_USAGE_TYPE"
	SkipReasonNoUnitPrice    = "NO_UNIT_PRICE"
	SkipReasonNoNationalID   = "NO_NATIONAL_ID"
	SkipReasonBuildRowFailed = "BUILD_ROW_FAILED"
)

// GovClaimSource 代表組出單列申報資料所需的原始查詢結果。
// 指標欄位代表來源可能缺漏，由 service 判斷是否跳過該列。
type GovClaimSource struct {
	CaseID               uuid.UUID
	CaseCode             string
	CaseName             string
	Region               string
	CaseNationalIDCipher []byte
	CaseNationalIDMasked string
	HomeAddress          string
	ServiceCategory      int
	ServiceUsageType     int

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
	CaseCode string
	CaseName string
	Region   string
	FileName string
	RowCount int
	Checksum string
	Bytes    []byte
}

// ClaimSkip 代表某個案有幾列因資料缺漏被跳過。
type ClaimSkip struct {
	CaseID   uuid.UUID
	CaseName string
	Reason   string
	Count    int
}

// GovClaimJob 代表一次政府申報匯出工作。
// Skipped 只在建立當下有值，不會寫入資料庫，因此讀取歷史工作時為空。
type GovClaimJob struct {
	ID           uuid.UUID
	JobType      string
	PeriodYM     string
	Region       string
	Mode         GovClaimMode
	Status       string
	TotalCases   int
	TotalRows    int
	Files        []GovClaimCaseFile
	Skipped      []ClaimSkip
	ErrorMessage string
	CreatedAt    time.Time
	FinishedAt   *time.Time
}

// ExportJobCreate 代表建立匯出工作時要寫入 export_jobs 的欄位。
type ExportJobCreate struct {
	JobType   string
	PeriodYM  string
	Region    string
	Format    string
	CaseIDs   []uuid.UUID
	Precheck  *PrecheckReport
	CreatedBy uuid.UUID
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
