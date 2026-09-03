package merge

import "strings"

// ParseReportedValue 將司機匯報表的儲存格文字轉為回報狀態。
//
// 表單只會出現「有坐」「沒坐」兩種用語，其餘（空白、備註誤填）一律視為未回報，
// 不建立來源紀錄——匯入與人工補登共用這一份判斷，避免兩處各自認定造成統計不一致。
func ParseReportedValue(raw string) (string, bool) {
	value := strings.TrimSpace(raw)
	switch {
	case strings.Contains(value, "有坐"):
		return "boarded", true
	case strings.Contains(value, "沒坐"):
		return "absent", true
	default:
		return "", false
	}
}
