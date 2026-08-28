package export

import (
	"bytes"
	"fmt"
	"strings"

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

func sanitizeSheetName(name string, fallback string) string {
	invalidChars := []string{"\\", "/", "?", "*", ":", "[", "]", "'"}
	for _, ch := range invalidChars {
		name = strings.ReplaceAll(name, ch, "_")
	}
	name = strings.TrimSpace(name)
	if name == "" {
		name = fallback
	}
	runes := []rune(name)
	if len(runes) > 31 {
		name = string(runes[:31])
	}
	return name
}

// GenerateTripSummaryExcel 產生車輛趟數表 Excel 檔案。
func GenerateTripSummaryExcel(periodYM string, vehicles []TripSummaryExportVehicle) ([]byte, error) {
	f := excelize.NewFile()
	defer f.Close()

	defaultSheet := "Sheet1"
	isFirst := true

	headerStyle, err := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "FFFFFF", Size: 11, Family: "Microsoft JhengHei"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"1D5B79"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create header style: %w", err)
	}

	subtotalStyle, err := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "1D5B79", Size: 11, Family: "Microsoft JhengHei"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"E1EFF5"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "right", Vertical: "center"},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create subtotal style: %w", err)
	}

	if len(vehicles) == 0 {
		f.SetCellValue(defaultSheet, "A1", "長照交通接送 車輛趟數表")
		f.SetCellValue(defaultSheet, "A2", fmt.Sprintf("月份：%s", periodYM))
		f.SetCellValue(defaultSheet, "A4", "此月份與條件下無搭乘紀錄")
		if err := f.SetSheetDimension(defaultSheet, "A1:E4"); err != nil {
			return nil, fmt.Errorf("failed to set empty trip summary dimension: %w", err)
		}
	}

	usedSheetNames := make(map[string]bool)
	for vIdx, v := range vehicles {
		rawName := v.VehicleName
		if rawName == "" {
			rawName = v.PlateNo
		}
		if rawName == "" {
			rawName = fmt.Sprintf("車輛%d", vIdx+1)
		}

		sheetName := sanitizeSheetName(rawName, fmt.Sprintf("車輛%d", vIdx+1))
		if usedSheetNames[sheetName] {
			sheetName = sanitizeSheetName(fmt.Sprintf("%s_%s", sheetName, v.PlateNo), fmt.Sprintf("車輛%d", vIdx+1))
		}
		usedSheetNames[sheetName] = true

		if isFirst {
			if err := f.SetSheetName(defaultSheet, sheetName); err != nil {
				return nil, fmt.Errorf("failed to rename trip summary sheet: %w", err)
			}
			isFirst = false
		} else {
			if _, err := f.NewSheet(sheetName); err != nil {
				return nil, fmt.Errorf("failed to create trip summary sheet %q: %w", sheetName, err)
			}
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
		if err := f.SetSheetDimension(sheetName, fmt.Sprintf("A1:E%d", rowNum)); err != nil {
			return nil, fmt.Errorf("failed to set trip summary sheet dimension: %w", err)
		}
	}

	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		return nil, fmt.Errorf("failed to write excel buffer: %w", err)
	}

	return buf.Bytes(), nil
}
