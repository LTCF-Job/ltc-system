package calendar

import (
	"fmt"
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
func CalculateExpectedRides(year, month int, input CaseScheduleCalendarInput) ([]ExpectedRide, error) {
	var results []ExpectedRide
	days, err := CalculateScheduleDays(year, month, input)
	if err != nil {
		return nil, err
	}
	for _, day := range days {
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
	return results, nil
}

// CalculateScheduleDays applies monthly override, holiday, weekly, then fixed priority.
// An explicit monthly entry, including tripCount 0, is allowed to override a holiday.
func CalculateScheduleDays(year, month int, input CaseScheduleCalendarInput) ([]ScheduleDay, error) {
	var results []ScheduleDay
	var err error
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
		if (input.ClaimEndDate != nil && d.After(*input.ClaimEndDate)) || d.Before(input.EffectiveFrom) || (input.EffectiveTo != nil && d.After(*input.EffectiveTo)) {
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
				day.Legs, err = scheduleLegs(cfg.TripCount, cfg.Legs, input.Legs)
				if err != nil {
					return nil, fmt.Errorf("monthly schedule %s: %w", dateStr, err)
				}
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
				day.Legs, err = scheduleLegs(cfg.TripCount, cfg.Legs, input.Legs)
				if err != nil {
					return nil, fmt.Errorf("weekly schedule weekday %d: %w", weekday, err)
				}
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
	return results, nil
}

func scheduleLegs(tripCount int16, legs, fallback []LegInput) ([]LegInput, error) {
	if tripCount > 0 && int(tripCount) <= len(legs) {
		return legs[:tripCount], nil
	}
	if tripCount > 0 && int(tripCount) <= len(fallback) {
		return fallback[:tripCount], nil
	}
	return nil, fmt.Errorf("trip count %d has no complete leg definition", tripCount)
}
