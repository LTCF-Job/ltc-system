package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"ltc-system/apps/api/internal/domain/crypto"
	"ltc-system/apps/api/internal/domain/namenorm"
	"ltc-system/apps/api/internal/platform/config"
)

var (
	ErrInvalidTripPattern         = errors.New("trip pattern must match schedule legs count")
	ErrLegTimesNotOrdered         = errors.New("schedule leg departure times must be strictly increasing")
	ErrInvalidScheduleWeekday     = errors.New("schedule weekdays must be unique values from 1 to 7")
	ErrInvalidScheduleLegSeq      = errors.New("schedule leg sequence must be unique and within trip pattern")
	ErrInvalidScheduleDirection   = errors.New("schedule leg direction must be outbound or inbound")
	ErrInvalidScheduleTime        = errors.New("schedule leg departure time must use HH:MM format")
	ErrInvalidSchedulePrice       = errors.New("schedule unit price must be greater than zero")
	ErrInvalidScheduleDistance    = errors.New("schedule distance must be greater than zero")
	ErrInvalidScheduleDuration    = errors.New("schedule service duration must be between 1 and 240 minutes")
	ErrInvalidScheduleDateRange   = errors.New("schedule effective end date must not be before start date")
	ErrCaseNotFound               = errors.New("case not found")
	ErrCaseNameRequired           = errors.New("case name is required")
	ErrNationalIDNotConfigured    = errors.New("national id is not configured")
	ErrRevealAuditUnavailable     = errors.New("reveal audit is unavailable")
	ErrInvalidNationalIDFormat    = errors.New("invalid national id format")
	ErrDuplicateNationalID        = errors.New("national id already exists")
	ErrDuplicateCandidateNotFound = errors.New("duplicate candidate not found")
	ErrDuplicateCandidateResolved = errors.New("duplicate candidate already resolved")
	ErrInvalidDuplicateDecision   = errors.New("invalid duplicate candidate decision")
)

// CaseService 封裝個案、單位、車輛、司機與排班之業務邏輯。
type CaseService struct {
	cfg         *config.Config
	caseRepo    CaseStore
	siteRepo    SiteFinder
	auditRepo   AuditWriter
	renderer    ProfileRenderer
	txRunner    TransactionRunner
	stagingRepo DuplicateStagingStore
}

// NewCaseService 建立 CaseService 實例。
func NewCaseService(
	cfg *config.Config,
	caseRepo CaseStore,
	siteRepo SiteFinder,
	auditRepo AuditWriter,
	renderer ProfileRenderer,
	stagingRepo DuplicateStagingStore,
	txRunners ...TransactionRunner,
) *CaseService {
	var txRunner TransactionRunner
	if len(txRunners) > 0 {
		txRunner = txRunners[0]
	}
	return &CaseService{
		cfg:         cfg,
		caseRepo:    caseRepo,
		siteRepo:    siteRepo,
		auditRepo:   auditRepo,
		renderer:    renderer,
		txRunner:    txRunner,
		stagingRepo: stagingRepo,
	}
}

// CreateCaseRequest 代表新增個案之請求參數。
type CreateCaseRequest struct {
	ID                     uuid.UUID
	Name                   string
	NationalID             string
	AllowInvalidNationalID bool
	HouseholdType          *string
	Gender                 *string
	BirthDate              *time.Time
	BirthDateRaw           *string
	CareContactRole        *string
	CareContactName        *string
	RegisteredAddress      *string
	HomeAddress            *string
	Region                 *string
	LTCLevel               *string
	ServiceCategory        *int
	ServiceUsageType       *int
	ClaimEndDate           *time.Time
	Status                 string
	Remarks                *string
}

// buildCaseEntity 組裝個案實體並套用身分證字號加密與生日 raw 保留規則，供
// CreateCase 與 ResolveDuplicateCandidate（confirmed_new 分支）共用同一段邏輯。
// AllowInvalidNationalID 僅供批次匯入路徑使用：格式不合法時不擋列，改標記
// NationalIDInvalid 並讓三個身分證欄位留空，待使用者於待維護頁重新輸入。
func (s *CaseService) buildCaseEntity(req CreateCaseRequest) (Case, error) {
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		return Case{}, ErrCaseNameRequired
	}
	req.NationalID = strings.TrimSpace(strings.ToUpper(req.NationalID))

	var cipherText, hmacIdx []byte
	var maskedID string
	var nationalIDInvalid bool
	if req.NationalID != "" {
		if !crypto.ValidateNationalID(req.NationalID) {
			if !req.AllowInvalidNationalID {
				return Case{}, ErrInvalidNationalIDFormat
			}
			nationalIDInvalid = true
		} else {
			hmacIdx = crypto.Index(req.NationalID, s.cfg.HMACKey)

			var err error
			cipherText, err = crypto.Encrypt(req.NationalID, s.cfg.EncryptionKey)
			if err != nil {
				return Case{}, fmt.Errorf("failed to encrypt national id: %w", err)
			}
			maskedID = crypto.Mask(req.NationalID)
		}
	}

	normName := namenorm.Normalize(req.Name)
	if req.Status == "" {
		req.Status = "active"
	}

	birthDate := req.BirthDate
	birthDateRaw := req.BirthDateRaw
	if birthDate != nil {
		birthDateRaw = nil
	}

	return Case{
		ID:                req.ID,
		Name:              req.Name,
		NameNormalized:    normName,
		NationalIDCipher:  cipherText,
		NationalIDHMAC:    hmacIdx,
		NationalIDMasked:  maskedID,
		NationalIDInvalid: nationalIDInvalid,
		HouseholdType:     req.HouseholdType,
		Gender:            req.Gender,
		BirthDate:         birthDate,
		BirthDateRaw:      birthDateRaw,
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
	}, nil
}

// CreateCase 建立個案主檔；僅姓名為必要輸入，身分證字號提供時仍需通過格式檢查與加密雜湊產生，
// 不再檢查唯一性（個案身分證字號與姓名皆允許重複）。
func (s *CaseService) CreateCase(ctx context.Context, req CreateCaseRequest, actorID uuid.UUID, actorRole, ip, ua string) (*Case, error) {
	entity, err := s.buildCaseEntity(req)
	if err != nil {
		return nil, err
	}

	if err := s.caseRepo.Create(ctx, &entity); err != nil {
		return nil, fmt.Errorf("failed to create case: %w", err)
	}

	if s.auditRepo != nil {
		entityIDStr := entity.ID.String()
		if err := s.auditRepo.Write(ctx, AuditEntry{
			ActorID:    &actorID,
			ActorRole:  &actorRole,
			Action:     "create",
			EntityType: "cases",
			EntityID:   &entityIDStr,
			AfterData:  newCaseAuditSnapshot(&entity),
			IPAddress:  &ip,
			UserAgent:  &ua,
		}); err != nil {
			// 個案已完成寫入；稽核失敗不可讓用戶端誤以為可安全重試建立。
			slog.Error("case_audit_write_failed", slog.String("action", "create"), slog.String("case_id", entity.ID.String()), slog.Any("error", err))
		}
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
	NationalID          *string
	CareContactRole     *string
	CareContactName     *string
	RegisteredAddress   *string
	Remarks             *string
}

// caseAuditSnapshot 是個案異動的固定稽核白名單；不得直接序列化 Case，避免把
// 身分證密文、HMAC、明文身分證、地址或照護聯絡資訊寫入長期保存的 audit_log。
type caseAuditSnapshot struct {
	NameMasked        string     `json:"nameMasked"`
	Region            *string    `json:"region,omitempty"`
	LTCLevel          *string    `json:"ltcLevel,omitempty"`
	HouseholdType     *string    `json:"householdType,omitempty"`
	Gender            *string    `json:"gender,omitempty"`
	BirthDate         *time.Time `json:"birthDate,omitempty"`
	ServiceCategory   *int       `json:"serviceCategory,omitempty"`
	ServiceUsageType  *int       `json:"serviceUsageType,omitempty"`
	ClaimEndDate      *time.Time `json:"claimEndDate,omitempty"`
	Status            string     `json:"status"`
	SiteID            *uuid.UUID `json:"siteId,omitempty"`
	OutboundVehicleID *uuid.UUID `json:"outboundVehicleId,omitempty"`
	InboundVehicleID  *uuid.UUID `json:"inboundVehicleId,omitempty"`
	NationalIDInvalid bool       `json:"nationalIdInvalid,omitempty"`
}

func newCaseAuditSnapshot(c *Case) caseAuditSnapshot {
	if c == nil {
		return caseAuditSnapshot{}
	}
	return caseAuditSnapshot{
		NameMasked:        maskAuditName(c.Name),
		Region:            c.Region,
		LTCLevel:          c.LTCLevel,
		HouseholdType:     c.HouseholdType,
		Gender:            c.Gender,
		BirthDate:         c.BirthDate,
		ServiceCategory:   c.ServiceCategory,
		ServiceUsageType:  c.ServiceUsageType,
		ClaimEndDate:      c.ClaimEndDate,
		Status:            c.Status,
		SiteID:            c.SiteID,
		OutboundVehicleID: c.OutboundVehicleID,
		InboundVehicleID:  c.InboundVehicleID,
		NationalIDInvalid: c.NationalIDInvalid,
	}
}

// UpdateCase 更新個案主檔資料，僅套用有提供的欄位。
func (s *CaseService) UpdateCase(ctx context.Context, id uuid.UUID, in UpdateCaseInput, actorID uuid.UUID, actorRole, ip, ua string) (*Case, error) {
	entity, err := s.caseRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	before := newCaseAuditSnapshot(entity)

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
		entity.BirthDateRaw = nil
	}
	if in.NationalID != nil {
		nid := strings.TrimSpace(strings.ToUpper(*in.NationalID))
		if !crypto.ValidateNationalID(nid) {
			return nil, ErrInvalidNationalIDFormat
		}
		cipherText, err := crypto.Encrypt(nid, s.cfg.EncryptionKey)
		if err != nil {
			return nil, fmt.Errorf("failed to encrypt national id: %w", err)
		}
		entity.NationalIDCipher = cipherText
		entity.NationalIDHMAC = crypto.Index(nid, s.cfg.HMACKey)
		entity.NationalIDMasked = crypto.Mask(nid)
		entity.NationalIDInvalid = false
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
	if s.auditRepo != nil {
		entityIDStr := entity.ID.String()
		if err := s.auditRepo.Write(ctx, AuditEntry{
			ActorID:    &actorID,
			ActorRole:  &actorRole,
			Action:     "update",
			EntityType: "cases",
			EntityID:   &entityIDStr,
			BeforeData: before,
			AfterData:  newCaseAuditSnapshot(entity),
			IPAddress:  &ip,
			UserAgent:  &ua,
		}); err != nil {
			slog.Error("case_audit_write_failed", slog.String("action", "update"), slog.String("case_id", entity.ID.String()), slog.Any("error", err))
		}
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
				BeforeData: newCaseAuditSnapshot(before),
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
// PUT 採完整替換語意：nil 的 ID 代表清除欄位，raw name 僅用於保留待人工關聯的來源名稱。
func (s *CaseService) UpdateCaseTransportPreference(ctx context.Context, caseID uuid.UUID, siteID, outboundVehicleID, inboundVehicleID *uuid.UUID, siteNameRaw, outboundVehicleNameRaw, inboundVehicleNameRaw string, auditContexts ...AuditContext) (*Case, error) {
	var before *Case
	if s.auditRepo != nil {
		var err error
		before, err = s.caseRepo.GetByID(ctx, caseID)
		if err != nil {
			return nil, err
		}
		if before == nil {
			return nil, ErrCaseNotFound
		}
	}
	if err := s.caseRepo.UpsertTransportPreference(ctx, caseID, siteID, outboundVehicleID, inboundVehicleID, siteNameRaw, outboundVehicleNameRaw, inboundVehicleNameRaw); err != nil {
		return nil, err
	}
	after, err := s.caseRepo.GetByID(ctx, caseID)
	if err != nil {
		return nil, err
	}
	if after == nil {
		return nil, ErrCaseNotFound
	}
	if s.auditRepo != nil {
		entry := AuditEntry{
			Action:     "update_transport_preference",
			EntityType: "cases",
			BeforeData: newCaseAuditSnapshot(before),
			AfterData:  newCaseAuditSnapshot(after),
		}
		if len(auditContexts) > 0 {
			actor := auditContexts[0]
			entityIDStr := caseID.String()
			entry.ActorID = &actor.ActorID
			entry.ActorRole = &actor.ActorRole
			entry.EntityID = &entityIDStr
			entry.IPAddress = &actor.IPAddress
			entry.UserAgent = &actor.UserAgent
		}
		if err := s.auditRepo.Write(ctx, entry); err != nil {
			// 交通偏好已完成更新；事後稽核故障不可讓用戶端誤以為可安全重試。
			slog.Error("case_audit_write_failed", slog.String("action", "update_transport_preference"), slog.String("case_id", caseID.String()), slog.Any("error", err))
		}
	}
	return after, nil
}

// FindPossibleDuplicate 依身分證字號（非空時）或正規化姓名比對既有個案，供批次匯入
// 於 dry-run 階段查重使用；找不到相符個案時回傳 nil、nil。
func (s *CaseService) FindPossibleDuplicate(ctx context.Context, nationalID, name string) (*Case, error) {
	nationalID = strings.TrimSpace(strings.ToUpper(nationalID))
	if nationalID != "" {
		hmacIdx := crypto.Index(nationalID, s.cfg.HMACKey)
		found, err := s.caseRepo.GetByHMAC(ctx, hmacIdx)
		if err != nil {
			if errors.Is(err, ErrCaseNotFound) {
				return nil, nil
			}
			return nil, err
		}
		return found, nil
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

// StageDuplicateCandidateInput 代表批次匯入單列疑似重複個案之暫存輸入；NationalID
// 為明文，僅在本次呼叫內傳遞，寫入前立即加密，不落地、不回傳、不記錄。
type StageDuplicateCandidateInput struct {
	FileHash               string
	RowKey                 string
	RowIndex               int
	SheetName              string
	Name                   string
	NationalID             string
	HouseholdType          *string
	Gender                 *string
	BirthDate              *time.Time
	BirthDateRaw           *string
	CareContactRole        *string
	CareContactName        *string
	RegisteredAddress      *string
	HomeAddress            *string
	Region                 *string
	ServiceCategory        *int
	ServiceUsageType       *int
	SiteID                 *uuid.UUID
	SiteNameRaw            string
	OutboundVehicleID      *uuid.UUID
	OutboundVehicleNameRaw string
	InboundVehicleID       *uuid.UUID
	InboundVehicleNameRaw  string
	Remarks                *string
	DuplicateCaseID        uuid.UUID
}

func emptyToNil(v string) *string {
	if v == "" {
		return nil
	}
	return &v
}

// StageDuplicateCandidate 將疑似重複個案的整列資料寫入暫存，不建立 cases 資料列；
// 身分證字號比照 CreateCase 立即加密，暫存表全程不留明文。
func (s *CaseService) StageDuplicateCandidate(ctx context.Context, in StageDuplicateCandidateInput) (uuid.UUID, bool, error) {
	name := strings.TrimSpace(in.Name)
	nationalID := strings.TrimSpace(strings.ToUpper(in.NationalID))

	var cipherText, hmacIdx []byte
	var maskedID string
	var nationalIDInvalid bool
	if nationalID != "" {
		if !crypto.ValidateNationalID(nationalID) {
			nationalIDInvalid = true
		} else {
			hmacIdx = crypto.Index(nationalID, s.cfg.HMACKey)
			var err error
			cipherText, err = crypto.Encrypt(nationalID, s.cfg.EncryptionKey)
			if err != nil {
				return uuid.Nil, false, fmt.Errorf("failed to encrypt national id: %w", err)
			}
			maskedID = crypto.Mask(nationalID)
		}
	}

	birthDate := in.BirthDate
	birthDateRaw := in.BirthDateRaw
	if birthDate != nil {
		birthDateRaw = nil
	}

	cand := DuplicateCandidate{
		FileHash:               in.FileHash,
		RowKey:                 in.RowKey,
		RowIndex:               in.RowIndex,
		SheetName:              in.SheetName,
		Name:                   name,
		NameNormalized:         namenorm.Normalize(name),
		NationalIDCipher:       cipherText,
		NationalIDHMAC:         hmacIdx,
		NationalIDMasked:       maskedID,
		NationalIDInvalid:      nationalIDInvalid,
		HouseholdType:          in.HouseholdType,
		Gender:                 in.Gender,
		BirthDate:              birthDate,
		BirthDateRaw:           birthDateRaw,
		CareContactRole:        in.CareContactRole,
		CareContactName:        in.CareContactName,
		RegisteredAddress:      in.RegisteredAddress,
		HomeAddress:            in.HomeAddress,
		Region:                 in.Region,
		ServiceCategory:        in.ServiceCategory,
		ServiceUsageType:       in.ServiceUsageType,
		SiteID:                 in.SiteID,
		SiteNameRaw:            emptyToNil(in.SiteNameRaw),
		OutboundVehicleID:      in.OutboundVehicleID,
		OutboundVehicleNameRaw: emptyToNil(in.OutboundVehicleNameRaw),
		InboundVehicleID:       in.InboundVehicleID,
		InboundVehicleNameRaw:  emptyToNil(in.InboundVehicleNameRaw),
		Remarks:                in.Remarks,
		DuplicateCaseID:        in.DuplicateCaseID,
	}

	return s.stagingRepo.Insert(ctx, cand)
}

// ListDuplicateCandidates 取得所有待裁決的疑似重複個案暫存列。
func (s *CaseService) ListDuplicateCandidates(ctx context.Context) ([]DuplicateCandidate, error) {
	return s.stagingRepo.ListPending(ctx)
}

// RevealDuplicateCandidateNationalID 解密單筆暫存列的明文身分證字號供裁決頁比對，並留存稽核日誌；
// 比照 RevealCaseNationalID 的「加密儲存、稽核後解密顯示」模式。
func (s *CaseService) RevealDuplicateCandidateNationalID(ctx context.Context, id uuid.UUID, actorID uuid.UUID, actorRole, ip, ua string) (string, error) {
	cand, err := s.stagingRepo.GetByID(ctx, id)
	if err != nil {
		return "", err
	}
	if cand == nil {
		return "", ErrDuplicateCandidateNotFound
	}
	if len(cand.NationalIDCipher) == 0 {
		return "", ErrNationalIDNotConfigured
	}
	if s.auditRepo == nil {
		return "", ErrRevealAuditUnavailable
	}

	entityIDStr := id.String()
	if err := s.auditRepo.Write(ctx, AuditEntry{
		ActorID:    &actorID,
		ActorRole:  &actorRole,
		Action:     "reveal_pii",
		EntityType: "case_import_duplicate_rows",
		EntityID:   &entityIDStr,
		IPAddress:  &ip,
		UserAgent:  &ua,
	}); err != nil {
		return "", fmt.Errorf("%w: %v", ErrRevealAuditUnavailable, err)
	}

	plainID, err := crypto.Decrypt(cand.NationalIDCipher, s.cfg.EncryptionKey)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt national id: %w", err)
	}
	return plainID, nil
}

// ResolveDuplicateCandidate 裁決一筆疑似重複個案：confirmed_new 建立新個案並綁定交通偏好；
// merged_existing 只在目標既有個案對應欄位為空時才寫入暫存列的值，備註一律 append，不覆蓋既有資料。
func (s *CaseService) ResolveDuplicateCandidate(ctx context.Context, id uuid.UUID, decision string, targetCaseID *uuid.UUID, mergeRemarks bool, actorID uuid.UUID, actorRole, ip, ua string) (*Case, error) {
	if decision != "confirmed_new" && decision != "merged_existing" {
		return nil, ErrInvalidDuplicateDecision
	}

	var result *Case
	resolveFn := func(txCtx context.Context) error {
		cand, err := s.stagingRepo.GetByID(txCtx, id)
		if err != nil {
			return err
		}
		if cand == nil || cand.Status != "pending" {
			return ErrDuplicateCandidateNotFound
		}

		var resultingCaseID uuid.UUID
		if decision == "confirmed_new" {
			created, err := s.resolveDuplicateAsNewCase(txCtx, cand, actorID, actorRole, ip, ua)
			if err != nil {
				return err
			}
			result = created
			resultingCaseID = created.ID
		} else {
			target := cand.DuplicateCaseID
			if targetCaseID != nil {
				target = *targetCaseID
			}
			merged, err := s.mergeDuplicateIntoExisting(txCtx, target, cand, mergeRemarks, actorID, actorRole, ip, ua)
			if err != nil {
				return err
			}
			result = merged
			resultingCaseID = merged.ID
		}

		rowsAffected, err := s.stagingRepo.Resolve(txCtx, id, decision, actorID, &resultingCaseID)
		if err != nil {
			return err
		}
		if rowsAffected == 0 {
			return ErrDuplicateCandidateResolved
		}
		return nil
	}

	if s.txRunner != nil {
		if err := s.txRunner.WithTx(ctx, resolveFn); err != nil {
			return nil, err
		}
	} else if err := resolveFn(ctx); err != nil {
		return nil, err
	}
	return result, nil
}

func (s *CaseService) resolveDuplicateAsNewCase(ctx context.Context, cand *DuplicateCandidate, actorID uuid.UUID, actorRole, ip, ua string) (*Case, error) {
	entity := Case{
		ID:                uuid.New(),
		Name:              cand.Name,
		NameNormalized:    cand.NameNormalized,
		NationalIDCipher:  cand.NationalIDCipher,
		NationalIDHMAC:    cand.NationalIDHMAC,
		NationalIDMasked:  cand.NationalIDMasked,
		NationalIDInvalid: cand.NationalIDInvalid,
		HouseholdType:     cand.HouseholdType,
		Gender:            cand.Gender,
		BirthDate:         cand.BirthDate,
		BirthDateRaw:      cand.BirthDateRaw,
		CareContactRole:   cand.CareContactRole,
		CareContactName:   cand.CareContactName,
		RegisteredAddress: cand.RegisteredAddress,
		HomeAddress:       cand.HomeAddress,
		Region:            cand.Region,
		ServiceCategory:   cand.ServiceCategory,
		ServiceUsageType:  cand.ServiceUsageType,
		Status:            "active",
		Remarks:           cand.Remarks,
	}
	if err := s.caseRepo.Create(ctx, &entity); err != nil {
		return nil, fmt.Errorf("failed to create case from duplicate candidate: %w", err)
	}
	if err := s.caseRepo.UpsertTransportPreference(ctx, entity.ID, cand.SiteID, cand.OutboundVehicleID, cand.InboundVehicleID,
		derefOrEmpty(cand.SiteNameRaw), derefOrEmpty(cand.OutboundVehicleNameRaw), derefOrEmpty(cand.InboundVehicleNameRaw)); err != nil {
		return nil, fmt.Errorf("failed to set transport preference for confirmed duplicate: %w", err)
	}

	if s.auditRepo != nil {
		entityIDStr := entity.ID.String()
		if err := s.auditRepo.Write(ctx, AuditEntry{
			ActorID: &actorID, ActorRole: &actorRole, Action: "create", EntityType: "cases", EntityID: &entityIDStr,
			AfterData: newCaseAuditSnapshot(&entity), IPAddress: &ip, UserAgent: &ua,
		}); err != nil {
			return nil, fmt.Errorf("failed to write confirm-duplicate audit: %w", err)
		}
	}
	return &entity, nil
}

// mergeDuplicateIntoExisting 只補齊既有個案本來是空的欄位，不覆蓋既有值；備註一律
// append 而非取代，避免裁決動作意外抹掉既有個案已經記錄的資訊。
func (s *CaseService) mergeDuplicateIntoExisting(ctx context.Context, targetCaseID uuid.UUID, cand *DuplicateCandidate, mergeRemarks bool, actorID uuid.UUID, actorRole, ip, ua string) (*Case, error) {
	entity, err := s.caseRepo.GetByID(ctx, targetCaseID)
	if err != nil {
		return nil, err
	}
	if entity == nil {
		return nil, ErrCaseNotFound
	}
	before := newCaseAuditSnapshot(entity)

	if entity.HouseholdType == nil {
		entity.HouseholdType = cand.HouseholdType
	}
	if entity.Gender == nil {
		entity.Gender = cand.Gender
	}
	if entity.BirthDate == nil {
		entity.BirthDate = cand.BirthDate
		entity.BirthDateRaw = cand.BirthDateRaw
	}
	if entity.CareContactRole == nil {
		entity.CareContactRole = cand.CareContactRole
	}
	if entity.CareContactName == nil {
		entity.CareContactName = cand.CareContactName
	}
	if entity.RegisteredAddress == nil {
		entity.RegisteredAddress = cand.RegisteredAddress
	}
	if entity.HomeAddress == nil {
		entity.HomeAddress = cand.HomeAddress
	}
	if entity.Region == nil {
		entity.Region = cand.Region
	}
	if mergeRemarks && cand.Remarks != nil && strings.TrimSpace(*cand.Remarks) != "" {
		merged := strings.TrimSpace(derefOrEmpty(entity.Remarks))
		note := "[匯入合併備註] " + strings.TrimSpace(*cand.Remarks)
		if merged == "" {
			merged = note
		} else {
			merged = merged + "\n" + note
		}
		entity.Remarks = &merged
	}

	if err := s.caseRepo.Update(ctx, entity); err != nil {
		return nil, err
	}
	if entity.SiteID == nil && cand.SiteID != nil || entity.OutboundVehicleID == nil && cand.OutboundVehicleID != nil || entity.InboundVehicleID == nil && cand.InboundVehicleID != nil {
		siteID := entity.SiteID
		if siteID == nil {
			siteID = cand.SiteID
		}
		outboundID := entity.OutboundVehicleID
		if outboundID == nil {
			outboundID = cand.OutboundVehicleID
		}
		inboundID := entity.InboundVehicleID
		if inboundID == nil {
			inboundID = cand.InboundVehicleID
		}
		if err := s.caseRepo.UpsertTransportPreference(ctx, entity.ID, siteID, outboundID, inboundID,
			derefOrEmpty(cand.SiteNameRaw), derefOrEmpty(cand.OutboundVehicleNameRaw), derefOrEmpty(cand.InboundVehicleNameRaw)); err != nil {
			return nil, fmt.Errorf("failed to backfill transport preference on merge: %w", err)
		}
	}

	after, err := s.caseRepo.GetByID(ctx, entity.ID)
	if err != nil {
		return nil, err
	}
	if s.auditRepo != nil {
		entityIDStr := entity.ID.String()
		if err := s.auditRepo.Write(ctx, AuditEntry{
			ActorID: &actorID, ActorRole: &actorRole, Action: "conflict_resolve", EntityType: "cases", EntityID: &entityIDStr,
			BeforeData: before, AfterData: newCaseAuditSnapshot(after), IPAddress: &ip, UserAgent: &ua,
		}); err != nil {
			return nil, fmt.Errorf("failed to write merge-duplicate audit: %w", err)
		}
	}
	return after, nil
}

func derefOrEmpty(v *string) string {
	if v == nil {
		return ""
	}
	return *v
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
	if err := s.auditRepo.Write(ctx, AuditEntry{
		ActorID: &actorID, ActorRole: &actorRole, Action: "import_skip", EntityType: "case_import", EntityID: &entityID,
		AfterData: item, IPAddress: &ip, UserAgent: &ua,
	}); err != nil {
		slog.Error("case import skipped-row audit write failed", "row_index", item.RowIndex, "error", err)
	}
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

	if _, err := s.caseRepo.GetByID(ctx, req.CaseID); err != nil {
		return nil, fmt.Errorf("case not found: %w", err)
	}

	// 排班所屬單位不要求與案主地區一致，允許跨區指派。
	if _, err := s.siteRepo.GetByID(ctx, req.SiteID); err != nil {
		return nil, fmt.Errorf("site not found: %w", err)
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
	if req.DistanceKM <= 0 {
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
