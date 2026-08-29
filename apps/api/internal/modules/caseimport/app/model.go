package app

import (
	"strconv"
	"strings"
	"time"

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

// WeekdayScheduleDetail 代表個案在單一星期的排班趟數與時段。
type WeekdayScheduleDetail struct {
	Weekday    int16  `json:"weekday"`
	TripCount  int16  `json:"tripCount"`
	DepartTime string `json:"departTime,omitempty"`
	ReturnTime string `json:"returnTime,omitempty"`
}

// CaseImportRowResult 代表個案批次匯入單列解析結果。
type CaseImportRowResult struct {
	RowIndex           int                     `json:"rowIndex"`
	SheetName          string                  `json:"sheetName"`
	Name               string                  `json:"name"`
	NationalID         string                  `json:"nationalId,omitempty"`
	Phone              string                  `json:"phone,omitempty"`
	HouseholdType      string                  `json:"householdType,omitempty"`
	Gender             string                  `json:"gender,omitempty"`
	BirthDate          string                  `json:"birthDate,omitempty"`
	CareContactRole    string                  `json:"careContactRole,omitempty"`
	CareContactName    string                  `json:"careContactName,omitempty"`
	RegisteredAddress  string                  `json:"registeredAddress,omitempty"`
	HomeAddress        string                  `json:"homeAddress,omitempty"`
	Region             string                  `json:"region"`
	ClaimStartDate     string                  `json:"claimStartDate"`
	ServiceCategory    int                     `json:"serviceCategory"`
	ServiceUsageType   int                     `json:"serviceUsageType"`
	SiteName           string                  `json:"siteName"`
	SiteID             *uuid.UUID              `json:"siteId,omitempty"`
	Weekdays           []int16                 `json:"weekdays"`
	WeekdaysText       string                  `json:"weekdaysText,omitempty"`
	WeekdaySchedules   []WeekdayScheduleDetail `json:"weekdaySchedules,omitempty"`
	OutboundTime       string                  `json:"outboundTime,omitempty"`
	InboundTime        string                  `json:"inboundTime,omitempty"`
	OutboundVehicle    string                  `json:"outboundVehicle,omitempty"`
	InboundVehicle     string                  `json:"inboundVehicle,omitempty"`
	TripPattern        int16                   `json:"tripPattern"`
	DistanceKM         float64                 `json:"distanceKm"`
	UnitPrice          float64                 `json:"unitPrice"`
	ServiceDurationMin int16                   `json:"serviceDurationMin"`
	Note               string                  `json:"note,omitempty"`
	IsProfileWorkbook  bool                    `json:"isProfileWorkbook"`
	IsDraft            bool                    `json:"isDraft"`
	WarningMessage     string                  `json:"warningMessage,omitempty"`
	ErrorMessage       string                  `json:"errorMessage,omitempty"`
	RawValues          map[string]string       `json:"rawValues,omitempty"`
}

// CaseImportSkippedRow 保留未寫入資料庫的來源列與欄位錯誤。
type CaseImportSkippedRow struct {
	RowIndex  int               `json:"rowIndex"`
	CaseName  string            `json:"caseName"`
	Reasons   []string          `json:"reasons"`
	RawValues map[string]string `json:"rawValues"`
}

// CaseImportCommitResult 回傳正式匯入成功與略過的列，供操作人員補正來源資料。
type CaseImportCommitResult struct {
	ImportedCount int                    `json:"importedCount"`
	SkippedRows   []CaseImportSkippedRow `json:"skippedRows"`
}

func parseProfileBirthDate(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if parsed, err := time.Parse("2006-01-02", value); err == nil {
		return parsed.Format("2006-01-02")
	}
	parts := strings.FieldsFunc(value, func(r rune) bool { return r == '/' || r == '-' || r == '.' })
	if len(parts) != 3 {
		return ""
	}
	year, yearErr := strconv.Atoi(parts[0])
	month, monthErr := strconv.Atoi(parts[1])
	day, dayErr := strconv.Atoi(parts[2])
	if yearErr != nil || monthErr != nil || dayErr != nil {
		return ""
	}
	if year < 1911 {
		year += 1911
	}
	parsed := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
	if parsed.Year() != year || int(parsed.Month()) != month || parsed.Day() != day {
		return ""
	}
	return parsed.Format("2006-01-02")
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
