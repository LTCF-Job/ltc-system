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
	ErrInvalidNationalIDFormat    = errors.New("invalid national id format")
	ErrDuplicateNationalID        = errors.New("national id already exists")
	ErrDuplicateCandidateNotFound = errors.New("duplicate candidate not found")
	ErrDuplicateCandidateResolved = errors.New("duplicate candidate already resolved")
	ErrInvalidDuplicateDecision   = errors.New("invalid duplicate candidate decision")
	ErrScheduleOverlap            = errors.New("schedule effective period overlaps an existing schedule")
	ErrScheduleInvalidReference   = errors.New("schedule references a case or vehicle that does not exist")
)

// CaseService 封裝個案、據點、車輛、司機與排班之業務邏輯。
type CaseService struct {
	cfg         *config.Config
	caseRepo    CaseStore
	auditRepo   AuditWriter
	renderer    ProfileRenderer
	txRunner    TransactionRunner
	stagingRepo DuplicateStagingStore
}

// NewCaseService 建立 CaseService 實例。
func NewCaseService(
	cfg *config.Config,
	caseRepo CaseStore,
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
		auditRepo:   auditRepo,
		renderer:    renderer,
		txRunner:    txRunner,
		stagingRepo: stagingRepo,
	}
}

// CreateCaseRequest 代表新增個案之請求參數。SiteID／SiteNameRaw 是個案直接關聯的據點：
// 手動新增路徑由 transport 層要求 SiteID 必填，匯入路徑允許只提供 SiteNameRaw（比對不到
// 主檔時落入待維護，待人工補齊）。
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
	LTCLevel               *string
	ServiceCategory        *int
	ServiceUsageType       *int
	ClaimEndDate           *time.Time
	Status                 string
	Remarks                *string
	SiteID                 *uuid.UUID
	SiteNameRaw            *string
	CaregiverID            *uuid.UUID
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
		LTCLevel:          req.LTCLevel,
		ServiceCategory:   req.ServiceCategory,
		ServiceUsageType:  req.ServiceUsageType,
		ClaimEndDate:      req.ClaimEndDate,
		Status:            req.Status,
		Remarks:           req.Remarks,
		SiteID:            req.SiteID,
		SiteNameRaw:       req.SiteNameRaw,
		CaregiverID:       req.CaregiverID,
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

	return s.attachPlainNationalID(&entity), nil
}

// ListCases 查詢個案清單（回傳明碼身分證）。unresolvedLink 為 true 時僅回傳
// 據點／去回程車輛任一比對不到主檔（raw name 有值但對應 ID 為 null）的個案；
// excludePending 為 true 時排除這類待維護個案。
func (s *CaseService) ListCases(ctx context.Context, status, q, region string, page, pageSize int, unresolvedLink, excludePending bool) ([]Case, int64, error) {
	list, total, err := s.caseRepo.List(ctx, status, q, region, page, pageSize, unresolvedLink, excludePending)
	if err != nil {
		return nil, 0, err
	}
	return s.attachPlainNationalIDs(list), total, nil
}

// GetCaseByID 取得單筆個案主檔明細。
func (s *CaseService) GetCaseByID(ctx context.Context, id uuid.UUID) (*Case, error) {
	c, err := s.caseRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return s.attachPlainNationalID(c), nil
}

// UpdateCaseInput 代表更新個案主檔所需之輸入，欄位為 nil 表示不變更。設定 SiteID
// 即視為完成關聯，清空匯入時保留的原始據點名稱（與 caregiver 相同慣例）。
type UpdateCaseInput struct {
	Name                *string
	HomeAddress         *string
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
	SiteID              *uuid.UUID
	CaregiverID         *uuid.UUID
}

// caseAuditSnapshot 是個案異動的固定稽核白名單；不得直接序列化 Case，避免把
// 身分證密文、HMAC、明文身分證、地址或照護聯絡資訊寫入長期保存的 audit_log。
type caseAuditSnapshot struct {
	NameMasked        string     `json:"nameMasked"`
	LTCLevel          *string    `json:"ltcLevel,omitempty"`
	HouseholdType     *string    `json:"householdType,omitempty"`
	Gender            *string    `json:"gender,omitempty"`
	BirthDate         *time.Time `json:"birthDate,omitempty"`
	ServiceCategory   *int       `json:"serviceCategory,omitempty"`
	ServiceUsageType  *int       `json:"serviceUsageType,omitempty"`
	ClaimEndDate      *time.Time `json:"claimEndDate,omitempty"`
	Status            string     `json:"status"`
	SiteID            *uuid.UUID `json:"siteId,omitempty"`
	CaregiverID       *uuid.UUID `json:"caregiverId,omitempty"`
	NationalIDInvalid bool       `json:"nationalIdInvalid,omitempty"`
}

func newCaseAuditSnapshot(c *Case) caseAuditSnapshot {
	if c == nil {
		return caseAuditSnapshot{}
	}
	return caseAuditSnapshot{
		NameMasked:        maskAuditName(c.Name),
		LTCLevel:          c.LTCLevel,
		HouseholdType:     c.HouseholdType,
		Gender:            c.Gender,
		BirthDate:         c.BirthDate,
		ServiceCategory:   c.ServiceCategory,
		ServiceUsageType:  c.ServiceUsageType,
		ClaimEndDate:      c.ClaimEndDate,
		Status:            c.Status,
		SiteID:            c.SiteID,
		CaregiverID:       c.CaregiverID,
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
	if in.SiteID != nil {
		entity.SiteID = in.SiteID
		entity.SiteNameRaw = nil
	}
	if in.CaregiverID != nil {
		entity.CaregiverID = in.CaregiverID
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
	return s.attachPlainNationalID(entity), nil
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

// RelinkSiteByName 讓據點新增或改名時，重新比對名稱相符的待維護個案；只有唯一命中才
// 自動關聯，回傳實際關聯的筆數。
func (s *CaseService) RelinkSiteByName(ctx context.Context, name string, actorID uuid.UUID, actorRole, ip, ua string) (int, error) {
	return s.relinkByName(ctx, "auto_relink_site", actorID, actorRole, ip, ua, func(txCtx context.Context) ([]uuid.UUID, error) {
		return s.caseRepo.RelinkSiteByName(txCtx, name)
	}, map[string]any{"siteNameRaw": name})
}

// RelinkCaregiverByName 讓照護人員新增或改名時，重新比對名稱相符的待維護個案；只有
// 唯一命中才自動關聯，回傳實際關聯的筆數。
func (s *CaseService) RelinkCaregiverByName(ctx context.Context, name string, actorID uuid.UUID, actorRole, ip, ua string) (int, error) {
	return s.relinkByName(ctx, "auto_relink_caregiver", actorID, actorRole, ip, ua, func(txCtx context.Context) ([]uuid.UUID, error) {
		return s.caseRepo.RelinkCaregiverByName(txCtx, name)
	}, map[string]any{"careContactName": name})
}

// RelinkAllPendingSites 供待維護頁「重新比對」按鈕使用。
func (s *CaseService) RelinkAllPendingSites(ctx context.Context, actorID uuid.UUID, actorRole, ip, ua string) (int, error) {
	names, err := s.caseRepo.ListPendingSiteNames(ctx)
	if err != nil {
		return 0, err
	}
	total := 0
	for _, name := range names {
		n, err := s.RelinkSiteByName(ctx, name, actorID, actorRole, ip, ua)
		if err != nil {
			slog.Error("relink_all_pending_sites_failed", slog.String("name", name), slog.Any("error", err))
			continue
		}
		total += n
	}
	return total, nil
}

// RelinkAllPendingCaregivers 供待維護頁「重新比對」按鈕使用。
func (s *CaseService) RelinkAllPendingCaregivers(ctx context.Context, actorID uuid.UUID, actorRole, ip, ua string) (int, error) {
	names, err := s.caseRepo.ListPendingCaregiverNames(ctx)
	if err != nil {
		return 0, err
	}
	total := 0
	for _, name := range names {
		n, err := s.RelinkCaregiverByName(ctx, name, actorID, actorRole, ip, ua)
		if err != nil {
			slog.Error("relink_all_pending_caregivers_failed", slog.String("name", name), slog.Any("error", err))
			continue
		}
		total += n
	}
	return total, nil
}

// relinkByName 是 RelinkSiteByName／RelinkCaregiverByName 共用的比對＋稽核外殼。
func (s *CaseService) relinkByName(ctx context.Context, action string, actorID uuid.UUID, actorRole, ip, ua string, relink func(context.Context) ([]uuid.UUID, error), afterData map[string]any) (int, error) {
	var ids []uuid.UUID
	fn := func(txCtx context.Context) error {
		relinked, err := relink(txCtx)
		if err != nil {
			return err
		}
		ids = relinked
		if s.auditRepo == nil {
			return nil
		}
		for _, id := range ids {
			entityIDStr := id.String()
			// 稽核與資料寫入同一交易；稽核失敗回滾，避免關聯已生效卻沒留下紀錄。
			if err := s.auditRepo.Write(txCtx, AuditEntry{
				ActorID:    &actorID,
				ActorRole:  &actorRole,
				Action:     action,
				EntityType: "cases",
				EntityID:   &entityIDStr,
				AfterData:  afterData,
				IPAddress:  &ip,
				UserAgent:  &ua,
			}); err != nil {
				return fmt.Errorf("failed to write auto-relink audit: %w", err)
			}
		}
		return nil
	}
	var err error
	if s.txRunner != nil {
		err = s.txRunner.WithTx(ctx, fn)
	} else {
		err = fn(ctx)
	}
	if err != nil {
		return 0, err
	}
	return len(ids), nil
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
	FileHash          string
	RowKey            string
	RowIndex          int
	SheetName         string
	Name              string
	NationalID        string
	HouseholdType     *string
	Gender            *string
	BirthDate         *time.Time
	BirthDateRaw      *string
	CareContactRole   *string
	CareContactName   *string
	RegisteredAddress *string
	HomeAddress       *string
	ServiceCategory   *int
	ServiceUsageType  *int
	SiteID            *uuid.UUID
	SiteNameRaw       string
	CaregiverID       *uuid.UUID
	Remarks           *string
	DuplicateCaseID   uuid.UUID
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
		FileHash:          in.FileHash,
		RowKey:            in.RowKey,
		RowIndex:          in.RowIndex,
		SheetName:         in.SheetName,
		Name:              name,
		NameNormalized:    namenorm.Normalize(name),
		NationalIDCipher:  cipherText,
		NationalIDHMAC:    hmacIdx,
		NationalIDMasked:  maskedID,
		NationalIDInvalid: nationalIDInvalid,
		HouseholdType:     in.HouseholdType,
		Gender:            in.Gender,
		BirthDate:         birthDate,
		BirthDateRaw:      birthDateRaw,
		CareContactRole:   in.CareContactRole,
		CareContactName:   in.CareContactName,
		RegisteredAddress: in.RegisteredAddress,
		HomeAddress:       in.HomeAddress,
		ServiceCategory:   in.ServiceCategory,
		ServiceUsageType:  in.ServiceUsageType,
		SiteID:            in.SiteID,
		SiteNameRaw:       emptyToNil(in.SiteNameRaw),
		CaregiverID:       in.CaregiverID,
		Remarks:           in.Remarks,
		DuplicateCaseID:   in.DuplicateCaseID,
	}

	return s.stagingRepo.Insert(ctx, cand)
}

// ListDuplicateCandidates 取得所有待裁決的疑似重複個案暫存列（含解密後的明碼身分證字號）。
func (s *CaseService) ListDuplicateCandidates(ctx context.Context) ([]DuplicateCandidate, error) {
	list, err := s.stagingRepo.ListPending(ctx)
	if err != nil {
		return nil, err
	}
	return s.attachPlainNationalIDsToCandidate(list), nil
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
	return s.attachPlainNationalID(result), nil
}

// DiscardDuplicateCandidate 忽略一筆疑似重複個案：直接刪除暫存列，不建立也不合併任何個案。
// 刪除而非標記終結狀態是刻意的——使用者要的是把資料從系統移除；重新匯入同一份檔案時該筆
// 會再次進入待維護，屬預期行為。稽核只保留可追溯的非個資欄位，暫存列本身帶有身分證密文
// 與地址，不寫入 audit_log。
func (s *CaseService) DiscardDuplicateCandidate(ctx context.Context, id, actorID uuid.UUID, actorRole, ip, ua string) error {
	cand, err := s.stagingRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if cand == nil {
		return ErrDuplicateCandidateNotFound
	}
	if cand.Status != "pending" {
		return ErrDuplicateCandidateResolved
	}

	discardFn := func(txCtx context.Context) error {
		rowsAffected, err := s.stagingRepo.Delete(txCtx, id)
		if err != nil {
			return err
		}
		if rowsAffected == 0 {
			return ErrDuplicateCandidateResolved
		}

		if s.auditRepo != nil {
			entityIDStr := id.String()
			if err := s.auditRepo.Write(txCtx, AuditEntry{
				ActorID:    &actorID,
				ActorRole:  &actorRole,
				Action:     "ignore",
				EntityType: "case_import_duplicate_rows",
				EntityID:   &entityIDStr,
				BeforeData: newDuplicateCandidateAuditSnapshot(cand),
				IPAddress:  &ip,
				UserAgent:  &ua,
			}); err != nil {
				return fmt.Errorf("failed to write duplicate candidate discard audit: %w", err)
			}
		}
		return nil
	}

	if s.txRunner != nil {
		return s.txRunner.WithTx(ctx, discardFn)
	}
	return discardFn(ctx)
}

// newDuplicateCandidateAuditSnapshot 只取足以追溯來源列的非個資欄位；暫存列的身分證密文、
// HMAC、遮罩值與地址一律不寫入稽核。
func newDuplicateCandidateAuditSnapshot(cand *DuplicateCandidate) map[string]any {
	return map[string]any{
		"id":              cand.ID.String(),
		"name":            cand.Name,
		"rowIndex":        cand.RowIndex,
		"sheetName":       cand.SheetName,
		"fileHash":        cand.FileHash,
		"rowKey":          cand.RowKey,
		"duplicateCaseId": cand.DuplicateCaseID.String(),
	}
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
		ServiceCategory:   cand.ServiceCategory,
		ServiceUsageType:  cand.ServiceUsageType,
		Status:            "active",
		Remarks:           cand.Remarks,
		SiteID:            cand.SiteID,
		SiteNameRaw:       cand.SiteNameRaw,
		CaregiverID:       cand.CaregiverID,
	}
	if err := s.caseRepo.Create(ctx, &entity); err != nil {
		return nil, fmt.Errorf("failed to create case from duplicate candidate: %w", err)
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
	if entity.SiteID == nil && entity.SiteNameRaw == nil {
		entity.SiteID = cand.SiteID
		entity.SiteNameRaw = cand.SiteNameRaw
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

// decryptNationalID 解密身分證密文；密文為空或解密失敗時回傳空字串，不中斷呼叫端的清單／明細查詢。
func (s *CaseService) decryptNationalID(cipher []byte) string {
	if len(cipher) == 0 {
		return ""
	}
	plain, err := crypto.Decrypt(cipher, s.cfg.EncryptionKey)
	if err != nil {
		slog.Warn("decrypt national id failed", slog.Any("error", err))
		return ""
	}
	return plain
}

// attachPlainNationalID 將解密後的明碼身分證字號填入單筆個案，供 API 回應直接顯示明碼。
func (s *CaseService) attachPlainNationalID(c *Case) *Case {
	if c == nil {
		return c
	}
	c.NationalID = s.decryptNationalID(c.NationalIDCipher)
	return c
}

// attachPlainNationalIDs 為個案清單逐筆解密身分證字號。
func (s *CaseService) attachPlainNationalIDs(list []Case) []Case {
	for i := range list {
		list[i].NationalID = s.decryptNationalID(list[i].NationalIDCipher)
	}
	return list
}

// attachPlainNationalIDToCandidate 將解密後的明碼身分證字號填入單筆疑似重複個案暫存列。
func (s *CaseService) attachPlainNationalIDToCandidate(c *DuplicateCandidate) *DuplicateCandidate {
	if c == nil {
		return c
	}
	c.NationalID = s.decryptNationalID(c.NationalIDCipher)
	return c
}

// attachPlainNationalIDsToCandidate 為疑似重複個案暫存列清單逐筆解密身分證字號。
func (s *CaseService) attachPlainNationalIDsToCandidate(list []DuplicateCandidate) []DuplicateCandidate {
	for i := range list {
		list[i].NationalID = s.decryptNationalID(list[i].NationalIDCipher)
	}
	return list
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

// CreateScheduleRequest 代表建立個案排班設定之請求參數。據點已改由個案本身持有，
// 排班不再各自指定。
type CreateScheduleRequest struct {
	CaseID             uuid.UUID
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
