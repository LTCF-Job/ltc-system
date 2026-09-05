package app

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"time"

	"github.com/google/uuid"
)

// CommitDriverReport 正式寫入匯報表：先把使用者確認的欄位對應存回 form_columns，
// 清掉這份匯報表在本次涵蓋日期的既有匯入資料，再逐列交給 ride 模組重新展開。
//
// 匯入語意是覆蓋而非疊加，重匯同一個月的結果與只匯一次相同。清除與重寫落在同一個
// 交易內；任何資料庫層級的失敗都整份回滾，避免留下只刪不寫的空月份。
//
// 未宣告月份時，解析層級的失敗仍逐列略過；宣告整月覆蓋時，日期無法解析屬於阻斷性錯誤，
// 整份不清除也不寫入。
//
// yearMonth 為選填的宣告匯入月份（YYYY-MM）。有宣告時清除整個月，未宣告時只清除
// 檔案實際涵蓋的日期；檔案沒有任何有效列時不執行清除，避免傳錯空檔清空整月資料。
func (s *DriverReportService) CommitDriverReport(
	ctx context.Context,
	formID uuid.UUID,
	r io.Reader,
	decisions []ColumnDecision,
	yearMonth string,
	actor Actor,
) (*CommitResult, error) {
	if s.txRunner == nil {
		return nil, errors.New("driver report service: transaction runner not configured")
	}

	monthStart, monthDeclared, err := parseYearMonth(yearMonth)
	if err != nil {
		return nil, err
	}

	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("讀取上傳檔案失敗: %w", err)
	}

	preview, err := s.ParseDriverReport(ctx, formID, bytes.NewReader(data), yearMonth)
	if err != nil {
		return nil, err
	}
	if monthDeclared && !preview.CanCommit {
		return nil, ErrImportHasBlockingErrors
	}

	form, err := s.repo.GetForm(ctx, formID)
	if err != nil {
		return nil, err
	}
	if form == nil {
		return nil, ErrFormNotFound
	}

	tables, _, err := s.excel.ReadTables(data)
	if err != nil {
		return nil, err
	}
	rows := tables[0]

	result := &CommitResult{
		SkippedRows: []SkippedRow{},
		Warnings:    []ImportWarningItem{},
	}
	importable := collectImportableRows(preview.PreviewRows, result)

	txErr := s.txRunner.WithTx(ctx, func(txCtx context.Context) error {
		if err := s.persistColumnDecisions(txCtx, formID, preview, decisions); err != nil {
			return err
		}

		allMapped, err := s.repo.ListColumnsWithMapping(txCtx, formID.String(), "mapped")
		if err != nil {
			return err
		}
		currentHeaders := make(map[string]struct{}, len(preview.Columns))
		for _, column := range preview.Columns {
			currentHeaders[column.ColumnHeader] = struct{}{}
		}
		mappedCount := 0
		for _, column := range allMapped {
			if _, found := currentHeaders[column.ColumnHeader]; found {
				mappedCount++
			}
		}
		result.MappedColumns = mappedCount

		if err := s.clearPreviousImport(txCtx, formID, importable, monthStart, monthDeclared); err != nil {
			return err
		}

		submittedAt := time.Now().UTC()
		for _, row := range importable {
			// 保留這一列所有欄位的原始值，含尚未對應個案的欄位：日後在待維護頁面完成
			// 綁定時，直接用這裡存的 form_submissions 回填搭乘紀錄，不必重新上傳檔案。
			answers := map[string]string{}
			for _, col := range preview.Columns {
				answers[col.ColumnHeader] = cellAt(rows[row.preview.RowIndex-1], col.ColumnIndex-1)
			}

			driverID := parseOptionalUUID(row.preview.DriverID)
			written, err := s.rideIngestor.IngestSubmission(txCtx, formID, form.VehicleID, Submission{
				ServiceDate: row.serviceDate,
				SubmittedAt: submittedAt,
				DriverRaw:   row.preview.DriverRaw,
				DriverID:    driverID,
				Remark:      row.preview.Remark,
				Answers:     answers,
			})
			if err != nil {
				return fmt.Errorf("第 %d 列寫入搭乘紀錄失敗：%w", row.preview.RowIndex, err)
			}

			// 比對到司機時順便同步當天出勤月曆；比對不到的維持既有「駕駛人待維護」流程，
			// 不在這裡處理。
			if driverID != nil {
				if err := s.attendanceRegistrar.SyncFromImport(txCtx, *driverID, row.serviceDate); err != nil {
					return fmt.Errorf("第 %d 列同步司機出勤失敗：%w", row.preview.RowIndex, err)
				}
			}

			result.ImportedRows++
			result.RideRecordRows += written
			if row.preview.WarningMessage != "" {
				result.Warnings = append(result.Warnings, ImportWarningItem{RowIndex: row.preview.RowIndex, Message: row.preview.WarningMessage})
			}
		}

		if result.ImportedRows == 0 {
			// 全部列都被跳過（月份不符或格式錯誤）時代表沒有任何一列真的匯入，
			// 不更新最後匯入時間，避免使用者誤以為這次上傳已經成功。
			return nil
		}
		return s.repo.MarkImported(txCtx, formID, submittedAt)
	})
	if txErr != nil {
		return nil, txErr
	}

	s.writeImportAudit(ctx, formID, result, actor)

	return result, nil
}

// importableRow 是一列已確定可寫入的匯報資料，serviceDate 為其解析後的服務日期。
type importableRow struct {
	preview     RowPreview
	serviceDate time.Time
}

// collectImportableRows 挑出可寫入的列，其餘連同原因記入 result.SkippedRows。
func collectImportableRows(previewRows []RowPreview, result *CommitResult) []importableRow {
	out := make([]importableRow, 0, len(previewRows))
	for _, row := range previewRows {
		if row.ErrorMessage != "" {
			result.SkippedRows = append(result.SkippedRows, SkippedRow{
				RowIndex:   row.RowIndex,
				ReportDate: row.ReportDate,
				Reasons:    []string{row.ErrorMessage},
			})
			continue
		}

		serviceDate, err := time.Parse("2006-01-02", row.ServiceDate)
		if err != nil {
			result.SkippedRows = append(result.SkippedRows, SkippedRow{
				RowIndex:   row.RowIndex,
				ReportDate: row.ReportDate,
				Reasons:    []string{"服務日期無法轉換"},
			})
			continue
		}

		out = append(out, importableRow{preview: row, serviceDate: serviceDate})
	}
	return out
}

// clearPreviousImport 清掉本次要覆蓋的既有匯入資料。
//
// 沒有任何可寫入的列時不清除：那通常是傳錯檔案，清空整月的代價遠高於少覆蓋一次。
func (s *DriverReportService) clearPreviousImport(
	ctx context.Context,
	formID uuid.UUID,
	importable []importableRow,
	monthStart time.Time,
	monthDeclared bool,
) error {
	if len(importable) == 0 {
		return nil
	}

	var dates []time.Time
	if monthDeclared {
		dates = daysInMonth(monthStart)
	} else {
		seen := map[string]bool{}
		for _, row := range importable {
			key := row.preview.ServiceDate
			if seen[key] {
				continue
			}
			seen[key] = true
			dates = append(dates, row.serviceDate)
		}
	}

	if _, err := s.rideIngestor.ClearImportedDates(ctx, formID, dates); err != nil {
		return err
	}
	return nil
}

// writeImportAudit 留下匯入留痕。稽核寫入失敗不推翻已完成的匯入，只記錄於伺服器日誌。
func (s *DriverReportService) writeImportAudit(ctx context.Context, formID uuid.UUID, result *CommitResult, actor Actor) {
	if s.auditRepo == nil {
		return
	}
	entityID := formID.String()
	if err := s.auditRepo.Write(ctx, AuditEntry{
		ActorID:    &actor.ActorID,
		ActorRole:  &actor.ActorRole,
		Action:     "import",
		EntityType: "driver_report_forms",
		EntityID:   &entityID,
		AfterData:  result,
		IPAddress:  &actor.IPAddress,
		UserAgent:  &actor.UserAgent,
	}); err != nil {
		slog.Warn("Failed to write driver report import audit",
			slog.String("formId", entityID),
			slog.String("error", err.Error()))
	}
}

// persistColumnDecisions 先把檔案中的所有個案欄位登記成 form_columns，再套用使用者
// 在預覽畫面所做的對應決定；沒有決定的欄位維持既有狀態（首次出現即 pending）。
func (s *DriverReportService) persistColumnDecisions(
	ctx context.Context,
	formID uuid.UUID,
	preview *PreviewResult,
	decisions []ColumnDecision,
) error {
	drafts := make([]ColumnDraft, 0, len(preview.Columns))
	for _, c := range preview.Columns {
		drafts = append(drafts, ColumnDraft{
			ColumnIndex:     c.ColumnIndex,
			ColumnHeader:    c.ColumnHeader,
			CleanedName:     c.CleanedName,
			Kind:            "ride",
			SuggestedCaseID: c.SuggestedCaseID,
			SuggestionScore: c.SuggestionScore,
		})
	}
	if err := s.repo.UpsertColumns(ctx, formID, drafts); err != nil {
		return err
	}

	for _, d := range decisions {
		status := d.MappingStatus
		if status == "" {
			status = "pending"
		}
		if status == "mapped" && (d.CaseID == nil || d.LegSeq == nil) {
			return fmt.Errorf("欄位「%s」標記為已對應，但缺少個案或趟次", d.ColumnHeader)
		}
		if err := s.repo.UpdateColumnMappingByHeader(ctx, formID, d.ColumnHeader, status, d.CaseID, d.LegSeq); err != nil {
			return err
		}
	}
	return nil
}

func parseOptionalUUID(raw string) *uuid.UUID {
	if raw == "" {
		return nil
	}
	parsed, err := uuid.Parse(raw)
	if err != nil {
		return nil
	}
	return &parsed
}
