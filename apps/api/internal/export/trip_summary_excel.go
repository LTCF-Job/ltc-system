package export

import (
	"bytes"
	"fmt"

	"github.com/xuri/excelize/v2"
)

// TripSummaryExportCaseRow 代表單一個案之趟數資料。
type TripSummaryExportCaseRow struct {
	CaseCode      string
	CaseName      string
	OutboundCount int
	InboundCount  int
	TotalCount    int
}

// TripSummaryExportVehicle 代表單一車輛之趟數資料。
type TripSummaryExportVehicle struct {
	VehicleName      string
	PlateNo          string
	Rows             []TripSummaryExportCaseRow
	SubtotalOutbound int
	SubtotalInbound  int
	SubtotalTotal    int
}

// GenerateTripSummaryExcel 產生車輛趟數表 Excel 檔案。
func GenerateTripSummaryExcel(periodYM string, vehicles []TripSummaryExportVehicle) ([]byte, error) {
	f := excelize.NewFile()
	defer f.Close()

	defaultSheet := "Sheet1"
	isFirst := true

	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "#FFFFFF"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#1D5B79"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
	})

	subtotalStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#E1EFF5"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "right"},
	})

	if len(vehicles) == 0 {
		f.SetCellValue(defaultSheet, "A1", "此月份與條件下無搭乘紀錄")
	}

	for _, v := range vehicles {
		sheetName := v.VehicleName
		if isFirst {
			f.SetSheetName(defaultSheet, sheetName)
			isFirst = false
		} else {
			f.NewSheet(sheetName)
		}

		// 表頭
		f.SetCellValue(sheetName, "A1", fmt.Sprintf("長照交通接送 車輛趟數表 (%s)", periodYM))
		f.SetCellValue(sheetName, "A2", fmt.Sprintf("車輛名稱：%s (%s)", v.VehicleName, v.PlateNo))

		headers := []string{"個案編號", "個案姓名", "去程趟數", "回程趟數", "個人合計"}
		for colIdx, h := range headers {
			cell, _ := excelize.CoordinatesToCellName(colIdx+1, 4)
			f.SetCellValue(sheetName, cell, h)
			f.SetCellStyle(sheetName, cell, cell, headerStyle)
		}

		rowNum := 5
		for _, r := range v.Rows {
			f.SetCellValue(sheetName, fmt.Sprintf("A%d", rowNum), r.CaseCode)
			f.SetCellValue(sheetName, fmt.Sprintf("B%d", rowNum), r.CaseName)
			f.SetCellValue(sheetName, fmt.Sprintf("C%d", rowNum), r.OutboundCount)
			f.SetCellValue(sheetName, fmt.Sprintf("D%d", rowNum), r.InboundCount)
			f.SetCellValue(sheetName, fmt.Sprintf("E%d", rowNum), r.TotalCount)
			rowNum++
		}

		// 車輛小計列
		f.SetCellValue(sheetName, fmt.Sprintf("A%d", rowNum), "車輛小計")
		f.SetCellValue(sheetName, fmt.Sprintf("B%d", rowNum), "")
		f.SetCellValue(sheetName, fmt.Sprintf("C%d", rowNum), v.SubtotalOutbound)
		f.SetCellValue(sheetName, fmt.Sprintf("D%d", rowNum), v.SubtotalInbound)
		f.SetCellValue(sheetName, fmt.Sprintf("E%d", rowNum), v.SubtotalTotal)
		for c := 1; c <= 5; c++ {
			cell, _ := excelize.CoordinatesToCellName(c, rowNum)
			f.SetCellStyle(sheetName, cell, cell, subtotalStyle)
		}

		f.SetColWidth(sheetName, "A", "A", 16)
		f.SetColWidth(sheetName, "B", "B", 18)
		f.SetColWidth(sheetName, "C", "E", 14)
	}

	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		return nil, fmt.Errorf("failed to write excel buffer: %w", err)
	}

	return buf.Bytes(), nil
}
