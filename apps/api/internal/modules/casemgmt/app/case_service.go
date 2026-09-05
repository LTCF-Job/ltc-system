package app

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"ltc-system/apps/api/internal/domain/crypto"
	"ltc-system/apps/api/internal/domain/namenorm"
	"ltc-system/apps/api/internal/platform/config"
)

var (
	ErrSiteRegionMismatch       = errors.New("site region does not match case region")
	ErrCaseRegionUnset          = errors.New("個案尚未設定所屬區域，無法建立排班")
	ErrInvalidTripPattern       = errors.New("trip pattern must match schedule legs count")
	ErrLegTimesNotOrdered       = errors.New("schedule leg departure times must be strictly increasing")
	ErrInvalidScheduleWeekday   = errors.New("schedule weekdays must be unique values from 1 to 7")
	ErrInvalidScheduleLegSeq    = errors.New("schedule leg sequence must be unique and within trip pattern")
	ErrInvalidScheduleDirection = errors.New("schedule leg direction must be outbound or inbound")
	ErrInvalidScheduleTime      = errors.New("schedule leg departure time must use HH:MM format")
	ErrInvalidSchedulePrice     = errors.New("schedule unit price must be greater than zero")
	ErrInvalidScheduleDistance  = errors.New("schedule distance must not be negative")
	ErrInvalidScheduleDuration  = errors.New("schedule service duration must be between 1 and 240 minutes")
	ErrInvalidScheduleDateRange = errors.New("schedule effective end date must not be before start date")
	ErrCaseNotFound             = errors.New("case not found")
	ErrCaseNameRequired         = errors.New("case name is required")
	ErrNationalIDNotConfigured  = errors.New("national id is not configured")
	ErrRevealAuditUnavailable   = errors.New("reveal audit is unavailable")
)

// CaseService 封裝個案、單位、車輛、司機與排班之業務邏輯。
type CaseService struct {
	cfg       *config.Config
	caseRepo  CaseStore
	siteRepo  SiteFinder
	auditRepo AuditWriter
	renderer  ProfileRenderer
	txRunner  TransactionRunner
}

// NewCaseService 建立 CaseService 實例。
func NewCaseService(
	cfg *config.Config,
	caseRepo CaseStore,
	siteRepo SiteFinder,
	auditRepo AuditWriter,
	renderer ProfileRenderer,
	txRunners ...TransactionRunner,
) *CaseService {
	var txRunner TransactionRunner
	if len(txRunners) > 0 {
		txRunner = txRunners[0]
	}
	return &CaseService{
		cfg:       cfg,
		caseRepo:  caseRepo,
		siteRepo:  siteRepo,
		auditRepo: auditRepo,
		renderer:  renderer,
		txRunner:  txRunner,
	}
}

// CreateCaseRequest 代表新增個案之請求參數。
type CreateCaseRequest struct {
	Name              string
	NationalID        string
	HouseholdType     *string
	Gender            *string
	BirthDate         *time.Time
	CareContactRole   *string
	CareContactName   *string
	RegisteredAddress *string
	HomeAddress       *string
	Region            *string
	LTCLevel          *string
	ServiceCategory   *int
	ServiceUsageType  *int
	ClaimEndDate      *time.Time
	Status            string
	Remarks           *string
}

// CreateCase 建立個案主檔；僅姓名為必要輸入，身分證字號提供時仍需通過格式檢查與加密雜湊產生，
// 不再檢查唯一性（個案身分證字號與姓名皆允許重複）。
func (s *CaseService) CreateCase(ctx context.Context, req CreateCaseRequest, actorID uuid.UUID, actorRole, ip, ua string) (*Case, error) {
	req.NationalID = strings.TrimSpace(strings.ToUpper(req.NationalID))

	var cipherText, hmacIdx []byte
	var maskedID string
	if req.NationalID != "" {
		if !crypto.ValidateNationalID(req.NationalID) {
			return nil, errors.New("invalid national ID format")
		}

		hmacIdx = crypto.Index(req.NationalID, s.cfg.HMACKey)

		var err error
		cipherText, err = crypto.Encrypt(req.NationalID, s.cfg.EncryptionKey)
		if err != nil {
			return nil, fmt.Errorf("failed to encrypt national id: %w", err)
		}
		maskedID = crypto.Mask(req.NationalID)
	}

	normName := namenorm.Normalize(req.Name)
	if req.Status == "" {
		req.Status = "active"
	}

	entity := Case{
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
		ClaimEndDate:      req.ClaimEndDate,
		Status:            req.Status,
		Remarks:           req.Remarks,
	}

	if err := s.caseRepo.Create(ctx, &entity); err != nil {
		return nil, fmt.Errorf("failed to create case: %w", err)
	}

	if s.auditRepo != nil {
		entityIDStr := entity.ID.String()
		_ = s.auditRepo.Write(ctx, AuditEntry{
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

// ListCases 查詢個案清單（回傳遮罩身分證）。unresolvedLink 為 true 時僅回傳
// 單位／去回程車輛任一比對不到主檔（raw name 有值但對應 ID 為 null）的個案；
// excludePending 為 true 時排除這類待維護個案。
func (s *CaseService) ListCases(ctx context.Context, region, status, q string, page, pageSize int, unresolvedLink, excludePending bool) ([]Case, int64, error) {
	return s.caseRepo.List(ctx, region, status, q, page, pageSize, unresolvedLink, excludePending)
}

// GetCaseByID 取得單筆個案主檔明細。
func (s *CaseService) GetCaseByID(ctx context.Context, id uuid.UUID) (*Case, error) {
	return s.caseRepo.GetByID(ctx, id)
}

// UpdateCaseInput 代表更新個案主檔所需之輸入，欄位為 nil 表示不變更。
type UpdateCaseInput struct {
	Name                *string
	HomeAddress         *string
	Region              *string
	LTCLevel            *string
	ServiceCategory     *int
	ServiceUsageType    *int
	ClaimEndDate        *time.Time
	ClaimEndDatePresent bool
	Status              *string
	HouseholdType       *string
	Gender              *string
	BirthDate           *time.Time
	BirthDatePresent    bool
	CareContactRole     *string
	CareContactName     *string
	RegisteredAddress   *string
	Remarks             *string
}

// UpdateCase 更新個案主檔資料，僅套用有提供的欄位。
func (s *CaseService) UpdateCase(ctx context.Context, id uuid.UUID, in UpdateCaseInput) (*Case, error) {
	entity, err := s.caseRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if in.Name != nil {
		name := strings.TrimSpace(*in.Name)
		if name == "" {
			return nil, ErrCaseNameRequired
		}
		entity.Name = name
		entity.NameNormalized = namenorm.Normalize(name)
	}
	if in.HomeAddress != nil {
		entity.HomeAddress = in.HomeAddress
	}
	if in.Region != nil {
		entity.Region = in.Region
	}
	if in.LTCLevel != nil {
		entity.LTCLevel = in.LTCLevel
	}
	if in.ServiceCategory != nil {
		entity.ServiceCategory = in.ServiceCategory
	}
	if in.ServiceUsageType != nil {
		entity.ServiceUsageType = in.ServiceUsageType
	}
	if in.ClaimEndDatePresent {
		entity.ClaimEndDate = in.ClaimEndDate
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
	if in.BirthDatePresent {
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
	if in.Remarks != nil {
		entity.Remarks = in.Remarks
	}

	if err := s.caseRepo.Update(ctx, entity); err != nil {
		return nil, err
	}
	return entity, nil
}

// Delete 軟刪除個案並收斂其生效中排班。
func (s *CaseService) Delete(ctx context.Context, id, actorID uuid.UUID, actorRole, ip, ua string) error {
	before, err := s.caseRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if before == nil {
		return ErrCaseNotFound
	}

	deleteFn := func(txCtx context.Context) error {
		ok, err := s.caseRepo.SoftDelete(txCtx, id, actorID)
		if err != nil {
			return fmt.Errorf("failed to soft delete case: %w", err)
		}
		if !ok {
			return ErrCaseNotFound
		}

		if err := s.caseRepo.CloseOpenSchedules(txCtx, id); err != nil {
			return fmt.Errorf("failed to close open schedules: %w", err)
		}

		if s.auditRepo != nil {
			entityIDStr := id.String()
			if err := s.auditRepo.Write(txCtx, AuditEntry{
				ActorID:    &actorID,
				ActorRole:  &actorRole,
				Action:     "delete",
				EntityType: "cases",
				EntityID:   &entityIDStr,
				BeforeData: before,
				IPAddress:  &ip,
				UserAgent:  &ua,
			}); err != nil {
				return fmt.Errorf("failed to write case deletion audit: %w", err)
			}
		}
		return nil
	}

	if s.txRunner != nil {
		return s.txRunner.WithTx(ctx, deleteFn)
	}
	return deleteFn(ctx)
}

// UpdateCaseTransportPreference 更新個案的交通偏好（所屬單位與去回程車輛），回傳更新後的個案主檔。
// 三個 ID 皆為 nil 表示維持現況，僅提供的欄位會被寫入；raw name 字串只在對應 ID
// 為 nil 且需要保留原始名稱待人工關聯時才有意義。
func (s *CaseService) UpdateCaseTransportPreference(ctx context.Context, caseID uuid.UUID, siteID, outboundVehicleID, inboundVehicleID *uuid.UUID, siteNameRaw, outboundVehicleNameRaw, inboundVehicleNameRaw string) (*Case, error) {
	if err := s.caseRepo.UpsertTransportPreference(ctx, caseID, siteID, outboundVehicleID, inboundVehicleID, siteNameRaw, outboundVehicleNameRaw, inboundVehicleNameRaw); err != nil {
		return nil, err
	}
	return s.caseRepo.GetByID(ctx, caseID)
}

// FindPossibleDuplicate 依身分證字號（非空時）或正規化姓名比對既有個案，供批次匯入
// 於 dry-run 階段查重使用；找不到相符個案時回傳 nil、nil。
func (s *CaseService) FindPossibleDuplicate(ctx context.Context, nationalID, name string) (*Case, error) {
	nationalID = strings.TrimSpace(strings.ToUpper(nationalID))
	if nationalID != "" {
		hmacIdx := crypto.Index(nationalID, s.cfg.HMACKey)
		return s.caseRepo.GetByHMAC(ctx, hmacIdx)
	}

	matches, err := s.caseRepo.GetByNameNormalized(ctx, namenorm.Normalize(name))
	if err != nil {
		return nil, err
	}
	if len(matches) == 0 {
		return nil, nil
	}
	return &matches[0], nil
}

// GetActiveScheduleForCaseOnDate 取得個案於指定日期生效之排班；查無資料時回傳 nil、nil，
// 底層查詢失敗時回傳 error（呼叫端不應將兩者混為一談）。
func (s *CaseService) GetActiveScheduleForCaseOnDate(ctx context.Context, caseID uuid.UUID, serviceDate time.Time) (*CaseSchedule, error) {
	return s.caseRepo.GetActiveScheduleForCaseOnDate(ctx, caseID, serviceDate)
}

// RevealCaseNationalID 解密個案身分證並留存稽核日誌。
func (s *CaseService) RevealCaseNationalID(ctx context.Context, caseID uuid.UUID, actorID uuid.UUID, actorRole, ip, ua string) (string, error) {
	caseEntity, err := s.caseRepo.GetByID(ctx, caseID)
	if err != nil {
		return "", err
	}
	if caseEntity == nil {
		return "", ErrCaseNotFound
	}
	if len(caseEntity.NationalIDCipher) == 0 {
		return "", ErrNationalIDNotConfigured
	}

	if s.auditRepo == nil {
		return "", ErrRevealAuditUnavailable
	}

	entityIDStr := caseID.String()
	if err := s.auditRepo.Write(ctx, AuditEntry{
		ActorID:    &actorID,
		ActorRole:  &actorRole,
		Action:     "reveal_pii",
		EntityType: "cases",
		EntityID:   &entityIDStr,
		IPAddress:  &ip,
		UserAgent:  &ua,
	}); err != nil {
		return "", fmt.Errorf("%w: %v", ErrRevealAuditUnavailable, err)
	}

	plainID, err := crypto.Decrypt(caseEntity.NationalIDCipher, s.cfg.EncryptionKey)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt national id: %w", err)
	}

	return plainID, nil
}

// CaseImportSkippedRow 是寫入稽核日誌的略過列快照。json tag 即為 audit_log 中
// import_skip 紀錄的資料契約，不得與呼叫端的型別分歧。
type CaseImportSkippedRow struct {
	RowID     string            `json:"rowId"`
	RowIndex  int               `json:"rowIndex"`
	CaseName  string            `json:"caseName"`
	Reasons   []string          `json:"reasons"`
	RawValues map[string]string `json:"rawValues"`
}

// sanitizeCaseImportAuditRow 移除匯入略過列中的明文個資；完整原始值仍可留在
// 目前回應給操作人員，但不可再寫入長期保存的 audit_log。
func sanitizeCaseImportAuditRow(item CaseImportSkippedRow) CaseImportSkippedRow {
	item.CaseName = maskAuditName(item.CaseName)
	values := make(map[string]string, len(item.RawValues))
	for key, value := range item.RawValues {
		normalizedKey := strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(strings.TrimSpace(key), " ", ""), "_", ""))
		switch {
		case strings.Contains(normalizedKey, "身分證"), strings.Contains(normalizedKey, "nationalid"):
			values[key] = crypto.Mask(value)
		case strings.Contains(normalizedKey, "地址"), strings.Contains(normalizedKey, "居住"), strings.Contains(normalizedKey, "戶籍"), strings.Contains(normalizedKey, "聯絡"), strings.Contains(normalizedKey, "電話"), strings.Contains(normalizedKey, "手機"):
			values[key] = "[REDACTED]"
		case strings.Contains(normalizedKey, "姓名"), strings.EqualFold(normalizedKey, "name"):
			values[key] = maskAuditName(value)
		default:
			values[key] = value
		}
	}
	item.RawValues = values
	return item
}

func maskAuditName(name string) string {
	runes := []rune(strings.TrimSpace(name))
	switch len(runes) {
	case 0:
		return ""
	case 1:
		return "○"
	case 2:
		return string(runes[:1]) + "○"
	default:
		return string(runes[:1]) + "○" + string(runes[len(runes)-1:])
	}
}

// RecordSkippedCaseImport 保留未寫入的來源列，讓操作人員能回查補正原因與原始欄位。
func (s *CaseService) RecordSkippedCaseImport(ctx context.Context, item CaseImportSkippedRow, actorID uuid.UUID, actorRole, ip, ua string) {
	if s.auditRepo == nil {
		return
	}
	item = sanitizeCaseImportAuditRow(item)
	entityID := fmt.Sprintf("row-%d", item.RowIndex)
	_ = s.auditRepo.Write(ctx, AuditEntry{
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
func (s *CaseService) CreateCaseSchedule(ctx context.Context, req CreateScheduleRequest) (*CaseSchedule, error) {
	if int(req.TripPattern) != len(req.Legs) {
		return nil, ErrInvalidTripPattern
	}
	if err := validateScheduleRequest(req); err != nil {
		return nil, err
	}

	caseObj, err := s.caseRepo.GetByID(ctx, req.CaseID)
	if err != nil {
		return nil, fmt.Errorf("case not found: %w", err)
	}

	siteObj, err := s.siteRepo.GetByID(ctx, req.SiteID)
	if err != nil {
		return nil, fmt.Errorf("site not found: %w", err)
	}

	if caseObj.Region == nil {
		return nil, ErrCaseRegionUnset
	}
	if *caseObj.Region != siteObj.Region {
		return nil, ErrSiteRegionMismatch
	}

	var legs []ScheduleLeg
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

		legs = append(legs, ScheduleLeg{
			LegSeq:     l.LegSeq,
			Direction:  l.Direction,
			Period:     period,
			DepartTime: l.DepartTime,
			RunNo:      1,
			VehicleID:  l.VehicleID,
		})
	}

	entity := CaseSchedule{
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

func validateScheduleRequest(req CreateScheduleRequest) error {
	weekdays := make(map[int16]struct{}, len(req.Weekdays))
	for _, weekday := range req.Weekdays {
		if weekday < 1 || weekday > 7 {
			return ErrInvalidScheduleWeekday
		}
		if _, exists := weekdays[weekday]; exists {
			return ErrInvalidScheduleWeekday
		}
		weekdays[weekday] = struct{}{}
	}
	if req.UnitPrice <= 0 {
		return ErrInvalidSchedulePrice
	}
	if req.DistanceKM < 0 {
		return ErrInvalidScheduleDistance
	}
	if req.ServiceDurationMin < 1 || req.ServiceDurationMin > 240 {
		return ErrInvalidScheduleDuration
	}
	if req.EffectiveTo != nil && req.EffectiveTo.Before(req.EffectiveFrom) {
		return ErrInvalidScheduleDateRange
	}

	legSeqs := make(map[int16]struct{}, len(req.Legs))
	for _, leg := range req.Legs {
		if leg.LegSeq < 1 || leg.LegSeq > req.TripPattern {
			return ErrInvalidScheduleLegSeq
		}
		if _, exists := legSeqs[leg.LegSeq]; exists {
			return ErrInvalidScheduleLegSeq
		}
		legSeqs[leg.LegSeq] = struct{}{}
		if leg.Direction != "outbound" && leg.Direction != "inbound" {
			return ErrInvalidScheduleDirection
		}
		parsedTime, err := time.Parse("15:04", leg.DepartTime)
		if err != nil || parsedTime.Format("15:04") != leg.DepartTime {
			return ErrInvalidScheduleTime
		}
	}
	return nil
}
