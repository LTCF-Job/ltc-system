package app

import (
	"regexp"
	"strings"
)

// ExtractSpreadsheetID 從 Google 試算表完整 URL 或 ID 中擷取純 spreadsheetId。
func ExtractSpreadsheetID(input string) string {
	input = strings.TrimSpace(input)
	if input == "" {
		return ""
	}

	// 匹配 https://docs.google.com/spreadsheets/d/{ID}/...
	re := regexp.MustCompile(`/spreadsheets/d/([a-zA-Z0-9-_]+)`)
	matches := re.FindStringSubmatch(input)
	if len(matches) > 1 {
		return matches[1]
	}

	// 若非網址格式且不含斜線，視為直接傳入 ID
	if !strings.Contains(input, "/") && !strings.Contains(input, "?") {
		return input
	}

	return input
}
