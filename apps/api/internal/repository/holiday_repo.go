package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
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
func (r *HolidayRepository) List(ctx context.Context, startDate, endDate time.Time, region string) ([]HolidayEntity, error) {
	if r.db == nil {
		return []HolidayEntity{}, nil
	}

	query := `
		SELECT holiday_date, name, region, source, created_at
		FROM holidays
		WHERE holiday_date >= $1 AND holiday_date <= $2
		  AND ($3 = '' OR region IS NULL OR region = $3)
		ORDER BY holiday_date ASC
	`
	rows, err := r.db.Query(ctx, query, startDate, endDate, region)
	if err != nil {
		return nil, fmt.Errorf("failed to query holidays: %w", err)
	}
	defer rows.Close()

	var holidays []HolidayEntity
	for rows.Next() {
		var h HolidayEntity
		if err := rows.Scan(&h.HolidayDate, &h.Name, &h.Region, &h.Source, &h.CreatedAt); err != nil {
			return nil, err
		}
		holidays = append(holidays, h)
	}
	return holidays, nil
}

// GetHolidayMap 取得特定月份之假日集合（格式 YYYY-MM-DD -> true），供日曆計算使用。
func (r *HolidayRepository) GetHolidayMap(ctx context.Context, year, month int, region string) (map[string]bool, error) {
	result := make(map[string]bool)
	if r.db == nil {
		return result, nil
	}

	firstDay := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	lastDay := firstDay.AddDate(0, 1, -1)

	holidays, err := r.List(ctx, firstDay, lastDay, region)
	if err != nil {
		return nil, err
	}

	for _, h := range holidays {
		result[h.HolidayDate.Format("2006-01-02")] = true
	}
	return result, nil
}

// Upsert 新增或更新單筆假日。
func (r *HolidayRepository) Upsert(ctx context.Context, h *HolidayEntity) error {
	if r.db == nil {
		return nil
	}

	query := `
		INSERT INTO holidays (holiday_date, name, region, source)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (holiday_date) DO UPDATE
		SET name = EXCLUDED.name, region = EXCLUDED.region, source = EXCLUDED.source
		RETURNING created_at
	`
	return r.db.QueryRow(ctx, query, h.HolidayDate, h.Name, h.Region, h.Source).Scan(&h.CreatedAt)
}

// BatchUpsert 批次匯入國定假日。
func (r *HolidayRepository) BatchUpsert(ctx context.Context, holidays []HolidayEntity) error {
	if r.db == nil || len(holidays) == 0 {
		return nil
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	query := `
		INSERT INTO holidays (holiday_date, name, region, source)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (holiday_date) DO UPDATE
		SET name = EXCLUDED.name, region = EXCLUDED.region, source = EXCLUDED.source
	`

	for _, h := range holidays {
		if _, err := tx.Exec(ctx, query, h.HolidayDate, h.Name, h.Region, h.Source); err != nil {
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
