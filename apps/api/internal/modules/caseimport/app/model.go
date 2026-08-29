package app

import (
	"github.com/google/uuid"
)

// CaseImportErrorItem 代表單筆匯入錯誤明細。
type CaseImportErrorItem struct {
	RowIndex int    `json:"rowIndex"`
	CaseName string `json:"caseName,omitempty"`
	Field    string `json:"field,omitempty"`
	Message  string `json:"message"`
}

// CaseImportWarningItem 代表單筆匯入警告或預設值提醒明細。
type CaseImportWarningItem struct {
	RowIndex int    `json:"rowIndex"`
	CaseName string `json:"caseName,omitempty"`
	Field    string `json:"field,omitempty"`
	Message  string `json:"message"`
}

// CaseImportRowResult 代表個案批次匯入單列解析結果。
type CaseImportRowResult struct {
	RowIndex          int               `json:"rowIndex"`
	SheetName         string            `json:"sheetName"`
	Name              string            `json:"name"`
	NationalID        string            `json:"nationalId,omitempty"`
	Phone             string            `json:"phone,omitempty"`
	HouseholdType     string            `json:"householdType,omitempty"`
	Gender            string            `json:"gender,omitempty"`
	BirthDate         string            `json:"birthDate,omitempty"`
	CareContactRole   string            `json:"careContactRole,omitempty"`
	CareContactName   string            `json:"careContactName,omitempty"`
	RegisteredAddress string            `json:"registeredAddress,omitempty"`
	HomeAddress       string            `json:"homeAddress,omitempty"`
	Region            string            `json:"region"`
	ClaimStartDate    string            `json:"claimStartDate"`
	ServiceCategory   int               `json:"serviceCategory"`
	ServiceUsageType  int               `json:"serviceUsageType"`
	SiteName          string            `json:"siteName"`
	SiteID            *uuid.UUID        `json:"siteId,omitempty"`
	OutboundVehicle   string            `json:"outboundVehicle,omitempty"`
	InboundVehicle    string            `json:"inboundVehicle,omitempty"`
	Remarks           string            `json:"remarks,omitempty"`
	IsDuplicate       bool              `json:"isDuplicate"`
	DuplicateCaseCode string            `json:"duplicateCaseCode,omitempty"`
	DuplicateCaseName string            `json:"duplicateCaseName,omitempty"`
	DuplicateCaseID   *uuid.UUID        `json:"duplicateCaseId,omitempty"`
	WarningMessage    string            `json:"warningMessage,omitempty"`
	ErrorMessage      string            `json:"errorMessage,omitempty"`
	RawValues         map[string]string `json:"rawValues,omitempty"`
}

// CaseImportSkippedRow 保留未寫入資料庫的來源列與欄位錯誤。
type CaseImportSkippedRow struct {
	RowIndex  int               `json:"rowIndex"`
	CaseName  string            `json:"caseName"`
	Reasons   []string          `json:"reasons"`
	RawValues map[string]string `json:"rawValues"`
}

// CaseImportCommitResult 回傳正式匯入成功與略過的列，供操作人員補正來源資料。
// Warnings 承載已建立個案但仍需人工處理的提示（如據點/車輛未比對到）。
type CaseImportCommitResult struct {
	ImportedCount int                     `json:"importedCount"`
	SkippedRows   []CaseImportSkippedRow  `json:"skippedRows"`
	Warnings      []CaseImportWarningItem `json:"warnings,omitempty"`
}

// CaseImportPreviewResult 批次匯入預覽與統計結構體。
type CaseImportPreviewResult struct {
	TotalRows   int                      `json:"totalRows"`
	ValidRows   int                      `json:"validRows"`
	ErrorRows   int                      `json:"errorRows"`
	WarningRows int                      `json:"warningRows"`
	PreviewRows []map[string]interface{} `json:"previewRows"`
	Errors      []CaseImportErrorItem    `json:"errors"`
	Warnings    []CaseImportWarningItem  `json:"warnings"`
	Rows        []CaseImportRowResult    `json:"rows,omitempty"`
}
