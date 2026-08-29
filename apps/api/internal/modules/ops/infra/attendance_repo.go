package infra

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"ltc-system/apps/api/internal/modules/ops/app"
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
		SELECT a.id, a.driver_id, d.name, a.record_date, a.status, a.note, a.created_at, a.updated_at
		FROM attendance_records a
		JOIN drivers d ON d.id = a.driver_id
		%s
		ORDER BY a.record_date ASC, d.name ASC
	`, whereClause)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query attendance records: %w", err)
	}
	defer rows.Close()

	list := []app.AttendanceRecord{}
	for rows.Next() {
		var item app.AttendanceRecord
		if err := rows.Scan(
			&item.ID, &item.DriverID, &item.DriverName, &item.RecordDate,
			&item.Status, &item.Note, &item.CreatedAt, &item.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan attendance record: %w", err)
		}
		list = append(list, item)
	}

	return list, nil
}

// Upsert 寫入或更新司機單日出勤紀錄（依 driver_id 與 record_date 唯一鍵）。
func (r *AttendanceRepository) Upsert(ctx context.Context, driverID uuid.UUID, recordDate time.Time, status string, note *string) (*app.AttendanceRecord, error) {
	if r.db == nil {
		return &app.AttendanceRecord{
			ID:         uuid.New(),
			DriverID:   driverID,
			RecordDate: recordDate,
			Status:     status,
			Note:       note,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}, nil
	}

	query := `
		INSERT INTO attendance_records (driver_id, record_date, status, note, created_at, updated_at)
		VALUES ($1, $2, $3, $4, now(), now())
		ON CONFLICT (driver_id, record_date)
		DO UPDATE SET status = EXCLUDED.status, note = EXCLUDED.note, updated_at = now()
		RETURNING id, created_at, updated_at
	`
	var item app.AttendanceRecord
	item.DriverID = driverID
	item.RecordDate = recordDate
	item.Status = status
	item.Note = note

	if err := r.db.QueryRow(ctx, query, driverID, recordDate, status, note).Scan(&item.ID, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return nil, fmt.Errorf("failed to upsert attendance record: %w", err)
	}

	return &item, nil
}
