package namenorm

import (
	"regexp"
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

var (
	// Google Sheets 重複欄位後綴正則（例如 " 2", " 3"）
	reGoogleSuffix = regexp.MustCompile(`\s+\d+$`)
	// 開頭序號正則（例如 "1.", "10. ", "123．"）
	reLeadingIndex = regexp.MustCompile(`^\s*\d+[.、．\s]+`)
	// 括號及其內容（包含全形與半形）
	reParentheses = regexp.MustCompile(`\([^)]*\)|（[^）]*）`)
	// 末尾方括號正則
	reBracketSuffix = regexp.MustCompile(`\[([^\]]+)\]\s*$`)
)

// Normalize 對姓名與地址進行 NFKC、去空白、異體字替換與小寫正規化。
func Normalize(s string) string {
	// 1. Unicode NFKC 正規化（全形轉半形等）
	nfkc := norm.NFKC.String(s)

	// 2. 移除所有空白字元
	var noSpace strings.Builder
	for _, r := range nfkc {
		if !unicode.IsSpace(r) {
			noSpace.WriteRune(r)
		}
	}
	str := noSpace.String()

	// 3. 異體字映射
	var variantReplaced strings.Builder
	for _, r := range str {
		if standard, exists := CharacterVariants[r]; exists {
			variantReplaced.WriteRune(standard)
		} else {
			variantReplaced.WriteRune(r)
		}
	}

	// 4. 轉小寫
	return strings.ToLower(variantReplaced.String())
}

// ParsedHeader 代表表單欄名解析後的結構體。
type ParsedHeader struct {
	Original    string
	Direction   string // "outbound", "inbound", 或 ""
	Kind        string // "ride", "meta", "issue", "unknown"
	CleanedName string
}

// ParseColumnHeader 解析 Google 表單之欄名標題並提取方向與乾淨姓名。
func ParseColumnHeader(header string) ParsedHeader {
	orig := header
	header = strings.TrimSpace(header)

	// 系統欄判斷；並存的舊寫法來自 Google 表單匯出檔，仍需辨識以便解析歷史檔案
	switch header {
	case "民國日期", "駕駛人", "時間戳記", "今天日期", "今日駕駛人":
		return ParsedHeader{
			Original: orig,
			Kind:     "meta",
		}
	}
	if strings.Contains(header, "問題回報") || header == "備註" {
		return ParsedHeader{
			Original: orig,
			Kind:     "issue",
		}
	}

	// 1. 去除末尾的 " 2", " 3" 等 Google 重複欄後綴
	header = reGoogleSuffix.ReplaceAllString(header, "")

	// 2. 取末尾方括號內容為方向標記
	direction := ""
	kind := "unknown"
	bracketMatches := reBracketSuffix.FindStringSubmatch(header)
	if len(bracketMatches) > 1 {
		bracketContent := strings.TrimSpace(bracketMatches[1])
		if strings.Contains(bracketContent, "去程") {
			direction = "outbound"
			kind = "ride"
		} else if strings.Contains(bracketContent, "回程") {
			direction = "inbound"
			kind = "ride"
		}
		// 移除方括號
		header = reBracketSuffix.ReplaceAllString(header, "")
	}

	// 3. 去除開頭序號（例如 "1."）
	header = reLeadingIndex.ReplaceAllString(header, "")

	// 4. 去除 * 之後的備註文字
	if idx := strings.Index(header, "*"); idx != -1 {
		header = header[:idx]
	}

	// 5. 去除所有括號及其內容
	header = reParentheses.ReplaceAllString(header, "")

	// 6. 套用 Normalize 取得乾淨姓名
	cleaned := Normalize(header)

	return ParsedHeader{
		Original:    orig,
		Direction:   direction,
		Kind:        kind,
		CleanedName: cleaned,
	}
}

// LevenshteinDistance 計算兩字串之編輯距離。
func LevenshteinDistance(s1, s2 string) int {
	r1, r2 := []rune(s1), []rune(s2)
	n1, n2 := len(r1), len(r2)

	dp := make([][]int, n1+1)
	for i := range dp {
		dp[i] = make([]int, n2+1)
		dp[i][0] = i
	}
	for j := 0; j <= n2; j++ {
		dp[0][j] = j
	}

	for i := 1; i <= n1; i++ {
		for j := 1; j <= n2; j++ {
			cost := 0
			if r1[i-1] != r2[j-1] {
				cost = 1
			}
			minVal := dp[i-1][j] + 1
			if dp[i][j-1]+1 < minVal {
				minVal = dp[i][j-1] + 1
			}
			if dp[i-1][j-1]+cost < minVal {
				minVal = dp[i-1][j-1] + cost
			}
			dp[i][j] = minVal
		}
	}
	return dp[n1][n2]
}

// ScoreCandidate 比對乾淨候選姓名與目標正規化姓名並計算推薦分數。
func ScoreCandidate(cleanedName, targetNormalized string) float64 {
	if cleanedName == "" || targetNormalized == "" {
		return 0.0
	}
	if cleanedName == targetNormalized {
		return 1.0
	}
	if LevenshteinDistance(cleanedName, targetNormalized) <= 1 {
		return 0.6
	}
	return 0.0
}
