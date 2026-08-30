package infra

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/xuri/excelize/v2"
)

// ExcelAdapter 是 caregiver 模組唯一接觸 excelize 的地方：對外只交換位元組與純文字
// 儲存格，讓 app 層不需認識任何試算表 SDK 型別。
type ExcelAdapter struct{}

// NewExcelAdapter 建立 ExcelAdapter 實例。
func NewExcelAdapter() ExcelAdapter { return ExcelAdapter{} }

// ReadTables 將 Excel 位元組解碼為逐工作表的儲存格文字。
func (r ExcelAdapter) ReadTables(data []byte) ([][][]string, []string, error) {
	f, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		return nil, nil, fmt.Errorf("開啟 Excel 檔案失敗: %w", err)
	}
	defer f.Close()

	var tables [][][]string
	var sheetNames []string
	for _, sheet := range f.GetSheetList() {
		rows, err := f.GetRows(sheet)
		if err == nil && len(rows) > 0 {
			tables = append(tables, rows)
			sheetNames = append(sheetNames, sheet)
		}
	}

	if len(tables) == 0 {
		return nil, nil, errors.New("excel 檔案中無工作表資料")
	}
	return tables, sheetNames, nil
}

// RenderCaregiverImportTemplate 產生照護人員批次匯入標準 Excel 檔案位元組。
func (r ExcelAdapter) RenderCaregiverImportTemplate() ([]byte, error) {
	f := excelize.NewFile()
	defer f.Close()

	sheetName := "照護人員匯入範本"
	f.SetSheetName("Sheet1", sheetName)

	headers := []string{"類型*", "單位", "姓名*", "聯絡方式", "備註"}

	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "FFFFFF", Size: 11, Family: "Microsoft JhengHei"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"2065D1"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: true},
	})

	for colIdx, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(colIdx+1, 1)
		_ = f.SetCellValue(sheetName, cell, h)
	}
	_ = f.SetRowHeight(sheetName, 1, 28)
	lastCol, _ := excelize.CoordinatesToCellName(len(headers), 1)
	_ = f.SetCellStyle(sheetName, "A1", lastCol, headerStyle)

	sampleRows := [][]interface{}{
		{"個管", "竹北日照中心", "陳小華", "0912-345-678", "熟悉輪椅移位協助"},
		{"專護", "竹南日照據點", "王大明", "0987-654-321", ""},
	}
	for rIdx, rData := range sampleRows {
		rowNum := rIdx + 2
		for cIdx, val := range rData {
			cell, _ := excelize.CoordinatesToCellName(cIdx+1, rowNum)
			_ = f.SetCellValue(sheetName, cell, val)
		}
	}

	noteRow := len(sampleRows) + 3
	noteCell, _ := excelize.CoordinatesToCellName(1, noteRow)
	_ = f.SetCellValue(sheetName, noteCell, "＊姓名與類型為必填，類型請填寫「個管」或「專護」；單位請填寫既有據點名稱，找不到相符據點時仍會建立資料，匯入後可於「待維護」頁籤補建關聯。")

	buf, err := f.WriteToBuffer()
	if err != nil {
		return nil, fmt.Errorf("failed to generate excel buffer: %w", err)
	}
	return buf.Bytes(), nil
}
