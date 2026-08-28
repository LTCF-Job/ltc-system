package service

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
	"ltc-system/apps/api/internal/repository"
)

// ErrInvalidIngestToken 代表 Webhook 呼叫端提供的 ingest token 無效或不存在。
var ErrInvalidIngestToken = errors.New("invalid ingest token")

// RideService 封裝 Google 表單回報解析、正規化、混車合併、衝突裁決與更正。
type RideService struct {
	formRepo    *repository.FormRepository
	driverRepo  *repository.DriverRepository
	caseRepo    *repository.CaseRepository
	vehicleRepo *repository.VehicleRepository
	auditRepo   *repository.AuditRepository
}

// NewRideService 建立 RideService 實例。
func NewRideService(
	formRepo *repository.FormRepository,
	driverRepo *repository.DriverRepository,
	caseRepo *repository.CaseRepository,
	vehicleRepo *repository.VehicleRepository,
	auditRepo *repository.AuditRepository,
) *RideService {
	return &RideService{
		formRepo:    formRepo,
		driverRepo:  driverRepo,
		caseRepo:    caseRepo,
		vehicleRepo: vehicleRepo,
		auditRepo:   auditRepo,
	}
}

// ProcessFormWebhookRequest 代表 Google Form 提交的 Webhook Payload。
type ProcessFormWebhookRequest struct {
	Timestamp   string                 `json:"timestamp"`
	ServiceDate string                 `json:"serviceDate"`
	DriverRaw   string                 `json:"driverRaw"`
	IssueText   string                 `json:"issueText"`
	Answers     map[string]interface{} `json:"answers"`
}

// IngestWebhook 處理 Google 表單回報並執行欄位正規化、四趟展開與混車合併。
func (s *RideService) IngestWebhook(ctx context.Context, secret string, req ProcessFormWebhookRequest) error {
	formID, defaultVehicleID, err := s.formRepo.GetFormBySecret(ctx, secret)
	if err != nil {
		return ErrInvalidIngestToken
	}

	// 依規格書 5.2，若服務日期為空（如總計列或標頭）直接跳過處理
	req.ServiceDate = strings.TrimSpace(req.ServiceDate)
	if req.ServiceDate == "" {
		slog.Warn("Skip submission because serviceDate is empty (likely summary row)")
		return nil
	}

	serviceDate, err := time.Parse("2006-01-02", req.ServiceDate)
	if err != nil {
		// 支援斜線日期格式（YYYY/MM/DD、YYYY/M/D）
		serviceDate, err = time.Parse("2006/01/02", req.ServiceDate)
		if err != nil {
			serviceDate, err = time.Parse("2006/1/2", req.ServiceDate)
			if err != nil {
				return fmt.Errorf("invalid service date format: %s", req.ServiceDate)
			}
		}
	}

	submittedAt := time.Now().UTC()
	if req.Timestamp != "" {
		if t, err := time.Parse(time.RFC3339, req.Timestamp); err == nil {
			submittedAt = t
		}
	}

	var driverID *uuid.UUID
	req.DriverRaw = strings.TrimSpace(req.DriverRaw)
	if req.DriverRaw != "" {
		driverNorm := namenorm.Normalize(req.DriverRaw)
		if d, _ := s.driverRepo.GetByNameNormalized(ctx, driverNorm); d != nil {
			driverID = &d.ID
		}
	}

	rawPayload := map[string]interface{}{
		"timestamp":   req.Timestamp,
		"serviceDate": req.ServiceDate,
		"driverRaw":   req.DriverRaw,
		"issueText":   req.IssueText,
		"answers":     req.Answers,
	}

	submissionID, err := s.formRepo.SaveFormSubmission(
		ctx, formID, serviceDate, submittedAt, req.DriverRaw, driverID, "webhook", rawPayload, req.IssueText,
	)
	if err != nil {
		return fmt.Errorf("failed to save form submission: %w", err)
	}

	columns, err := s.formRepo.GetFormColumns(ctx, formID)
	if err != nil {
		return fmt.Errorf("failed to get form columns: %w", err)
	}

	for _, col := range columns {
		if col.MappingStatus != "mapped" || col.CaseID == nil || col.LegSeq == nil {
			continue
		}

		colKey := fmt.Sprintf("%d", col.ColumnIndex)
		valRaw, exists := req.Answers[colKey]
		if !exists {
			valRaw = req.Answers[col.ColumnHeader]
		}
		if valRaw == nil {
			continue
		}

		valStr := strings.TrimSpace(fmt.Sprintf("%v", valRaw))
		var reported string
		if strings.Contains(valStr, "有坐") {
			reported = "boarded"
		} else if strings.Contains(valStr, "沒坐") {
			reported = "absent"
		} else {
			// 非明確搭乘標記不建立來源紀錄
			continue
		}

		caseID := *col.CaseID
		baseLegSeq := *col.LegSeq

		sched, _ := s.caseRepo.GetActiveScheduleForCaseOnDate(ctx, caseID, serviceDate)
		var targetLegSeqs []int16

		// 四趟展開規則（R4 / §5.5）：表單第 1 趟展開為 1、3 趟；第 2 趟展開為 2、4 趟
		if sched != nil && sched.TripPattern == 4 {
			if baseLegSeq == 1 {
				targetLegSeqs = []int16{1, 3}
			} else if baseLegSeq == 2 {
				targetLegSeqs = []int16{2, 4}
			} else {
				targetLegSeqs = []int16{baseLegSeq}
			}
		} else {
			targetLegSeqs = []int16{baseLegSeq}
		}

		for _, legSeq := range targetLegSeqs {
			_ = s.formRepo.InsertRideSource(
				ctx, submissionID, caseID, serviceDate, legSeq, defaultVehicleID, driverID, reported, col.ColumnIndex,
			)

			s.recalculateRideRecord(ctx, caseID, serviceDate, legSeq, defaultVehicleID, driverID)
		}
	}

	return nil
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

	// 查詢當日排班設定預設車輛與主要司機
	sched, _ := s.caseRepo.GetActiveScheduleForCaseOnDate(ctx, caseID, serviceDate)
	if sched != nil {
		for _, l := range sched.Legs {
			if l.LegSeq == legSeq && l.VehicleID != nil {
				defaultVehicleID = *l.VehicleID
				if primaryDriver, _ := s.driverRepo.GetPrimaryDriverForVehicleOnDate(ctx, defaultVehicleID, serviceDate); primaryDriver != nil {
					defaultDriverID = &primaryDriver.ID
				}
				break
			}
		}
	}

	// 載入該 slot 的全部 sources
	sources := []merge.RideSourceInput{
		{
			VehicleID:   defaultVehicleID,
			DriverID:    defaultDriverID,
			Reported:    "boarded",
			SubmittedAt: time.Now().UTC(),
		},
	}

	result := merge.MergeRideSources(sources, existingState, defaultVehicleID, defaultDriverID)

	rec := repository.RideRecordEntity{
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
		_ = s.auditRepo.Insert(ctx, &repository.AuditLogEntity{
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
) (*repository.RideRecordEntity, error) {
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

	rec := repository.RideRecordEntity{
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
		_ = s.auditRepo.Insert(ctx, &repository.AuditLogEntity{
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
