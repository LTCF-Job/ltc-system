package infra

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/xuri/excelize/v2"
	"ltc-system/apps/api/internal/modules/reporting/app"
)

// rocMonthLabel 把民國 5 碼（11507）轉成表頭用的 115-07。
func rocMonthLabel(periodYM string) string {
	if len(periodYM) != 5 {
		return periodYM
	}
	return periodYM[:3] + "-" + periodYM[3:]
}

// RenderSiteTripSummary 產生據點趟數彙總表：一據點一張工作表，每位個案一列，
// 每個選取月份一欄，最後一欄為跨月合計，末列為據點小計。
func (ExcelRenderer) RenderSiteTripSummary(report app.SiteTripSummaryReport) ([]byte, error) {
	f := excelize.NewFile()
	defer f.Close()

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

	monthLabels := make([]string, 0, len(report.PeriodYMs))
	for _, periodYM := range report.PeriodYMs {
		monthLabels = append(monthLabels, rocMonthLabel(periodYM))
	}
	// 個案姓名 + 各月份 + 跨月合計
	lastCol := len(report.PeriodYMs) + 2

	defaultSheet := f.GetSheetName(0)
	isFirst := true

	if len(report.Sites) == 0 {
		f.SetCellValue(defaultSheet, "A1", "長照交通接送 據點趟數彙總表")
		f.SetCellValue(defaultSheet, "A2", fmt.Sprintf("統計月份：%s", strings.Join(monthLabels, "、")))
		f.SetCellValue(defaultSheet, "A4", "所選據點底下沒有可統計的個案")
		if err := f.SetSheetDimension(defaultSheet, "A1:E4"); err != nil {
			return nil, fmt.Errorf("failed to set empty site trip summary dimension: %w", err)
		}
	}

	usedSheetNames := make(map[string]bool)
	for sIdx, site := range report.Sites {
		fallback := fmt.Sprintf("據點%d", sIdx+1)
		sheetName := sanitizeSheetName(site.SiteName, fallback)
		if usedSheetNames[sheetName] {
			sheetName = sanitizeSheetName(fmt.Sprintf("%s_%d", sheetName, sIdx+1), fallback)
		}
		usedSheetNames[sheetName] = true

		if isFirst {
			if err := f.SetSheetName(defaultSheet, sheetName); err != nil {
				return nil, fmt.Errorf("failed to rename site trip summary sheet: %w", err)
			}
			isFirst = false
		} else {
			if _, err := f.NewSheet(sheetName); err != nil {
				return nil, fmt.Errorf("failed to create site trip summary sheet %q: %w", sheetName, err)
			}
		}

		f.SetCellValue(sheetName, "A1", "長照交通接送 據點趟數彙總表")
		siteLabel := site.SiteName
		if site.Region != "" {
			siteLabel = fmt.Sprintf("%s（%s）", site.SiteName, site.Region)
		}
		f.SetCellValue(sheetName, "A2", fmt.Sprintf("據點：%s", siteLabel))
		f.SetCellValue(sheetName, "A3", fmt.Sprintf("統計月份：%s", strings.Join(monthLabels, "、")))

		headers := append([]string{"個案姓名"}, monthLabels...)
		headers = append(headers, "跨月合計")
		for colIdx, h := range headers {
			cell, err := excelize.CoordinatesToCellName(colIdx+1, 5)
			if err != nil {
				return nil, fmt.Errorf("failed to get header cell axis: %w", err)
			}
			f.SetCellValue(sheetName, cell, h)
			f.SetCellStyle(sheetName, cell, cell, headerStyle)
		}

		rowNum := 6
		for _, row := range site.Rows {
			f.SetCellValue(sheetName, fmt.Sprintf("A%d", rowNum), row.CaseName)
			for i, periodYM := range report.PeriodYMs {
				cell, err := excelize.CoordinatesToCellName(i+2, rowNum)
				if err != nil {
					return nil, fmt.Errorf("failed to get data cell axis: %w", err)
				}
				f.SetCellValue(sheetName, cell, row.Counts[periodYM])
			}
			totalCell, err := excelize.CoordinatesToCellName(lastCol, rowNum)
			if err != nil {
				return nil, fmt.Errorf("failed to get total cell axis: %w", err)
			}
			f.SetCellValue(sheetName, totalCell, row.Total)
			rowNum++
		}

		// 據點小計列
		f.SetCellValue(sheetName, fmt.Sprintf("A%d", rowNum), "據點小計")
		for i, periodYM := range report.PeriodYMs {
			cell, err := excelize.CoordinatesToCellName(i+2, rowNum)
			if err != nil {
				return nil, fmt.Errorf("failed to get subtotal cell axis: %w", err)
			}
			f.SetCellValue(sheetName, cell, site.Subtotals[periodYM])
		}
		subtotalTotalCell, err := excelize.CoordinatesToCellName(lastCol, rowNum)
		if err != nil {
			return nil, fmt.Errorf("failed to get subtotal total cell axis: %w", err)
		}
		f.SetCellValue(sheetName, subtotalTotalCell, site.SubtotalTotal)
		for c := 1; c <= lastCol; c++ {
			cell, err := excelize.CoordinatesToCellName(c, rowNum)
			if err != nil {
				return nil, fmt.Errorf("failed to get subtotal style cell axis: %w", err)
			}
			f.SetCellStyle(sheetName, cell, cell, subtotalStyle)
		}

		lastColName, err := excelize.ColumnNumberToName(lastCol)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve last column name: %w", err)
		}
		f.SetColWidth(sheetName, "A", "A", 18)
		f.SetColWidth(sheetName, "B", lastColName, 14)
		if err := f.SetSheetDimension(sheetName, fmt.Sprintf("A1:%s%d", lastColName, rowNum)); err != nil {
			return nil, fmt.Errorf("failed to set site trip summary sheet dimension: %w", err)
		}
	}

	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		return nil, fmt.Errorf("failed to write excel buffer: %w", err)
	}
	return buf.Bytes(), nil
}
