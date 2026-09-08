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

// SaveFormSubmission 寫入這一天原始 payload 與中繼資訊；一車一天只有一筆現行資料，
// 同一天再次上傳時原地更新，而不是疊加出多筆（見 docs/decisions/driver-report-import-overwrite.md）。
//
// driver_id 用 COALESCE 保留既有值：這次沒能解析出司機（EXCLUDED 為 NULL）不代表
// 之前經人工綁定或先前解析成功的司機是錯的，重傳不能把已修好的司機又蓋回未比對。
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
		ON CONFLICT (form_id, service_date) DO UPDATE
		SET submitted_at = EXCLUDED.submitted_at,
		    driver_name_raw = EXCLUDED.driver_name_raw,
		    driver_id = COALESCE(EXCLUDED.driver_id, form_submissions.driver_id),
		    source = EXCLUDED.source,
		    payload = EXCLUDED.payload,
		    issue_text = EXCLUDED.issue_text,
		    anomaly_flags = EXCLUDED.anomaly_flags
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

// InsertRideSource 寫入單筆來源搭乘回報；submittedAt 是這筆值實際記錄下來的時間，
// 各自固定不隨同一天之後的其他上傳而變動（見 ListRideSourcesForSlot 的說明）。
func (r *RideRepository) InsertRideSource(
	ctx context.Context,
	submissionID, caseID uuid.UUID,
	serviceDate time.Time,
	legSeq int16,
	vehicleID uuid.UUID,
	driverID *uuid.UUID,
	reported string,
	colIdx int,
	submittedAt time.Time,
) error {
	query := `
		INSERT INTO ride_sources (
			id, submission_id, case_id, service_date, leg_seq, vehicle_id, driver_id, reported, source_column_index, submitted_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`
	db := pgxdb.FromContext(ctx, r.db)
	_, err := db.Exec(ctx, query,
		uuid.New(), submissionID, caseID, serviceDate, legSeq, vehicleID, driverID, reported, colIdx, submittedAt,
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
		LEFT JOIN vehicles v ON df.vehicle_id = v.id AND v.deleted_at IS NULL
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

// ListUnmatchedDriverSubmissions 取出目前駕駛人姓名比對不到司機主檔的既有回報，含
// VehicleID、SubmittedAt 與完整原始答案，供司機補綁定後不需重新上傳原始檔案即可
// 逐欄重新比對寫入（見 RideService.BackfillDriver）。
func (r *RideRepository) ListUnmatchedDriverSubmissions(ctx context.Context) ([]app.UnmatchedDriverSubmission, error) {
	query := `
		SELECT fs.id, fs.form_id, COALESCE(df.title, ''), df.vehicle_id, COALESCE(v.display_name, ''),
		       fs.service_date, fs.submitted_at, fs.driver_name_raw, fs.payload->'answers'
		FROM form_submissions fs
		LEFT JOIN driver_report_forms df ON fs.form_id = df.id
		LEFT JOIN vehicles v ON df.vehicle_id = v.id AND v.deleted_at IS NULL
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
		var answersRaw []byte
		if err := rows.Scan(&u.SubmissionID, &u.FormID, &u.FormTitle, &u.VehicleID, &u.VehicleName, &u.ServiceDate, &u.SubmittedAt, &u.DriverNameRaw, &answersRaw); err != nil {
			return nil, err
		}
		u.Answers = map[string]string{}
		if len(answersRaw) > 0 {
			if err := json.Unmarshal(answersRaw, &u.Answers); err != nil {
				return nil, err
			}
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

// ListRideSourceSlotsForSubmission 取出某筆提交紀錄目前展開出的所有搭乘來源 slot。
// form_submissions 被 ride_sources 與 ride_source_row_conflicts 以 ON DELETE CASCADE 參照，
// 刪除提交紀錄會連帶刪掉這些來源，因此呼叫端必須先取得 slot 清單，刪除後逐一重算搭乘紀錄，
// 否則會留下對不上任何來源的 ride_records。
func (r *RideRepository) ListRideSourceSlotsForSubmission(ctx context.Context, submissionID uuid.UUID) ([]app.RideSourceSlot, error) {
	rows, err := pgxdb.FromContext(ctx, r.db).Query(ctx, `
		SELECT case_id, service_date, leg_seq, vehicle_id
		FROM ride_sources
		WHERE submission_id = $1
	`, submissionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []app.RideSourceSlot
	for rows.Next() {
		var slot app.RideSourceSlot
		if err := rows.Scan(&slot.CaseID, &slot.ServiceDate, &slot.LegSeq, &slot.VehicleID); err != nil {
			return nil, err
		}
		out = append(out, slot)
	}
	return out, rows.Err()
}

// DeleteSubmission 移除一筆提交紀錄；rowsAffected=0 代表該列不存在。呼叫端負責先確認
// 沒有搭乘來源掛在這筆提交上。
func (r *RideRepository) DeleteSubmission(ctx context.Context, submissionID uuid.UUID) (int64, error) {
	tag, err := pgxdb.FromContext(ctx, r.db).Exec(ctx,
		`DELETE FROM form_submissions WHERE id = $1`, submissionID)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

// UpdateSubmissionDriverID 回填某筆提交紀錄的司機。
func (r *RideRepository) UpdateSubmissionDriverID(ctx context.Context, submissionID, driverID uuid.UUID) error {
	_, err := pgxdb.FromContext(ctx, r.db).Exec(ctx,
		`UPDATE form_submissions SET driver_id = $2 WHERE id = $1`, submissionID, driverID)
	return err
}

// UpsertRideSourceRowConflict 暫存或更新一筆「同車同個案」衝突。同一 slot 只保留一筆
// 未解決的衝突（見 migration 000035 的 partial unique index）：第二次上傳到同一個未
// 解決的衝突時，只更新新值，previous_* 保持第一次偵測到衝突時的既有資料不動。
func (r *RideRepository) UpsertRideSourceRowConflict(ctx context.Context, in app.RowConflictInput) (uuid.UUID, error) {
	query := `
		INSERT INTO ride_source_row_conflicts (
			id, form_id, vehicle_id, case_id, service_date, leg_seq, source_column_index,
			previous_submission_id, previous_reported, previous_driver_id, previous_submitted_at,
			new_submission_id, new_reported, new_driver_id, new_submitted_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
		ON CONFLICT (vehicle_id, case_id, service_date, leg_seq) WHERE resolved_at IS NULL DO UPDATE
		SET new_submission_id = EXCLUDED.new_submission_id,
		    new_reported = EXCLUDED.new_reported,
		    new_driver_id = EXCLUDED.new_driver_id,
		    new_submitted_at = EXCLUDED.new_submitted_at,
		    detected_at = now()
		RETURNING id
	`
	var conflictID uuid.UUID
	db := pgxdb.FromContext(ctx, r.db)
	err := db.QueryRow(ctx, query,
		uuid.New(), in.FormID, in.VehicleID, in.CaseID, in.ServiceDate, in.LegSeq, in.SourceColumnIndex,
		in.PreviousSubmissionID, in.PreviousReported, in.PreviousDriverID, in.PreviousSubmittedAt,
		in.NewSubmissionID, in.NewReported, in.NewDriverID, in.NewSubmittedAt,
	).Scan(&conflictID)
	return conflictID, err
}

// ListPendingRowConflicts 取出目前所有尚未解決的「同車同個案」衝突，附上顯示用名稱。
func (r *RideRepository) ListPendingRowConflicts(ctx context.Context) ([]app.RowConflict, error) {
	query := `
		SELECT c.id, c.form_id, COALESCE(df.title, ''), c.vehicle_id, COALESCE(v.display_name, ''),
		       c.case_id, COALESCE(cs.name, ''), c.service_date, c.leg_seq, c.source_column_index,
		       c.previous_submission_id, c.previous_reported, c.previous_driver_id, COALESCE(pd.name, ''), c.previous_submitted_at,
		       c.new_submission_id, c.new_reported, c.new_driver_id, COALESCE(nd.name, ''), c.new_submitted_at,
		       c.detected_at
		FROM ride_source_row_conflicts c
		LEFT JOIN driver_report_forms df ON c.form_id = df.id
		LEFT JOIN vehicles v ON c.vehicle_id = v.id AND v.deleted_at IS NULL
		LEFT JOIN cases cs ON c.case_id = cs.id
		LEFT JOIN drivers pd ON c.previous_driver_id = pd.id
		LEFT JOIN drivers nd ON c.new_driver_id = nd.id
		WHERE c.resolved_at IS NULL
		ORDER BY c.detected_at DESC
	`
	db := pgxdb.FromContext(ctx, r.db)
	rows, err := db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []app.RowConflict
	for rows.Next() {
		var c app.RowConflict
		if err := rows.Scan(
			&c.ID, &c.FormID, &c.FormTitle, &c.VehicleID, &c.VehicleName,
			&c.CaseID, &c.CaseName, &c.ServiceDate, &c.LegSeq, &c.SourceColumnIndex,
			&c.PreviousSubmissionID, &c.PreviousReported, &c.PreviousDriverID, &c.PreviousDriverName, &c.PreviousSubmittedAt,
			&c.NewSubmissionID, &c.NewReported, &c.NewDriverID, &c.NewDriverName, &c.NewSubmittedAt,
			&c.DetectedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// DeleteRowConflict 移除一筆尚未裁決的同車同個案衝突，供「忽略此筆」把資料從系統刪除；
// rowsAffected=0 代表該列不存在或已被裁決。既有搭乘來源與紀錄一律不動，維持既有值。
func (r *RideRepository) DeleteRowConflict(ctx context.Context, conflictID uuid.UUID) (int64, error) {
	db := pgxdb.FromContext(ctx, r.db)
	tag, err := db.Exec(ctx, `DELETE FROM ride_source_row_conflicts WHERE id = $1 AND resolved_at IS NULL`, conflictID)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

// ResolveRowConflict 裁決一筆同車同個案衝突；resolved=false 代表已被他人裁決過，
// 不會覆寫既有裁決。useNew=true 時回傳需要重放寫入搭乘來源的欄位。
func (r *RideRepository) ResolveRowConflict(ctx context.Context, conflictID uuid.UUID, useNew bool, operatorID uuid.UUID) (*app.AppliedRowConflict, bool, error) {
	resolution := "kept_previous"
	if useNew {
		resolution = "used_new"
	}
	query := `
		UPDATE ride_source_row_conflicts
		SET resolved_at = now(), resolved_by = $2, resolution = $3
		WHERE id = $1 AND resolved_at IS NULL
		RETURNING case_id, service_date, leg_seq, vehicle_id, source_column_index,
		          new_submission_id, new_reported, new_driver_id, new_submitted_at
	`
	db := pgxdb.FromContext(ctx, r.db)
	var applied app.AppliedRowConflict
	err := db.QueryRow(ctx, query, conflictID, operatorID, resolution).Scan(
		&applied.CaseID, &applied.ServiceDate, &applied.LegSeq, &applied.VehicleID, &applied.SourceColumnIndex,
		&applied.NewSubmissionID, &applied.NewReported, &applied.NewDriverID, &applied.NewSubmittedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, false, nil
		}
		return nil, false, err
	}
	if !useNew {
		return nil, true, nil
	}
	return &applied, true, nil
}

// GetRideRecordForSlot 查詢指定個案、日期、時段之既有搭乘主紀錄。
func (r *RideRepository) GetRideRecordForSlot(ctx context.Context, caseID uuid.UUID, serviceDate time.Time, legSeq int16) (*app.RideRecord, error) {
	query := `
		SELECT id, case_id, service_date, leg_seq, merged_status, effective_status,
		       vehicle_id, driver_id, has_conflict, conflict_resolved_at, conflict_resolved_by,
		       to_char(depart_time_override, 'HH24:MI'), duration_min_override, not_claimed_aa09,
		       corrected_by, corrected_at, correction_reason, COALESCE(based_on_fingerprint, ''), created_at, updated_at
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
	effectiveStatus app.PatchValue[string],
	vehicleID app.PatchValue[uuid.UUID],
	driverID app.PatchValue[uuid.UUID],
	departTimeOverride app.PatchValue[string],
	durationMinOverride app.PatchValue[int16],
	notClaimedAA09 app.PatchValue[bool],
	reason app.PatchValue[string],
	operatorID uuid.UUID,
) error {
	query := `
		UPDATE ride_records
		SET effective_status = CASE WHEN $2 THEN $3 ELSE effective_status END,
		    vehicle_id = CASE WHEN $4 THEN $5 ELSE vehicle_id END,
		    driver_id = CASE WHEN $6 THEN $7 ELSE driver_id END,
		    depart_time_override = CASE WHEN $8 THEN $9::time ELSE depart_time_override END,
		    duration_min_override = CASE WHEN $10 THEN $11 ELSE duration_min_override END,
		    not_claimed_aa09 = CASE WHEN $12 THEN COALESCE($13, false) ELSE not_claimed_aa09 END,
		    correction_reason = CASE WHEN $14 THEN $15 ELSE correction_reason END,
		    corrected_by = $16,
		    corrected_at = now(),
		    updated_at = now()
		WHERE id = $1
	`
	db := pgxdb.FromContext(ctx, r.db)
	_, err := db.Exec(ctx, query,
		rideID,
		effectiveStatus.Present, effectiveStatus.Value,
		vehicleID.Present, vehicleID.Value,
		driverID.Present, driverID.Value,
		departTimeOverride.Present, departTimeOverride.Value,
		durationMinOverride.Present, durationMinOverride.Value,
		notClaimedAA09.Present, notClaimedAA09.Value,
		reason.Present, reason.Value, operatorID,
	)
	return err
}

// CorrectRideRecordWithFingerprint 以單一 UPDATE 原子保存人工更正與其來源快照。
func (r *RideRepository) CorrectRideRecordWithFingerprint(
	ctx context.Context,
	rideID uuid.UUID,
	effectiveStatus app.PatchValue[string],
	vehicleID app.PatchValue[uuid.UUID],
	driverID app.PatchValue[uuid.UUID],
	departTimeOverride app.PatchValue[string],
	durationMinOverride app.PatchValue[int16],
	notClaimedAA09 app.PatchValue[bool],
	reason app.PatchValue[string],
	operatorID uuid.UUID,
	fingerprint string,
) error {
	query := `
		UPDATE ride_records
		SET effective_status = CASE WHEN $2 THEN $3 ELSE effective_status END,
		    vehicle_id = CASE WHEN $4 THEN $5 ELSE vehicle_id END,
		    driver_id = CASE WHEN $6 THEN $7 ELSE driver_id END,
		    depart_time_override = CASE WHEN $8 THEN $9::time ELSE depart_time_override END,
		    duration_min_override = CASE WHEN $10 THEN $11 ELSE duration_min_override END,
		    not_claimed_aa09 = CASE WHEN $12 THEN COALESCE($13, false) ELSE not_claimed_aa09 END,
		    correction_reason = CASE WHEN $14 THEN $15 ELSE correction_reason END,
		    corrected_by = $16,
		    corrected_at = now(),
		    based_on_fingerprint = $17,
		    updated_at = now()
		WHERE id = $1
	`
	_, err := pgxdb.FromContext(ctx, r.db).Exec(ctx, query,
		rideID,
		effectiveStatus.Present, effectiveStatus.Value,
		vehicleID.Present, vehicleID.Value,
		driverID.Present, driverID.Value,
		departTimeOverride.Present, departTimeOverride.Value,
		durationMinOverride.Present, durationMinOverride.Value,
		notClaimedAA09.Present, notClaimedAA09.Value,
		reason.Present, reason.Value, operatorID, fingerprint,
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

// ListImportedMonths 統計每份匯報表各月份由匯入寫入的提交筆數與最後一次匯入時間。
//
// 只算 source = 'import'：人工補登的提交不應讓使用者以為那個月是匯入來的。
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
		       corrected_by, corrected_at, correction_reason, COALESCE(based_on_fingerprint, ''), created_at, updated_at
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
