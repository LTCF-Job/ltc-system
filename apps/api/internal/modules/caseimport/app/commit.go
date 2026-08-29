package app

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// stringPointer 將空字串轉為 nil，供選填欄位寫入時使用。
func stringPointer(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

// CommitCases 將通過檢核的個案資料正式寫入資料庫。
//
// 交易語意：每一列（個案主檔＋接送車輛偏好＋稽核紀錄）在單一事務中全有或全無
// 提交；列與列之間彼此獨立——某列寫入失敗只回滾該列自身的變更並記為略過列
// （附原因與原始欄位供人工補正），不影響已提交的前列，也不會中止整批匯入其餘
// 待處理的列。
//
// 選擇「逐列獨立事務」而非「整批單一事務＋逐列 savepoint」：匯入列彼此沒有
// 跨列一致性需求，逐列事務讓已成功的列立即落地可見、避免整批因單一長交易
// 持有鎖與可見性延遲，且與 CaseRepository 既有逐方法自管事務的慣例一致。
//
// includeDuplicateRows 是使用者於預覽階段勾選「仍要匯入」的列號集合；dry-run
// 標記為 IsDuplicate 的列若未在此集合中，直接記為略過，不寫入資料庫。
func (s *ImportService) CommitCases(ctx context.Context, preview *CaseImportPreviewResult, includeDuplicateRows map[int]bool, actor Actor) (*CaseImportCommitResult, error) {
	if preview == nil || len(preview.Rows) == 0 {
		return &CaseImportCommitResult{}, nil
	}
	if s.txRunner == nil {
		return nil, errors.New("import service: transaction runner not configured")
	}

	result := &CaseImportCommitResult{}
	for _, row := range preview.Rows {
		if row.ErrorMessage != "" {
			item := skippedRow(row)
			result.SkippedRows = append(result.SkippedRows, item)
			if s.cases != nil {
				s.cases.RecordSkipped(ctx, item, actor)
			}
			continue
		}

		if row.IsDuplicate && !includeDuplicateRows[row.RowIndex] {
			item := CaseImportSkippedRow{RowIndex: row.RowIndex, CaseName: row.Name, Reasons: []string{"偵測為重複個案，未勾選匯入"}, RawValues: row.RawValues}
			result.SkippedRows = append(result.SkippedRows, item)
			if s.cases != nil {
				s.cases.RecordSkipped(ctx, item, actor)
			}
			continue
		}

		caseReq := NewCase{
			Code:              "IMP-" + strings.ToUpper(uuid.New().String()[:8]),
			Name:              row.Name,
			NationalID:        row.NationalID,
			HouseholdType:     stringPointer(row.HouseholdType),
			Gender:            stringPointer(row.Gender),
			CareContactRole:   stringPointer(row.CareContactRole),
			CareContactName:   stringPointer(row.CareContactName),
			RegisteredAddress: stringPointer(row.RegisteredAddress),
			HomeAddress:       stringPointer(row.HomeAddress),
			Region:            stringPointer(row.Region),
			ServiceCategory:   row.ServiceCategory,
			ServiceUsageType:  row.ServiceUsageType,
			Status:            "active",
			Remarks:           stringPointer(row.Remarks),
		}
		if row.BirthDate != "" {
			if birthDate, err := time.Parse("2006-01-02", row.BirthDate); err == nil {
				caseReq.BirthDate = &birthDate
			}
		}
		if row.ClaimStartDate != "" {
			if claimStart, err := time.Parse("2006-01-02", row.ClaimStartDate); err == nil {
				caseReq.ClaimStartDate = &claimStart
			}
		}

		// 據點／去回程車輛各自獨立比對：比對到則寫入 ID，比對不到但有填名稱則保留
		// 原始名稱待人工關聯，兩種情況都不影響個案主檔本身的建立。
		siteID, siteNameRaw, siteWarning := s.resolveSite(ctx, row.SiteName)
		outboundID, outboundNameRaw, outboundWarning := s.resolveVehicle(ctx, row.OutboundVehicle, "接送車輛(去)")
		inboundID, inboundNameRaw, inboundWarning := s.resolveVehicle(ctx, row.InboundVehicle, "接送車輛(回)")
		for _, w := range []string{siteWarning, outboundWarning, inboundWarning} {
			if w != "" {
				result.Warnings = append(result.Warnings, CaseImportWarningItem{RowIndex: row.RowIndex, CaseName: row.Name, Message: w})
			}
		}

		txErr := s.txRunner.WithTx(ctx, func(txCtx context.Context) error {
			caseID, err := s.cases.CreateCase(txCtx, caseReq, actor)
			if err != nil {
				return fmt.Errorf("個案建立失敗：%w", err)
			}

			if siteID != nil || outboundID != nil || inboundID != nil || siteNameRaw != "" || outboundNameRaw != "" || inboundNameRaw != "" {
				if err := s.prefRepo.UpsertTransportPreference(txCtx, caseID, siteID, outboundID, inboundID, siteNameRaw, outboundNameRaw, inboundNameRaw); err != nil {
					return fmt.Errorf("儲存接送車輛偏好失敗：%w", err)
				}
			}

			return nil
		})

		if txErr != nil {
			item := CaseImportSkippedRow{RowIndex: row.RowIndex, CaseName: row.Name, Reasons: []string{txErr.Error()}, RawValues: row.RawValues}
			result.SkippedRows = append(result.SkippedRows, item)
			if s.cases != nil {
				s.cases.RecordSkipped(ctx, item, actor)
			}
			continue
		}

		result.ImportedCount++
	}

	return result, nil
}

// resolveSite 依名稱比對既有據點；查無資料時回傳空 ID 與原始名稱，並附上待人工關聯的提示。
func (s *ImportService) resolveSite(ctx context.Context, name string) (id *uuid.UUID, nameRaw string, warning string) {
	if name == "" || s.siteRepo == nil {
		return nil, "", ""
	}
	site, err := s.siteRepo.GetByName(ctx, name)
	if err != nil || site == nil {
		return nil, name, fmt.Sprintf("據點「%s」未於車輛/據點管理中找到，已建立個案並保留原始名稱待人工關聯", name)
	}
	return &site.ID, "", ""
}

// resolveVehicle 依顯示名稱比對既有車輛；查無資料時回傳空 ID 與原始名稱，並附上待人工關聯的提示。
func (s *ImportService) resolveVehicle(ctx context.Context, name, fieldLabel string) (id *uuid.UUID, nameRaw string, warning string) {
	if name == "" || s.vehicleRepo == nil {
		return nil, "", ""
	}
	vehicle, err := s.vehicleRepo.GetByDisplayName(ctx, name)
	if err != nil || vehicle == nil {
		return nil, name, fmt.Sprintf("%s『%s』未於車輛/據點管理中找到，已建立個案並保留原始名稱待人工關聯", fieldLabel, name)
	}
	return &vehicle.ID, "", ""
}

func skippedRow(row CaseImportRowResult) CaseImportSkippedRow {
	reasons := strings.Split(row.ErrorMessage, "；")
	return CaseImportSkippedRow{RowIndex: row.RowIndex, CaseName: row.Name, Reasons: reasons, RawValues: row.RawValues}
}
