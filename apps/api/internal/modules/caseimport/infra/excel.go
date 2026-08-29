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
		"個案姓名*", "身分證字號*", "申報地區*(苗栗/新竹)", "聯絡電話", "住家地址*",
		"開始申報日*(YYYY-MM-DD)", "服務類別*(1:補助/2:自費)", "服務使用類型*(1:社區長照/2:社區據點/3:輔具中心/4:身障日照)", "所屬據點*",
		"單趟里程(公里)*", "申報單價(元)", "服務時長(分鐘)",
		"週一趟數(0:不搭/1:單去/2:來回/4:四趟)", "週一去程時間(HH:mm)", "週一回程時間(HH:mm)",
		"週二趟數(0:不搭/1:單去/2:來回/4:四趟)", "週二去程時間(HH:mm)", "週二回程時間(HH:mm)",
		"週三趟數(0:不搭/1:單去/2:來回/4:四趟)", "週三去程時間(HH:mm)", "週三回程時間(HH:mm)",
		"週四趟數(0:不搭/1:單去/2:來回/4:四趟)", "週四去程時間(HH:mm)", "週四回程時間(HH:mm)",
		"週五趟數(0:不搭/1:單去/2:來回/4:四趟)", "週五去程時間(HH:mm)", "週五回程時間(HH:mm)",
		"週六趟數(0:不搭/1:單去/2:來回/4:四趟)", "週六去程時間(HH:mm)", "週六回程時間(HH:mm)",
		"週日趟數(0:不搭/1:單去/2:來回/4:四趟)", "週日去程時間(HH:mm)", "週日回程時間(HH:mm)",
		"備註",
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
	_ = f.SetCellStyle(sheetName, "A1", "AH1", headerStyle)

	sampleRows := [][]interface{}{
		{"張曾阿妹", "A202559750", "苗栗", "0912345678", "苗栗縣竹南鎮大營路123號", "2026-07-01", 1, 2, "竹南日照據點", 5.0, 115, 10, 2, "09:00", "16:00", 2, "09:00", "16:00", 2, "09:00", "16:00", 2, "09:00", "16:00", 2, "09:00", "16:00", 0, "", "", 0, "", "", "週一至週五固定來回"},
		{"李國盛", "J123458899", "新竹", "0922334455", "新竹縣竹北市文興路一段200號", "2026-07-01", 2, 1, "竹北日照中心", 8.0, 200, 20, 2, "09:30", "15:30", 1, "09:30", "", 2, "09:30", "15:30", 0, "", "", 2, "09:30", "15:30", 0, "", "", 0, "", "", "週二僅早上去程"},
		{"王大同", "K123456780", "苗栗", "0933445566", "苗栗市中正路50號", "2026-07-01", 1, 4, "苗栗復健據點", 6.5, 115, 15, 0, "", "", 0, "", "", 0, "", "", 4, "08:30", "16:30", 0, "", "", 0, "", "", 0, "", "", "週四四趟個案"},
	}

	for rIdx, rData := range sampleRows {
		rowNum := rIdx + 2
		for cIdx, val := range rData {
			cell, _ := excelize.CoordinatesToCellName(cIdx+1, rowNum)
			_ = f.SetCellValue(sheetName, cell, val)
		}
	}
	_ = f.SetSheetDimension(sheetName, fmt.Sprintf("A1:AH%d", len(sampleRows)+1))

	buf, err := f.WriteToBuffer()
	if err != nil {
		return nil, fmt.Errorf("failed to generate excel buffer: %w", err)
	}
	return buf.Bytes(), nil
}
