package infra

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"ltc-system/apps/api/internal/modules/holiday/app"
	"ltc-system/apps/api/internal/platform/pgxdb"
)

// HolidayRepository 提供 holidays 資料表之存取操作。
type HolidayRepository struct {
	db *pgxpool.Pool
}

// NewHolidayRepository 建立 HolidayRepository 實例。
func NewHolidayRepository(db *pgxpool.Pool) *HolidayRepository {
	return &HolidayRepository{db: db}
}

// List 依據日期區間與地區取得國定假日清單。
func (r *HolidayRepository) List(ctx context.Context, startDate, endDate time.Time) ([]app.Holiday, error) {
	if r.db == nil {
		return []app.Holiday{}, nil
	}

	query := `
		SELECT holiday_date, name, source, is_day_off, created_at
		FROM holidays
		WHERE holiday_date >= $1 AND holiday_date <= $2
		ORDER BY holiday_date ASC
	`
	rows, err := r.db.Query(ctx, query, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to query holidays: %w", err)
	}
	defer rows.Close()

	var holidays []app.Holiday
	for rows.Next() {
		var h app.Holiday
		if err := rows.Scan(&h.HolidayDate, &h.Name, &h.Source, &h.IsDayOff, &h.CreatedAt); err != nil {
			return nil, err
		}
		holidays = append(holidays, h)
	}
	return holidays, nil
}

// GetByDate 取得單一日期的假日，供異動稽核保存 before snapshot。
func (r *HolidayRepository) GetByDate(ctx context.Context, date time.Time) (*app.Holiday, error) {
	if r.db == nil {
		return nil, fmt.Errorf("holiday database is not configured")
	}
	var h app.Holiday
	err := pgxdb.FromContext(ctx, r.db).QueryRow(ctx, `
		SELECT holiday_date, name, source, is_day_off, created_at
		FROM holidays
		WHERE holiday_date = $1
	`, date).Scan(&h.HolidayDate, &h.Name, &h.Source, &h.IsDayOff, &h.CreatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get holiday %s: %w", date.Format("2006-01-02"), err)
	}
	return &h, nil
}

// GetHolidayMap 取得特定月份之假日集合（格式 YYYY-MM-DD -> true），供日曆計算使用。
func (r *HolidayRepository) GetHolidayMap(ctx context.Context, year, month int) (map[string]bool, error) {
	result := make(map[string]bool)
	if r.db == nil {
		return result, nil
	}

	firstDay := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	lastDay := firstDay.AddDate(0, 1, -1)

	holidays, err := r.List(ctx, firstDay, lastDay)
	if err != nil {
		return nil, err
	}

	for _, h := range holidays {
		if h.IsDayOff {
			result[h.HolidayDate.Format("2006-01-02")] = true
		}
	}
	return result, nil
}

// Upsert 新增或更新單筆假日。
func (r *HolidayRepository) Upsert(ctx context.Context, h *app.Holiday) error {
	if r.db == nil {
		return nil
	}

	query := `
		INSERT INTO holidays (holiday_date, name, source, is_day_off)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (holiday_date) DO UPDATE
		SET name = EXCLUDED.name, source = EXCLUDED.source, is_day_off = EXCLUDED.is_day_off
		RETURNING created_at
	`
	return r.db.QueryRow(ctx, query, h.HolidayDate, h.Name, h.Source, h.IsDayOff).Scan(&h.CreatedAt)
}

// BatchUpsert 批次匯入國定假日。
func (r *HolidayRepository) BatchUpsert(ctx context.Context, holidays []app.Holiday) error {
	if r.db == nil || len(holidays) == 0 {
		return nil
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	query := `
		INSERT INTO holidays (holiday_date, name, source, is_day_off)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (holiday_date) DO UPDATE
		SET name = EXCLUDED.name, source = EXCLUDED.source, is_day_off = EXCLUDED.is_day_off
		WHERE holidays.source <> 'manual'
	`

	for _, h := range holidays {
		if _, err := tx.Exec(ctx, query, h.HolidayDate, h.Name, h.Source, h.IsDayOff); err != nil {
			return fmt.Errorf("failed to upsert holiday %s: %w", h.HolidayDate.Format("2006-01-02"), err)
		}
	}

	return tx.Commit(ctx)
}

// Delete 刪除特定日期之假日紀錄。
func (r *HolidayRepository) Delete(ctx context.Context, date time.Time) error {
	if r.db == nil {
		return nil
	}

	query := `DELETE FROM holidays WHERE holiday_date = $1`
	_, err := r.db.Exec(ctx, query, date)
	return err
}
