package infra

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/xuri/excelize/v2"
)

// ExcelAdapter 是 driverreport 唯一接觸 excelize 的地方：對外只交換位元組與純文字
// 儲存格，讓 app 層不需認識任何試算表 SDK 型別。
type ExcelAdapter struct{}

// NewExcelAdapter 建立 ExcelAdapter 實例。
func NewExcelAdapter() ExcelAdapter { return ExcelAdapter{} }

// driverReportSheetName 是範本與匯入檔的工作表名稱。
const driverReportSheetName = "司機接送匯報"

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

// RenderDriverReportTemplate 產生司機接送匯報表空白範本。
//
// 範本刻意不放示範資料列：匯入是逐列讀到檔案結尾，任何非空的示範列都會被當成
// 真實匯報寫進搭乘紀錄。填寫說明改掛在表頭儲存格的註解上。
func (r ExcelAdapter) RenderDriverReportTemplate(vehicleName string, caseColumns []string) ([]byte, error) {
	f := excelize.NewFile()
	defer f.Close()

	f.SetSheetName("Sheet1", driverReportSheetName)

	headers := make([]string, 0, len(caseColumns)+3)
	headers = append(headers, "民國日期", "駕駛人")
	headers = append(headers, caseColumns...)
	headers = append(headers, "備註")

	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "FFFFFF", Size: 11, Family: "Microsoft JhengHei"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"2065D1"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: true},
	})

	for colIdx, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(colIdx+1, 1)
		_ = f.SetCellValue(driverReportSheetName, cell, h)
	}
	_ = f.SetRowHeight(driverReportSheetName, 1, 40)
	lastCol, _ := excelize.CoordinatesToCellName(len(headers), 1)
	_ = f.SetCellStyle(driverReportSheetName, "A1", lastCol, headerStyle)
	_ = f.SetColWidth(driverReportSheetName, "A", "B", 14)
	if len(caseColumns) > 0 {
		firstCase, _ := excelize.ColumnNumberToName(3)
		lastCase, _ := excelize.ColumnNumberToName(len(headers) - 1)
		_ = f.SetColWidth(driverReportSheetName, firstCase, lastCase, 18)
	}

	_ = f.AddComment(driverReportSheetName, excelize.Comment{
		Cell: "A1",
		Text: fmt.Sprintf("%s 每日接送匯報。一列一天，民國日期請填七碼（例如 1150302），也接受 115/3/2。", vehicleName),
	})
	_ = f.AddComment(driverReportSheetName, excelize.Comment{
		Cell: "B1",
		Text: "填寫當日實際駕駛此車的司機姓名，需與司機主檔一致才會自動關聯。",
	})
	if len(caseColumns) > 0 {
		_ = f.AddComment(driverReportSheetName, excelize.Comment{
			Cell: "C1",
			Text: "每個個案趟次欄只填「有坐」或「沒坐」；留白視為未回報，不會建立搭乘紀錄。",
		})
	}
	_ = f.AddComment(driverReportSheetName, excelize.Comment{
		Cell: lastCol,
		Text: "當日問題回報或補充說明，可留白。",
	})

	_ = f.SetSheetDimension(driverReportSheetName, fmt.Sprintf("A1:%s", lastCol))

	buf, err := f.WriteToBuffer()
	if err != nil {
		return nil, fmt.Errorf("failed to generate excel buffer: %w", err)
	}
	return buf.Bytes(), nil
}
