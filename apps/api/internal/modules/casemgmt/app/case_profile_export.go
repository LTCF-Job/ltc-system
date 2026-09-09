package app

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"ltc-system/apps/api/internal/domain/crypto"
	"ltc-system/apps/api/internal/platform/clock"
)

// GenerateCaseProfileWorkbook 匯出與來源工作簿一致的個案彙整欄位。
// caseIDs 為空時匯出全部個案；非空時只匯出指定個案，欄位與順序不受影響。
func (s *CaseService) GenerateCaseProfileWorkbook(ctx context.Context, caseIDs []uuid.UUID) ([]byte, error) {
	lister, ok := s.caseRepo.(interface {
		ListAll(ctx context.Context) ([]Case, error)
	})
	if !ok {
		return nil, fmt.Errorf("case repository does not support complete listing")
	}
	cases, err := lister.ListAll(ctx)
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
	for index, item := range cases {
		id := ""
		if len(item.NationalIDCipher) > 0 {
			id, err = crypto.Decrypt(item.NationalIDCipher, s.cfg.EncryptionKey)
			if err != nil {
				return nil, fmt.Errorf("decrypt case %s: %w", item.ID, err)
			}
		}
		birthday, age := "", ""
		if item.BirthDate != nil {
			// 生日輸出西元 YYYY/MM/DD；匯入端的 parseProfileBirthDate 以「年份小於 1911 才視為民國」
			// 判定，因此這個格式可以原樣回灌，不會被誤讀成民國年。
			birthday = item.BirthDate.Format("2006/01/02")
			age = fmt.Sprintf("%d", clock.Now().Year()-item.BirthDate.Year())
		}
		value := func(v *string) string {
			if v == nil {
				return ""
			}
			return *v
		}
		rows = append(rows, CaseProfileRow{
			Seq:           fmt.Sprintf("%d", index+1),
			Name:          item.Name,
			HouseholdType: value(item.HouseholdType),
			NationalID:    id,
			Gender:        value(item.Gender),
			Birthday:      birthday,
			Age:           age,
			SiteName:      item.SiteName,
			// 接送車輛的去回程欄位保留版面但暫不使用，兩欄一律留白。
			OutboundVehicle:   "",
			InboundVehicle:    "",
			CareContactRole:   caregiverRoleLabel(item.CaregiverType),
			CareContactName:   item.CaregiverName,
			RegisteredAddress: value(item.RegisteredAddress),
			HomeAddress:       value(item.HomeAddress),
			Remarks:           value(item.Remarks),
		})
	}

	return s.renderer.RenderCaseProfileWorkbook(rows)
}

// caregiverRoleLabel 把 caregivers.type 轉為工作表使用的中文角色；未設定類型時留白。
func caregiverRoleLabel(caregiverType string) string {
	switch caregiverType {
	case "case_manager":
		return "個管"
	case "specialist":
		return "照專"
	default:
		return ""
	}
}
