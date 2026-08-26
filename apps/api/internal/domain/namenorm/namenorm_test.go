package namenorm

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNormalize(t *testing.T) {
	// 異體字相符
	assert.Equal(t, Normalize("劉温月妹"), Normalize("劉溫月妹"))
	assert.Equal(t, "劉溫月妹", Normalize("劉温月妹"))

	// 罕用字不損壞 (𣵛, U+23D5B)
	assert.Equal(t, "吳𣵛桂", Normalize(" 吳𣵛桂 "))

	// 全形轉半形與去除空格
	assert.Equal(t, "新竹縣竹北市光明六路264號", Normalize("新竹縣竹北市光明六路 ２６４ 號"))

	// 英文小寫轉換
	assert.Equal(t, "abc123", Normalize(" ABC １２３ "))
}

func TestParseColumnHeader(t *testing.T) {
	tests := []struct {
		name          string
		header        string
		wantKind      string
		wantDirection string
		wantCleaned   string
	}{
		{
			name:          "一般去程序號與空白",
			header:        "1. 張詹竹妹 [去程]",
			wantKind:      "ride",
			wantDirection: "outbound",
			wantCleaned:   "張詹竹妹",
		},
		{
			name:          "Google 重複欄位後綴",
			header:        "2.曾細嬌 [去程] 2",
			wantKind:      "ride",
			wantDirection: "outbound",
			wantCleaned:   "曾細嬌",
		},
		{
			name:          "姓名後雙空格",
			header:        "3.鄧甜妹  [去程]",
			wantKind:      "ride",
			wantDirection: "outbound",
			wantCleaned:   "鄧甜妹",
		},
		{
			name:          "罕用字加括號業務註記",
			header:        "1.吳𣵛桂(去程竹3) [去程]",
			wantKind:      "ride",
			wantDirection: "outbound",
			wantCleaned:   "吳𣵛桂",
		},
		{
			name:          "星號註記與(早午)去程標記",
			header:        "1.陳素貞(4趟)*可能下午回竹1車載 [(早午)去程]",
			wantKind:      "ride",
			wantDirection: "outbound",
			wantCleaned:   "陳素貞",
		},
		{
			name:          "(早午)回程",
			header:        "6.顏湯月淑 (改2趟) *可能下午回,竹1車載 [(早午)回程]",
			wantKind:      "ride",
			wantDirection: "inbound",
			wantCleaned:   "顏湯月淑",
		},
		{
			name:          "分號星號複合註記",
			header:        "8.葉秀珍 (4趟) *早去午回竹3載;下午回竹1車載 [去程]",
			wantKind:      "ride",
			wantDirection: "outbound",
			wantCleaned:   "葉秀珍",
		},
		{
			name:          "異常方向標記 [第 3 列]",
			header:        "1.曾蕭碧雲(W1.5下午回程竹1車載) [第 3 列]",
			wantKind:      "unknown",
			wantDirection: "",
			wantCleaned:   "曾蕭碧雲",
		},
		{
			name:        "系統欄 時間戳記",
			header:      "時間戳記",
			wantKind:    "meta",
			wantCleaned: "",
		},
		{
			name:        "系統欄 今天日期",
			header:      "今天日期",
			wantKind:    "meta",
			wantCleaned: "",
		},
		{
			name:        "系統欄 今日駕駛人",
			header:      "今日駕駛人",
			wantKind:    "meta",
			wantCleaned: "",
		},
		{
			name:        "問題回報欄",
			header:      "問題回報",
			wantKind:    "issue",
			wantCleaned: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed := ParseColumnHeader(tt.header)
			assert.Equal(t, tt.wantKind, parsed.Kind)
			assert.Equal(t, tt.wantDirection, parsed.Direction)
			if tt.wantCleaned != "" {
				assert.Equal(t, tt.wantCleaned, parsed.CleanedName)
			}
		})
	}
}

func TestScoreCandidate(t *testing.T) {
	// 完全相符
	assert.Equal(t, 1.0, ScoreCandidate("張詹竹妹", "張詹竹妹"))
	// 編輯距離 1
	assert.Equal(t, 0.6, ScoreCandidate("張詹竹妹", "張詹竹"))
	// 不相符
	assert.Equal(t, 0.0, ScoreCandidate("王大明", "張詹竹妹"))
}
