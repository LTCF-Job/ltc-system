package app

import (
	"time"

	"github.com/google/uuid"
)

// Caregiver 代表一位照護人員。SiteID 為 nil 且 SiteNameRaw 有值時，表示匯入時的
// 單位名稱未比對到既有據點，待人工於「待維護」畫面補建關聯。
type Caregiver struct {
	ID          uuid.UUID
	SiteID      *uuid.UUID
	SiteName    string
	SiteNameRaw string
	Name        string
	Type        string
	Contact     string
	Notes       string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// 照護人員類型為固定二選一，與姓名同為必填。
const (
	CaregiverTypeCaseManager = "case_manager"
	CaregiverTypeSpecialist  = "specialist"
)

// IsValidCaregiverType 檢查類型是否為既定選項之一。
func IsValidCaregiverType(t string) bool {
	return t == CaregiverTypeCaseManager || t == CaregiverTypeSpecialist
}

// CaregiverImportErrorItem 代表單筆匯入錯誤明細。
type CaregiverImportErrorItem struct {
	RowIndex int    `json:"rowIndex"`
	Name     string `json:"name,omitempty"`
	Field    string `json:"field,omitempty"`
	Message  string `json:"message"`
}

// CaregiverImportWarningItem 代表單筆匯入警告明細：資料已建立但仍需人工處理。
// Field 為 "site" 表示單位待關聯既有據點，為 "contact"／"notes" 表示該欄位缺漏待補齊。
type CaregiverImportWarningItem struct {
	RowIndex int    `json:"rowIndex"`
	Name     string `json:"name,omitempty"`
	Field    string `json:"field,omitempty"`
	Message  string `json:"message"`
}

// CaregiverImportRowResult 代表照護人員批次匯入單列解析結果。
type CaregiverImportRowResult struct {
	RowIndex               int               `json:"rowIndex"`
	SiteName               string            `json:"siteName,omitempty"`
	SiteID                 *uuid.UUID        `json:"siteId,omitempty"`
	Name                   string            `json:"name"`
	Type                   string            `json:"type,omitempty"`
	Contact                string            `json:"contact,omitempty"`
	Notes                  string            `json:"notes,omitempty"`
	WarningMessage         string            `json:"warningMessage,omitempty"`
	RawValues              map[string]string `json:"rawValues,omitempty"`
	IsDuplicate            bool              `json:"isDuplicate,omitempty"`
	DuplicateCaregiverID   *uuid.UUID        `json:"duplicateCaregiverId,omitempty"`
	DuplicateCaregiverName string            `json:"duplicateCaregiverName,omitempty"`
}

// CaregiverDuplicateRef 是查重比對到的既有照護人員基本資訊，供匯入預覽提示使用。
type CaregiverDuplicateRef struct {
	ID   uuid.UUID
	Name string
}

// CaregiverImportSkippedRow 保留姓名缺漏、未寫入資料庫的來源列。
type CaregiverImportSkippedRow struct {
	RowIndex  int               `json:"rowIndex"`
	Name      string            `json:"name"`
	Reasons   []string          `json:"reasons"`
	RawValues map[string]string `json:"rawValues"`
}

// CaregiverImportCommitResult 回傳正式匯入成功與略過的列。Warnings 承載已建立但
// 仍待人工補齊聯絡方式／備註或關聯據點的提示。
type CaregiverImportCommitResult struct {
	ImportedCount int                          `json:"importedCount"`
	SkippedRows   []CaregiverImportSkippedRow  `json:"skippedRows"`
	Warnings      []CaregiverImportWarningItem `json:"warnings,omitempty"`
}

// CaregiverImportPreviewResult 批次匯入預覽與統計結構體。
type CaregiverImportPreviewResult struct {
	TotalRows   int                          `json:"totalRows"`
	ValidRows   int                          `json:"validRows"`
	ErrorRows   int                          `json:"errorRows"`
	WarningRows int                          `json:"warningRows"`
	PreviewRows []map[string]interface{}     `json:"previewRows"`
	Errors      []CaregiverImportErrorItem   `json:"errors"`
	Warnings    []CaregiverImportWarningItem `json:"warnings"`
	Rows        []CaregiverImportRowResult   `json:"rows,omitempty"`
}
