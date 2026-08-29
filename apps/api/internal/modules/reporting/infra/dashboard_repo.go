package repository

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// VehicleTripTrendData 代表車輛趟數趨勢資料。
type VehicleTripTrendData struct {
	VehicleName string
	PlateNo     string
	TripCount   int
}

// DashboardRepository 提供儀表板相關資料庫查詢。
type DashboardRepository struct {
	db *pgxpool.Pool
}

// NewDashboardRepository 建立 DashboardRepository 實例。
func NewDashboardRepository(db *pgxpool.Pool) *DashboardRepository {
	return &DashboardRepository{db: db}
}

// GetActiveCasesCount 查詢有效個案總數。
func (r *DashboardRepository) GetActiveCasesCount(ctx context.Context) (int, error) {
	if r.db == nil {
		return 0, nil
	}
	var count int
	err := r.db.QueryRow(ctx, "SELECT COUNT(*) FROM cases WHERE status = 'active'").Scan(&count)
	return count, err
}

// GetReportedTripsCount 查詢指定區間之已搭乘總趟數。
func (r *DashboardRepository) GetReportedTripsCount(ctx context.Context, start, end time.Time) (int, error) {
	if r.db == nil {
		return 0, nil
	}
	var count int
	query := `
		SELECT COUNT(*) 
		FROM ride_records 
		WHERE service_date >= $1 AND service_date < $2 AND effective_status = 'boarded'
	`
	err := r.db.QueryRow(ctx, query, start, end).Scan(&count)
	return count, err
}

// GetPendingConflictsCount 查詢未裁決之混車衝突數量。
func (r *DashboardRepository) GetPendingConflictsCount(ctx context.Context) (int, error) {
	if r.db == nil {
		return 0, nil
	}
	var count int
	query := `
		SELECT COUNT(*) 
		FROM ride_records 
		WHERE has_conflict = true AND conflict_resolved_at IS NULL
	`
	err := r.db.QueryRow(ctx, query).Scan(&count)
	return count, err
}

// GetPendingFormColumnsCount 查詢待對應之表單欄位數量。
func (r *DashboardRepository) GetPendingFormColumnsCount(ctx context.Context) (int, error) {
	if r.db == nil {
		return 0, nil
	}
	var count int
	query := `
		SELECT COUNT(*) 
		FROM form_columns 
		WHERE mapping_status = 'pending'
	`
	err := r.db.QueryRow(ctx, query).Scan(&count)
	return count, err
}

// GetVehicleTripTrends 查詢各車輛於指定區間之趟數趨勢。
func (r *DashboardRepository) GetVehicleTripTrends(ctx context.Context, start, end time.Time) ([]VehicleTripTrendData, error) {
	if r.db == nil {
		return []VehicleTripTrendData{}, nil
	}
	query := `
		SELECT v.display_name, v.plate_no, COUNT(r.id) as trips
		FROM vehicles v
		LEFT JOIN ride_records r ON r.vehicle_id = v.id 
		  AND r.service_date >= $1 AND r.service_date < $2 
		  AND r.effective_status = 'boarded'
		GROUP BY v.id, v.display_name, v.plate_no
		ORDER BY v.display_name ASC
	`
	rows, err := r.db.Query(ctx, query, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var trends []VehicleTripTrendData
	for rows.Next() {
		var item VehicleTripTrendData
		if err := rows.Scan(&item.VehicleName, &item.PlateNo, &item.TripCount); err == nil {
			trends = append(trends, item)
		}
	}
	return trends, nil
}

// GetAttendanceDistribution 查詢指定區間之司機出勤分佈。
func (r *DashboardRepository) GetAttendanceDistribution(ctx context.Context, start, end time.Time) (map[string]int, error) {
	dist := make(map[string]int)
	if r.db == nil {
		return dist, nil
	}
	query := `
		SELECT status, COUNT(*)
		FROM attendance_records
		WHERE record_date >= $1 AND record_date < $2
		GROUP BY status
	`
	rows, err := r.db.Query(ctx, query, start, end)
	if err != nil {
		return dist, err
	}
	defer rows.Close()

	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err == nil {
			dist[status] = count
		}
	}
	return dist, nil
}
