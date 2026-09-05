package merge

import "strings"

// ParseReportedValue 將司機匯報表的儲存格文字轉為回報狀態。
//
// 表單狀態只接受明確白名單，其餘（空白、備註誤填）一律視為未回報，不建立來源紀錄。
func ParseReportedValue(raw string) (string, bool) {
	value := strings.TrimSpace(raw)
	switch value {
	case "有坐", "有搭乘":
		return "boarded", true
	case "沒坐", "沒有坐", "未搭乘", "沒有搭乘":
		return "absent", true
	default:
		return "", false
	}
}
