package app

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"ltc-system/apps/api/internal/domain/merge"
	"ltc-system/apps/api/internal/domain/namenorm"
	"ltc-system/apps/api/internal/domain/rocdate"
)

// 司機接送匯報表的固定欄位：第一欄民國日期、第二欄駕駛人、最後一欄備註，
// 中間全部是「個案＋趟次」欄位。
const (
	headerReportDate = "民國日期"
	headerDriver     = "駕駛人"
	headerRemark     = "備註"
)

// minCaseColumns 是一份可用匯報表至少要有的欄數（日期＋駕駛人＋一個個案欄＋備註）。
const minReportColumns = 4

// ErrFormNotFound 代表指定的匯報表不存在。
var ErrFormNotFound = errors.New("driver report form not found")

// ErrInvalidYearMonth 代表宣告的匯入月份格式不是 YYYY-MM。
var ErrInvalidYearMonth = errors.New("匯入月份格式錯誤，請使用 YYYY-MM")

// ParseDriverReport 解析上傳的匯報表 .xlsx，產生欄位對應與每日匯報列的預覽。
//
// 解析階段不寫入任何資料：未對應的欄位會附上推薦個案與趟次，交由使用者在預覽畫面
// 就地確認後，才於 CommitDriverReport 一併寫回 form_columns 與搭乘紀錄。
//
// yearMonth 為選填的宣告匯入月份（YYYY-MM）。有宣告時，檔案內任一有效列落在該月之外
// 就整份拒絕；匯入是以月覆蓋，放行等於讓傳錯檔案清空另一個月的資料。
func (s *DriverReportService) ParseDriverReport(ctx context.Context, formID uuid.UUID, r io.Reader, yearMonth string) (*PreviewResult, error) {
	monthStart, monthDeclared, err := parseYearMonth(yearMonth)
	if err != nil {
		return nil, err
	}

	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("讀取上傳檔案失敗: %w", err)
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
	headerRowIdx, header, err := findReportHeader(rows)
	if err != nil {
		return nil, err
	}

	remarkIdx := len(header) - 1
	caseHeaders := header[2:remarkIdx]

	columns, err := s.buildColumnPreviews(ctx, formID, caseHeaders)
	if err != nil {
		return nil, err
	}

	result := &PreviewResult{
		FormID:      form.ID.String(),
		VehicleID:   form.VehicleID.String(),
		VehicleName: form.VehicleDisplayName,
		Columns:     columns,
		PreviewRows: []RowPreview{},
		Errors:      []ImportErrorItem{},
		Warnings:    []ImportWarningItem{},
	}

	for i := headerRowIdx + 1; i < len(rows); i++ {
		rowNum := i + 1
		dateRaw := cellAt(rows[i], 0)
		if strings.TrimSpace(dateRaw) == "" {
			// 空白日期代表總計列或表尾空列，直接略過而不計入統計。
			continue
		}

		row := RowPreview{RowIndex: rowNum, ReportDate: dateRaw}
		result.TotalRows++

		serviceDate, err := parseReportDate(dateRaw)
		if err != nil {
			row.ErrorMessage = fmt.Sprintf("日期格式無法解析（%s），請填寫民國日期如 1150302", dateRaw)
			result.Errors = append(result.Errors, ImportErrorItem{RowIndex: rowNum, Field: headerReportDate, Message: row.ErrorMessage})
			result.ErrorRows++
			result.PreviewRows = append(result.PreviewRows, row)
			continue
		}
		row.ServiceDate = serviceDate.Format("2006-01-02")

		row.DriverRaw = strings.TrimSpace(cellAt(rows[i], 1))
		if row.DriverRaw == "" {
			row.WarningMessage = appendMessage(row.WarningMessage, "未填寫駕駛人，該日搭乘紀錄將沒有司機")
		} else if driver, _ := s.driverRepo.GetByNameNormalized(ctx, namenorm.Normalize(row.DriverRaw)); driver != nil {
			row.DriverID = driver.ID.String()
			row.DriverName = driver.Name
		} else {
			row.WarningMessage = appendMessage(row.WarningMessage, fmt.Sprintf("駕駛人「%s」未比對到司機主檔", row.DriverRaw))
		}

		row.Remark = strings.TrimSpace(cellAt(rows[i], remarkIdx))

		for ci := range columns {
			value := cellAt(rows[i], columns[ci].ColumnIndex-1)
			reported, ok := merge.ParseReportedValue(value)
			if !ok {
				continue
			}
			if reported == "boarded" {
				row.BoardedCount++
				columns[ci].BoardedCount++
			} else {
				row.AbsentCount++
				columns[ci].AbsentCount++
			}
		}

		if row.WarningMessage != "" {
			result.Warnings = append(result.Warnings, ImportWarningItem{RowIndex: rowNum, Message: row.WarningMessage})
			result.WarningRows++
		}
		result.ValidRows++
		result.PreviewRows = append(result.PreviewRows, row)
	}

	if monthDeclared {
		if err := assertRowsWithinMonth(result.PreviewRows, monthStart); err != nil {
			return nil, err
		}
	}

	for _, c := range columns {
		if c.MappingStatus == "pending" {
			result.UnmappedColumns++
		}
	}
	result.Columns = columns

	return result, nil
}

// parseYearMonth 解析宣告的匯入月份，回傳該月第一天；空字串代表未宣告月份。
func parseYearMonth(raw string) (time.Time, bool, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}, false, nil
	}
	start, err := time.Parse("2006-01", raw)
	if err != nil {
		return time.Time{}, false, fmt.Errorf("%w：%s", ErrInvalidYearMonth, raw)
	}
	return start, true, nil
}

// assertRowsWithinMonth 確認所有成功解析日期的列都落在宣告月份內。
func assertRowsWithinMonth(rows []RowPreview, monthStart time.Time) error {
	prefix := monthStart.Format("2006-01")
	for _, row := range rows {
		if row.ServiceDate == "" || strings.HasPrefix(row.ServiceDate, prefix) {
			continue
		}
		return fmt.Errorf("第 %d 列的日期 %s 不屬於宣告匯入的 %s，請確認上傳的是該月份的檔案", row.RowIndex, row.ServiceDate, prefix)
	}
	return nil
}

// daysInMonth 列出該月的每一天，作為覆蓋式重匯的清除範圍。
func daysInMonth(monthStart time.Time) []time.Time {
	next := monthStart.AddDate(0, 1, 0)
	days := make([]time.Time, 0, 31)
	for d := monthStart; d.Before(next); d = d.AddDate(0, 0, 1) {
		days = append(days, d)
	}
	return days
}

// buildColumnPreviews 把檔案表頭與既有 form_columns 對照起來；沒對應過的欄位
// 以姓名相似度推薦個案，並由 [去程]／[回程] 標記推薦趟次。
func (s *DriverReportService) buildColumnPreviews(ctx context.Context, formID uuid.UUID, caseHeaders []string) ([]ColumnPreview, error) {
	existing, err := s.repo.ListColumnsWithMapping(ctx, formID.String(), "")
	if err != nil {
		return nil, err
	}
	known := make(map[string]ColumnMapping, len(existing))
	for _, c := range existing {
		known[c.ColumnHeader] = c
	}

	var candidates []CaseRef
	columns := make([]ColumnPreview, 0, len(caseHeaders))

	for i, h := range caseHeaders {
		parsed := namenorm.ParseColumnHeader(h)
		col := ColumnPreview{
			ColumnIndex:   i + 3,
			ColumnHeader:  h,
			CleanedName:   parsed.CleanedName,
			Direction:     parsed.Direction,
			MappingStatus: "pending",
		}

		if prev, ok := known[h]; ok {
			col.ColumnID = prev.ID
			col.MappingStatus = prev.MappingStatus
			col.CaseID = prev.CaseID
			col.CaseName = prev.CaseName
			col.LegSeq = prev.LegSeq
		}

		if col.MappingStatus == "pending" {
			if candidates == nil {
				if candidates, err = s.caseRepo.ListActiveCases(ctx); err != nil {
					return nil, err
				}
			}
			if id, name, score := bestCaseMatch(parsed.CleanedName, candidates); score > 0 {
				col.SuggestedCaseID = &id
				col.SuggestedCaseName = &name
				col.SuggestionScore = score
			}
			if leg := legSeqForDirection(parsed.Direction); leg != nil {
				col.SuggestedLegSeq = leg
			}
		}

		columns = append(columns, col)
	}
	return columns, nil
}

// bestCaseMatch 取相似度最高的個案；同分時保留先出現者，避免推薦結果隨查詢順序跳動。
func bestCaseMatch(cleanedName string, candidates []CaseRef) (id, name string, score float64) {
	for _, c := range candidates {
		if s := namenorm.ScoreCandidate(cleanedName, c.NameNormalized); s > score {
			id, name, score = c.ID, c.Name, s
		}
	}
	return id, name, score
}

// legSeqForDirection 把去程／回程對應到表單趟次；四趟個案的展開由 ride 模組負責。
func legSeqForDirection(direction string) *int16 {
	switch direction {
	case "outbound":
		leg := int16(1)
		return &leg
	case "inbound":
		leg := int16(2)
		return &leg
	default:
		return nil
	}
}

// findReportHeader 找出表頭列並驗證固定欄位；表頭前允許有標題或空白列。
func findReportHeader(rows [][]string) (int, []string, error) {
	for idx, row := range rows {
		header := trimTrailingEmpty(row)
		if len(header) < minReportColumns {
			continue
		}
		if !strings.Contains(header[0], "日期") || !strings.Contains(header[1], "駕駛") {
			continue
		}
		last := strings.TrimSpace(header[len(header)-1])
		if last != headerRemark && !strings.Contains(last, "問題回報") {
			return 0, nil, fmt.Errorf("最後一欄應為「%s」，目前為「%s」", headerRemark, last)
		}
		return idx, header, nil
	}
	return 0, nil, fmt.Errorf("找不到匯報表表頭，第一列應為「%s、%s、各個案趟次欄、%s」", headerReportDate, headerDriver, headerRemark)
}

// parseReportDate 解析民國日期。接受 1150302、115/3/2、115-03-02 三種寫法，
// 並保留西元 2026-03-02 作為後備，讓從其他系統另存的檔案不會整份匯不進來。
func parseReportDate(raw string) (time.Time, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return time.Time{}, errors.New("empty date")
	}

	if digits, err := strconv.Atoi(value); err == nil && len(value) >= 6 {
		return rocdate.FromROC(digits)
	}

	separator := ""
	for _, sep := range []string{"/", "-", "."} {
		if strings.Contains(value, sep) {
			separator = sep
			break
		}
	}
	if separator == "" {
		return time.Time{}, fmt.Errorf("unrecognised date %q", raw)
	}

	parts := strings.Split(value, separator)
	if len(parts) != 3 {
		return time.Time{}, fmt.Errorf("unrecognised date %q", raw)
	}
	year, err := strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil {
		return time.Time{}, fmt.Errorf("unrecognised date %q", raw)
	}
	month, err := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err != nil {
		return time.Time{}, fmt.Errorf("unrecognised date %q", raw)
	}
	day, err := strconv.Atoi(strings.TrimSpace(parts[2]))
	if err != nil {
		return time.Time{}, fmt.Errorf("unrecognised date %q", raw)
	}
	if year < 1000 {
		return rocdate.FromROC(year*10000 + month*100 + day)
	}
	return rocdate.FromROC((year-1911)*10000 + month*100 + day)
}

func cellAt(row []string, idx int) string {
	if idx < 0 || idx >= len(row) {
		return ""
	}
	return row[idx]
}

func trimTrailingEmpty(row []string) []string {
	end := len(row)
	for end > 0 && strings.TrimSpace(row[end-1]) == "" {
		end--
	}
	return row[:end]
}

func appendMessage(existing, next string) string {
	if existing == "" {
		return next
	}
	return existing + "；" + next
}
