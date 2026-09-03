package infra

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"ltc-system/apps/api/internal/modules/reporting/app"
)

// GovClaimRepository 查詢政府申報所需的趟次、排班、個案、司機與車輛資料。
type GovClaimRepository struct {
	db *pgxpool.Pool
}

// NewGovClaimRepository 建立 GovClaimRepository 實例。
func NewGovClaimRepository(db *pgxpool.Pool) *GovClaimRepository {
	return &GovClaimRepository{db: db}
}

// govClaimSourceQuery 一次撈齊組列所需欄位。
// schedule_legs 用 LEFT JOIN：對不到排班趟次時 direction 為 NULL，交由 app 明確計入跳過清單，
// 而不是被 INNER JOIN 靜默濾掉，讓資料缺漏在匯出結果上看不見。
const govClaimSourceQuery = `
	SELECT
		c.id, c.name, COALESCE(c.region, ''),
		c.national_id_cipher, COALESCE(c.national_id_masked, ''),
		COALESCE(c.home_address, ''), c.service_category, c.service_usage_type,
		r.service_date, r.leg_seq, r.not_claimed_aa09,
		l.direction,
		to_char(COALESCE(r.depart_time_override, l.depart_time), 'HH24:MI'),
		COALESCE(r.duration_min_override, s.service_duration_min)::int,
		s.service_code, s.unit_price::float8, s.distance_km::float8,
		st.address, COALESCE(v.plate_no, ''),
		d.id, d.national_id_cipher
	FROM ride_records r
	JOIN cases c ON c.id = r.case_id
	JOIN case_schedules s ON s.case_id = c.id AND s.effective_range @> r.service_date
	JOIN sites st ON st.id = s.site_id
	LEFT JOIN schedule_legs l ON l.schedule_id = s.id AND l.leg_seq = r.leg_seq
	LEFT JOIN vehicles v ON v.id = r.vehicle_id
	LEFT JOIN drivers d ON d.id = r.driver_id
	WHERE r.service_date >= $1 AND r.service_date < $2
	  AND r.effective_status = 'boarded'
	  AND ($3 = '' OR c.region = $3)
	  AND (COALESCE(cardinality($4::uuid[]), 0) = 0 OR c.id = ANY($4::uuid[]))
	ORDER BY c.name, r.leg_seq, r.service_date
`

// QueryGovClaimSources 查詢指定期間內可申報的搭乘紀錄。
// 只取 effective_status = 'boarded'：缺席與未回報不申報。not_claimed_aa09 不是過濾條件，
// 它只決定申報列的第 17 欄，該列仍須出現。
func (r *GovClaimRepository) QueryGovClaimSources(
	ctx context.Context,
	start, end time.Time,
	region string,
	caseIDs []uuid.UUID,
) ([]app.GovClaimSource, error) {
	if r.db == nil {
		return []app.GovClaimSource{}, nil
	}
	if caseIDs == nil {
		caseIDs = []uuid.UUID{}
	}

	rows, err := r.db.Query(ctx, govClaimSourceQuery, start, end, region, caseIDs)
	if err != nil {
		return nil, fmt.Errorf("query gov claim sources: %w", err)
	}
	defer rows.Close()

	result := make([]app.GovClaimSource, 0)
	for rows.Next() {
		var row govClaimSourceRow
		if err := rows.Scan(
			&row.CaseID, &row.CaseName, &row.Region,
			&row.CaseNationalIDCipher, &row.CaseNationalIDMasked,
			&row.HomeAddress, &row.ServiceCategory, &row.ServiceUsageType,
			&row.ServiceDate, &row.LegSeq, &row.NotClaimedAA09,
			&row.Direction, &row.DepartTime, &row.DurationMin,
			&row.ServiceCode, &row.UnitPrice, &row.DistanceKM,
			&row.SiteAddress, &row.PlateNo,
			&row.DriverID, &row.DriverNationalIDCipher,
		); err != nil {
			return nil, fmt.Errorf("scan gov claim source: %w", err)
		}
		result = append(result, row.toApp())
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate gov claim sources: %w", err)
	}

	return result, nil
}
