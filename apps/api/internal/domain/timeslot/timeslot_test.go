package timeslot

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEndTime(t *testing.T) {
	tests := []struct {
		name        string
		depart      time.Time
		durationMin int
		wantHour    int
		wantMin     int
		wantErr     bool
		errTarget   error
	}{
		{
			name:        "09:40 + 10 分鐘 -> 09:50",
			depart:      time.Date(2026, 7, 1, 9, 40, 0, 0, time.UTC),
			durationMin: 10,
			wantHour:    9,
			wantMin:     50,
		},
		{
			name:        "09:55 + 10 分鐘 (跨小時進位) -> 10:05",
			depart:      time.Date(2026, 7, 1, 9, 55, 0, 0, time.UTC),
			durationMin: 10,
			wantHour:    10,
			wantMin:     5,
		},
		{
			name:        "09:00 + 60 分鐘 -> 10:00",
			depart:      time.Date(2026, 7, 1, 9, 0, 0, 0, time.UTC),
			durationMin: 60,
			wantHour:    10,
			wantMin:     0,
		},
		{
			name:        "23:55 + 10 分鐘 (跨日) -> 報錯",
			depart:      time.Date(2026, 7, 1, 23, 55, 0, 0, time.UTC),
			durationMin: 10,
			wantErr:     true,
			errTarget:   ErrSpansAcrossDay,
		},
		{
			name:        "跨年但日期日數相同仍視為跨日",
			depart:      time.Date(2026, 12, 31, 23, 59, 0, 0, time.UTC),
			durationMin: 1,
			wantErr:     true,
			errTarget:   ErrSpansAcrossDay,
		},
		{
			name:        "超過四小時 -> 報錯",
			depart:      time.Date(2026, 7, 1, 9, 0, 0, 0, time.UTC),
			durationMin: 241,
			wantErr:     true,
			errTarget:   ErrExcessiveDuration,
		},
		{
			name:        "時長 <= 0 -> 報錯",
			depart:      time.Date(2026, 7, 1, 9, 0, 0, 0, time.UTC),
			durationMin: 0,
			wantErr:     true,
			errTarget:   ErrNegativeDuration,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, m, err := EndTime(tt.depart, tt.durationMin)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errTarget != nil {
					assert.ErrorIs(t, err, tt.errTarget)
				}
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantHour, h)
			assert.Equal(t, tt.wantMin, m)
		})
	}
}

func TestEndTimeFromHourMin(t *testing.T) {
	h, m, err := EndTimeFromHourMin(9, 55, 10)
	require.NoError(t, err)
	assert.Equal(t, 10, h)
	assert.Equal(t, 5, m)

	_, _, err = EndTimeFromHourMin(25, 0, 10)
	assert.Error(t, err)
}
