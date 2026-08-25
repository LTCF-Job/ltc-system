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
