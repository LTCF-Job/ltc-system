package app

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"time"

	"github.com/google/uuid"
)

// CommitDriverReport 正式寫入匯報表：確認欄位對應後逐列比對既有資料，沒問題的直接
// 寫入、值不同的進待維護等待使用者選擇，整份寫入落在同一交易內、失敗即回滾（每次
// 上傳是獨立事件，不整段覆蓋既有資料，見 docs/decisions/driver-report-import-overwrite.md）。
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

	// 只驗格式；宣告月份與否不再決定整份是否寫入，錯誤列一律由 collectImportableRows 逐列略過。
	if _, _, err := parseYearMonth(yearMonth); err != nil {
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
	if len(tables) == 0 || len(tables[0]) == 0 {
		return nil, errors.New("匯入檔案沒有可解析的工作表")
	}
	rows := tables[0]

	result := &CommitResult{
		SkippedRows: []SkippedRow{},
		Warnings:    []ImportWarningItem{},
		Status:      "pending",
	}
	importable := collectImportableRows(preview.PreviewRows, result)

	txErr := s.txRunner.WithTx(ctx, func(txCtx context.Context) error {
		if locker, ok := s.repo.(DriverReportImportLocker); ok {
			if err := locker.LockDriverReportImport(txCtx, formID, yearMonth); err != nil {
				return fmt.Errorf("鎖定匯入月份失敗：%w", err)
			}
		}
		backfillTargets, err := s.persistColumnDecisions(txCtx, formID, preview, decisions)
		if err != nil {
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

		// 不再先清除本次涵蓋日期的既有資料：每次上傳是獨立事件，逐列比對交由
		// RideIngestor.IngestSubmission 內部處理——沒問題的直接寫入，值不同的進待維護，
		// 這台車其他未出現在本次檔案的資料完全不受影響（見
		// docs/decisions/driver-report-import-overwrite.md）。
		submittedAt := s.now()
		for _, row := range importable {
			// 保留這一列所有欄位的原始值，含尚未對應個案的欄位：日後在待維護頁面完成
			// 綁定時，直接用這裡存的 form_submissions 回填搭乘紀錄，不必重新上傳檔案。
			answers := map[string]string{}
			for _, col := range preview.Columns {
				answers[col.ColumnHeader] = cellAt(rows[row.preview.RowIndex-1], col.ColumnIndex-1)
			}

			driverID := parseOptionalUUID(row.preview.DriverID)
			outcome, err := s.rideIngestor.IngestSubmission(txCtx, formID, form.VehicleID, Submission{
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
			result.RideRecordRows += outcome.Written
			result.ReaffirmedRows += outcome.Reaffirmed
			result.PendingConflictRows += outcome.Staged
			if row.preview.WarningMessage != "" {
				result.Warnings = append(result.Warnings, ImportWarningItem{RowIndex: row.preview.RowIndex, Message: row.preview.WarningMessage})
			}
		}

		// 必須跑在上面的逐列寫入之後：form_submissions 是「一車一天一筆」原地更新，
		// 先補寫會讀到這幾天更新前的舊答案，再被本次的新值比出一筆並不存在的衝突。
		// 檔案涵蓋的日期全部排除（不只本次宣告的月份）：跨月檔案是逐月各送一次 commit，
		// 尚未輪到的那些月份此刻還是上一次上傳的舊值，補進去就會被下一輪比出假衝突。
		fileDates := collectFileServiceDates(preview.PreviewRows)
		for _, target := range backfillTargets {
			written, err := s.rideIngestor.BackfillColumn(txCtx, formID, form.VehicleID, target.columnHeader, target.columnIndex, target.caseID, target.legSeq, fileDates)
			if err != nil {
				return fmt.Errorf("欄位「%s」補寫先前搭乘紀錄失敗：%w", target.columnHeader, err)
			}
			result.BackfilledRows += written
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
	result.Status = "succeeded"

	// 稽核留下檔案雜湊，事後才追得出某筆搭乘來源出自哪一次上傳；這不是重複判斷的依據
	fileHash := fmt.Sprintf("sha256:%x", sha256.Sum256(data))
	s.writeImportAudit(ctx, formID, yearMonth, fileHash, result, actor)

	return result, nil
}

// importableRow 是一列已確定可寫入的匯報資料，serviceDate 為其解析後的服務日期。
type importableRow struct {
	preview     RowPreview
	serviceDate time.Time
}

// collectFileServiceDates 取出這份檔案解析得出的所有服務日期，含落在宣告月份以外、
// 這一輪不寫入的列——那些日期會在各自月份的 commit 寫入本次的值，回填不該先用舊值蓋過去。
func collectFileServiceDates(previewRows []RowPreview) []time.Time {
	out := make([]time.Time, 0, len(previewRows))
	for _, row := range previewRows {
		if row.ServiceDate == "" {
			continue
		}
		if date, err := time.Parse("2006-01-02", row.ServiceDate); err == nil {
			out = append(out, date)
		}
	}
	return out
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

// writeImportAudit 留下匯入留痕。稽核寫入失敗不推翻已完成的匯入，只記錄於伺服器日誌。
func (s *DriverReportService) writeImportAudit(ctx context.Context, formID uuid.UUID, yearMonth, fileHash string, result *CommitResult, actor Actor) {
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
		AfterData:  result.AuditSnapshot(formID, yearMonth, fileHash),
		IPAddress:  &actor.IPAddress,
		UserAgent:  &actor.UserAgent,
	}); err != nil {
		slog.Warn("Failed to write driver report import audit",
			slog.String("formId", entityID),
			slog.String("error", err.Error()))
	}
}

// backfillTarget 是本次匯入剛從待維護變成已對應的欄位，其先前月份的既有回報需要補寫。
type backfillTarget struct {
	columnHeader string
	columnIndex  int
	caseID       uuid.UUID
	legSeq       int16
}

// persistColumnDecisions 先把檔案中的所有個案欄位登記成 form_columns，再套用使用者
// 在預覽畫面所做的對應決定；沒有決定的欄位維持既有狀態（首次出現即 pending）。
// 回傳這次剛完成對應、需要補寫既有回報的欄位。
func (s *DriverReportService) persistColumnDecisions(
	ctx context.Context,
	formID uuid.UUID,
	preview *PreviewResult,
	decisions []ColumnDecision,
) ([]backfillTarget, error) {
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
		return nil, err
	}

	var targets []backfillTarget
	for _, d := range decisions {
		status := d.MappingStatus
		if status == "" {
			status = "pending"
		}
		if status == "mapped" && (d.CaseID == nil || d.LegSeq == nil) {
			return nil, fmt.Errorf("欄位「%s」標記為已對應，但缺少個案或趟次", d.ColumnHeader)
		}
		columnIndex, previousStatus, err := s.repo.UpdateColumnMappingByHeader(ctx, formID, d.ColumnHeader, status, d.CaseID, d.LegSeq)
		if err != nil {
			return nil, err
		}
		// 只有真的從非 mapped 變成 mapped 才補寫，比照待維護頁手動綁定的同一套條件；
		// previousStatus 為空代表這個表頭不在這份表單，沒有既有回報可補
		if status != "mapped" || previousStatus == "mapped" || previousStatus == "" {
			continue
		}
		caseID, err := uuid.Parse(*d.CaseID)
		if err != nil {
			return nil, fmt.Errorf("欄位「%s」的個案編號格式錯誤：%w", d.ColumnHeader, err)
		}
		targets = append(targets, backfillTarget{
			columnHeader: d.ColumnHeader,
			columnIndex:  columnIndex,
			caseID:       caseID,
			legSeq:       *d.LegSeq,
		})
	}
	return targets, nil
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
