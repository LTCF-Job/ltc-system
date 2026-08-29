package infra

import (
	"bytes"
	"fmt"

	"github.com/xuri/excelize/v2"
	"ltc-system/apps/api/internal/domain/govform"
)

// RenderGovClaim 使用 excelize/v2 產生完全符合政府規範之申報 Excel 檔案位元組。
func (ExcelRenderer) RenderGovClaim(rows []govform.ClaimRow) ([]byte, error) {
	f := excelize.NewFile()
	defer f.Close()

	// 確保預設工作表名稱更名為「工作表1」
	defaultSheet := f.GetSheetName(0)
	if err := f.SetSheetName(defaultSheet, govform.GovClaimSheetName); err != nil {
		return nil, fmt.Errorf("failed to rename gov claim sheet: %w", err)
	}

	// 1. 寫入第 1 列標題（33 欄）
	for colIdx, header := range govform.Headers33 {
		cellAxis, err := excelize.CoordinatesToCellName(colIdx+1, 1)
		if err != nil {
			return nil, fmt.Errorf("failed to get header cell axis: %w", err)
		}
		if err := f.SetCellValue(govform.GovClaimSheetName, cellAxis, header); err != nil {
			return nil, fmt.Errorf("failed to set header cell value: %w", err)
		}
	}

	// 2. 逐列寫入資料
	for rIdx, row := range rows {
		excelRow := rIdx + 2
		for cIdx, val := range row.Cells {
			cellAxis, err := excelize.CoordinatesToCellName(cIdx+1, excelRow)
			if err != nil {
				return nil, fmt.Errorf("failed to get data cell axis: %w", err)
			}

			// 嚴格型別寫入：數值型別使用數值儲存，空值與字串使用字串儲存
			if err := f.SetCellValue(govform.GovClaimSheetName, cellAxis, val); err != nil {
				return nil, fmt.Errorf("failed to set cell value at %s: %w", cellAxis, err)
			}
		}
	}

	lastRow := len(rows) + 1
	if lastRow < 1 {
		lastRow = 1
	}
	if err := f.SetSheetDimension(govform.GovClaimSheetName, fmt.Sprintf("A1:AG%d", lastRow)); err != nil {
		return nil, fmt.Errorf("failed to set gov claim sheet dimension: %w", err)
	}

	buf := new(bytes.Buffer)
	if err := f.Write(buf); err != nil {
		return nil, fmt.Errorf("failed to write excel buffer: %w", err)
	}

	return buf.Bytes(), nil
}
