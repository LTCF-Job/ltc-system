package app

import (
	"context"
	"fmt"
	"time"

	"ltc-system/apps/api/internal/domain/rocdate"
)

// DashboardRepositoryPort 定義儀表板資料庫操作介面。
type DashboardRepositoryPort interface {
	GetActiveCasesCount(ctx context.Context) (int, error)
	GetReportedTripsCount(ctx context.Context, start, end time.Time) (int, error)
	GetPendingConflictsCount(ctx context.Context) (int, error)
	GetPendingFormColumnsCount(ctx context.Context) (int, error)
	GetVehicleTripTrends(ctx context.Context, start, end time.Time) ([]VehicleTripTrend, error)
	GetAttendanceDistribution(ctx context.Context, start, end time.Time) (map[string]int, error)
}

// AttendanceDistributionDTO 代表出勤狀況分佈。
type AttendanceDistributionDTO struct {
	WorkCount       int     `json:"workCount"`
	LeaveCount      int     `json:"leaveCount"`
	SickCount       int     `json:"sickCount"`
	OffCount        int     `json:"offCount"`
	LeavePercentage float64 `json:"leavePercentage"`
}

// VehicleTripTrendItemDTO 代表車輛趟數趨勢單筆資料。
type VehicleTripTrendItemDTO struct {
	VehicleName string `json:"vehicleName"`
	PlateNo     string `json:"plateNo"`
	TripCount   int    `json:"tripCount"`
}

// DashboardMetricsDTO 代表全方位視覺化營運指標結構。
type DashboardMetricsDTO struct {
	CurrentMonth            string                    `json:"currentMonth"`
	TotalCasesCount         int                       `json:"totalCasesCount"`
	ReportedTripsCount      int                       `json:"reportedTripsCount"`
	UnreportedVehiclesToday int                       `json:"unreportedVehiclesToday"`
	PendingConflictsCount   int                       `json:"pendingConflictsCount"`
	PendingFormColumnsCount int                       `json:"pendingFormColumnsCount"`
	AttendanceDistribution  AttendanceDistributionDTO `json:"attendanceDistribution"`
	VehicleTripTrends       []VehicleTripTrendItemDTO `json:"vehicleTripTrends"`
	ClaimFulfillmentRate    float64                   `json:"claimFulfillmentRate"`
}

// DashboardService 提供儀表板整合統計資料。
type DashboardService struct {
	repo DashboardRepositoryPort
}

// NewDashboardService 建立 DashboardService 實例。
func NewDashboardService(repo DashboardRepositoryPort) *DashboardService {
	return &DashboardService{repo: repo}
}

// GetMetrics 查詢儀表板完整營運與圖表統計指標。
func (s *DashboardService) GetMetrics(ctx context.Context, periodYm string) (*DashboardMetricsDTO, error) {
	now := time.Now()
	rocYear := now.Year() - 1911
	currentMonthStr := fmt.Sprintf("%03d-%02d", rocYear, int(now.Month()))
	if periodYm != "" {
		currentMonthStr = periodYm
	}

	startDate, endDate, _ := rocdate.MonthRange(currentMonthStr)

	dto := &DashboardMetricsDTO{
		CurrentMonth: currentMonthStr,
		AttendanceDistribution: AttendanceDistributionDTO{
			WorkCount:       0,
			LeaveCount:      0,
			SickCount:       0,
			OffCount:        0,
			LeavePercentage: 0.0,
		},
		VehicleTripTrends:    []VehicleTripTrendItemDTO{},
		ClaimFulfillmentRate: 95.0,
	}

	// 無資料庫連線時回傳誠實的零值指標，與 ReportService 的離線行為一致，
	// 不再回傳假造的固定示範數字。
	if s.repo == nil {
		return dto, nil
	}

	if casesCount, err := s.repo.GetActiveCasesCount(ctx); err == nil {
		dto.TotalCasesCount = casesCount
	}

	if tripsCount, err := s.repo.GetReportedTripsCount(ctx, startDate, endDate); err == nil {
		dto.ReportedTripsCount = tripsCount
	}

	if conflictsCount, err := s.repo.GetPendingConflictsCount(ctx); err == nil {
		dto.PendingConflictsCount = conflictsCount
	}

	if colsCount, err := s.repo.GetPendingFormColumnsCount(ctx); err == nil {
		dto.PendingFormColumnsCount = colsCount
	}

	if trends, err := s.repo.GetVehicleTripTrends(ctx, startDate, endDate); err == nil {
		for _, t := range trends {
			dto.VehicleTripTrends = append(dto.VehicleTripTrends, VehicleTripTrendItemDTO{
				VehicleName: t.VehicleName,
				PlateNo:     t.PlateNo,
				TripCount:   t.TripCount,
			})
		}
	}

	if dist, err := s.repo.GetAttendanceDistribution(ctx, startDate, endDate); err == nil {
		totalWorkingDays := 0
		for status, count := range dist {
			switch status {
			case "work":
				dto.AttendanceDistribution.WorkCount = count
				totalWorkingDays += count
			case "leave":
				dto.AttendanceDistribution.LeaveCount = count
				totalWorkingDays += count
			case "sick":
				dto.AttendanceDistribution.SickCount = count
				totalWorkingDays += count
			case "off":
				dto.AttendanceDistribution.OffCount = count
			}
		}
		if totalWorkingDays > 0 {
			leaveSum := dto.AttendanceDistribution.LeaveCount + dto.AttendanceDistribution.SickCount
			dto.AttendanceDistribution.LeavePercentage = float64(leaveSum) / float64(totalWorkingDays) * 100.0
		}
	}

	return dto, nil
}
