package infra

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"ltc-system/apps/api/internal/modules/task/app"
)

// TaskRepository 提供排程任務與背景檢核所需之資料庫查詢。
type TaskRepository struct {
	db *pgxpool.Pool
}

// NewTaskRepository 建立 TaskRepository 實例。
func NewTaskRepository(db *pgxpool.Pool) *TaskRepository {
	return &TaskRepository{db: db}
}

// GetReportedRideSlotsInRange 查詢區間內已回報且非未回報狀態之趟次清單。
func (r *TaskRepository) GetReportedRideSlotsInRange(ctx context.Context, start, end time.Time) ([]app.ReportedRideSlotOnDate, error) {
	if r.db == nil {
		return []app.ReportedRideSlotOnDate{}, nil
	}

	query := `
		SELECT case_id, service_date, leg_seq
		FROM ride_records
		WHERE service_date >= $1 AND service_date <= $2 AND effective_status != 'unreported'
	`
	rows, err := r.db.Query(ctx, query, start, end)
	if err != nil {
		return nil, fmt.Errorf("failed to query reported rides: %w", err)
	}
	defer rows.Close()

	var list []app.ReportedRideSlotOnDate
	for rows.Next() {
		var slot app.ReportedRideSlotOnDate
		if err := rows.Scan(&slot.CaseID, &slot.ServiceDate, &slot.LegSeq); err != nil {
			return nil, fmt.Errorf("failed to scan reported ride slot: %w", err)
		}
		list = append(list, slot)
	}
	return list, nil
}

// GetMonthEndRideStats 統計指定日期區間之搭乘狀態分佈與衝突數量。
func (r *TaskRepository) GetMonthEndRideStats(ctx context.Context, start, end time.Time) (app.MonthEndRideStats, error) {
	var stats app.MonthEndRideStats
	if r.db == nil {
		return stats, nil
	}

	query := `
		SELECT 
			COUNT(*) as total,
			COUNT(CASE WHEN effective_status = 'boarded' THEN 1 END) as boarded,
			COUNT(CASE WHEN effective_status = 'unreported' THEN 1 END) as unreported,
			COUNT(CASE WHEN has_conflict = true AND conflict_resolved_at IS NULL THEN 1 END) as conflicts
		FROM ride_records
		WHERE service_date >= $1 AND service_date <= $2
	`
	err := r.db.QueryRow(ctx, query, start, end).Scan(
		&stats.TotalRides, &stats.BoardedRides, &stats.UnreportedRides, &stats.ConflictCount,
	)
	if err != nil {
		return stats, fmt.Errorf("failed to query month end ride stats: %w", err)
	}
	return stats, nil
}
