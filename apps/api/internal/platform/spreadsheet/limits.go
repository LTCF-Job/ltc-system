// Package spreadsheet 提供各模組 infra 解碼試算表時共用的規模上限，
// 阻擋以壓縮炸彈（zip bomb）撐爆記憶體的上傳檔案。
package spreadsheet

import "fmt"

// 解析規模上限。數值依本系統實際匯入量訂定：個案主檔一次匯入數千列、司機接送匯報
// 單月一張工作表約 31 列，正常檔案皆遠低於此；上限僅用來擋掉解壓後暴增的惡意檔案。
const (
	MaxSheets       = 16
	MaxRowsPerSheet = 50000
	MaxTotalCells   = 2000000
)

// LimitCounter 在逐列讀取試算表的過程中累計規模並即時攔截超限檔案。零值即可使用。
// 超限一律回報錯誤而非截斷，避免使用者以為匯入成功但資料不全。
type LimitCounter struct {
	sheets int
	rows   int
	cells  int
}

// BeginSheet 開始累計一張新的工作表，工作表數超過上限時回傳錯誤。
func (c *LimitCounter) BeginSheet() error {
	c.sheets++
	c.rows = 0
	if c.sheets > MaxSheets {
		return fmt.Errorf("Excel 檔案工作表數超過上限 %d 張，請精簡後再匯入", MaxSheets)
	}
	return nil
}

// AddRow 累計目前工作表的一列，列數或總儲存格數超過上限時回傳錯誤。
func (c *LimitCounter) AddRow(sheet string, cells int) error {
	c.rows++
	if c.rows > MaxRowsPerSheet {
		return fmt.Errorf("工作表「%s」列數超過上限 %d 列，請分批匯入", sheet, MaxRowsPerSheet)
	}
	c.cells += cells
	if c.cells > MaxTotalCells {
		return fmt.Errorf("Excel 檔案儲存格總數超過上限 %d 格，請分批匯入", MaxTotalCells)
	}
	return nil
}

// TrimTrailingEmptyRows 去除表格尾端的空白列，維持與 excelize GetRows 相同的輸出語意。
func TrimTrailingEmptyRows(rows [][]string) [][]string {
	end := len(rows)
	for end > 0 {
		blank := true
		for _, cell := range rows[end-1] {
			if cell != "" {
				blank = false
				break
			}
		}
		if !blank {
			break
		}
		end--
	}
	return rows[:end]
}
