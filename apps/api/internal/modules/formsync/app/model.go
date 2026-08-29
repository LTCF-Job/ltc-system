package app

import (
	"time"

	"github.com/google/uuid"
)

// GoogleForm 代表 google_forms 查詢實體。
type GoogleForm struct {
	ID                 uuid.UUID
	VehicleID          uuid.UUID
	SheetID            string
	VehicleDisplayName string
	FormTitle          string
	Region             string
	LastSyncedAt       *time.Time
	Status             string
}

// ColumnMapping 代表包含關聯資訊的表單欄位對應實體。
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
	SuggestionScore   float64
}

// DriveFile 代表雲端硬碟中的一份試算表，是 formsync 需要的最小資訊。
type DriveFile struct {
	ID           string
	Name         string
	MimeType     string
	ModifiedTime string
}

// SpreadsheetInfo 代表一份試算表的結構與分頁。
type SpreadsheetInfo struct {
	SpreadsheetID string
	Title         string
	SheetTabs     []string
}
