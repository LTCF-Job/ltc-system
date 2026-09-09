package app

import (
	"github.com/google/uuid"
)

// CaseImportErrorItem 代表單筆匯入錯誤明細。
type CaseImportErrorItem struct {
	RowID    string `json:"rowId,omitempty"`
	RowIndex int    `json:"rowIndex"`
	CaseName string `json:"caseName,omitempty"`
	Field    string `json:"field,omitempty"`
	Message  string `json:"message"`
}

// CaseImportWarningItem 代表單筆匯入警告或預設值提醒明細。
type CaseImportWarningItem struct {
	RowID    string `json:"rowId,omitempty"`
	RowIndex int    `json:"rowIndex"`
	CaseName string `json:"caseName,omitempty"`
	Field    string `json:"field,omitempty"`
	Message  string `json:"message"`
}

// CaseImportRowResult 代表個案批次匯入單列解析結果。
type CaseImportRowResult struct {
	RowID              string            `json:"rowId"`
	RowIndex           int               `json:"rowIndex"`
	SheetName          string            `json:"sheetName"`
	Name               string            `json:"name"`
	NationalID         string            `json:"nationalId,omitempty"`
	NationalIDInvalid  bool              `json:"nationalIdInvalid"`
	HouseholdType      string            `json:"householdType,omitempty"`
	Gender             string            `json:"gender,omitempty"`
	BirthDate          string            `json:"birthDate,omitempty"`
	BirthDateRaw       string            `json:"birthDateRaw,omitempty"`
	BirthDateInvalid   bool              `json:"birthDateInvalid"`
	CareContactRole    string            `json:"careContactRole,omitempty"`
	CareContactName    string            `json:"careContactName,omitempty"`
	CaregiverUnmatched bool              `json:"caregiverUnmatched"`
	RegisteredAddress  string            `json:"registeredAddress,omitempty"`
	HomeAddress        string            `json:"homeAddress,omitempty"`
	ServiceCategory    int               `json:"serviceCategory"`
	ServiceUsageType   int               `json:"serviceUsageType"`
	SiteName           string            `json:"siteName"`
	SiteID             *uuid.UUID        `json:"siteId,omitempty"`
	OutboundVehicle    string            `json:"outboundVehicle,omitempty"`
	InboundVehicle     string            `json:"inboundVehicle,omitempty"`
	Remarks            string            `json:"remarks,omitempty"`
	IsDuplicate        bool              `json:"isDuplicate"`
	DuplicateCaseName  string            `json:"duplicateCaseName,omitempty"`
	DuplicateCaseID    *uuid.UUID        `json:"duplicateCaseId,omitempty"`
	WarningMessage     string            `json:"warningMessage,omitempty"`
	ErrorMessage       string            `json:"errorMessage,omitempty"`
	RawValues          map[string]string `json:"rawValues,omitempty"`
}

// CaseImportSkippedRow 保留未寫入資料庫的來源列與欄位錯誤。
type CaseImportSkippedRow struct {
	RowID     string            `json:"rowId"`
	RowIndex  int               `json:"rowIndex"`
	CaseName  string            `json:"caseName"`
	Reasons   []string          `json:"reasons"`
	RawValues map[string]string `json:"rawValues"`
}

// CaseImportCommitResult 回傳正式匯入成功與略過的列，供操作人員補正來源資料。
// Warnings 承載已建立個案但仍需人工處理的提示（如據點/車輛未比對到）。
type CaseImportCommitResult struct {
	ImportedCount        int                     `json:"importedCount"`
	AlreadyImportedCount int                     `json:"alreadyImportedCount"`
	FailedCount          int                     `json:"failedCount"`
	StagedDuplicateCount int                     `json:"stagedDuplicateCount"`
	SkippedRows          []CaseImportSkippedRow  `json:"skippedRows"`
	FailedRows           []CaseImportSkippedRow  `json:"failedRows"`
	Warnings             []CaseImportWarningItem `json:"warnings,omitempty"`
}

// CaseImportPreviewResult 批次匯入預覽與統計結構體。
type CaseImportPreviewResult struct {
	FileHash    string                   `json:"fileHash,omitempty"`
	TotalRows   int                      `json:"totalRows"`
	ValidRows   int                      `json:"validRows"`
	ErrorRows   int                      `json:"errorRows"`
	WarningRows int                      `json:"warningRows"`
	PreviewRows []map[string]interface{} `json:"previewRows"`
	Errors      []CaseImportErrorItem    `json:"errors"`
	Warnings    []CaseImportWarningItem  `json:"warnings"`
	Rows        []CaseImportRowResult    `json:"rows,omitempty"`
}
