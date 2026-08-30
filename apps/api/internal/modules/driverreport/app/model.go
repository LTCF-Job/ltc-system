package app

import (
	"time"

	"github.com/google/uuid"
)

// ReportForm 代表一台車的司機接送匯報表（driver_report_forms）。
type ReportForm struct {
	ID                 uuid.UUID
	VehicleID          uuid.UUID
	VehicleDisplayName string
	Title              string
	Region             string
	LastImportedAt     *time.Time
	Status             string
	TotalColumns       int
	MappedColumns      int
	PendingColumns     int
	SubmissionCount    int
}

// ColumnMapping 代表包含關聯資訊的匯報表欄位對應實體。
type ColumnMapping struct {
	ID                string
	FormID            string
	FormTitle         string
	VehicleName       string
	ColumnIndex       int
	ColumnHeader      string
	CleanedName       string
	Kind              string
	MappingStatus     string
	CaseID            *string
	CaseName          *string
	LegSeq            *int16
	SuggestedCaseID   *string
	SuggestedCaseName *string
	SuggestedLegSeq   *int16
	SuggestionScore   float64
}

// ColumnDraft 是解析檔案表頭後要寫回 form_columns 的欄位定義。
type ColumnDraft struct {
	ColumnIndex     int
	ColumnHeader    string
	CleanedName     string
	Kind            string
	SuggestedCaseID *string
	SuggestionScore float64
}

// ColumnPreview 是預覽階段回傳給前端就地確認對應的單一欄位。
type ColumnPreview struct {
	ColumnID          string  `json:"columnId,omitempty"`
	ColumnIndex       int     `json:"columnIndex"`
	ColumnHeader      string  `json:"columnHeader"`
	CleanedName       string  `json:"cleanedName"`
	Direction         string  `json:"direction,omitempty"`
	MappingStatus     string  `json:"mappingStatus"`
	CaseID            *string `json:"caseId,omitempty"`
	CaseName          *string `json:"caseName,omitempty"`
	LegSeq            *int16  `json:"legSeq,omitempty"`
	SuggestedCaseID   *string `json:"suggestedCaseId,omitempty"`
	SuggestedCaseName *string `json:"suggestedCaseName,omitempty"`
	SuggestedLegSeq   *int16  `json:"suggestedLegSeq,omitempty"`
	SuggestionScore   float64 `json:"suggestionScore"`
	BoardedCount      int     `json:"boardedCount"`
	AbsentCount       int     `json:"absentCount"`
}

// RowPreview 是單日匯報列的解析結果。
type RowPreview struct {
	RowIndex       int    `json:"rowIndex"`
	ReportDate     string `json:"reportDate"`
	ServiceDate    string `json:"serviceDate"`
	DriverRaw      string `json:"driverRaw"`
	DriverID       string `json:"driverId,omitempty"`
	DriverName     string `json:"driverName,omitempty"`
	Remark         string `json:"remark,omitempty"`
	BoardedCount   int    `json:"boardedCount"`
	AbsentCount    int    `json:"absentCount"`
	ErrorMessage   string `json:"errorMessage,omitempty"`
	WarningMessage string `json:"warningMessage,omitempty"`
}

// ImportErrorItem 代表單筆匯入錯誤明細。
type ImportErrorItem struct {
	RowIndex int    `json:"rowIndex"`
	Field    string `json:"field,omitempty"`
	Message  string `json:"message"`
}

// ImportWarningItem 代表單筆匯入警告明細。
type ImportWarningItem struct {
	RowIndex int    `json:"rowIndex"`
	Field    string `json:"field,omitempty"`
	Message  string `json:"message"`
}

// PreviewResult 是 dryRun 階段回傳的完整預覽：欄位對應與每日匯報列各一段。
type PreviewResult struct {
	FormID          string              `json:"formId"`
	VehicleID       string              `json:"vehicleId"`
	VehicleName     string              `json:"vehicleName"`
	TotalRows       int                 `json:"totalRows"`
	ValidRows       int                 `json:"validRows"`
	ErrorRows       int                 `json:"errorRows"`
	WarningRows     int                 `json:"warningRows"`
	UnmappedColumns int                 `json:"unmappedColumns"`
	Columns         []ColumnPreview     `json:"columns"`
	PreviewRows     []RowPreview        `json:"previewRows"`
	Errors          []ImportErrorItem   `json:"errors"`
	Warnings        []ImportWarningItem `json:"warnings"`
}

// ColumnDecision 是使用者在預覽畫面對單一欄位所做的對應決定。
type ColumnDecision struct {
	ColumnHeader  string  `json:"columnHeader"`
	MappingStatus string  `json:"mappingStatus"`
	CaseID        *string `json:"caseId"`
	LegSeq        *int16  `json:"legSeq"`
}

// ColumnMappingUpdate 是欄位對應設定頁的批次更新項目，以欄位 ID 為對象。
type ColumnMappingUpdate struct {
	ColumnID      string  `json:"columnId"`
	MappingStatus string  `json:"mappingStatus"`
	CaseID        *string `json:"caseId"`
	LegSeq        *int16  `json:"legSeq"`
}

// SkippedRow 保留未寫入資料庫的來源列與原因。
type SkippedRow struct {
	RowIndex   int      `json:"rowIndex"`
	ReportDate string   `json:"reportDate"`
	Reasons    []string `json:"reasons"`
}

// CommitResult 回傳正式匯入寫入與略過的結果。
type CommitResult struct {
	ImportedRows   int                 `json:"importedRows"`
	RideRecordRows int                 `json:"rideRecordRows"`
	MappedColumns  int                 `json:"mappedColumns"`
	SkippedRows    []SkippedRow        `json:"skippedRows"`
	Warnings       []ImportWarningItem `json:"warnings,omitempty"`
}
