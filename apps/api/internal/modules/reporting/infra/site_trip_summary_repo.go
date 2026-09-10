package infra

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"ltc-system/apps/api/internal/modules/reporting/app"
	"ltc-system/apps/api/internal/platform/pgxdb"
)

// SiteTripSummaryRepository 查詢據點趟數彙總表所需的逐案逐月趟數。
type SiteTripSummaryRepository struct {
	db *pgxpool.Pool
}

// NewSiteTripSummaryRepository 建立 SiteTripSummaryRepository 實例。
func NewSiteTripSummaryRepository(db *pgxpool.Pool) *SiteTripSummaryRepository {
	return &SiteTripSummaryRepository{db: db}
}

// siteTripCountQuery 逐案逐月統計趟數。
//
// 趟次的過濾條件（boarded、非待維護、無未裁決衝突）刻意與 govClaimSourceQuery 一字不差：
// 這張彙總表的用途就是拿來對帳申報檔，兩邊口徑一旦不同，使用者算出來的趟數對不上報出去
// 的列數，卻查不出是哪一邊錯。任一邊調整條件都必須同步調整另一邊。
//
// CROSS JOIN months 搭配 LEFT JOIN ride_records 是這支查詢的重點：據點底下的每一位個案
// × 每一個選取月份都保證有一列，該月零趟就是 0 而不是整列消失。使用者要的是「這個據點
// 的五個個案分別跑了幾趟」，零趟本身就是答案的一部分。
const siteTripCountQuery = `
	WITH months AS (
		SELECT * FROM unnest($2::date[], $3::date[], $4::text[]) AS m(start_date, end_date, period_ym)
	), target_cases AS (
		SELECT c.id, c.name, c.site_id, COALESCE(st.name, '') AS site_name
		FROM cases c
		JOIN case_pending_status ps ON ps.case_id = c.id
		LEFT JOIN sites st ON st.id = c.site_id
		WHERE c.deleted_at IS NULL
		  AND NOT ps.is_pending
		  AND c.site_id = ANY($1::uuid[])
	)
	SELECT tc.id, tc.name, tc.site_id, tc.site_name, m.period_ym, COUNT(r.id)::int
	FROM target_cases tc
	CROSS JOIN months m
	LEFT JOIN ride_records r
	       ON r.case_id = tc.id
	      AND r.service_date >= m.start_date
	      AND r.service_date < m.end_date
	      AND r.effective_status = 'boarded'
	      AND (r.has_conflict = false OR r.conflict_resolved_at IS NOT NULL)
	GROUP BY tc.id, tc.name, tc.site_id, tc.site_name, m.period_ym
	ORDER BY tc.name, tc.id, m.period_ym
`

// QuerySiteTripCounts 查詢指定據點與月份的逐案趟數。
func (r *SiteTripSummaryRepository) QuerySiteTripCounts(
	ctx context.Context,
	siteIDs []uuid.UUID,
	months []app.ClaimMonth,
) ([]app.SiteTripCount, error) {
	if r.db == nil {
		return nil, fmt.Errorf("site trip summary database is not configured")
	}
	if len(siteIDs) == 0 || len(months) == 0 {
		return []app.SiteTripCount{}, nil
	}

	starts := make([]time.Time, len(months))
	ends := make([]time.Time, len(months))
	periods := make([]string, len(months))
	for i, m := range months {
		starts[i] = m.StartDate
		ends[i] = m.EndDate
		periods[i] = m.PeriodYM
	}

	rows, err := r.db.Query(ctx, siteTripCountQuery, pgxdb.UUIDStrings(siteIDs), starts, ends, periods)
	if err != nil {
		return nil, fmt.Errorf("query site trip counts: %w", err)
	}
	defer rows.Close()

	result := make([]app.SiteTripCount, 0)
	for rows.Next() {
		var item app.SiteTripCount
		if err := rows.Scan(&item.CaseID, &item.CaseName, &item.SiteID, &item.SiteName, &item.PeriodYM, &item.TripCount); err != nil {
			return nil, fmt.Errorf("scan site trip count: %w", err)
		}
		result = append(result, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate site trip counts: %w", err)
	}
	return result, nil
}
