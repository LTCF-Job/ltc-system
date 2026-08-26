package rocdate

import (
	"errors"
	"fmt"
	"time"
)

var (
	ErrBeforeROCYear = errors.New("date is before ROC year 1 (1912-01-01)")
	ErrInvalidROCVal = errors.New("invalid ROC date integer")
)

// ToROC 將西元日期轉換為民國 7 碼整數（例如 2026-07-01 -> 1150701）。
func ToROC(d time.Time) (int, error) {
	year := d.Year()
	if year < 1912 {
		return 0, ErrBeforeROCYear
	}
	rocYear := year - 1911
	month := int(d.Month())
	day := d.Day()

	return rocYear*10000 + month*100 + day, nil
}

// FromROC 將民國 7 碼整數（例如 1150701）轉換為西元 time.Time。
func FromROC(v int) (time.Time, error) {
	if v < 10101 {
		return time.Time{}, ErrInvalidROCVal
	}
	rocYear := v / 10000
	month := (v % 10000) / 100
	day := v % 100

	if rocYear < 1 || month < 1 || month > 12 || day < 1 || day > 31 {
		return time.Time{}, ErrInvalidROCVal
	}

	ceYear := rocYear + 1911
	t := time.Date(ceYear, time.Month(month), day, 0, 0, 0, 0, time.UTC)

	// 檢查日期溢位（例如 2 月 30 日或非閏年 2 月 29 日）
	if t.Year() != ceYear || int(t.Month()) != month || t.Day() != day {
		return time.Time{}, ErrInvalidROCVal
	}

	return t, nil
}

// FormatROCString 將民國整數格式化為 7 碼字串（補零）。
func FormatROCString(v int) string {
	return fmt.Sprintf("%07d", v)
}

// FormatROCYearMonth 將西元年與月格式化為 API 民國年月字串（例如 2026, 7 -> "115-07"）。
func FormatROCYearMonth(year, month int) string {
	rocYear := year - 1911
	return fmt.Sprintf("%03d-%02d", rocYear, month)
}

// ParseROCYearMonth 解析民國年月字串（例如 "115-07" -> 2026, 7）。
func ParseROCYearMonth(str string) (int, int, error) {
	var rocYear, month int
	if n, _ := fmt.Sscanf(str, "%d-%d", &rocYear, &month); n == 2 {
		if rocYear >= 1 && month >= 1 && month <= 12 {
			return rocYear + 1911, month, nil
		}
	}
	if n, _ := fmt.Sscanf(str, "%03d%02d", &rocYear, &month); n == 2 {
		if rocYear >= 1 && month >= 1 && month <= 12 {
			return rocYear + 1911, month, nil
		}
	}
	return 0, 0, errors.New("invalid ROC year-month format, expected RRR-MM or RRRMM")
}

