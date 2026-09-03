package infra

import (
	"context"
	"time"

	"github.com/google/uuid"
	"ltc-system/apps/api/internal/modules/ride/app"
	"ltc-system/apps/api/internal/platform/pgxdb"
)

// ListRideSourcesForSlot 取得單一 slot 已寫入的全部回報來源，供混車合併運算。
func (r *RideRepository) ListRideSourcesForSlot(
	ctx context.Context,
	caseID uuid.UUID,
	serviceDate time.Time,
	legSeq int16,
) ([]app.RideSourceRow, error) {
	query := `
		SELECT rs.vehicle_id, rs.driver_id, rs.reported, fs.submitted_at
		FROM ride_sources rs
		JOIN form_submissions fs ON rs.submission_id = fs.id
		WHERE rs.case_id = $1 AND rs.service_date = $2 AND rs.leg_seq = $3
		ORDER BY fs.submitted_at ASC
	`
	db := pgxdb.FromContext(ctx, r.db)
	rows, err := db.Query(ctx, query, caseID, serviceDate, legSeq)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sources []app.RideSourceRow
	for rows.Next() {
		var s app.RideSourceRow
		if err := rows.Scan(&s.VehicleID, &s.DriverID, &s.Reported, &s.SubmittedAt); err != nil {
			return nil, err
		}
		sources = append(sources, s)
	}
	return sources, rows.Err()
}

// ListCalendarCases 取得該月份有生效排班的個案與其趟次時段。
//
// effective_range 與查詢區間有交集即納入，個案是否真的要出車由 domain/calendar
// 依 weekdays 與單位營業日逐日推算。
func (r *RideRepository) ListCalendarCases(
	ctx context.Context,
	start, end time.Time,
	region, keyword string,
) ([]app.CalendarCase, error) {
	query := `
		SELECT c.id, c.name, c.region, c.claim_end_date,
		       cs.id, cs.trip_pattern, cs.weekdays, s.open_days,
		       lower(cs.effective_range), upper(cs.effective_range)
		FROM cases c
		JOIN case_schedules cs ON cs.case_id = c.id AND cs.effective_range && daterange($1::date, $2::date, '[)')
		JOIN sites s ON cs.site_id = s.id
		WHERE c.status = 'active'
		  AND ($3 = '' OR c.region = $3)
		  AND ($4 = '' OR c.name ILIKE '%' || $4 || '%')
		ORDER BY c.name ASC
	`
	db := pgxdb.FromContext(ctx, r.db)
	rows, err := db.Query(ctx, query, start, end, region, keyword)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cases []app.CalendarCase
	scheduleIDs := map[uuid.UUID]int{}
	for rows.Next() {
		var c app.CalendarCase
		var scheduleID uuid.UUID
		var effectiveTo *time.Time
		if err := rows.Scan(
			&c.ID, &c.Name, &c.Region, &c.ClaimEndDate,
			&scheduleID, &c.TripPattern, &c.Weekdays, &c.SiteOpenDays,
			&c.EffectiveFrom, &effectiveTo,
		); err != nil {
			return nil, err
		}
		c.EffectiveTo = effectiveTo
		scheduleIDs[scheduleID] = len(cases)
		cases = append(cases, c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(cases) == 0 {
		return cases, nil
	}

	ids := make([]uuid.UUID, 0, len(scheduleIDs))
	for id := range scheduleIDs {
		ids = append(ids, id)
	}

	legRows, err := db.Query(ctx, `
		SELECT sl.schedule_id, sl.leg_seq, sl.direction, to_char(sl.depart_time, 'HH24:MI'),
		       sl.vehicle_id, COALESCE(v.display_name, '')
		FROM schedule_legs sl
		LEFT JOIN vehicles v ON sl.vehicle_id = v.id
		WHERE sl.schedule_id = ANY($1)
		ORDER BY sl.leg_seq ASC
	`, ids)
	if err != nil {
		return nil, err
	}
	defer legRows.Close()

	for legRows.Next() {
		var scheduleID uuid.UUID
		var leg app.CalendarLeg
		if err := legRows.Scan(&scheduleID, &leg.LegSeq, &leg.Direction, &leg.DepartTime, &leg.VehicleID, &leg.VehicleName); err != nil {
			return nil, err
		}
		if idx, ok := scheduleIDs[scheduleID]; ok {
			cases[idx].Legs = append(cases[idx].Legs, leg)
		}
	}
	return cases, legRows.Err()
}

// ListRideRecordsInRange 取得區間內的搭乘紀錄，附上車輛與司機顯示名稱。
func (r *RideRepository) ListRideRecordsInRange(
	ctx context.Context,
	start, end time.Time,
	region, keyword string,
) ([]app.RideRecord, error) {
	query := `
		SELECT rr.id, rr.case_id, c.name, rr.service_date, rr.leg_seq,
		       rr.merged_status, rr.effective_status,
		       rr.vehicle_id, COALESCE(v.display_name, ''), rr.driver_id, COALESCE(d.name, ''),
		       rr.has_conflict, rr.conflict_resolved_at, rr.conflict_resolved_by,
		       to_char(rr.depart_time_override, 'HH24:MI'), rr.duration_min_override, rr.not_claimed_aa09,
		       rr.corrected_by, rr.corrected_at, rr.correction_reason, rr.created_at, rr.updated_at
		FROM ride_records rr
		JOIN cases c ON rr.case_id = c.id
		LEFT JOIN vehicles v ON rr.vehicle_id = v.id
		LEFT JOIN drivers d ON rr.driver_id = d.id
		WHERE rr.service_date >= $1 AND rr.service_date < $2
		  AND ($3 = '' OR c.region = $3)
		  AND ($4 = '' OR c.name ILIKE '%' || $4 || '%' OR c.code ILIKE '%' || $4 || '%')
		ORDER BY rr.service_date ASC, rr.leg_seq ASC
	`
	db := pgxdb.FromContext(ctx, r.db)
	rows, err := db.Query(ctx, query, start, end, region, keyword)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []app.RideRecord
	for rows.Next() {
		var rec app.RideRecord
		if err := rows.Scan(
			&rec.ID, &rec.CaseID, &rec.CaseName, &rec.ServiceDate, &rec.LegSeq,
			&rec.MergedStatus, &rec.EffectiveStatus,
			&rec.VehicleID, &rec.VehicleName, &rec.DriverID, &rec.DriverName,
			&rec.HasConflict, &rec.ConflictResolvedAt, &rec.ConflictResolvedBy,
			&rec.DepartTimeOverride, &rec.DurationMinOverride, &rec.NotClaimedAA09,
			&rec.CorrectedBy, &rec.CorrectedAt, &rec.CorrectionReason, &rec.CreatedAt, &rec.UpdatedAt,
		); err != nil {
			return nil, err
		}
		records = append(records, rec)
	}
	return records, rows.Err()
}

// ListPendingConflicts 取得待裁決之混車衝突清單，涉及車輛聚合自 ride_sources 中已回報「有坐」的來源。
func (r *RideRepository) ListPendingConflicts(ctx context.Context, start, end time.Time, keyword string, page, pageSize int) ([]app.ConflictRide, int64, error) {
	offset := (page - 1) * pageSize
	query := `
		SELECT rr.id, rr.case_id, c.name, rr.service_date, rr.leg_seq, COALESCE(veh.vehicles, ARRAY[]::text[]),
		       count(*) OVER()
		FROM ride_records rr
		JOIN cases c ON rr.case_id = c.id
		LEFT JOIN LATERAL (
			SELECT array_agg(DISTINCT v.display_name ORDER BY v.display_name) AS vehicles
			FROM ride_sources rs
			JOIN vehicles v ON rs.vehicle_id = v.id
			WHERE rs.case_id = rr.case_id AND rs.service_date = rr.service_date AND rs.leg_seq = rr.leg_seq
			  AND rs.reported = 'boarded'
		) veh ON true
		WHERE rr.has_conflict = true AND rr.conflict_resolved_at IS NULL
		  AND rr.service_date >= $1 AND rr.service_date <= $2
		  AND ($3 = '' OR c.name ILIKE '%' || $3 || '%' OR c.code ILIKE '%' || $3 || '%')
		ORDER BY rr.service_date ASC, rr.leg_seq ASC
		LIMIT $4 OFFSET $5
	`
	db := pgxdb.FromContext(ctx, r.db)
	rows, err := db.Query(ctx, query, start, end, keyword, pageSize, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var items []app.ConflictRide
	var total int64
	for rows.Next() {
		var item app.ConflictRide
		if err := rows.Scan(&item.ID, &item.CaseID, &item.CaseName, &item.ServiceDate, &item.LegSeq, &item.Vehicles, &total); err != nil {
			return nil, 0, err
		}
		items = append(items, item)
	}
	return items, total, rows.Err()
}

// ListImportErrorSubmissions 取得帶有異常標記之匯入紀錄清單。
func (r *RideRepository) ListImportErrorSubmissions(ctx context.Context, start, end time.Time, keyword string, page, pageSize int) ([]app.ImportErrorSubmission, int64, error) {
	offset := (page - 1) * pageSize
	query := `
		SELECT id, service_date, driver_name_raw, anomaly_flags, payload::text, count(*) OVER()
		FROM form_submissions
		WHERE array_length(anomaly_flags, 1) > 0
		  AND service_date >= $1 AND service_date <= $2
		  AND ($3 = '' OR driver_name_raw ILIKE '%' || $3 || '%')
		ORDER BY service_date DESC, submitted_at DESC
		LIMIT $4 OFFSET $5
	`
	db := pgxdb.FromContext(ctx, r.db)
	rows, err := db.Query(ctx, query, start, end, keyword, pageSize, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var items []app.ImportErrorSubmission
	var total int64
	for rows.Next() {
		var item app.ImportErrorSubmission
		if err := rows.Scan(&item.ID, &item.ServiceDate, &item.DriverNameRaw, &item.AnomalyFlags, &item.RawPayload, &total); err != nil {
			return nil, 0, err
		}
		items = append(items, item)
	}
	return items, total, rows.Err()
}
