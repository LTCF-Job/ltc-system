package infra

import (
	"bytes"
	"fmt"

	"github.com/xuri/excelize/v2"
	"ltc-system/apps/api/internal/modules/ops/app"
)

// RenderBlankMaintenanceTemplate 依車輛清單產生空白維修保養檢查表格，每車一個分頁。
func (ExcelRenderer) RenderBlankMaintenanceTemplate(vehicles []app.VehicleLabel) ([]byte, error) {
	f := excelize.NewFile()
	defer f.Close()

	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "FFFFFF", Size: 11, Family: "Microsoft JhengHei"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"1D5B79"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "B0C4DE", Style: 1},
			{Type: "top", Color: "B0C4DE", Style: 1},
			{Type: "bottom", Color: "B0C4DE", Style: 1},
			{Type: "right", Color: "B0C4DE", Style: 1},
		},
	})

	gridStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Color: "000000", Size: 11, Family: "Microsoft JhengHei"},
		Alignment: &excelize.Alignment{Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "CCCCCC", Style: 1},
			{Type: "top", Color: "CCCCCC", Style: 1},
			{Type: "bottom", Color: "CCCCCC", Style: 1},
			{Type: "right", Color: "CCCCCC", Style: 1},
		},
	})

	isFirst := true
	defaultSheet := f.GetSheetName(0)

	headers := []string{"日期", "車號", "里程數", "保養項目", "廠商", "金額", "備註", "簽名"}

	for _, v := range vehicles {
		sheetName := v.DisplayName
		if sheetName == "" {
			sheetName = v.PlateNo
		}

		if isFirst {
			f.SetSheetName(defaultSheet, sheetName)
			isFirst = false
		} else {
			f.NewSheet(sheetName)
		}

		// 表頭資訊
		f.SetCellValue(sheetName, "A1", fmt.Sprintf("長照交通接送 車輛定期維修保養紀錄表 (%s)", v.DisplayName))
		f.SetCellValue(sheetName, "A2", fmt.Sprintf("車牌號碼：%s", v.PlateNo))

		// 欄位標題
		for colIdx, h := range headers {
			cell, _ := excelize.CoordinatesToCellName(colIdx+1, 4)
			f.SetCellValue(sheetName, cell, h)
			f.SetCellStyle(sheetName, cell, cell, headerStyle)
		}

		// 預留 12 列空白手寫列（規格書 §8.3）
		for r := 5; r <= 16; r++ {
			f.SetCellValue(sheetName, fmt.Sprintf("B%d", r), v.PlateNo)
			for c := 1; c <= len(headers); c++ {
				cell, _ := excelize.CoordinatesToCellName(c, r)
				f.SetCellStyle(sheetName, cell, cell, gridStyle)
			}
			f.SetRowHeight(sheetName, r, 24)
		}

		f.SetColWidth(sheetName, "A", "A", 14)
		f.SetColWidth(sheetName, "B", "B", 14)
		f.SetColWidth(sheetName, "C", "C", 14)
		f.SetColWidth(sheetName, "D", "D", 26)
		f.SetColWidth(sheetName, "E", "E", 18)
		f.SetColWidth(sheetName, "F", "F", 12)
		f.SetColWidth(sheetName, "G", "G", 22)
		f.SetColWidth(sheetName, "H", "H", 14)
		f.SetSheetDimension(sheetName, "A1:H16")
	}

	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		return nil, fmt.Errorf("failed to write blank maintenance excel: %w", err)
	}

	return buf.Bytes(), nil
}

// ExcelRenderer 以 excelize 產生營運表單，是 ops 唯一接觸試算表 SDK 的地方。
type ExcelRenderer struct{}

// NewExcelRenderer 建立 ExcelRenderer 實例。
func NewExcelRenderer() ExcelRenderer { return ExcelRenderer{} }
