package app

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"time"

	"github.com/google/uuid"
)

// CommitDriverReport 正式寫入匯報表：先把使用者確認的欄位對應存回 form_columns，
// 再逐列交給 ride 模組展開為搭乘來源與搭乘紀錄。
//
// 採「逐列略過」而非全有全無：單列日期無法解析時只略過該列並記錄原因，其餘日期
// 照常寫入，避免一整個月的匯報因為一列打錯而全部匯不進來。
func (s *DriverReportService) CommitDriverReport(
	ctx context.Context,
	formID uuid.UUID,
	r io.Reader,
	decisions []ColumnDecision,
	actor Actor,
) (*CommitResult, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("讀取上傳檔案失敗: %w", err)
	}

	preview, err := s.ParseDriverReport(ctx, formID, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	form, err := s.repo.GetForm(ctx, formID)
	if err != nil {
		return nil, err
	}
	if form == nil {
		return nil, ErrFormNotFound
	}

	if err := s.persistColumnDecisions(ctx, formID, preview, decisions); err != nil {
		return nil, err
	}

	mapped, err := s.repo.ListColumnsWithMapping(ctx, formID.String(), "mapped")
	if err != nil {
		return nil, err
	}

	result := &CommitResult{
		MappedColumns: len(mapped),
		SkippedRows:   []SkippedRow{},
		Warnings:      []ImportWarningItem{},
	}

	if len(mapped) == 0 {
		return nil, fmt.Errorf("尚未有任何欄位對應到個案，請先於預覽畫面完成對應")
	}

	tables, _, err := s.excel.ReadTables(data)
	if err != nil {
		return nil, err
	}
	rows := tables[0]

	submittedAt := time.Now().UTC()
	for _, row := range preview.PreviewRows {
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

		answers := map[string]string{}
		for _, col := range mapped {
			answers[col.ColumnHeader] = cellAt(rows[row.RowIndex-1], col.ColumnIndex-1)
		}

		submission := Submission{
			ServiceDate: serviceDate,
			SubmittedAt: submittedAt,
			DriverRaw:   row.DriverRaw,
			DriverID:    parseOptionalUUID(row.DriverID),
			Remark:      row.Remark,
			Answers:     answers,
		}

		written, err := s.rideIngestor.IngestSubmission(ctx, formID, form.VehicleID, submission)
		if err != nil {
			result.SkippedRows = append(result.SkippedRows, SkippedRow{
				RowIndex:   row.RowIndex,
				ReportDate: row.ReportDate,
				Reasons:    []string{"寫入搭乘紀錄失敗：" + err.Error()},
			})
			continue
		}

		result.ImportedRows++
		result.RideRecordRows += written
		if row.WarningMessage != "" {
			result.Warnings = append(result.Warnings, ImportWarningItem{RowIndex: row.RowIndex, Message: row.WarningMessage})
		}
	}

	if err := s.repo.MarkImported(ctx, formID, submittedAt); err != nil {
		return nil, err
	}

	s.writeImportAudit(ctx, formID, result, actor)

	return result, nil
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

// persistColumnDecisions 先把檔案中的所有個案欄位登錄成 form_columns，再套用使用者
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
