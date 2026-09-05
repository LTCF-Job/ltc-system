package app

import (
	"context"
	"time"

	"github.com/google/uuid"
	"ltc-system/apps/api/internal/domain/govform"
	"ltc-system/apps/api/internal/domain/rocdate"
)

// ReportRepositoryPort 定義報表資料存取介面。
type ReportRepositoryPort interface {
	QueryTripSummaryData(ctx context.Context, startDate, endDate time.Time, region *string, vehicleID *uuid.UUID) ([]ReportVehicleTripSummary, error)
	QueryHsinchuScheduleData(ctx context.Context, siteID *uuid.UUID, vehicleID *uuid.UUID) ([]ReportHsinchuScheduleRow, error)
}

// TripSummaryCaseRow 代表單一個案之趟數統計。
type TripSummaryCaseRow struct {
	CaseID        string `json:"caseId"`
	CaseName      string `json:"caseName"`
	OutboundCount int    `json:"outboundCount"`
	InboundCount  int    `json:"inboundCount"`
	TotalCount    int    `json:"totalCount"`
}

// TripSummaryVehicle 代表單一車輛之趟數分組。
type TripSummaryVehicle struct {
	VehicleID        string               `json:"vehicleId"`
	VehicleName      string               `json:"vehicleName"`
	PlateNo          string               `json:"plateNo"`
	DriverName       *string              `json:"driverName,omitempty"`
	Rows             []TripSummaryCaseRow `json:"rows"`
	SubtotalOutbound int                  `json:"subtotalOutbound"`
	SubtotalInbound  int                  `json:"subtotalInbound"`
	SubtotalTotal    int                  `json:"subtotalTotal"`
}

// TripSummaryReport 代表車輛趟數表報表總體結構。
type TripSummaryReport struct {
	PeriodYM           string               `json:"periodYm"`
	Region             *string              `json:"region,omitempty"`
	GeneratedAt        string               `json:"generatedAt"`
	Vehicles           []TripSummaryVehicle `json:"vehicles"`
	GrandTotalOutbound int                  `json:"grandTotalOutbound"`
	GrandTotalInbound  int                  `json:"grandTotalInbound"`
	GrandTotal         int                  `json:"grandTotal"`
}

// HsinchuScheduleItem 代表新竹接送時刻表單一排班項目。
type HsinchuScheduleItem struct {
	Direction   string  `json:"direction"` // outbound, inbound
	RunNo       int16   `json:"runNo"`     // 趟次 (1, 2, 3...)
	CaseName    string  `json:"caseName"`
	Note        *string `json:"note,omitempty"`
	DepartTime  string  `json:"departTime"`
	Origin      string  `json:"origin"`
	ArriveTime  *string `json:"arriveTime,omitempty"`
	Destination string  `json:"destination"`
	VehicleName string  `json:"vehicleName"`
	SiteName    string  `json:"siteName"`
}

// HsinchuScheduleReport 代表新竹接送時刻表完整結構。
type HsinchuScheduleReport struct {
	GeneratedAt string                `json:"generatedAt"`
	SiteName    *string               `json:"siteName,omitempty"`
	VehicleName *string               `json:"vehicleName,omitempty"`
	Outbound    []HsinchuScheduleItem `json:"outbound"`
	Inbound     []HsinchuScheduleItem `json:"inbound"`
}

// ReportService 提供報表資料彙總與 Excel 檔案產生服務。
type ReportService struct {
	repo     ReportRepositoryPort
	renderer Renderer
}

// NewReportService 建立 ReportService 實例。
func NewReportService(repo ReportRepositoryPort, renderer Renderer) *ReportService {
	return &ReportService{repo: repo, renderer: renderer}
}

// GetTripSummary 查詢車輛趟數表結構化資料。
func (s *ReportService) GetTripSummary(ctx context.Context, periodYm string, region *string, vehicleID *uuid.UUID) (*TripSummaryReport, error) {
	report := &TripSummaryReport{
		PeriodYM:    periodYm,
		Region:      region,
		GeneratedAt: time.Now().Format("2006-01-02 15:04:05"),
		Vehicles:    []TripSummaryVehicle{},
	}

	if s.repo == nil {
		return report, nil
	}

	startDate, endDate, _, err := rocdate.MonthRangeStrict(periodYm)
	if err != nil {
		return nil, err
	}
	vehSummaries, err := s.repo.QueryTripSummaryData(ctx, startDate, endDate, region, vehicleID)
	if err != nil {
		return nil, err
	}

	for _, vs := range vehSummaries {
		vehSummary := TripSummaryVehicle{
			VehicleID:   vs.Vehicle.ID.String(),
			VehicleName: vs.Vehicle.DisplayName,
			PlateNo:     vs.Vehicle.PlateNo,
			Rows:        []TripSummaryCaseRow{},
		}

		for _, r := range vs.Rows {
			vehSummary.Rows = append(vehSummary.Rows, TripSummaryCaseRow{
				CaseID:        r.CaseID.String(),
				CaseName:      r.CaseName,
				OutboundCount: r.OutboundCount,
				InboundCount:  r.InboundCount,
				TotalCount:    r.TotalCount,
			})
			vehSummary.SubtotalOutbound += r.OutboundCount
			vehSummary.SubtotalInbound += r.InboundCount
			vehSummary.SubtotalTotal += r.TotalCount
		}

		if len(vehSummary.Rows) > 0 {
			report.Vehicles = append(report.Vehicles, vehSummary)
			report.GrandTotalOutbound += vehSummary.SubtotalOutbound
			report.GrandTotalInbound += vehSummary.SubtotalInbound
			report.GrandTotal += vehSummary.SubtotalTotal
		}
	}

	return report, nil
}

// GenerateTripSummaryExcel 產生車輛趟數表 Excel 檔案。
func (s *ReportService) GenerateTripSummaryExcel(ctx context.Context, periodYm string, region *string, vehicleID *uuid.UUID) ([]byte, error) {
	report, err := s.GetTripSummary(ctx, periodYm, region, vehicleID)
	if err != nil {
		return nil, err
	}

	return s.renderer.RenderTripSummary(report.PeriodYM, report.Vehicles)
}

// GetHsinchuSchedule 查詢新竹接送時刻表排班結構。
func (s *ReportService) GetHsinchuSchedule(ctx context.Context, siteID *uuid.UUID, vehicleID *uuid.UUID) (*HsinchuScheduleReport, error) {
	report := &HsinchuScheduleReport{
		GeneratedAt: time.Now().Format("2006-01-02 15:04:05"),
		Outbound:    []HsinchuScheduleItem{},
		Inbound:     []HsinchuScheduleItem{},
	}

	if s.repo == nil {
		return report, nil
	}

	items, err := s.repo.QueryHsinchuScheduleData(ctx, siteID, vehicleID)
	if err != nil {
		return nil, err
	}

	for _, item := range items {
		var origin, destination string
		if item.Direction == "outbound" {
			origin = item.HomeAddress
			destination = item.SiteAddress
		} else {
			origin = item.SiteAddress
			destination = item.HomeAddress
		}

		schedItem := HsinchuScheduleItem{
			Direction:   item.Direction,
			RunNo:       item.RunNo,
			CaseName:    item.CaseName,
			Note:        item.Note,
			DepartTime:  item.DepartTime,
			Origin:      origin,
			ArriveTime:  item.ArriveTime,
			Destination: destination,
			VehicleName: item.VehicleName,
			SiteName:    item.SiteName,
		}

		if item.Direction == "outbound" {
			report.Outbound = append(report.Outbound, schedItem)
		} else {
			report.Inbound = append(report.Inbound, schedItem)
		}
	}

	return report, nil
}

// GenerateHsinchuScheduleExcel 產生符合規格書的新竹接送時刻表 Excel 檔案。
func (s *ReportService) GenerateHsinchuScheduleExcel(ctx context.Context, siteID *uuid.UUID, vehicleID *uuid.UUID) ([]byte, error) {
	report, err := s.GetHsinchuSchedule(ctx, siteID, vehicleID)
	if err != nil {
		return nil, err
	}

	return s.renderer.RenderHsinchuSchedule(report.Outbound, report.Inbound)
}

// GenerateGovClaimExcel 產生政府申報 Excel 檔案。
func (s *ReportService) GenerateGovClaimExcel(rows []govform.ClaimRow) ([]byte, error) {
	return s.renderer.RenderGovClaim(rows)
}
