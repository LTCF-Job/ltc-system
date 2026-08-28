package service

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/xuri/excelize/v2"
	"ltc-system/apps/api/internal/domain/crypto"
)

// GenerateCaseProfileWorkbook 匯出與來源工作簿一致的個案彙整欄位。
func (s *MasterService) GenerateCaseProfileWorkbook(ctx context.Context) ([]byte, error) {
	cases, _, err := s.caseRepo.List(ctx, "", "", "", 1, 10000)
	if err != nil {
		return nil, fmt.Errorf("list case profiles: %w", err)
	}
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
	for i, item := range cases {
		id, err := crypto.Decrypt(item.NationalIDCipher, s.cfg.EncryptionKey)
		if err != nil {
			return nil, fmt.Errorf("decrypt case %s: %w", item.ID, err)
		}
		birthday, age := "", ""
		if item.BirthDate != nil {
			birthday = fmt.Sprintf("%03d/%02d/%02d", item.BirthDate.Year()-1911, item.BirthDate.Month(), item.BirthDate.Day())
			age = fmt.Sprintf("%d", time.Now().Year()-item.BirthDate.Year())
		}
		value := func(v *string) string {
			if v == nil {
				return ""
			}
			return *v
		}
		row := []interface{}{i + 1, i + 1, item.Name, value(item.HouseholdType), id, value(item.Gender), birthday, age, item.SiteName, item.OutboundVehicle, item.InboundVehicle, value(item.CareContactRole), value(item.CareContactName), value(item.RegisteredAddress), item.HomeAddress, ""}
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
