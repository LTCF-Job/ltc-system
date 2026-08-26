package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// VehicleRepository 提供 vehicles 資料表之存取操作。
type VehicleRepository struct {
	db *pgxpool.Pool
}

// NewVehicleRepository 建立 VehicleRepository 實例。
func NewVehicleRepository(db *pgxpool.Pool) *VehicleRepository {
	return &VehicleRepository{db: db}
}

// List 取得車輛清單。
func (r *VehicleRepository) List(ctx context.Context, region, q string, page, pageSize int) ([]VehicleEntity, int64, error) {
	if r.db == nil {
		return []VehicleEntity{}, 0, nil
	}
	offset := (page - 1) * pageSize
	query := `
		SELECT id, plate_no, display_name, region, status, created_at, updated_at
		FROM vehicles
		WHERE ($1 = '' OR region = $1)
		  AND ($2 = '' OR plate_no ILIKE '%' || $2 || '%' OR display_name ILIKE '%' || $2 || '%')
		ORDER BY display_name ASC
		LIMIT $3 OFFSET $4
	`
	rows, err := r.db.Query(ctx, query, region, q, pageSize, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query vehicles: %w", err)
	}
	defer rows.Close()

	var list []VehicleEntity
	for rows.Next() {
		var v VehicleEntity
		if err := rows.Scan(&v.ID, &v.PlateNo, &v.DisplayName, &v.Region, &v.Status, &v.CreatedAt, &v.UpdatedAt); err != nil {
			return nil, 0, err
		}
		list = append(list, v)
	}

	var total int64
	countQuery := `
		SELECT COUNT(*) FROM vehicles
		WHERE ($1 = '' OR region = $1)
		  AND ($2 = '' OR plate_no ILIKE '%' || $2 || '%' OR display_name ILIKE '%' || $2 || '%')
	`
	_ = r.db.QueryRow(ctx, countQuery, region, q).Scan(&total)

	return list, total, nil
}

// GetByID 依 UUID 取得車輛。
func (r *VehicleRepository) GetByID(ctx context.Context, id uuid.UUID) (*VehicleEntity, error) {
	query := `
		SELECT id, plate_no, display_name, region, status, created_at, updated_at
		FROM vehicles WHERE id = $1
	`
	var v VehicleEntity
	err := r.db.QueryRow(ctx, query, id).Scan(&v.ID, &v.PlateNo, &v.DisplayName, &v.Region, &v.Status, &v.CreatedAt, &v.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &v, nil
}

// GetByDisplayName 依顯示名稱尋找（支援匯入比對）。
func (r *VehicleRepository) GetByDisplayName(ctx context.Context, displayName string) (*VehicleEntity, error) {
	query := `
		SELECT id, plate_no, display_name, region, status, created_at, updated_at
		FROM vehicles WHERE display_name = $1 LIMIT 1
	`
	var v VehicleEntity
	err := r.db.QueryRow(ctx, query, displayName).Scan(&v.ID, &v.PlateNo, &v.DisplayName, &v.Region, &v.Status, &v.CreatedAt, &v.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &v, nil
}

// Create 新增車輛。
func (r *VehicleRepository) Create(ctx context.Context, v *VehicleEntity) error {
	query := `
		INSERT INTO vehicles (id, plate_no, display_name, region, status)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING created_at, updated_at
	`
	if v.ID == uuid.Nil {
		v.ID = uuid.New()
	}
	return r.db.QueryRow(ctx, query, v.ID, v.PlateNo, v.DisplayName, v.Region, v.Status).
		Scan(&v.CreatedAt, &v.UpdatedAt)
}

// Update 修改車輛。
func (r *VehicleRepository) Update(ctx context.Context, v *VehicleEntity) error {
	query := `
		UPDATE vehicles
		SET plate_no = $2, display_name = $3, region = $4, status = $5, updated_at = now()
		WHERE id = $1
		RETURNING updated_at
	`
	return r.db.QueryRow(ctx, query, v.ID, v.PlateNo, v.DisplayName, v.Region, v.Status).
		Scan(&v.UpdatedAt)
}
