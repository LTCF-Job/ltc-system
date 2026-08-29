package app

import (
	"context"
	"fmt"
	"time"

	"ltc-system/apps/api/internal/domain/crypto"
)

// GenerateCaseProfileWorkbook 匯出與來源工作簿一致的個案彙整欄位。
func (s *CaseService) GenerateCaseProfileWorkbook(ctx context.Context) ([]byte, error) {
	cases, _, err := s.caseRepo.List(ctx, "", "", "", 1, 10000)
	if err != nil {
		return nil, fmt.Errorf("list case profiles: %w", err)
	}

	rows := make([]CaseProfileRow, 0, len(cases))
	for _, item := range cases {
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
		rows = append(rows, CaseProfileRow{
			Name:              item.Name,
			HouseholdType:     value(item.HouseholdType),
			NationalID:        id,
			Gender:            value(item.Gender),
			Birthday:          birthday,
			Age:               age,
			SiteName:          item.SiteName,
			OutboundVehicle:   item.OutboundVehicle,
			InboundVehicle:    item.InboundVehicle,
			CareContactRole:   value(item.CareContactRole),
			CareContactName:   value(item.CareContactName),
			RegisteredAddress: value(item.RegisteredAddress),
			HomeAddress:       item.HomeAddress,
		})
	}

	return s.renderer.RenderCaseProfileWorkbook(rows)
}
