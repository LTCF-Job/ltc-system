package google_test

import (
	"context"
	"testing"

	"ltc-system/apps/api/internal/adapter/google"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
			actual := google.ExtractSpreadsheetID(tt.input)
			assert.Equal(t, tt.expected, actual)
		})
	}
}

func TestOfflineGoogleClient(t *testing.T) {
	ctx := context.Background()
	client, err := google.NewClient(ctx, "")
	require.NoError(t, err)
	require.NotNil(t, client)

	t.Run("離線模式列出試算表清單", func(t *testing.T) {
		sheets, err := client.ListDriveSheets(ctx)
		require.NoError(t, err)
		assert.NotEmpty(t, sheets)
		assert.Contains(t, sheets[0].Name, "竹北一車")
	})

	t.Run("離線模式解析試算表結構", func(t *testing.T) {
		info, err := client.GetSpreadsheetInfo(ctx, "https://docs.google.com/spreadsheets/d/demo-id/edit", "")
		require.NoError(t, err)
		assert.Equal(t, "demo-id", info.SpreadsheetID)
		assert.NotEmpty(t, info.SheetTabs)
	})

	t.Run("離線模式讀取資料列", func(t *testing.T) {
		rows, err := client.ReadSheetRows(ctx, "demo-id", "8月回報", "")
		require.NoError(t, err)
		assert.NotEmpty(t, rows)
		assert.Equal(t, "時間戳記", rows[0][0])
	})
}
