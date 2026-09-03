package transport

import (
	"ltc-system/apps/api/internal/modules/driverreport/app"
)

// FormListItemDTO 是匯報表清單的 API 回應項目。
type FormListItemDTO struct {
	ID              string  `json:"id"`
	VehicleID       string  `json:"vehicleId"`
	VehicleName     string  `json:"vehicleName"`
	Title           string  `json:"title"`
	Region          string  `json:"region"`
	LastImportedAt  *string `json:"lastImportedAt"`
	TotalColumns    int     `json:"totalColumns"`
	MappedColumns   int     `json:"mappedColumns"`
	PendingColumns  int     `json:"pendingColumns"`
	SubmissionCount int     `json:"submissionCount"`
	Status          string  `json:"status"`
}

// FormColumnDTO 是欄位對應設定頁的 API 回應項目。
type FormColumnDTO struct {
	ID                string  `json:"id"`
	FormID            string  `json:"formId"`
	FormTitle         string  `json:"formTitle"`
	VehicleName       string  `json:"vehicleName"`
	ColumnIndex       int     `json:"columnIndex"`
	ColumnHeader      string  `json:"columnHeader"`
	CleanedName       string  `json:"cleanedName"`
	Kind              string  `json:"kind"`
	MappingStatus     string  `json:"mappingStatus"`
	CaseID            *string `json:"caseId"`
	CaseName          *string `json:"caseName"`
	LegSeq            *int16  `json:"legSeq"`
	SuggestedCaseID   *string `json:"suggestedCaseId"`
	SuggestedCaseName *string `json:"suggestedCaseName"`
	SuggestedLegSeq   *int16  `json:"suggestedLegSeq"`
	SuggestionScore   float64 `json:"suggestionScore"`
}

// ImportedMonthDTO 是「某份匯報表某個月已匯入多少筆」的 API 回應項目。
type ImportedMonthDTO struct {
	FormID          string `json:"formId"`
	YearMonth       string `json:"yearMonth"`
	SubmissionCount int    `json:"submissionCount"`
	LastImportedAt  string `json:"lastImportedAt"`
}

// MonthSubmissionDTO 是總覽頁鑽取單一月份時，逐日回報明細頁籤的 API 回應項目。
type MonthSubmissionDTO struct {
	ServiceDate   string            `json:"serviceDate"`
	DriverNameRaw string            `json:"driverNameRaw"`
	Remark        string            `json:"remark"`
	Answers       map[string]string `json:"answers"`
}

// MonthRideEntryDTO 是總覽頁鑽取單一月份時，逐個案搭乘紀錄頁籤的 API 回應項目。
type MonthRideEntryDTO struct {
	CaseID      string  `json:"caseId"`
	CaseName    string  `json:"caseName"`
	ServiceDate string  `json:"serviceDate"`
	LegSeq      int16   `json:"legSeq"`
	Reported    string  `json:"reported"`
	DriverID    *string `json:"driverId"`
	DriverName  string  `json:"driverName"`
	VehicleID   string  `json:"vehicleId"`
}

// MonthDetailDTO 是總覽頁鑽取單一匯報表、單一月份時的完整內容。
type MonthDetailDTO struct {
	Submissions []MonthSubmissionDTO `json:"submissions"`
	RideEntries []MonthRideEntryDTO  `json:"rideEntries"`
}

// CreateFormRequest 是建立車輛匯報表的請求。
type CreateFormRequest struct {
	VehicleID string `json:"vehicleId" binding:"required"`
	Title     string `json:"title" binding:"required"`
}

// UpdateColumnMappingRequest 是單一欄位對應更新的請求。
type UpdateColumnMappingRequest struct {
	MappingStatus string  `json:"mappingStatus" binding:"required,oneof=pending mapped ignored"`
	CaseID        *string `json:"caseId"`
	LegSeq        *int16  `json:"legSeq"`
}

// BatchMappingRequest 是欄位對應批次更新的請求。
type BatchMappingRequest struct {
	Mappings []app.ColumnMappingUpdate `json:"mappings" binding:"required,min=1"`
}

// SubmissionReviewDTO 是待維護資料頁籤以匯報表列為單位的 API 回應項目。
type SubmissionReviewDTO struct {
	SubmissionID string          `json:"submissionId"`
	FormTitle    string          `json:"formTitle"`
	VehicleName  string          `json:"vehicleName"`
	ServiceDate  string          `json:"serviceDate"`
	CaseIssues   []FormColumnDTO `json:"caseIssues"`
	DriverIssue  *DriverIssueDTO `json:"driverIssue,omitempty"`
}

// DriverIssueDTO 代表這一列的駕駛人姓名比對不到司機主檔。
type DriverIssueDTO struct {
	DriverNameRaw string `json:"driverNameRaw"`
}

// BindDriverRequest 是把某個未比對駕駛人姓名綁定到指定司機的請求。
type BindDriverRequest struct {
	DriverNameRaw string `json:"driverNameRaw" binding:"required"`
	DriverID      string `json:"driverId" binding:"required"`
}

func toFormListItemDTO(f app.ReportForm) FormListItemDTO {
	var lastImported *string
	if f.LastImportedAt != nil {
		formatted := f.LastImportedAt.Format("2006-01-02 15:04:05")
		lastImported = &formatted
	}
	return FormListItemDTO{
		ID:              f.ID.String(),
		VehicleID:       f.VehicleID.String(),
		VehicleName:     f.VehicleDisplayName,
		Title:           f.Title,
		Region:          f.Region,
		LastImportedAt:  lastImported,
		TotalColumns:    f.TotalColumns,
		MappedColumns:   f.MappedColumns,
		PendingColumns:  f.PendingColumns,
		SubmissionCount: f.SubmissionCount,
		Status:          f.Status,
	}
}

func toImportedMonthDTO(m app.ImportedMonth) ImportedMonthDTO {
	return ImportedMonthDTO{
		FormID:          m.FormID.String(),
		YearMonth:       m.YearMonth,
		SubmissionCount: m.SubmissionCount,
		LastImportedAt:  m.LastImportedAt.Format("2006-01-02 15:04:05"),
	}
}

func toSubmissionReviewDTO(r app.SubmissionReview) SubmissionReviewDTO {
	issues := make([]FormColumnDTO, 0, len(r.CaseIssues))
	for _, c := range r.CaseIssues {
		issues = append(issues, toFormColumnDTO(c))
	}
	var driverIssue *DriverIssueDTO
	if r.DriverIssue != nil {
		driverIssue = &DriverIssueDTO{DriverNameRaw: r.DriverIssue.DriverNameRaw}
	}
	return SubmissionReviewDTO{
		SubmissionID: r.SubmissionID,
		FormTitle:    r.FormTitle,
		VehicleName:  r.VehicleName,
		ServiceDate:  r.ServiceDate,
		CaseIssues:   issues,
		DriverIssue:  driverIssue,
	}
}

func toMonthDetailDTO(d app.MonthDetail) MonthDetailDTO {
	submissions := make([]MonthSubmissionDTO, 0, len(d.Submissions))
	for _, s := range d.Submissions {
		submissions = append(submissions, MonthSubmissionDTO{
			ServiceDate:   s.ServiceDate.Format("2006-01-02"),
			DriverNameRaw: s.DriverNameRaw,
			Remark:        s.Remark,
			Answers:       s.Answers,
		})
	}
	rideEntries := make([]MonthRideEntryDTO, 0, len(d.RideEntries))
	for _, e := range d.RideEntries {
		var driverID *string
		if e.DriverID != nil {
			id := e.DriverID.String()
			driverID = &id
		}
		rideEntries = append(rideEntries, MonthRideEntryDTO{
			CaseID:      e.CaseID.String(),
			CaseName:    e.CaseName,
			ServiceDate: e.ServiceDate.Format("2006-01-02"),
			LegSeq:      e.LegSeq,
			Reported:    e.Reported,
			DriverID:    driverID,
			DriverName:  e.DriverName,
			VehicleID:   e.VehicleID.String(),
		})
	}
	return MonthDetailDTO{Submissions: submissions, RideEntries: rideEntries}
}

func toFormColumnDTO(c app.ColumnMapping) FormColumnDTO {
	return FormColumnDTO{
		ID:                c.ID,
		FormID:            c.FormID,
		FormTitle:         c.FormTitle,
		VehicleName:       c.VehicleName,
		ColumnIndex:       c.ColumnIndex,
		ColumnHeader:      c.ColumnHeader,
		CleanedName:       c.CleanedName,
		Kind:              c.Kind,
		MappingStatus:     c.MappingStatus,
		CaseID:            c.CaseID,
		CaseName:          c.CaseName,
		LegSeq:            c.LegSeq,
		SuggestedCaseID:   c.SuggestedCaseID,
		SuggestedCaseName: c.SuggestedCaseName,
		SuggestedLegSeq:   c.SuggestedLegSeq,
		SuggestionScore:   c.SuggestionScore,
	}
}
