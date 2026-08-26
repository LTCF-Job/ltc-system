package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// FormColumnEntity 代表 form_columns 實體。
type FormColumnEntity struct {
	ID              uuid.UUID  `json:"id"`
	FormID          uuid.UUID  `json:"formId"`
	ColumnIndex     int        `json:"columnIndex"`
	ColumnHeader    string     `json:"columnHeader"`
	CleanedName     string     `json:"cleanedName"`
	Kind            string     `json:"kind"`
	MappingStatus   string     `json:"mappingStatus"`
	CaseID          *uuid.UUID `json:"caseId,omitempty"`
	LegSeq          *int16     `json:"legSeq,omitempty"`
	SuggestedCaseID *uuid.UUID `json:"suggestedCaseId,omitempty"`
	SuggestionScore float64    `json:"suggestionScore"`
	CreatedAt       time.Time  `json:"createdAt"`
	UpdatedAt       time.Time  `json:"updatedAt"`
}

// RideRecordEntity 代表 ride_records 實體。
type RideRecordEntity struct {
	ID                  uuid.UUID  `json:"id"`
	CaseID              uuid.UUID  `json:"caseId"`
	CaseName            string     `json:"caseName,omitempty"`
	ServiceDate         time.Time  `json:"serviceDate"`
	LegSeq              int16      `json:"legSeq"`
	MergedStatus        string     `json:"mergedStatus"`
	EffectiveStatus     string     `json:"effectiveStatus"`
	VehicleID           uuid.UUID  `json:"vehicleId"`
	VehicleName         string     `json:"vehicleName,omitempty"`
	DriverID            *uuid.UUID `json:"driverId,omitempty"`
	DriverName          string     `json:"driverName,omitempty"`
	HasConflict         bool       `json:"hasConflict"`
	ConflictResolvedAt  *time.Time `json:"conflictResolvedAt,omitempty"`
	ConflictResolvedBy  *uuid.UUID `json:"conflictResolvedBy,omitempty"`
	DepartTimeOverride  *string    `json:"departTimeOverride,omitempty"`
	DurationMinOverride *int16     `json:"durationMinOverride,omitempty"`
	NotClaimedAA09      bool       `json:"notClaimedAa09"`
	CorrectedBy         *uuid.UUID `json:"correctedBy,omitempty"`
	CorrectedAt         *time.Time `json:"correctedAt,omitempty"`
	CorrectionReason    *string    `json:"correctionReason,omitempty"`
	CreatedAt           time.Time  `json:"createdAt"`
	UpdatedAt           time.Time  `json:"updatedAt"`
}

// FormRepository 提供表單、提交紀錄、回報來源與搭乘紀錄之存取。
type FormRepository struct {
	db *pgxpool.Pool
}

// NewFormRepository 建立 FormRepository 實例。
func NewFormRepository(db *pgxpool.Pool) *FormRepository {
	return &FormRepository{db: db}
}

// GetFormBySecret 依 Webhook Token 尋找註冊表單。
func (r *FormRepository) GetFormBySecret(ctx context.Context, secret string) (uuid.UUID, uuid.UUID, error) {
	query := `SELECT id, vehicle_id FROM google_forms WHERE ingest_secret_ref = $1 AND status = 'active' LIMIT 1`
	var formID, vehicleID uuid.UUID
	err := r.db.QueryRow(ctx, query, secret).Scan(&formID, &vehicleID)
	return formID, vehicleID, err
}

// GetFormColumns 取得特定表單之所有欄位定義。
func (r *FormRepository) GetFormColumns(ctx context.Context, formID uuid.UUID) ([]FormColumnEntity, error) {
	query := `
		SELECT id, form_id, column_index, column_header, cleaned_name, kind, mapping_status,
		       case_id, leg_seq, suggested_case_id, suggestion_score, created_at, updated_at
		FROM form_columns
		WHERE form_id = $1
		ORDER BY column_index ASC
	`
	rows, err := r.db.Query(ctx, query, formID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cols []FormColumnEntity
	for rows.Next() {
		var col FormColumnEntity
		if err := rows.Scan(
			&col.ID, &col.FormID, &col.ColumnIndex, &col.ColumnHeader, &col.CleanedName, &col.Kind, &col.MappingStatus,
			&col.CaseID, &col.LegSeq, &col.SuggestedCaseID, &col.SuggestionScore, &col.CreatedAt, &col.UpdatedAt,
		); err != nil {
			return nil, err
		}
		cols = append(cols, col)
	}
	return cols, nil
}

// SaveFormSubmission 先完整寫入原始 payload 與中繼資訊。
func (r *FormRepository) SaveFormSubmission(
	ctx context.Context,
	formID uuid.UUID,
	serviceDate time.Time,
	submittedAt time.Time,
	driverNameRaw string,
	driverID *uuid.UUID,
	source string,
	payload map[string]interface{},
	issueText string,
) (uuid.UUID, error) {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return uuid.Nil, err
	}

	submissionID := uuid.New()
	query := `
		INSERT INTO form_submissions (
			id, form_id, service_date, submitted_at, driver_name_raw, driver_id, source, payload, issue_text
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (form_id, service_date, submitted_at) DO UPDATE
		SET payload = EXCLUDED.payload, issue_text = EXCLUDED.issue_text
		RETURNING id
	`
	err = r.db.QueryRow(ctx, query,
		submissionID, formID, serviceDate, submittedAt, driverNameRaw, driverID, source, payloadBytes, issueText,
	).Scan(&submissionID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to save form submission: %w", err)
	}
	return submissionID, nil
}

// InsertRideSource 寫入單筆來源搭乘回報。
func (r *FormRepository) InsertRideSource(
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
	_, err := r.db.Exec(ctx, query,
		uuid.New(), submissionID, caseID, serviceDate, legSeq, vehicleID, driverID, reported, colIdx,
	)
	return err
}

// GetRideRecordForSlot 查詢指定個案、日期、時段之既有搭乘主紀錄。
func (r *FormRepository) GetRideRecordForSlot(ctx context.Context, caseID uuid.UUID, serviceDate time.Time, legSeq int16) (*RideRecordEntity, error) {
	query := `
		SELECT id, case_id, service_date, leg_seq, merged_status, effective_status,
		       vehicle_id, driver_id, has_conflict, conflict_resolved_at, conflict_resolved_by,
		       to_char(depart_time_override, 'HH24:MI'), duration_min_override, not_claimed_aa09,
		       corrected_by, corrected_at, correction_reason, created_at, updated_at
		FROM ride_records
		WHERE case_id = $1 AND service_date = $2 AND leg_seq = $3
		LIMIT 1
	`
	var rec RideRecordEntity
	err := r.db.QueryRow(ctx, query, caseID, serviceDate, legSeq).Scan(
		&rec.ID, &rec.CaseID, &rec.ServiceDate, &rec.LegSeq, &rec.MergedStatus, &rec.EffectiveStatus,
		&rec.VehicleID, &rec.DriverID, &rec.HasConflict, &rec.ConflictResolvedAt, &rec.ConflictResolvedBy,
		&rec.DepartTimeOverride, &rec.DurationMinOverride, &rec.NotClaimedAA09,
		&rec.CorrectedBy, &rec.CorrectedAt, &rec.CorrectionReason, &rec.CreatedAt, &rec.UpdatedAt,
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
func (r *FormRepository) UpsertRideRecord(ctx context.Context, rec *RideRecordEntity) error {
	query := `
		INSERT INTO ride_records (
			id, case_id, service_date, leg_seq, merged_status, effective_status,
			vehicle_id, driver_id, has_conflict, conflict_resolved_at, conflict_resolved_by,
			depart_time_override, duration_min_override,
			not_claimed_aa09, corrected_by, corrected_at, correction_reason
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17
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
		    updated_at = now()
		RETURNING created_at, updated_at
	`
	if rec.ID == uuid.Nil {
		rec.ID = uuid.New()
	}
	return r.db.QueryRow(ctx, query,
		rec.ID, rec.CaseID, rec.ServiceDate, rec.LegSeq, rec.MergedStatus, rec.EffectiveStatus,
		rec.VehicleID, rec.DriverID, rec.HasConflict, rec.ConflictResolvedAt, rec.ConflictResolvedBy,
		rec.DepartTimeOverride, rec.DurationMinOverride,
		rec.NotClaimedAA09, rec.CorrectedBy, rec.CorrectedAt, rec.CorrectionReason,
	).Scan(&rec.CreatedAt, &rec.UpdatedAt)
}

// CorrectRideRecord 人工更正搭乘紀錄。
func (r *FormRepository) CorrectRideRecord(
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
	_, err := r.db.Exec(ctx, query,
		rideID, effectiveStatus, vehicleID, driverID, departTimeOverride,
		durationMinOverride, notClaimedAA09, reason, operatorID,
	)
	return err
}

// GoogleFormEntity 代表 google_forms 查詢實體。
type GoogleFormEntity struct {
	ID                 uuid.UUID
	VehicleID          uuid.UUID
	VehicleDisplayName string
	FormTitle          string
	Region             string
	LastSyncedAt       *time.Time
	Status             string
}

// FormColumnMappingEntity 代表包含關聯資訊的表單欄位對應實體。
type FormColumnMappingEntity struct {
	ID                string
	FormID            string
	FormTitle         string
	VehicleName       string
	ColumnIndex       int
	ColumnHeader      string
	CleanedName       string
	Kind              string
	MappingStatus     string
	CaseID            *string
	CaseName          *string
	LegSeq            *int16
	SuggestedCaseID   *string
	SuggestedCaseName *string
	SuggestionScore   float64
}

// ListGoogleForms 查詢所有 Google 表單與所屬車輛資訊。
func (r *FormRepository) ListGoogleForms(ctx context.Context) ([]GoogleFormEntity, error) {
	if r.db == nil {
		return nil, nil
	}

	query := `
		SELECT f.id, f.vehicle_id, COALESCE(v.display_name, '未知車輛'), f.form_title, COALESCE(v.region, 'hsinchu'),
		       f.last_synced_at, f.status
		FROM google_forms f
		LEFT JOIN vehicles v ON f.vehicle_id = v.id
		ORDER BY f.created_at ASC
	`
	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var forms []GoogleFormEntity
	for rows.Next() {
		var f GoogleFormEntity
		if err := rows.Scan(&f.ID, &f.VehicleID, &f.VehicleDisplayName, &f.FormTitle, &f.Region, &f.LastSyncedAt, &f.Status); err == nil {
			forms = append(forms, f)
		}
	}
	return forms, nil
}

// ListColumnsWithMapping 查詢表單欄位對應狀態。
func (r *FormRepository) ListColumnsWithMapping(ctx context.Context, mappingStatus string) ([]FormColumnMappingEntity, error) {
	if r.db == nil {
		return nil, nil
	}

	query := `
		SELECT fc.id::text, fc.form_id::text, COALESCE(gf.form_title, ''), COALESCE(v.display_name, ''),
		       fc.column_index, fc.column_header, fc.cleaned_name, fc.kind, fc.mapping_status,
		       fc.case_id::text, COALESCE(c.name, ''), fc.leg_seq,
		       fc.suggested_case_id::text, COALESCE(sc.name, ''), fc.suggestion_score
		FROM form_columns fc
		LEFT JOIN google_forms gf ON fc.form_id = gf.id
		LEFT JOIN vehicles v ON gf.vehicle_id = v.id
		LEFT JOIN cases c ON fc.case_id = c.id
		LEFT JOIN cases sc ON fc.suggested_case_id = sc.id
		WHERE 1=1
	`
	var args []interface{}
	if mappingStatus != "" {
		query += " AND fc.mapping_status = $1"
		args = append(args, mappingStatus)
	}
	query += " ORDER BY fc.form_id, fc.column_index ASC"

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cols []FormColumnMappingEntity
	for rows.Next() {
		var c FormColumnMappingEntity
		var caseID, caseName, suggID, suggName *string
		if err := rows.Scan(
			&c.ID, &c.FormID, &c.FormTitle, &c.VehicleName,
			&c.ColumnIndex, &c.ColumnHeader, &c.CleanedName, &c.Kind, &c.MappingStatus,
			&caseID, &caseName, &c.LegSeq,
			&suggID, &suggName, &c.SuggestionScore,
		); err == nil {
			c.CaseID = caseID
			c.CaseName = caseName
			c.SuggestedCaseID = suggID
			c.SuggestedCaseName = suggName
			cols = append(cols, c)
		}
	}
	return cols, nil
}

// UpdateColumnMappingById 更新指定欄位的對應狀態與個案綁定。
func (r *FormRepository) UpdateColumnMappingById(ctx context.Context, colID string, status string, caseID *string, legSeq *int16) error {
	if r.db == nil {
		return nil
	}

	query := `
		UPDATE form_columns
		SET mapping_status = $2,
		    case_id = $3::uuid,
		    leg_seq = $4,
		    updated_at = now()
		WHERE id = $1::uuid
	`
	_, err := r.db.Exec(ctx, query, colID, status, caseID, legSeq)
	return err
}

