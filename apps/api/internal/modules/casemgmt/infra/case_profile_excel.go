package export

import (
	"bytes"

	"github.com/xuri/excelize/v2"
)

// CaseProfileRow 代表個案彙整工作簿之單筆已解密資料，由呼叫端組裝完成後傳入渲染。
type CaseProfileRow struct {
	Name              string
	HouseholdType     string
	NationalID        string
	Gender            string
	Birthday          string
	Age               string
	SiteName          string
	OutboundVehicle   string
	InboundVehicle    string
	CareContactRole   string
	CareContactName   string
	RegisteredAddress string
	HomeAddress       string
}

// GenerateCaseProfileWorkbook 匯出與來源工作簿一致的個案彙整欄位。
// 純粹負責試算表渲染，不做資料查詢或解密（那是呼叫端的職責）。
func GenerateCaseProfileWorkbook(rows []CaseProfileRow) ([]byte, error) {
	f := excelize.NewFile()
	defer f.Close()
	sheet := "進系統個案個資"
	f.SetSheetName("Sheet1", sheet)
	// A 欄依操作需求補上序號；C 至 P 嚴格沿用來源工作表的表頭與欄位位置。
	headers := []string{"序號", "", "姓名", "戶別", "身分證字號", "性別", "生日", "歲數", "據點", "接送車輛(去)", "接送車輛(回)", "個管or照專", "姓名", "戶籍", "居住地", "REMARK"}
	for i, value := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		_ = f.SetCellValue(sheet, cell, value)
	}
	style, _ := f.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true}, Alignment: &excelize.Alignment{Horizontal: "center"}})
	_ = f.SetCellStyle(sheet, "A1", "P1", style)
	for i, item := range rows {
		row := []interface{}{
			i + 1, i + 1, item.Name, item.HouseholdType, item.NationalID, item.Gender, item.Birthday, item.Age,
			item.SiteName, item.OutboundVehicle, item.InboundVehicle, item.CareContactRole, item.CareContactName,
			item.RegisteredAddress, item.HomeAddress, "",
		}
		for j, cellValue := range row {
			cell, _ := excelize.CoordinatesToCellName(j+1, i+2)
			_ = f.SetCellValue(sheet, cell, cellValue)
		}
	}
	_ = f.SetColWidth(sheet, "A", "P", 18)
	_ = f.SetColWidth(sheet, "N", "O", 42)
	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
