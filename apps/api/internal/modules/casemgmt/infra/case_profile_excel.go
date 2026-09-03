package infra

import (
	"bytes"

	"github.com/xuri/excelize/v2"
	"ltc-system/apps/api/internal/modules/casemgmt/app"
)

// RenderCaseProfileWorkbook 匯出與來源工作簿一致的個案彙整欄位。
// 純粹負責試算表渲染，不做資料查詢或解密（那是呼叫端的職責）。
func (ExcelRenderer) RenderCaseProfileWorkbook(rows []app.CaseProfileRow) ([]byte, error) {
	f := excelize.NewFile()
	defer f.Close()
	sheet := "進系統個案個資"
	f.SetSheetName("Sheet1", sheet)
	// A 至 N 嚴格沿用來源工作表的表頭與欄位位置。
	headers := []string{"姓名", "戶別", "身分證字號", "性別", "生日", "歲數", "單位", "接送車輛(去)", "接送車輛(回)", "個管or照專", "姓名", "戶籍", "居住地", "備註"}
	for i, value := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		_ = f.SetCellValue(sheet, cell, value)
	}
	style, _ := f.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true}, Alignment: &excelize.Alignment{Horizontal: "center"}})
	_ = f.SetCellStyle(sheet, "A1", "N1", style)
	for i, item := range rows {
		row := []interface{}{
			item.Name, item.HouseholdType, item.NationalID, item.Gender, item.Birthday, item.Age,
			item.SiteName, item.OutboundVehicle, item.InboundVehicle, item.CareContactRole, item.CareContactName,
			item.RegisteredAddress, item.HomeAddress, "",
		}
		for j, cellValue := range row {
			cell, _ := excelize.CoordinatesToCellName(j+1, i+2)
			_ = f.SetCellValue(sheet, cell, cellValue)
		}
	}
	_ = f.SetColWidth(sheet, "A", "N", 18)
	_ = f.SetColWidth(sheet, "L", "M", 42)
	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// ExcelRenderer 以 excelize 產生個案彙整表，是 casemgmt 唯一接觸試算表 SDK 的地方。
type ExcelRenderer struct{}

// NewExcelRenderer 建立 ExcelRenderer 實例。
func NewExcelRenderer() ExcelRenderer { return ExcelRenderer{} }
