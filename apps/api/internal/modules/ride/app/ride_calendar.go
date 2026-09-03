package app

import (
	"context"
	"time"

	"github.com/google/uuid"
	"ltc-system/apps/api/internal/domain/calendar"
)

// CalendarCellRecord 是月曆單一格內的一筆搭乘紀錄。
type CalendarCellRecord struct {
	ID                  string  `json:"id"`
	CaseID              string  `json:"caseId"`
	CaseName            string  `json:"caseName,omitempty"`
	ServiceDate         string  `json:"serviceDate"`
	LegSeq              int16   `json:"legSeq"`
	Direction           string  `json:"direction,omitempty"`
	MergedStatus        string  `json:"mergedStatus"`
	EffectiveStatus     string  `json:"effectiveStatus"`
	HasConflict         bool    `json:"hasConflict"`
	VehicleID           string  `json:"vehicleId,omitempty"`
	VehicleName         string  `json:"vehicleName,omitempty"`
	DriverID            string  `json:"driverId,omitempty"`
	DriverName          string  `json:"driverName,omitempty"`
	DepartTimeOverride  *string `json:"departTimeOverride,omitempty"`
	DurationMinOverride *int16  `json:"durationMinOverride,omitempty"`
	ScheduledDepartTime string  `json:"scheduledDepartTime,omitempty"`
	NotClaimedAA09      bool    `json:"notClaimedAa09"`
	CorrectedAt         *string `json:"correctedAt,omitempty"`
	CorrectionReason    *string `json:"correctionReason,omitempty"`
	Sources             []any   `json:"sources"`
}

// CalendarCell 是月曆的單日格子。
type CalendarCell struct {
	Date              string               `json:"date"`
	DayOfWeek         int                  `json:"dayOfWeek"`
	IsExpected        bool                 `json:"isExpected"`
	ExpectedTripCount int                  `json:"expectedTripCount"`
	IsHoliday         bool                 `json:"isHoliday"`
	Records           []CalendarCellRecord `json:"records"`
}

// CalendarRow 是月曆的單一個案列。
type CalendarRow struct {
	CaseID      string                  `json:"caseId"`
	CaseName    string                  `json:"caseName"`
	Region      string                  `json:"region"`
	TripPattern int16                   `json:"tripPattern"`
	Days        map[string]CalendarCell `json:"days"`
}

// CalendarMatrix 是搭乘月曆表的完整回應。
type CalendarMatrix struct {
	Month       string        `json:"month"`
	TotalCases  int           `json:"totalCases"`
	DaysInMonth int           `json:"daysInMonth"`
	Cases       []CalendarRow `json:"cases"`
}

// GetCalendar 依年月彙整搭乘月曆矩陣：應搭乘日由排班推算，實際狀態取自 ride_records。
//
// 沒有排班的個案不會出現在月曆上；有排班但當日無紀錄的格子維持 isExpected 而
// records 為空，前端據此顯示「未回報」。
func (s *RideService) GetCalendar(ctx context.Context, year, month int, region, keyword string) (*CalendarMatrix, error) {
	start := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 1, 0)
	daysInMonth := end.AddDate(0, 0, -1).Day()

	cases, err := s.formRepo.ListCalendarCases(ctx, start, end, region, keyword)
	if err != nil {
		return nil, err
	}

	records, err := s.formRepo.ListRideRecordsInRange(ctx, start, end, region, keyword)
	if err != nil {
		return nil, err
	}

	byCaseDate := map[string][]RideRecord{}
	for _, rec := range records {
		key := rec.CaseID.String() + "|" + rec.ServiceDate.Format("2006-01-02")
		byCaseDate[key] = append(byCaseDate[key], rec)
	}

	matrix := &CalendarMatrix{
		Month:       time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC).Format("2006-01"),
		DaysInMonth: daysInMonth,
		Cases:       make([]CalendarRow, 0, len(cases)),
	}

	for _, c := range cases {
		row := CalendarRow{
			CaseID:      c.ID.String(),
			CaseName:    c.Name,
			Region:      c.Region,
			TripPattern: c.TripPattern,
			Days:        map[string]CalendarCell{},
		}

		legTimes := map[int16]string{}
		legs := make([]calendar.LegInput, 0, len(c.Legs))
		for _, l := range c.Legs {
			legs = append(legs, calendar.LegInput{LegSeq: l.LegSeq, Direction: l.Direction, DepartTime: l.DepartTime})
			legTimes[l.LegSeq] = l.DepartTime
		}

		days := calendar.CalculateScheduleDays(year, month, calendar.CaseScheduleCalendarInput{
			CaseID:        c.ID,
			ClaimEndDate:  c.ClaimEndDate,
			EffectiveFrom: c.EffectiveFrom,
			EffectiveTo:   c.EffectiveTo,
			Weekdays:      c.Weekdays,
			SiteOpenDays:  c.SiteOpenDays,
			Legs:          legs,
		})

		for _, day := range days {
			dateStr := day.Date.Format("2006-01-02")
			cell := CalendarCell{
				Date:              dateStr,
				DayOfWeek:         day.Weekday,
				IsExpected:        len(day.Legs) > 0,
				ExpectedTripCount: len(day.Legs),
				IsHoliday:         day.IsHoliday,
				Records:           []CalendarCellRecord{},
			}
			for _, rec := range byCaseDate[c.ID.String()+"|"+dateStr] {
				cell.Records = append(cell.Records, toCalendarCellRecord(rec, c.Name, legTimes[rec.LegSeq]))
			}
			row.Days[dateStr] = cell
		}

		matrix.Cases = append(matrix.Cases, row)
	}
	matrix.TotalCases = len(matrix.Cases)

	return matrix, nil
}

func toCalendarCellRecord(rec RideRecord, caseName, scheduledDepartTime string) CalendarCellRecord {
	out := CalendarCellRecord{
		ID:                  rec.ID.String(),
		CaseID:              rec.CaseID.String(),
		CaseName:            caseName,
		ServiceDate:         rec.ServiceDate.Format("2006-01-02"),
		LegSeq:              rec.LegSeq,
		MergedStatus:        rec.MergedStatus,
		EffectiveStatus:     rec.EffectiveStatus,
		HasConflict:         rec.HasConflict,
		VehicleName:         rec.VehicleName,
		DriverName:          rec.DriverName,
		DepartTimeOverride:  rec.DepartTimeOverride,
		DurationMinOverride: rec.DurationMinOverride,
		ScheduledDepartTime: scheduledDepartTime,
		NotClaimedAA09:      rec.NotClaimedAA09,
		CorrectionReason:    rec.CorrectionReason,
		Sources:             []any{},
	}
	if rec.VehicleID != uuid.Nil {
		out.VehicleID = rec.VehicleID.String()
	}
	if rec.DriverID != nil {
		out.DriverID = rec.DriverID.String()
	}
	if rec.CorrectedAt != nil {
		formatted := rec.CorrectedAt.Format("2006-01-02 15:04:05")
		out.CorrectedAt = &formatted
	}
	if rec.LegSeq%2 == 1 {
		out.Direction = "outbound"
	} else {
		out.Direction = "inbound"
	}
	return out
}
