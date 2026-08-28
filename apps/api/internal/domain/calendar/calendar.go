package calendar

import (
	"time"

	"github.com/google/uuid"
)

type ExpectedRide struct {
	CaseID      uuid.UUID
	ServiceDate time.Time
	LegSeq      int16
	Direction   string
	DepartTime  string
}

type DayScheduleInput struct {
	TripCount int16
	Legs      []LegInput
}

type WeekdayScheduleInput struct {
	TripCount int16
	Legs      []LegInput
}

type CaseScheduleCalendarInput struct {
	CaseID         uuid.UUID
	ClaimStartDate time.Time
	ClaimEndDate   *time.Time
	EffectiveFrom  time.Time
	EffectiveTo    *time.Time
	Weekdays       []int16
	SiteOpenDays   []int16
	Holidays       map[string]bool
	Legs           []LegInput
	WeeklyConfigs  map[int]WeekdayScheduleInput
	MonthlyConfigs map[string]DayScheduleInput
}

type LegInput struct {
	LegSeq     int16
	Direction  string
	DepartTime string
}

type ScheduleDayStatus string

const (
	ScheduleDayScheduled       ScheduleDayStatus = "scheduled"
	ScheduleDayHoliday         ScheduleDayStatus = "holiday"
	ScheduleDayManualAbsent    ScheduleDayStatus = "manual_absent"
	ScheduleDayManualScheduled ScheduleDayStatus = "manual_scheduled"
	ScheduleDayNonScheduled    ScheduleDayStatus = "non_scheduled"
)

type ScheduleDay struct {
	Date             time.Time
	Weekday          int
	IsWeekend        bool
	IsHoliday        bool
	IsManualOverride bool
	Source           string
	Status           ScheduleDayStatus
	Legs             []LegInput
}

// CalculateExpectedRides returns only dates and legs that should be reported.
func CalculateExpectedRides(year, month int, input CaseScheduleCalendarInput) []ExpectedRide {
	var results []ExpectedRide
	for _, day := range CalculateScheduleDays(year, month, input) {
		for _, leg := range day.Legs {
			results = append(results, ExpectedRide{
				CaseID:      input.CaseID,
				ServiceDate: day.Date,
				LegSeq:      leg.LegSeq,
				Direction:   leg.Direction,
				DepartTime:  leg.DepartTime,
			})
		}
	}
	return results
}

// CalculateScheduleDays applies monthly override, holiday, weekly, then fixed priority.
// An explicit monthly entry, including tripCount 0, is allowed to override a holiday.
func CalculateScheduleDays(year, month int, input CaseScheduleCalendarInput) []ScheduleDay {
	var results []ScheduleDay
	weekdayMap := make(map[int]bool)
	for _, wd := range input.Weekdays {
		weekdayMap[int(wd)] = true
	}
	siteOpenMap := make(map[int]bool)
	for _, wd := range input.SiteOpenDays {
		siteOpenMap[int(wd)] = true
	}

	firstDay := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	lastDay := firstDay.AddDate(0, 1, -1)
	for d := firstDay; !d.After(lastDay); d = d.AddDate(0, 0, 1) {
		if d.Before(input.ClaimStartDate) || (input.ClaimEndDate != nil && d.After(*input.ClaimEndDate)) || d.Before(input.EffectiveFrom) || (input.EffectiveTo != nil && d.After(*input.EffectiveTo)) {
			continue
		}

		weekday := int(d.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		dateStr := d.Format("2006-01-02")
		isHoliday := input.Holidays != nil && input.Holidays[dateStr]
		day := ScheduleDay{
			Date:      d,
			Weekday:   weekday,
			IsWeekend: weekday >= 6,
			IsHoliday: isHoliday,
			Source:    "fixed",
			Status:    ScheduleDayNonScheduled,
		}

		if cfg, ok := input.MonthlyConfigs[dateStr]; ok {
			day.IsManualOverride = true
			day.Source = "monthly"
			if cfg.TripCount > 0 {
				day.Status = ScheduleDayManualScheduled
				day.Legs = scheduleLegs(cfg.TripCount, cfg.Legs, input.Legs)
			} else {
				day.Status = ScheduleDayManualAbsent
			}
			results = append(results, day)
			continue
		}

		if isHoliday {
			day.Source = "holiday"
			day.Status = ScheduleDayHoliday
			results = append(results, day)
			continue
		}

		if cfg, ok := input.WeeklyConfigs[weekday]; ok {
			day.Source = "weekly"
			if cfg.TripCount > 0 && siteOpenMap[weekday] {
				day.Status = ScheduleDayScheduled
				day.Legs = scheduleLegs(cfg.TripCount, cfg.Legs, input.Legs)
			}
			results = append(results, day)
			continue
		}

		if siteOpenMap[weekday] && weekdayMap[weekday] {
			day.Status = ScheduleDayScheduled
			day.Legs = input.Legs
		}
		results = append(results, day)
	}
	return results
}

func scheduleLegs(tripCount int16, legs, fallback []LegInput) []LegInput {
	if tripCount > 0 && int(tripCount) <= len(legs) {
		return legs[:tripCount]
	}
	if tripCount > 0 && int(tripCount) <= len(fallback) {
		return fallback[:tripCount]
	}
	switch tripCount {
	case 1:
		return []LegInput{{LegSeq: 1, Direction: "outbound", DepartTime: "09:00"}}
	case 2:
		return []LegInput{{LegSeq: 1, Direction: "outbound", DepartTime: "09:00"}, {LegSeq: 2, Direction: "inbound", DepartTime: "16:00"}}
	case 4:
		return []LegInput{{LegSeq: 1, Direction: "outbound", DepartTime: "08:30"}, {LegSeq: 2, Direction: "inbound", DepartTime: "11:30"}, {LegSeq: 3, Direction: "outbound", DepartTime: "13:30"}, {LegSeq: 4, Direction: "inbound", DepartTime: "16:30"}}
	default:
		return fallback
	}
}
