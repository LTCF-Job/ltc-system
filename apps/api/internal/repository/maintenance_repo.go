package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// MaintenanceLogEntity 代表 maintenance_logs 資料表實體。
type MaintenanceLogEntity struct {
	ID          uuid.UUID `json:"id"`
	VehicleID   uuid.UUID `json:"vehicleId"`
	VehicleName string    `json:"vehicleName,omitempty"`
	PlateNo     string    `json:"plateNo,omitempty"`
	ServiceDate time.Time `json:"serviceDate"`
	Mileage     float64   `json:"mileage"`
	Items       string    `json:"items"`
	Vendor      *string   `json:"vendor,omitempty"`
	Cost        float64   `json:"cost"`
	ReceiptURL  *string   `json:"receiptUrl,omitempty"`
	Note        *string   `json:"note,omitempty"`
	CreatedBy   uuid.UUID `json:"createdBy"`
	CreatedAt   time.Time `json:"createdAt"`
}

// MaintenanceRepository 提供車輛維修保養紀錄之資料存取。
type MaintenanceRepository struct {
	db *pgxpool.Pool
}

// NewMaintenanceRepository 建立 MaintenanceRepository 實例。
func NewMaintenanceRepository(db *pgxpool.Pool) *MaintenanceRepository {
	return &MaintenanceRepository{db: db}
}

// List 查詢維修保養紀錄清單，支援分頁與車輛／日期篩選。
func (r *MaintenanceRepository) List(ctx context.Context, page, pageSize int, vehicleID *uuid.UUID, startDate, endDate *time.Time) ([]MaintenanceLogEntity, int, error) {
	if r.db == nil {
		return []MaintenanceLogEntity{}, 0, nil
	}

	whereClause := "WHERE 1=1"
	args := []interface{}{}
	argIdx := 1

	if vehicleID != nil {
		whereClause += fmt.Sprintf(" AND m.vehicle_id = $%d", argIdx)
		args = append(args, *vehicleID)
		argIdx++
	}
	if startDate != nil {
		whereClause += fmt.Sprintf(" AND m.service_date >= $%d", argIdx)
		args = append(args, *startDate)
		argIdx++
	}
	if endDate != nil {
		whereClause += fmt.Sprintf(" AND m.service_date <= $%d", argIdx)
		args = append(args, *endDate)
		argIdx++
	}

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM maintenance_logs m %s", whereClause)
	var total int
	if err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count maintenance logs: %w", err)
	}

	offset := (page - 1) * pageSize
	dataQuery := fmt.Sprintf(`
		SELECT m.id, m.vehicle_id, v.display_name, v.plate_no, m.service_date,
		       m.mileage, m.items, m.vendor, m.cost, m.receipt_url, m.note,
		       m.created_by, m.created_at
		FROM maintenance_logs m
		JOIN vehicles v ON v.id = m.vehicle_id
		%s
		ORDER BY m.service_date DESC, m.created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argIdx, argIdx+1)

	args = append(args, pageSize, offset)
	rows, err := r.db.Query(ctx, dataQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query maintenance logs: %w", err)
	}
	defer rows.Close()

	list := []MaintenanceLogEntity{}
	for rows.Next() {
		var item MaintenanceLogEntity
		if err := rows.Scan(
			&item.ID, &item.VehicleID, &item.VehicleName, &item.PlateNo, &item.ServiceDate,
			&item.Mileage, &item.Items, &item.Vendor, &item.Cost, &item.ReceiptURL, &item.Note,
			&item.CreatedBy, &item.CreatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("failed to scan maintenance log: %w", err)
		}
		list = append(list, item)
	}

	return list, total, nil
}

// Create 新增單筆維修保養紀錄。
func (r *MaintenanceRepository) Create(ctx context.Context, item *MaintenanceLogEntity) error {
	if r.db == nil {
		item.ID = uuid.New()
		item.CreatedAt = time.Now()
		return nil
	}

	query := `
		INSERT INTO maintenance_logs (
			vehicle_id, service_date, mileage, items, vendor, cost, receipt_url, note, created_by
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, created_at
	`
	return r.db.QueryRow(ctx, query,
		item.VehicleID, item.ServiceDate, item.Mileage, item.Items, item.Vendor,
		item.Cost, item.ReceiptURL, item.Note, item.CreatedBy,
	).Scan(&item.ID, &item.CreatedAt)
}

// Update 修改維修保養紀錄。
func (r *MaintenanceRepository) Update(ctx context.Context, item *MaintenanceLogEntity) error {
	if r.db == nil {
		return nil
	}

	query := `
		UPDATE maintenance_logs
		SET vehicle_id = $1, service_date = $2, mileage = $3, items = $4,
		    vendor = $5, cost = $6, receipt_url = $7, note = $8
		WHERE id = $9
	`
	cmdTag, err := r.db.Exec(ctx, query,
		item.VehicleID, item.ServiceDate, item.Mileage, item.Items,
		item.Vendor, item.Cost, item.ReceiptURL, item.Note, item.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update maintenance log: %w", err)
	}
	if cmdTag.RowsAffected() == 0 {
		return fmt.Errorf("maintenance log not found")
	}
	return nil
}

// Delete 刪除維修保養紀錄。
func (r *MaintenanceRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if r.db == nil {
		return nil
	}

	query := "DELETE FROM maintenance_logs WHERE id = $1"
	cmdTag, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete maintenance log: %w", err)
	}
	if cmdTag.RowsAffected() == 0 {
		return fmt.Errorf("maintenance log not found")
	}
	return nil
}
