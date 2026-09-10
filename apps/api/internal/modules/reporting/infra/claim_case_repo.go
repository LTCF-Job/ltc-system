package infra

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"ltc-system/apps/api/internal/modules/reporting/app"
	"ltc-system/apps/api/internal/platform/pgxdb"
)

// ClaimCaseRepository 把據點或區域換算成申報對象個案。
type ClaimCaseRepository struct {
	db *pgxpool.Pool
}

// NewClaimCaseRepository 建立 ClaimCaseRepository 實例。
func NewClaimCaseRepository(db *pgxpool.Pool) *ClaimCaseRepository {
	return &ClaimCaseRepository{db: db}
}

// claimCaseSelect 是兩支解析查詢共用的投影與過濾條件。
//
// 兩個條件是批次模式獨有的，不能省：
//   - deleted_at IS NULL：govClaimSourceQuery 沒有這個條件，因為逐案勾選模式是靠前端
//     傳來的 caseIds 白名單擋住軟刪除個案。批次模式沒有白名單，不自己擋就會把軟刪除
//     個案一起報出去。
//   - NOT ps.is_pending：待維護全站隔離（見 docs/decisions/pending-data-visibility.md），
//     與 govClaimSourceQuery 一致。
const claimCaseSelect = `
	SELECT c.id, c.name, c.site_id, COALESCE(st.name, ''), COALESCE(st.region, '')
	FROM cases c
	JOIN case_pending_status ps ON ps.case_id = c.id
`

const claimCaseBySitesQuery = claimCaseSelect + `
	LEFT JOIN sites st ON st.id = c.site_id
	WHERE c.deleted_at IS NULL
	  AND NOT ps.is_pending
	  AND c.site_id = ANY($1::uuid[])
	ORDER BY c.name, c.id
`

// 區域用 INNER JOIN：區域只存在於 sites.region，沒綁據點的個案不屬於任何區域。
// 比對用 btrim 後等值而非 ILIKE '%...%'——case_repo 的模糊比對是給關鍵字搜尋用的，
// 在這裡「新竹」會連「新竹縣」一起命中，多報一整批不該報的個案。
const claimCaseByRegionsQuery = claimCaseSelect + `
	JOIN sites st ON st.id = c.site_id
	WHERE c.deleted_at IS NULL
	  AND NOT ps.is_pending
	  AND btrim(st.region) = ANY($1::text[])
	ORDER BY c.name, c.id
`

// ListCasesBySites 解析指定據點底下的申報對象個案。
func (r *ClaimCaseRepository) ListCasesBySites(ctx context.Context, siteIDs []uuid.UUID) ([]app.ScopedCase, error) {
	if len(siteIDs) == 0 {
		return []app.ScopedCase{}, nil
	}
	return r.query(ctx, claimCaseBySitesQuery, pgxdb.UUIDStrings(siteIDs))
}

// ListCasesByRegions 解析指定區域底下的申報對象個案。
func (r *ClaimCaseRepository) ListCasesByRegions(ctx context.Context, regions []string) ([]app.ScopedCase, error) {
	if len(regions) == 0 {
		return []app.ScopedCase{}, nil
	}
	return r.query(ctx, claimCaseByRegionsQuery, regions)
}

func (r *ClaimCaseRepository) query(ctx context.Context, sql string, arg interface{}) ([]app.ScopedCase, error) {
	if r.db == nil {
		return nil, fmt.Errorf("claim case database is not configured")
	}

	rows, err := r.db.Query(ctx, sql, arg)
	if err != nil {
		return nil, fmt.Errorf("query claim cases: %w", err)
	}
	defer rows.Close()

	result := make([]app.ScopedCase, 0)
	for rows.Next() {
		var item app.ScopedCase
		if err := rows.Scan(&item.ID, &item.Name, &item.SiteID, &item.SiteName, &item.Region); err != nil {
			return nil, fmt.Errorf("scan claim case: %w", err)
		}
		result = append(result, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate claim cases: %w", err)
	}
	return result, nil
}
