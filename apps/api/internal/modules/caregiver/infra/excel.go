package infra

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/xuri/excelize/v2"
	"ltc-system/apps/api/internal/platform/spreadsheet"
)

// ExcelAdapter 是 caregiver 模組唯一接觸 excelize 的地方：對外只交換位元組與純文字
// 儲存格，讓 app 層不需認識任何試算表 SDK 型別。
type ExcelAdapter struct{}

// NewExcelAdapter 建立 ExcelAdapter 實例。
func NewExcelAdapter() ExcelAdapter { return ExcelAdapter{} }

// ReadTables 將 Excel 位元組解碼為逐工作表的儲存格文字；超過解析規模上限時回報錯誤。
func (r ExcelAdapter) ReadTables(data []byte) ([][][]string, []string, error) {
	if err := spreadsheet.ValidateXLSXZip(data); err != nil {
		return nil, nil, err
	}
	f, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		return nil, nil, fmt.Errorf("開啟 Excel 檔案失敗: %w", err)
	}
	defer f.Close()

	var tables [][][]string
	var sheetNames []string
	var counter spreadsheet.LimitCounter
	for _, sheet := range f.GetSheetList() {
		rows, err := readSheetRows(f, sheet, &counter)
		if err != nil {
			return nil, nil, err
		}
		if len(rows) > 0 {
			tables = append(tables, rows)
			sheetNames = append(sheetNames, sheet)
		}
	}

	if len(tables) == 0 {
		return nil, nil, errors.New("excel 檔案中無工作表資料")
	}
	return tables, sheetNames, nil
}

// readSheetRows 讀出單一工作表的儲存格文字；無法解析的工作表回傳空表格，沿用既有的略過行為。
func readSheetRows(f *excelize.File, sheet string, counter *spreadsheet.LimitCounter) ([][]string, error) {
	if err := counter.BeginSheet(); err != nil {
		return nil, err
	}
	// 逐列串流而非 GetRows：壓縮炸彈必須在展開成完整表格之前就被攔下。
	it, err := f.Rows(sheet)
	if err != nil {
		return nil, nil
	}
	defer it.Close()

	var rows [][]string
	for it.Next() {
		cols, err := it.Columns()
		if err != nil {
			return nil, nil
		}
		if err := counter.AddRow(sheet, len(cols)); err != nil {
			return nil, err
		}
		rows = append(rows, cols)
	}
	return spreadsheet.TrimTrailingEmptyRows(rows), nil
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
		{"專護", "竹南日照單位", "王大明", "0987-654-321", ""},
	}
	for rIdx, rData := range sampleRows {
		rowNum := rIdx + 2
		for cIdx, val := range rData {
			cell, _ := excelize.CoordinatesToCellName(cIdx+1, rowNum)
			_ = f.SetCellValue(sheetName, cell, val)
		}
	}

	// 說明文字以儲存格註解附加在標題列，而非寫在資料列下方，避免被解析器誤判為一筆資料
	// （GetRows 會把任何非空白列都當成候選資料列，寫在表格下方的純文字列會被視為缺漏必填欄位的錯誤列）。
	_ = f.AddComment(sheetName, excelize.Comment{
		Cell: "A1",
		Text: "＊姓名與類型為必填，類型請填寫「個管」或「專護」；單位請填寫既有單位名稱，找不到相符單位時仍會建立資料，匯入後可於「待維護」頁籤補建關聯。",
	})

	buf, err := f.WriteToBuffer()
	if err != nil {
		return nil, fmt.Errorf("failed to generate excel buffer: %w", err)
	}
	return buf.Bytes(), nil
}
