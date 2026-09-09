package app

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"log/slog"
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

// rideCorrectionAuditSnapshot 是更正 PATCH 的固定快照；用 Present 欄位保留三態語意，
// 不直接把含有自由文字的 request 寫入稽核資料。
type rideCorrectionAuditSnapshot struct {
	RideID                 uuid.UUID  `json:"rideId"`
	EffectiveStatusPresent bool       `json:"effectiveStatusPresent"`
	EffectiveStatus        *string    `json:"effectiveStatus,omitempty"`
	VehicleIDPresent       bool       `json:"vehicleIdPresent"`
	VehicleID              *uuid.UUID `json:"vehicleId,omitempty"`
	DriverIDPresent        bool       `json:"driverIdPresent"`
	DriverID               *uuid.UUID `json:"driverId,omitempty"`
	DepartTimePresent      bool       `json:"departTimeOverridePresent"`
	DepartTimeOverride     *string    `json:"departTimeOverride,omitempty"`
	DurationPresent        bool       `json:"durationMinOverridePresent"`
	DurationMinOverride    *int16     `json:"durationMinOverride,omitempty"`
	NotClaimedAA09Present  bool       `json:"notClaimedAa09Present"`
	NotClaimedAA09         *bool      `json:"notClaimedAa09,omitempty"`
	ReasonPresent          bool       `json:"reasonPresent"`
	BasedOnFingerprint     string     `json:"basedOnFingerprint,omitempty"`
}

// rideConflictResolutionAuditSnapshot 是衝突裁決後的非敏感固定快照，不保存裁決自由文字。
type rideConflictResolutionAuditSnapshot struct {
	ID          uuid.UUID  `json:"id"`
	VehicleID   uuid.UUID  `json:"vehicleId"`
	DriverID    *uuid.UUID `json:"driverId,omitempty"`
	HasConflict bool       `json:"hasConflict"`
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

func newRideCorrectionAuditSnapshot(rideID uuid.UUID, req CorrectRideRecordRequest) rideCorrectionAuditSnapshot {
	return rideCorrectionAuditSnapshot{
		RideID:                 rideID,
		EffectiveStatusPresent: req.EffectiveStatus.Present,
		EffectiveStatus:        req.EffectiveStatus.Value,
		VehicleIDPresent:       req.VehicleID.Present,
		VehicleID:              req.VehicleID.Value,
		DriverIDPresent:        req.DriverID.Present,
		DriverID:               req.DriverID.Value,
		DepartTimePresent:      req.DepartTimeOverride.Present,
		DepartTimeOverride:     req.DepartTimeOverride.Value,
		DurationPresent:        req.DurationMinOverride.Present,
		DurationMinOverride:    req.DurationMinOverride.Value,
		NotClaimedAA09Present:  req.NotClaimedAA09.Present,
		NotClaimedAA09:         req.NotClaimedAA09.Value,
		ReasonPresent:          req.Reason.Present,
		BasedOnFingerprint:     valueOrEmpty(req.BasedOnFingerprint),
	}
}

func valueOrEmpty(value *string) string {
	if value == nil {
		return ""
	}
	return *value
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

// IngestResult 彙整一次逐欄寫入的結果，把「新增」「無變化的重複回報」「進待維護」分開計算，
// 讓匯入結果訊息能區分這三種情況，而不是用單一數字掩蓋掉需要使用者處理的衝突。
type IngestResult struct {
	Written    int // 這台車在這個 slot 第一次出現，直接寫入
	Reaffirmed int // 值與這台車既有資料相同的重複回報，未產生新來源
	Staged     int // 值與這台車既有資料不同，已進入待維護等待使用者選擇
}

// IngestSubmission 將一列匯報展開為搭乘來源與搭乘紀錄；回傳值把新增、無變化重複回報、
// 進待維護三種結果分開計算。
//
// 呼叫端已決定匯報表與車輛（一台車一份匯報表），本方法只負責欄位對應查表、
// 四趟展開與混車合併。
func (s *RideService) IngestSubmission(ctx context.Context, formID, defaultVehicleID uuid.UUID, req ProcessSubmissionRequest) (IngestResult, error) {
	if req.ServiceDate.IsZero() {
		return IngestResult{}, errors.New("service date is required")
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
			return IngestResult{}, fmt.Errorf("failed to resolve driver: %w", err)
		}
		if d != nil {
			driverID = &d.ID
		}
	}

	columns, err := s.formRepo.GetFormColumns(ctx, formID)
	if err != nil {
		return IngestResult{}, fmt.Errorf("failed to get form columns: %w", err)
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
		return IngestResult{}, fmt.Errorf("failed to save form submission: %w", err)
	}

	var result IngestResult
	if driverID == nil {
		// 駕駛人比對不到司機主檔：留在 form_submissions 待維護，不展開成搭乘來源，
		// 避免一筆缺司機的資料先出現在司機日曆等其他頁面，等使用者綁定後才由
		// BackfillDriver 補寫。
		return result, nil
	}
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
			return result, fmt.Errorf("failed to load active schedule: %w", err)
		}

		for _, legSeq := range expandLegSeqs(*col.LegSeq, sched) {
			outcome, err := s.reconcileRideSource(ctx, formID, submissionID, caseID, req.ServiceDate, legSeq, defaultVehicleID, driverID, reported, col.ColumnIndex, submittedAt)
			if err != nil {
				return result, fmt.Errorf("failed to reconcile ride source for case %s on %s: %w",
					caseID, req.ServiceDate.Format("2006-01-02"), err)
			}
			switch outcome {
			case reconcileInserted:
				result.Written++
			case reconcileReaffirmed:
				result.Reaffirmed++
			case reconcileStaged:
				result.Staged++
			}
		}
	}

	return result, nil
}

// reconcileOutcome 是逐格寫入前比對既有資料後的處理結果。
type reconcileOutcome int

const (
	reconcileInserted reconcileOutcome = iota
	reconcileReaffirmed
	reconcileStaged
)

// reconcileRideSource 是「同一台車同一個案」逐列比對的唯一入口，IngestSubmission、
// BackfillColumn、BackfillDriver 都透過它決定要直接寫入、視為無變化的重複回報，
// 還是進待維護等待使用者選擇——三個進入點必須共用同一套判斷，否則行為會彼此不一致。
//
// 判斷依這台車在這個 slot（case_id, service_date, leg_seq, vehicle_id）目前最新的
// 一筆來源：不存在就直接寫入；回報值與司機都相同視為重複回報；任一不同則暫存衝突，
// 保留既有來源不動，等使用者裁決要保留哪一筆（見 docs/decisions/driver-report-import-overwrite.md）。
func (s *RideService) reconcileRideSource(
	ctx context.Context,
	formID, submissionID, caseID uuid.UUID,
	serviceDate time.Time,
	legSeq int16,
	vehicleID uuid.UUID,
	driverID *uuid.UUID,
	reported string,
	colIdx int,
	submittedAt time.Time,
) (reconcileOutcome, error) {
	existing, err := s.formRepo.ListRideSourcesForSlot(ctx, caseID, serviceDate, legSeq)
	if err != nil {
		return 0, fmt.Errorf("failed to load existing ride sources for slot: %w", err)
	}

	var current *RideSourceRow
	for i := range existing {
		if existing[i].VehicleID == vehicleID {
			current = &existing[i] // 已依 submitted_at DESC 排序，第一筆即這台車最新的來源
			break
		}
	}

	if current == nil {
		if err := s.formRepo.InsertRideSource(ctx, submissionID, caseID, serviceDate, legSeq, vehicleID, driverID, reported, colIdx, submittedAt); err != nil {
			return 0, err
		}
		if err := s.recalculateRideRecord(ctx, caseID, serviceDate, legSeq, vehicleID, driverID); err != nil {
			return 0, err
		}
		return reconcileInserted, nil
	}

	if current.Reported == reported && sameDriver(current.DriverID, driverID) {
		return reconcileReaffirmed, nil
	}

	if _, err := s.formRepo.UpsertRideSourceRowConflict(ctx, RowConflictInput{
		FormID:               formID,
		VehicleID:            vehicleID,
		CaseID:               caseID,
		ServiceDate:          serviceDate,
		LegSeq:               legSeq,
		SourceColumnIndex:    colIdx,
		PreviousSubmissionID: current.SubmissionID,
		PreviousReported:     current.Reported,
		PreviousDriverID:     current.DriverID,
		PreviousSubmittedAt:  current.SubmittedAt,
		NewSubmissionID:      submissionID,
		NewReported:          reported,
		NewDriverID:          driverID,
		NewSubmittedAt:       submittedAt,
	}); err != nil {
		return 0, err
	}
	return reconcileStaged, nil
}

// sameDriver 比較兩個可為 nil 的司機 ID 是否代表同一人；兩者皆為 nil 視為相同。
func sameDriver(a, b *uuid.UUID) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

// BackfillColumn 用某欄位既有回報中已存的原始儲存格文字，補寫剛完成個案對應的搭乘紀錄，
// 不需要重新上傳原始檔案；只處理這一欄，其他欄位已寫入的搭乘來源不受影響。
// skipDates 列出「值另有來源、不該用既有 payload 補」的服務日期，呼叫端沒有這種日期時傳 nil。
func (s *RideService) BackfillColumn(
	ctx context.Context,
	formID, defaultVehicleID uuid.UUID,
	columnHeader string,
	columnIndex int,
	caseID uuid.UUID,
	legSeq int16,
	skipDates []time.Time,
) (int, error) {
	answers, err := s.formRepo.ListSubmissionAnswersForColumn(ctx, formID, columnHeader)
	if err != nil {
		return 0, fmt.Errorf("failed to list submission answers: %w", err)
	}

	skip := make(map[string]struct{}, len(skipDates))
	for _, d := range skipDates {
		skip[d.Format("2006-01-02")] = struct{}{}
	}

	written := 0
	for _, a := range answers {
		// 這些日期的權威值是呼叫端手上那份檔案，payload 可能還是上一次上傳的舊值
		if _, skipped := skip[a.ServiceDate.Format("2006-01-02")]; skipped {
			continue
		}
		if a.DriverID == nil {
			// 司機仍待維護：留在 form_submissions，等司機也綁定後由 BackfillDriver 補寫，
			// 避免一筆缺司機的資料先出現在司機日曆等其他頁面。
			continue
		}
		reported, ok := merge.ParseReportedValue(a.Value)
		if !ok {
			continue
		}

		sched, err := s.caseRepo.GetActiveScheduleForCaseOnDate(ctx, caseID, a.ServiceDate)
		if err != nil {
			return written, fmt.Errorf("failed to load active schedule: %w", err)
		}
		for _, seq := range expandLegSeqs(legSeq, sched) {
			outcome, err := s.reconcileRideSource(ctx, formID, a.SubmissionID, caseID, a.ServiceDate, seq, defaultVehicleID, a.DriverID, reported, columnIndex, a.SubmittedAt)
			if err != nil {
				return written, fmt.Errorf("failed to reconcile ride source for case %s on %s: %w",
					caseID, a.ServiceDate.Format("2006-01-02"), err)
			}
			if outcome == reconcileInserted {
				written++
			}
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
//
// 司機比對不到時這一列完全沒有展開成 ride_sources（見 IngestSubmission 的閘門），所以
// 這裡是從表單既有欄位對應與這筆提交存的原始答案逐欄重新比對寫入，不是更新既有來源；
// 每一格仍透過 reconcileRideSource 判斷，若這台車在該 slot 已有其他資料則進待維護，
// 不會無條件覆蓋。
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

		columns, err := s.formRepo.GetFormColumns(ctx, u.FormID)
		if err != nil {
			return backfilled, dates, fmt.Errorf("failed to get form columns: %w", err)
		}
		for _, col := range columns {
			if col.MappingStatus != "mapped" || col.CaseID == nil || col.LegSeq == nil {
				continue
			}
			value, exists := u.Answers[col.ColumnHeader]
			if !exists {
				continue
			}
			reported, ok := merge.ParseReportedValue(value)
			if !ok {
				continue
			}

			caseID := *col.CaseID
			sched, err := s.caseRepo.GetActiveScheduleForCaseOnDate(ctx, caseID, u.ServiceDate)
			if err != nil {
				return backfilled, dates, fmt.Errorf("failed to load active schedule: %w", err)
			}
			for _, legSeq := range expandLegSeqs(*col.LegSeq, sched) {
				if _, err := s.reconcileRideSource(ctx, u.FormID, u.SubmissionID, caseID, u.ServiceDate, legSeq, u.VehicleID, &driverID, reported, col.ColumnIndex, u.SubmittedAt); err != nil {
					return backfilled, dates, fmt.Errorf("failed to reconcile ride source for case %s on %s: %w",
						caseID, u.ServiceDate.Format("2006-01-02"), err)
				}
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

// ErrRowConflictAlreadyResolved 代表這筆同車同個案衝突已被他人裁決過。
var ErrRowConflictAlreadyResolved = errors.New("row conflict already resolved")

// ListRowConflicts 轉呼叫 repo，供 driverreport 彙整待維護清單。
func (s *RideService) ListRowConflicts(ctx context.Context) ([]RowConflict, error) {
	items, err := s.formRepo.ListPendingRowConflicts(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list row conflicts: %w", err)
	}
	if items == nil {
		items = []RowConflict{}
	}
	return items, nil
}

// DeleteSubmission 移除一筆匯報提交紀錄，供待維護清單的「忽略此筆」使用。
//
// form_submissions 被 ride_sources 與 ride_source_row_conflicts 以 ON DELETE CASCADE 參照，
// 直接刪除會連帶砍掉搭乘來源，而 ride_records 只在寫入路徑重算，會留下對不上任何來源的
// 搭乘紀錄。因此先取得這筆提交展開出的 slot，刪除後逐一重算，讓搭乘紀錄與剩餘來源一致。
// 實務上會出現在待維護清單的是 driver_id IS NULL 的提交，IngestSubmission 對這類提交提早
// 返回、不展開搭乘來源，所以 slot 清單通常是空的；重算路徑是為了不依賴這個假設。
// 交易由呼叫端（DriverReportService）建立，這裡沿用 context 上的同一連線。
func (s *RideService) DeleteSubmission(ctx context.Context, submissionID uuid.UUID) (int64, error) {
	slots, err := s.formRepo.ListRideSourceSlotsForSubmission(ctx, submissionID)
	if err != nil {
		return 0, fmt.Errorf("failed to list ride source slots for submission: %w", err)
	}

	rowsAffected, err := s.formRepo.DeleteSubmission(ctx, submissionID)
	if err != nil {
		return 0, fmt.Errorf("failed to delete submission: %w", err)
	}
	if rowsAffected == 0 {
		return 0, nil
	}

	for _, slot := range slots {
		if err := s.recalculateRideRecord(ctx, slot.CaseID, slot.ServiceDate, slot.LegSeq, slot.VehicleID, nil); err != nil {
			return 0, err
		}
	}
	return rowsAffected, nil
}

// DeleteRowConflict 移除一筆尚未裁決的同車同個案衝突，供待維護清單的「忽略此筆」使用。
// 既有搭乘來源與搭乘紀錄一律不動，維持衝突發生前的值。
func (s *RideService) DeleteRowConflict(ctx context.Context, conflictID uuid.UUID) (int64, error) {
	rowsAffected, err := s.formRepo.DeleteRowConflict(ctx, conflictID)
	if err != nil {
		return 0, fmt.Errorf("failed to delete row conflict: %w", err)
	}
	return rowsAffected, nil
}

// ResolveRowConflict 裁決一筆同車同個案衝突：useNew 時把暫存的新值實際寫入搭乘來源並
// 重算搭乘紀錄，否則單純標記已解決、保留既有資料不動。回傳值供呼叫端在司機有變更時
// 同步出勤月曆，比照初次匯入與司機補綁定的既有流程。
func (s *RideService) ResolveRowConflict(ctx context.Context, conflictID uuid.UUID, useNew bool, operatorID uuid.UUID) (appliedDriverID *uuid.UUID, appliedServiceDate *time.Time, err error) {
	applied, resolved, err := s.formRepo.ResolveRowConflict(ctx, conflictID, useNew, operatorID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to resolve row conflict: %w", err)
	}
	if !resolved {
		return nil, nil, ErrRowConflictAlreadyResolved
	}
	if !useNew || applied == nil {
		return nil, nil, nil
	}

	if err := s.formRepo.InsertRideSource(ctx, applied.NewSubmissionID, applied.CaseID, applied.ServiceDate, applied.LegSeq, applied.VehicleID, applied.NewDriverID, applied.NewReported, applied.SourceColumnIndex, applied.NewSubmittedAt); err != nil {
		return nil, nil, fmt.Errorf("failed to apply resolved row conflict: %w", err)
	}
	if err := s.recalculateRideRecord(ctx, applied.CaseID, applied.ServiceDate, applied.LegSeq, applied.VehicleID, applied.NewDriverID); err != nil {
		return nil, nil, err
	}
	if applied.NewDriverID == nil {
		return nil, nil, nil
	}
	return applied.NewDriverID, &applied.ServiceDate, nil
}

// ListImportedMonths 統計每份匯報表各月份已匯入的提交筆數與最後一次匯入時間。
//
// 月份不另外寫入成欄位，一律由 form_submissions.service_date 推得，避免統計與實際資料不同步。
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
	EffectiveStatus     PatchValue[string]    `json:"effectiveStatus"`
	VehicleID           PatchValue[uuid.UUID] `json:"vehicleId"`
	DriverID            PatchValue[uuid.UUID] `json:"driverId"`
	DepartTimeOverride  PatchValue[string]    `json:"departTimeOverride"`
	DurationMinOverride PatchValue[int16]     `json:"durationMinOverride"`
	NotClaimedAA09      PatchValue[bool]      `json:"notClaimedAa09"`
	Reason              PatchValue[string]    `json:"reason"`
	BasedOnFingerprint  *string               `json:"basedOnFingerprint"`
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
	if req.EffectiveStatus.Present && req.EffectiveStatus.Value == nil {
		return ErrInvalidRideCorrectionField
	}
	if req.VehicleID.Present && req.VehicleID.Value == nil {
		return ErrInvalidRideCorrectionField
	}
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
		if err := s.auditRepo.Write(ctx, AuditEntry{
			ActorID:    &actorID,
			ActorRole:  &actorRole,
			Action:     "correct",
			EntityType: "ride_records",
			EntityID:   &entityIDStr,
			BeforeData: newRideAuditSnapshot(before),
			AfterData:  newRideCorrectionAuditSnapshot(rideID, req),
			IPAddress:  &ip,
			UserAgent:  &ua,
		}); err != nil {
			slog.Error("ride correction audit write failed", "action", "correct", "entity_id", rideID.String(), "error", err)
		}
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

	// 人工補登只能落在有效排班已定義的趟次；即使請求自行指定車輛，也不能
	// 藉此建立不存在於排班的任意 leg。
	if s.caseRepo == nil {
		return nil, ErrInvalidManualRideLeg
	}
	sched, err := s.caseRepo.GetActiveScheduleForCaseOnDate(ctx, req.CaseID, serviceDate)
	if err != nil {
		return nil, fmt.Errorf("failed to load active schedule: %w", err)
	}
	if sched == nil {
		return nil, ErrInvalidManualRideLeg
	}
	if len(sched.Weekdays) > 0 {
		weekday := int16(serviceDate.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		weekdayScheduled := false
		for _, scheduledWeekday := range sched.Weekdays {
			if scheduledWeekday == weekday {
				weekdayScheduled = true
				break
			}
		}
		if !weekdayScheduled {
			return nil, ErrInvalidManualRideLeg
		}
	}
	var scheduledLeg *ScheduleLeg
	for i := range sched.Legs {
		if sched.Legs[i].LegSeq == req.LegSeq {
			scheduledLeg = &sched.Legs[i]
			break
		}
	}
	if scheduledLeg == nil {
		return nil, ErrInvalidManualRideLeg
	}

	var vehicleID uuid.UUID
	if req.VehicleID != nil && *req.VehicleID != uuid.Nil {
		vehicleID = *req.VehicleID
	} else if scheduledLeg.VehicleID != nil {
		vehicleID = *scheduledLeg.VehicleID
	}
	if vehicleID == uuid.Nil {
		return nil, ErrManualRideVehicleRequired
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
		if err := s.auditRepo.Write(ctx, AuditEntry{
			ActorID:    &actorID,
			ActorRole:  &actorRole,
			Action:     "manual_report",
			EntityType: "ride_records",
			EntityID:   &entityIDStr,
			BeforeData: newRideAuditSnapshot(existingRec),
			AfterData:  newRideAuditSnapshot(&rec),
			IPAddress:  &ip,
			UserAgent:  &ua,
		}); err != nil {
			slog.Error("manual ride audit write failed", "action", "manual_report", "entity_id", rec.ID.String(), "error", err)
		}
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
func (s *RideService) ResolveConflict(ctx context.Context, rideID uuid.UUID, req ResolveConflictInput, actorID uuid.UUID, actorRole string, requestMetadata ...string) error {
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
		ip, ua := "", ""
		if len(requestMetadata) > 0 {
			ip = requestMetadata[0]
		}
		if len(requestMetadata) > 1 {
			ua = requestMetadata[1]
		}
		if err := s.auditRepo.Write(ctx, AuditEntry{
			ActorID:    &actorID,
			ActorRole:  &actorRole,
			Action:     "resolve_conflict",
			EntityType: "ride_records",
			EntityID:   &entityIDStr,
			BeforeData: newRideAuditSnapshot(before),
			AfterData: rideConflictResolutionAuditSnapshot{
				ID:          rideID,
				VehicleID:   req.VehicleID,
				DriverID:    req.DriverID,
				HasConflict: false,
			},
			IPAddress: &ip,
			UserAgent: &ua,
		}); err != nil {
			slog.Error("ride conflict audit write failed", "action", "resolve_conflict", "entity_id", rideID.String(), "error", err)
		}
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
	RawPayload  string
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
			RawPayload:  r.RawPayload,
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
