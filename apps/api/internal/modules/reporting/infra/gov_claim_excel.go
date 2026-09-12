package infra

import (
	"bytes"
	"fmt"

	"github.com/xuri/excelize/v2"
	"ltc-system/apps/api/internal/domain/govform"
)

// 政府申報表的版面規格。數值取自主管機關提供的 .xls 範本實測值
// （字型 DFKai-SB 12pt、必填欄位紅字、表頭置中加框、資料列無框線），
// 讓系統產出的檔案與承辦人員慣用的範本在視覺上一致。
const (
	govClaimFontName      = "DFKai-SB" // 標楷體
	govClaimFontSize      = 12.0
	govClaimHeaderRowHigh = 99.0
	govClaimDataRowHigh   = 16.5
	// excelize 的 Color 需為 6 碼 RGB，它會自行補上 alpha 前綴；
	// 若在此再帶 alpha 會產出 10 碼色碼，Excel 以外的讀取器會判為非法 aRGB。
	govClaimRequiredColor = "FF0000" // 必填欄位標題紅字
	govClaimNormalColor   = "000000"
)

// govClaimColWidths 為 33 欄的欄寬（Excel 字元寬度單位），取自範本實測值。
var govClaimColWidths = [33]float64{
	14.62, 23.75, 14.62, 10.75, 10.12, 10.25,
	18.12, 16.87, 19.62, 16.87, 19.62, 16.87,
	18.12, 18.12, 18.12, 18.12, 15.37, 15.75,
	43.37, 19.37, 19.37, 19.37, 24.25, 17.37,
	39.62, 40.62, 18.62, 18.62, 18.62, 18.62,
	18.62, 18.75, 31.37,
}

// govClaimWrapFromCol 起（含）之後的標題才自動換行。範本前兩欄（身分證字號、
// 服務日期）標題較短且未設換行，維持一致以免欄寬被撐開。
const govClaimWrapFromCol = 3

// govClaimStyles 收納標題與資料列所需的樣式 ID。
type govClaimStyles struct {
	requiredNoWrap int
	requiredWrap   int
	normalWrap     int
	data           int
}

// headerStyle 依欄號（1-based）挑選對應的標題樣式。
func (s govClaimStyles) headerStyle(col int) int {
	if !govform.IsRequiredHeader(col) {
		return s.normalWrap
	}
	if col < govClaimWrapFromCol {
		return s.requiredNoWrap
	}
	return s.requiredWrap
}

// newGovClaimStyles 建立標題（必填／選填、換行與否）與資料列樣式。
// 標題四邊加細框線；資料列僅置中，與範本一致不加框線。
func newGovClaimStyles(f *excelize.File) (govClaimStyles, error) {
	border := []excelize.Border{
		{Type: "top", Color: govClaimNormalColor, Style: 1},
		{Type: "bottom", Color: govClaimNormalColor, Style: 1},
		{Type: "left", Color: govClaimNormalColor, Style: 1},
		{Type: "right", Color: govClaimNormalColor, Style: 1},
	}
	headerStyle := func(color string, wrap bool) (int, error) {
		return f.NewStyle(&excelize.Style{
			Font:      &excelize.Font{Family: govClaimFontName, Size: govClaimFontSize, Color: color},
			Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: wrap},
			Border:    border,
		})
	}

	var (
		styles govClaimStyles
		err    error
	)

	if styles.requiredNoWrap, err = headerStyle(govClaimRequiredColor, false); err != nil {
		return govClaimStyles{}, fmt.Errorf("failed to create required header style: %w", err)
	}
	if styles.requiredWrap, err = headerStyle(govClaimRequiredColor, true); err != nil {
		return govClaimStyles{}, fmt.Errorf("failed to create wrapped required header style: %w", err)
	}
	if styles.normalWrap, err = headerStyle(govClaimNormalColor, true); err != nil {
		return govClaimStyles{}, fmt.Errorf("failed to create normal header style: %w", err)
	}

	styles.data, err = f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Family: govClaimFontName, Size: govClaimFontSize, Color: govClaimNormalColor},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
	})
	if err != nil {
		return govClaimStyles{}, fmt.Errorf("failed to create data style: %w", err)
	}

	return styles, nil
}

// RenderGovClaim 使用 excelize/v2 產生完全符合政府規範之申報 Excel 檔案位元組。
func (ExcelRenderer) RenderGovClaim(rows []govform.ClaimRow) ([]byte, error) {
	f := excelize.NewFile()
	defer f.Close()

	// 確保預設工作表名稱更名為「工作表1」
	defaultSheet := f.GetSheetName(0)
	if err := f.SetSheetName(defaultSheet, govform.GovClaimSheetName); err != nil {
		return nil, fmt.Errorf("failed to rename gov claim sheet: %w", err)
	}

	styles, err := newGovClaimStyles(f)
	if err != nil {
		return nil, err
	}

	// 1. 寫入第 1 列標題（33 欄），必填欄位套用紅字樣式
	for colIdx, header := range govform.Headers33 {
		cellAxis, err := excelize.CoordinatesToCellName(colIdx+1, 1)
		if err != nil {
			return nil, fmt.Errorf("failed to get header cell axis: %w", err)
		}
		if err := f.SetCellValue(govform.GovClaimSheetName, cellAxis, header); err != nil {
			return nil, fmt.Errorf("failed to set header cell value: %w", err)
		}

		if err := f.SetCellStyle(govform.GovClaimSheetName, cellAxis, cellAxis, styles.headerStyle(colIdx+1)); err != nil {
			return nil, fmt.Errorf("failed to set header style at %s: %w", cellAxis, err)
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

	// 3. 套用版面：欄寬、表頭列高、資料列樣式與列高
	for colIdx, width := range govClaimColWidths {
		colName, err := excelize.ColumnNumberToName(colIdx + 1)
		if err != nil {
			return nil, fmt.Errorf("failed to get column name: %w", err)
		}
		if err := f.SetColWidth(govform.GovClaimSheetName, colName, colName, width); err != nil {
			return nil, fmt.Errorf("failed to set column width for %s: %w", colName, err)
		}
	}

	if err := f.SetRowHeight(govform.GovClaimSheetName, 1, govClaimHeaderRowHigh); err != nil {
		return nil, fmt.Errorf("failed to set header row height: %w", err)
	}

	if len(rows) > 0 {
		lastCell, err := excelize.CoordinatesToCellName(len(govform.Headers33), len(rows)+1)
		if err != nil {
			return nil, fmt.Errorf("failed to get last data cell axis: %w", err)
		}
		if err := f.SetCellStyle(govform.GovClaimSheetName, "A2", lastCell, styles.data); err != nil {
			return nil, fmt.Errorf("failed to set data style: %w", err)
		}
		for rIdx := range rows {
			if err := f.SetRowHeight(govform.GovClaimSheetName, rIdx+2, govClaimDataRowHigh); err != nil {
				return nil, fmt.Errorf("failed to set data row height: %w", err)
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
