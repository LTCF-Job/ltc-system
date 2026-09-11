package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
)

var ErrCaseImportAlreadyCommitted = errors.New("case import row already committed")

// caseRegistrarDuplicateNationalIDText 比對 CaseRegistrar.CreateCase 回傳的底層錯誤文字，
// 用以辨識「身分證字號已存在」這個情境。CaseRegistrar 是跨模組邊界（見 ports.go），
// caseimport 不得直接 import casemgmt 取得其 ErrDuplicateNationalID sentinel 做 errors.Is
// 比對（違反 layering-rules.md 的模組邊界），只能靠錯誤文字判斷；文字需與
// casemgmt/app/case_service.go 的 ErrDuplicateNationalID 保持一致。
const caseRegistrarDuplicateNationalIDText = "national id already exists"

// stringPointer 將空字串轉為 nil，供選填欄位寫入時使用。
func stringPointer(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

// CommitCases 將通過檢核的個案資料以逐列獨立事務正式寫入資料庫，單列失敗僅回滾該列並記為略過，不影響其餘列。
// 疑似重複個案一律建立為待裁決暫存列，不直接建立個案；生日/身分證字號格式錯誤不擋列，
// 個案照常建立並標記待補正，只有系統性查詢失敗（ErrorMessage 非空）才整列擋下。
func (s *ImportService) CommitCases(ctx context.Context, preview *CaseImportPreviewResult, actor Actor) (*CaseImportCommitResult, error) {
	if preview == nil || len(preview.Rows) == 0 {
		return &CaseImportCommitResult{}, nil
	}
	if s.txRunner == nil {
		return nil, errors.New("import service: transaction runner not configured")
	}

	result := &CaseImportCommitResult{
		SkippedRows: []CaseImportSkippedRow{},
		FailedRows:  []CaseImportSkippedRow{},
	}
	recordSkipped := func(row CaseImportSkippedRow) {
		result.SkippedRows = append(result.SkippedRows, row)
		if s.cases != nil {
			s.cases.RecordSkipped(ctx, row, actor)
		}
	}
	recordFailed := func(row CaseImportSkippedRow) {
		result.FailedCount++
		result.FailedRows = append(result.FailedRows, row)
		if s.cases != nil {
			s.cases.RecordSkipped(ctx, row, actor)
		}
	}
	for _, row := range preview.Rows {
		if row.ErrorMessage != "" {
			recordSkipped(skippedRow(row))
			continue
		}

		// 重傳同一份檔案時，先前建立的個案會在 re-parse 被自己判成疑似重複；
		// 冪等鍵必須比重複分支先判，否則那些列會變成假的待裁決項目。
		if s.idempotency != nil && preview.FileHash != "" {
			committed, err := s.idempotency.IsCaseImportRowCommitted(ctx, preview.FileHash, importRowKey(row.RowID, row.RowIndex))
			if err != nil {
				slog.Error("case import idempotency lookup failed", "row_index", row.RowIndex, "error", err)
				recordFailed(caseImportFailureRow(row, "匯入紀錄查詢失敗，請稍後重試"))
				continue
			}
			if committed {
				result.AlreadyImportedCount++
				recordSkipped(caseImportFailureRow(row, "此檔案的此列已完成匯入，略過重試"))
				continue
			}
		}

		// 據點與照護人員各自獨立比對：比對到則寫入 ID，比對不到但有填名稱則保留
		// 原始名稱待人工關聯，兩種情況都不影響個案主檔本身的建立。
		siteID, siteNameRaw, siteWarning, err := s.resolveSite(ctx, row.SiteName)
		if err != nil {
			slog.Error("case import site lookup failed", "row_index", row.RowIndex, "error", err)
			recordFailed(caseImportFailureRow(row, "據點查詢失敗，請稍後重試"))
			continue
		}
		caregiverID, caregiverRole, caregiverWarning, err := s.resolveCaregiver(ctx, row.CareContactName, row.CareContactRole)
		if err != nil {
			slog.Error("case import caregiver lookup failed", "row_index", row.RowIndex, "error", err)
			recordFailed(caseImportFailureRow(row, "照護人員查詢失敗，請稍後重試"))
			continue
		}
		for _, w := range []string{siteWarning, caregiverWarning} {
			if w != "" {
				result.Warnings = append(result.Warnings, CaseImportWarningItem{RowIndex: row.RowIndex, CaseName: row.Name, Message: w})
			}
		}

		if row.IsDuplicate {
			if s.duplicateStager == nil {
				recordFailed(caseImportFailureRow(row, "重複個案待裁決功能尚未設定"))
				continue
			}
			if row.DuplicateCaseID == nil {
				slog.Error("case import duplicate row missing duplicate case id", "row_index", row.RowIndex)
				recordFailed(caseImportFailureRow(row, "重複個案資料不完整，請重新整理後重試"))
				continue
			}
			_, alreadyStaged, err := s.duplicateStager.StageDuplicateRow(ctx, preview.FileHash, importRowKey(row.RowID, row.RowIndex), StageDuplicateCandidate{
				RowIndex:          row.RowIndex,
				SheetName:         row.SheetName,
				Name:              row.Name,
				NationalID:        row.NationalID,
				HouseholdType:     stringPointer(row.HouseholdType),
				Gender:            stringPointer(row.Gender),
				BirthDate:         parseBirthDate(row.BirthDate),
				BirthDateRaw:      stringPointer(row.BirthDateRaw),
				CareContactRole:   stringPointer(caregiverRole),
				CareContactName:   stringPointer(row.CareContactName),
				RegisteredAddress: stringPointer(row.RegisteredAddress),
				HomeAddress:       stringPointer(row.HomeAddress),
				ServiceCategory:   row.ServiceCategory,
				ServiceUsageType:  row.ServiceUsageType,
				Remarks:           stringPointer(row.Remarks),
				SiteID:            siteID,
				SiteNameRaw:       siteNameRaw,
				CaregiverID:       caregiverID,
				DuplicateCaseID:   *row.DuplicateCaseID,
			})
			if err != nil {
				slog.Error("case import duplicate staging failed", "row_index", row.RowIndex, "error", err)
				recordFailed(caseImportFailureRow(row, "疑似重複個案暫存失敗，請稍後重試"))
				continue
			}
			if alreadyStaged {
				// 與「此列已完成匯入」同一組語意：AlreadyImportedCount 的每一筆都要同時
				// 記進 SkippedRows，前端的略過筆數是用兩者相減算出來的
				result.AlreadyImportedCount++
				recordSkipped(caseImportFailureRow(row, "此檔案的此列已建立為待裁決項目，略過重試"))
			} else {
				result.StagedDuplicateCount++
			}
			continue
		}

		caseReq := NewCase{
			ID:                     uuid.New(),
			Name:                   row.Name,
			NationalID:             row.NationalID,
			AllowInvalidNationalID: true,
			HouseholdType:          stringPointer(row.HouseholdType),
			Gender:                 stringPointer(row.Gender),
			BirthDate:              parseBirthDate(row.BirthDate),
			BirthDateRaw:           stringPointer(row.BirthDateRaw),
			CareContactRole:        stringPointer(caregiverRole),
			CareContactName:        stringPointer(row.CareContactName),
			RegisteredAddress:      stringPointer(row.RegisteredAddress),
			HomeAddress:            stringPointer(row.HomeAddress),
			ServiceCategory:        row.ServiceCategory,
			ServiceUsageType:       row.ServiceUsageType,
			Status:                 "active",
			Remarks:                stringPointer(row.Remarks),
			SiteID:                 siteID,
			SiteNameRaw:            siteNameRaw,
			CaregiverID:            caregiverID,
		}

		txErr := s.txRunner.WithTx(ctx, func(txCtx context.Context) error {
			if s.idempotency != nil && preview.FileHash != "" {
				claimed, err := s.idempotency.ClaimCaseImportRow(txCtx, preview.FileHash, importRowKey(row.RowID, row.RowIndex), caseReq.ID)
				if err != nil {
					return fmt.Errorf("記錄匯入冪等鍵失敗：%w", err)
				}
				if !claimed {
					return ErrCaseImportAlreadyCommitted
				}
			}
			if s.cases == nil {
				return errors.New("case registrar not configured")
			}
			if _, err := s.cases.CreateCase(txCtx, caseReq, actor); err != nil {
				return fmt.Errorf("個案建立失敗：%w", err)
			}

			return nil
		})

		if txErr != nil {
			if errors.Is(txErr, ErrCaseImportAlreadyCommitted) {
				result.AlreadyImportedCount++
				item := caseImportFailureRow(row, "此檔案的此列已完成匯入，略過重試")
				recordSkipped(item)
			} else if strings.Contains(txErr.Error(), caseRegistrarDuplicateNationalIDText) {
				slog.Error("case import row transaction failed: duplicate national id", "row_index", row.RowIndex, "error", txErr)
				recordFailed(caseImportFailureRow(row, "身分證字號與本檔其他列重複"))
			} else {
				slog.Error("case import row transaction failed", "row_index", row.RowIndex, "error", txErr)
				recordFailed(caseImportFailureRow(row, "資料列匯入失敗，請檢查資料或稍後重試"))
			}
			continue
		}

		result.ImportedCount++
	}

	return result, nil
}

// parseBirthDate 將 "2006-01-02" 格式的生日字串轉為 *time.Time；解析失敗回傳 nil，
// 呼叫端改用 BirthDateRaw 保留原始字串。
func parseBirthDate(value string) *time.Time {
	if value == "" {
		return nil
	}
	if parsed, err := time.Parse("2006-01-02", value); err == nil {
		return &parsed
	}
	return nil
}

// sitePendingPlaceholder 是據點欄位完全空白時的哨兵值，與 migration 000044 對既有資料
// 使用的佔位字串一致；cases.site_id 與 site_name_raw 不可同時為 NULL（ck_cases_site_present），
// 若在此直接回傳空字串會讓建立個案時違反該約束，因此空白一律視為「待補齊」而非「無需處理」。
const sitePendingPlaceholder = "（待補齊據點）"

// resolveSite 依名稱比對既有據點；查無資料時回傳空 ID 與原始名稱，並附上待人工關聯的提示。
// 完全空白時同樣落入待維護，不可回傳空字串（見 sitePendingPlaceholder 說明）。
func (s *ImportService) resolveSite(ctx context.Context, name string) (id *uuid.UUID, nameRaw string, warning string, err error) {
	if name == "" {
		return nil, sitePendingPlaceholder, "據點未填寫，已建立個案並列入待維護，補齊據點後即可離開待維護清單", nil
	}
	if s.siteRepo == nil {
		return nil, name, "", nil
	}
	site, err := s.siteRepo.GetByName(ctx, name)
	if err != nil {
		if errors.Is(err, ErrLookupNotFound) {
			return nil, name, fmt.Sprintf("據點「%s」未於據點管理中找到，已建立個案並保留原始名稱待人工關聯", name), nil
		}
		return nil, "", "", err
	}
	if site == nil {
		return nil, name, fmt.Sprintf("據點「%s」未於據點管理中找到，已建立個案並保留原始名稱待人工關聯", name), nil
	}
	return &site.ID, "", "", nil
}

// caregiverTypeOf 把工作表的「個管or照專」文字轉為主檔類型；無法對應時回傳空字串。
// 「專護」是照專的舊稱，沿用 caregiver 匯入既有的相容處理。
func caregiverTypeOf(role string) string {
	switch strings.TrimSpace(role) {
	case "個管":
		return "case_manager"
	case "照專", "專護":
		return "specialist"
	default:
		return ""
	}
}

// caregiverRoleOf 把主檔類型轉回工作表的中文角色；未設定類型時留白。
func caregiverRoleOf(caregiverType string) string {
	switch caregiverType {
	case "case_manager":
		return "個管"
	case "specialist":
		return "照專"
	default:
		return ""
	}
}

// resolveCaregiver 以姓名比對照護人員主檔：同名唯一即採用，角色一律以主檔為準；
// 只有同名多筆時才用工作表的「個管or照專」消歧，消歧後仍不唯一就視為比對不到。
// 比對不到時回傳空 ID 與原始角色文字，個案照常建立並由 caregiver_pending 落入待維護。
func (s *ImportService) resolveCaregiver(ctx context.Context, name, role string) (id *uuid.UUID, resolvedRole string, warning string, err error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, "", "", nil
	}
	if s.caregiverRepo == nil {
		return nil, role, "", nil
	}
	matches, err := s.caregiverRepo.FindByName(ctx, name)
	if err != nil {
		if errors.Is(err, ErrLookupNotFound) {
			matches = nil
		} else {
			return nil, "", "", err
		}
	}
	switch len(matches) {
	case 0:
		return nil, role, fmt.Sprintf("照護人員「%s」未於照護人員管理中找到，已建立個案並列入待維護", name), nil
	case 1:
		return &matches[0].ID, caregiverRoleOf(matches[0].Type), "", nil
	}

	wanted := caregiverTypeOf(role)
	if wanted == "" {
		return nil, role, fmt.Sprintf("照護人員「%s」有多筆同名資料，需以「個管or照專」指定，已建立個案並列入待維護", name), nil
	}
	var narrowed []CaregiverRef
	for _, m := range matches {
		if m.Type == wanted {
			narrowed = append(narrowed, m)
		}
	}
	if len(narrowed) != 1 {
		return nil, role, fmt.Sprintf("照護人員「%s」以「%s」仍無法唯一對應，已建立個案並列入待維護", name, strings.TrimSpace(role)), nil
	}
	return &narrowed[0].ID, caregiverRoleOf(narrowed[0].Type), "", nil
}

func skippedRow(row CaseImportRowResult) CaseImportSkippedRow {
	reasons := strings.Split(row.ErrorMessage, "；")
	return CaseImportSkippedRow{RowID: row.RowID, RowIndex: row.RowIndex, CaseName: row.Name, Reasons: reasons, RawValues: row.RawValues}
}

func caseImportFailureRow(row CaseImportRowResult, reason string) CaseImportSkippedRow {
	return CaseImportSkippedRow{
		RowID:     row.RowID,
		RowIndex:  row.RowIndex,
		CaseName:  row.Name,
		Reasons:   []string{reason},
		RawValues: row.RawValues,
	}
}

func importRowKey(rowID string, rowIndex int) string {
	if rowID != "" {
		return rowID
	}
	return fmt.Sprintf("legacy:%d", rowIndex)
}
