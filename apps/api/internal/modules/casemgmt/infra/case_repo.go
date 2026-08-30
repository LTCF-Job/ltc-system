package infra

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"ltc-system/apps/api/internal/modules/casemgmt/app"
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

// List 取得個案清單（預設回傳遮罩身分證）。unresolvedLink 為 true 時僅回傳據點／去回程車輛
// 任一比對不到主檔（raw name 有值但對應 ID 為 null）的個案。
func (r *CaseRepository) List(ctx context.Context, region, status, q string, page, pageSize int, unresolvedLink bool) ([]app.Case, int64, error) {
	offset := (page - 1) * pageSize
	query := `
		SELECT c.id, c.code, c.name, c.name_normalized, c.national_id_cipher, c.national_id_hmac, c.national_id_masked,
		       c.household_type, c.gender, c.birth_date, c.care_contact_role, c.care_contact_name, c.registered_address,
		       p.site_id, COALESCE(st.name, ''), p.outbound_vehicle_id, COALESCE(vo.display_name, ''), p.inbound_vehicle_id, COALESCE(vi.display_name, ''),
		       c.home_address, c.region, c.ltc_level, c.service_category, c.service_usage_type, c.claim_start_date, c.claim_end_date,
		       c.status, c.remarks, c.created_at, c.updated_at
		FROM cases c
		LEFT JOIN case_transport_preferences p ON p.case_id = c.id
		LEFT JOIN sites st ON st.id = p.site_id
		LEFT JOIN vehicles vo ON vo.id = p.outbound_vehicle_id
		LEFT JOIN vehicles vi ON vi.id = p.inbound_vehicle_id
		WHERE ($1 = '' OR c.region = $1)
		  AND ($2 = '' OR c.status = $2)
		  AND ($3 = '' OR c.name ILIKE '%' || $3 || '%' OR c.code ILIKE '%' || $3 || '%' OR c.home_address ILIKE '%' || $3 || '%')
		  AND ($6 = false OR (
		        (p.site_id IS NULL AND p.site_name_raw IS NOT NULL) OR
		        (p.outbound_vehicle_id IS NULL AND p.outbound_vehicle_name_raw IS NOT NULL) OR
		        (p.inbound_vehicle_id IS NULL AND p.inbound_vehicle_name_raw IS NOT NULL)
		      ))
		ORDER BY c.code ASC
		LIMIT $4 OFFSET $5
	`
	rows, err := r.db.Query(ctx, query, region, status, q, pageSize, offset, unresolvedLink)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query cases: %w", err)
	}
	defer rows.Close()

	var list []app.Case
	for rows.Next() {
		var c app.Case
		if err := rows.Scan(
			&c.ID, &c.Code, &c.Name, &c.NameNormalized, &c.NationalIDCipher, &c.NationalIDHMAC, &c.NationalIDMasked,
			&c.HouseholdType, &c.Gender, &c.BirthDate, &c.CareContactRole, &c.CareContactName, &c.RegisteredAddress,
			&c.SiteID, &c.SiteName, &c.OutboundVehicleID, &c.OutboundVehicle, &c.InboundVehicleID, &c.InboundVehicle,
			&c.HomeAddress, &c.Region, &c.LTCLevel, &c.ServiceCategory, &c.ServiceUsageType, &c.ClaimStartDate, &c.ClaimEndDate,
			&c.Status, &c.Remarks, &c.CreatedAt, &c.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}
		list = append(list, c)
	}

	var total int64
	countQuery := `
		SELECT COUNT(*) FROM cases c
		LEFT JOIN case_transport_preferences p ON p.case_id = c.id
		WHERE ($1 = '' OR c.region = $1)
		  AND ($2 = '' OR c.status = $2)
		  AND ($3 = '' OR c.name ILIKE '%' || $3 || '%' OR c.code ILIKE '%' || $3 || '%' OR c.home_address ILIKE '%' || $3 || '%')
		  AND ($4 = false OR (
		        (p.site_id IS NULL AND p.site_name_raw IS NOT NULL) OR
		        (p.outbound_vehicle_id IS NULL AND p.outbound_vehicle_name_raw IS NOT NULL) OR
		        (p.inbound_vehicle_id IS NULL AND p.inbound_vehicle_name_raw IS NOT NULL)
		      ))
	`
	_ = r.db.QueryRow(ctx, countQuery, region, status, q, unresolvedLink).Scan(&total)

	return list, total, nil
}

// UpsertTransportPreference 寫入個案的據點與去回程車輛偏好。nil 的 ID 以 COALESCE 保留
// 既有值（避免部分更新把未提供的欄位覆寫為 null）；raw name 隨對應 ID 一併寫入或清空。
func (r *CaseRepository) UpsertTransportPreference(ctx context.Context, caseID uuid.UUID, siteID, outboundVehicleID, inboundVehicleID *uuid.UUID, siteNameRaw, outboundVehicleNameRaw, inboundVehicleNameRaw string) error {
	db := pgxdb.FromContext(ctx, r.db)
	_, err := db.Exec(ctx, `INSERT INTO case_transport_preferences (case_id, site_id, outbound_vehicle_id, inbound_vehicle_id, site_name_raw, outbound_vehicle_name_raw, inbound_vehicle_name_raw)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (case_id) DO UPDATE SET
			site_id = COALESCE(EXCLUDED.site_id, case_transport_preferences.site_id),
			outbound_vehicle_id = COALESCE(EXCLUDED.outbound_vehicle_id, case_transport_preferences.outbound_vehicle_id),
			inbound_vehicle_id = COALESCE(EXCLUDED.inbound_vehicle_id, case_transport_preferences.inbound_vehicle_id),
			site_name_raw = CASE WHEN EXCLUDED.site_id IS NOT NULL THEN NULL WHEN EXCLUDED.site_name_raw IS NOT NULL THEN EXCLUDED.site_name_raw ELSE case_transport_preferences.site_name_raw END,
			outbound_vehicle_name_raw = CASE WHEN EXCLUDED.outbound_vehicle_id IS NOT NULL THEN NULL WHEN EXCLUDED.outbound_vehicle_name_raw IS NOT NULL THEN EXCLUDED.outbound_vehicle_name_raw ELSE case_transport_preferences.outbound_vehicle_name_raw END,
			inbound_vehicle_name_raw = CASE WHEN EXCLUDED.inbound_vehicle_id IS NOT NULL THEN NULL WHEN EXCLUDED.inbound_vehicle_name_raw IS NOT NULL THEN EXCLUDED.inbound_vehicle_name_raw ELSE case_transport_preferences.inbound_vehicle_name_raw END,
			updated_at = now()`,
		caseID, siteID, outboundVehicleID, inboundVehicleID,
		nullIfEmpty(siteNameRaw), nullIfEmpty(outboundVehicleNameRaw), nullIfEmpty(inboundVehicleNameRaw))
	return err
}

// nullIfEmpty 將空字串轉為 nil，讓 SQL 端可用 IS NOT NULL 判斷是否有提供 raw name。
func nullIfEmpty(v string) *string {
	if v == "" {
		return nil
	}
	return &v
}

// GetByID 依 UUID 取得個案。
func (r *CaseRepository) GetByID(ctx context.Context, id uuid.UUID) (*app.Case, error) {
	query := `
		SELECT c.id, c.code, c.name, c.name_normalized, c.national_id_cipher, c.national_id_hmac, c.national_id_masked,
		       c.household_type, c.gender, c.birth_date, c.care_contact_role, c.care_contact_name, c.registered_address,
		       p.site_id, COALESCE(st.name, ''), p.outbound_vehicle_id, COALESCE(vo.display_name, ''), p.inbound_vehicle_id, COALESCE(vi.display_name, ''),
		       c.home_address, c.region, c.ltc_level, c.service_category, c.service_usage_type, c.claim_start_date, c.claim_end_date,
		       c.status, c.remarks, c.created_at, c.updated_at
		FROM cases c
		LEFT JOIN case_transport_preferences p ON p.case_id = c.id
		LEFT JOIN sites st ON st.id = p.site_id
		LEFT JOIN vehicles vo ON vo.id = p.outbound_vehicle_id
		LEFT JOIN vehicles vi ON vi.id = p.inbound_vehicle_id
		WHERE c.id = $1
	`
	var c app.Case
	db := pgxdb.FromContext(ctx, r.db)
	err := db.QueryRow(ctx, query, id).Scan(
		&c.ID, &c.Code, &c.Name, &c.NameNormalized, &c.NationalIDCipher, &c.NationalIDHMAC, &c.NationalIDMasked,
		&c.HouseholdType, &c.Gender, &c.BirthDate, &c.CareContactRole, &c.CareContactName, &c.RegisteredAddress,
		&c.SiteID, &c.SiteName, &c.OutboundVehicleID, &c.OutboundVehicle, &c.InboundVehicleID, &c.InboundVehicle,
		&c.HomeAddress, &c.Region, &c.LTCLevel, &c.ServiceCategory, &c.ServiceUsageType, &c.ClaimStartDate, &c.ClaimEndDate,
		&c.Status, &c.Remarks, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

// GetByHMAC 依 HMAC 索引檢查是否已存在。
func (r *CaseRepository) GetByHMAC(ctx context.Context, hmac []byte) (*app.Case, error) {
	query := `
		SELECT id, code, name, name_normalized, national_id_cipher, national_id_hmac, national_id_masked,
		       home_address, region, ltc_level, service_category, service_usage_type, claim_start_date, claim_end_date,
		       status, created_at, updated_at
		FROM cases WHERE national_id_hmac = $1 LIMIT 1
	`
	var c app.Case
	db := pgxdb.FromContext(ctx, r.db)
	err := db.QueryRow(ctx, query, hmac).Scan(
		&c.ID, &c.Code, &c.Name, &c.NameNormalized, &c.NationalIDCipher, &c.NationalIDHMAC, &c.NationalIDMasked,
		&c.HomeAddress, &c.Region, &c.LTCLevel, &c.ServiceCategory, &c.ServiceUsageType, &c.ClaimStartDate, &c.ClaimEndDate,
		&c.Status, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

// GetByNameNormalized 依正規化姓名搜尋個案。
func (r *CaseRepository) GetByNameNormalized(ctx context.Context, nameNorm string) ([]app.Case, error) {
	query := `
		SELECT id, code, name, name_normalized, national_id_cipher, national_id_hmac, national_id_masked,
		       home_address, region, ltc_level, service_category, service_usage_type, claim_start_date, claim_end_date,
		       status, created_at, updated_at
		FROM cases WHERE name_normalized = $1
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
			&c.ID, &c.Code, &c.Name, &c.NameNormalized, &c.NationalIDCipher, &c.NationalIDHMAC, &c.NationalIDMasked,
			&c.HomeAddress, &c.Region, &c.LTCLevel, &c.ServiceCategory, &c.ServiceUsageType, &c.ClaimStartDate, &c.ClaimEndDate,
			&c.Status, &c.CreatedAt, &c.UpdatedAt,
		); err != nil {
			return nil, err
		}
		list = append(list, c)
	}
	return list, nil
}

// Create 新增個案主檔。
func (r *CaseRepository) Create(ctx context.Context, c *app.Case) error {
	query := `
		INSERT INTO cases (
			id, code, name, name_normalized, national_id_cipher, national_id_hmac, national_id_masked,
			household_type, gender, birth_date, care_contact_role, care_contact_name, registered_address,
			home_address, region, ltc_level, service_category, service_usage_type, claim_start_date, claim_end_date, status, remarks
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22)
		RETURNING created_at, updated_at
	`
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	db := pgxdb.FromContext(ctx, r.db)
	return db.QueryRow(ctx, query,
		c.ID, c.Code, c.Name, c.NameNormalized, c.NationalIDCipher, c.NationalIDHMAC, c.NationalIDMasked,
		c.HouseholdType, c.Gender, c.BirthDate, c.CareContactRole, c.CareContactName, c.RegisteredAddress,
		c.HomeAddress, c.Region, c.LTCLevel, c.ServiceCategory, c.ServiceUsageType, c.ClaimStartDate, c.ClaimEndDate, c.Status, c.Remarks,
	).Scan(&c.CreatedAt, &c.UpdatedAt)
}

// Update 修改個案資料。
func (r *CaseRepository) Update(ctx context.Context, c *app.Case) error {
	query := `
		UPDATE cases
		SET name = $2, name_normalized = $3, home_address = $4, region = $5, ltc_level = $6,
		    service_category = $7, service_usage_type = $8, claim_start_date = $9, claim_end_date = $10,
		    status = $11, household_type = $12, gender = $13, birth_date = $14,
		    care_contact_role = $15, care_contact_name = $16, registered_address = $17, remarks = $18, updated_at = now()
		WHERE id = $1
		RETURNING updated_at
	`
	return r.db.QueryRow(ctx, query,
		c.ID, c.Name, c.NameNormalized, c.HomeAddress, c.Region, c.LTCLevel,
		c.ServiceCategory, c.ServiceUsageType, c.ClaimStartDate, c.ClaimEndDate, c.Status,
		c.HouseholdType, c.Gender, c.BirthDate, c.CareContactRole, c.CareContactName, c.RegisteredAddress, c.Remarks,
	).Scan(&c.UpdatedAt)
}

// CreateSchedule 建立排班設定與對應的 legs（包在同一個事務中）。
// 若 ctx 已掛載外層事務（見 pgxdb.TxRunner），排班與 legs 寫入會併入該事務，
// 由外層決定 commit／rollback；否則自行開啟並管理事務。
func (r *CaseRepository) CreateSchedule(ctx context.Context, s *app.CaseSchedule) error {
	if tx, ok := pgxdb.TxFromContext(ctx); ok {
		return r.insertSchedule(ctx, tx, s)
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if err := r.insertSchedule(ctx, tx, s); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *CaseRepository) insertSchedule(ctx context.Context, tx pgxdb.Querier, s *app.CaseSchedule) error {
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}

	querySchedule := `
		INSERT INTO case_schedules (
			id, case_id, site_id, effective_range, weekdays, trip_pattern, unit_price, distance_km,
			service_duration_min, service_code, note
		) VALUES ($1, $2, $3, daterange($4, $5, '[]'), $6, $7, $8, $9, $10, $11, $12)
		RETURNING created_at, updated_at
	`
	var toVal *time.Time
	if s.EffectiveTo != nil {
		toVal = s.EffectiveTo
	}
	err := tx.QueryRow(ctx, querySchedule,
		s.ID, s.CaseID, s.SiteID, s.EffectiveFrom, toVal, s.Weekdays, s.TripPattern,
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
		SELECT s.id, s.case_id, s.site_id, st.name as site_name, s.weekdays, s.trip_pattern,
		       s.unit_price, s.distance_km, s.service_duration_min, s.service_code, s.note,
		       s.created_at, s.updated_at
		FROM case_schedules s
		JOIN sites st ON s.site_id = st.id
		WHERE s.case_id = $1
		  AND s.effective_range @> $2::date
		LIMIT 1
	`
	var s app.CaseSchedule
	err := r.db.QueryRow(ctx, query, caseID, serviceDate).Scan(
		&s.ID, &s.CaseID, &s.SiteID, &s.SiteName, &s.Weekdays, &s.TripPattern,
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
		LEFT JOIN vehicles v ON l.vehicle_id = v.id
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

	return &s, nil
}

// GetActiveSchedulesForMonth 查詢特定月份所有在案個案的有效排班與 Legs。
func (r *CaseRepository) GetActiveSchedulesForMonth(ctx context.Context, year, month int, region string) ([]app.ActiveCaseScheduleInfo, error) {
	if r.db == nil {
		return []app.ActiveCaseScheduleInfo{}, nil
	}

	firstDay := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	lastDay := firstDay.AddDate(0, 1, -1)

	query := `
		SELECT c.id, c.code, c.name, COALESCE(c.region, ''), COALESCE(c.claim_start_date, c.created_at::date), c.claim_end_date,
		       s.id as schedule_id, s.site_id, st.open_days,
		       lower(s.effective_range) as eff_from,
		       CASE WHEN upper_inf(s.effective_range) THEN NULL ELSE to_char(upper(s.effective_range), 'YYYY-MM-DD') END as eff_to_str,
		       s.weekdays, s.trip_pattern
		FROM cases c
		JOIN case_schedules s ON c.id = s.case_id
		JOIN sites st ON s.site_id = st.id
		WHERE c.status = 'active'
		  AND ($1 = '' OR c.region = $1)
		  AND s.effective_range && daterange($2, $3, '[]')
		ORDER BY c.code ASC
	`
	rows, err := r.db.Query(ctx, query, region, firstDay, lastDay)
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
			&sr.info.CaseID, &sr.info.CaseCode, &sr.info.CaseName, &sr.info.Region,
			&sr.info.ClaimStartDate, &sr.info.ClaimEndDate,
			&sr.scheduleID, &sr.info.SiteID, &sr.info.SiteOpenDays,
			&sr.info.EffectiveFrom, &effToStr,
			&sr.info.Weekdays, &sr.info.TripPattern,
		); err != nil {
			return nil, err
		}
		if effToStr != nil && *effToStr != "" {
			if t, err := time.Parse("2006-01-02", *effToStr); err == nil {
				sr.info.EffectiveTo = &t
			}
		}
		list = append(list, sr)
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
		lRows.Close()
		results = append(results, item.info)
	}

	return results, nil
}

// ListNameIndex 取得在案個案的姓名索引。只取比對姓名需要的欄位，避免為了推薦
// 匯報欄位對應而把整份含密文的個案主檔載進記憶體。
func (r *CaseRepository) ListNameIndex(ctx context.Context) ([]app.CaseNameRef, error) {
	if r.db == nil {
		return nil, nil
	}

	rows, err := r.db.Query(ctx, `
		SELECT id, code, name, name_normalized
		FROM cases
		WHERE status = 'active'
		ORDER BY code ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to query case name index: %w", err)
	}
	defer rows.Close()

	var list []app.CaseNameRef
	for rows.Next() {
		var c app.CaseNameRef
		if err := rows.Scan(&c.ID, &c.Code, &c.Name, &c.NameNormalized); err != nil {
			return nil, err
		}
		list = append(list, c)
	}
	return list, rows.Err()
}
