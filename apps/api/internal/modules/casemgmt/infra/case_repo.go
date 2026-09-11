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
	"ltc-system/apps/api/internal/domain/rocdate"
	"ltc-system/apps/api/internal/modules/casemgmt/app"
	"ltc-system/apps/api/internal/platform/clock"
	"ltc-system/apps/api/internal/platform/pgxdb"
)

// CaseRepository 提供 cases, case_schedules 與 schedule_legs 資料表之存取操作。
type CaseRepository struct {
	db *pgxpool.Pool
}

// NewCaseRepository 建立 CaseRepository 實例。
func NewCaseRepository(db *pgxpool.Pool) *CaseRepository {
	return &CaseRepository{db: db}
}

// List 取得個案清單（預設回傳遮罩身分證）。待維護的判定由 case_pending_status view 提供，
// 是全專案唯一一份定義；unresolvedLink 為 true 時只回傳待維護個案，excludePending 為 true 時
// 排除待維護個案，供主列表與「待維護」分頁互斥呈現。
func (r *CaseRepository) List(ctx context.Context, status, q, region string, page, pageSize int, unresolvedLink, excludePending bool) ([]app.Case, int64, error) {
	offset := (page - 1) * pageSize
	query := `
		SELECT c.id, c.name, c.name_normalized, c.national_id_cipher, c.national_id_hmac, c.national_id_masked, c.national_id_invalid,
		       c.household_type, c.gender, c.birth_date, c.birth_date_raw, c.care_contact_role, c.care_contact_name, c.registered_address,
		       c.site_id, COALESCE(st.name, ''), c.site_name_raw, c.caregiver_id, COALESCE(cg.name, ''), COALESCE(cg.type, ''),
		       c.home_address, c.ltc_level, c.service_category, c.service_usage_type, c.claim_end_date,
		       c.status, c.remarks, c.created_at, c.updated_at
		FROM cases c
		LEFT JOIN sites st ON st.id = c.site_id
		LEFT JOIN caregivers cg ON cg.id = c.caregiver_id
		JOIN case_pending_status ps ON ps.case_id = c.id
		WHERE c.deleted_at IS NULL
		  AND ($1 = '' OR c.status = $1)
		  AND ($2 = '' OR c.name ILIKE '%' || $2 || '%' OR c.home_address ILIKE '%' || $2 || '%')
		  AND ($5 = false OR ps.is_pending)
		  AND ($6 = false OR NOT ps.is_pending)
		  AND ($7 = '' OR COALESCE(st.region, '') ILIKE '%' || $7 || '%')
		ORDER BY c.created_at DESC, c.name ASC
		LIMIT $3 OFFSET $4
	`
	rows, err := r.db.Query(ctx, query, status, q, pageSize, offset, unresolvedLink, excludePending, region)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query cases: %w", err)
	}
	defer rows.Close()

	var list []app.Case
	for rows.Next() {
		var c app.Case
		if err := rows.Scan(
			&c.ID, &c.Name, &c.NameNormalized, &c.NationalIDCipher, &c.NationalIDHMAC, &c.NationalIDMasked, &c.NationalIDInvalid,
			&c.HouseholdType, &c.Gender, &c.BirthDate, &c.BirthDateRaw, &c.CareContactRole, &c.CareContactName, &c.RegisteredAddress,
			&c.SiteID, &c.SiteName, &c.SiteNameRaw, &c.CaregiverID, &c.CaregiverName, &c.CaregiverType,
			&c.HomeAddress, &c.LTCLevel, &c.ServiceCategory, &c.ServiceUsageType, &c.ClaimEndDate,
			&c.Status, &c.Remarks, &c.CreatedAt, &c.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}
		list = append(list, c)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("failed to iterate cases: %w", err)
	}

	var total int64
	countQuery := `
		SELECT COUNT(*) FROM cases c
		LEFT JOIN sites st ON st.id = c.site_id
		JOIN case_pending_status ps ON ps.case_id = c.id
		WHERE c.deleted_at IS NULL
		  AND ($1 = '' OR c.status = $1)
		  AND ($2 = '' OR c.name ILIKE '%' || $2 || '%' OR c.home_address ILIKE '%' || $2 || '%')
		  AND ($3 = false OR ps.is_pending)
		  AND ($4 = false OR NOT ps.is_pending)
		  AND ($5 = '' OR COALESCE(st.region, '') ILIKE '%' || $5 || '%')
	`
	if err := r.db.QueryRow(ctx, countQuery, status, q, unresolvedLink, excludePending, region).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count cases: %w", err)
	}

	return list, total, nil
}

// ListAll 取得完整的未刪除且非待維護個案資料集，供個案主檔匯出使用，不受 UI 分頁上限影響。
func (r *CaseRepository) ListAll(ctx context.Context) ([]app.Case, error) {
	query := `
		SELECT c.id, c.name, c.name_normalized, c.national_id_cipher, c.national_id_hmac, c.national_id_masked, c.national_id_invalid,
		       c.household_type, c.gender, c.birth_date, c.birth_date_raw, c.care_contact_role, c.care_contact_name, c.registered_address,
		       c.site_id, COALESCE(st.name, ''), c.site_name_raw, c.caregiver_id, COALESCE(cg.name, ''), COALESCE(cg.type, ''),
		       c.home_address, c.ltc_level, c.service_category, c.service_usage_type, c.claim_end_date,
		       c.status, c.remarks, c.created_at, c.updated_at
		FROM cases c
		LEFT JOIN sites st ON st.id = c.site_id
		LEFT JOIN caregivers cg ON cg.id = c.caregiver_id
		JOIN case_pending_status ps ON ps.case_id = c.id
		WHERE c.deleted_at IS NULL
		  AND NOT ps.is_pending
		ORDER BY c.created_at DESC, c.name ASC
	`
	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query all cases: %w", err)
	}
	defer rows.Close()

	list := make([]app.Case, 0)
	for rows.Next() {
		var c app.Case
		if err := rows.Scan(
			&c.ID, &c.Name, &c.NameNormalized, &c.NationalIDCipher, &c.NationalIDHMAC, &c.NationalIDMasked, &c.NationalIDInvalid,
			&c.HouseholdType, &c.Gender, &c.BirthDate, &c.BirthDateRaw, &c.CareContactRole, &c.CareContactName, &c.RegisteredAddress,
			&c.SiteID, &c.SiteName, &c.SiteNameRaw, &c.CaregiverID, &c.CaregiverName, &c.CaregiverType,
			&c.HomeAddress, &c.LTCLevel, &c.ServiceCategory, &c.ServiceUsageType, &c.ClaimEndDate,
			&c.Status, &c.Remarks, &c.CreatedAt, &c.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan all cases: %w", err)
		}
		list = append(list, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate all cases: %w", err)
	}
	return list, nil
}

// ClaimCaseImportRow 以檔案雜湊與來源列鍵保護個案匯入重試；呼叫端應在列交易內執行。
func (r *CaseRepository) ClaimCaseImportRow(ctx context.Context, fileHash, rowKey string, caseID uuid.UUID) (bool, error) {
	db := pgxdb.FromContext(ctx, r.db)
	var claimedID uuid.UUID
	err := db.QueryRow(ctx, `
		INSERT INTO case_import_idempotency (file_hash, row_key, case_id)
		VALUES ($1, $2, $3)
		ON CONFLICT (file_hash, row_key) DO NOTHING
		RETURNING id
	`, fileHash, rowKey, caseID).Scan(&claimedID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return false, nil
		}
		return false, err
	}
	return claimedID != uuid.Nil, nil
}

// IsCaseImportRowCommitted 查詢某份檔案的某一列是否已建立過個案。
func (r *CaseRepository) IsCaseImportRowCommitted(ctx context.Context, fileHash, rowKey string) (bool, error) {
	db := pgxdb.FromContext(ctx, r.db)
	var exists bool
	err := db.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM case_import_idempotency WHERE file_hash = $1 AND row_key = $2)
	`, fileHash, rowKey).Scan(&exists)
	if err != nil {
		return false, err
	}
	return exists, nil
}

// GetByID 依 UUID 取得個案。
func (r *CaseRepository) GetByID(ctx context.Context, id uuid.UUID) (*app.Case, error) {
	query := `
		SELECT c.id, c.name, c.name_normalized, c.national_id_cipher, c.national_id_hmac, c.national_id_masked, c.national_id_invalid,
		       c.household_type, c.gender, c.birth_date, c.birth_date_raw, c.care_contact_role, c.care_contact_name, c.registered_address,
		       c.site_id, COALESCE(st.name, ''), c.site_name_raw, c.caregiver_id, COALESCE(cg.name, ''), COALESCE(cg.type, ''),
		       c.home_address, c.ltc_level, c.service_category, c.service_usage_type, c.claim_end_date,
		       c.status, c.remarks, c.created_at, c.updated_at
		FROM cases c
		LEFT JOIN sites st ON st.id = c.site_id
		LEFT JOIN caregivers cg ON cg.id = c.caregiver_id
		WHERE c.id = $1 AND c.deleted_at IS NULL
	`
	var c app.Case
	db := pgxdb.FromContext(ctx, r.db)
	err := db.QueryRow(ctx, query, id).Scan(
		&c.ID, &c.Name, &c.NameNormalized, &c.NationalIDCipher, &c.NationalIDHMAC, &c.NationalIDMasked, &c.NationalIDInvalid,
		&c.HouseholdType, &c.Gender, &c.BirthDate, &c.BirthDateRaw, &c.CareContactRole, &c.CareContactName, &c.RegisteredAddress,
		&c.SiteID, &c.SiteName, &c.SiteNameRaw, &c.CaregiverID, &c.CaregiverName, &c.CaregiverType,
		&c.HomeAddress, &c.LTCLevel, &c.ServiceCategory, &c.ServiceUsageType, &c.ClaimEndDate,
		&c.Status, &c.Remarks, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, app.ErrCaseNotFound
		}
		return nil, err
	}
	return &c, nil
}

// GetByHMAC 依 HMAC 索引檢查是否已存在。
func (r *CaseRepository) GetByHMAC(ctx context.Context, hmac []byte) (*app.Case, error) {
	query := `
		SELECT id, name, name_normalized, national_id_cipher, national_id_hmac, national_id_masked,
		       home_address, ltc_level, service_category, service_usage_type, claim_end_date,
		       status, created_at, updated_at
		FROM cases WHERE national_id_hmac = $1 AND deleted_at IS NULL LIMIT 1
	`
	var c app.Case
	db := pgxdb.FromContext(ctx, r.db)
	err := db.QueryRow(ctx, query, hmac).Scan(
		&c.ID, &c.Name, &c.NameNormalized, &c.NationalIDCipher, &c.NationalIDHMAC, &c.NationalIDMasked,
		&c.HomeAddress, &c.LTCLevel, &c.ServiceCategory, &c.ServiceUsageType, &c.ClaimEndDate,
		&c.Status, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, app.ErrCaseNotFound
		}
		return nil, err
	}
	return &c, nil
}

// GetByNameNormalized 依正規化姓名搜尋個案。
func (r *CaseRepository) GetByNameNormalized(ctx context.Context, nameNorm string) ([]app.Case, error) {
	query := `
		SELECT id, name, name_normalized, national_id_cipher, national_id_hmac, national_id_masked,
		       home_address, ltc_level, service_category, service_usage_type, claim_end_date,
		       status, created_at, updated_at
		FROM cases WHERE name_normalized = $1 AND deleted_at IS NULL
	`
	rows, err := r.db.Query(ctx, query, nameNorm)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []app.Case
	for rows.Next() {
		var c app.Case
		if err := rows.Scan(
			&c.ID, &c.Name, &c.NameNormalized, &c.NationalIDCipher, &c.NationalIDHMAC, &c.NationalIDMasked,
			&c.HomeAddress, &c.LTCLevel, &c.ServiceCategory, &c.ServiceUsageType, &c.ClaimEndDate,
			&c.Status, &c.CreatedAt, &c.UpdatedAt,
		); err != nil {
			return nil, err
		}
		list = append(list, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate case name matches: %w", err)
	}
	return list, nil
}

// Create 新增個案主檔。
func (r *CaseRepository) Create(ctx context.Context, c *app.Case) error {
	query := `
		INSERT INTO cases (
			id, name, name_normalized, national_id_cipher, national_id_hmac, national_id_masked, national_id_invalid,
			household_type, gender, birth_date, birth_date_raw, care_contact_role, care_contact_name, registered_address,
			home_address, ltc_level, service_category, service_usage_type, claim_end_date, status, remarks,
			site_id, site_name_raw, caregiver_id
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24)
		RETURNING created_at, updated_at
	`
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	db := pgxdb.FromContext(ctx, r.db)
	err := db.QueryRow(ctx, query,
		c.ID, c.Name, c.NameNormalized, c.NationalIDCipher, c.NationalIDHMAC, c.NationalIDMasked, c.NationalIDInvalid,
		c.HouseholdType, c.Gender, c.BirthDate, c.BirthDateRaw, c.CareContactRole, c.CareContactName, c.RegisteredAddress,
		c.HomeAddress, c.LTCLevel, c.ServiceCategory, c.ServiceUsageType, c.ClaimEndDate, c.Status, c.Remarks,
		c.SiteID, c.SiteNameRaw, c.CaregiverID,
	).Scan(&c.CreatedAt, &c.UpdatedAt)
	return handleCaseDBError(err)
}

// Update 修改個案資料；走 pgxdb.FromContext 以支援外層交易（裁決疑似重複個案的
// merged_existing 分支需要在同一交易內呼叫）。
func (r *CaseRepository) Update(ctx context.Context, c *app.Case) error {
	query := `
		UPDATE cases
		SET name = $2, name_normalized = $3, home_address = $4, ltc_level = $5,
		    service_category = $6, service_usage_type = $7, claim_end_date = $8,
		    status = $9, household_type = $10, gender = $11, birth_date = $12, birth_date_raw = $13,
		    care_contact_role = $14, care_contact_name = $15, registered_address = $16, remarks = $17,
		    national_id_cipher = $18, national_id_hmac = $19, national_id_masked = $20, national_id_invalid = $21,
		    site_id = $22, site_name_raw = $23, caregiver_id = $24,
		    updated_at = now()
		WHERE id = $1 AND deleted_at IS NULL
		RETURNING updated_at
	`
	db := pgxdb.FromContext(ctx, r.db)
	err := db.QueryRow(ctx, query,
		c.ID, c.Name, c.NameNormalized, c.HomeAddress, c.LTCLevel,
		c.ServiceCategory, c.ServiceUsageType, c.ClaimEndDate, c.Status,
		c.HouseholdType, c.Gender, c.BirthDate, c.BirthDateRaw, c.CareContactRole, c.CareContactName, c.RegisteredAddress, c.Remarks,
		c.NationalIDCipher, c.NationalIDHMAC, c.NationalIDMasked, c.NationalIDInvalid,
		c.SiteID, c.SiteNameRaw, c.CaregiverID,
	).Scan(&c.UpdatedAt)
	return handleCaseDBError(err)
}

// handleCaseDBError 將 national_id_hmac 唯一索引衝突轉譯為可辨識的 domain error，
// 供 handler 回應 409 而非未分類的 500。
func handleCaseDBError(err error) error {
	if err == nil {
		return nil
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" && strings.Contains(pgErr.ConstraintName, "national_id_hmac") {
		return app.ErrDuplicateNationalID
	}
	return err
}

// handleScheduleDBError 將排班寫入時的 pg 錯誤轉譯為可辨識的 domain error：
// 生效期間重疊（no_overlapping_case_schedule exclusion constraint）轉為 ErrScheduleOverlap，
// 外鍵失效（選到不存在的個案或車輛）轉為 ErrScheduleInvalidReference，
// 讓 transport 層只需比對 domain sentinel，不必直接依賴 pgconn。
func handleScheduleDBError(err error) error {
	if err == nil {
		return nil
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		if pgErr.Code == "23P01" && pgErr.ConstraintName == "no_overlapping_case_schedule" {
			return app.ErrScheduleOverlap
		}
		if pgErr.Code == "23503" {
			return app.ErrScheduleInvalidReference
		}
	}
	return err
}

// CreateSchedule 建立排班設定與對應的 legs（包在同一個事務中）。
// 若 ctx 已掛載外層事務（見 pgxdb.TxRunner），排班與 legs 寫入會併入該事務，
// 由外層決定 commit／rollback；否則自行開啟並管理事務。
func (r *CaseRepository) CreateSchedule(ctx context.Context, s *app.CaseSchedule) error {
	if tx, ok := pgxdb.TxFromContext(ctx); ok {
		return handleScheduleDBError(r.insertSchedule(ctx, tx, s))
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if err := r.insertSchedule(ctx, tx, s); err != nil {
		return handleScheduleDBError(err)
	}
	return tx.Commit(ctx)
}

func (r *CaseRepository) insertSchedule(ctx context.Context, tx pgxdb.Querier, s *app.CaseSchedule) error {
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}

	querySchedule := `
		INSERT INTO case_schedules (
			id, case_id, effective_range, weekdays, trip_pattern, unit_price, distance_km,
			service_duration_min, service_code, note
		) VALUES ($1, $2, daterange($3, $4, '[)'), $5, $6, $7, $8, $9, $10, $11)
		RETURNING created_at, updated_at
	`
	var toVal *time.Time
	if s.EffectiveTo != nil {
		exclusiveEnd := s.EffectiveTo.AddDate(0, 0, 1)
		toVal = &exclusiveEnd
	}
	err := tx.QueryRow(ctx, querySchedule,
		s.ID, s.CaseID, s.EffectiveFrom, toVal, s.Weekdays, s.TripPattern,
		s.UnitPrice, s.DistanceKM, s.ServiceDurationMin, s.ServiceCode, s.Note,
	).Scan(&s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to insert case_schedule: %w", err)
	}

	queryLeg := `
		INSERT INTO schedule_legs (
			id, schedule_id, leg_seq, direction, period, depart_time, arrive_time, run_no, vehicle_id
		) VALUES ($1, $2, $3, $4, $5, $6::time, $7::time, $8, $9)
		RETURNING created_at
	`
	for i := range s.Legs {
		leg := &s.Legs[i]
		if leg.ID == uuid.Nil {
			leg.ID = uuid.New()
		}
		leg.ScheduleID = s.ID
		var arriveVal *string
		if leg.ArriveTime != nil {
			arriveVal = leg.ArriveTime
		}
		err = tx.QueryRow(ctx, queryLeg,
			leg.ID, leg.ScheduleID, leg.LegSeq, leg.Direction, leg.Period,
			leg.DepartTime, arriveVal, leg.RunNo, leg.VehicleID,
		).Scan(&leg.CreatedAt)
		if err != nil {
			return fmt.Errorf("failed to insert schedule_leg %d: %w", leg.LegSeq, err)
		}
	}

	return nil
}

// GetActiveScheduleForCaseOnDate 查詢個案在指定日期的有效排班與時段細節。
func (r *CaseRepository) GetActiveScheduleForCaseOnDate(ctx context.Context, caseID uuid.UUID, serviceDate time.Time) (*app.CaseSchedule, error) {
	query := `
		SELECT s.id, s.case_id, s.weekdays, s.trip_pattern,
		       s.unit_price, s.distance_km, s.service_duration_min, s.service_code, s.note,
		       s.created_at, s.updated_at
		FROM case_schedules s
		WHERE s.case_id = $1
		  AND s.effective_range @> $2::date
		LIMIT 1
	`
	var s app.CaseSchedule
	err := r.db.QueryRow(ctx, query, caseID, serviceDate).Scan(
		&s.ID, &s.CaseID, &s.Weekdays, &s.TripPattern,
		&s.UnitPrice, &s.DistanceKM, &s.ServiceDurationMin, &s.ServiceCode, &s.Note,
		&s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	// 查詢 Legs
	legQuery := `
		SELECT l.id, l.schedule_id, l.leg_seq, l.direction, l.period,
		       to_char(l.depart_time, 'HH24:MI') as depart_time,
		       to_char(l.arrive_time, 'HH24:MI') as arrive_time,
		       l.run_no, l.vehicle_id, v.display_name as vehicle_name, l.created_at
		FROM schedule_legs l
		LEFT JOIN vehicles v ON l.vehicle_id = v.id AND v.deleted_at IS NULL
		WHERE l.schedule_id = $1
		ORDER BY l.leg_seq ASC
	`
	rows, err := r.db.Query(ctx, legQuery, s.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var leg app.ScheduleLeg
		if err := rows.Scan(
			&leg.ID, &leg.ScheduleID, &leg.LegSeq, &leg.Direction, &leg.Period,
			&leg.DepartTime, &leg.ArriveTime, &leg.RunNo, &leg.VehicleID, &leg.VehicleName, &leg.CreatedAt,
		); err != nil {
			return nil, err
		}
		s.Legs = append(s.Legs, leg)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate schedule legs: %w", err)
	}

	return &s, nil
}

// GetActiveSchedulesForMonth 查詢特定月份所有在案個案的有效排班與 Legs。
func (r *CaseRepository) GetActiveSchedulesForMonth(ctx context.Context, year, month int) ([]app.ActiveCaseScheduleInfo, error) {
	if r.db == nil {
		return nil, fmt.Errorf("case database is not configured")
	}

	firstDay := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	lastDay := firstDay.AddDate(0, 1, 0)

	query := `
		SELECT c.id, c.name, c.claim_end_date,
		       s.id as schedule_id,
		       lower(s.effective_range) as eff_from,
		       CASE WHEN upper_inf(s.effective_range) THEN NULL ELSE to_char(upper(s.effective_range) - 1, 'YYYY-MM-DD') END as eff_to_str,
		       s.weekdays, s.trip_pattern
		FROM cases c
		JOIN case_schedules s ON c.id = s.case_id
		JOIN case_pending_status ps ON ps.case_id = c.id
		WHERE c.status = 'active' AND c.deleted_at IS NULL AND NOT ps.is_pending
		  AND s.effective_range && daterange($1, $2, '[)')
		ORDER BY c.created_at DESC, c.name ASC
	`
	rows, err := r.db.Query(ctx, query, firstDay, lastDay)
	if err != nil {
		return nil, fmt.Errorf("failed to query monthly active schedules: %w", err)
	}
	defer rows.Close()

	type scheduleRow struct {
		info       app.ActiveCaseScheduleInfo
		scheduleID uuid.UUID
	}
	var list []scheduleRow

	for rows.Next() {
		var sr scheduleRow
		var effToStr *string
		if err := rows.Scan(
			&sr.info.CaseID, &sr.info.CaseName,
			&sr.info.ClaimEndDate,
			&sr.scheduleID,
			&sr.info.EffectiveFrom, &effToStr,
			&sr.info.Weekdays, &sr.info.TripPattern,
		); err != nil {
			return nil, err
		}
		if effToStr != nil && *effToStr != "" {
			t, err := rocdate.ParseDate(*effToStr)
			if err != nil {
				return nil, fmt.Errorf("parse schedule effective end date: %w", err)
			}
			sr.info.EffectiveTo = &t
		}
		list = append(list, sr)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate monthly active schedules: %w", err)
	}

	var results []app.ActiveCaseScheduleInfo
	for _, item := range list {
		legQuery := `
			SELECT id, schedule_id, leg_seq, direction, period,
			       to_char(depart_time, 'HH24:MI') as depart_time,
			       to_char(arrive_time, 'HH24:MI') as arrive_time,
			       run_no, vehicle_id, created_at
			FROM schedule_legs
			WHERE schedule_id = $1
			ORDER BY leg_seq ASC
		`
		lRows, err := r.db.Query(ctx, legQuery, item.scheduleID)
		if err != nil {
			return nil, err
		}
		for lRows.Next() {
			var leg app.ScheduleLeg
			if err := lRows.Scan(
				&leg.ID, &leg.ScheduleID, &leg.LegSeq, &leg.Direction, &leg.Period,
				&leg.DepartTime, &leg.ArriveTime, &leg.RunNo, &leg.VehicleID, &leg.CreatedAt,
			); err != nil {
				lRows.Close()
				return nil, err
			}
			item.info.Legs = append(item.info.Legs, leg)
		}
		if err := lRows.Err(); err != nil {
			lRows.Close()
			return nil, fmt.Errorf("failed to iterate monthly schedule legs: %w", err)
		}
		lRows.Close()
		results = append(results, item.info)
	}

	return results, nil
}

// ListNameIndex 取得在案個案的姓名索引。只取比對姓名需要的欄位，避免為了推薦
// 匯報欄位對應而把整份含密文的個案主檔載進記憶體。
func (r *CaseRepository) ListNameIndex(ctx context.Context) ([]app.CaseNameRef, error) {
	if r.db == nil {
		return nil, fmt.Errorf("case database is not configured")
	}

	rows, err := r.db.Query(ctx, `
		SELECT c.id, c.name, c.name_normalized
		FROM cases c
		JOIN case_pending_status ps ON ps.case_id = c.id
		WHERE c.status = 'active' AND c.deleted_at IS NULL AND NOT ps.is_pending
		ORDER BY c.name ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to query case name index: %w", err)
	}
	defer rows.Close()

	var list []app.CaseNameRef
	for rows.Next() {
		var c app.CaseNameRef
		if err := rows.Scan(&c.ID, &c.Name, &c.NameNormalized); err != nil {
			return nil, err
		}
		list = append(list, c)
	}
	return list, rows.Err()
}

// SoftDelete 軟刪除個案，回傳 false 代表該筆已被刪除過。
func (r *CaseRepository) SoftDelete(ctx context.Context, id, actorID uuid.UUID) (bool, error) {
	db := pgxdb.FromContext(ctx, r.db)
	tag, err := db.Exec(ctx, `UPDATE cases SET deleted_at = now(), deleted_by = $2 WHERE id = $1 AND deleted_at IS NULL`, id, actorID)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() == 1, nil
}

// sitePendingPlaceholder 是據點欄位完全空白時的哨兵值，與 caseimport.sitePendingPlaceholder
// 及 migration 000044 一致；完全空白視為「待補齊」而非可比對的名稱，不可自動關聯。
const sitePendingPlaceholder = "（待補齊據點）"

// RelinkSiteByName 依名稱重新比對待維護個案的據點：只有這個名稱在 sites 恰好命中一筆才
// 寫入，跨區域同名（uq_site_name_region 只保證同區域內唯一）或查無資料一律不動。
func (r *CaseRepository) RelinkSiteByName(ctx context.Context, name string) ([]uuid.UUID, error) {
	if name == "" || name == sitePendingPlaceholder {
		return nil, nil
	}
	db := pgxdb.FromContext(ctx, r.db)
	rows, err := db.Query(ctx, `
		UPDATE cases c
		SET site_id = s.id, site_name_raw = NULL, updated_at = now()
		FROM (SELECT id FROM sites WHERE name = $1 LIMIT 1) s
		WHERE c.deleted_at IS NULL
		  AND c.site_id IS NULL
		  AND c.site_name_raw = $1
		  AND (SELECT count(*) FROM sites WHERE name = $1) = 1
		RETURNING c.id
	`, name)
	if err != nil {
		return nil, fmt.Errorf("failed to relink cases by site name: %w", err)
	}
	defer rows.Close()
	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// RelinkCaregiverByName 依名稱重新比對待維護個案的照護人員，比對規則與匯入時的
// resolveCaregiver（見 caseimport/app/commit.go）一致。
func (r *CaseRepository) RelinkCaregiverByName(ctx context.Context, name string) ([]uuid.UUID, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, nil
	}
	db := pgxdb.FromContext(ctx, r.db)
	var ids []uuid.UUID

	// 同名唯一即採用，不看角色。
	rows1, err := db.Query(ctx, `
		WITH grouped AS (
			SELECT btrim(name) AS name, (array_agg(id))[1] AS id, count(*) AS cnt
			FROM caregivers
			WHERE btrim(name) = $1
			GROUP BY btrim(name)
		)
		UPDATE cases c
		SET caregiver_id = m.id, updated_at = now()
		FROM grouped m
		WHERE c.deleted_at IS NULL
		  AND c.caregiver_id IS NULL
		  AND btrim(c.care_contact_name) = $1
		  AND m.cnt = 1
		RETURNING c.id
	`, name)
	if err != nil {
		return nil, fmt.Errorf("failed to relink cases by caregiver name (unique): %w", err)
	}
	for rows1.Next() {
		var id uuid.UUID
		if err := rows1.Scan(&id); err != nil {
			rows1.Close()
			return nil, err
		}
		ids = append(ids, id)
	}
	if err := rows1.Err(); err != nil {
		rows1.Close()
		return nil, err
	}
	rows1.Close()

	// 同名多筆時才用 care_contact_role 消歧，過濾後仍唯一才採用。
	rows2, err := db.Query(ctx, `
		WITH grouped AS (
			SELECT btrim(name) AS name, type, (array_agg(id))[1] AS id, count(*) AS cnt
			FROM caregivers
			WHERE btrim(name) = $1
			GROUP BY btrim(name), type
		)
		UPDATE cases c
		SET caregiver_id = m.id, updated_at = now()
		FROM grouped m
		WHERE c.deleted_at IS NULL
		  AND c.caregiver_id IS NULL
		  AND btrim(c.care_contact_name) = $1
		  AND m.cnt = 1
		  AND m.type = CASE btrim(c.care_contact_role)
		                   WHEN '個管' THEN 'case_manager'
		                   WHEN '照專' THEN 'specialist'
		                   WHEN '專護' THEN 'specialist'
		                   ELSE NULL
		               END
		RETURNING c.id
	`, name)
	if err != nil {
		return nil, fmt.Errorf("failed to relink cases by caregiver name (role-disambiguated): %w", err)
	}
	defer rows2.Close()
	for rows2.Next() {
		var id uuid.UUID
		if err := rows2.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows2.Err()
}

// ListPendingSiteNames 列出目前待維護個案中相異的據點原始名稱。
func (r *CaseRepository) ListPendingSiteNames(ctx context.Context) ([]string, error) {
	rows, err := r.db.Query(ctx, `SELECT DISTINCT site_name_raw FROM cases WHERE site_pending AND deleted_at IS NULL`)
	if err != nil {
		return nil, fmt.Errorf("failed to list pending site names: %w", err)
	}
	defer rows.Close()
	var names []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		names = append(names, name)
	}
	return names, rows.Err()
}

// ListPendingCaregiverNames 列出目前待維護個案中相異的照護人員原始姓名。
func (r *CaseRepository) ListPendingCaregiverNames(ctx context.Context) ([]string, error) {
	rows, err := r.db.Query(ctx, `SELECT DISTINCT btrim(care_contact_name) FROM cases WHERE caregiver_pending AND deleted_at IS NULL`)
	if err != nil {
		return nil, fmt.Errorf("failed to list pending caregiver names: %w", err)
	}
	defer rows.Close()
	var names []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		names = append(names, name)
	}
	return names, rows.Err()
}

// CloseOpenSchedules 收斂該個案所有生效中排班的區間至今天，供刪除個案時使用。
func (r *CaseRepository) CloseOpenSchedules(ctx context.Context, caseID uuid.UUID) error {
	db := pgxdb.FromContext(ctx, r.db)
	_, err := db.Exec(ctx, `
		UPDATE case_schedules
		SET effective_range = daterange(lower(effective_range), $2::date, '[)'), updated_at = now()
		WHERE case_id = $1 AND upper_inf(effective_range)
	`, caseID, clock.Today())
	return err
}
