package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ReportRepository 提供報表統計查詢操作。
type ReportRepository struct {
	db *pgxpool.Pool
}

// NewReportRepository 建立 ReportRepository 實例。
func NewReportRepository(db *pgxpool.Pool) *ReportRepository {
	return &ReportRepository{db: db}
}

// GetTripSummary 取得特定月份各車輛之個案去回程與合計趟數。
func (r *ReportRepository) GetTripSummary(ctx context.Context, year, month int, vehicleID *uuid.UUID) ([]VehicleTripSummary, error) {
	if r.db == nil {
		return []VehicleTripSummary{}, nil
	}

	firstDay := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	lastDay := firstDay.AddDate(0, 1, -1)

	// 1. 取得目標車輛清單
	vehicleQuery := `
		SELECT id, plate_no, display_name
		FROM vehicles
		WHERE ($1::uuid IS NULL OR id = $1)
		  AND status = 'active'
		ORDER BY display_name ASC
	`
	rows, err := r.db.Query(ctx, vehicleQuery, vehicleID)
	if err != nil {
		return nil, fmt.Errorf("failed to query vehicles for report: %w", err)
	}
	defer rows.Close()

	var vehicles []VehicleEntity
	for rows.Next() {
		var v VehicleEntity
		if err := rows.Scan(&v.ID, &v.PlateNo, &v.DisplayName); err != nil {
			return nil, err
		}
		vehicles = append(vehicles, v)
	}

	var results []VehicleTripSummary

	// 2. 針對每台車輛查詢當月搭乘紀錄統計
	summaryQuery := `
		SELECT 
			c.id AS case_id,
			c.code AS case_code,
			c.name AS case_name,
			COUNT(CASE WHEN sl.direction = 'outbound' THEN 1 END) AS outbound_count,
			COUNT(CASE WHEN sl.direction = 'inbound' THEN 1 END) AS inbound_count,
			COUNT(*) AS total_count
		FROM ride_records rr
		JOIN cases c ON rr.case_id = c.id
		JOIN case_schedules cs ON rr.schedule_id = cs.id
		JOIN schedule_legs sl ON cs.id = sl.schedule_id AND rr.leg_seq = sl.leg_seq
		WHERE rr.vehicle_id = $1
		  AND rr.service_date >= $2 AND rr.service_date <= $3
		  AND rr.effective_status = 'boarded'
		GROUP BY c.id, c.code, c.name
		ORDER BY c.name ASC
	`

	for _, v := range vehicles {
		vSummary := VehicleTripSummary{
			VehicleID:   v.ID,
			PlateNo:     v.PlateNo,
			DisplayName: v.DisplayName,
			Rows:        []TripSummaryRow{},
		}

		sRows, err := r.db.Query(ctx, summaryQuery, v.ID, firstDay, lastDay)
		if err != nil {
			return nil, fmt.Errorf("failed to aggregate trips for vehicle %s: %w", v.DisplayName, err)
		}

		for sRows.Next() {
			var row TripSummaryRow
			if err := sRows.Scan(&row.CaseID, &row.CaseCode, &row.CaseName, &row.OutboundCount, &row.InboundCount, &row.TotalCount); err != nil {
				sRows.Close()
				return nil, err
			}
			vSummary.Rows = append(vSummary.Rows, row)
			vSummary.TotalOutboundCount += row.OutboundCount
			vSummary.TotalInboundCount += row.InboundCount
			vSummary.GrandTotalCount += row.TotalCount
		}
		sRows.Close()

		results = append(results, vSummary)
	}

	return results, nil
}
