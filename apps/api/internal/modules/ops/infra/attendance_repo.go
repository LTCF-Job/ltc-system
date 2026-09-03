package infra

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"ltc-system/apps/api/internal/modules/ops/app"
	"ltc-system/apps/api/internal/platform/pgxdb"
)

// AttendanceRepository 提供司機出勤與請假紀錄之資料存取。
type AttendanceRepository struct {
	db *pgxpool.Pool
}

// NewAttendanceRepository 建立 AttendanceRepository 實例。
func NewAttendanceRepository(db *pgxpool.Pool) *AttendanceRepository {
	return &AttendanceRepository{db: db}
}

// GetMonthRecords 查詢特定月份內所有司機或指定司機之出勤紀錄。
func (r *AttendanceRepository) GetMonthRecords(ctx context.Context, startDate, endDate time.Time, driverID *uuid.UUID) ([]app.AttendanceRecord, error) {
	if r.db == nil {
		return []app.AttendanceRecord{}, nil
	}

	whereClause := "WHERE a.record_date >= $1 AND a.record_date < $2"
	args := []interface{}{startDate, endDate}
	argIdx := 3

	if driverID != nil {
		whereClause += fmt.Sprintf(" AND a.driver_id = $%d", argIdx)
		args = append(args, *driverID)
	}

	query := fmt.Sprintf(`
		SELECT a.id, a.driver_id, d.name, a.record_date, a.status, a.note, a.source, a.created_at, a.updated_at
		FROM attendance_records a
		JOIN drivers d ON d.id = a.driver_id
		%s
		ORDER BY a.record_date ASC, d.name ASC
	`, whereClause)

	db := pgxdb.FromContext(ctx, r.db)
	rows, err := db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query attendance records: %w", err)
	}
	defer rows.Close()

	list := []app.AttendanceRecord{}
	for rows.Next() {
		var item app.AttendanceRecord
		if err := rows.Scan(
			&item.ID, &item.DriverID, &item.DriverName, &item.RecordDate,
			&item.Status, &item.Note, &item.Source, &item.CreatedAt, &item.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan attendance record: %w", err)
		}
		list = append(list, item)
	}

	return list, nil
}

// GetOne 查詢單一司機單日的出勤紀錄；不存在時回傳 nil, nil。
func (r *AttendanceRepository) GetOne(ctx context.Context, driverID uuid.UUID, recordDate time.Time) (*app.AttendanceRecord, error) {
	if r.db == nil {
		return nil, nil
	}

	db := pgxdb.FromContext(ctx, r.db)
	var item app.AttendanceRecord
	item.DriverID = driverID
	err := db.QueryRow(ctx, `
		SELECT id, record_date, status, note, source, created_at, updated_at
		FROM attendance_records
		WHERE driver_id = $1 AND record_date = $2
	`, driverID, recordDate).Scan(&item.ID, &item.RecordDate, &item.Status, &item.Note, &item.Source, &item.CreatedAt, &item.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get attendance record: %w", err)
	}
	return &item, nil
}

// Upsert 寫入或更新司機單日出勤紀錄（依 driver_id 與 record_date 唯一鍵）。
func (r *AttendanceRepository) Upsert(ctx context.Context, driverID uuid.UUID, recordDate time.Time, status string, note *string, source string) (*app.AttendanceRecord, error) {
	if r.db == nil {
		return &app.AttendanceRecord{
			ID:         uuid.New(),
			DriverID:   driverID,
			RecordDate: recordDate,
			Status:     status,
			Note:       note,
			Source:     source,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}, nil
	}

	query := `
		INSERT INTO attendance_records (driver_id, record_date, status, note, source, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, now(), now())
		ON CONFLICT (driver_id, record_date)
		DO UPDATE SET status = EXCLUDED.status, note = EXCLUDED.note, source = EXCLUDED.source, updated_at = now()
		RETURNING id, created_at, updated_at
	`
	var item app.AttendanceRecord
	item.DriverID = driverID
	item.RecordDate = recordDate
	item.Status = status
	item.Note = note
	item.Source = source

	db := pgxdb.FromContext(ctx, r.db)
	if err := db.QueryRow(ctx, query, driverID, recordDate, status, note, source).Scan(&item.ID, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return nil, fmt.Errorf("failed to upsert attendance record: %w", err)
	}

	return &item, nil
}

// UpsertConflict 記錄一筆匯入與人工登記不一致的待維護衝突。
//
// 已解決（resolved）且人工狀態跟上次解決時相同時維持已解決，不因為重複匯入同一批
// 資料就反覆打擾使用者；除此之外（尚未處理，或人工狀態在上次解決後又變了）一律
// 視為新的待處理，重新開啟給使用者確認。
func (r *AttendanceRepository) UpsertConflict(ctx context.Context, driverID uuid.UUID, recordDate time.Time, existingStatus, importedStatus string) error {
	if r.db == nil {
		return nil
	}

	db := pgxdb.FromContext(ctx, r.db)
	_, err := db.Exec(ctx, `
		INSERT INTO attendance_import_conflicts (driver_id, record_date, existing_status, imported_status, status)
		VALUES ($1, $2, $3, $4, 'pending')
		ON CONFLICT (driver_id, record_date) DO UPDATE SET
			existing_status = EXCLUDED.existing_status,
			imported_status = EXCLUDED.imported_status,
			status = CASE
				WHEN attendance_import_conflicts.status = 'resolved'
				     AND attendance_import_conflicts.resolved_choice = 'keep_manual'
				     AND attendance_import_conflicts.existing_status = EXCLUDED.existing_status
				THEN 'resolved'
				ELSE 'pending'
			END,
			resolved_choice = CASE
				WHEN attendance_import_conflicts.status = 'resolved'
				     AND attendance_import_conflicts.resolved_choice = 'keep_manual'
				     AND attendance_import_conflicts.existing_status = EXCLUDED.existing_status
				THEN attendance_import_conflicts.resolved_choice
				ELSE NULL
			END,
			resolved_by = CASE
				WHEN attendance_import_conflicts.status = 'resolved'
				     AND attendance_import_conflicts.resolved_choice = 'keep_manual'
				     AND attendance_import_conflicts.existing_status = EXCLUDED.existing_status
				THEN attendance_import_conflicts.resolved_by
				ELSE NULL
			END,
			resolved_at = CASE
				WHEN attendance_import_conflicts.status = 'resolved'
				     AND attendance_import_conflicts.resolved_choice = 'keep_manual'
				     AND attendance_import_conflicts.existing_status = EXCLUDED.existing_status
				THEN attendance_import_conflicts.resolved_at
				ELSE NULL
			END,
			updated_at = now()
	`, driverID, recordDate, existingStatus, importedStatus)
	if err != nil {
		return fmt.Errorf("failed to upsert attendance import conflict: %w", err)
	}
	return nil
}

// ListConflicts 查詢待維護衝突清單，status 為空字串時不篩選。
func (r *AttendanceRepository) ListConflicts(ctx context.Context, status string) ([]app.AttendanceImportConflict, error) {
	if r.db == nil {
		return []app.AttendanceImportConflict{}, nil
	}

	whereClause := ""
	args := []interface{}{}
	if status != "" {
		whereClause = "WHERE c.status = $1"
		args = append(args, status)
	}

	query := fmt.Sprintf(`
		SELECT c.id, c.driver_id, d.name, c.record_date, c.existing_status, c.imported_status,
		       c.status, c.resolved_choice, c.created_at, c.updated_at
		FROM attendance_import_conflicts c
		JOIN drivers d ON d.id = c.driver_id
		%s
		ORDER BY c.record_date DESC, d.name ASC
	`, whereClause)

	db := pgxdb.FromContext(ctx, r.db)
	rows, err := db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query attendance import conflicts: %w", err)
	}
	defer rows.Close()

	list := []app.AttendanceImportConflict{}
	for rows.Next() {
		var item app.AttendanceImportConflict
		if err := rows.Scan(
			&item.ID, &item.DriverID, &item.DriverName, &item.RecordDate, &item.ExistingStatus,
			&item.ImportedStatus, &item.Status, &item.ResolvedChoice, &item.CreatedAt, &item.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan attendance import conflict: %w", err)
		}
		list = append(list, item)
	}
	return list, nil
}

// GetConflict 依 ID 查詢單一待維護衝突；不存在時回傳 nil, nil。
func (r *AttendanceRepository) GetConflict(ctx context.Context, id uuid.UUID) (*app.AttendanceImportConflict, error) {
	if r.db == nil {
		return nil, nil
	}

	db := pgxdb.FromContext(ctx, r.db)
	var item app.AttendanceImportConflict
	item.ID = id
	err := db.QueryRow(ctx, `
		SELECT c.driver_id, d.name, c.record_date, c.existing_status, c.imported_status,
		       c.status, c.resolved_choice, c.created_at, c.updated_at
		FROM attendance_import_conflicts c
		JOIN drivers d ON d.id = c.driver_id
		WHERE c.id = $1
	`, id).Scan(
		&item.DriverID, &item.DriverName, &item.RecordDate, &item.ExistingStatus,
		&item.ImportedStatus, &item.Status, &item.ResolvedChoice, &item.CreatedAt, &item.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get attendance import conflict: %w", err)
	}
	return &item, nil
}

// ResolveConflict 把一筆待維護衝突標記為已解決。
func (r *AttendanceRepository) ResolveConflict(ctx context.Context, id uuid.UUID, choice string, actorID *uuid.UUID) error {
	if r.db == nil {
		return nil
	}

	db := pgxdb.FromContext(ctx, r.db)
	tag, err := db.Exec(ctx, `
		UPDATE attendance_import_conflicts
		SET status = 'resolved', resolved_choice = $2, resolved_by = $3, resolved_at = now(), updated_at = now()
		WHERE id = $1
	`, id, choice, actorID)
	if err != nil {
		return fmt.Errorf("failed to resolve attendance import conflict: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return app.ErrAttendanceConflictNotFound
	}
	return nil
}
