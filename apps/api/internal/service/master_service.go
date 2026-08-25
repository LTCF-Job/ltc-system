package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"ltc-system/apps/api/internal/config"
	"ltc-system/apps/api/internal/domain/crypto"
	"ltc-system/apps/api/internal/domain/namenorm"
	"ltc-system/apps/api/internal/middleware"
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
	db          *pgxpool.Pool
	caseRepo    *repository.CaseRepository
	siteRepo    *repository.SiteRepository
	vehicleRepo *repository.VehicleRepository
	driverRepo  *repository.DriverRepository
}

// NewMasterService 建立 MasterService 實例。
func NewMasterService(
	cfg *config.Config,
	db *pgxpool.Pool,
	caseRepo *repository.CaseRepository,
	siteRepo *repository.SiteRepository,
	vehicleRepo *repository.VehicleRepository,
	driverRepo *repository.DriverRepository,
) *MasterService {
	return &MasterService{
		cfg:         cfg,
		db:          db,
		caseRepo:    caseRepo,
		siteRepo:    siteRepo,
		vehicleRepo: vehicleRepo,
		driverRepo:  driverRepo,
	}
}

// CreateCaseRequest 代表新增個案之請求參數。
type CreateCaseRequest struct {
	Code             string     `json:"code"`
	Name             string     `json:"name" binding:"required"`
	NationalID       string     `json:"nationalId" binding:"required"`
	HomeAddress      string     `json:"homeAddress" binding:"required"`
	Region           string     `json:"region" binding:"required"`
	LTCLevel         *string    `json:"ltcLevel"`
	ServiceCategory  int        `json:"serviceCategory"`
	ServiceUsageType int        `json:"serviceUsageType"`
	ClaimStartDate   time.Time  `json:"claimStartDate" binding:"required"`
	ClaimEndDate     *time.Time `json:"claimEndDate"`
	Status           string     `json:"status"`
}

// CreateCase 驗證並建立新個案，自動進行身分證欄位加密、HMAC 索引與遮罩。
func (s *MasterService) CreateCase(ctx context.Context, req CreateCaseRequest, actorID uuid.UUID, actorRole, ip, ua string) (*repository.CaseEntity, error) {
	req.NationalID = strings.ToUpper(strings.TrimSpace(req.NationalID))
	if !crypto.ValidateNationalID(req.NationalID) {
		return nil, crypto.ErrInvalidNationalID
	}

	// 檢查身分證是否重複
	hmacIdx := crypto.Index(req.NationalID, s.cfg.HMACKey)
	if s.db != nil {
		existing, _ := s.caseRepo.GetByHMAC(ctx, hmacIdx)
		if existing != nil {
			return nil, ErrDuplicateNationalID
		}
	}

	cipherText, err := crypto.Encrypt(req.NationalID, s.cfg.EncryptionKey)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt national id: %w", err)
	}

	if req.ServiceCategory == 0 {
		req.ServiceCategory = 1
	}
	if req.ServiceUsageType == 0 {
		req.ServiceUsageType = 2
	}
	if req.Status == "" {
		req.Status = "active"
	}
	if req.Code == "" {
		req.Code = fmt.Sprintf("C%d", time.Now().UnixNano()%10000)
	}

	entity := repository.CaseEntity{
		ID:               uuid.New(),
		Code:             req.Code,
		Name:             req.Name,
		NameNormalized:   namenorm.Normalize(req.Name),
		NationalIDCipher: cipherText,
		NationalIDHMAC:   hmacIdx,
		NationalIDMasked: crypto.Mask(req.NationalID),
		HomeAddress:      req.HomeAddress,
		Region:           req.Region,
		LTCLevel:         req.LTCLevel,
		ServiceCategory:  req.ServiceCategory,
		ServiceUsageType: req.ServiceUsageType,
		ClaimStartDate:   req.ClaimStartDate,
		ClaimEndDate:     req.ClaimEndDate,
		Status:           req.Status,
	}

	if s.db != nil {
		if err := s.caseRepo.Create(ctx, &entity); err != nil {
			return nil, fmt.Errorf("failed to create case: %w", err)
		}
		_ = middleware.RecordAuditLog(ctx, s.db, middleware.AuditLogEntry{
			ActorID:    actorID,
			ActorRole:  actorRole,
			Action:     "create",
			EntityType: "cases",
			EntityID:   entity.ID.String(),
			AfterData:  entity,
			IPAddress:  ip,
			UserAgent:  ua,
		})
	}

	return &entity, nil
}

// RevealCaseNationalID 解密個案身分證並留存稽核日誌。
func (s *MasterService) RevealCaseNationalID(ctx context.Context, caseID uuid.UUID, actorID uuid.UUID, actorRole, ip, ua string) (string, error) {
	if s.db == nil {
		return "", errors.New("database unavailable")
	}

	caseEntity, err := s.caseRepo.GetByID(ctx, caseID)
	if err != nil {
		return "", err
	}

	plainID, err := crypto.Decrypt(caseEntity.NationalIDCipher, s.cfg.EncryptionKey)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt national id: %w", err)
	}

	_ = middleware.RecordAuditLog(ctx, s.db, middleware.AuditLogEntry{
		ActorID:    actorID,
		ActorRole:  actorRole,
		Action:     "reveal_pii",
		EntityType: "cases",
		EntityID:   caseID.String(),
		IPAddress:  ip,
		UserAgent:  ua,
	})

	return plainID, nil
}

// CreateScheduleRequest 代表建立個案排班設定之請求參數。
type CreateScheduleRequest struct {
	CaseID             uuid.UUID                      `json:"caseId" binding:"required"`
	SiteID             uuid.UUID                      `json:"siteId" binding:"required"`
	EffectiveFrom      time.Time                      `json:"effectiveFrom" binding:"required"`
	EffectiveTo        *time.Time                     `json:"effectiveTo"`
	Weekdays           []int16                        `json:"weekdays" binding:"required"`
	TripPattern        int16                          `json:"tripPattern" binding:"required"`
	UnitPrice          float64                        `json:"unitPrice"`
	DistanceKM         float64                        `json:"distanceKm" binding:"required"`
	ServiceDurationMin int16                          `json:"serviceDurationMin"`
	ServiceCode        string                         `json:"serviceCode"`
	Note               *string                        `json:"note"`
	Legs               []CreateScheduleLegRequest     `json:"legs" binding:"required"`
}

// CreateScheduleLegRequest 代表排班各時段時序參數。
type CreateScheduleLegRequest struct {
	LegSeq     int16      `json:"legSeq" binding:"required"`
	Direction  string     `json:"direction" binding:"required"`
	Period     string     `json:"period" binding:"required"`
	DepartTime string     `json:"departTime" binding:"required"` // "09:40"
	ArriveTime *string    `json:"arriveTime"`
	RunNo      int16      `json:"runNo"`
	VehicleID  *uuid.UUID `json:"vehicleId"`
}

// CreateCaseSchedule 驗證排班規則（Leg 數量、方向、時段嚴格遞增）並存入資料庫。
func (s *MasterService) CreateCaseSchedule(ctx context.Context, req CreateScheduleRequest) (*repository.CaseScheduleEntity, error) {
	// 驗證 leg 數量與 tripPattern
	if int(req.TripPattern) != len(req.Legs) {
		return nil, ErrInvalidTripPattern
	}

	// 驗證方向組合與時段嚴格遞增
	if req.TripPattern == 2 {
		if req.Legs[0].Direction != "outbound" || req.Legs[1].Direction != "inbound" {
			return nil, errors.New("pattern 2 must have legs in outbound, inbound order")
		}
	} else if req.TripPattern == 4 {
		if req.Legs[0].Direction != "outbound" || req.Legs[1].Direction != "inbound" ||
			req.Legs[2].Direction != "outbound" || req.Legs[3].Direction != "inbound" {
			return nil, errors.New("pattern 4 must have legs in out/in/out/in order")
		}
	}

	for i := 0; i < len(req.Legs)-1; i++ {
		if req.Legs[i].DepartTime >= req.Legs[i+1].DepartTime {
			return nil, ErrLegTimesNotOrdered
		}
	}

	if req.UnitPrice <= 0 {
		req.UnitPrice = 115.00
	}
	if req.ServiceDurationMin <= 0 {
		req.ServiceDurationMin = 10
	}
	if req.ServiceCode == "" {
		req.ServiceCode = "BD03"
	}

	var legs []repository.ScheduleLegEntity
	for _, l := range req.Legs {
		runNo := l.RunNo
		if runNo <= 0 {
			runNo = 1
		}
		legs = append(legs, repository.ScheduleLegEntity{
			ID:         uuid.New(),
			LegSeq:     l.LegSeq,
			Direction:  l.Direction,
			Period:     l.Period,
			DepartTime: l.DepartTime,
			ArriveTime: l.ArriveTime,
			RunNo:      runNo,
			VehicleID:  l.VehicleID,
		})
	}

	entity := repository.CaseScheduleEntity{
		ID:                 uuid.New(),
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

	if s.db != nil {
		if err := s.caseRepo.CreateSchedule(ctx, &entity); err != nil {
			return nil, fmt.Errorf("failed to save case schedule: %w", err)
		}
	}

	return &entity, nil
}
