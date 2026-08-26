package calendar

import (
	"time"

	"github.com/google/uuid"
)

// ExpectedRide 代表預期應搭乘之日期與時段序列。
type ExpectedRide struct {
	CaseID      uuid.UUID
	ServiceDate time.Time
	LegSeq      int16
	Direction   string
	DepartTime  string
}

// CaseScheduleCalendarInput 提供計算應搭乘日曆所需之排班參數。
type CaseScheduleCalendarInput struct {
	CaseID         uuid.UUID
	ClaimStartDate time.Time
	ClaimEndDate   *time.Time
	EffectiveFrom  time.Time
	EffectiveTo    *time.Time
	Weekdays       []int16 // 1..7 (週一..週日)
	SiteOpenDays   []int16
	Holidays       map[string]bool // "2026-07-01": true
	Legs           []LegInput
}

// LegInput 代表排班中的時段定義。
type LegInput struct {
	LegSeq     int16
	Direction  string
	DepartTime string
}

// CalculateExpectedRides 依據 R4/R6/R8 規則計算特定月份個案預期搭乘之全部趟次。
func CalculateExpectedRides(year, month int, input CaseScheduleCalendarInput) []ExpectedRide {
	var results []ExpectedRide

	// 檢查星期集合
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
		// 1. 檢查排班星期與據點開放日
		goWeekday := int(d.Weekday())
		if goWeekday == 0 {
			goWeekday = 7 // 週日為 7
		}
		if !weekdayMap[goWeekday] || !siteOpenMap[goWeekday] {
			continue
		}

		// 2. 檢查個案開始與結束申報日 (R8)
		if d.Before(input.ClaimStartDate) {
			continue
		}
		if input.ClaimEndDate != nil && d.After(*input.ClaimEndDate) {
			continue
		}

		// 3. 檢查排班有效區間
		if d.Before(input.EffectiveFrom) {
			continue
		}
		if input.EffectiveTo != nil && d.After(*input.EffectiveTo) {
			continue
		}

		// 4. 排除國定假日
		dateStr := d.Format("2006-01-02")
		if input.Holidays[dateStr] {
			continue
		}

		// 5. 展開該排班之所有 legs
		for _, leg := range input.Legs {
			results = append(results, ExpectedRide{
				CaseID:      input.CaseID,
				ServiceDate: d,
				LegSeq:      leg.LegSeq,
				Direction:   leg.Direction,
				DepartTime:  leg.DepartTime,
			})
		}
	}

	return results
}
