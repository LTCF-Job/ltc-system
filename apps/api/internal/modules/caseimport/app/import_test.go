package app

import (
	"bytes"
	"context"
	"errors"
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

	// 表頭與匯出的「進系統個案個資」逐欄一致，讓匯出檔可以直接回灌。
	headerRow := rows[0]
	assert.Equal(t, []string{
		"姓名", "戶別", "身分證字號", "性別", "生日", "據點", "接送車輛(去)", "接送車輛(回)",
		"個管or照專", "照護人員", "戶籍", "居住地", "備註",
	}, headerRow)
	assert.NotContains(t, headerRow, "週一趟數(0:不搭/1:單去/2:來回/4:四趟)")
}

func TestParseCases_TemplateExcel(t *testing.T) {
	excelBytes, err := importinfra.NewExcelAdapter().RenderCaseImportTemplate()
	require.NoError(t, err)
	require.NotEmpty(t, excelBytes)

	svc := NewImportService(nil, nil, nil, nil, nil, nil, nil, importinfra.NewExcelAdapter(), importinfra.NewExcelAdapter(), nil)
	preview, err := svc.ParseCases(context.Background(), bytes.NewReader(excelBytes), "template.xlsx")
	require.NoError(t, err)
	require.NotNil(t, preview)

	// 示範列不再帶「例：」前綴，因此原封不動上傳會把這兩筆虛構個案當成真資料解析出來
	// （已與使用者確認採此行為）；使用者必須先刪除示範列再上傳。
	assert.Equal(t, 2, preview.TotalRows)
	assert.Equal(t, 2, preview.ValidRows)
	assert.Equal(t, 0, preview.ErrorRows)
	require.Len(t, preview.Rows, 2)
	assert.Equal(t, "王小明", preview.Rows[0].Name)
	assert.Equal(t, "張香香", preview.Rows[1].Name)
}

// 姓名空白但同列其他欄位有值，代表使用者漏填，必須列成錯誤列而不是靜默丟棄。
func TestParseCases_BlankNameRowBecomesError(t *testing.T) {
	f := excelize.NewFile()
	defer f.Close()
	sheet := f.GetSheetName(0)
	headers := []string{"姓名*", "戶別", "身分證字號", "性別", "生日", "據點", "接送車輛(去)", "接送車輛(回)", "個管or照專", "照護人員", "戶籍", "居住地", "備註"}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		require.NoError(t, f.SetCellValue(sheet, cell, h))
	}
	require.NoError(t, f.SetCellValue(sheet, "A2", "王大明"))
	require.NoError(t, f.SetCellValue(sheet, "B3", "一般戶"))
	require.NoError(t, f.SetCellValue(sheet, "M3", "姓名漏填"))

	buf, err := f.WriteToBuffer()
	require.NoError(t, err)

	svc := NewImportService(nil, nil, nil, nil, nil, nil, nil, importinfra.NewExcelAdapter(), importinfra.NewExcelAdapter(), nil)
	preview, err := svc.ParseCases(context.Background(), bytes.NewReader(buf.Bytes()), "blank-name.xlsx")
	require.NoError(t, err)
	require.NotNil(t, preview)

	assert.Equal(t, 2, preview.TotalRows)
	assert.Equal(t, 1, preview.ErrorRows)
	require.Len(t, preview.Errors, 1)
	assert.Equal(t, 3, preview.Errors[0].RowIndex)
	assert.Contains(t, preview.Errors[0].Message, "姓名未填寫")
}

// TestParseCases_ProfileWorkbook 驗證個案姓名（姓名欄）與照護人員姓名（照護人員欄）
// 各自對應到獨立欄位，不會互相覆蓋。
func TestParseCases_ProfileWorkbook(t *testing.T) {
	f := excelize.NewFile()
	defer f.Close()
	sheetName := "進系統個案個資"
	f.SetSheetName("Sheet1", sheetName)
	headers := []string{"姓名", "戶別", "身分證字號", "性別", "生日", "據點", "接送車輛(去)", "接送車輛(回)", "個管or照專", "照護人員", "戶籍", "居住地", "備註"}
	for i, header := range headers {
		cell, err := excelize.CoordinatesToCellName(i+1, 1)
		require.NoError(t, err)
		require.NoError(t, f.SetCellValue(sheetName, cell, header))
	}
	row := []interface{}{"王小明", "一般", "A202559750", "男", "045/06/15", "竹南日照", "竹南1車", "竹南2車", "個管", "陳小華", "苗栗縣竹南鎮戶籍地址", "苗栗縣竹南鎮居住地址", "需輪椅"}
	for i, value := range row {
		cell, err := excelize.CoordinatesToCellName(i+1, 2)
		require.NoError(t, err)
		require.NoError(t, f.SetCellValue(sheetName, cell, value))
	}

	buf, err := f.WriteToBuffer()
	require.NoError(t, err)

	svc := NewImportService(nil, nil, nil, nil, nil, nil, nil, importinfra.NewExcelAdapter(), importinfra.NewExcelAdapter(), nil)
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
	// 接送車輛兩欄保留版面但不匯入：來源檔即使填了車輛，解析結果一律留空。
	assert.Empty(t, got.OutboundVehicle)
	assert.Empty(t, got.InboundVehicle)
	assert.Equal(t, "需輪椅", got.Remarks)
	assert.False(t, got.IsDuplicate)
}

// TestParseCases_CaregiverUnmatchedWarnsInPreview 驗證照護人員姓名比對不到主檔時，
// 預覽階段就會標記警告，不用等到正式匯入才發現會落入待維護。
func TestParseCases_CaregiverUnmatchedWarnsInPreview(t *testing.T) {
	f := excelize.NewFile()
	defer f.Close()
	sheetName := "進系統個案個資"
	f.SetSheetName("Sheet1", sheetName)
	headers := []string{"姓名", "個管or照專", "照護人員"}
	for i, header := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		require.NoError(t, f.SetCellValue(sheetName, cell, header))
	}
	require.NoError(t, f.SetCellValue(sheetName, "A2", "王小明"))
	require.NoError(t, f.SetCellValue(sheetName, "B2", "個管"))
	require.NoError(t, f.SetCellValue(sheetName, "C2", "查無此人"))
	buf, err := f.WriteToBuffer()
	require.NoError(t, err)

	svc := NewImportService(nil, nil, nil, nil, nil, fakeCaregiverLookup{}, nil, importinfra.NewExcelAdapter(), importinfra.NewExcelAdapter(), nil)
	preview, err := svc.ParseCases(context.Background(), bytes.NewReader(buf.Bytes()), "profile.xlsx")
	require.NoError(t, err)
	require.Len(t, preview.Rows, 1)

	got := preview.Rows[0]
	assert.Equal(t, 0, preview.ErrorRows)
	assert.Equal(t, 1, preview.WarningRows)
	assert.True(t, got.CaregiverUnmatched)
	assert.Contains(t, got.WarningMessage, "照護人員")
	assert.Contains(t, got.WarningMessage, "查無此人")
}

// TestParseCases_SiteHeader 驗證「據點」欄位標題可正確解析為 SiteName。
func TestParseCases_SiteHeader(t *testing.T) {
	f := excelize.NewFile()
	defer f.Close()
	sheetName := "進系統個案個資"
	f.SetSheetName("Sheet1", sheetName)
	headers := []string{"姓名", "戶別", "身分證字號", "性別", "生日", "據點"}
	for i, header := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		require.NoError(t, f.SetCellValue(sheetName, cell, header))
	}
	values := []interface{}{"馮玉英", "", "", "", "", "竹南日照據點"}
	for i, value := range values {
		cell, _ := excelize.CoordinatesToCellName(i+1, 2)
		require.NoError(t, f.SetCellValue(sheetName, cell, value))
	}
	buf, err := f.WriteToBuffer()
	require.NoError(t, err)

	preview, err := NewImportService(nil, nil, nil, nil, nil, nil, nil, importinfra.NewExcelAdapter(), importinfra.NewExcelAdapter(), nil).ParseCases(context.Background(), bytes.NewReader(buf.Bytes()), "profile.xlsx")
	require.NoError(t, err)
	require.Len(t, preview.Rows, 1)
	assert.Equal(t, "竹南日照據點", preview.Rows[0].SiteName)
}

// TestParseCases_LegacySiteHeaderNoLongerRecognized 驗證舊版「單位」欄位標題不再被
// 解析為據點：使用者需改用新範本的「據點」標頭，這是使用者明確拍板的行為變更。
func TestParseCases_LegacySiteHeaderNoLongerRecognized(t *testing.T) {
	f := excelize.NewFile()
	defer f.Close()
	sheetName := "進系統個案個資"
	f.SetSheetName("Sheet1", sheetName)
	headers := []string{"姓名", "戶別", "身分證字號", "性別", "生日", "單位"}
	for i, header := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		require.NoError(t, f.SetCellValue(sheetName, cell, header))
	}
	values := []interface{}{"馮玉英", "", "", "", "", "竹南日照單位"}
	for i, value := range values {
		cell, _ := excelize.CoordinatesToCellName(i+1, 2)
		require.NoError(t, f.SetCellValue(sheetName, cell, value))
	}
	buf, err := f.WriteToBuffer()
	require.NoError(t, err)

	preview, err := NewImportService(nil, nil, nil, nil, nil, nil, nil, importinfra.NewExcelAdapter(), importinfra.NewExcelAdapter(), nil).ParseCases(context.Background(), bytes.NewReader(buf.Bytes()), "profile.xlsx")
	require.NoError(t, err)
	require.Len(t, preview.Rows, 1)
	assert.Empty(t, preview.Rows[0].SiteName, "舊「單位」標頭不再被讀取")
}

// TestParseCases_OnlyNameRequired 驗證除姓名外全部欄位皆選填，缺漏不再擋錯。
func TestParseCases_OnlyNameRequired(t *testing.T) {
	f := excelize.NewFile()
	defer f.Close()
	sheetName := "進系統個案個資"
	f.SetSheetName("Sheet1", sheetName)
	headers := []string{"姓名", "戶別", "身分證字號", "性別", "生日", "據點", "接送車輛(去)", "接送車輛(回)", "個管or照專", "照護人員", "戶籍", "居住地", "備註"}
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

	preview, err := NewImportService(nil, nil, nil, nil, nil, nil, nil, importinfra.NewExcelAdapter(), importinfra.NewExcelAdapter(), nil).ParseCases(context.Background(), bytes.NewReader(buf.Bytes()), "profile.xlsx")
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

	preview, err := NewImportService(nil, nil, nil, nil, nil, nil, nil, importinfra.NewExcelAdapter(), importinfra.NewExcelAdapter(), nil).ParseCases(context.Background(), bytes.NewReader(buf.Bytes()), "profile.xlsx")
	require.NoError(t, err)
	assert.Equal(t, 1, preview.TotalRows, "全空白列不應計入總筆數")
	assert.Equal(t, 1, preview.ValidRows)
	assert.Equal(t, 0, preview.ErrorRows, "全空白列應直接忽略，不應歸入錯誤列")
	require.Len(t, preview.Rows, 1)
	assert.Equal(t, "馮玉英", preview.Rows[0].Name)
}

// 匯出的生日是民國年 RRR/MM/DD，匯出檔要能原樣回灌；西元 YYYY/MM/DD 寫法也要能解析成同一個日期。
func TestParseCases_GregorianBirthDateRoundTrip(t *testing.T) {
	f := excelize.NewFile()
	defer f.Close()
	sheet := f.GetSheetName(0)
	headers := []string{"姓名", "戶別", "身分證字號", "性別", "生日", "據點", "接送車輛(去)", "接送車輛(回)", "個管or照專", "照護人員", "戶籍", "居住地", "備註"}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		require.NoError(t, f.SetCellValue(sheet, cell, h))
	}
	// 第 2 列西元、第 3 列民國：兩種寫法都要解析成同一個日期。
	values := [][]interface{}{
		{"西元寫法", "", "", "女", "1956/06/15", "竹南日照", "", "", "", "", "", "", ""},
		{"民國寫法", "", "", "女", "045/06/15", "竹南日照", "", "", "", "", "", "", ""},
	}
	for r, row := range values {
		for i, v := range row {
			cell, _ := excelize.CoordinatesToCellName(i+1, r+2)
			require.NoError(t, f.SetCellValue(sheet, cell, v))
		}
	}
	buf, err := f.WriteToBuffer()
	require.NoError(t, err)

	svc := NewImportService(nil, nil, nil, nil, nil, nil, nil, importinfra.NewExcelAdapter(), importinfra.NewExcelAdapter(), nil)
	preview, err := svc.ParseCases(context.Background(), bytes.NewReader(buf.Bytes()), "export.xlsx")
	require.NoError(t, err)
	require.Len(t, preview.Rows, 2)
	assert.Equal(t, "1956-06-15", preview.Rows[0].BirthDate)
	assert.Equal(t, "1956-06-15", preview.Rows[1].BirthDate)
	assert.False(t, preview.Rows[0].BirthDateInvalid)
	assert.False(t, preview.Rows[1].BirthDateInvalid)
}

// TestParseCases_ReportsBirthDateFormatError 驗證生日格式錯誤不擋列，改標記為待補正 warning。
func TestParseCases_ReportsBirthDateFormatError(t *testing.T) {
	f := excelize.NewFile()
	defer f.Close()
	sheetName := "進系統個案個資"
	f.SetSheetName("Sheet1", sheetName)
	headers := []string{"姓名", "戶別", "身分證字號", "性別", "生日", "據點", "接送車輛(去)", "接送車輛(回)", "個管or照專", "照護人員", "戶籍", "居住地"}
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

	preview, err := NewImportService(nil, nil, nil, nil, nil, nil, nil, importinfra.NewExcelAdapter(), importinfra.NewExcelAdapter(), nil).ParseCases(context.Background(), bytes.NewReader(buf.Bytes()), "profile.xlsx")
	require.NoError(t, err)
	require.Len(t, preview.Rows, 1)
	assert.Equal(t, 0, preview.ErrorRows)
	assert.Equal(t, 1, preview.WarningRows)
	assert.Equal(t, "", preview.Rows[0].ErrorMessage)
	assert.True(t, preview.Rows[0].BirthDateInvalid)
	assert.Equal(t, "錯誤生日", preview.Rows[0].BirthDateRaw)
	assert.Contains(t, preview.Rows[0].WarningMessage, "生日：格式錯誤")
	assert.Equal(t, "", preview.Rows[0].RawValues["戶別"])
	assert.Equal(t, "錯誤生日", preview.Rows[0].RawValues["生日"])
}

// TestParseCases_InvalidNationalID_SetsWarningNotError 驗證身分證字號格式錯誤不擋列，
// 改標記為待補正 warning，個案仍可正式匯入。
func TestParseCases_InvalidNationalID_SetsWarningNotError(t *testing.T) {
	f := excelize.NewFile()
	defer f.Close()
	sheetName := "進系統個案個資"
	f.SetSheetName("Sheet1", sheetName)
	headers := []string{"姓名", "身分證字號"}
	for i, header := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		require.NoError(t, f.SetCellValue(sheetName, cell, header))
	}
	require.NoError(t, f.SetCellValue(sheetName, "A2", "格式錯誤個案"))
	require.NoError(t, f.SetCellValue(sheetName, "B2", "NOT-VALID"))
	buf, err := f.WriteToBuffer()
	require.NoError(t, err)

	preview, err := NewImportService(nil, nil, nil, nil, nil, nil, nil, importinfra.NewExcelAdapter(), importinfra.NewExcelAdapter(), nil).ParseCases(context.Background(), bytes.NewReader(buf.Bytes()), "profile.xlsx")
	require.NoError(t, err)
	require.Len(t, preview.Rows, 1)
	assert.Equal(t, 0, preview.ErrorRows)
	assert.Equal(t, 1, preview.WarningRows)
	assert.Equal(t, "", preview.Rows[0].ErrorMessage)
	assert.True(t, preview.Rows[0].NationalIDInvalid)
	assert.Contains(t, preview.Rows[0].WarningMessage, "身分證字號：格式錯誤")
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
		"A202559750": {CaseID: dupCaseID, CaseName: "王小明"},
	}}

	svc := NewImportService(nil, finder, nil, nil, nil, nil, nil, importinfra.NewExcelAdapter(), importinfra.NewExcelAdapter(), nil)
	preview, err := svc.ParseCases(context.Background(), bytes.NewReader(buf.Bytes()), "profile.xlsx")
	require.NoError(t, err)
	require.Len(t, preview.Rows, 1)

	got := preview.Rows[0]
	assert.Equal(t, 0, preview.ErrorRows)
	assert.Equal(t, 1, preview.ValidRows)
	assert.Equal(t, 1, preview.WarningRows)
	assert.True(t, got.IsDuplicate)
	assert.Equal(t, "王小明", got.DuplicateCaseName)
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
		"王小明": {CaseID: dupCaseID, CaseName: "王小明"},
	}}

	svc := NewImportService(nil, finder, nil, nil, nil, nil, nil, importinfra.NewExcelAdapter(), importinfra.NewExcelAdapter(), nil)
	preview, err := svc.ParseCases(context.Background(), bytes.NewReader(buf.Bytes()), "profile.xlsx")
	require.NoError(t, err)
	require.Len(t, preview.Rows, 1)
	assert.True(t, preview.Rows[0].IsDuplicate)
	assert.Equal(t, "王小明", preview.Rows[0].DuplicateCaseName)
}

func TestParseCases_DuplicateLookupFailureMarksRowAsError(t *testing.T) {
	f := excelize.NewFile()
	defer f.Close()
	sheetName := "進系統個案個資"
	f.SetSheetName("Sheet1", sheetName)
	require.NoError(t, f.SetCellValue(sheetName, "A1", "姓名"))
	require.NoError(t, f.SetCellValue(sheetName, "A2", "王小明"))
	buf, err := f.WriteToBuffer()
	require.NoError(t, err)

	svc := NewImportService(nil, failingDuplicateFinder{}, nil, nil, nil, nil, nil, importinfra.NewExcelAdapter(), importinfra.NewExcelAdapter(), nil)
	preview, err := svc.ParseCases(context.Background(), bytes.NewReader(buf.Bytes()), "profile.xlsx")
	require.NoError(t, err)
	require.Len(t, preview.Rows, 1)

	assert.Equal(t, 1, preview.ErrorRows)
	assert.Equal(t, 0, preview.ValidRows)
	assert.Contains(t, preview.Rows[0].ErrorMessage, "重複個案查詢失敗")
	assert.Len(t, preview.Errors, 1)
}

func TestParseCases_EmptyAndCorruptedFiles(t *testing.T) {
	svc := NewImportService(nil, nil, nil, nil, nil, nil, nil, importinfra.NewExcelAdapter(), importinfra.NewExcelAdapter(), nil)

	// 測試不支援的副檔名應回傳錯誤
	_, err := svc.ParseCases(context.Background(), strings.NewReader(""), "empty.csv")
	assert.Error(t, err, "非 .xlsx 副檔名應回傳錯誤")

	assert.ErrorIs(t, err, ErrUnsupportedFileType, "副檔名不符要能被分辨成檔案格式錯誤")

	// 測試損毀的 Excel 檔案
	corrupted := []byte{0x50, 0x4B, 0x03, 0x04, 0x00, 0x00, 0x00}
	_, err = svc.ParseCases(context.Background(), bytes.NewReader(corrupted), "bad.xlsx")
	assert.Error(t, err, "損毀的 Excel 應回傳錯誤")
	assert.ErrorIs(t, err, ErrFileUnreadable, "損毀檔案要能被分辨成無法讀取")
}

// TestParseCases_TemplateMismatchIsAnError 鎖住「範本用錯」的回饋：找不到姓名欄時
// 必須回傳可辨識的錯誤，而不是靜默回傳零列讓畫面顯示「總筆數 0」卻沒有原因。
func TestParseCases_TemplateMismatchIsAnError(t *testing.T) {
	f := excelize.NewFile()
	defer f.Close()
	sheetName := "工作表1"
	f.SetSheetName("Sheet1", sheetName)
	for i, header := range []string{"編號", "備註", "金額"} {
		cell, err := excelize.CoordinatesToCellName(i+1, 1)
		require.NoError(t, err)
		require.NoError(t, f.SetCellValue(sheetName, cell, header))
	}

	var buf bytes.Buffer
	require.NoError(t, f.Write(&buf))

	svc := NewImportService(nil, nil, nil, nil, nil, nil, nil, importinfra.NewExcelAdapter(), importinfra.NewExcelAdapter(), nil)
	_, err := svc.ParseCases(context.Background(), bytes.NewReader(buf.Bytes()), "wrong.xlsx")

	require.ErrorIs(t, err, ErrTemplateMismatch)
	assert.Contains(t, err.Error(), sheetName, "錯誤訊息要指出是哪個工作表對不上範本")
}

func TestParseCasesFromExcel_RealFile(t *testing.T) {
	filePath := filepath.Join("..", "..", "..", "..", "source", "彙整-個案資料(竹南.頭份).xlsx")
	f, err := os.Open(filePath)
	if err != nil {
		t.Skip("Sample file not found, skipping real file test")
		return
	}
	defer f.Close()

	svc := NewImportService(nil, nil, nil, nil, nil, nil, nil, importinfra.NewExcelAdapter(), importinfra.NewExcelAdapter(), nil)
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

type failingDuplicateFinder struct{}

func (failingDuplicateFinder) FindDuplicate(context.Context, string, string) (*DuplicateRef, error) {
	return nil, errors.New("database unavailable")
}

func (f fakeDuplicateFinder) FindDuplicate(_ context.Context, nationalID, name string) (*DuplicateRef, error) {
	if nationalID != "" {
		return f.byNationalID[nationalID], nil
	}
	return f.byName[name], nil
}

// TestParseCases_InvalidNationalIDFallsBackToNameDuplicateCheck 驗證身分證字號格式錯誤時，
// 重複比對改用姓名：這一列不會寫入身分證字號，拿格式錯誤的字串算 HMAC 必定比不到任何個案，
// 卻會讓姓名比對整個被跳過，使重複個案直接建立成第二筆個案。
func TestParseCases_InvalidNationalIDFallsBackToNameDuplicateCheck(t *testing.T) {
	f := excelize.NewFile()
	defer f.Close()
	sheetName := "進系統個案個資"
	f.SetSheetName("Sheet1", sheetName)
	require.NoError(t, f.SetCellValue(sheetName, "A1", "姓名"))
	require.NoError(t, f.SetCellValue(sheetName, "B1", "身分證字號"))
	require.NoError(t, f.SetCellValue(sheetName, "A2", "王小明"))
	require.NoError(t, f.SetCellValue(sheetName, "B2", "A1234"))
	buf, err := f.WriteToBuffer()
	require.NoError(t, err)

	dupCaseID := uuid.New()
	finder := fakeDuplicateFinder{byName: map[string]*DuplicateRef{
		"王小明": {CaseID: dupCaseID, CaseName: "王小明"},
	}}

	svc := NewImportService(nil, finder, nil, nil, nil, nil, nil, importinfra.NewExcelAdapter(), importinfra.NewExcelAdapter(), nil)
	preview, err := svc.ParseCases(context.Background(), bytes.NewReader(buf.Bytes()), "profile.xlsx")
	require.NoError(t, err)
	require.Len(t, preview.Rows, 1)

	got := preview.Rows[0]
	assert.Equal(t, 0, preview.ErrorRows)
	assert.True(t, got.NationalIDInvalid)
	assert.True(t, got.IsDuplicate, "身分證字號格式錯誤的列仍須以姓名比對出重複個案")
	require.NotNil(t, got.DuplicateCaseID)
	assert.Equal(t, dupCaseID, *got.DuplicateCaseID)
}
