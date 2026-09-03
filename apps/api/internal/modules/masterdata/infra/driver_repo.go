package infra

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"ltc-system/apps/api/internal/modules/masterdata/app"
	"ltc-system/apps/api/internal/platform/pgxdb"
)

// driverRow 是 drivers 資料表的一列。
type driverRow struct {
	ID                uuid.UUID
	Code              string
	Name              string
	NameNormalized    string
	NationalIDCipher  []byte
	NationalIDHMAC    []byte
	NationalIDMasked  string
	Email             *string
	Region            string
	Status            string
	LicenseClass      *string
	LicenseExpiryDate *time.Time
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

func (r driverRow) toApp() app.Driver {
	return app.Driver{
		ID:                r.ID,
		Code:              r.Code,
		Name:              r.Name,
		NameNormalized:    r.NameNormalized,
		NationalIDCipher:  r.NationalIDCipher,
		NationalIDHMAC:    r.NationalIDHMAC,
		NationalIDMasked:  r.NationalIDMasked,
		Email:             r.Email,
		Region:            r.Region,
		Status:            r.Status,
		LicenseClass:      r.LicenseClass,
		LicenseExpiryDate: r.LicenseExpiryDate,
		CreatedAt:         r.CreatedAt,
		UpdatedAt:         r.UpdatedAt,
	}
}

func (r *driverRow) scanTargets() []interface{} {
	return []interface{}{
		&r.ID, &r.Code, &r.Name, &r.NameNormalized, &r.NationalIDCipher, &r.NationalIDHMAC, &r.NationalIDMasked,
		&r.Email, &r.Region, &r.Status, &r.LicenseClass, &r.LicenseExpiryDate, &r.CreatedAt, &r.UpdatedAt,
	}
}

const driverColumns = `id, code, name, name_normalized, national_id_cipher, national_id_hmac, national_id_masked,
	       email, region, status, license_class, license_expiry_date, created_at, updated_at`

const driverColumnsWithAlias = `d.id, d.code, d.name, d.name_normalized, d.national_id_cipher, d.national_id_hmac,
	       d.national_id_masked, d.email, d.region, d.status, d.license_class, d.license_expiry_date,
	       d.created_at, d.updated_at`

// DriverRepository 提供 drivers 與 driver_assignments 資料表之存取操作。
type DriverRepository struct {
	db *pgxpool.Pool
}

// NewDriverRepository 建立 DriverRepository 實例。
func NewDriverRepository(db *pgxpool.Pool) *DriverRepository {
	return &DriverRepository{db: db}
}

// List 取得司機清單。
func (r *DriverRepository) List(ctx context.Context, region, q, status string, page, pageSize int) ([]app.Driver, int64, error) {
	if r.db == nil {
		return []app.Driver{}, 0, nil
	}
	offset := (page - 1) * pageSize
	query := `
		SELECT ` + driverColumns + `
		FROM drivers
		WHERE deleted_at IS NULL
		  AND ($1 = '' OR region = $1)
		  AND ($2 = '' OR name ILIKE '%' || $2 || '%' OR code ILIKE '%' || $2 || '%' OR COALESCE(email, '') ILIKE '%' || $2 || '%')
		  AND ($3 = '' OR status = $3)
		ORDER BY code ASC
		LIMIT $4 OFFSET $5
	`
	rows, err := r.db.Query(ctx, query, region, q, status, pageSize, offset)
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
		WHERE deleted_at IS NULL
		  AND ($1 = '' OR region = $1)
		  AND ($2 = '' OR name ILIKE '%' || $2 || '%' OR code ILIKE '%' || $2 || '%' OR COALESCE(email, '') ILIKE '%' || $2 || '%')
		  AND ($3 = '' OR status = $3)
	`
	_ = r.db.QueryRow(ctx, countQuery, region, q, status).Scan(&total)

	return list, total, nil
}

// GetByID 依 UUID 取得司機。
func (r *DriverRepository) GetByID(ctx context.Context, id uuid.UUID) (*app.Driver, error) {
	return r.getOne(ctx, `SELECT `+driverColumns+` FROM drivers WHERE id = $1 AND deleted_at IS NULL`, id)
}

// GetByHMAC 依身分證 HMAC 索引檢查是否已存在。
func (r *DriverRepository) GetByHMAC(ctx context.Context, hmac []byte) (*app.Driver, error) {
	return r.getOne(ctx, `SELECT `+driverColumns+` FROM drivers WHERE national_id_hmac = $1 AND deleted_at IS NULL LIMIT 1`, hmac)
}

// GetByNameNormalized 依正規化姓名搜尋司機。
func (r *DriverRepository) GetByNameNormalized(ctx context.Context, nameNorm string) (*app.Driver, error) {
	return r.getOne(ctx, `SELECT `+driverColumns+` FROM drivers WHERE name_normalized = $1 AND deleted_at IS NULL LIMIT 1`, nameNorm)
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
			email, region, status, license_class, license_expiry_date
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING created_at, updated_at
	`
	if d.ID == uuid.Nil {
		d.ID = uuid.New()
	}
	return r.db.QueryRow(ctx, query,
		d.ID, d.Code, d.Name, d.NameNormalized, d.NationalIDCipher, d.NationalIDHMAC, d.NationalIDMasked,
		d.Email, d.Region, d.Status, d.LicenseClass, d.LicenseExpiryDate,
	).Scan(&d.CreatedAt, &d.UpdatedAt)
}

// Update 修改司機基本資料。
func (r *DriverRepository) Update(ctx context.Context, d *app.Driver) error {
	query := `
		UPDATE drivers
		SET name = $2, name_normalized = $3, email = $4, region = $5, status = $6,
		    license_class = $7, license_expiry_date = $8, updated_at = now()
		WHERE id = $1 AND deleted_at IS NULL
		RETURNING updated_at
	`
	return r.db.QueryRow(ctx, query, d.ID, d.Name, d.NameNormalized, d.Email, d.Region, d.Status,
		d.LicenseClass, d.LicenseExpiryDate).
		Scan(&d.UpdatedAt)
}

// AssignVehicle 建立司機車輛指派期間。
func (r *DriverRepository) AssignVehicle(ctx context.Context, a *app.DriverAssignment) error {
	query := `
		INSERT INTO driver_assignments (
			id, driver_id, vehicle_id, effective_range
		) VALUES ($1, $2, $3, daterange($4, $5, '[]'))
		RETURNING created_at
	`
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	return r.db.QueryRow(ctx, query, a.ID, a.DriverID, a.VehicleID, a.EffectiveFrom, a.EffectiveTo).
		Scan(&a.CreatedAt)
}

// ListDriversForVehicleOnDate 查詢某車輛在特定日期生效的所有司機，依司機代碼排序。
func (r *DriverRepository) ListDriversForVehicleOnDate(ctx context.Context, vehicleID uuid.UUID, serviceDate time.Time) ([]app.Driver, error) {
	if r.db == nil {
		return nil, nil
	}
	query := `
		SELECT ` + driverColumnsWithAlias + `
		FROM driver_assignments a
		JOIN drivers d ON a.driver_id = d.id
		WHERE a.vehicle_id = $1
		  AND a.effective_range @> $2::date
		  AND d.deleted_at IS NULL
		ORDER BY d.code ASC
	`
	rows, err := r.db.Query(ctx, query, vehicleID, serviceDate)
	if err != nil {
		return nil, fmt.Errorf("failed to query drivers for vehicle: %w", err)
	}
	defer rows.Close()

	var list []app.Driver
	for rows.Next() {
		var d driverRow
		if err := rows.Scan(d.scanTargets()...); err != nil {
			return nil, err
		}
		list = append(list, d.toApp())
	}
	return list, rows.Err()
}

// ListByVehicleIDsOnDate 一次查出多台車在特定日期生效的司機，供車輛清單批次帶出。
func (r *DriverRepository) ListByVehicleIDsOnDate(ctx context.Context, vehicleIDs []uuid.UUID, on time.Time) (map[uuid.UUID][]app.Driver, error) {
	result := make(map[uuid.UUID][]app.Driver, len(vehicleIDs))
	if r.db == nil || len(vehicleIDs) == 0 {
		return result, nil
	}
	query := `
		SELECT a.vehicle_id, ` + driverColumnsWithAlias + `
		FROM driver_assignments a
		JOIN drivers d ON a.driver_id = d.id
		WHERE a.vehicle_id = ANY($1::uuid[])
		  AND a.effective_range @> $2::date
		  AND d.deleted_at IS NULL
		ORDER BY d.code ASC
	`
	rows, err := r.db.Query(ctx, query, pgxdb.UUIDStrings(vehicleIDs), on)
	if err != nil {
		return nil, fmt.Errorf("failed to query drivers by vehicles: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var vehicleID uuid.UUID
		var d driverRow
		if err := rows.Scan(append([]interface{}{&vehicleID}, d.scanTargets()...)...); err != nil {
			return nil, err
		}
		result[vehicleID] = append(result[vehicleID], d.toApp())
	}
	return result, rows.Err()
}

// ReplaceVehicleDrivers 以 effectiveFrom 為界，將車輛的司機集合換成 driverIDs。
func (r *DriverRepository) ReplaceVehicleDrivers(ctx context.Context, vehicleID uuid.UUID, driverIDs []uuid.UUID, effectiveFrom time.Time) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	driverIDStrings := pgxdb.UUIDStrings(driverIDs)

	// 被移出本車的司機，其指派在 effectiveFrom 當日結束；尚未生效的指派直接刪除
	if _, err := tx.Exec(ctx, `
		DELETE FROM driver_assignments
		WHERE vehicle_id = $1 AND NOT (driver_id = ANY($2::uuid[])) AND lower(effective_range) >= $3::date
	`, vehicleID, driverIDStrings, effectiveFrom); err != nil {
		return fmt.Errorf("failed to drop future assignments: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		UPDATE driver_assignments
		SET effective_range = daterange(lower(effective_range), $3::date, '[)')
		WHERE vehicle_id = $1 AND NOT (driver_id = ANY($2::uuid[])) AND effective_range @> $3::date
	`, vehicleID, driverIDStrings, effectiveFrom); err != nil {
		return fmt.Errorf("failed to close assignments: %w", err)
	}

	// 已在本車且指派仍生效的司機不動，避免每次儲存都切出一段沒有意義的歷史
	rows, err := tx.Query(ctx, `
		SELECT driver_id FROM driver_assignments
		WHERE vehicle_id = $1 AND effective_range @> $2::date
	`, vehicleID, effectiveFrom)
	if err != nil {
		return fmt.Errorf("failed to query current assignments: %w", err)
	}
	unchanged := make(map[uuid.UUID]bool)
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return err
		}
		unchanged[id] = true
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}

	// 新加入的司機：先收掉他在別台車尚未結束的指派，再掛到本車
	for _, driverID := range driverIDs {
		if unchanged[driverID] {
			continue
		}
		if _, err := tx.Exec(ctx, `
			DELETE FROM driver_assignments
			WHERE driver_id = $1 AND lower(effective_range) >= $2::date
		`, driverID, effectiveFrom); err != nil {
			return fmt.Errorf("failed to drop future driver assignment: %w", err)
		}
		if _, err := tx.Exec(ctx, `
			UPDATE driver_assignments
			SET effective_range = daterange(lower(effective_range), $2::date, '[)')
			WHERE driver_id = $1 AND effective_range @> $2::date
		`, driverID, effectiveFrom); err != nil {
			return fmt.Errorf("failed to close driver assignment: %w", err)
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO driver_assignments (id, driver_id, vehicle_id, effective_range)
			VALUES ($1, $2, $3, daterange($4::date, NULL, '[)'))
		`, uuid.New(), driverID, vehicleID, effectiveFrom); err != nil {
			return fmt.Errorf("failed to insert assignment: %w", err)
		}
	}

	return tx.Commit(ctx)
}

// SoftDelete 軟刪除司機，回傳 false 代表該筆已被刪除過。
func (r *DriverRepository) SoftDelete(ctx context.Context, id, actorID uuid.UUID) (bool, error) {
	tag, err := r.db.Exec(ctx, `UPDATE drivers SET deleted_at = now(), deleted_by = $2 WHERE id = $1 AND deleted_at IS NULL`, id, actorID)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() == 1, nil
}

// CloseActiveAssignments 收斂該司機所有生效中車輛指派的區間至今天。
func (r *DriverRepository) CloseActiveAssignments(ctx context.Context, driverID uuid.UUID) error {
	_, err := r.db.Exec(ctx, `
		UPDATE driver_assignments
		SET effective_range = daterange(lower(effective_range), CURRENT_DATE, '[)')
		WHERE driver_id = $1 AND upper_inf(effective_range)
	`, driverID)
	return err
}
