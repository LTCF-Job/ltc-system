package infra

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xuri/excelize/v2"
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
