package infra

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"ltc-system/apps/api/internal/modules/ops/app"
)

// FuelRepository 提供車輛油資紀錄之資料存取。
type FuelRepository struct {
	db *pgxpool.Pool
}

// NewFuelRepository 建立 FuelRepository 實例。
func NewFuelRepository(db *pgxpool.Pool) *FuelRepository {
	return &FuelRepository{db: db}
}

// List 查詢油資紀錄清單，支援分頁與車輛／司機／日期／關鍵字模糊篩選。
func (r *FuelRepository) List(ctx context.Context, page, pageSize int, vehicleID, driverID *uuid.UUID, startDate, endDate *time.Time, q string) ([]app.FuelLog, int, error) {
	if r.db == nil {
		return []app.FuelLog{}, 0, nil
	}

	whereClause := "WHERE 1=1"
	args := []interface{}{}
	argIdx := 1

	if vehicleID != nil {
		whereClause += fmt.Sprintf(" AND f.vehicle_id = $%d", argIdx)
		args = append(args, *vehicleID)
		argIdx++
	}
	if driverID != nil {
		whereClause += fmt.Sprintf(" AND f.driver_id = $%d", argIdx)
		args = append(args, *driverID)
		argIdx++
	}
	if startDate != nil {
		whereClause += fmt.Sprintf(" AND f.fuel_date >= $%d", argIdx)
		args = append(args, *startDate)
		argIdx++
	}
	if endDate != nil {
		whereClause += fmt.Sprintf(" AND f.fuel_date <= $%d", argIdx)
		args = append(args, *endDate)
		argIdx++
	}
	if q != "" {
		whereClause += fmt.Sprintf(" AND (v.plate_no ILIKE '%%' || $%d || '%%' OR v.display_name ILIKE '%%' || $%d || '%%' OR COALESCE(d.name, '') ILIKE '%%' || $%d || '%%')", argIdx, argIdx, argIdx)
		args = append(args, q)
		argIdx++
	}

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM fuel_logs f JOIN vehicles v ON v.id = f.vehicle_id LEFT JOIN drivers d ON d.id = f.driver_id %s", whereClause)
	var total int
	if err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count fuel logs: %w", err)
	}

	offset := (page - 1) * pageSize
	dataQuery := fmt.Sprintf(`
		SELECT f.id, f.vehicle_id, v.display_name, v.plate_no,
		       f.driver_id, d.name, f.fuel_date, f.liters, f.cost,
		       f.receipt_url, f.created_by, f.created_at
		FROM fuel_logs f
		JOIN vehicles v ON v.id = f.vehicle_id
		LEFT JOIN drivers d ON d.id = f.driver_id
		%s
		ORDER BY f.fuel_date DESC, f.created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argIdx, argIdx+1)

	args = append(args, pageSize, offset)
	rows, err := r.db.Query(ctx, dataQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query fuel logs: %w", err)
	}
	defer rows.Close()

	list := []app.FuelLog{}
	for rows.Next() {
		var item app.FuelLog
		if err := rows.Scan(
			&item.ID, &item.VehicleID, &item.VehicleName, &item.PlateNo,
			&item.DriverID, &item.DriverName, &item.FuelDate, &item.Liters,
			&item.Cost, &item.ReceiptURL, &item.CreatedBy, &item.CreatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("failed to scan fuel log: %w", err)
		}
		list = append(list, item)
	}

	return list, total, nil
}

// Create 新增單筆油資紀錄。
func (r *FuelRepository) Create(ctx context.Context, item *app.FuelLog) error {
	if r.db == nil {
		item.ID = uuid.New()
		item.CreatedAt = time.Now()
		return nil
	}

	query := `
		INSERT INTO fuel_logs (
			vehicle_id, driver_id, fuel_date, liters, cost, receipt_url, created_by
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at
	`
	return r.db.QueryRow(ctx, query,
		item.VehicleID, item.DriverID, item.FuelDate, item.Liters, item.Cost,
		item.ReceiptURL, item.CreatedBy,
	).Scan(&item.ID, &item.CreatedAt)
}

// Update 修改油資紀錄。
func (r *FuelRepository) Update(ctx context.Context, item *app.FuelLog) error {
	if r.db == nil {
		return nil
	}

	query := `
		UPDATE fuel_logs
		SET vehicle_id = $1, driver_id = $2, fuel_date = $3,
		    liters = $4, cost = $5, receipt_url = $6
		WHERE id = $7
	`
	cmdTag, err := r.db.Exec(ctx, query,
		item.VehicleID, item.DriverID, item.FuelDate, item.Liters, item.Cost,
		item.ReceiptURL, item.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update fuel log: %w", err)
	}
	if cmdTag.RowsAffected() == 0 {
		return fmt.Errorf("fuel log not found")
	}
	return nil
}

// Delete 刪除油資紀錄。
func (r *FuelRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if r.db == nil {
		return nil
	}

	query := "DELETE FROM fuel_logs WHERE id = $1"
	cmdTag, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete fuel log: %w", err)
	}
	if cmdTag.RowsAffected() == 0 {
		return fmt.Errorf("fuel log not found")
	}
	return nil
}
