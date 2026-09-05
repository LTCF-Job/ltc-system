package infra

import (
	"context"
	"errors"
	"fmt"

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

// FindIncompleteActiveCases 查詢缺少必填資料（身分證、地址、服務類別或服務使用類型）的有效個案。
func (r *PrecheckRepository) FindIncompleteActiveCases(ctx context.Context, scope app.ClaimScope) ([]app.IncompleteCase, error) {
	if r.db == nil {
		return nil, errors.New("precheck database is not configured")
	}

	query := `
		SELECT c.id, c.name
		FROM cases c
		WHERE c.status = 'active'
		  AND EXISTS (
			SELECT 1 FROM ride_records r
			WHERE r.case_id = c.id
			  AND r.service_date >= $1::date
			  AND r.service_date < $2::date
		  )
		  AND ($3::text IS NULL OR c.region = $3)
		  AND (COALESCE(cardinality($4::uuid[]), 0) = 0 OR c.id = ANY($4::uuid[]))
		  AND (COALESCE(c.home_address, '') = '' OR c.service_category IS NULL OR c.service_usage_type IS NULL OR COALESCE(c.national_id_masked, '') = '')
	`
	rows, err := r.db.Query(ctx, query, scope.StartDate, scope.EndDate, scope.Region, scope.CaseIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []app.IncompleteCase
	for rows.Next() {
		var item app.IncompleteCase
		if err := rows.Scan(&item.ID, &item.Name); err != nil {
			return nil, fmt.Errorf("scan incomplete case: %w", err)
		}
		result = append(result, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate incomplete cases: %w", err)
	}
	return result, nil
}

// FindUnresolvedConflicts 查詢指定區域中存在未裁決衝突的搭乘紀錄。
func (r *PrecheckRepository) FindUnresolvedConflicts(ctx context.Context, scope app.ClaimScope) ([]app.UnresolvedConflict, error) {
	if r.db == nil {
		return nil, errors.New("precheck database is not configured")
	}

	query := `
		SELECT r.id, c.name, r.service_date
		FROM ride_records r
		JOIN cases c ON r.case_id = c.id
		WHERE r.service_date >= $1::date AND r.service_date < $2::date
		  AND ($3::text IS NULL OR c.region = $3)
		  AND (COALESCE(cardinality($4::uuid[]), 0) = 0 OR c.id = ANY($4::uuid[]))
		  AND r.has_conflict = true AND r.conflict_resolved_at IS NULL
	`
	rows, err := r.db.Query(ctx, query, scope.StartDate, scope.EndDate, scope.Region, scope.CaseIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []app.UnresolvedConflict
	for rows.Next() {
		var item app.UnresolvedConflict
		if err := rows.Scan(&item.RideID, &item.CaseName, &item.ServiceDate); err != nil {
			return nil, fmt.Errorf("scan unresolved conflict: %w", err)
		}
		result = append(result, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate unresolved conflicts: %w", err)
	}
	return result, nil
}
