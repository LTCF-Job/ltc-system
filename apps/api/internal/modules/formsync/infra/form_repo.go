package infra

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"ltc-system/apps/api/internal/modules/formsync/app"
)

// FormRepository 提供 Google 表單登錄與欄位對應之資料存取。
type FormRepository struct {
	db *pgxpool.Pool
}

// NewFormRepository 建立 FormRepository 實例。
func NewFormRepository(db *pgxpool.Pool) *FormRepository {
	return &FormRepository{db: db}
}

// ListGoogleForms 查詢所有 Google 表單與所屬車輛資訊。
func (r *FormRepository) ListGoogleForms(ctx context.Context) ([]app.GoogleForm, error) {
	if r.db == nil {
		return nil, nil
	}

	query := `
		SELECT f.id, f.vehicle_id, f.sheet_id, COALESCE(v.display_name, '未知車輛'), f.form_title, COALESCE(v.region, 'hsinchu'),
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

	var forms []app.GoogleForm
	for rows.Next() {
		var f app.GoogleForm
		if err := rows.Scan(&f.ID, &f.VehicleID, &f.SheetID, &f.VehicleDisplayName, &f.FormTitle, &f.Region, &f.LastSyncedAt, &f.Status); err == nil {
			forms = append(forms, f)
		}
	}
	return forms, nil
}

// ListColumnsWithMapping 查詢表單欄位對應狀態。
func (r *FormRepository) ListColumnsWithMapping(ctx context.Context, mappingStatus string) ([]app.ColumnMapping, error) {
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

	var cols []app.ColumnMapping
	for rows.Next() {
		var c app.ColumnMapping
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

// CreateGoogleForm 於資料庫中建立 Google 表單關聯紀錄。
func (r *FormRepository) CreateGoogleForm(ctx context.Context, id, vehicleID uuid.UUID, title, sheetID, secretRef string) error {
	if r.db == nil {
		return nil
	}

	query := `
		INSERT INTO google_forms (id, vehicle_id, title, sheet_id, ingest_secret_ref, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, 'active', now(), now())
		ON CONFLICT (sheet_id) DO UPDATE
		SET title = EXCLUDED.title,
		    vehicle_id = EXCLUDED.vehicle_id,
		    updated_at = now()
	`
	_, err := r.db.Exec(ctx, query, id, vehicleID, title, sheetID, secretRef)
	return err
}

// DeleteGoogleForm 從資料庫中刪除 Google 表單關聯。
func (r *FormRepository) DeleteGoogleForm(ctx context.Context, formID uuid.UUID) error {
	if r.db == nil {
		return nil
	}

	query := `DELETE FROM google_forms WHERE id = $1`
	_, err := r.db.Exec(ctx, query, formID)
	return err
}

// SaveFormColumns 將試算表解析出的標題欄位儲存至 form_columns。
func (r *FormRepository) SaveFormColumns(ctx context.Context, formID uuid.UUID, headers []string) error {
	if r.db == nil || len(headers) == 0 {
		return nil
	}

	for idx, h := range headers {
		kind := "unknown"
		cleaned := strings.TrimSpace(h)
		if strings.Contains(h, "時間戳記") || strings.Contains(h, "今天日期") || strings.Contains(h, "駕駛") {
			kind = "meta"
		} else if strings.Contains(h, "問題") || strings.Contains(h, "回報") {
			kind = "issue"
		} else if strings.Contains(h, "去") || strings.Contains(h, "回") || strings.Contains(h, "車") {
			kind = "ride"
		}

		query := `
			INSERT INTO form_columns (id, form_id, column_index, column_header, cleaned_name, kind, mapping_status, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, 'pending', now(), now())
			ON CONFLICT (form_id, column_index) DO UPDATE
			SET column_header = EXCLUDED.column_header,
			    cleaned_name = EXCLUDED.cleaned_name,
			    kind = EXCLUDED.kind,
			    updated_at = now()
		`
		_, _ = r.db.Exec(ctx, query, uuid.New(), formID, idx+1, h, cleaned, kind)
	}
	return nil
}
