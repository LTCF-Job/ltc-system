package calendar

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestCalculateExpectedRides(t *testing.T) {
	caseID := uuid.New()
	input := CaseScheduleCalendarInput{
		CaseID:         caseID,
		ClaimStartDate: time.Date(2026, 7, 10, 0, 0, 0, 0, time.UTC), // 7/10 開始申報 (R8)
		EffectiveFrom:  time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		Weekdays:       []int16{1, 2, 3, 4, 5},
		SiteOpenDays:   []int16{1, 2, 3, 4, 5},
		Holidays: map[string]bool{
			"2026-07-15": true, // 假設 7/15 停駛
		},
		Legs: []LegInput{
			{LegSeq: 1, Direction: "outbound", DepartTime: "09:40"},
			{LegSeq: 2, Direction: "inbound", DepartTime: "16:00"},
		},
	}

	rides := CalculateExpectedRides(2026, 7, input)
	assert.NotEmpty(t, rides)

	for _, r := range rides {
		// 必須 >= 2026-07-10
		assert.False(t, r.ServiceDate.Before(input.ClaimStartDate))
		// 不應包含 7/15
		assert.NotEqual(t, "2026-07-15", r.ServiceDate.Format("2006-01-02"))
	}
}

func TestCalculateExpectedRides_PriorityOrder(t *testing.T) {
	caseID := uuid.New()

	// 固定排班：週一至週五 2 趟
	fixedLegs := []LegInput{
		{LegSeq: 1, Direction: "outbound", DepartTime: "09:00"},
		{LegSeq: 2, Direction: "inbound", DepartTime: "16:00"},
	}

	input := CaseScheduleCalendarInput{
		CaseID:         caseID,
		ClaimStartDate: time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
		EffectiveFrom:  time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
		Weekdays:       []int16{1, 2, 3, 4, 5},
		SiteOpenDays:   []int16{1, 2, 3, 4, 5},
		Legs:           fixedLegs,
		// 當周排班：週二 4 趟、週三 0 趟
		WeeklyConfigs: map[int]WeekdayScheduleInput{
			2: {TripCount: 4},
			3: {TripCount: 0},
		},
		// 當月排班：
		// 2026-07-14 (週二)：當月請假設為 0 趟 (覆寫當周 4 趟)
		// 2026-07-15 (週三)：當月特開 2 趟 (覆寫當周 0 趟)
		// 2026-07-16 (週四)：當月特開 4 趟 (覆寫固定 2 趟)
		MonthlyConfigs: map[string]DayScheduleInput{
			"2026-07-14": {TripCount: 0},
			"2026-07-15": {TripCount: 2},
			"2026-07-16": {TripCount: 4},
		},
	}

	rides := CalculateExpectedRides(2026, 7, input)

	dateRideCount := make(map[string]int)
	for _, r := range rides {
		ds := r.ServiceDate.Format("2006-01-02")
		dateRideCount[ds]++
	}

	// 1. 驗證當月優先 (2026-07-14 週二)：當月請假 0 趟 > 當周 4 趟
	assert.Equal(t, 0, dateRideCount["2026-07-14"], "7/14 應由當月排班 (0 趟) 覆寫當周 4 趟")

	// 2. 驗證當月優先 (2026-07-15 週三)：當月特開 2 趟 > 當周 0 趟
	assert.Equal(t, 2, dateRideCount["2026-07-15"], "7/15 應由當月排班 (2 趟) 覆寫當周 0 趟")

	// 3. 驗證當月優先 (2026-07-16 週四)：當月特開 4 趟 > 固定 2 趟
	assert.Equal(t, 4, dateRideCount["2026-07-16"], "7/16 應由當月排班 (4 趟) 覆寫固定 2 趟")

	// 4. 驗證當周優先 (2026-07-21 週二)：無當月設定，採用當周 4 趟 > 固定 2 趟
	assert.Equal(t, 4, dateRideCount["2026-07-21"], "7/21 應採用當周排班 (4 趟)")

	// 5. 驗證當周優先 (2026-07-22 週三)：無當月設定，採用當周 0 趟 > 固定 2 趟
	assert.Equal(t, 0, dateRideCount["2026-07-22"], "7/22 應採用當周排班 (0 趟)")

	// 6. 驗證回落固定 (2026-07-24 週五)：當月與當周皆無設定，回落固定 2 趟
	assert.Equal(t, 2, dateRideCount["2026-07-24"], "7/24 應回落至固定常態排班 (2 趟)")
}
