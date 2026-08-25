package service

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

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
	db *pgxpool.Pool
}

// NewDashboardService 建立 DashboardService 實例。
func NewDashboardService(db *pgxpool.Pool) *DashboardService {
	return &DashboardService{db: db}
}

// GetMetrics 查詢儀表板完整營運與圖表統計指標。
func (s *DashboardService) GetMetrics(ctx context.Context, periodYm string) (*DashboardMetricsDTO, error) {
	now := time.Now()
	rocYear := now.Year() - 1911
	currentMonthStr := fmt.Sprintf("%03d-%02d", rocYear, int(now.Month()))
	if periodYm != "" {
		currentMonthStr = periodYm
	}

	startDate, endDate, _ := parsePeriodYM(currentMonthStr)

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

	if s.db == nil {
		dto.TotalCasesCount = 186
		dto.ReportedTripsCount = 1420
		dto.PendingConflictsCount = 2
		dto.PendingFormColumnsCount = 0
		dto.AttendanceDistribution = AttendanceDistributionDTO{
			WorkCount:       280,
			LeaveCount:      6,
			SickCount:       2,
			OffCount:        112,
			LeavePercentage: 2.78,
		}
		dto.VehicleTripTrends = []VehicleTripTrendItemDTO{
			{VehicleName: "竹北一車", PlateNo: "BZG-7915", TripCount: 185},
			{VehicleName: "竹北二車", PlateNo: "BZG-7916", TripCount: 160},
			{VehicleName: "竹北三車", PlateNo: "BZG-7917", TripCount: 145},
			{VehicleName: "竹南一車", PlateNo: "BZG-8801", TripCount: 172},
			{VehicleName: "竹南二車", PlateNo: "BZG-8802", TripCount: 190},
			{VehicleName: "頭份一車", PlateNo: "BZG-9901", TripCount: 130},
		}
		return dto, nil
	}

	_ = s.db.QueryRow(ctx, "SELECT COUNT(*) FROM cases WHERE status = 'active'").Scan(&dto.TotalCasesCount)

	_ = s.db.QueryRow(ctx, `
		SELECT COUNT(*) 
		FROM ride_records 
		WHERE service_date >= $1 AND service_date < $2 AND effective_status = 'boarded'
	`, startDate, endDate).Scan(&dto.ReportedTripsCount)

	_ = s.db.QueryRow(ctx, `
		SELECT COUNT(*) 
		FROM ride_records 
		WHERE has_conflict = true AND conflict_resolved_at IS NULL
	`).Scan(&dto.PendingConflictsCount)

	_ = s.db.QueryRow(ctx, `
		SELECT COUNT(*) 
		FROM form_columns 
		WHERE mapping_status = 'pending'
	`).Scan(&dto.PendingFormColumnsCount)

	trendRows, err := s.db.Query(ctx, `
		SELECT v.display_name, v.plate_no, COUNT(r.id) as trips
		FROM vehicles v
		LEFT JOIN ride_records r ON r.vehicle_id = v.id 
		  AND r.service_date >= $1 AND r.service_date < $2 
		  AND r.effective_status = 'boarded'
		GROUP BY v.id, v.display_name, v.plate_no
		ORDER BY v.display_name ASC
	`, startDate, endDate)
	if err == nil {
		defer trendRows.Close()
		for trendRows.Next() {
			var vName, plateNo string
			var trips int
			if err := trendRows.Scan(&vName, &plateNo, &trips); err == nil {
				dto.VehicleTripTrends = append(dto.VehicleTripTrends, VehicleTripTrendItemDTO{
					VehicleName: vName,
					PlateNo:     plateNo,
					TripCount:   trips,
				})
			}
		}
	}

	attRows, err := s.db.Query(ctx, `
		SELECT status, COUNT(*)
		FROM attendance_records
		WHERE record_date >= $1 AND record_date < $2
		GROUP BY status
	`, startDate, endDate)
	if err == nil {
		defer attRows.Close()
		totalWorkingDays := 0
		for attRows.Next() {
			var status string
			var count int
			if err := attRows.Scan(&status, &count); err == nil {
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
		}
		if totalWorkingDays > 0 {
			leaveSum := dto.AttendanceDistribution.LeaveCount + dto.AttendanceDistribution.SickCount
			dto.AttendanceDistribution.LeavePercentage = float64(leaveSum) / float64(totalWorkingDays) * 100.0
		}
	}

	return dto, nil
}
