package infra

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/xuri/excelize/v2"
	"ltc-system/apps/api/internal/platform/spreadsheet"
)

// ExcelAdapter 是 caseimport 唯一接觸 excelize 的地方：對外只交換位元組與純文字
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

// RenderCaseImportTemplate 產生個案批次匯入標準 Excel 檔案位元組。
func (r ExcelAdapter) RenderCaseImportTemplate() ([]byte, error) {
	f := excelize.NewFile()
	defer f.Close()

	sheetName := "個案匯入範本"
	f.SetSheetName("Sheet1", sheetName)

	// 表頭與匯出的「進系統個案個資」逐欄一致，讓匯出檔可以直接回灌。序號與歲數只佔版面，
	// 解析時不取值；接送車輛兩欄保留版面但暫不使用。
	headers := []string{
		"序號", "姓名", "戶別", "身分證字號", "性別", "生日", "歲數", "據點", "接送車輛(去)", "接送車輛(回)",
		"個管or照專", "姓名", "戶籍", "居住地", "備註",
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

	// 示範列是純虛構的隨機資料，不帶任何前綴標記，因此解析時會被當成一般資料列。
	// 使用者必須先刪除這兩列再上傳，否則會匯入這兩筆假個案。
	sampleRows := [][]interface{}{
		{1, "王小明", "一般戶", "D140397675", "男", "1943/11/08", 83, "竹南日照據點", "", "", "個管", "林佩宜", "苗栗縣竹南鎮公館里5鄰12號", "苗栗縣竹南鎮大營路123號", "行動不便需輪椅"},
		{2, "張香香", "低收入戶", "K211281700", "女", "1952/03/22", 74, "竹北日照中心", "", "", "照專", "蔡孟儒", "新竹縣竹北市斗崙里8鄰27號", "新竹縣竹北市文興路一段200號", ""},
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

	// 說明文字只能掛在表頭儲存格；放進資料列會被逐列 parser 讀成幽靈個案。
	_ = f.AddComment(sheetName, excelize.Comment{
		Cell: "B1",
		Text: "＊姓名為必填。生日請填西元 YYYY/MM/DD（民國 045/06/15 這種寫法也接受）。" +
			"序號與歲數僅供對照，匯入時不會使用；接送車輛(去)/(回) 目前保留欄位但不匯入。" +
			"個管or照專與其右方的姓名，會以姓名比對照護人員主檔（同名多筆時才依個管／照專區分）。",
	})

	buf, err := f.WriteToBuffer()
	if err != nil {
		return nil, fmt.Errorf("failed to generate excel buffer: %w", err)
	}
	return buf.Bytes(), nil
}
