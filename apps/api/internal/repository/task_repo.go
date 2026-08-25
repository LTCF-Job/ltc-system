package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ReportedRideSlot 代表已回報趟次之辨識鍵。
type ReportedRideSlot struct {
	CaseID uuid.UUID
	LegSeq int16
}

// MonthEndRideStats 代表月統計搭乘數據。
type MonthEndRideStats struct {
	TotalRides      int
	BoardedRides    int
	UnreportedRides int
	ConflictCount   int
}

// TaskRepository 提供排程任務與背景檢核所需之資料庫查詢。
type TaskRepository struct {
	db *pgxpool.Pool
}

// NewTaskRepository 建立 TaskRepository 實例。
func NewTaskRepository(db *pgxpool.Pool) *TaskRepository {
	return &TaskRepository{db: db}
}

// GetReportedRideSlots 查詢指定日期已回報且非未回報狀態之趟次清單。
func (r *TaskRepository) GetReportedRideSlots(ctx context.Context, targetDate time.Time) ([]ReportedRideSlot, error) {
	if r.db == nil {
		return []ReportedRideSlot{}, nil
	}

	query := `
		SELECT case_id, leg_seq
		FROM ride_records
		WHERE service_date = $1 AND effective_status != 'unreported'
	`
	rows, err := r.db.Query(ctx, query, targetDate)
	if err != nil {
		return nil, fmt.Errorf("failed to query reported rides: %w", err)
	}
	defer rows.Close()

	var list []ReportedRideSlot
	for rows.Next() {
		var slot ReportedRideSlot
		if err := rows.Scan(&slot.CaseID, &slot.LegSeq); err != nil {
			return nil, fmt.Errorf("failed to scan reported ride slot: %w", err)
		}
		list = append(list, slot)
	}
	return list, nil
}

// GetMonthEndRideStats 統計指定日期區間之搭乘狀態分佈與衝突數量。
func (r *TaskRepository) GetMonthEndRideStats(ctx context.Context, start, end time.Time) (MonthEndRideStats, error) {
	var stats MonthEndRideStats
	if r.db == nil {
		return stats, nil
	}

	query := `
		SELECT 
			COUNT(*) as total,
			COUNT(CASE WHEN effective_status = 'boarded' THEN 1 END) as boarded,
			COUNT(CASE WHEN effective_status = 'unreported' THEN 1 END) as unreported,
			COUNT(CASE WHEN has_conflict = true THEN 1 END) as conflicts
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
