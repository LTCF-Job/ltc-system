package app

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xuri/excelize/v2"
	importinfra "ltc-system/apps/api/internal/modules/caseimport/infra"
)

func TestGenerateCaseImportTemplateExcel_Structure(t *testing.T) {
	excelBytes, err := importinfra.NewExcelAdapter().RenderCaseImportTemplate()
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
	require.GreaterOrEqual(t, len(rows), 3, "範本應包含 1 列標題與至少 2 列範例資料")

	headerRow := rows[0]
	assert.Equal(t, "姓名*", headerRow[0])
	assert.Contains(t, headerRow, "身分證字號")
	assert.Contains(t, headerRow, "據點")
	assert.Contains(t, headerRow, "接送車輛(去)")
	assert.Contains(t, headerRow, "接送車輛(回)")
	assert.Contains(t, headerRow, "姓名(個管/照專)")
	assert.Contains(t, headerRow, "REMARK")
	assert.NotContains(t, headerRow, "週一趟數(0:不搭/1:單去/2:來回/4:四趟)")
}

func TestParseCases_TemplateExcel(t *testing.T) {
	excelBytes, err := importinfra.NewExcelAdapter().RenderCaseImportTemplate()
	require.NoError(t, err)
	require.NotEmpty(t, excelBytes)

	svc := NewImportService(nil, nil, nil, nil, nil, importinfra.NewExcelAdapter(), importinfra.NewExcelAdapter(), nil)
	preview, err := svc.ParseCases(context.Background(), bytes.NewReader(excelBytes), "template.xlsx")
	require.NoError(t, err)
	require.NotNil(t, preview)

	assert.Equal(t, 2, preview.TotalRows)
	assert.Equal(t, 2, preview.ValidRows)
	assert.Equal(t, 0, preview.ErrorRows)
}

// TestParseCases_ProfileWorkbook 驗證表頭「姓名」出現兩次時，第一個對應個案姓名，
// 出現在「個管or照專」欄之後的第二個對應個管/照專姓名，不互相覆蓋。
func TestParseCases_ProfileWorkbook(t *testing.T) {
	f := excelize.NewFile()
	defer f.Close()
	sheetName := "進系統個案個資"
	f.SetSheetName("Sheet1", sheetName)
	headers := []string{"序號", "姓名", "戶別", "身分證字號", "性別", "生日", "歲數", "據點", "接送車輛(去)", "接送車輛(回)", "個管or照專", "姓名", "戶籍", "居住地", "REMARK"}
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

	svc := NewImportService(nil, nil, nil, nil, nil, importinfra.NewExcelAdapter(), importinfra.NewExcelAdapter(), nil)
	preview, err := svc.ParseCases(context.Background(), bytes.NewReader(buf.Bytes()), "彙整-個案資料(竹南.頭份).xlsx")
	require.NoError(t, err)
	require.Len(t, preview.Rows, 1)

	got := preview.Rows[0]
	assert.Equal(t, "王小明", got.Name)
	assert.Equal(t, "一般", got.HouseholdType)
	assert.Equal(t, "男", got.Gender)
	assert.Equal(t, "1956-06-15", got.BirthDate)
	assert.Equal(t, "個管", got.CareContactRole)
	assert.Equal(t, "陳小華", got.CareContactName)
	assert.Equal(t, "苗栗縣竹南鎮戶籍地址", got.RegisteredAddress)
	assert.Equal(t, "苗栗縣竹南鎮居住地址", got.HomeAddress)
	assert.Equal(t, "竹南1車", got.OutboundVehicle)
	assert.Equal(t, "竹南2車", got.InboundVehicle)
	assert.Equal(t, "需輪椅", got.Remarks)
	assert.False(t, got.IsDuplicate)
}

// TestParseCases_OnlyNameRequired 驗證除姓名外全部欄位皆選填，缺漏不再擋錯。
func TestParseCases_OnlyNameRequired(t *testing.T) {
	f := excelize.NewFile()
	defer f.Close()
	sheetName := "進系統個案個資"
	f.SetSheetName("Sheet1", sheetName)
	headers := []string{"姓名", "戶別", "身分證字號", "性別", "生日", "據點", "接送車輛(去)", "接送車輛(回)", "個管or照專", "姓名", "戶籍", "居住地", "REMARK"}
	for i, header := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		require.NoError(t, f.SetCellValue(sheetName, cell, header))
	}
	values := []interface{}{"馮玉英", "", "", "", "", "", "", "", "", "", "", "", ""}
	for i, value := range values {
		cell, _ := excelize.CoordinatesToCellName(i+1, 2)
		require.NoError(t, f.SetCellValue(sheetName, cell, value))
	}
	buf, err := f.WriteToBuffer()
	require.NoError(t, err)

	preview, err := NewImportService(nil, nil, nil, nil, nil, importinfra.NewExcelAdapter(), importinfra.NewExcelAdapter(), nil).ParseCases(context.Background(), bytes.NewReader(buf.Bytes()), "profile.xlsx")
	require.NoError(t, err)
	require.Len(t, preview.Rows, 1)
	assert.Equal(t, 0, preview.ErrorRows)
	assert.Equal(t, 1, preview.ValidRows)
	assert.Empty(t, preview.Rows[0].ErrorMessage)
}

// TestParseCases_IgnoresFullyBlankRow 驗證全空白列（僅姓名以外欄位有值時仍視為空白）
// 直接忽略，不計入總筆數也不歸入錯誤列。
func TestParseCases_IgnoresFullyBlankRow(t *testing.T) {
	f := excelize.NewFile()
	defer f.Close()
	sheetName := "進系統個案個資"
	f.SetSheetName("Sheet1", sheetName)
	headers := []string{"姓名", "戶別", "身分證字號", "性別", "生日", "據點"}
	for i, header := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		require.NoError(t, f.SetCellValue(sheetName, cell, header))
	}
	blankRow := []interface{}{"", "", "", "", "", ""}
	for i, value := range blankRow {
		cell, _ := excelize.CoordinatesToCellName(i+1, 2)
		require.NoError(t, f.SetCellValue(sheetName, cell, value))
	}
	validRow := []interface{}{"馮玉英", "一般", "", "", "", "竹南日照"}
	for i, value := range validRow {
		cell, _ := excelize.CoordinatesToCellName(i+1, 3)
		require.NoError(t, f.SetCellValue(sheetName, cell, value))
	}
	buf, err := f.WriteToBuffer()
	require.NoError(t, err)

	preview, err := NewImportService(nil, nil, nil, nil, nil, importinfra.NewExcelAdapter(), importinfra.NewExcelAdapter(), nil).ParseCases(context.Background(), bytes.NewReader(buf.Bytes()), "profile.xlsx")
	require.NoError(t, err)
	assert.Equal(t, 1, preview.TotalRows, "全空白列不應計入總筆數")
	assert.Equal(t, 1, preview.ValidRows)
	assert.Equal(t, 0, preview.ErrorRows, "全空白列應直接忽略，不應歸入錯誤列")
	require.Len(t, preview.Rows, 1)
	assert.Equal(t, "馮玉英", preview.Rows[0].Name)
}

// TestParseCases_ReportsBirthDateFormatError 驗證生日格式錯誤仍回報 warning/error，其餘欄位不擋。
func TestParseCases_ReportsBirthDateFormatError(t *testing.T) {
	f := excelize.NewFile()
	defer f.Close()
	sheetName := "進系統個案個資"
	f.SetSheetName("Sheet1", sheetName)
	headers := []string{"姓名", "戶別", "身分證字號", "性別", "生日", "據點", "接送車輛(去)", "接送車輛(回)", "個管or照專", "姓名", "戶籍", "居住地"}
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

	preview, err := NewImportService(nil, nil, nil, nil, nil, importinfra.NewExcelAdapter(), importinfra.NewExcelAdapter(), nil).ParseCases(context.Background(), bytes.NewReader(buf.Bytes()), "profile.xlsx")
	require.NoError(t, err)
	require.Len(t, preview.Rows, 1)
	assert.Equal(t, 1, preview.ErrorRows)
	assert.Contains(t, preview.Rows[0].ErrorMessage, "生日：格式錯誤")
	assert.Equal(t, "", preview.Rows[0].RawValues["戶別"])
	assert.Equal(t, "錯誤生日", preview.Rows[0].RawValues["生日"])
}

// TestParseCases_FlagsDuplicateByNationalID 驗證身分證字號比對到既有個案時標記為重複，
// 但不列為錯誤（仍計入 validRows）。
func TestParseCases_FlagsDuplicateByNationalID(t *testing.T) {
	f := excelize.NewFile()
	defer f.Close()
	sheetName := "進系統個案個資"
	f.SetSheetName("Sheet1", sheetName)
	headers := []string{"姓名", "身分證字號"}
	for i, header := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		require.NoError(t, f.SetCellValue(sheetName, cell, header))
	}
	require.NoError(t, f.SetCellValue(sheetName, "A2", "王小明"))
	require.NoError(t, f.SetCellValue(sheetName, "B2", "A202559750"))
	buf, err := f.WriteToBuffer()
	require.NoError(t, err)

	dupCaseID := uuid.New()
	finder := fakeDuplicateFinder{byNationalID: map[string]*DuplicateRef{
		"A202559750": {CaseID: dupCaseID, CaseCode: "C0001"},
	}}

	svc := NewImportService(nil, finder, nil, nil, nil, importinfra.NewExcelAdapter(), importinfra.NewExcelAdapter(), nil)
	preview, err := svc.ParseCases(context.Background(), bytes.NewReader(buf.Bytes()), "profile.xlsx")
	require.NoError(t, err)
	require.Len(t, preview.Rows, 1)

	got := preview.Rows[0]
	assert.Equal(t, 0, preview.ErrorRows)
	assert.Equal(t, 1, preview.ValidRows)
	assert.Equal(t, 1, preview.WarningRows)
	assert.True(t, got.IsDuplicate)
	assert.Equal(t, "C0001", got.DuplicateCaseCode)
	require.NotNil(t, got.DuplicateCaseID)
	assert.Equal(t, dupCaseID, *got.DuplicateCaseID)
}

// TestParseCases_FlagsDuplicateByNameWhenNationalIDBlank 驗證身分證字號空白時改以姓名比對重複。
func TestParseCases_FlagsDuplicateByNameWhenNationalIDBlank(t *testing.T) {
	f := excelize.NewFile()
	defer f.Close()
	sheetName := "進系統個案個資"
	f.SetSheetName("Sheet1", sheetName)
	require.NoError(t, f.SetCellValue(sheetName, "A1", "姓名"))
	require.NoError(t, f.SetCellValue(sheetName, "A2", "王小明"))
	buf, err := f.WriteToBuffer()
	require.NoError(t, err)

	dupCaseID := uuid.New()
	finder := fakeDuplicateFinder{byName: map[string]*DuplicateRef{
		"王小明": {CaseID: dupCaseID, CaseCode: "C0002"},
	}}

	svc := NewImportService(nil, finder, nil, nil, nil, importinfra.NewExcelAdapter(), importinfra.NewExcelAdapter(), nil)
	preview, err := svc.ParseCases(context.Background(), bytes.NewReader(buf.Bytes()), "profile.xlsx")
	require.NoError(t, err)
	require.Len(t, preview.Rows, 1)
	assert.True(t, preview.Rows[0].IsDuplicate)
	assert.Equal(t, "C0002", preview.Rows[0].DuplicateCaseCode)
}

func TestParseCases_EmptyAndCorruptedFiles(t *testing.T) {
	svc := NewImportService(nil, nil, nil, nil, nil, importinfra.NewExcelAdapter(), importinfra.NewExcelAdapter(), nil)

	// 測試不支援的副檔名應回傳錯誤
	_, err := svc.ParseCases(context.Background(), strings.NewReader(""), "empty.csv")
	assert.Error(t, err, "非 .xlsx 副檔名應回傳錯誤")

	// 測試損毀的 Excel 檔案
	corrupted := []byte{0x50, 0x4B, 0x03, 0x04, 0x00, 0x00, 0x00}
	_, err = svc.ParseCases(context.Background(), bytes.NewReader(corrupted), "bad.xlsx")
	assert.Error(t, err, "損毀的 Excel 應回傳錯誤")
}

func TestParseCasesFromExcel_RealFile(t *testing.T) {
	filePath := filepath.Join("..", "..", "..", "..", "source", "彙整-個案資料(竹南.頭份).xlsx")
	f, err := os.Open(filePath)
	if err != nil {
		t.Skip("Sample file not found, skipping real file test")
		return
	}
	defer f.Close()

	svc := NewImportService(nil, nil, nil, nil, nil, importinfra.NewExcelAdapter(), importinfra.NewExcelAdapter(), nil)
	preview, err := svc.ParseCasesFromExcel(context.Background(), f)
	require.NoError(t, err)
	require.NotNil(t, preview)

	assert.Greater(t, preview.TotalRows, 0)
	assert.Equal(t, preview.TotalRows, preview.ValidRows)
	t.Logf("Parsed %d case rows, %d valid, %d with warnings",
		preview.TotalRows, preview.ValidRows, preview.WarningRows)
}

type fakeDuplicateFinder struct {
	byNationalID map[string]*DuplicateRef
	byName       map[string]*DuplicateRef
}

func (f fakeDuplicateFinder) FindDuplicate(_ context.Context, nationalID, name string) (*DuplicateRef, error) {
	if nationalID != "" {
		return f.byNationalID[nationalID], nil
	}
	return f.byName[name], nil
}
