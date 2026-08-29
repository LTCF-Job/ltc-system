package app

import (
	"errors"
	"regexp"
	"strings"

	"golang.org/x/text/unicode/norm"
)

var reDriverHeader = regexp.MustCompile(`^(.+?)\((\S+)\s+([A-Z][0-9]{9})\)$`)

// stringPointer 將空字串轉為 nil，供選填欄位寫入時使用。
func stringPointer(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

// ParseWeekdays 解析「每週據點開放時間」自由文字（規格書 6.2）。
func ParseWeekdays(s string) ([]int16, error) {
	s = norm.NFKC.String(s)
	s = strings.ReplaceAll(s, " ", "")
	s = strings.ReplaceAll(s, "\t", "")
	s = strings.ReplaceAll(s, "周", "週")

	dayMap := map[rune]int16{
		'一': 1, '二': 2, '三': 3, '四': 4, '五': 5, '六': 6, '日': 7, '天': 7,
	}

	rangeSeparators := []string{"到", "~", "-"}
	for _, sep := range rangeSeparators {
		if strings.Contains(s, sep) {
			parts := strings.Split(s, sep)
			if len(parts) == 2 {
				r1 := []rune(parts[0])
				r2 := []rune(parts[1])
				var startDay, endDay int16
				for _, r := range r1 {
					if d, ok := dayMap[r]; ok {
						startDay = d
					}
				}
				for _, r := range r2 {
					if d, ok := dayMap[r]; ok {
						endDay = d
					}
				}
				if startDay > 0 && endDay >= startDay {
					var res []int16
					for d := startDay; d <= endDay; d++ {
						res = append(res, d)
					}
					return res, nil
				}
			}
		}
	}

	var res []int16
	seen := make(map[int16]bool)
	runes := []rune(s)
	for i := 0; i < len(runes); i++ {
		if runes[i] == '週' && i+1 < len(runes) {
			if d, ok := dayMap[runes[i+1]]; ok {
				if !seen[d] {
					seen[d] = true
					res = append(res, d)
				}
			}
		}
	}

	if len(res) == 0 {
		return nil, errors.New("cannot parse weekdays from text")
	}

	return res, nil
}
