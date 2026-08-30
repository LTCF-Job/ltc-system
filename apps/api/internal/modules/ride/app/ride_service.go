package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"ltc-system/apps/api/internal/domain/merge"
	"ltc-system/apps/api/internal/domain/namenorm"
)

// RideService 封裝司機接送匯報的展開、正規化、混車合併、衝突裁決與更正。
type RideService struct {
	formRepo   RideRecordStore
	driverRepo DriverResolver
	caseRepo   ScheduleReader
	auditRepo  AuditWriter
}

// NewRideService 建立 RideService 實例。
func NewRideService(
	formRepo RideRecordStore,
	driverRepo DriverResolver,
	caseRepo ScheduleReader,
	auditRepo AuditWriter,
) *RideService {
	return &RideService{
		formRepo:   formRepo,
		driverRepo: driverRepo,
		caseRepo:   caseRepo,
		auditRepo:  auditRepo,
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
		submittedAt = time.Now().UTC()
	}

	driverID := req.DriverID
	req.DriverRaw = strings.TrimSpace(req.DriverRaw)
	if driverID == nil && req.DriverRaw != "" {
		if d, _ := s.driverRepo.GetByNameNormalized(ctx, namenorm.Normalize(req.DriverRaw)); d != nil {
			driverID = &d.ID
		}
	}

	rawPayload := map[string]interface{}{
		"serviceDate": req.ServiceDate.Format("2006-01-02"),
		"driverRaw":   req.DriverRaw,
		"remark":      req.Remark,
		"answers":     req.Answers,
	}

	submissionID, err := s.formRepo.SaveFormSubmission(
		ctx, formID, req.ServiceDate, submittedAt, req.DriverRaw, driverID, "import", rawPayload, req.Remark,
	)
	if err != nil {
		return 0, fmt.Errorf("failed to save form submission: %w", err)
	}

	columns, err := s.formRepo.GetFormColumns(ctx, formID)
	if err != nil {
		return 0, fmt.Errorf("failed to get form columns: %w", err)
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
		sched, _ := s.caseRepo.GetActiveScheduleForCaseOnDate(ctx, caseID, req.ServiceDate)

		for _, legSeq := range expandLegSeqs(*col.LegSeq, sched) {
			if err := s.formRepo.InsertRideSource(
				ctx, submissionID, caseID, req.ServiceDate, legSeq, defaultVehicleID, driverID, reported, col.ColumnIndex,
			); err != nil {
				slog.Warn("Failed to insert ride source",
					slog.String("caseId", caseID.String()),
					slog.String("serviceDate", req.ServiceDate.Format("2006-01-02")),
					slog.String("error", err.Error()))
				continue
			}

			s.recalculateRideRecord(ctx, caseID, req.ServiceDate, legSeq, defaultVehicleID, driverID)
			written++
		}
	}

	return written, nil
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
func (s *RideService) recalculateRideRecord(
	ctx context.Context,
	caseID uuid.UUID,
	serviceDate time.Time,
	legSeq int16,
	defaultVehicleID uuid.UUID,
	defaultDriverID *uuid.UUID,
) {
	// 查詢既有紀錄以保護人工裁決與更正
	existingRec, _ := s.formRepo.GetRideRecordForSlot(ctx, caseID, serviceDate, legSeq)

	var existingState *merge.ExistingRecordState
	if existingRec != nil {
		existingState = &merge.ExistingRecordState{
			HasConflict:        existingRec.HasConflict,
			ConflictResolvedAt: existingRec.ConflictResolvedAt,
			ResolvedVehicleID:  &existingRec.VehicleID,
			ResolvedDriverID:   existingRec.DriverID,
			CorrectedAt:        existingRec.CorrectedAt,
			CorrectedBy:        existingRec.CorrectedBy,
			EffectiveStatus:    existingRec.EffectiveStatus,
			CorrectedVehicle:   &existingRec.VehicleID,
			CorrectedDriver:    existingRec.DriverID,
		}
	}

	// 查詢當日排班設定預設車輛與司機
	sched, _ := s.caseRepo.GetActiveScheduleForCaseOnDate(ctx, caseID, serviceDate)
	if sched != nil {
		for _, l := range sched.Legs {
			if l.LegSeq == legSeq && l.VehicleID != nil {
				defaultVehicleID = *l.VehicleID
				// 一台車當日可能有多位司機，無從判斷是誰出車時留空由人工指定
				if drivers, _ := s.driverRepo.ListDriversForVehicleOnDate(ctx, defaultVehicleID, serviceDate); len(drivers) == 1 {
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
		slog.Warn("Failed to load ride sources for slot",
			slog.String("caseId", caseID.String()),
			slog.String("serviceDate", serviceDate.Format("2006-01-02")),
			slog.String("error", err.Error()))
		return
	}
	if len(rows) == 0 {
		return
	}

	sources := make([]merge.RideSourceInput, 0, len(rows))
	for _, row := range rows {
		sources = append(sources, merge.RideSourceInput{
			VehicleID:   row.VehicleID,
			DriverID:    row.DriverID,
			Reported:    row.Reported,
			SubmittedAt: row.SubmittedAt,
		})
	}

	result := merge.MergeRideSources(sources, existingState, defaultVehicleID, defaultDriverID)

	rec := RideRecord{
		CaseID:          caseID,
		ServiceDate:     serviceDate,
		LegSeq:          legSeq,
		MergedStatus:    result.MergedStatus,
		EffectiveStatus: result.EffectiveStatus,
		VehicleID:       result.SelectedVehicle,
		DriverID:        result.SelectedDriver,
		HasConflict:     result.HasConflict,
	}
	if existingRec != nil {
		rec.ID = existingRec.ID
		rec.ConflictResolvedAt = existingRec.ConflictResolvedAt
		rec.ConflictResolvedBy = existingRec.ConflictResolvedBy
		rec.CorrectedAt = existingRec.CorrectedAt
		rec.CorrectedBy = existingRec.CorrectedBy
		rec.CorrectionReason = existingRec.CorrectionReason
		rec.NotClaimedAA09 = existingRec.NotClaimedAA09
	}

	_ = s.formRepo.UpsertRideRecord(ctx, &rec)
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
	err := s.formRepo.CorrectRideRecord(
		ctx, rideID, req.EffectiveStatus, req.VehicleID, req.DriverID,
		req.DepartTimeOverride, req.DurationMinOverride, req.NotClaimedAA09, req.Reason, actorID,
	)
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

	serviceDate, err := time.Parse("2006-01-02", req.ServiceDate)
	if err != nil {
		return nil, fmt.Errorf("無效的服務日期格式：%s", req.ServiceDate)
	}

	// 車輛未指定時由排班回退取得預設車輛
	var vehicleID uuid.UUID
	if req.VehicleID != nil && *req.VehicleID != uuid.Nil {
		vehicleID = *req.VehicleID
	} else if s.caseRepo != nil {
		sched, _ := s.caseRepo.GetActiveScheduleForCaseOnDate(ctx, req.CaseID, serviceDate)
		if sched != nil {
			for _, l := range sched.Legs {
				if l.LegSeq == req.LegSeq && l.VehicleID != nil {
					vehicleID = *l.VehicleID
					break
				}
			}
		}
	}

	existingRec, _ := s.formRepo.GetRideRecordForSlot(ctx, req.CaseID, serviceDate, req.LegSeq)
	now := time.Now().UTC()

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
