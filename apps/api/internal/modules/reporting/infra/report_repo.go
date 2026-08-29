package infra

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"ltc-system/apps/api/internal/modules/reporting/app"
)

// ReportRepository 提供報表統計查詢操作。
type ReportRepository struct {
	db *pgxpool.Pool
}

// NewReportRepository 建立 ReportRepository 實例。
func NewReportRepository(db *pgxpool.Pool) *ReportRepository {
	return &ReportRepository{db: db}
}

// QueryTripSummaryData 查詢車輛趟數表所需之資料庫聚合資料。
func (r *ReportRepository) QueryTripSummaryData(ctx context.Context, startDate, endDate time.Time, region *string, vehicleID *uuid.UUID) ([]app.ReportVehicleTripSummary, error) {
	if r.db == nil {
		return []app.ReportVehicleTripSummary{}, nil
	}

	vehQuery := "SELECT id, plate_no, display_name, region FROM vehicles WHERE 1=1"
	var args []interface{}
	argIdx := 1
	if region != nil && *region != "" {
		vehQuery += fmt.Sprintf(" AND region = $%d", argIdx)
		args = append(args, *region)
		argIdx++
	}
	if vehicleID != nil {
		vehQuery += fmt.Sprintf(" AND id = $%d", argIdx)
		args = append(args, *vehicleID)
		argIdx++
	}
	vehQuery += " ORDER BY display_name ASC"

	vehRows, err := r.db.Query(ctx, vehQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query vehicles for report: %w", err)
	}
	defer vehRows.Close()

	var vehicles []app.ReportVehicleItem
	for vehRows.Next() {
		var v app.ReportVehicleItem
		if err := vehRows.Scan(&v.ID, &v.PlateNo, &v.DisplayName, &v.Region); err == nil {
			vehicles = append(vehicles, v)
		}
	}

	var results []app.ReportVehicleTripSummary
	statQuery := `
		SELECT 
			c.id, c.code, c.name,
			COALESCE(SUM(CASE WHEN r.leg_seq IN (1, 3) THEN 1 ELSE 0 END), 0) AS outbound_count,
			COALESCE(SUM(CASE WHEN r.leg_seq IN (2, 4) THEN 1 ELSE 0 END), 0) AS inbound_count,
			COUNT(r.id) AS total_count
		FROM cases c
		JOIN ride_records r ON r.case_id = c.id
		WHERE r.vehicle_id = $1
		  AND r.service_date >= $2 AND r.service_date < $3
		  AND r.effective_status = 'boarded'
		GROUP BY c.id, c.code, c.name
		ORDER BY c.code ASC
	`

	for _, v := range vehicles {
		rRows, err := r.db.Query(ctx, statQuery, v.ID, startDate, endDate)
		if err != nil {
			continue
		}

		var rows []app.ReportTripSummaryCaseRow
		for rRows.Next() {
			var row app.ReportTripSummaryCaseRow
			if err := rRows.Scan(&row.CaseID, &row.CaseCode, &row.CaseName, &row.OutboundCount, &row.InboundCount, &row.TotalCount); err == nil {
				rows = append(rows, row)
			}
		}
		rRows.Close()

		if len(rows) > 0 {
			results = append(results, app.ReportVehicleTripSummary{
				Vehicle: v,
				Rows:    rows,
			})
		}
	}

	return results, nil
}

// QueryHsinchuScheduleData 查詢新竹接送時刻表排班資料。
func (r *ReportRepository) QueryHsinchuScheduleData(ctx context.Context, siteID *uuid.UUID, vehicleID *uuid.UUID) ([]app.ReportHsinchuScheduleRow, error) {
	if r.db == nil {
		return []app.ReportHsinchuScheduleRow{}, nil
	}

	query := `
		SELECT 
			l.direction, l.run_no, c.code, c.name, cs.note,
			to_char(l.depart_time, 'HH24:MI') as depart_time,
			to_char(l.arrive_time, 'HH24:MI') as arrive_time,
			COALESCE(c.home_address, ''), s.address as site_address,
			COALESCE(v.display_name, '') as vehicle_name,
			s.name as site_name
		FROM schedule_legs l
		JOIN case_schedules cs ON cs.id = l.schedule_id
		JOIN cases c ON c.id = cs.case_id
		JOIN sites s ON s.id = cs.site_id
		LEFT JOIN vehicles v ON v.id = l.vehicle_id
		WHERE c.region = 'hsinchu'
		  AND c.status = 'active'
	`
	var args []interface{}
	argIdx := 1

	if siteID != nil {
		query += fmt.Sprintf(" AND s.id = $%d", argIdx)
		args = append(args, *siteID)
		argIdx++
	}
	if vehicleID != nil {
		query += fmt.Sprintf(" AND l.vehicle_id = $%d", argIdx)
		args = append(args, *vehicleID)
		argIdx++
	}

	query += " ORDER BY l.direction DESC, l.run_no ASC, l.depart_time ASC"

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query hsinchu schedule: %w", err)
	}
	defer rows.Close()

	var result []app.ReportHsinchuScheduleRow
	for rows.Next() {
		var item app.ReportHsinchuScheduleRow
		if err := rows.Scan(
			&item.Direction, &item.RunNo, &item.CaseCode, &item.CaseName, &item.Note,
			&item.DepartTime, &item.ArriveTime, &item.HomeAddress, &item.SiteAddress,
			&item.VehicleName, &item.SiteName,
		); err != nil {
			return nil, fmt.Errorf("failed to scan schedule item: %w", err)
		}
		result = append(result, item)
	}

	return result, nil
}
