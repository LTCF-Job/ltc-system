package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"ltc-system/apps/api/internal/domain/merge"
	"ltc-system/apps/api/internal/domain/namenorm"
	"ltc-system/apps/api/internal/middleware"
	"ltc-system/apps/api/internal/repository"
)

// RideService 封裝 Google 表單回報解析、正規化、混車合併、衝突裁決與更正。
type RideService struct {
	db          *pgxpool.Pool
	formRepo    *repository.FormRepository
	driverRepo  *repository.DriverRepository
	caseRepo    *repository.CaseRepository
	vehicleRepo *repository.VehicleRepository
}

// NewRideService 建立 RideService 實例。
func NewRideService(
	db *pgxpool.Pool,
	formRepo *repository.FormRepository,
	driverRepo *repository.DriverRepository,
	caseRepo *repository.CaseRepository,
	vehicleRepo *repository.VehicleRepository,
) *RideService {
	return &RideService{
		db:          db,
		formRepo:    formRepo,
		driverRepo:  driverRepo,
		caseRepo:    caseRepo,
		vehicleRepo: vehicleRepo,
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

// IngestWebhook 處理 Google 表單單筆提交：落原始資料 -> 欄位解析 -> 四趟展開 -> 混車合併。
func (s *RideService) IngestWebhook(ctx context.Context, secret string, req ProcessFormWebhookRequest) error {
	if s.db == nil {
		return errors.New("database unavailable")
	}

	formID, defaultVehicleID, err := s.formRepo.GetFormBySecret(ctx, secret)
	if err != nil {
		return middleware.ErrInvalidToken
	}

	// 1. 檢核今天日期，若為空則依規格書 5.2 整列跳過
	req.ServiceDate = strings.TrimSpace(req.ServiceDate)
	if req.ServiceDate == "" {
		slog.Warn("Skip submission because serviceDate is empty (likely summary row)")
		return nil
	}

	serviceDate, err := time.Parse("2006-01-02", req.ServiceDate)
	if err != nil {
		// 嘗試常見斜線格式 2026/07/01 或 2026/7/1
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

	// 2. 司機姓名比對（§5.6）
	var driverID *uuid.UUID
	req.DriverRaw = strings.TrimSpace(req.DriverRaw)
	if req.DriverRaw != "" {
		driverNorm := namenorm.Normalize(req.DriverRaw)
		if d, _ := s.driverRepo.GetByNameNormalized(ctx, driverNorm); d != nil {
			driverID = &d.ID
		}
	}

	// 3. 先落 form_submissions.payload 原始資料（原則 B-2）
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

	// 4. 取得該表單已對應的欄位定義
	columns, err := s.formRepo.GetFormColumns(ctx, formID)
	if err != nil {
		return fmt.Errorf("failed to get form columns: %w", err)
	}

	// 5. 逐欄解析有綁定之個案與時段
	for _, col := range columns {
		if col.MappingStatus != "mapped" || col.CaseID == nil || col.LegSeq == nil {
			continue
		}

		// 從 answers 取得儲存格值
		colKey := fmt.Sprintf("%d", col.ColumnIndex)
		valRaw, exists := req.Answers[colKey]
		if !exists {
			// 若用標題當 key 亦可相容
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
			// 其它自由文字或空白依 §5.3 不產生來源紀錄
			continue
		}

		caseID := *col.CaseID
		baseLegSeq := *col.LegSeq

		// 檢查該個案在當日之排班設定以確認 tripPattern
		sched, _ := s.caseRepo.GetActiveScheduleForCaseOnDate(ctx, caseID, serviceDate)
		var targetLegSeqs []int16

		if sched != nil && sched.TripPattern == 4 {
			// 四趟展開規則（R4 / §5.5）：outbound 展開為 1, 3；inbound 展開為 2, 4
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
			// 寫入 ride_sources
			_ = s.formRepo.InsertRideSource(
				ctx, submissionID, caseID, serviceDate, legSeq, defaultVehicleID, driverID, reported, col.ColumnIndex,
			)

			// 執行混車合併
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

// CorrectRideRecord 人工更正搭乘紀錄並寫入稽核留痕（§4.7）。
func (s *RideService) CorrectRideRecord(
	ctx context.Context,
	rideID uuid.UUID,
	req CorrectRideRecordRequest,
	actorID uuid.UUID,
	actorRole, ip, ua string,
) error {
	if s.db == nil {
		return errors.New("database unavailable")
	}

	err := s.formRepo.CorrectRideRecord(
		ctx, rideID, req.EffectiveStatus, req.VehicleID, req.DriverID,
		req.DepartTimeOverride, req.DurationMinOverride, req.NotClaimedAA09, req.Reason, actorID,
	)
	if err != nil {
		return fmt.Errorf("failed to correct ride record: %w", err)
	}

	_ = middleware.RecordAuditLog(ctx, s.db, middleware.AuditLogEntry{
		ActorID:    actorID,
		ActorRole:  actorRole,
		Action:     "correct",
		EntityType: "ride_records",
		EntityID:   rideID.String(),
		AfterData:  req,
		IPAddress:  ip,
		UserAgent:  ua,
	})

	return nil
}
