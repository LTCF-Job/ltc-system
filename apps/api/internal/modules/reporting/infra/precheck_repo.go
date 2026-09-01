package infra

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"ltc-system/apps/api/internal/modules/reporting/app"
)

// PrecheckRepository 提供申報前置檢核所需之資料查詢。
type PrecheckRepository struct {
	db *pgxpool.Pool
}

// NewPrecheckRepository 建立 PrecheckRepository 實例。
func NewPrecheckRepository(db *pgxpool.Pool) *PrecheckRepository {
	return &PrecheckRepository{db: db}
}

// FindIncompleteActiveCases 查詢缺少必填資料（身分證、地址或服務類型）的有效個案。
func (r *PrecheckRepository) FindIncompleteActiveCases(ctx context.Context, region string) ([]app.IncompleteCase, error) {
	if r.db == nil {
		return []app.IncompleteCase{}, nil
	}

	query := `
		SELECT id, name
		FROM cases
		WHERE ($1 = '' OR region = $1) AND status = 'active'
		  AND (COALESCE(home_address, '') = '' OR service_usage_type IS NULL OR COALESCE(national_id_masked, '') = '')
	`
	rows, err := r.db.Query(ctx, query, region)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []app.IncompleteCase
	for rows.Next() {
		var item app.IncompleteCase
		if err := rows.Scan(&item.ID, &item.Name); err == nil {
			result = append(result, item)
		}
	}
	return result, nil
}

// FindUnresolvedConflicts 查詢指定區域中存在未裁決衝突的搭乘紀錄。
func (r *PrecheckRepository) FindUnresolvedConflicts(ctx context.Context, region string) ([]app.UnresolvedConflict, error) {
	if r.db == nil {
		return []app.UnresolvedConflict{}, nil
	}

	query := `
		SELECT r.id, c.name, r.service_date
		FROM ride_records r
		JOIN cases c ON r.case_id = c.id
		WHERE ($1 = '' OR c.region = $1) AND r.has_conflict = true AND r.conflict_resolved_at IS NULL
	`
	rows, err := r.db.Query(ctx, query, region)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []app.UnresolvedConflict
	for rows.Next() {
		var item app.UnresolvedConflict
		if err := rows.Scan(&item.RideID, &item.CaseName, &item.ServiceDate); err == nil {
			result = append(result, item)
		}
	}
	return result, nil
}
