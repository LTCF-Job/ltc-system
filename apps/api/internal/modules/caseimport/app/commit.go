package app

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

func buildImportLegs(row CaseImportRowResult) []NewScheduleLeg {
	switch row.TripPattern {
	case 1:
		return []NewScheduleLeg{
			{LegSeq: 1, Direction: "outbound", DepartTime: row.OutboundTime},
		}
	case 4:
		return []NewScheduleLeg{
			{LegSeq: 1, Direction: "outbound", DepartTime: "08:30"},
			{LegSeq: 2, Direction: "inbound", DepartTime: "11:30"},
			{LegSeq: 3, Direction: "outbound", DepartTime: "13:30"},
			{LegSeq: 4, Direction: "inbound", DepartTime: "16:30"},
		}
	default:
		return []NewScheduleLeg{
			{LegSeq: 1, Direction: "outbound", DepartTime: row.OutboundTime},
			{LegSeq: 2, Direction: "inbound", DepartTime: row.InboundTime},
		}
	}
}

// CommitCases 將通過檢核的個案資料正式寫入資料庫。
//
// 交易語意：每一列（個案主檔＋接送車輛偏好＋排班設定＋稽核紀錄）在單一事務中
// 全有或全無提交；列與列之間彼此獨立——某列寫入失敗只回滾該列自身的變更並
// 記為略過列（附原因與原始欄位供人工補正），不影響已提交的前列，也不會中止
// 整批匯入其餘待處理的列。
//
// 選擇「逐列獨立事務」而非「整批單一事務＋逐列 savepoint」：匯入列彼此沒有
// 跨列一致性需求，逐列事務讓已成功的列立即落地可見、避免整批因單一長交易
// 持有鎖與可見性延遲，且與 CaseRepository 既有逐方法自管事務的慣例
// （CreateSchedule、HolidayRepository.BatchUpsert）一致。
func (s *ImportService) CommitCases(ctx context.Context, preview *CaseImportPreviewResult, actor Actor) (*CaseImportCommitResult, error) {
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

		claimStart, err := time.Parse("2006-01-02", row.ClaimStartDate)
		if err != nil {
			claimStart = time.Now()
		}

		// 身分證字號為 cases 資料表的 NOT NULL UNIQUE 欄位，且需通過檢查碼驗證，
		// 目前 schema 無法表示「尚無身分證的草稿個案」；與其合成一組可能矇混通過
		// 檢查碼、進而寫入加密儲存與唯一索引的假身分證，誠實地將此列標記為
		// 待人工補件，留待 schema 支援後再處理。
		if row.NationalID == "" {
			item := CaseImportSkippedRow{RowIndex: row.RowIndex, CaseName: row.Name, Reasons: []string{"身分證字號空白，需人工補件後再匯入"}, RawValues: row.RawValues}
			result.SkippedRows = append(result.SkippedRows, item)
			if s.cases != nil {
				s.cases.RecordSkipped(ctx, item, actor)
			}
			continue
		}
		// 住家地址是接送路線的依據，合成一個「待補」字串會讓錯誤資料看起來像
		// 已完成匯入；缺址的列一律標記為待人工補件。
		if strings.TrimSpace(row.HomeAddress) == "" {
			item := CaseImportSkippedRow{RowIndex: row.RowIndex, CaseName: row.Name, Reasons: []string{"住家地址空白，需人工補件後再匯入"}, RawValues: row.RawValues}
			result.SkippedRows = append(result.SkippedRows, item)
			if s.cases != nil {
				s.cases.RecordSkipped(ctx, item, actor)
			}
			continue
		}

		status := "active"
		if row.IsDraft {
			status = "suspended"
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
			HomeAddress:       row.HomeAddress,
			Region:            row.Region,
			ClaimStartDate:    claimStart,
			ServiceCategory:   row.ServiceCategory,
			ServiceUsageType:  row.ServiceUsageType,
			Status:            status,
		}
		if row.BirthDate != "" {
			if birthDate, err := time.Parse("2006-01-02", row.BirthDate); err == nil {
				caseReq.BirthDate = &birthDate
			}
		}

		// 據點／車輛主檔為既有唯讀參照資料，於事務外先查詢即可；查無對應資料時整列直接略過，不需開啟事務。
		var transportSiteID, outboundVehicleID, inboundVehicleID uuid.UUID
		if row.IsProfileWorkbook && s.siteRepo != nil && s.vehicleRepo != nil {
			site, siteErr := s.siteRepo.GetByName(ctx, row.SiteName)
			outbound, outboundErr := s.vehicleRepo.GetByDisplayName(ctx, row.OutboundVehicle)
			inbound, inboundErr := s.vehicleRepo.GetByDisplayName(ctx, row.InboundVehicle)
			if siteErr != nil || outboundErr != nil || inboundErr != nil {
				item := CaseImportSkippedRow{RowIndex: row.RowIndex, CaseName: row.Name, Reasons: []string{"據點或去／回程車輛未建檔"}, RawValues: row.RawValues}
				result.SkippedRows = append(result.SkippedRows, item)
				continue
			}
			transportSiteID, outboundVehicleID, inboundVehicleID = site.ID, outbound.ID, inbound.ID
		}

		var scheduleSiteID uuid.UUID
		if s.siteRepo != nil {
			siteList, _ := s.siteRepo.List(ctx, row.Region, 1, 100)
			for _, st := range siteList {
				if st.Name == row.SiteName || strings.Contains(st.Name, row.SiteName) {
					scheduleSiteID = st.ID
					break
				}
			}
			if scheduleSiteID == uuid.Nil && len(siteList) > 0 {
				scheduleSiteID = siteList[0].ID
			}
		}

		txErr := s.txRunner.WithTx(ctx, func(txCtx context.Context) error {
			caseID, err := s.cases.CreateCase(txCtx, caseReq, actor)
			if err != nil {
				return fmt.Errorf("個案建立失敗：%w", err)
			}

			if transportSiteID != uuid.Nil {
				if err := s.prefRepo.UpsertTransportPreference(txCtx, caseID, transportSiteID, outboundVehicleID, inboundVehicleID); err != nil {
					return fmt.Errorf("儲存接送車輛偏好失敗：%w", err)
				}
			}

			if scheduleSiteID != uuid.Nil && len(row.Weekdays) > 0 && !row.IsProfileWorkbook {
				schedNote := row.Note
				if err := s.cases.CreateSchedule(txCtx, NewSchedule{
					CaseID:             caseID,
					SiteID:             scheduleSiteID,
					EffectiveFrom:      claimStart,
					Weekdays:           row.Weekdays,
					TripPattern:        row.TripPattern,
					UnitPrice:          row.UnitPrice,
					DistanceKM:         row.DistanceKM,
					ServiceDurationMin: row.ServiceDurationMin,
					ServiceCode:        "BD03",
					Note:               &schedNote,
					Legs:               buildImportLegs(row),
				}, actor); err != nil {
					return fmt.Errorf("建立排班設定失敗：%w", err)
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

func skippedRow(row CaseImportRowResult) CaseImportSkippedRow {
	reasons := strings.Split(row.ErrorMessage, "；")
	return CaseImportSkippedRow{RowIndex: row.RowIndex, CaseName: row.Name, Reasons: reasons, RawValues: row.RawValues}
}
