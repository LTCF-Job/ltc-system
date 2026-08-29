package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"ltc-system/apps/api/internal/config"
	"ltc-system/apps/api/internal/domain/crypto"
	"ltc-system/apps/api/internal/domain/namenorm"
	"ltc-system/apps/api/internal/repository"
)

var (
	ErrDuplicateNationalID = errors.New("national ID already registered")
	ErrSiteRegionMismatch  = errors.New("site region does not match case region")
	ErrInvalidTripPattern  = errors.New("trip pattern must match schedule legs count")
	ErrLegTimesNotOrdered  = errors.New("schedule leg departure times must be strictly increasing")
)

// MasterService 封裝個案、據點、車輛、司機與排班之業務邏輯。
type MasterService struct {
	cfg         *config.Config
	caseRepo    *repository.CaseRepository
	siteRepo    *repository.SiteRepository
	vehicleRepo *repository.VehicleRepository
	driverRepo  *repository.DriverRepository
	auditRepo   *repository.AuditRepository
}

// NewMasterService 建立 MasterService 實例。
func NewMasterService(
	cfg *config.Config,
	caseRepo *repository.CaseRepository,
	siteRepo *repository.SiteRepository,
	vehicleRepo *repository.VehicleRepository,
	driverRepo *repository.DriverRepository,
	auditRepo *repository.AuditRepository,
) *MasterService {
	return &MasterService{
		cfg:         cfg,
		caseRepo:    caseRepo,
		siteRepo:    siteRepo,
		vehicleRepo: vehicleRepo,
		driverRepo:  driverRepo,
		auditRepo:   auditRepo,
	}
}

// CreateCaseRequest 代表新增個案之請求參數。
type CreateCaseRequest struct {
	Code              string
	Name              string
	NationalID        string
	HouseholdType     *string
	Gender            *string
	BirthDate         *time.Time
	CareContactRole   *string
	CareContactName   *string
	RegisteredAddress *string
	HomeAddress       string
	Region            string
	LTCLevel          *string
	ServiceCategory   int
	ServiceUsageType  int
	ClaimStartDate    time.Time
	ClaimEndDate      *time.Time
	Status            string
}

// CreateCase 建立個案主檔，執行身分證查重、加密與雜湊產生。
func (s *MasterService) CreateCase(ctx context.Context, req CreateCaseRequest, actorID uuid.UUID, actorRole, ip, ua string) (*repository.CaseEntity, error) {
	req.NationalID = strings.TrimSpace(strings.ToUpper(req.NationalID))
	if !crypto.ValidateNationalID(req.NationalID) {
		return nil, errors.New("invalid national ID format")
	}

	normName := namenorm.Normalize(req.Name)
	hmacIdx := crypto.Index(req.NationalID, s.cfg.HMACKey)

	existing, _ := s.caseRepo.GetByHMAC(ctx, hmacIdx)
	if existing != nil {
		return nil, ErrDuplicateNationalID
	}

	cipherText, err := crypto.Encrypt(req.NationalID, s.cfg.EncryptionKey)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt national id: %w", err)
	}

	maskedID := crypto.Mask(req.NationalID)
	if req.Status == "" {
		req.Status = "active"
	}
	if req.ServiceCategory == 0 {
		req.ServiceCategory = 1
	}
	if req.ServiceUsageType == 0 {
		req.ServiceUsageType = 2
	}

	entity := repository.CaseEntity{
		Code:              req.Code,
		Name:              req.Name,
		NameNormalized:    normName,
		NationalIDCipher:  cipherText,
		NationalIDHMAC:    hmacIdx,
		NationalIDMasked:  maskedID,
		HouseholdType:     req.HouseholdType,
		Gender:            req.Gender,
		BirthDate:         req.BirthDate,
		CareContactRole:   req.CareContactRole,
		CareContactName:   req.CareContactName,
		RegisteredAddress: req.RegisteredAddress,
		HomeAddress:       req.HomeAddress,
		Region:            req.Region,
		LTCLevel:          req.LTCLevel,
		ServiceCategory:   req.ServiceCategory,
		ServiceUsageType:  req.ServiceUsageType,
		ClaimStartDate:    req.ClaimStartDate,
		ClaimEndDate:      req.ClaimEndDate,
		Status:            req.Status,
	}

	if err := s.caseRepo.Create(ctx, &entity); err != nil {
		return nil, fmt.Errorf("failed to create case: %w", err)
	}

	if s.auditRepo != nil {
		entityIDStr := entity.ID.String()
		_ = s.auditRepo.Insert(ctx, &repository.AuditLogEntity{
			ActorID:    &actorID,
			ActorRole:  &actorRole,
			Action:     "create",
			EntityType: "cases",
			EntityID:   &entityIDStr,
			AfterData:  entity,
			IPAddress:  &ip,
			UserAgent:  &ua,
		})
	}

	return &entity, nil
}

// ListCases 查詢個案清單（回傳遮罩身分證）。
func (s *MasterService) ListCases(ctx context.Context, region, status, q string, page, pageSize int) ([]repository.CaseEntity, int64, error) {
	return s.caseRepo.List(ctx, region, status, q, page, pageSize)
}

// GetCaseByID 取得單筆個案主檔明細。
func (s *MasterService) GetCaseByID(ctx context.Context, id uuid.UUID) (*repository.CaseEntity, error) {
	return s.caseRepo.GetByID(ctx, id)
}

// UpdateCaseInput 代表更新個案主檔所需之輸入，欄位為 nil 表示不變更。
type UpdateCaseInput struct {
	Name              *string
	HomeAddress       *string
	Region            *string
	LTCLevel          *string
	ServiceCategory   *int
	ServiceUsageType  *int
	Status            *string
	HouseholdType     *string
	Gender            *string
	BirthDate         *time.Time
	CareContactRole   *string
	CareContactName   *string
	RegisteredAddress *string
}

// UpdateCase 更新個案主檔資料，僅套用有提供的欄位。
func (s *MasterService) UpdateCase(ctx context.Context, id uuid.UUID, in UpdateCaseInput) (*repository.CaseEntity, error) {
	entity, err := s.caseRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if in.Name != nil {
		entity.Name = *in.Name
	}
	if in.HomeAddress != nil {
		entity.HomeAddress = *in.HomeAddress
	}
	if in.Region != nil {
		entity.Region = *in.Region
	}
	if in.LTCLevel != nil {
		entity.LTCLevel = in.LTCLevel
	}
	if in.ServiceCategory != nil {
		entity.ServiceCategory = *in.ServiceCategory
	}
	if in.ServiceUsageType != nil {
		entity.ServiceUsageType = *in.ServiceUsageType
	}
	if in.Status != nil {
		entity.Status = *in.Status
	}
	if in.HouseholdType != nil {
		entity.HouseholdType = in.HouseholdType
	}
	if in.Gender != nil {
		entity.Gender = in.Gender
	}
	if in.BirthDate != nil {
		entity.BirthDate = in.BirthDate
	}
	if in.CareContactRole != nil {
		entity.CareContactRole = in.CareContactRole
	}
	if in.CareContactName != nil {
		entity.CareContactName = in.CareContactName
	}
	if in.RegisteredAddress != nil {
		entity.RegisteredAddress = in.RegisteredAddress
	}

	if err := s.caseRepo.Update(ctx, entity); err != nil {
		return nil, err
	}
	return entity, nil
}

// UpdateCaseTransportPreference 更新個案的交通偏好（所屬據點與去回程車輛），回傳更新後的個案主檔。
func (s *MasterService) UpdateCaseTransportPreference(ctx context.Context, caseID, siteID, outboundVehicleID, inboundVehicleID uuid.UUID) (*repository.CaseEntity, error) {
	if err := s.caseRepo.UpsertTransportPreference(ctx, caseID, siteID, outboundVehicleID, inboundVehicleID); err != nil {
		return nil, err
	}
	return s.caseRepo.GetByID(ctx, caseID)
}

// GetActiveScheduleForCaseOnDate 取得個案於指定日期生效之排班；查無資料時回傳 nil、nil，
// 底層查詢失敗時回傳 error（呼叫端不應將兩者混為一談）。
func (s *MasterService) GetActiveScheduleForCaseOnDate(ctx context.Context, caseID uuid.UUID, serviceDate time.Time) (*repository.CaseScheduleEntity, error) {
	return s.caseRepo.GetActiveScheduleForCaseOnDate(ctx, caseID, serviceDate)
}

// RevealCaseNationalID 解密個案身分證並留存稽核日誌。
func (s *MasterService) RevealCaseNationalID(ctx context.Context, caseID uuid.UUID, actorID uuid.UUID, actorRole, ip, ua string) (string, error) {
	caseEntity, err := s.caseRepo.GetByID(ctx, caseID)
	if err != nil {
		return "", err
	}

	plainID, err := crypto.Decrypt(caseEntity.NationalIDCipher, s.cfg.EncryptionKey)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt national id: %w", err)
	}

	if s.auditRepo != nil {
		entityIDStr := caseID.String()
		_ = s.auditRepo.Insert(ctx, &repository.AuditLogEntity{
			ActorID:    &actorID,
			ActorRole:  &actorRole,
			Action:     "reveal_pii",
			EntityType: "cases",
			EntityID:   &entityIDStr,
			IPAddress:  &ip,
			UserAgent:  &ua,
		})
	}

	return plainID, nil
}

// RecordSkippedCaseImport 保留未寫入的來源列，讓操作人員能回查補正原因與原始欄位。
func (s *MasterService) RecordSkippedCaseImport(ctx context.Context, item CaseImportSkippedRow, actorID uuid.UUID, actorRole, ip, ua string) {
	if s.auditRepo == nil {
		return
	}
	entityID := fmt.Sprintf("row-%d", item.RowIndex)
	_ = s.auditRepo.Insert(ctx, &repository.AuditLogEntity{
		ActorID: &actorID, ActorRole: &actorRole, Action: "import_skip", EntityType: "case_import", EntityID: &entityID,
		AfterData: item, IPAddress: &ip, UserAgent: &ua,
	})
}

// CreateScheduleRequest 代表建立個案排班設定之請求參數。
type CreateScheduleRequest struct {
	CaseID             uuid.UUID
	SiteID             uuid.UUID
	EffectiveFrom      time.Time
	EffectiveTo        *time.Time
	Weekdays           []int16
	TripPattern        int16
	UnitPrice          float64
	DistanceKM         float64
	ServiceDurationMin int16
	ServiceCode        string
	Note               *string
	Legs               []CreateScheduleLegItemRequest
}

// CreateScheduleLegItemRequest 代表排班單趟設定之請求參數。
type CreateScheduleLegItemRequest struct {
	LegSeq     int16
	Direction  string
	DepartTime string
	VehicleID  *uuid.UUID
}

// CreateCaseSchedule 建立個案之有效排班設定並校驗趟次時段與遞增順序。
func (s *MasterService) CreateCaseSchedule(ctx context.Context, req CreateScheduleRequest) (*repository.CaseScheduleEntity, error) {
	if int(req.TripPattern) != len(req.Legs) {
		return nil, ErrInvalidTripPattern
	}

	caseObj, err := s.caseRepo.GetByID(ctx, req.CaseID)
	if err != nil {
		return nil, fmt.Errorf("case not found: %w", err)
	}

	siteObj, err := s.siteRepo.GetByID(ctx, req.SiteID)
	if err != nil {
		return nil, fmt.Errorf("site not found: %w", err)
	}

	if caseObj.Region != siteObj.Region {
		return nil, ErrSiteRegionMismatch
	}

	var legs []repository.ScheduleLegEntity
	var lastTime string
	for _, l := range req.Legs {
		if lastTime != "" && l.DepartTime <= lastTime {
			return nil, ErrLegTimesNotOrdered
		}
		lastTime = l.DepartTime

		period := "am"
		if l.DepartTime >= "12:00" {
			period = "pm"
		}

		legs = append(legs, repository.ScheduleLegEntity{
			LegSeq:     l.LegSeq,
			Direction:  l.Direction,
			Period:     period,
			DepartTime: l.DepartTime,
			RunNo:      1,
			VehicleID:  l.VehicleID,
		})
	}

	entity := repository.CaseScheduleEntity{
		CaseID:             req.CaseID,
		SiteID:             req.SiteID,
		EffectiveFrom:      req.EffectiveFrom,
		EffectiveTo:        req.EffectiveTo,
		Weekdays:           req.Weekdays,
		TripPattern:        req.TripPattern,
		UnitPrice:          req.UnitPrice,
		DistanceKM:         req.DistanceKM,
		ServiceDurationMin: req.ServiceDurationMin,
		ServiceCode:        req.ServiceCode,
		Note:               req.Note,
		Legs:               legs,
	}

	if err := s.caseRepo.CreateSchedule(ctx, &entity); err != nil {
		return nil, fmt.Errorf("failed to save case schedule: %w", err)
	}

	return &entity, nil
}
