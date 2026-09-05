package app

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"ltc-system/apps/api/internal/domain/merge"
	"ltc-system/apps/api/internal/domain/namenorm"
	"ltc-system/apps/api/internal/domain/rocdate"
	"ltc-system/apps/api/internal/platform/clock"
)

// RideService 封裝司機接送匯報的展開、正規化、混車合併、衝突裁決與更正。
type RideService struct {
	formRepo        RideRecordStore
	driverRepo      DriverResolver
	caseRepo        ScheduleReader
	auditRepo       AuditWriter
	missingProvider MissingReportProvider
}

var ErrStaleCorrection = errors.New("correction is based on stale ride sources")

// rideAuditSnapshot 避免把查詢組裝出的個案／司機顯示名稱寫入稽核資料。
type rideAuditSnapshot struct {
	ID              uuid.UUID  `json:"id"`
	CaseID          uuid.UUID  `json:"caseId"`
	ServiceDate     time.Time  `json:"serviceDate"`
	LegSeq          int16      `json:"legSeq"`
	EffectiveStatus string     `json:"effectiveStatus"`
	VehicleID       uuid.UUID  `json:"vehicleId"`
	DriverID        *uuid.UUID `json:"driverId,omitempty"`
	HasConflict     bool       `json:"hasConflict"`
}

func newRideAuditSnapshot(item *RideRecord) rideAuditSnapshot {
	if item == nil {
		return rideAuditSnapshot{}
	}
	return rideAuditSnapshot{
		ID:              item.ID,
		CaseID:          item.CaseID,
		ServiceDate:     item.ServiceDate,
		LegSeq:          item.LegSeq,
		EffectiveStatus: item.EffectiveStatus,
		VehicleID:       item.VehicleID,
		DriverID:        item.DriverID,
		HasConflict:     item.HasConflict,
	}
}

// sourceFingerprint 以穩定排序的來源 ID 與內容建立更正依據快照。
func sourceFingerprint(rows []RideSourceRow, serviceDate time.Time, legSeq int16) string {
	parts := make([]string, 0, len(rows))
	for _, row := range rows {
		driver := ""
		if row.DriverID != nil {
			driver = row.DriverID.String()
		}
		parts = append(parts, fmt.Sprintf("%s|%s|%s|%s|%s", row.SourceID, row.VehicleID, driver, row.Reported, row.SubmittedAt.UTC().Format(time.RFC3339Nano)))
	}
	sort.Strings(parts)
	h := sha256.New()
	_, _ = fmt.Fprintf(h, "%s|%d|%s|%s", serviceDate.Format("2006-01-02"), legSeq, strings.Join(parts, ";"), fmt.Sprint(len(rows)))
	return fmt.Sprintf("sha256:%x", h.Sum(nil))
}

// NewRideService 建立 RideService 實例。
func NewRideService(
	formRepo RideRecordStore,
	driverRepo DriverResolver,
	caseRepo ScheduleReader,
	auditRepo AuditWriter,
	missingProvider MissingReportProvider,
) *RideService {
	return &RideService{
		formRepo:        formRepo,
		driverRepo:      driverRepo,
		caseRepo:        caseRepo,
		auditRepo:       auditRepo,
		missingProvider: missingProvider,
	}
}

// ProcessSubmissionRequest 代表一列司機接送匯報；Answers 以欄位表頭為鍵。
type ProcessSubmissionRequest struct {
	ServiceDate time.Time
	SubmittedAt time.Time
	DriverRaw   string
	DriverID    *uuid.UUID
	Remark      string
	Answers     map[string]string
}

// IngestSubmission 將一列匯報展開為搭乘來源與搭乘紀錄，回傳實際寫入的搭乘紀錄筆數。
//
// 呼叫端已決定匯報表與車輛（一台車一份匯報表），本方法只負責欄位對應查表、
// 四趟展開與混車合併。
func (s *RideService) IngestSubmission(ctx context.Context, formID, defaultVehicleID uuid.UUID, req ProcessSubmissionRequest) (int, error) {
	if req.ServiceDate.IsZero() {
		return 0, errors.New("service date is required")
	}

	submittedAt := req.SubmittedAt
	if submittedAt.IsZero() {
		submittedAt = clock.Now()
	}

	driverID := req.DriverID
	req.DriverRaw = strings.TrimSpace(req.DriverRaw)
	if driverID == nil && req.DriverRaw != "" {
		d, err := s.driverRepo.GetByNameNormalized(ctx, namenorm.Normalize(req.DriverRaw))
		if err != nil {
			return 0, fmt.Errorf("failed to resolve driver: %w", err)
		}
		if d != nil {
			driverID = &d.ID
		}
	}

	columns, err := s.formRepo.GetFormColumns(ctx, formID)
	if err != nil {
		return 0, fmt.Errorf("failed to get form columns: %w", err)
	}

	anomalyFlags := detectSubmissionAnomalies(columns, req.Answers)

	rawPayload := map[string]interface{}{
		"serviceDate": req.ServiceDate.Format("2006-01-02"),
		"driverRaw":   req.DriverRaw,
		"remark":      req.Remark,
		"answers":     req.Answers,
	}

	submissionID, err := s.formRepo.SaveFormSubmission(
		ctx, formID, req.ServiceDate, submittedAt, req.DriverRaw, driverID, "import", rawPayload, req.Remark, anomalyFlags,
	)
	if err != nil {
		return 0, fmt.Errorf("failed to save form submission: %w", err)
	}

	written := 0
	for _, col := range columns {
		if col.MappingStatus != "mapped" || col.CaseID == nil || col.LegSeq == nil {
			continue
		}

		value, exists := req.Answers[col.ColumnHeader]
		if !exists {
			continue
		}
		reported, ok := merge.ParseReportedValue(value)
		if !ok {
			continue
		}

		caseID := *col.CaseID
		sched, err := s.caseRepo.GetActiveScheduleForCaseOnDate(ctx, caseID, req.ServiceDate)
		if err != nil {
			return 0, fmt.Errorf("failed to load active schedule: %w", err)
		}

		for _, legSeq := range expandLegSeqs(*col.LegSeq, sched) {
			if err := s.formRepo.InsertRideSource(
				ctx, submissionID, caseID, req.ServiceDate, legSeq, defaultVehicleID, driverID, reported, col.ColumnIndex,
			); err != nil {
				return 0, fmt.Errorf("failed to insert ride source for case %s on %s: %w",
					caseID, req.ServiceDate.Format("2006-01-02"), err)
			}

			if err := s.recalculateRideRecord(ctx, caseID, req.ServiceDate, legSeq, defaultVehicleID, driverID); err != nil {
				return 0, err
			}
			written++
		}
	}

	return written, nil
}

// BackfillColumn 用某欄位既有回報中已存的原始儲存格文字，補寫剛完成個案對應的搭乘紀錄，
// 不需要重新上傳原始檔案；只處理這一欄，其他欄位已寫入的搭乘來源不受影響。
func (s *RideService) BackfillColumn(
	ctx context.Context,
	formID, defaultVehicleID uuid.UUID,
	columnHeader string,
	columnIndex int,
	caseID uuid.UUID,
	legSeq int16,
) (int, error) {
	answers, err := s.formRepo.ListSubmissionAnswersForColumn(ctx, formID, columnHeader)
	if err != nil {
		return 0, fmt.Errorf("failed to list submission answers: %w", err)
	}

	written := 0
	for _, a := range answers {
		reported, ok := merge.ParseReportedValue(a.Value)
		if !ok {
			continue
		}

		sched, err := s.caseRepo.GetActiveScheduleForCaseOnDate(ctx, caseID, a.ServiceDate)
		if err != nil {
			return written, fmt.Errorf("failed to load active schedule: %w", err)
		}
		for _, seq := range expandLegSeqs(legSeq, sched) {
			if err := s.formRepo.InsertRideSource(
				ctx, a.SubmissionID, caseID, a.ServiceDate, seq, defaultVehicleID, a.DriverID, reported, columnIndex,
			); err != nil {
				return written, fmt.Errorf("failed to insert ride source for case %s on %s: %w",
					caseID, a.ServiceDate.Format("2006-01-02"), err)
			}
			if err := s.recalculateRideRecord(ctx, caseID, a.ServiceDate, seq, defaultVehicleID, a.DriverID); err != nil {
				return written, err
			}
			written++
		}
	}
	return written, nil
}

// ListSubmissionsForForms 轉呼叫 repo，供 driverreport 彙整待維護清單。
func (s *RideService) ListSubmissionsForForms(ctx context.Context, formIDs []uuid.UUID) ([]SubmissionFull, error) {
	return s.formRepo.ListSubmissionsForForms(ctx, formIDs)
}

// ListUnmatchedDriverSubmissions 轉呼叫 repo，供 driverreport 彙整待維護清單。
func (s *RideService) ListUnmatchedDriverSubmissions(ctx context.Context) ([]UnmatchedDriverSubmission, error) {
	return s.formRepo.ListUnmatchedDriverSubmissions(ctx)
}

// ListSubmissionsForFormMonth 轉呼叫 repo，供 driverreport 的總覽頁鑽取單一月份的逐日回報明細。
func (s *RideService) ListSubmissionsForFormMonth(ctx context.Context, formID uuid.UUID, monthStart, monthEnd time.Time) ([]MonthSubmissionDetail, error) {
	return s.formRepo.ListSubmissionsForFormMonth(ctx, formID, monthStart, monthEnd)
}

// ListRideEntriesForFormMonth 轉呼叫 repo，供 driverreport 的總覽頁鑽取單一月份的逐個案搭乘紀錄。
func (s *RideService) ListRideEntriesForFormMonth(ctx context.Context, formID uuid.UUID, monthStart, monthEnd time.Time) ([]MonthRideEntry, error) {
	return s.formRepo.ListRideEntriesForFormMonth(ctx, formID, monthStart, monthEnd)
}

// BackfillDriver 把姓名正規化後相符、目前比對不到司機主檔的既有回報一次回填為指定
// 司機，不需要重新上傳原始檔案；回傳實際回填的提交筆數，以及這些回報涉及的服務日期
// （去重），供呼叫端同步司機出勤月曆。
func (s *RideService) BackfillDriver(ctx context.Context, driverNameRaw string, driverID uuid.UUID) (int, []time.Time, error) {
	target := namenorm.Normalize(driverNameRaw)
	if target == "" {
		return 0, nil, nil
	}

	unmatched, err := s.formRepo.ListUnmatchedDriverSubmissions(ctx)
	if err != nil {
		return 0, nil, fmt.Errorf("failed to list unmatched driver submissions: %w", err)
	}

	backfilled := 0
	seenDates := map[string]bool{}
	var dates []time.Time
	for _, u := range unmatched {
		if namenorm.Normalize(u.DriverNameRaw) != target {
			continue
		}
		if err := s.formRepo.UpdateSubmissionDriverID(ctx, u.SubmissionID, driverID); err != nil {
			return backfilled, dates, fmt.Errorf("failed to update submission driver: %w", err)
		}

		sources, err := s.formRepo.ListRideSourcesForSubmission(ctx, u.SubmissionID)
		if err != nil {
			return backfilled, dates, fmt.Errorf("failed to list ride sources for submission: %w", err)
		}
		for _, src := range sources {
			if err := s.formRepo.UpdateRideSourceDriverID(ctx, src.ID, driverID); err != nil {
				return backfilled, dates, fmt.Errorf("failed to update ride source driver: %w", err)
			}
			if err := s.recalculateRideRecord(ctx, src.CaseID, src.ServiceDate, src.LegSeq, src.VehicleID, &driverID); err != nil {
				return backfilled, dates, err
			}
		}
		backfilled++
		dateKey := u.ServiceDate.Format("2006-01-02")
		if !seenDates[dateKey] {
			seenDates[dateKey] = true
			dates = append(dates, u.ServiceDate)
		}
	}
	return backfilled, dates, nil
}

// ClearImportedDates 移除指定匯報表在這些服務日期已寫入的匯入資料，讓重匯成為覆蓋而非疊加。
// 回傳刪除的提交紀錄筆數。
//
// 只刪本匯報表產生的 form_submissions，ride_sources 由 ON DELETE CASCADE 連帶清除；
// 其他車輛對同一 slot 的混車來源保持不動，清除後逐 slot 重算合併結果。
func (s *RideService) ClearImportedDates(ctx context.Context, formID uuid.UUID, dates []time.Time) (int, error) {
	if len(dates) == 0 {
		return 0, nil
	}

	// 來源列刪除後就查不到受影響的 slot，必須在刪除前收集
	slots, err := s.formRepo.ListRideSourceSlotsForForm(ctx, formID, dates)
	if err != nil {
		return 0, fmt.Errorf("failed to list affected ride slots: %w", err)
	}

	removed, err := s.formRepo.DeleteFormSubmissions(ctx, formID, dates)
	if err != nil {
		return 0, fmt.Errorf("failed to delete form submissions: %w", err)
	}

	for _, slot := range slots {
		rows, err := s.formRepo.ListRideSourcesForSlot(ctx, slot.CaseID, slot.ServiceDate, slot.LegSeq)
		if err != nil {
			return 0, fmt.Errorf("failed to load ride sources for slot: %w", err)
		}
		// 來源全部清空的 slot 不能靠重算修正，否則會留下沒有來源支撐的過期紀錄
		if len(rows) == 0 {
			if err := s.formRepo.DeleteDerivedRideRecord(ctx, slot.CaseID, slot.ServiceDate, slot.LegSeq); err != nil {
				return 0, fmt.Errorf("failed to delete derived ride record: %w", err)
			}
			continue
		}
		// 預設車輛取自剩下的來源，不能沿用剛被移除的那台車：全員回報「沒坐」時
		// merge 會退回預設值，用已清掉的車輛會把錯誤的車寫進搭乘紀錄
		if err := s.recalculateRideRecord(ctx, slot.CaseID, slot.ServiceDate, slot.LegSeq, rows[0].VehicleID, nil); err != nil {
			return 0, err
		}
	}

	return removed, nil
}

// ListImportedMonths 統計每份匯報表各月份已匯入的提交筆數與最後一次匯入時間。
//
// 月份不落地成欄位，一律由 form_submissions.service_date 推得，避免統計與實際資料不同步。
func (s *RideService) ListImportedMonths(ctx context.Context) ([]ImportedMonth, error) {
	months, err := s.formRepo.ListImportedMonths(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list imported months: %w", err)
	}
	if months == nil {
		return []ImportedMonth{}, nil
	}
	return months, nil
}

// detectSubmissionAnomalies 找出這列匯報中無法辨識的欄位值與未完成對應的欄位。
// 空白儲存格代表未回報，不視為異常。
func detectSubmissionAnomalies(columns []FormColumn, answers map[string]string) []string {
	var flags []string
	for _, col := range columns {
		value, exists := answers[col.ColumnHeader]
		if !exists || strings.TrimSpace(value) == "" {
			continue
		}
		if col.MappingStatus != "mapped" {
			flags = append(flags, fmt.Sprintf("unmapped_column:%s", col.ColumnHeader))
			continue
		}
		if _, ok := merge.ParseReportedValue(value); !ok {
			flags = append(flags, fmt.Sprintf("unparsed_value:%s:%s", col.ColumnHeader, value))
		}
	}
	return flags
}

// expandLegSeqs 套用四趟展開規則（R4 / §5.5）：表單第 1 趟展開為 1、3 趟；
// 第 2 趟展開為 2、4 趟。其餘趟數維持原趟次。
func expandLegSeqs(baseLegSeq int16, sched *CaseSchedule) []int16 {
	if sched == nil || sched.TripPattern != 4 {
		return []int16{baseLegSeq}
	}
	switch baseLegSeq {
	case 1:
		return []int16{1, 3}
	case 2:
		return []int16{2, 4}
	default:
		return []int16{baseLegSeq}
	}
}

// recalculateRideRecord 重新執行單筆 slot 之混車合併運算並更新主表。
//
// 匯入路徑把整段包在同一個交易內，任何一次讀寫失敗都會讓後續語句全部失效，
// 因此這裡不吞錯誤：錯誤原樣往上傳，讓呼叫端能回報真正的根因並回滾。
func (s *RideService) recalculateRideRecord(
	ctx context.Context,
	caseID uuid.UUID,
	serviceDate time.Time,
	legSeq int16,
	defaultVehicleID uuid.UUID,
	defaultDriverID *uuid.UUID,
) error {
	// 查詢既有紀錄以保護人工裁決與更正
	existingRec, err := s.formRepo.GetRideRecordForSlot(ctx, caseID, serviceDate, legSeq)
	if err != nil {
		return fmt.Errorf("failed to load existing ride record: %w", err)
	}

	// 查詢當日排班設定預設車輛與司機
	sched, err := s.caseRepo.GetActiveScheduleForCaseOnDate(ctx, caseID, serviceDate)
	if err != nil {
		return fmt.Errorf("failed to load active schedule: %w", err)
	}
	if sched != nil {
		for _, l := range sched.Legs {
			if l.LegSeq == legSeq && l.VehicleID != nil {
				defaultVehicleID = *l.VehicleID
				// 一台車當日可能有多位司機，無從判斷是誰出車時留空由人工指定
				drivers, err := s.driverRepo.ListDriversForVehicleOnDate(ctx, defaultVehicleID, serviceDate)
				if err != nil {
					return fmt.Errorf("failed to load scheduled drivers: %w", err)
				}
				if len(drivers) == 1 {
					defaultDriverID = &drivers[0].ID
				}
				break
			}
		}
	}

	// 載入該 slot 的全部來源列。此處必須讀實際寫入的 reported 值：先前是以固定的
	// "boarded" 當唯一來源，會讓匯報「沒坐」的個案在月曆上顯示成有坐。
	rows, err := s.formRepo.ListRideSourcesForSlot(ctx, caseID, serviceDate, legSeq)
	if err != nil {
		return fmt.Errorf("failed to load ride sources for slot: %w", err)
	}
	if len(rows) == 0 {
		return s.formRepo.DeleteDerivedRideRecord(ctx, caseID, serviceDate, legSeq)
	}

	fingerprint := sourceFingerprint(rows, serviceDate, legSeq)
	correctionIsCurrent := existingRec != nil &&
		(existingRec.BasedOnFingerprint == "" || existingRec.BasedOnFingerprint == fingerprint)

	var existingState *merge.ExistingRecordState
	if existingRec != nil {
		existingState = &merge.ExistingRecordState{
			HasConflict:        existingRec.HasConflict,
			ConflictResolvedAt: existingRec.ConflictResolvedAt,
			ResolvedVehicleID:  &existingRec.VehicleID,
			ResolvedDriverID:   existingRec.DriverID,
		}
		if correctionIsCurrent {
			existingState.CorrectedAt = existingRec.CorrectedAt
			existingState.CorrectedBy = existingRec.CorrectedBy
			existingState.EffectiveStatus = existingRec.EffectiveStatus
			existingState.CorrectedVehicle = &existingRec.VehicleID
			existingState.CorrectedDriver = existingRec.DriverID
		}
	}

	sources := make([]merge.RideSourceInput, 0, len(rows))
	for _, row := range rows {
		sources = append(sources, merge.RideSourceInput{
			SourceID:       row.SourceID,
			SourcePriority: row.SourcePriority,
			VehicleID:      row.VehicleID,
			DriverID:       row.DriverID,
			Reported:       row.Reported,
			SubmittedAt:    row.SubmittedAt,
		})
	}

	result := merge.MergeRideSources(sources, existingState, defaultVehicleID, defaultDriverID)

	rec := RideRecord{
		CaseID:             caseID,
		ServiceDate:        serviceDate,
		LegSeq:             legSeq,
		MergedStatus:       result.MergedStatus,
		EffectiveStatus:    result.EffectiveStatus,
		VehicleID:          result.SelectedVehicle,
		DriverID:           result.SelectedDriver,
		HasConflict:        result.HasConflict,
		BasedOnFingerprint: fingerprint,
	}
	if existingRec != nil {
		rec.ID = existingRec.ID
		rec.ConflictResolvedAt = existingRec.ConflictResolvedAt
		rec.ConflictResolvedBy = existingRec.ConflictResolvedBy
		if correctionIsCurrent {
			rec.CorrectedAt = existingRec.CorrectedAt
			rec.CorrectedBy = existingRec.CorrectedBy
			rec.CorrectionReason = existingRec.CorrectionReason
			rec.NotClaimedAA09 = existingRec.NotClaimedAA09
		}
	}

	if err := s.formRepo.UpsertRideRecord(ctx, &rec); err != nil {
		return fmt.Errorf("failed to upsert ride record: %w", err)
	}
	return nil
}

// CorrectRideRecordRequest 代表更正搭乘紀錄之請求結構體。
type CorrectRideRecordRequest struct {
	EffectiveStatus     *string    `json:"effectiveStatus"`
	VehicleID           *uuid.UUID `json:"vehicleId"`
	DriverID            *uuid.UUID `json:"driverId"`
	DepartTimeOverride  *string    `json:"departTimeOverride"`
	DurationMinOverride *int16     `json:"durationMinOverride"`
	NotClaimedAA09      *bool      `json:"notClaimedAa09"`
	Reason              *string    `json:"reason"`
	BasedOnFingerprint  *string    `json:"basedOnFingerprint"`
}

// ManualReportRideRequest 代表人工補登或編輯回報內容之請求結構體。
type ManualReportRideRequest struct {
	ID                  *string    `json:"id"`
	CaseID              uuid.UUID  `json:"caseId"`
	ServiceDate         string     `json:"serviceDate"`
	LegSeq              int16      `json:"legSeq"`
	EffectiveStatus     string     `json:"effectiveStatus"`
	VehicleID           *uuid.UUID `json:"vehicleId"`
	DriverID            *uuid.UUID `json:"driverId"`
	DepartTimeOverride  *string    `json:"departTimeOverride"`
	DurationMinOverride *int16     `json:"durationMinOverride"`
	NotClaimedAA09      *bool      `json:"notClaimedAa09"`
	Reason              *string    `json:"reason"`
}

// CorrectRideRecord 人工更正搭乘紀錄並寫入稽核留痕（§4.7）。
func (s *RideService) CorrectRideRecord(
	ctx context.Context,
	rideID uuid.UUID,
	req CorrectRideRecordRequest,
	actorID uuid.UUID,
	actorRole, ip, ua string,
) error {
	before, err := s.formRepo.GetRideRecordByID(ctx, rideID)
	if err != nil {
		return fmt.Errorf("failed to load ride record: %w", err)
	}
	if before == nil {
		return ErrRideNotFound
	}
	sources, err := s.formRepo.ListRideSourcesForSlot(ctx, before.CaseID, before.ServiceDate, before.LegSeq)
	if err != nil {
		return fmt.Errorf("failed to load ride sources for correction: %w", err)
	}
	fingerprint := sourceFingerprint(sources, before.ServiceDate, before.LegSeq)
	if req.BasedOnFingerprint != nil && *req.BasedOnFingerprint != fingerprint {
		return ErrStaleCorrection
	}
	if store, ok := s.formRepo.(CorrectionFingerprintingStore); ok {
		err = store.CorrectRideRecordWithFingerprint(
			ctx, rideID, req.EffectiveStatus, req.VehicleID, req.DriverID,
			req.DepartTimeOverride, req.DurationMinOverride, req.NotClaimedAA09,
			req.Reason, actorID, fingerprint,
		)
	} else {
		err = s.formRepo.CorrectRideRecord(
			ctx, rideID, req.EffectiveStatus, req.VehicleID, req.DriverID,
			req.DepartTimeOverride, req.DurationMinOverride, req.NotClaimedAA09, req.Reason, actorID,
		)
		if err == nil {
			if store, ok := s.formRepo.(CorrectionFingerprintStore); ok {
				err = store.SetCorrectionFingerprint(ctx, rideID, fingerprint)
			}
		}
	}
	if err != nil {
		return fmt.Errorf("failed to correct ride record: %w", err)
	}

	if s.auditRepo != nil {
		entityIDStr := rideID.String()
		_ = s.auditRepo.Write(ctx, AuditEntry{
			ActorID:    &actorID,
			ActorRole:  &actorRole,
			Action:     "correct",
			EntityType: "ride_records",
			EntityID:   &entityIDStr,
			AfterData:  req,
			IPAddress:  &ip,
			UserAgent:  &ua,
		})
	}

	return nil
}

// ManualReportRide 人工輸入回報內容並儲存搭乘紀錄。
func (s *RideService) ManualReportRide(
	ctx context.Context,
	req ManualReportRideRequest,
	actorID uuid.UUID,
	actorRole, ip, ua string,
) (*RideRecord, error) {
	if req.EffectiveStatus != "boarded" && req.EffectiveStatus != "absent" {
		return nil, fmt.Errorf("無效的搭乘狀態：%s", req.EffectiveStatus)
	}

	serviceDate, err := rocdate.ParseDate(req.ServiceDate)
	if err != nil {
		return nil, fmt.Errorf("無效的服務日期格式：%s", req.ServiceDate)
	}

	// 車輛未指定時由排班回退取得預設車輛
	var vehicleID uuid.UUID
	if req.VehicleID != nil && *req.VehicleID != uuid.Nil {
		vehicleID = *req.VehicleID
	} else if s.caseRepo != nil {
		sched, err := s.caseRepo.GetActiveScheduleForCaseOnDate(ctx, req.CaseID, serviceDate)
		if err != nil {
			return nil, fmt.Errorf("failed to load active schedule: %w", err)
		}
		if sched != nil {
			for _, l := range sched.Legs {
				if l.LegSeq == req.LegSeq && l.VehicleID != nil {
					vehicleID = *l.VehicleID
					break
				}
			}
		}
	}

	existingRec, err := s.formRepo.GetRideRecordForSlot(ctx, req.CaseID, serviceDate, req.LegSeq)
	if err != nil {
		return nil, fmt.Errorf("failed to load existing ride record: %w", err)
	}
	now := clock.Now()

	rec := RideRecord{
		CaseID:              req.CaseID,
		ServiceDate:         serviceDate,
		LegSeq:              req.LegSeq,
		MergedStatus:        req.EffectiveStatus,
		EffectiveStatus:     req.EffectiveStatus,
		VehicleID:           vehicleID,
		DriverID:            req.DriverID,
		HasConflict:         false,
		DepartTimeOverride:  req.DepartTimeOverride,
		DurationMinOverride: req.DurationMinOverride,
		CorrectedBy:         &actorID,
		CorrectedAt:         &now,
		CorrectionReason:    req.Reason,
	}
	if req.NotClaimedAA09 != nil {
		rec.NotClaimedAA09 = *req.NotClaimedAA09
	}

	if existingRec != nil {
		rec.ID = existingRec.ID
	} else {
		rec.ID = uuid.New()
	}

	if err := s.formRepo.UpsertRideRecord(ctx, &rec); err != nil {
		return nil, fmt.Errorf("failed to upsert ride record: %w", err)
	}

	if s.auditRepo != nil {
		entityIDStr := rec.ID.String()
		_ = s.auditRepo.Write(ctx, AuditEntry{
			ActorID:    &actorID,
			ActorRole:  &actorRole,
			Action:     "manual_report",
			EntityType: "ride_records",
			EntityID:   &entityIDStr,
			AfterData:  req,
			IPAddress:  &ip,
			UserAgent:  &ua,
		})
	}

	return &rec, nil
}

// GetRecord 取得單筆搭乘紀錄詳情，查無資料回 ErrRideNotFound。
func (s *RideService) GetRecord(ctx context.Context, id uuid.UUID) (*RideRecord, error) {
	rec, err := s.formRepo.GetRideRecordByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get ride record: %w", err)
	}
	if rec == nil {
		return nil, ErrRideNotFound
	}
	return rec, nil
}

// ResolveConflictInput 代表裁決混車衝突之請求結構體。
type ResolveConflictInput struct {
	VehicleID uuid.UUID
	DriverID  *uuid.UUID
	Reason    *string
}

// ResolveConflict 人工裁決同車衝突回報，把裁決結果寫回搭乘紀錄並留存稽核。
func (s *RideService) ResolveConflict(ctx context.Context, rideID uuid.UUID, req ResolveConflictInput, actorID uuid.UUID, actorRole string) error {
	before, err := s.formRepo.GetRideRecordByID(ctx, rideID)
	if err != nil {
		return fmt.Errorf("failed to load ride record: %w", err)
	}
	if before == nil {
		return ErrRideNotFound
	}

	resolved, err := s.formRepo.ResolveConflict(ctx, rideID, req.VehicleID, req.DriverID, req.Reason, actorID)
	if err != nil {
		return fmt.Errorf("failed to resolve conflict: %w", err)
	}
	if !resolved {
		return ErrConflictAlreadyResolved
	}

	if s.auditRepo != nil {
		entityIDStr := rideID.String()
		_ = s.auditRepo.Write(ctx, AuditEntry{
			ActorID:    &actorID,
			ActorRole:  &actorRole,
			Action:     "resolve_conflict",
			EntityType: "ride_records",
			EntityID:   &entityIDStr,
			BeforeData: newRideAuditSnapshot(before),
			AfterData:  req,
		})
	}

	return nil
}

// IssueRide 是「異常集中處理」分頁的單一列，三種 issueType 共用同一個形狀。
type IssueRide struct {
	ID          string
	CaseID      string
	CaseName    string
	ServiceDate time.Time
	LegSeq      int16
	Description string
	Vehicles    []string
}

// ListIssues 依 issueType 分派查詢「異常集中處理」清單，month 格式為 YYYY-MM。
func (s *RideService) ListIssues(ctx context.Context, issueType string, year, month int, region, keyword string, page, pageSize int) ([]IssueRide, int64, error) {
	start := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 1, 0).Add(-time.Second)

	switch issueType {
	case "conflict":
		return s.listConflictIssues(ctx, start, end, keyword, page, pageSize)
	case "unreported":
		return s.listUnreportedIssues(ctx, year, month, region, page, pageSize)
	case "import_error":
		return s.listImportErrorIssues(ctx, start, end, keyword, page, pageSize)
	default:
		return nil, 0, fmt.Errorf("unknown issue type: %s", issueType)
	}
}

func (s *RideService) listConflictIssues(ctx context.Context, start, end time.Time, keyword string, page, pageSize int) ([]IssueRide, int64, error) {
	rows, total, err := s.formRepo.ListPendingConflicts(ctx, start, end, keyword, page, pageSize)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list pending conflicts: %w", err)
	}
	items := make([]IssueRide, 0, len(rows))
	for _, r := range rows {
		items = append(items, IssueRide{
			ID:          r.ID.String(),
			CaseID:      r.CaseID.String(),
			CaseName:    r.CaseName,
			ServiceDate: r.ServiceDate,
			LegSeq:      r.LegSeq,
			Description: conflictDescription(r),
			Vehicles:    r.Vehicles,
		})
	}
	return items, total, nil
}

func (s *RideService) listUnreportedIssues(ctx context.Context, year, month int, region string, page, pageSize int) ([]IssueRide, int64, error) {
	if s.missingProvider == nil {
		return []IssueRide{}, 0, nil
	}
	rows, err := s.missingProvider.ListMissingForMonth(ctx, year, month, region)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list missing reports: %w", err)
	}

	total := int64(len(rows))
	from := (page - 1) * pageSize
	if from > len(rows) {
		from = len(rows)
	}
	to := from + pageSize
	if to > len(rows) {
		to = len(rows)
	}

	items := make([]IssueRide, 0, to-from)
	for _, r := range rows[from:to] {
		items = append(items, IssueRide{
			ID:          fmt.Sprintf("unreported:%s:%s:%d", r.CaseID, r.ServiceDate.Format("2006-01-02"), r.LegSeq),
			CaseID:      r.CaseID.String(),
			CaseName:    r.CaseName,
			ServiceDate: r.ServiceDate,
			LegSeq:      r.LegSeq,
			Description: unreportedDescription(r),
		})
	}
	return items, total, nil
}

func (s *RideService) listImportErrorIssues(ctx context.Context, start, end time.Time, keyword string, page, pageSize int) ([]IssueRide, int64, error) {
	rows, total, err := s.formRepo.ListImportErrorSubmissions(ctx, start, end, keyword, page, pageSize)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list import error submissions: %w", err)
	}
	items := make([]IssueRide, 0, len(rows))
	for _, r := range rows {
		items = append(items, IssueRide{
			ID:          r.ID.String(),
			CaseName:    r.DriverNameRaw,
			ServiceDate: r.ServiceDate,
			Description: describeAnomalyFlags(r.AnomalyFlags),
		})
	}
	return items, total, nil
}

func conflictDescription(r ConflictRide) string {
	return fmt.Sprintf("同一趟次有 %d 台車輛回報「有坐」，需人工裁決", len(r.Vehicles))
}

func unreportedDescription(r MissingRide) string {
	return "應搭乘但尚未有任何司機回報"
}

func describeAnomalyFlags(flags []string) string {
	if len(flags) == 0 {
		return "匯入資料異常"
	}
	return strings.Join(flags, "；")
}
