package service

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xuri/excelize/v2"
)

func TestParseWeekdays(t *testing.T) {
	tests := []struct {
		input string
		want  []int16
	}{
		{"週一到週五", []int16{1, 2, 3, 4, 5}},
		{"周一到周五", []int16{1, 2, 3, 4, 5}},
		{"週一~週五", []int16{1, 2, 3, 4, 5}},
		{"週四、週五", []int16{4, 5}},
		{"週二，週四下午去回", []int16{2, 4}},
		{"周一早上來回", []int16{1}},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := ParseWeekdays(tt.input)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGenerateCaseImportTemplateExcel_Structure(t *testing.T) {
	excelBytes, err := GenerateCaseImportTemplateExcel()
	require.NoError(t, err)
	require.NotEmpty(t, excelBytes)

	f, err := excelize.OpenReader(bytes.NewReader(excelBytes))
	require.NoError(t, err)
	defer f.Close()

	sheetName := "個案匯入範本"
	sheets := f.GetSheetList()
	assert.Contains(t, sheets, sheetName)

	rows, err := f.GetRows(sheetName)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(rows), 4, "範本應包含 1 列標題與 3 列範例資料")

	headerRow := rows[0]
	assert.Equal(t, "個案姓名*", headerRow[0])
	assert.Equal(t, "身分證字號*", headerRow[1])
	assert.Equal(t, "申報地區*(苗栗/新竹)", headerRow[2])
	assert.Contains(t, headerRow, "週一趟數(0:不搭/1:單去/2:來回/4:四趟)")
	assert.Contains(t, headerRow, "單趟里程(公里)*")
}

func TestGenerateCaseImportTemplateCSV_Structure(t *testing.T) {
	csvData := GenerateCaseImportTemplateCSV()
	assert.True(t, strings.HasPrefix(csvData, "\uFEFF個案姓名*"))
	lines := strings.Split(strings.TrimSpace(csvData), "\r\n")
	require.Equal(t, 4, len(lines), "CSV 範本應包含 1 列標題與 3 筆範例資料")
	assert.Contains(t, lines[0], "週一趟數")
	assert.Contains(t, lines[1], "張曾阿妹")
	assert.Contains(t, lines[2], "李國盛")
	assert.Contains(t, lines[3], "王大同")
}

func TestParseCases_TemplateCSV_DailySchedules(t *testing.T) {
	csvData := GenerateCaseImportTemplateCSV()
	svc := NewImportService(nil, nil, nil, nil, nil, nil)

	preview, err := svc.ParseCases(strings.NewReader(csvData), "template.csv")
	require.NoError(t, err)
	require.NotNil(t, preview)

	assert.Equal(t, 3, preview.TotalRows)
	assert.Equal(t, 3, preview.ValidRows)
	assert.Equal(t, 0, preview.ErrorRows)
	assert.Equal(t, 3, len(preview.PreviewRows))

	// 檢查第一筆：張曾阿妹 (週一至週五 2趟)
	row1 := preview.Rows[0]
	assert.Equal(t, "張曾阿妹", row1.Name)
	assert.Equal(t, "A202559750", row1.NationalID)
	assert.Equal(t, int16(2), row1.TripPattern)
	assert.Equal(t, []int16{1, 2, 3, 4, 5}, row1.Weekdays)

	// 檢查第二筆：李國盛 (週一/三/五 2趟，週二 1趟 -> 每日趟數不同)
	row2 := preview.Rows[1]
	assert.Equal(t, "李國盛", row2.Name)
	assert.Equal(t, []int16{1, 2, 3, 5}, row2.Weekdays)
	assert.NotEmpty(t, row2.WeekdaySchedules)

	// 檢查第三筆：王大同 (週四 4趟)
	row3 := preview.Rows[2]
	assert.Equal(t, "王大同", row3.Name)
	assert.Equal(t, int16(4), row3.TripPattern)
	assert.Equal(t, []int16{4}, row3.Weekdays)
}

func TestParseCases_TemplateExcel(t *testing.T) {
	excelBytes, err := GenerateCaseImportTemplateExcel()
	require.NoError(t, err)
	require.NotEmpty(t, excelBytes)

	svc := NewImportService(nil, nil, nil, nil, nil, nil)
	preview, err := svc.ParseCases(bytes.NewReader(excelBytes), "template.xlsx")
	require.NoError(t, err)
	require.NotNil(t, preview)

	assert.Equal(t, 3, preview.TotalRows)
	assert.Equal(t, 3, preview.ValidRows)
	assert.Equal(t, 0, preview.ErrorRows)
	assert.Equal(t, 3, len(preview.PreviewRows))
}

func TestParseCases_ProfileWorkbook(t *testing.T) {
	f := excelize.NewFile()
	defer f.Close()
	sheetName := "進系統個案個資"
	f.SetSheetName("Sheet1", sheetName)
	headers := []string{"序號", "姓名", "戶別", "身分證字號", "性別", "生日", "歲數", "據點", "接送車輛(去)", "接送車輛(回)", "個管or照專", "聯絡人", "戶籍", "居住地", "REMARK"}
	for i, header := range headers {
		cell, err := excelize.CoordinatesToCellName(i+1, 1)
		require.NoError(t, err)
		require.NoError(t, f.SetCellValue(sheetName, cell, header))
	}
	row := []interface{}{1, "王小明", "一般", "A202559750", "男", "045/06/15", 70, "竹南日照", "竹南1車", "竹南2車", "個管", "陳小華", "苗栗縣竹南鎮戶籍地址", "苗栗縣竹南鎮居住地址", "需輪椅"}
	for i, value := range row {
		cell, err := excelize.CoordinatesToCellName(i+1, 2)
		require.NoError(t, err)
		require.NoError(t, f.SetCellValue(sheetName, cell, value))
	}

	buf, err := f.WriteToBuffer()
	require.NoError(t, err)

	svc := NewImportService(nil, nil, nil, nil, nil, nil)
	preview, err := svc.ParseCases(bytes.NewReader(buf.Bytes()), "彙整-個案資料(竹南.頭份).xlsx")
	require.NoError(t, err)
	require.Len(t, preview.Rows, 1)

	got := preview.Rows[0]
	assert.True(t, got.IsProfileWorkbook)
	assert.Equal(t, "一般", got.HouseholdType)
	assert.Equal(t, "男", got.Gender)
	assert.Equal(t, "1956-06-15", got.BirthDate)
	assert.Equal(t, "個管", got.CareContactRole)
	assert.Equal(t, "陳小華", got.CareContactName)
	assert.Equal(t, "苗栗縣竹南鎮戶籍地址", got.RegisteredAddress)
	assert.Equal(t, "苗栗縣竹南鎮居住地址", got.HomeAddress)
	assert.Equal(t, "竹南1車", got.OutboundVehicle)
	assert.Equal(t, "竹南2車", got.InboundVehicle)
	assert.Equal(t, "miaoli", got.Region)
	assert.True(t, got.IsDraft)
	assert.Empty(t, got.Weekdays)
}

func TestParseCases_ProfileWorkbookReportsSkippedFields(t *testing.T) {
	f := excelize.NewFile()
	defer f.Close()
	sheetName := "進系統個案個資"
	f.SetSheetName("Sheet1", sheetName)
	headers := []string{"姓名", "戶別", "身分證字號", "性別", "生日", "據點", "接送車輛(去)", "接送車輛(回)", "個管or照專", "聯絡人", "戶籍", "居住地"}
	for i, header := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		require.NoError(t, f.SetCellValue(sheetName, cell, header))
	}
	values := []interface{}{"馮玉英", "", "A202559750", "女", "錯誤生日", "竹南日照", "竹南1車", "竹南2車", "個管", "陳小華", "戶籍地址", "居住地址"}
	for i, value := range values {
		cell, _ := excelize.CoordinatesToCellName(i+1, 2)
		require.NoError(t, f.SetCellValue(sheetName, cell, value))
	}
	buf, err := f.WriteToBuffer()
	require.NoError(t, err)

	preview, err := NewImportService(nil, nil, nil, nil, nil, nil).ParseCases(bytes.NewReader(buf.Bytes()), "profile.xlsx")
	require.NoError(t, err)
	require.Len(t, preview.Rows, 1)
	assert.Equal(t, 1, preview.ErrorRows)
	assert.Contains(t, preview.Rows[0].ErrorMessage, "戶別：空白")
	assert.Contains(t, preview.Rows[0].ErrorMessage, "生日：格式錯誤")
	assert.Equal(t, "", preview.Rows[0].RawValues["戶別"])
	assert.Equal(t, "錯誤生日", preview.Rows[0].RawValues["生日"])
}

func TestParseCases_EmptyAndCorruptedFiles(t *testing.T) {
	svc := NewImportService(nil, nil, nil, nil, nil, nil)

	// 測試空白 CSV 應回傳錯誤
	_, err := svc.ParseCases(strings.NewReader(""), "empty.csv")
	assert.Error(t, err, "空白 CSV 應回傳錯誤")

	// 測試損毀的 Excel 檔案
	corrupted := []byte{0x50, 0x4B, 0x03, 0x04, 0x00, 0x00, 0x00}
	_, err = svc.ParseCases(bytes.NewReader(corrupted), "bad.xlsx")
	assert.Error(t, err, "損毀的 Excel 應回傳錯誤")
}

func TestParseCasesFromExcel_RealFile(t *testing.T) {
	filePath := filepath.Join("..", "..", "..", "..", "source", "個案新增資料.xlsx")
	f, err := os.Open(filePath)
	if err != nil {
		t.Skip("Sample file not found, skipping real file test")
		return
	}
	defer f.Close()

	svc := NewImportService(nil, nil, nil, nil, nil, nil)
	preview, err := svc.ParseCasesFromExcel(f)
	require.NoError(t, err)
	require.NotNil(t, preview)

	assert.Greater(t, preview.TotalRows, 0)
	assert.Equal(t, preview.TotalRows, preview.ValidRows)
	t.Logf("Parsed %d case rows, %d valid, %d with warnings/draft",
		preview.TotalRows, preview.ValidRows, preview.WarningRows)
}
