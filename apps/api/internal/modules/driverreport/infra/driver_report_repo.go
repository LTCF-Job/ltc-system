package infra

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"ltc-system/apps/api/internal/modules/driverreport/app"
	"ltc-system/apps/api/internal/platform/pgxdb"
)

// ErrNoDatabase 讓離線啟動（本機無 DB）時的寫入回傳明確錯誤，
// 而不是回報一個從未發生的成功寫入。
var ErrNoDatabase = errors.New("資料庫未連線，無法寫入司機接送匯報資料")

// DriverReportRepository 提供匯報表與欄位對應之資料存取。
type DriverReportRepository struct {
	db *pgxpool.Pool
}

// NewDriverReportRepository 建立 DriverReportRepository 實例。
func NewDriverReportRepository(db *pgxpool.Pool) *DriverReportRepository {
	return &DriverReportRepository{db: db}
}

const formSelectColumns = `
	SELECT f.id, f.vehicle_id, COALESCE(v.display_name, '未知車輛'), f.title,
	       COALESCE(s.region, 'hsinchu'), f.last_imported_at, f.status,
	       (SELECT count(*) FROM form_columns fc WHERE fc.form_id = f.id),
	       (SELECT count(*) FROM form_columns fc WHERE fc.form_id = f.id AND fc.mapping_status = 'mapped'),
	       (SELECT count(*) FROM form_columns fc WHERE fc.form_id = f.id AND fc.mapping_status = 'pending'),
	       (SELECT count(*) FROM form_submissions fs WHERE fs.form_id = f.id)
	FROM driver_report_forms f
	LEFT JOIN vehicles v ON f.vehicle_id = v.id
	LEFT JOIN sites s ON v.site_id = s.id
`

// ListForms 查詢所有匯報表與其對應進度。
func (r *DriverReportRepository) ListForms(ctx context.Context) ([]app.ReportForm, error) {
	if r.db == nil {
		return nil, nil
	}

	db := pgxdb.FromContext(ctx, r.db)
	rows, err := db.Query(ctx, formSelectColumns+" ORDER BY f.created_at ASC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var forms []app.ReportForm
	for rows.Next() {
		f, err := scanReportForm(rows)
		if err != nil {
			return nil, err
		}
		forms = append(forms, f)
	}
	return forms, rows.Err()
}

// GetForm 依 ID 查詢單一匯報表；查無資料時回傳 nil 而非錯誤。
func (r *DriverReportRepository) GetForm(ctx context.Context, formID uuid.UUID) (*app.ReportForm, error) {
	if r.db == nil {
		return nil, nil
	}

	db := pgxdb.FromContext(ctx, r.db)
	rows, err := db.Query(ctx, formSelectColumns+" WHERE f.id = $1", formID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if !rows.Next() {
		return nil, rows.Err()
	}
	f, err := scanReportForm(rows)
	if err != nil {
		return nil, err
	}
	return &f, nil
}

func scanReportForm(rows pgx.Rows) (app.ReportForm, error) {
	var f app.ReportForm
	err := rows.Scan(
		&f.ID, &f.VehicleID, &f.VehicleDisplayName, &f.Title,
		&f.Region, &f.LastImportedAt, &f.Status,
		&f.TotalColumns, &f.MappedColumns, &f.PendingColumns, &f.SubmissionCount,
	)
	return f, err
}

// CreateForm 建立一台車的匯報表並回傳實際的匯報表 ID；同車重複建立時只更新名稱，
// 回傳的是既有那一份的 ID 而不是傳入的 id。
func (r *DriverReportRepository) CreateForm(ctx context.Context, id, vehicleID uuid.UUID, title string) (uuid.UUID, error) {
	if r.db == nil {
		return uuid.Nil, ErrNoDatabase
	}

	query := `
		INSERT INTO driver_report_forms (id, vehicle_id, title, status, created_at, updated_at)
		VALUES ($1, $2, $3, 'active', now(), now())
		ON CONFLICT (vehicle_id) DO UPDATE
		SET title = EXCLUDED.title,
		    updated_at = now()
		RETURNING id
	`
	var formID uuid.UUID
	if err := pgxdb.FromContext(ctx, r.db).QueryRow(ctx, query, id, vehicleID, title).Scan(&formID); err != nil {
		return uuid.Nil, err
	}
	return formID, nil
}

// DeleteForm 刪除匯報表。
func (r *DriverReportRepository) DeleteForm(ctx context.Context, formID uuid.UUID) error {
	if r.db == nil {
		return ErrNoDatabase
	}

	_, err := pgxdb.FromContext(ctx, r.db).Exec(ctx, `DELETE FROM driver_report_forms WHERE id = $1`, formID)
	return err
}

// ListColumnsWithMapping 查詢欄位對應狀態，可依匯報表與對應狀態篩選。
func (r *DriverReportRepository) ListColumnsWithMapping(ctx context.Context, formID, mappingStatus string) ([]app.ColumnMapping, error) {
	if r.db == nil {
		return nil, nil
	}

	query := `
		SELECT fc.id::text, fc.form_id::text, COALESCE(df.title, ''), COALESCE(v.display_name, ''),
		       fc.column_index, fc.column_header, fc.cleaned_name, fc.kind, fc.mapping_status,
		       fc.case_id::text, COALESCE(c.name, ''), fc.leg_seq,
		       fc.suggested_case_id::text, COALESCE(sc.name, ''), fc.suggestion_score
		FROM form_columns fc
		LEFT JOIN driver_report_forms df ON fc.form_id = df.id
		LEFT JOIN vehicles v ON df.vehicle_id = v.id
		LEFT JOIN cases c ON fc.case_id = c.id
		LEFT JOIN cases sc ON fc.suggested_case_id = sc.id
		WHERE ($1 = '' OR fc.form_id = $1::uuid)
		  AND ($2 = '' OR fc.mapping_status = $2)
		ORDER BY fc.form_id, fc.column_index ASC
	`
	db := pgxdb.FromContext(ctx, r.db)
	rows, err := db.Query(ctx, query, formID, mappingStatus)
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
		); err != nil {
			return nil, err
		}
		c.CaseID = caseID
		c.CaseName = caseName
		c.SuggestedCaseID = suggID
		c.SuggestedCaseName = suggName
		cols = append(cols, c)
	}
	return cols, rows.Err()
}

// UpsertColumns 登記檔案中出現的欄位。以表頭文字為衝突鍵，個案增減造成的欄號位移
// 不會覆蓋到別的個案；已對應過的欄位只更新欄號與推薦值，保留人工確認的對應結果。
func (r *DriverReportRepository) UpsertColumns(ctx context.Context, formID uuid.UUID, drafts []app.ColumnDraft) error {
	if r.db == nil {
		return ErrNoDatabase
	}
	if len(drafts) == 0 {
		return nil
	}

	batch := &pgx.Batch{}
	query := `
		INSERT INTO form_columns (id, form_id, column_index, column_header, cleaned_name, kind,
		                          mapping_status, suggested_case_id, suggestion_score, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, 'pending', $7::uuid, $8, now(), now())
		ON CONFLICT (form_id, column_header) DO UPDATE
		SET column_index = EXCLUDED.column_index,
		    cleaned_name = EXCLUDED.cleaned_name,
		    kind = EXCLUDED.kind,
		    suggested_case_id = COALESCE(form_columns.suggested_case_id, EXCLUDED.suggested_case_id),
		    suggestion_score = GREATEST(form_columns.suggestion_score, EXCLUDED.suggestion_score),
		    updated_at = now()
	`
	for _, d := range drafts {
		batch.Queue(query, uuid.New(), formID, d.ColumnIndex, d.ColumnHeader, d.CleanedName, d.Kind, d.SuggestedCaseID, d.SuggestionScore)
	}

	results := pgxdb.FromContext(ctx, r.db).SendBatch(ctx, batch)
	defer results.Close()
	for range drafts {
		if _, err := results.Exec(); err != nil {
			return err
		}
	}
	return nil
}

// UpdateColumnMappingByID 更新指定欄位的對應狀態與個案綁定。
func (r *DriverReportRepository) UpdateColumnMappingByID(ctx context.Context, colID, status string, caseID *string, legSeq *int16) error {
	if r.db == nil {
		return ErrNoDatabase
	}

	query := `
		UPDATE form_columns
		SET mapping_status = $2,
		    case_id = $3::uuid,
		    leg_seq = $4,
		    updated_at = now()
		WHERE id = $1::uuid
	`
	_, err := pgxdb.FromContext(ctx, r.db).Exec(ctx, query, colID, status, caseID, legSeq)
	return err
}

// UpdateColumnMappingByHeader 以表頭文字定位欄位並更新對應，供匯入預覽的就地確認使用。
func (r *DriverReportRepository) UpdateColumnMappingByHeader(ctx context.Context, formID uuid.UUID, header, status string, caseID *string, legSeq *int16) error {
	if r.db == nil {
		return ErrNoDatabase
	}

	query := `
		UPDATE form_columns
		SET mapping_status = $3,
		    case_id = $4::uuid,
		    leg_seq = $5,
		    updated_at = now()
		WHERE form_id = $1 AND column_header = $2
	`
	_, err := pgxdb.FromContext(ctx, r.db).Exec(ctx, query, formID, header, status, caseID, legSeq)
	return err
}

// MarkImported 記錄最後一次成功匯入的時間。
func (r *DriverReportRepository) MarkImported(ctx context.Context, formID uuid.UUID, importedAt time.Time) error {
	if r.db == nil {
		return ErrNoDatabase
	}

	_, err := pgxdb.FromContext(ctx, r.db).Exec(ctx, `UPDATE driver_report_forms SET last_imported_at = $2, updated_at = now() WHERE id = $1`, formID, importedAt)
	return err
}
