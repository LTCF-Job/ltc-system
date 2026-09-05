package app

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"ltc-system/apps/api/internal/domain/merge"
	"ltc-system/apps/api/internal/domain/namenorm"
	"ltc-system/apps/api/internal/domain/rocdate"

	"github.com/google/uuid"
)

// ErrVehicleRequired 代表建立匯報表時未指定車輛。
var ErrVehicleRequired = errors.New("vehicle is required")

// DriverReportService 負責司機接送匯報表的登記、範本產生、匯入與欄位對應。
type DriverReportService struct {
	repo                FormStore
	excel               SpreadsheetReader
	template            TemplateRenderer
	caseRepo            CaseLookup
	driverRepo          DriverResolver
	rideIngestor        RideIngestor
	attendanceRegistrar AttendanceRegistrar
	auditRepo           AuditWriter
	txRunner            TxRunner
}

// NewDriverReportService 建立 DriverReportService 實例。
func NewDriverReportService(
	repo FormStore,
	excel SpreadsheetReader,
	template TemplateRenderer,
	caseRepo CaseLookup,
	driverRepo DriverResolver,
	rideIngestor RideIngestor,
	attendanceRegistrar AttendanceRegistrar,
	auditRepo AuditWriter,
	txRunner TxRunner,
) *DriverReportService {
	return &DriverReportService{
		repo:                repo,
		excel:               excel,
		template:            template,
		caseRepo:            caseRepo,
		driverRepo:          driverRepo,
		rideIngestor:        rideIngestor,
		attendanceRegistrar: attendanceRegistrar,
		auditRepo:           auditRepo,
		txRunner:            txRunner,
	}
}

// ListForms 查詢所有車輛的匯報表與其對應進度。
func (s *DriverReportService) ListForms(ctx context.Context) ([]ReportForm, error) {
	forms, err := s.repo.ListForms(ctx)
	if err != nil {
		return nil, err
	}
	if forms == nil {
		return []ReportForm{}, nil
	}
	return forms, nil
}

// ListImportedMonths 查詢每份匯報表已匯入哪些月份、各有多少筆與最後匯入時間，
// 供批次上傳畫面判斷某台車某個月是否為重傳。
func (s *DriverReportService) ListImportedMonths(ctx context.Context) ([]ImportedMonth, error) {
	months, err := s.rideIngestor.ListImportedMonths(ctx)
	if err != nil {
		return nil, err
	}
	if months == nil {
		return []ImportedMonth{}, nil
	}
	return months, nil
}

// GetMonthDetail 查詢某份匯報表在指定月份（"YYYY-MM"）已匯入的完整內容，供總覽頁鑽取
// 單一月份時顯示逐日原始回報與展開後的個案搭乘紀錄，不需重新開啟原始檔案。
func (s *DriverReportService) GetMonthDetail(ctx context.Context, formID uuid.UUID, yearMonth string) (*MonthDetail, error) {
	monthStart, monthEnd, _, err := rocdate.MonthRangeStrict(yearMonth)
	if err != nil {
		return nil, err
	}

	submissions, err := s.rideIngestor.ListSubmissionsForFormMonth(ctx, formID, monthStart, monthEnd)
	if err != nil {
		return nil, err
	}
	rideEntries, err := s.rideIngestor.ListRideEntriesForFormMonth(ctx, formID, monthStart, monthEnd)
	if err != nil {
		return nil, err
	}
	if submissions == nil {
		submissions = []MonthSubmissionDetail{}
	}
	if rideEntries == nil {
		rideEntries = []MonthRideEntry{}
	}
	return &MonthDetail{Submissions: submissions, RideEntries: rideEntries}, nil
}

// CreateForm 為一台車建立匯報表；該車已有匯報表時只更新名稱並回傳既有的那一份。
func (s *DriverReportService) CreateForm(ctx context.Context, vehicleID, title string) (*ReportForm, error) {
	vehUUID, err := uuid.Parse(strings.TrimSpace(vehicleID))
	if err != nil {
		return nil, ErrVehicleRequired
	}
	if strings.TrimSpace(title) == "" {
		return nil, errors.New("匯報表名稱不可為空")
	}

	// 一台車一份匯報表，重複建立會撞上 uq_driver_report_forms_vehicle；此時要回傳既有那份的
	// ID，用新產生的 ID 去查會查不到，變成一個沒有原因的 500
	formID, err := s.repo.CreateForm(ctx, uuid.New(), vehUUID, strings.TrimSpace(title))
	if err != nil {
		return nil, err
	}
	return s.repo.GetForm(ctx, formID)
}

// DeleteForm 刪除匯報表；其欄位對應與匯報紀錄由資料庫的 ON DELETE CASCADE 一併清除。
func (s *DriverReportService) DeleteForm(ctx context.Context, formID string) error {
	parsed, err := uuid.Parse(formID)
	if err != nil {
		return ErrFormNotFound
	}
	return s.repo.DeleteForm(ctx, parsed)
}

// ListColumns 查詢欄位對應清單，可依匯報表與對應狀態篩選。
//
// 推薦趟次不入庫：它完全由表頭的 [去程]／[回程] 標記決定，改由查詢時即時推導，
// 避免前端各自再解析一次表頭而產生兩套規則。
func (s *DriverReportService) ListColumns(ctx context.Context, formID, mappingStatus string) ([]ColumnMapping, error) {
	cols, err := s.repo.ListColumnsWithMapping(ctx, formID, mappingStatus)
	if err != nil {
		return nil, err
	}

	out := make([]ColumnMapping, 0, len(cols))
	for _, c := range cols {
		c.SuggestedLegSeq = legSeqForDirection(namenorm.ParseColumnHeader(c.ColumnHeader).Direction)
		out = append(out, c)
	}
	return out, nil
}

// UpdateColumnMapping 更新單一欄位之對應狀態；欄位剛從待維護變成已對應時，順便用
// 既有回報中已存的原始資料回填搭乘紀錄，回傳補寫筆數，讓使用者不必重新上傳檔案。
func (s *DriverReportService) UpdateColumnMapping(ctx context.Context, colID, status string, caseID *string, legSeq *int16) (int, error) {
	if status == "mapped" && (caseID == nil || legSeq == nil) {
		return 0, errors.New("標記為已對應時必須同時指定個案與趟次")
	}

	var backfilled int
	txErr := s.txRunner.WithTx(ctx, func(txCtx context.Context) error {
		formID, header, columnIndex, previousStatus, err := s.repo.UpdateColumnMappingByID(txCtx, colID, status, caseID, legSeq)
		if err != nil {
			return err
		}
		if status != "mapped" || previousStatus == "mapped" {
			return nil
		}

		form, err := s.repo.GetForm(txCtx, formID)
		if err != nil {
			return err
		}
		if form == nil {
			return ErrFormNotFound
		}

		parsedCaseID, err := uuid.Parse(*caseID)
		if err != nil {
			return fmt.Errorf("個案編號格式錯誤: %w", err)
		}

		backfilled, err = s.rideIngestor.BackfillColumn(txCtx, formID, form.VehicleID, header, columnIndex, parsedCaseID, *legSeq)
		return err
	})
	if txErr != nil {
		return 0, txErr
	}
	return backfilled, nil
}

// MatchPendingColumnsByName 找出目前待維護欄位中，清理後姓名與傳入姓名相符（含近似）
// 的欄位，供新建個案後主動詢問使用者這批欄位是否也是同一人。
func (s *DriverReportService) MatchPendingColumnsByName(ctx context.Context, name string) ([]ColumnMapping, error) {
	pending, err := s.repo.ListColumnsWithMapping(ctx, "", "pending")
	if err != nil {
		return nil, err
	}
	return matchPendingColumnsForName(pending, name), nil
}

// BindPendingDriver 把某個比對不到司機主檔的原始姓名綁定到指定司機，回填既有回報已
// 寫入的搭乘紀錄，回傳實際回填的提交筆數；正規化姓名相同的其他待維護列會一併處理，
// 不需要重新上傳原始檔案。
func (s *DriverReportService) BindPendingDriver(ctx context.Context, driverNameRaw, driverID string) (int, error) {
	parsed, err := uuid.Parse(driverID)
	if err != nil {
		return 0, fmt.Errorf("司機編號格式錯誤: %w", err)
	}
	affected, dates, err := s.rideIngestor.BackfillDriver(ctx, driverNameRaw, parsed)
	if err != nil {
		return affected, err
	}
	// 補綁定跟初次匯入時當場比對成功一樣，都要同步司機出勤月曆，不然使用者要再手動登記。
	for _, d := range dates {
		if err := s.attendanceRegistrar.SyncFromImport(ctx, parsed, d); err != nil {
			return affected, fmt.Errorf("同步司機出勤失敗: %w", err)
		}
	}
	return affected, nil
}

// ListSubmissionReview 以匯報提交紀錄（一天一列）為單位彙整目前尚待處理的問題：該列
// 有欄位比對不到個案，或駕駛人比對不到司機主檔，兩者可能同時發生在同一列。
func (s *DriverReportService) ListSubmissionReview(ctx context.Context) ([]SubmissionReview, error) {
	pendingCols, err := s.repo.ListColumnsWithMapping(ctx, "", "pending")
	if err != nil {
		return nil, err
	}

	colsByForm := map[uuid.UUID][]ColumnMapping{}
	var formIDs []uuid.UUID
	for _, c := range pendingCols {
		fid, err := uuid.Parse(c.FormID)
		if err != nil {
			continue
		}
		if _, ok := colsByForm[fid]; !ok {
			formIDs = append(formIDs, fid)
		}
		colsByForm[fid] = append(colsByForm[fid], c)
	}

	var answerRows []SubmissionAnswerRow
	if len(formIDs) > 0 {
		if answerRows, err = s.rideIngestor.ListSubmissionsForForms(ctx, formIDs); err != nil {
			return nil, err
		}
	}

	driverIssues, err := s.rideIngestor.ListUnmatchedDriverSubmissions(ctx)
	if err != nil {
		return nil, err
	}

	order := make([]uuid.UUID, 0, len(answerRows)+len(driverIssues))
	reviews := map[uuid.UUID]*SubmissionReview{}
	ensure := func(id uuid.UUID, formTitle, vehicleName, serviceDate string) *SubmissionReview {
		r, ok := reviews[id]
		if !ok {
			r = &SubmissionReview{SubmissionID: id.String(), FormTitle: formTitle, VehicleName: vehicleName, ServiceDate: serviceDate}
			reviews[id] = r
			order = append(order, id)
		}
		return r
	}

	for _, row := range answerRows {
		var issues []ColumnMapping
		for _, col := range colsByForm[row.FormID] {
			value, ok := row.Answers[col.ColumnHeader]
			if !ok {
				continue
			}
			if _, reported := merge.ParseReportedValue(value); reported {
				issues = append(issues, col)
			}
		}
		if len(issues) == 0 {
			continue
		}
		r := ensure(row.SubmissionID, row.FormTitle, row.VehicleName, row.ServiceDate.Format("2006-01-02"))
		r.CaseIssues = issues
	}

	for _, d := range driverIssues {
		r := ensure(d.SubmissionID, d.FormTitle, d.VehicleName, d.ServiceDate.Format("2006-01-02"))
		r.DriverIssue = &DriverIssue{DriverNameRaw: d.DriverNameRaw}
	}

	out := make([]SubmissionReview, 0, len(order))
	for _, id := range order {
		out = append(out, *reviews[id])
	}
	return out, nil
}

// BatchMapping 批次更新欄位對應狀態，回傳成功更新筆數。
func (s *DriverReportService) BatchMapping(ctx context.Context, updates []ColumnMappingUpdate) (int, error) {
	count := 0
	for _, u := range updates {
		if _, err := s.UpdateColumnMapping(ctx, u.ColumnID, u.MappingStatus, u.CaseID, u.LegSeq); err != nil {
			return count, err
		}
		count++
	}
	return count, nil
}

// TemplateExcel 產生指定匯報表的空白範本；欄位由該車已對應的個案趟次組成，
// 尚未對應過任何欄位時只給出固定欄位，讓使用者先貼上實際表頭再匯入。
func (s *DriverReportService) TemplateExcel(ctx context.Context, formID uuid.UUID) ([]byte, string, error) {
	form, err := s.repo.GetForm(ctx, formID)
	if err != nil {
		return nil, "", err
	}
	if form == nil {
		return nil, "", ErrFormNotFound
	}

	cols, err := s.repo.ListColumnsWithMapping(ctx, formID.String(), "mapped")
	if err != nil {
		return nil, "", err
	}

	headers := make([]string, 0, len(cols))
	for _, c := range cols {
		headers = append(headers, c.ColumnHeader)
	}

	bytesOut, err := s.template.RenderDriverReportTemplate(form.VehicleDisplayName, headers)
	if err != nil {
		return nil, "", fmt.Errorf("產生匯報表範本失敗: %w", err)
	}
	return bytesOut, form.VehicleDisplayName, nil
}
