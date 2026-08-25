package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// IncompleteCaseEntity 代表資料不完整之個案。
type IncompleteCaseEntity struct {
	ID   uuid.UUID
	Name string
}

// UnresolvedConflictEntity 代表未裁決衝突之搭乘紀錄。
type UnresolvedConflictEntity struct {
	RideID      uuid.UUID
	CaseName    string
	ServiceDate time.Time
}

// PrecheckRepository 提供申報前置檢核所需之資料查詢。
type PrecheckRepository struct {
	db *pgxpool.Pool
}

// NewPrecheckRepository 建立 PrecheckRepository 實例。
func NewPrecheckRepository(db *pgxpool.Pool) *PrecheckRepository {
	return &PrecheckRepository{db: db}
}

// FindIncompleteActiveCases 查詢缺少必填資料（身分證、地址或服務類型）的有效個案。
func (r *PrecheckRepository) FindIncompleteActiveCases(ctx context.Context, region string) ([]IncompleteCaseEntity, error) {
	if r.db == nil {
		return []IncompleteCaseEntity{}, nil
	}

	query := `
		SELECT id, name
		FROM cases
		WHERE region = $1 AND status = 'active'
		  AND (home_address = '' OR service_usage_type IS NULL OR national_id_masked = '')
	`
	rows, err := r.db.Query(ctx, query, region)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []IncompleteCaseEntity
	for rows.Next() {
		var item IncompleteCaseEntity
		if err := rows.Scan(&item.ID, &item.Name); err == nil {
			result = append(result, item)
		}
	}
	return result, nil
}

// FindUnresolvedConflicts 查詢指定區域中存在未裁決衝突的搭乘紀錄。
func (r *PrecheckRepository) FindUnresolvedConflicts(ctx context.Context, region string) ([]UnresolvedConflictEntity, error) {
	if r.db == nil {
		return []UnresolvedConflictEntity{}, nil
	}

	query := `
		SELECT r.id, c.name, r.service_date
		FROM ride_records r
		JOIN cases c ON r.case_id = c.id
		WHERE c.region = $1 AND r.has_conflict = true AND r.conflict_resolved_at IS NULL
	`
	rows, err := r.db.Query(ctx, query, region)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []UnresolvedConflictEntity
	for rows.Next() {
		var item UnresolvedConflictEntity
		if err := rows.Scan(&item.RideID, &item.CaseName, &item.ServiceDate); err == nil {
			result = append(result, item)
		}
	}
	return result, nil
}
