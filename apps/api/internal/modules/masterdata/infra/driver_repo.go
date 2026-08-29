package infra

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"ltc-system/apps/api/internal/modules/masterdata/app"
)

// driverRow 是 drivers 資料表的一列。
type driverRow struct {
	ID               uuid.UUID
	Code             string
	Name             string
	NameNormalized   string
	NationalIDCipher []byte
	NationalIDHMAC   []byte
	NationalIDMasked string
	Email            *string
	Region           string
	Status           string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

func (r driverRow) toApp() app.Driver {
	return app.Driver{
		ID:               r.ID,
		Code:             r.Code,
		Name:             r.Name,
		NameNormalized:   r.NameNormalized,
		NationalIDCipher: r.NationalIDCipher,
		NationalIDHMAC:   r.NationalIDHMAC,
		NationalIDMasked: r.NationalIDMasked,
		Email:            r.Email,
		Region:           r.Region,
		Status:           r.Status,
		CreatedAt:        r.CreatedAt,
		UpdatedAt:        r.UpdatedAt,
	}
}

func (r *driverRow) scanTargets() []interface{} {
	return []interface{}{
		&r.ID, &r.Code, &r.Name, &r.NameNormalized, &r.NationalIDCipher, &r.NationalIDHMAC, &r.NationalIDMasked,
		&r.Email, &r.Region, &r.Status, &r.CreatedAt, &r.UpdatedAt,
	}
}

const driverColumns = `id, code, name, name_normalized, national_id_cipher, national_id_hmac, national_id_masked,
	       email, region, status, created_at, updated_at`

// DriverRepository 提供 drivers 與 driver_assignments 資料表之存取操作。
type DriverRepository struct {
	db *pgxpool.Pool
}

// NewDriverRepository 建立 DriverRepository 實例。
func NewDriverRepository(db *pgxpool.Pool) *DriverRepository {
	return &DriverRepository{db: db}
}

// List 取得司機清單。
func (r *DriverRepository) List(ctx context.Context, region, q string, page, pageSize int) ([]app.Driver, int64, error) {
	if r.db == nil {
		return []app.Driver{}, 0, nil
	}
	offset := (page - 1) * pageSize
	query := `
		SELECT ` + driverColumns + `
		FROM drivers
		WHERE ($1 = '' OR region = $1)
		  AND ($2 = '' OR name ILIKE '%' || $2 || '%' OR code ILIKE '%' || $2 || '%' OR COALESCE(email, '') ILIKE '%' || $2 || '%')
		ORDER BY code ASC
		LIMIT $3 OFFSET $4
	`
	rows, err := r.db.Query(ctx, query, region, q, pageSize, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query drivers: %w", err)
	}
	defer rows.Close()

	var list []app.Driver
	for rows.Next() {
		var d driverRow
		if err := rows.Scan(d.scanTargets()...); err != nil {
			return nil, 0, err
		}
		list = append(list, d.toApp())
	}

	var total int64
	countQuery := `
		SELECT COUNT(*) FROM drivers
		WHERE ($1 = '' OR region = $1)
		  AND ($2 = '' OR name ILIKE '%' || $2 || '%' OR code ILIKE '%' || $2 || '%' OR COALESCE(email, '') ILIKE '%' || $2 || '%')
	`
	_ = r.db.QueryRow(ctx, countQuery, region, q).Scan(&total)

	return list, total, nil
}

// GetByID 依 UUID 取得司機。
func (r *DriverRepository) GetByID(ctx context.Context, id uuid.UUID) (*app.Driver, error) {
	return r.getOne(ctx, `SELECT `+driverColumns+` FROM drivers WHERE id = $1`, id)
}

// GetByHMAC 依身分證 HMAC 索引檢查是否已存在。
func (r *DriverRepository) GetByHMAC(ctx context.Context, hmac []byte) (*app.Driver, error) {
	return r.getOne(ctx, `SELECT `+driverColumns+` FROM drivers WHERE national_id_hmac = $1 LIMIT 1`, hmac)
}

// GetByNameNormalized 依正規化姓名搜尋司機。
func (r *DriverRepository) GetByNameNormalized(ctx context.Context, nameNorm string) (*app.Driver, error) {
	return r.getOne(ctx, `SELECT `+driverColumns+` FROM drivers WHERE name_normalized = $1 LIMIT 1`, nameNorm)
}

func (r *DriverRepository) getOne(ctx context.Context, query string, args ...interface{}) (*app.Driver, error) {
	var d driverRow
	if err := r.db.QueryRow(ctx, query, args...).Scan(d.scanTargets()...); err != nil {
		return nil, err
	}
	driver := d.toApp()
	return &driver, nil
}

// Create 新增司機。
func (r *DriverRepository) Create(ctx context.Context, d *app.Driver) error {
	query := `
		INSERT INTO drivers (
			id, code, name, name_normalized, national_id_cipher, national_id_hmac, national_id_masked,
			email, region, status
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING created_at, updated_at
	`
	if d.ID == uuid.Nil {
		d.ID = uuid.New()
	}
	return r.db.QueryRow(ctx, query,
		d.ID, d.Code, d.Name, d.NameNormalized, d.NationalIDCipher, d.NationalIDHMAC, d.NationalIDMasked,
		d.Email, d.Region, d.Status,
	).Scan(&d.CreatedAt, &d.UpdatedAt)
}

// Update 修改司機基本資料。
func (r *DriverRepository) Update(ctx context.Context, d *app.Driver) error {
	query := `
		UPDATE drivers
		SET name = $2, name_normalized = $3, email = $4, region = $5, status = $6, updated_at = now()
		WHERE id = $1
		RETURNING updated_at
	`
	return r.db.QueryRow(ctx, query, d.ID, d.Name, d.NameNormalized, d.Email, d.Region, d.Status).
		Scan(&d.UpdatedAt)
}

// AssignVehicle 建立或更新司機車輛指派期間。
func (r *DriverRepository) AssignVehicle(ctx context.Context, a *app.DriverAssignment) error {
	query := `
		INSERT INTO driver_assignments (
			id, driver_id, vehicle_id, is_primary, effective_range
		) VALUES ($1, $2, $3, $4, daterange($5, $6, '[]'))
		RETURNING created_at
	`
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	return r.db.QueryRow(ctx, query, a.ID, a.DriverID, a.VehicleID, a.IsPrimary, a.EffectiveFrom, a.EffectiveTo).
		Scan(&a.CreatedAt)
}

// GetPrimaryDriverForVehicleOnDate 查詢某車輛在特定日期的主要司機。
func (r *DriverRepository) GetPrimaryDriverForVehicleOnDate(ctx context.Context, vehicleID uuid.UUID, serviceDate time.Time) (*app.Driver, error) {
	query := `
		SELECT d.id, d.code, d.name, d.name_normalized, d.national_id_cipher, d.national_id_hmac, d.national_id_masked,
		       d.email, d.region, d.status, d.created_at, d.updated_at
		FROM driver_assignments a
		JOIN drivers d ON a.driver_id = d.id
		WHERE a.vehicle_id = $1
		  AND a.is_primary = true
		  AND a.effective_range @> $2::date
		LIMIT 1
	`
	return r.getOne(ctx, query, vehicleID, serviceDate)
}
