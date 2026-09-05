package timeslot

import (
	"errors"
	"time"
)

var (
	ErrNegativeDuration = errors.New("duration must be greater than 0")
	ErrExcessiveDuration = errors.New("duration exceeds the maximum allowed minutes")
	ErrSpansAcrossDay   = errors.New("service end time exceeds same day (23:59)")
)

// EndTime 計算服務結束時間的時與分，處理跨小時進位並防止跨日。
func EndTime(depart time.Time, durationMin int) (hour, minute int, err error) {
	if durationMin <= 0 {
		return 0, 0, ErrNegativeDuration
	}
	if durationMin > 240 {
		return 0, 0, ErrExcessiveDuration
	}

	endTime := depart.Add(time.Duration(durationMin) * time.Minute)

	// 跨日視為排班設定錯誤
	if endTime.Year() != depart.Year() || endTime.Month() != depart.Month() || endTime.Day() != depart.Day() {
		return 0, 0, ErrSpansAcrossDay
	}

	return endTime.Hour(), endTime.Minute(), nil
}

// EndTimeFromHourMin 依據出發時、分與時長計算結束時與分。
func EndTimeFromHourMin(depHour, depMin int, durationMin int) (endHour, endMin int, err error) {
	if depHour < 0 || depHour > 23 || depMin < 0 || depMin > 59 {
		return 0, 0, errors.New("invalid departure hour or minute")
	}
	depart := time.Date(2000, 1, 1, depHour, depMin, 0, 0, time.UTC)
	return EndTime(depart, durationMin)
}
