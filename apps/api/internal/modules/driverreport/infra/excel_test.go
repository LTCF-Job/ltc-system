package infra

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xuri/excelize/v2"
)

// referenceWorkbook 是使用者提供的實際 Google 表單回覆匯出檔，用來確認解析器
// 面對真實資料（106 列、備註欄夾在中間、民國日期格式）的行為。
const referenceWorkbook = "../../../../../../docs/source/竹南2車 (回覆).xlsx"

func TestRenderDriverReportTemplate_RoundTrip(t *testing.T) {
	adapter := NewExcelAdapter()
	caseColumns := []string{"1.吳桂(去程竹3) [去程]", "1.吳桂(去程竹3) [回程]"}

	data, err := adapter.RenderDriverReportTemplate("竹南2車", caseColumns)
	require.NoError(t, err)
	require.Greater(t, len(data), 1000)

	f, err := excelize.OpenReader(bytes.NewReader(data))
	require.NoError(t, err, "產生的位元組必須能被試算表解析器重新開啟")
	defer f.Close()

	rows, err := f.GetRows(driverReportSheetName)
	require.NoError(t, err)
	require.Len(t, rows, 1, "範本不得含示範資料列，否則匯入時會被當成真實匯報寫入")

	assert.Equal(t,
		[]string{"民國日期", "駕駛人", "1.吳桂(去程竹3) [去程]", "1.吳桂(去程竹3) [回程]", "備註"},
		rows[0],
	)
}

func TestRenderDriverReportTemplate_WithoutMappedColumns(t *testing.T) {
	data, err := NewExcelAdapter().RenderDriverReportTemplate("竹北一車", nil)
	require.NoError(t, err)

	f, err := excelize.OpenReader(bytes.NewReader(data))
	require.NoError(t, err)
	defer f.Close()

	rows, err := f.GetRows(driverReportSheetName)
	require.NoError(t, err)
	assert.Equal(t, []string{"民國日期", "駕駛人", "備註"}, rows[0])
}

func TestReadTables_ReferenceWorkbook(t *testing.T) {
	data, err := os.ReadFile(filepath.FromSlash(referenceWorkbook))
	if err != nil {
		t.Skipf("找不到參考檔 %s，略過真實檔案解析測試", referenceWorkbook)
	}

	tables, sheetNames, err := NewExcelAdapter().ReadTables(data)
	require.NoError(t, err)
	require.NotEmpty(t, tables)
	require.NotEmpty(t, sheetNames)

	header := tables[0][0]
	assert.Equal(t, "時間戳記", header[0])
	assert.Equal(t, "今天日期", header[1])
	assert.Equal(t, "今日駕駛人", header[2])

	// 民國日期欄在原檔以 "115"mmdd 的數值格式呈現，excelize 讀出的是格式化後的字串。
	assert.Equal(t, "1150302", tables[0][1][1])
	assert.Equal(t, "林彥衡", tables[0][1][2])
	assert.Contains(t, []string{"有坐", "沒坐"}, tables[0][1][3])
}
