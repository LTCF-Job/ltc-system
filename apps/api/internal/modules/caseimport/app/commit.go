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

		// 據點／去回程車輛各自獨立比對：比對到則寫入 ID，比對不到但有填名稱則保留
		// 原始名稱待人工關聯，兩種情況都不影響個案主檔本身的建立。
		siteID, siteNameRaw, siteWarning, err := s.resolveSite(ctx, row.SiteName)
		if err != nil {
			slog.Error("case import site lookup failed", "row_index", row.RowIndex, "error", err)
			recordFailed(caseImportFailureRow(row, "據點查詢失敗，請稍後重試"))
			continue
		}
		outboundID, outboundNameRaw, outboundWarning, err := s.resolveVehicle(ctx, row.OutboundVehicle, "接送車輛(去)")
		if err != nil {
			slog.Error("case import outbound vehicle lookup failed", "row_index", row.RowIndex, "error", err)
			recordFailed(caseImportFailureRow(row, "去程車輛查詢失敗，請稍後重試"))
			continue
		}
		inboundID, inboundNameRaw, inboundWarning, err := s.resolveVehicle(ctx, row.InboundVehicle, "接送車輛(回)")
		if err != nil {
			slog.Error("case import inbound vehicle lookup failed", "row_index", row.RowIndex, "error", err)
			recordFailed(caseImportFailureRow(row, "回程車輛查詢失敗，請稍後重試"))
			continue
		}
		for _, w := range []string{siteWarning, outboundWarning, inboundWarning} {
			if w != "" {
				result.Warnings = append(result.Warnings, CaseImportWarningItem{RowIndex: row.RowIndex, CaseName: row.Name, Message: w})
			}
		}

		if row.IsDuplicate {
			if s.duplicateStager == nil {
				recordFailed(caseImportFailureRow(row, "重複個案待裁決功能尚未設定"))
				continue
			}
			_, alreadyStaged, err := s.duplicateStager.StageDuplicateRow(ctx, preview.FileHash, importRowKey(row.RowID, row.RowIndex), StageDuplicateCandidate{
				RowIndex:               row.RowIndex,
				SheetName:              row.SheetName,
				Name:                   row.Name,
				NationalID:             row.NationalID,
				HouseholdType:          stringPointer(row.HouseholdType),
				Gender:                 stringPointer(row.Gender),
				BirthDate:              parseBirthDate(row.BirthDate),
				BirthDateRaw:           stringPointer(row.BirthDateRaw),
				CareContactRole:        stringPointer(row.CareContactRole),
				CareContactName:        stringPointer(row.CareContactName),
				RegisteredAddress:      stringPointer(row.RegisteredAddress),
				HomeAddress:            stringPointer(row.HomeAddress),
				Region:                 stringPointer(row.Region),
				ServiceCategory:        row.ServiceCategory,
				ServiceUsageType:       row.ServiceUsageType,
				Remarks:                stringPointer(row.Remarks),
				SiteID:                 siteID,
				SiteNameRaw:            siteNameRaw,
				OutboundVehicleID:      outboundID,
				OutboundVehicleNameRaw: outboundNameRaw,
				InboundVehicleID:       inboundID,
				InboundVehicleNameRaw:  inboundNameRaw,
				DuplicateCaseID:        *row.DuplicateCaseID,
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
			CareContactRole:        stringPointer(row.CareContactRole),
			CareContactName:        stringPointer(row.CareContactName),
			RegisteredAddress:      stringPointer(row.RegisteredAddress),
			HomeAddress:            stringPointer(row.HomeAddress),
			Region:                 stringPointer(row.Region),
			ServiceCategory:        row.ServiceCategory,
			ServiceUsageType:       row.ServiceUsageType,
			Status:                 "active",
			Remarks:                stringPointer(row.Remarks),
			SiteID:                 siteID,
			SiteNameRaw:            siteNameRaw,
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
			caseID, err := s.cases.CreateCase(txCtx, caseReq, actor)
			if err != nil {
				return fmt.Errorf("個案建立失敗：%w", err)
			}

			if outboundID != nil || inboundID != nil || outboundNameRaw != "" || inboundNameRaw != "" {
				if s.prefRepo == nil {
					return errors.New("transport preference writer not configured")
				}
				if err := s.prefRepo.UpsertTransportPreference(txCtx, caseID, outboundID, inboundID, outboundNameRaw, inboundNameRaw); err != nil {
					return fmt.Errorf("儲存接送車輛偏好失敗：%w", err)
				}
			}

			return nil
		})

		if txErr != nil {
			if errors.Is(txErr, ErrCaseImportAlreadyCommitted) {
				result.AlreadyImportedCount++
				item := caseImportFailureRow(row, "此檔案的此列已完成匯入，略過重試")
				recordSkipped(item)
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

// resolveVehicle 依顯示名稱比對既有車輛；查無資料時回傳空 ID 與原始名稱，並附上待人工關聯的提示。
func (s *ImportService) resolveVehicle(ctx context.Context, name, fieldLabel string) (id *uuid.UUID, nameRaw string, warning string, err error) {
	if name == "" || s.vehicleRepo == nil {
		return nil, "", "", nil
	}
	vehicle, err := s.vehicleRepo.GetByDisplayName(ctx, name)
	if err != nil {
		if errors.Is(err, ErrLookupNotFound) {
			return nil, name, fmt.Sprintf("%s『%s』未於車輛管理中找到，已建立個案並保留原始名稱待人工關聯", fieldLabel, name), nil
		}
		return nil, "", "", err
	}
	if vehicle == nil {
		return nil, name, fmt.Sprintf("%s『%s』未於車輛管理中找到，已建立個案並保留原始名稱待人工關聯", fieldLabel, name), nil
	}
	return &vehicle.ID, "", "", nil
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
