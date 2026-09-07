package rocdate

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	ErrBeforeROCYear    = errors.New("date is before ROC year 1 (1912-01-01)")
	ErrInvalidROCVal    = errors.New("invalid ROC date integer")
	ErrInvalidDate      = errors.New("invalid date")
	ErrInvalidYearMonth = errors.New("invalid year-month format")
)

var yearMonthPattern = regexp.MustCompile(`^(?:(\d{3})-?(\d{2})|(\d{4})-(\d{2}))$`)

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
	if ceYear > 2100 {
		return time.Time{}, ErrInvalidROCVal
	}
	t := time.Date(ceYear, time.Month(month), day, 0, 0, 0, 0, time.UTC)

	// 檢查日期溢位（例如 2 月 30 日或非閏年 2 月 29 日）
	if t.Year() != ceYear || int(t.Month()) != month || t.Day() != day {
		return time.Time{}, ErrInvalidROCVal
	}

	return t, nil
}

// ParseDate 嚴格解析業務日期，接受民國 RRRMMDD／RRR/MM/DD／RRR-MM-DD，
// 以及西元 YYYY-MM-DD／YYYY/MM/DD／YYYY.MM.DD。回傳值固定為 UTC 的日期零點，
// 讓日期欄位不因執行環境時區而跨日。
func ParseDate(raw string) (time.Time, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return time.Time{}, ErrInvalidDate
	}

	if digits, err := strconv.Atoi(value); err == nil {
		if len(value) != 7 {
			return time.Time{}, ErrInvalidDate
		}
		parsed, err := FromROC(digits)
		if err != nil {
			return time.Time{}, ErrInvalidDate
		}
		return parsed, nil
	}

	separator := ""
	for _, candidate := range []string{"/", "-", "."} {
		if strings.Contains(value, candidate) {
			separator = candidate
			break
		}
	}
	if separator == "" {
		return time.Time{}, ErrInvalidDate
	}
	parts := strings.Split(value, separator)
	if len(parts) != 3 {
		return time.Time{}, ErrInvalidDate
	}
	year, yearErr := strconv.Atoi(strings.TrimSpace(parts[0]))
	month, monthErr := strconv.Atoi(strings.TrimSpace(parts[1]))
	day, dayErr := strconv.Atoi(strings.TrimSpace(parts[2]))
	if yearErr != nil || monthErr != nil || dayErr != nil {
		return time.Time{}, ErrInvalidDate
	}

	if len(strings.TrimSpace(parts[0])) == 3 {
		parsed, err := FromROC(year*10000 + month*100 + day)
		if err != nil {
			return time.Time{}, ErrInvalidDate
		}
		return parsed, nil
	}
	if len(strings.TrimSpace(parts[0])) != 4 || year < 1912 || year > 2100 {
		return time.Time{}, ErrInvalidDate
	}
	parsed := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
	if parsed.Year() != year || int(parsed.Month()) != month || parsed.Day() != day {
		return time.Time{}, ErrInvalidDate
	}
	return parsed, nil
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
	str = strings.TrimSpace(str)
	if len(str) == 6 && str[3] == '-' {
		str = str[:3] + str[4:]
	}
	if len(str) != 5 {
		return 0, 0, errors.New("invalid ROC year-month format, expected RRR-MM or RRRMM")
	}
	rocYear, err := strconv.Atoi(str[:3])
	if err != nil {
		return 0, 0, errors.New("invalid ROC year-month format, expected RRR-MM or RRRMM")
	}
	month, err := strconv.Atoi(str[3:])
	if err != nil || rocYear < 1 || month < 1 || month > 12 {
		return 0, 0, errors.New("invalid ROC year-month format, expected RRR-MM or RRRMM")
	}
	return rocYear + 1911, month, nil
}

// ParseYearMonth 嚴格解析民國 RRR-MM／RRRMM 或西元 YYYY-MM，拒絕不完整與
// 不存在的月份，不替非法輸入猜測目前月份。
func ParseYearMonth(raw string) (int, int, error) {
	matches := yearMonthPattern.FindStringSubmatch(strings.TrimSpace(raw))
	if matches == nil {
		return 0, 0, ErrInvalidYearMonth
	}

	var year, month int
	if matches[1] != "" {
		rocYear, err := strconv.Atoi(matches[1])
		if err != nil {
			return 0, 0, ErrInvalidYearMonth
		}
		year = rocYear + 1911
		month, err = strconv.Atoi(matches[2])
		if err != nil {
			return 0, 0, ErrInvalidYearMonth
		}
	} else {
		var err error
		year, err = strconv.Atoi(matches[3])
		if err != nil {
			return 0, 0, ErrInvalidYearMonth
		}
		month, err = strconv.Atoi(matches[4])
		if err != nil {
			return 0, 0, ErrInvalidYearMonth
		}
	}

	const maxGregorianYear = 2100
	if year < 1912 || year > maxGregorianYear || month < 1 || month > 12 {
		return 0, 0, ErrInvalidYearMonth
	}
	return year, month, nil
}

// MonthRangeStrict 回傳指定月份的左閉右開 UTC 日期範圍。
func MonthRangeStrict(periodYM string) (start, end time.Time, daysInMonth int, err error) {
	year, month, err := ParseYearMonth(periodYM)
	if err != nil {
		return time.Time{}, time.Time{}, 0, err
	}
	start = time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	end = start.AddDate(0, 1, 0)
	return start, end, end.AddDate(0, 0, -1).Day(), nil
}

// MonthRange 嚴格解析期別並回傳左閉右開的 UTC 日期範圍；非法期別一律回傳錯誤，
// 不得默默改用目前月份。
func MonthRange(periodYM string) (start, end time.Time, daysInMonth int, err error) {
	return MonthRangeStrict(periodYM)
}
