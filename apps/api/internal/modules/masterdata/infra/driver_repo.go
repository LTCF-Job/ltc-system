package infra

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"ltc-system/apps/api/internal/modules/masterdata/app"
	"ltc-system/apps/api/internal/platform/clock"
	"ltc-system/apps/api/internal/platform/pgxdb"
)

// driverRow 是 drivers 資料表的一列。
type driverRow struct {
	ID                     uuid.UUID
	Name                   string
	NameNormalized         string
	NationalIDCipher       []byte
	NationalIDHMAC         []byte
	NationalIDMasked       string
	Email                  *string
	Status                 string
	LicenseClass           *string
	LicenseExpiryDate      *time.Time
	Gender                 *string
	BirthDate              *time.Time
	HasProfessionalLicense bool
	EmploymentDate         *time.Time
	HasTransferCert        bool
	Remarks                *string
	CreatedAt              time.Time
	UpdatedAt              time.Time
}

func (r driverRow) toApp() app.Driver {
	return app.Driver{
		ID:                     r.ID,
		Name:                   r.Name,
		NameNormalized:         r.NameNormalized,
		NationalIDCipher:       r.NationalIDCipher,
		NationalIDHMAC:         r.NationalIDHMAC,
		NationalIDMasked:       r.NationalIDMasked,
		Email:                  r.Email,
		Status:                 r.Status,
		LicenseClass:           r.LicenseClass,
		LicenseExpiryDate:      r.LicenseExpiryDate,
		Gender:                 r.Gender,
		BirthDate:              r.BirthDate,
		HasProfessionalLicense: r.HasProfessionalLicense,
		EmploymentDate:         r.EmploymentDate,
		HasTransferCert:        r.HasTransferCert,
		Remarks:                r.Remarks,
		CreatedAt:              r.CreatedAt,
		UpdatedAt:              r.UpdatedAt,
	}
}

func (r *driverRow) scanTargets() []interface{} {
	return []interface{}{
		&r.ID, &r.Name, &r.NameNormalized, &r.NationalIDCipher, &r.NationalIDHMAC, &r.NationalIDMasked,
		&r.Email, &r.Status, &r.LicenseClass, &r.LicenseExpiryDate,
		&r.Gender, &r.BirthDate, &r.HasProfessionalLicense, &r.EmploymentDate, &r.HasTransferCert, &r.Remarks,
		&r.CreatedAt, &r.UpdatedAt,
	}
}

const driverColumns = `id, name, name_normalized, national_id_cipher, national_id_hmac, national_id_masked,
	       email, status, license_class, license_expiry_date,
	       gender, birth_date, has_professional_license, employment_date, has_transfer_cert, remarks,
	       created_at, updated_at`

const driverColumnsWithAlias = `d.id, d.name, d.name_normalized, d.national_id_cipher, d.national_id_hmac,
	       d.national_id_masked, d.email, d.status, d.license_class, d.license_expiry_date,
	       d.gender, d.birth_date, d.has_professional_license, d.employment_date, d.has_transfer_cert, d.remarks,
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
func (r *DriverRepository) List(ctx context.Context, q, status string, page, pageSize int) ([]app.Driver, int64, error) {
	if r.db == nil {
		return nil, 0, fmt.Errorf("driver database is not configured")
	}
	offset := (page - 1) * pageSize
	query := `
		SELECT ` + driverColumns + `
		FROM drivers
		WHERE deleted_at IS NULL
		  AND ($1 = '' OR name ILIKE '%' || $1 || '%' OR COALESCE(email, '') ILIKE '%' || $1 || '%')
		  AND ($2 = '' OR status = $2)
		ORDER BY name ASC
		LIMIT $3 OFFSET $4
	`
	rows, err := r.db.Query(ctx, query, q, status, pageSize, offset)
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
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("failed to iterate drivers: %w", err)
	}

	var total int64
	countQuery := `
		SELECT COUNT(*) FROM drivers
		WHERE deleted_at IS NULL
		  AND ($1 = '' OR name ILIKE '%' || $1 || '%' OR COALESCE(email, '') ILIKE '%' || $1 || '%')
		  AND ($2 = '' OR status = $2)
	`
	if err := r.db.QueryRow(ctx, countQuery, q, status).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count drivers: %w", err)
	}

	return list, total, nil
}

// ListAllActive 取得所有未刪除且啟用的司機，供完整業務資料集使用。
func (r *DriverRepository) ListAllActive(ctx context.Context) ([]app.Driver, error) {
	return r.listAllActive(ctx, "")
}

// ListAllActiveByQuery 取得所有啟用且姓名符合搜尋字串的司機。
func (r *DriverRepository) ListAllActiveByQuery(ctx context.Context, q string) ([]app.Driver, error) {
	return r.listAllActive(ctx, q)
}

func (r *DriverRepository) listAllActive(ctx context.Context, q string) ([]app.Driver, error) {
	if r.db == nil {
		return nil, fmt.Errorf("driver database is not configured")
	}
	query := `
		SELECT ` + driverColumns + `
		FROM drivers
		WHERE deleted_at IS NULL
		  AND status = 'active'
		  AND ($1 = '' OR name ILIKE '%' || $1 || '%')
		ORDER BY name ASC
	`
	rows, err := r.db.Query(ctx, query, q)
	if err != nil {
		return nil, fmt.Errorf("failed to query all active drivers: %w", err)
	}
	defer rows.Close()

	list := make([]app.Driver, 0)
	for rows.Next() {
		var d driverRow
		if err := rows.Scan(d.scanTargets()...); err != nil {
			return nil, fmt.Errorf("scan active driver: %w", err)
		}
		list = append(list, d.toApp())
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate active drivers: %w", err)
	}
	return list, nil
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
	if r.db == nil {
		return nil, fmt.Errorf("driver database is not configured")
	}
	var d driverRow
	if err := r.db.QueryRow(ctx, query, args...).Scan(d.scanTargets()...); err != nil {
		if err == pgx.ErrNoRows {
			return nil, app.ErrDriverNotFound
		}
		return nil, err
	}
	driver := d.toApp()
	return &driver, nil
}

// Create 新增司機。
func (r *DriverRepository) Create(ctx context.Context, d *app.Driver) error {
	query := `
		INSERT INTO drivers (
			id, name, name_normalized, national_id_cipher, national_id_hmac, national_id_masked,
			email, status, license_class, license_expiry_date,
			gender, birth_date, has_professional_license, employment_date, has_transfer_cert, remarks
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
		RETURNING created_at, updated_at
	`
	if d.ID == uuid.Nil {
		d.ID = uuid.New()
	}
	db := pgxdb.FromContext(ctx, r.db)
	err := db.QueryRow(ctx, query,
		d.ID, d.Name, d.NameNormalized, d.NationalIDCipher, d.NationalIDHMAC, d.NationalIDMasked,
		d.Email, d.Status, d.LicenseClass, d.LicenseExpiryDate,
		d.Gender, d.BirthDate, d.HasProfessionalLicense, d.EmploymentDate, d.HasTransferCert, d.Remarks,
	).Scan(&d.CreatedAt, &d.UpdatedAt)
	return handleDriverDBError(err)
}

// Update 修改司機基本資料。
func (r *DriverRepository) Update(ctx context.Context, d *app.Driver) error {
	query := `
		UPDATE drivers
		SET name = $2, name_normalized = $3, email = $4, status = $5,
		    national_id_cipher = $6, national_id_hmac = $7, national_id_masked = $8,
		    license_class = $9, license_expiry_date = $10,
		    gender = $11, birth_date = $12, has_professional_license = $13, employment_date = $14,
		    has_transfer_cert = $15, remarks = $16,
		    updated_at = now()
		WHERE id = $1 AND deleted_at IS NULL
		RETURNING updated_at
	`
	db := pgxdb.FromContext(ctx, r.db)
	err := db.QueryRow(ctx, query, d.ID, d.Name, d.NameNormalized, d.Email, d.Status,
		d.NationalIDCipher, d.NationalIDHMAC, d.NationalIDMasked,
		d.LicenseClass, d.LicenseExpiryDate,
		d.Gender, d.BirthDate, d.HasProfessionalLicense, d.EmploymentDate, d.HasTransferCert, d.Remarks).
		Scan(&d.UpdatedAt)
	return handleDriverDBError(err)
}

// handleDriverDBError 把身分證唯一索引的衝突轉成業務錯誤；
// 沒有對映時，重複身分證會以 500 冒出去而不是可讀的欄位錯誤。
func handleDriverDBError(err error) error {
	if err == nil {
		return nil
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" && strings.Contains(pgErr.ConstraintName, "national_id_hmac") {
		return app.ErrDuplicateNationalID
	}
	return err
}

// AssignVehicle 建立司機車輛指派期間。指定的司機/車輛不存在（外鍵違反）回傳
// app.ErrAssignmentReferenceInvalid；與既有指派期間重疊（違反不重疊限制）回傳
// app.ErrAssignmentOverlap，讓 transport 層能分流成正確的 HTTP 狀態碼。
func (r *DriverRepository) AssignVehicle(ctx context.Context, a *app.DriverAssignment) error {
	query := `
		INSERT INTO driver_assignments (
			id, driver_id, vehicle_id, effective_range
		) VALUES ($1, $2, $3, daterange($4, $5, '[)'))
		RETURNING created_at
	`
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	var exclusiveTo *time.Time
	if a.EffectiveTo != nil {
		end := a.EffectiveTo.AddDate(0, 0, 1)
		exclusiveTo = &end
	}
	db := pgxdb.FromContext(ctx, r.db)
	err := db.QueryRow(ctx, query, a.ID, a.DriverID, a.VehicleID, a.EffectiveFrom, exclusiveTo).
		Scan(&a.CreatedAt)
	if err != nil {
		return classifyAssignmentError(err)
	}
	return nil
}

// classifyAssignmentError 把 driver_assignments 寫入失敗的原始 pg 錯誤碼轉換成呼叫端可辨識
// 的 sentinel：23503（外鍵違反，指派了不存在的司機/車輛）與 23P01（排除限制違反，期間重疊）。
// 其餘錯誤原樣回傳，交由上層當成系統錯誤處理。
func classifyAssignmentError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23503":
			return app.ErrAssignmentReferenceInvalid
		case "23P01":
			return app.ErrAssignmentOverlap
		}
	}
	return err
}

// ListDriversForVehicleOnDate 查詢某車輛在特定日期生效的所有司機，依司機姓名排序。
func (r *DriverRepository) ListDriversForVehicleOnDate(ctx context.Context, vehicleID uuid.UUID, serviceDate time.Time) ([]app.Driver, error) {
	if r.db == nil {
		return nil, fmt.Errorf("driver database is not configured")
	}
	query := `
		SELECT ` + driverColumnsWithAlias + `
		FROM driver_assignments a
		JOIN drivers d ON a.driver_id = d.id
		WHERE a.vehicle_id = $1
		  AND a.effective_range @> $2::date
		  AND d.deleted_at IS NULL
		ORDER BY d.name ASC
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
	if r.db == nil {
		return nil, fmt.Errorf("driver database is not configured")
	}
	if len(vehicleIDs) == 0 {
		return result, nil
	}
	query := `
		SELECT a.vehicle_id, ` + driverColumnsWithAlias + `
		FROM driver_assignments a
		JOIN drivers d ON a.driver_id = d.id
		WHERE a.vehicle_id = ANY($1::uuid[])
		  AND a.effective_range @> $2::date
		  AND d.deleted_at IS NULL
		ORDER BY d.name ASC
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
			return classifyAssignmentError(err)
		}
	}

	return tx.Commit(ctx)
}

// SoftDelete 軟刪除司機，回傳 false 代表該筆已被刪除過。
func (r *DriverRepository) SoftDelete(ctx context.Context, id, actorID uuid.UUID) (bool, error) {
	tag, err := pgxdb.FromContext(ctx, r.db).Exec(ctx, `UPDATE drivers SET deleted_at = now(), deleted_by = $2 WHERE id = $1 AND deleted_at IS NULL`, id, actorID)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() == 1, nil
}

// CloseActiveAssignments 收斂該司機所有生效中車輛指派的區間至今天。
// 今天（含）之後才起算的指派直接刪除：把它們收斂到今天會產生 lower > upper 的
// 非法區間，而且那段期間本來就還沒發生，沒有需要保留的歷史。
func (r *DriverRepository) CloseActiveAssignments(ctx context.Context, driverID uuid.UUID) error {
	db := pgxdb.FromContext(ctx, r.db)
	today := clock.Today()
	if _, err := db.Exec(ctx, `
		DELETE FROM driver_assignments
		WHERE driver_id = $1 AND lower(effective_range) >= $2::date
	`, driverID, today); err != nil {
		return err
	}
	_, err := db.Exec(ctx, `
		UPDATE driver_assignments
		SET effective_range = daterange(lower(effective_range), $2::date, '[)')
		WHERE driver_id = $1 AND upper_inf(effective_range)
	`, driverID, today)
	return err
}
