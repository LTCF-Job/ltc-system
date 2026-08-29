package app_test

import (
	"testing"

	"ltc-system/apps/api/internal/modules/formsync/app"

	"github.com/stretchr/testify/assert"
)

func TestExtractSpreadsheetID(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "標準完整 Google 試算表編輯網址",
			input:    "https://docs.google.com/spreadsheets/d/1BxiMVs0XRA5nFMdKvBdBZjgmUUqptlbs74OgvE2upms/edit#gid=0",
			expected: "1BxiMVs0XRA5nFMdKvBdBZjgmUUqptlbs74OgvE2upms",
		},
		{
			name:     "簡短 Google 試算表網址",
			input:    "https://docs.google.com/spreadsheets/d/1BxiMVs0XRA5nFMdKvBdBZjgmUUqptlbs74OgvE2upms",
			expected: "1BxiMVs0XRA5nFMdKvBdBZjgmUUqptlbs74OgvE2upms",
		},
		{
			name:     "純試算表 ID",
			input:    "1BxiMVs0XRA5nFMdKvBdBZjgmUUqptlbs74OgvE2upms",
			expected: "1BxiMVs0XRA5nFMdKvBdBZjgmUUqptlbs74OgvE2upms",
		},
		{
			name:     "前後含空白字元之網址",
			input:    "   https://docs.google.com/spreadsheets/d/test-sheet-id-123/edit   ",
			expected: "test-sheet-id-123",
		},
		{
			name:     "空字串",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := app.ExtractSpreadsheetID(tt.input)
			assert.Equal(t, tt.expected, actual)
		})
	}
}
