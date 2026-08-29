package infra

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"ltc-system/apps/api/internal/modules/masterdata/app"
)

// vehicleRow 是 vehicles 資料表的一列。
type vehicleRow struct {
	ID          uuid.UUID
	PlateNo     string
	DisplayName string
	Region      string
	Status      string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (r vehicleRow) toApp() app.Vehicle {
	return app.Vehicle{
		ID:          r.ID,
		PlateNo:     r.PlateNo,
		DisplayName: r.DisplayName,
		Region:      r.Region,
		Status:      r.Status,
		CreatedAt:   r.CreatedAt,
		UpdatedAt:   r.UpdatedAt,
	}
}

const vehicleColumns = `id, plate_no, display_name, region, status, created_at, updated_at`

// VehicleRepository 提供 vehicles 資料表之存取操作。
type VehicleRepository struct {
	db *pgxpool.Pool
}

// NewVehicleRepository 建立 VehicleRepository 實例。
func NewVehicleRepository(db *pgxpool.Pool) *VehicleRepository {
	return &VehicleRepository{db: db}
}

// List 取得車輛清單。
func (r *VehicleRepository) List(ctx context.Context, region, q string, page, pageSize int) ([]app.Vehicle, int64, error) {
	if r.db == nil {
		return []app.Vehicle{}, 0, nil
	}
	offset := (page - 1) * pageSize
	query := `
		SELECT ` + vehicleColumns + `
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

	var list []app.Vehicle
	for rows.Next() {
		var v vehicleRow
		if err := rows.Scan(&v.ID, &v.PlateNo, &v.DisplayName, &v.Region, &v.Status, &v.CreatedAt, &v.UpdatedAt); err != nil {
			return nil, 0, err
		}
		list = append(list, v.toApp())
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
func (r *VehicleRepository) GetByID(ctx context.Context, id uuid.UUID) (*app.Vehicle, error) {
	return r.getOne(ctx, `SELECT `+vehicleColumns+` FROM vehicles WHERE id = $1`, id)
}

// GetByDisplayName 依顯示名稱尋找（支援匯入比對）。
func (r *VehicleRepository) GetByDisplayName(ctx context.Context, displayName string) (*app.Vehicle, error) {
	return r.getOne(ctx, `SELECT `+vehicleColumns+` FROM vehicles WHERE display_name = $1 LIMIT 1`, displayName)
}

func (r *VehicleRepository) getOne(ctx context.Context, query string, arg interface{}) (*app.Vehicle, error) {
	var v vehicleRow
	err := r.db.QueryRow(ctx, query, arg).
		Scan(&v.ID, &v.PlateNo, &v.DisplayName, &v.Region, &v.Status, &v.CreatedAt, &v.UpdatedAt)
	if err != nil {
		return nil, err
	}
	vehicle := v.toApp()
	return &vehicle, nil
}

// Create 新增車輛。
func (r *VehicleRepository) Create(ctx context.Context, v *app.Vehicle) error {
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
func (r *VehicleRepository) Update(ctx context.Context, v *app.Vehicle) error {
	query := `
		UPDATE vehicles
		SET plate_no = $2, display_name = $3, region = $4, status = $5, updated_at = now()
		WHERE id = $1
		RETURNING updated_at
	`
	return r.db.QueryRow(ctx, query, v.ID, v.PlateNo, v.DisplayName, v.Region, v.Status).
		Scan(&v.UpdatedAt)
}
