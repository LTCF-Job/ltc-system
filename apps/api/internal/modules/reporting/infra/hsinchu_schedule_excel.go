package infra

import (
	"bytes"
	"fmt"

	"github.com/xuri/excelize/v2"
	"ltc-system/apps/api/internal/modules/reporting/app"
)

// RenderHsinchuSchedule 產生符合規格書 §8.2 的新竹接送時刻表 Excel 檔案。
func (ExcelRenderer) RenderHsinchuSchedule(outbound, inbound []app.HsinchuScheduleItem) ([]byte, error) {
	f := excelize.NewFile()
	defer f.Close()

	sheetName := "新竹接送時刻表"
	defaultSheet := f.GetSheetName(0)
	if err := f.SetSheetName(defaultSheet, sheetName); err != nil {
		return nil, fmt.Errorf("failed to rename hsinchu schedule sheet: %w", err)
	}

	titleStyle, err := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 14, Color: "1D5B79", Family: "Microsoft JhengHei"},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create title style: %w", err)
	}

	sectionHeaderStyle, err := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "FFFFFF", Size: 11, Family: "Microsoft JhengHei"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"1D5B79"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create section header style: %w", err)
	}

	runHeaderStyle, err := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "1D5B79", Size: 11, Family: "Microsoft JhengHei"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"EAF2F8"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create run header style: %w", err)
	}

	// 第 1 列：主標題
	f.SetCellValue(sheetName, "A1", "長照交通接送 新竹區搭車順序時刻表")
	f.MergeCell(sheetName, "A1", "H1")
	f.SetCellStyle(sheetName, "A1", "H1", titleStyle)

	currentRow := 3

	writeSection := func(directionTitle string, items []app.HsinchuScheduleItem) {
		if len(items) == 0 {
			return
		}

		headers := []string{directionTitle, "趟次", "姓名", "備註", "出發時間", "出發地", "抵達時間", "目的地"}
		for colIdx, h := range headers {
			cell, _ := excelize.CoordinatesToCellName(colIdx+1, currentRow)
			f.SetCellValue(sheetName, cell, h)
			f.SetCellStyle(sheetName, cell, cell, sectionHeaderStyle)
		}
		currentRow++

		currentRun := int16(-1)
		for _, item := range items {
			var runText string
			if item.RunNo != currentRun {
				currentRun = item.RunNo
				runText = fmt.Sprintf("第%d趟", currentRun)
			} else {
				runText = "" // 僅在該趟第一列出現
			}

			noteVal := ""
			if item.Note != nil {
				noteVal = *item.Note
			}
			arriveVal := ""
			if item.ArriveTime != nil {
				arriveVal = *item.ArriveTime
			}

			f.SetCellValue(sheetName, fmt.Sprintf("A%d", currentRow), directionTitle)
			f.SetCellValue(sheetName, fmt.Sprintf("B%d", currentRow), runText)
			f.SetCellValue(sheetName, fmt.Sprintf("C%d", currentRow), item.CaseName)
			f.SetCellValue(sheetName, fmt.Sprintf("D%d", currentRow), noteVal)
			f.SetCellValue(sheetName, fmt.Sprintf("E%d", currentRow), item.DepartTime)
			f.SetCellValue(sheetName, fmt.Sprintf("F%d", currentRow), item.Origin)
			f.SetCellValue(sheetName, fmt.Sprintf("G%d", currentRow), arriveVal)
			f.SetCellValue(sheetName, fmt.Sprintf("H%d", currentRow), item.Destination)

			if runText != "" {
				cell, _ := excelize.CoordinatesToCellName(2, currentRow)
				f.SetCellStyle(sheetName, cell, cell, runHeaderStyle)
			}
			currentRow++
		}
		currentRow += 2
	}

	writeSection("去程", outbound)
	writeSection("回程", inbound)

	f.SetColWidth(sheetName, "A", "B", 12)
	f.SetColWidth(sheetName, "C", "C", 16)
	f.SetColWidth(sheetName, "D", "D", 20)
	f.SetColWidth(sheetName, "E", "E", 14)
	f.SetColWidth(sheetName, "F", "F", 30)
	f.SetColWidth(sheetName, "G", "G", 14)
	f.SetColWidth(sheetName, "H", "H", 30)
	if err := f.SetSheetDimension(sheetName, fmt.Sprintf("A1:H%d", currentRow)); err != nil {
		return nil, fmt.Errorf("failed to set hsinchu schedule sheet dimension: %w", err)
	}

	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		return nil, fmt.Errorf("failed to write hsinchu schedule excel: %w", err)
	}

	return buf.Bytes(), nil
}
