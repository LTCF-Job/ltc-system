package infra

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"ltc-system/apps/api/internal/modules/ride/app"
	"ltc-system/apps/api/internal/platform/pgxdb"
)

// RideRepository 提供表單、提交紀錄、回報來源與搭乘紀錄之存取。
type RideRepository struct {
	db *pgxpool.Pool
}

// NewRideRepository 建立 RideRepository 實例。
func NewRideRepository(db *pgxpool.Pool) *RideRepository {
	return &RideRepository{db: db}
}

// GetFormColumns 取得特定表單之所有欄位定義。
func (r *RideRepository) GetFormColumns(ctx context.Context, formID uuid.UUID) ([]app.FormColumn, error) {
	query := `
		SELECT id, form_id, column_index, column_header, cleaned_name, kind, mapping_status,
		       case_id, leg_seq, suggested_case_id, suggestion_score, created_at, updated_at
		FROM form_columns
		WHERE form_id = $1
		ORDER BY column_index ASC
	`
	db := pgxdb.FromContext(ctx, r.db)
	rows, err := db.Query(ctx, query, formID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cols []app.FormColumn
	for rows.Next() {
		var col app.FormColumn
		if err := rows.Scan(
			&col.ID, &col.FormID, &col.ColumnIndex, &col.ColumnHeader, &col.CleanedName, &col.Kind, &col.MappingStatus,
			&col.CaseID, &col.LegSeq, &col.SuggestedCaseID, &col.SuggestionScore, &col.CreatedAt, &col.UpdatedAt,
		); err != nil {
			return nil, err
		}
		cols = append(cols, col)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return cols, nil
}

// SaveFormSubmission 先完整寫入原始 payload 與中繼資訊。
func (r *RideRepository) SaveFormSubmission(
	ctx context.Context,
	formID uuid.UUID,
	serviceDate time.Time,
	submittedAt time.Time,
	driverNameRaw string,
	driverID *uuid.UUID,
	source string,
	payload map[string]interface{},
	issueText string,
	anomalyFlags []string,
) (uuid.UUID, error) {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return uuid.Nil, err
	}

	submissionID := uuid.New()
	query := `
		INSERT INTO form_submissions (
			id, form_id, service_date, submitted_at, driver_name_raw, driver_id, source, payload, issue_text, anomaly_flags
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (form_id, service_date, submitted_at) DO UPDATE
		SET payload = EXCLUDED.payload, issue_text = EXCLUDED.issue_text, anomaly_flags = EXCLUDED.anomaly_flags
		RETURNING id
	`
	db := pgxdb.FromContext(ctx, r.db)
	err = db.QueryRow(ctx, query,
		submissionID, formID, serviceDate, submittedAt, driverNameRaw, driverID, source, string(payloadBytes), issueText, anomalyFlags,
	).Scan(&submissionID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to save form submission: %w", err)
	}
	return submissionID, nil
}

// InsertRideSource 寫入單筆來源搭乘回報。
func (r *RideRepository) InsertRideSource(
	ctx context.Context,
	submissionID, caseID uuid.UUID,
	serviceDate time.Time,
	legSeq int16,
	vehicleID uuid.UUID,
	driverID *uuid.UUID,
	reported string,
	colIdx int,
) error {
	query := `
		INSERT INTO ride_sources (
			id, submission_id, case_id, service_date, leg_seq, vehicle_id, driver_id, reported, source_column_index
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	db := pgxdb.FromContext(ctx, r.db)
	_, err := db.Exec(ctx, query,
		uuid.New(), submissionID, caseID, serviceDate, legSeq, vehicleID, driverID, reported, colIdx,
	)
	return err
}

// ListSubmissionAnswersForColumn 取出某表單既有回報中，指定欄位表頭留在 payload 的原始
// 儲存格文字；欄位當初上傳時尚未對應個案也會存在 payload 裡，回填時不需要原始檔案。
func (r *RideRepository) ListSubmissionAnswersForColumn(ctx context.Context, formID uuid.UUID, columnHeader string) ([]app.SubmissionAnswer, error) {
	query := `
		SELECT id, service_date, driver_id, payload->'answers'->>$2
		FROM form_submissions
		WHERE form_id = $1 AND payload->'answers' ? $2
	`
	db := pgxdb.FromContext(ctx, r.db)
	rows, err := db.Query(ctx, query, formID, columnHeader)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []app.SubmissionAnswer
	for rows.Next() {
		var a app.SubmissionAnswer
		if err := rows.Scan(&a.SubmissionID, &a.ServiceDate, &a.DriverID, &a.Value); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// ListSubmissionsForForms 取出指定表單目前存在的所有回報列，含每欄留在 payload 的
// 原始儲存格文字，供 driverreport 彙整待維護清單時比對哪些欄位這一列「有回報」但仍
// 待對應個案。
func (r *RideRepository) ListSubmissionsForForms(ctx context.Context, formIDs []uuid.UUID) ([]app.SubmissionFull, error) {
	if len(formIDs) == 0 {
		return nil, nil
	}
	query := `
		SELECT fs.id, fs.form_id, COALESCE(df.title, ''), COALESCE(v.display_name, ''),
		       fs.service_date, fs.payload->'answers'
		FROM form_submissions fs
		LEFT JOIN driver_report_forms df ON fs.form_id = df.id
		LEFT JOIN vehicles v ON df.vehicle_id = v.id
		WHERE fs.form_id = ANY($1::uuid[])
	`
	db := pgxdb.FromContext(ctx, r.db)
	rows, err := db.Query(ctx, query, pgxdb.UUIDStrings(formIDs))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []app.SubmissionFull
	for rows.Next() {
		var s app.SubmissionFull
		var answersRaw []byte
		if err := rows.Scan(&s.SubmissionID, &s.FormID, &s.FormTitle, &s.VehicleName, &s.ServiceDate, &answersRaw); err != nil {
			return nil, err
		}
		s.Answers = map[string]string{}
		if len(answersRaw) > 0 {
			if err := json.Unmarshal(answersRaw, &s.Answers); err != nil {
				return nil, err
			}
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// ListUnmatchedDriverSubmissions 取出目前駕駛人姓名比對不到司機主檔的既有回報。
func (r *RideRepository) ListUnmatchedDriverSubmissions(ctx context.Context) ([]app.UnmatchedDriverSubmission, error) {
	query := `
		SELECT fs.id, fs.form_id, COALESCE(df.title, ''), COALESCE(v.display_name, ''),
		       fs.service_date, fs.driver_name_raw
		FROM form_submissions fs
		LEFT JOIN driver_report_forms df ON fs.form_id = df.id
		LEFT JOIN vehicles v ON df.vehicle_id = v.id
		WHERE fs.driver_id IS NULL AND COALESCE(fs.driver_name_raw, '') <> ''
	`
	db := pgxdb.FromContext(ctx, r.db)
	rows, err := db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []app.UnmatchedDriverSubmission
	for rows.Next() {
		var u app.UnmatchedDriverSubmission
		if err := rows.Scan(&u.SubmissionID, &u.FormID, &u.FormTitle, &u.VehicleName, &u.ServiceDate, &u.DriverNameRaw); err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

// UpdateSubmissionDriverID 回填某筆提交紀錄的司機。
func (r *RideRepository) UpdateSubmissionDriverID(ctx context.Context, submissionID, driverID uuid.UUID) error {
	_, err := pgxdb.FromContext(ctx, r.db).Exec(ctx,
		`UPDATE form_submissions SET driver_id = $2 WHERE id = $1`, submissionID, driverID)
	return err
}

// ListRideSourcesForSubmission 取出某筆提交紀錄已展開的搭乘來源，回填司機時需要逐筆
// 更新來源與重算搭乘紀錄。
func (r *RideRepository) ListRideSourcesForSubmission(ctx context.Context, submissionID uuid.UUID) ([]app.RideSourceForSubmission, error) {
	query := `
		SELECT id, case_id, service_date, leg_seq, vehicle_id
		FROM ride_sources
		WHERE submission_id = $1
	`
	db := pgxdb.FromContext(ctx, r.db)
	rows, err := db.Query(ctx, query, submissionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []app.RideSourceForSubmission
	for rows.Next() {
		var s app.RideSourceForSubmission
		if err := rows.Scan(&s.ID, &s.CaseID, &s.ServiceDate, &s.LegSeq, &s.VehicleID); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// UpdateRideSourceDriverID 回填某筆搭乘來源的司機。
func (r *RideRepository) UpdateRideSourceDriverID(ctx context.Context, sourceID, driverID uuid.UUID) error {
	_, err := pgxdb.FromContext(ctx, r.db).Exec(ctx,
		`UPDATE ride_sources SET driver_id = $2 WHERE id = $1`, sourceID, driverID)
	return err
}

// GetRideRecordForSlot 查詢指定個案、日期、時段之既有搭乘主紀錄。
func (r *RideRepository) GetRideRecordForSlot(ctx context.Context, caseID uuid.UUID, serviceDate time.Time, legSeq int16) (*app.RideRecord, error) {
	query := `
		SELECT id, case_id, service_date, leg_seq, merged_status, effective_status,
		       vehicle_id, driver_id, has_conflict, conflict_resolved_at, conflict_resolved_by,
		       to_char(depart_time_override, 'HH24:MI'), duration_min_override, not_claimed_aa09,
		       corrected_by, corrected_at, correction_reason, based_on_fingerprint, created_at, updated_at
		FROM ride_records
		WHERE case_id = $1 AND service_date = $2 AND leg_seq = $3
		LIMIT 1
	`
	var rec app.RideRecord
	db := pgxdb.FromContext(ctx, r.db)
	err := db.QueryRow(ctx, query, caseID, serviceDate, legSeq).Scan(
		&rec.ID, &rec.CaseID, &rec.ServiceDate, &rec.LegSeq, &rec.MergedStatus, &rec.EffectiveStatus,
		&rec.VehicleID, &rec.DriverID, &rec.HasConflict, &rec.ConflictResolvedAt, &rec.ConflictResolvedBy,
		&rec.DepartTimeOverride, &rec.DurationMinOverride, &rec.NotClaimedAA09,
		&rec.CorrectedBy, &rec.CorrectedAt, &rec.CorrectionReason, &rec.BasedOnFingerprint, &rec.CreatedAt, &rec.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &rec, nil
}

// UpsertRideRecord 建立或更新 ride_records 主紀錄。
func (r *RideRepository) UpsertRideRecord(ctx context.Context, rec *app.RideRecord) error {
	query := `
		INSERT INTO ride_records (
			id, case_id, service_date, leg_seq, merged_status, effective_status,
			vehicle_id, driver_id, has_conflict, conflict_resolved_at, conflict_resolved_by,
			depart_time_override, duration_min_override,
			not_claimed_aa09, corrected_by, corrected_at, correction_reason, based_on_fingerprint
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18
		)
		ON CONFLICT (case_id, service_date, leg_seq) DO UPDATE
		SET merged_status = EXCLUDED.merged_status,
		    effective_status = EXCLUDED.effective_status,
		    vehicle_id = EXCLUDED.vehicle_id,
		    driver_id = EXCLUDED.driver_id,
		    has_conflict = EXCLUDED.has_conflict,
		    conflict_resolved_at = EXCLUDED.conflict_resolved_at,
		    conflict_resolved_by = EXCLUDED.conflict_resolved_by,
		    depart_time_override = COALESCE(EXCLUDED.depart_time_override, ride_records.depart_time_override),
		    duration_min_override = COALESCE(EXCLUDED.duration_min_override, ride_records.duration_min_override),
		    not_claimed_aa09 = EXCLUDED.not_claimed_aa09,
		    corrected_by = EXCLUDED.corrected_by,
		    corrected_at = EXCLUDED.corrected_at,
		    correction_reason = EXCLUDED.correction_reason,
		    based_on_fingerprint = EXCLUDED.based_on_fingerprint,
		    updated_at = now()
		RETURNING created_at, updated_at
	`
	if rec.ID == uuid.Nil {
		rec.ID = uuid.New()
	}
	db := pgxdb.FromContext(ctx, r.db)
	return db.QueryRow(ctx, query,
		rec.ID, rec.CaseID, rec.ServiceDate, rec.LegSeq, rec.MergedStatus, rec.EffectiveStatus,
		rec.VehicleID, rec.DriverID, rec.HasConflict, rec.ConflictResolvedAt, rec.ConflictResolvedBy,
		rec.DepartTimeOverride, rec.DurationMinOverride,
		rec.NotClaimedAA09, rec.CorrectedBy, rec.CorrectedAt, rec.CorrectionReason, rec.BasedOnFingerprint,
	).Scan(&rec.CreatedAt, &rec.UpdatedAt)
}

// CorrectRideRecord 人工更正搭乘紀錄。
func (r *RideRepository) CorrectRideRecord(
	ctx context.Context,
	rideID uuid.UUID,
	effectiveStatus *string,
	vehicleID *uuid.UUID,
	driverID *uuid.UUID,
	departTimeOverride *string,
	durationMinOverride *int16,
	notClaimedAA09 *bool,
	reason *string,
	operatorID uuid.UUID,
) error {
	query := `
		UPDATE ride_records
		SET effective_status = COALESCE($2, effective_status),
		    vehicle_id = COALESCE($3, vehicle_id),
		    driver_id = COALESCE($4, driver_id),
		    depart_time_override = $5::time,
		    duration_min_override = $6,
		    not_claimed_aa09 = COALESCE($7, not_claimed_aa09),
		    correction_reason = $8,
		    corrected_by = $9,
		    corrected_at = now(),
		    updated_at = now()
		WHERE id = $1
	`
	db := pgxdb.FromContext(ctx, r.db)
	_, err := db.Exec(ctx, query,
		rideID, effectiveStatus, vehicleID, driverID, departTimeOverride,
		durationMinOverride, notClaimedAA09, reason, operatorID,
	)
	return err
}

// CorrectRideRecordWithFingerprint 以單一 UPDATE 原子保存人工更正與其來源快照。
func (r *RideRepository) CorrectRideRecordWithFingerprint(
	ctx context.Context,
	rideID uuid.UUID,
	effectiveStatus *string,
	vehicleID *uuid.UUID,
	driverID *uuid.UUID,
	departTimeOverride *string,
	durationMinOverride *int16,
	notClaimedAA09 *bool,
	reason *string,
	operatorID uuid.UUID,
	fingerprint string,
) error {
	query := `
		UPDATE ride_records
		SET effective_status = COALESCE($2, effective_status),
		    vehicle_id = COALESCE($3, vehicle_id),
		    driver_id = COALESCE($4, driver_id),
		    depart_time_override = $5::time,
		    duration_min_override = $6,
		    not_claimed_aa09 = COALESCE($7, not_claimed_aa09),
		    correction_reason = $8,
		    corrected_by = $9,
		    corrected_at = now(),
		    based_on_fingerprint = $10,
		    updated_at = now()
		WHERE id = $1
	`
	_, err := pgxdb.FromContext(ctx, r.db).Exec(ctx, query,
		rideID, effectiveStatus, vehicleID, driverID, departTimeOverride,
		durationMinOverride, notClaimedAA09, reason, operatorID, fingerprint,
	)
	return err
}

// SetCorrectionFingerprint 保存人工更正當下所依據的來源快照。
func (r *RideRepository) SetCorrectionFingerprint(ctx context.Context, rideID uuid.UUID, fingerprint string) error {
	_, err := pgxdb.FromContext(ctx, r.db).Exec(ctx, `
		UPDATE ride_records
		SET based_on_fingerprint = $2, updated_at = now()
		WHERE id = $1
	`, rideID, fingerprint)
	return err
}

// ListRideSourceSlotsForForm 列出指定匯報表在這些服務日期底下，由匯入寫入來源列的搭乘座標。
//
// 供覆蓋式重匯先取得受影響 slot：來源列刪除後就查不到它們，必須在刪除前收集。
// 篩選條件與 DeleteFormSubmissions 一致，兩者的範圍必須永遠相同。
func (r *RideRepository) ListRideSourceSlotsForForm(ctx context.Context, formID uuid.UUID, dates []time.Time) ([]app.RideSlot, error) {
	if len(dates) == 0 {
		return nil, nil
	}
	query := `
		SELECT DISTINCT rs.case_id, rs.service_date, rs.leg_seq
		FROM ride_sources rs
		JOIN form_submissions fs ON fs.id = rs.submission_id
		WHERE fs.form_id = $1 AND fs.service_date = ANY($2::date[]) AND fs.source = 'import'
	`
	db := pgxdb.FromContext(ctx, r.db)
	rows, err := db.Query(ctx, query, formID, dates)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var slots []app.RideSlot
	for rows.Next() {
		var slot app.RideSlot
		if err := rows.Scan(&slot.CaseID, &slot.ServiceDate, &slot.LegSeq); err != nil {
			return nil, err
		}
		slots = append(slots, slot)
	}
	return slots, rows.Err()
}

// DeleteFormSubmissions 刪除指定匯報表在這些服務日期、由匯入產生的提交紀錄，回傳刪除筆數。
//
// ride_sources 由 submission_id 的 ON DELETE CASCADE 連帶清除；只影響本匯報表，
// 其他車輛對同一 slot 的混車來源不受牽連。限定 source = 'import' 是因為覆蓋語意只涵蓋
// 匯入產生的資料，人工補登的提交不該被下一次重匯抹掉。
func (r *RideRepository) DeleteFormSubmissions(ctx context.Context, formID uuid.UUID, dates []time.Time) (int, error) {
	if len(dates) == 0 {
		return 0, nil
	}
	db := pgxdb.FromContext(ctx, r.db)
	tag, err := db.Exec(ctx, `DELETE FROM form_submissions WHERE form_id = $1 AND service_date = ANY($2::date[]) AND source = 'import'`, formID, dates)
	if err != nil {
		return 0, err
	}
	return int(tag.RowsAffected()), nil
}

// ListImportedMonths 統計每份匯報表各月份由匯入寫入的提交筆數與最後一次匯入時間。
//
// 只算 source = 'import'，與 DeleteFormSubmissions 的覆蓋範圍一致：人工補登的提交不會
// 被重匯覆蓋，也就不該讓使用者以為那個月是匯入來的。
func (r *RideRepository) ListImportedMonths(ctx context.Context) ([]app.ImportedMonth, error) {
	if r.db == nil {
		return nil, fmt.Errorf("ride database is not configured")
	}
	query := `
		SELECT form_id, to_char(service_date, 'YYYY-MM'), count(*), max(submitted_at)
		FROM form_submissions
		WHERE source = 'import'
		GROUP BY form_id, to_char(service_date, 'YYYY-MM')
		ORDER BY form_id, 2 DESC
	`
	db := pgxdb.FromContext(ctx, r.db)
	rows, err := db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var months []app.ImportedMonth
	for rows.Next() {
		var m app.ImportedMonth
		if err := rows.Scan(&m.FormID, &m.YearMonth, &m.SubmissionCount, &m.LastImportedAt); err != nil {
			return nil, err
		}
		months = append(months, m)
	}
	return months, rows.Err()
}

// ListSubmissionsForFormMonth 取出某份匯報表在 [monthStart, monthEnd) 區間內的逐日原始回報，
// 供總覽頁鑽取單一月份時直接顯示原始儲存格文字，不需重新開啟原始檔案。
func (r *RideRepository) ListSubmissionsForFormMonth(ctx context.Context, formID uuid.UUID, monthStart, monthEnd time.Time) ([]app.MonthSubmissionDetail, error) {
	query := `
		SELECT service_date, COALESCE(driver_name_raw, ''), COALESCE(payload->>'remark', ''), payload->'answers'
		FROM form_submissions
		WHERE form_id = $1 AND service_date >= $2 AND service_date < $3
		ORDER BY service_date ASC
	`
	db := pgxdb.FromContext(ctx, r.db)
	rows, err := db.Query(ctx, query, formID, monthStart, monthEnd)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []app.MonthSubmissionDetail
	for rows.Next() {
		var d app.MonthSubmissionDetail
		var answersRaw []byte
		if err := rows.Scan(&d.ServiceDate, &d.DriverNameRaw, &d.Remark, &answersRaw); err != nil {
			return nil, err
		}
		d.Answers = map[string]string{}
		if len(answersRaw) > 0 {
			if err := json.Unmarshal(answersRaw, &d.Answers); err != nil {
				return nil, err
			}
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

// ListRideEntriesForFormMonth 取出某份匯報表在 [monthStart, monthEnd) 區間內展開後的個案搭乘
// 紀錄，供總覽頁鑽取單一月份實際寫入了哪些個案與趟次。
func (r *RideRepository) ListRideEntriesForFormMonth(ctx context.Context, formID uuid.UUID, monthStart, monthEnd time.Time) ([]app.MonthRideEntry, error) {
	query := `
		SELECT rs.case_id, COALESCE(c.name, ''), rs.service_date, rs.leg_seq, rs.reported,
		       rs.driver_id, COALESCE(d.name, ''), rs.vehicle_id
		FROM ride_sources rs
		JOIN form_submissions fs ON fs.id = rs.submission_id
		LEFT JOIN cases c ON c.id = rs.case_id
		LEFT JOIN drivers d ON d.id = rs.driver_id
		WHERE fs.form_id = $1 AND rs.service_date >= $2 AND rs.service_date < $3
		ORDER BY rs.service_date ASC, rs.leg_seq ASC
	`
	db := pgxdb.FromContext(ctx, r.db)
	rows, err := db.Query(ctx, query, formID, monthStart, monthEnd)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []app.MonthRideEntry
	for rows.Next() {
		var e app.MonthRideEntry
		if err := rows.Scan(&e.CaseID, &e.CaseName, &e.ServiceDate, &e.LegSeq, &e.Reported,
			&e.DriverID, &e.DriverName, &e.VehicleID); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// DeleteDerivedRideRecord 刪除純由匯入衍生的搭乘紀錄；帶有人工更正、衝突裁決或
// 不申報標記的紀錄一律保留，避免覆蓋式重匯抹掉人工成果。
func (r *RideRepository) DeleteDerivedRideRecord(ctx context.Context, caseID uuid.UUID, serviceDate time.Time, legSeq int16) error {
	query := `
		DELETE FROM ride_records
		WHERE case_id = $1 AND service_date = $2 AND leg_seq = $3
		  AND corrected_at IS NULL
		  AND conflict_resolved_at IS NULL
		  AND not_claimed_aa09 = false
	`
	db := pgxdb.FromContext(ctx, r.db)
	_, err := db.Exec(ctx, query, caseID, serviceDate, legSeq)
	return err
}

// GetRideRecordByID 依 ID 查詢單筆搭乘紀錄，查無資料回傳 nil, nil。
func (r *RideRepository) GetRideRecordByID(ctx context.Context, id uuid.UUID) (*app.RideRecord, error) {
	query := `
		SELECT id, case_id, service_date, leg_seq, merged_status, effective_status,
		       vehicle_id, driver_id, has_conflict, conflict_resolved_at, conflict_resolved_by, conflict_resolution_note,
		       to_char(depart_time_override, 'HH24:MI'), duration_min_override, not_claimed_aa09,
		       corrected_by, corrected_at, correction_reason, based_on_fingerprint, created_at, updated_at
		FROM ride_records
		WHERE id = $1
	`
	var rec app.RideRecord
	db := pgxdb.FromContext(ctx, r.db)
	err := db.QueryRow(ctx, query, id).Scan(
		&rec.ID, &rec.CaseID, &rec.ServiceDate, &rec.LegSeq, &rec.MergedStatus, &rec.EffectiveStatus,
		&rec.VehicleID, &rec.DriverID, &rec.HasConflict, &rec.ConflictResolvedAt, &rec.ConflictResolvedBy, &rec.ConflictResolutionNote,
		&rec.DepartTimeOverride, &rec.DurationMinOverride, &rec.NotClaimedAA09,
		&rec.CorrectedBy, &rec.CorrectedAt, &rec.CorrectionReason, &rec.BasedOnFingerprint, &rec.CreatedAt, &rec.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &rec, nil
}

// ResolveConflict 裁決混車衝突；回傳 false 代表該筆已被他人裁決過，不會覆寫既有裁決。
func (r *RideRepository) ResolveConflict(ctx context.Context, rideID, vehicleID uuid.UUID, driverID *uuid.UUID, note *string, operatorID uuid.UUID) (bool, error) {
	query := `
		UPDATE ride_records
		SET vehicle_id = $2,
		    driver_id = $3,
		    conflict_resolution_note = $4,
		    conflict_resolved_at = now(),
		    conflict_resolved_by = $5,
		    updated_at = now()
		WHERE id = $1 AND has_conflict = true AND conflict_resolved_at IS NULL
	`
	db := pgxdb.FromContext(ctx, r.db)
	tag, err := db.Exec(ctx, query, rideID, vehicleID, driverID, note, operatorID)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() == 1, nil
}
