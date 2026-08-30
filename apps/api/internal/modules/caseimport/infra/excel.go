package infra

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/xuri/excelize/v2"
)

// ExcelAdapter 是 caseimport 唯一接觸 excelize 的地方：對外只交換位元組與純文字
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

// RenderCaseImportTemplate 產生個案批次匯入標準 Excel 檔案位元組。
func (r ExcelAdapter) RenderCaseImportTemplate() ([]byte, error) {
	f := excelize.NewFile()
	defer f.Close()

	sheetName := "個案匯入範本"
	f.SetSheetName("Sheet1", sheetName)

	headers := []string{
		"姓名*", "戶別", "身分證字號", "性別", "生日", "據點", "接送車輛(去)", "接送車輛(回)",
		"個管or照專", "姓名(個管/照專)", "戶籍", "居住地", "備註",
	}

	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "FFFFFF", Size: 11, Family: "Microsoft JhengHei"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"2065D1"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: true},
	})

	for colIdx, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(colIdx+1, 1)
		_ = f.SetCellValue(sheetName, cell, h)
	}
	_ = f.SetRowHeight(sheetName, 1, 32)
	lastCol, _ := excelize.CoordinatesToCellName(len(headers), 1)
	_ = f.SetCellStyle(sheetName, "A1", lastCol, headerStyle)

	sampleRows := [][]interface{}{
		{"張曾阿妹", "一般戶", "A202559750", "女", "034/06/15", "竹南日照據點", "竹南1車", "竹南2車", "個管", "陳小華", "苗栗縣竹南鎮戶籍地址", "苗栗縣竹南鎮大營路123號", "行動不便需輪椅"},
		{"李國盛", "低收入戶", "G121806465", "男", "039/02/20", "竹北日照中心", "竹北1車", "竹北2車", "照專", "王小明", "新竹縣竹北市戶籍地址", "新竹縣竹北市文興路一段200號", ""},
	}

	for rIdx, rData := range sampleRows {
		rowNum := rIdx + 2
		for cIdx, val := range rData {
			cell, _ := excelize.CoordinatesToCellName(cIdx+1, rowNum)
			_ = f.SetCellValue(sheetName, cell, val)
		}
	}
	lastRowCell, _ := excelize.CoordinatesToCellName(len(headers), len(sampleRows)+1)
	_ = f.SetSheetDimension(sheetName, fmt.Sprintf("A1:%s", lastRowCell))

	buf, err := f.WriteToBuffer()
	if err != nil {
		return nil, fmt.Errorf("failed to generate excel buffer: %w", err)
	}
	return buf.Bytes(), nil
}
