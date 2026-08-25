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
