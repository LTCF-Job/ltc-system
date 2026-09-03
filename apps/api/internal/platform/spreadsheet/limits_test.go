package spreadsheet

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLimitCounter(t *testing.T) {
	t.Run("正常規模不應報錯", func(t *testing.T) {
		var c LimitCounter
		require.NoError(t, c.BeginSheet())
		for i := 0; i < 1000; i++ {
			require.NoError(t, c.AddRow("Sheet1", 20))
		}
	})

	t.Run("超過工作表數上限應報錯", func(t *testing.T) {
		var c LimitCounter
		for i := 0; i < MaxSheets; i++ {
			require.NoError(t, c.BeginSheet())
		}
		assert.ErrorContains(t, c.BeginSheet(), "工作表數超過上限")
	})

	t.Run("超過單張工作表列數上限應報錯", func(t *testing.T) {
		var c LimitCounter
		require.NoError(t, c.BeginSheet())
		for i := 0; i < MaxRowsPerSheet; i++ {
			require.NoError(t, c.AddRow("Sheet1", 1))
		}
		assert.ErrorContains(t, c.AddRow("Sheet1", 1), "列數超過上限")
	})

	t.Run("列數上限以單張工作表為單位重新起算", func(t *testing.T) {
		var c LimitCounter
		require.NoError(t, c.BeginSheet())
		require.NoError(t, c.AddRow("Sheet1", 1))
		require.NoError(t, c.BeginSheet())
		require.NoError(t, c.AddRow("Sheet2", 1))
	})

	t.Run("超過總儲存格數上限應報錯", func(t *testing.T) {
		var c LimitCounter
		require.NoError(t, c.BeginSheet())
		assert.ErrorContains(t, c.AddRow("Sheet1", MaxTotalCells+1), "儲存格總數超過上限")
	})
}

func TestTrimTrailingEmptyRows(t *testing.T) {
	rows := [][]string{{"a"}, {"", ""}, {"b"}, {""}, {}}
	assert.Equal(t, [][]string{{"a"}, {"", ""}, {"b"}}, TrimTrailingEmptyRows(rows))
	assert.Empty(t, TrimTrailingEmptyRows([][]string{{""}, {}}))
	assert.Empty(t, TrimTrailingEmptyRows(nil))
}
