package infra

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xuri/excelize/v2"
	"ltc-system/apps/api/internal/platform/spreadsheet"
)

// TestRenderCaregiverImportTemplate_ReopensCleanly 依 excel-import-export-integrity
// skill 的要求：render 函式產生的位元組必須能被重新解析，而非只檢查寫入時無錯誤。
func TestRenderCaregiverImportTemplate_ReopensCleanly(t *testing.T) {
	data, err := NewExcelAdapter().RenderCaregiverImportTemplate()
	require.NoError(t, err)
	require.NotEmpty(t, data)

	f, err := excelize.OpenReader(bytes.NewReader(data))
	require.NoError(t, err, "範本位元組必須是可重新開啟的合法 xlsx")
	defer f.Close()

	sheetName := "照護人員匯入範本"
	assert.Contains(t, f.GetSheetList(), sheetName)

	rows, err := f.GetRows(sheetName)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(rows), 3, "範本應包含 1 列標題與至少 2 列範例資料")

	headerRow := rows[0]
	assert.Equal(t, "類型*", headerRow[0])
	assert.Equal(t, "單位", headerRow[1])
	assert.Equal(t, "姓名*", headerRow[2])
	assert.Contains(t, rows[1], "個管")
	assert.Contains(t, rows[2], "專護")
}

// TestReadTables_Limits 驗證解析規模上限確實接在 ReadTables 的讀取路徑上：
// 超限必須回報錯誤，而非靜默截斷或吃光記憶體。
func TestReadTables_Limits(t *testing.T) {
	adapter := NewExcelAdapter()

	t.Run("正常檔案應完整讀出且去除尾端空白列", func(t *testing.T) {
		f := excelize.NewFile()
		defer f.Close()
		require.NoError(t, f.SetCellValue("Sheet1", "A1", "姓名"))
		require.NoError(t, f.SetCellValue("Sheet1", "B1", "單位"))
		require.NoError(t, f.SetCellValue("Sheet1", "A2", "陳小華"))
		require.NoError(t, f.SetCellValue("Sheet1", "A5", ""))
		buf, err := f.WriteToBuffer()
		require.NoError(t, err)

		tables, names, err := adapter.ReadTables(buf.Bytes())
		require.NoError(t, err)
		require.Len(t, tables, 1)
		assert.Equal(t, []string{"Sheet1"}, names)
		require.Len(t, tables[0], 2)
		assert.Equal(t, []string{"姓名", "單位"}, tables[0][0])
		assert.Equal(t, "陳小華", tables[0][1][0])
	})

	t.Run("工作表數超過上限應回報錯誤", func(t *testing.T) {
		f := excelize.NewFile()
		defer f.Close()
		require.NoError(t, f.SetCellValue("Sheet1", "A1", "x"))
		for i := 2; i <= spreadsheet.MaxSheets+1; i++ {
			name := fmt.Sprintf("Sheet%d", i)
			_, err := f.NewSheet(name)
			require.NoError(t, err)
			require.NoError(t, f.SetCellValue(name, "A1", "x"))
		}
		buf, err := f.WriteToBuffer()
		require.NoError(t, err)

		_, _, err = adapter.ReadTables(buf.Bytes())
		require.ErrorContains(t, err, "工作表數超過上限")
	})

	t.Run("列數超過上限應回報錯誤", func(t *testing.T) {
		f := excelize.NewFile()
		defer f.Close()
		sw, err := f.NewStreamWriter("Sheet1")
		require.NoError(t, err)
		for r := 1; r <= spreadsheet.MaxRowsPerSheet+1; r++ {
			cell, err := excelize.CoordinatesToCellName(1, r)
			require.NoError(t, err)
			require.NoError(t, sw.SetRow(cell, []interface{}{"x"}))
		}
		require.NoError(t, sw.Flush())
		buf, err := f.WriteToBuffer()
		require.NoError(t, err)

		_, _, err = adapter.ReadTables(buf.Bytes())
		require.ErrorContains(t, err, "列數超過上限")
	})
}
