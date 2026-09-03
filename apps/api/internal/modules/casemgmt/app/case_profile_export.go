package app

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"ltc-system/apps/api/internal/domain/crypto"
)

// GenerateCaseProfileWorkbook 匯出與來源工作簿一致的個案彙整欄位。
// caseIDs 為空時匯出全部個案；非空時只匯出指定個案，欄位與順序不受影響。
func (s *CaseService) GenerateCaseProfileWorkbook(ctx context.Context, caseIDs []uuid.UUID) ([]byte, error) {
	cases, _, err := s.caseRepo.List(ctx, "", "", "", 1, 10000, false, false)
	if err != nil {
		return nil, fmt.Errorf("list case profiles: %w", err)
	}

	if len(caseIDs) > 0 {
		wanted := make(map[uuid.UUID]struct{}, len(caseIDs))
		for _, id := range caseIDs {
			wanted[id] = struct{}{}
		}
		filtered := make([]Case, 0, len(caseIDs))
		for _, item := range cases {
			if _, ok := wanted[item.ID]; ok {
				filtered = append(filtered, item)
			}
		}
		cases = filtered
	}

	rows := make([]CaseProfileRow, 0, len(cases))
	for _, item := range cases {
		id := ""
		if len(item.NationalIDCipher) > 0 {
			id, err = crypto.Decrypt(item.NationalIDCipher, s.cfg.EncryptionKey)
			if err != nil {
				return nil, fmt.Errorf("decrypt case %s: %w", item.ID, err)
			}
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
			HomeAddress:       value(item.HomeAddress),
		})
	}

	return s.renderer.RenderCaseProfileWorkbook(rows)
}
