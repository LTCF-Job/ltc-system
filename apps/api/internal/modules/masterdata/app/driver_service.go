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

// DriverService 封裝司機主檔業務邏輯：身分證加密、HMAC 索引、姓名正規化與車輛指派。
type DriverService struct {
	store     DriverStore
	cfg       *config.Config
	auditRepo AuditWriter
	txRunner  TransactionRunner
}

// NewDriverService 建立 DriverService 實例。
func NewDriverService(store DriverStore, cfg *config.Config, auditRepo AuditWriter, txRunners ...TransactionRunner) *DriverService {
	var txRunner TransactionRunner
	if len(txRunners) > 0 {
		txRunner = txRunners[0]
	}
	return &DriverService{store: store, cfg: cfg, auditRepo: auditRepo, txRunner: txRunner}
}

// List 查詢司機清單。
func (s *DriverService) List(ctx context.Context, region, q, status string, page, pageSize int) ([]Driver, int64, error) {
	return s.store.List(ctx, region, q, status, page, pageSize)
}

// driverLicenseClasses 是允許的駕照類別代碼，與 drivers.license_class 的 CHECK 約束一致。
var driverLicenseClasses = map[string]bool{
	"sedan":   true,
	"truck":   true,
	"bus":     true,
	"trailer": true,
}

// normalizeLicenseClass 將駕照類別正規化；空字串視為未填寫（nil）。
func normalizeLicenseClass(in *string) (*string, error) {
	if in == nil {
		return nil, nil
	}
	value := strings.TrimSpace(*in)
	if value == "" {
		return nil, nil
	}
	if !driverLicenseClasses[value] {
		return nil, ErrInvalidDriverLicenseClass
	}
	return &value, nil
}

// CreateDriverInput 代表新增司機所需之輸入。
type CreateDriverInput struct {
	Name                   string
	NationalID             string
	Email                  *string
	Region                 string
	LicenseClass           *string
	LicenseExpiryDate      *time.Time
	Gender                 *string
	BirthDate              *time.Time
	HasProfessionalLicense bool
	EmploymentDate         *time.Time
	HasTransferCert        bool
	InspectionDate         *time.Time
	Remarks                *string
}

// Create 新增司機：驗證身分證檢查碼，寫入加密密文與 HMAC 索引。actors 是可選的
// 稽核來源資訊，保留 application 測試與離線呼叫的相容性。
func (s *DriverService) Create(ctx context.Context, in CreateDriverInput, actors ...ActorContext) (*Driver, error) {
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" {
		return nil, ErrDriverNameRequired
	}
	nationalID := strings.ToUpper(strings.TrimSpace(in.NationalID))
	if !crypto.ValidateNationalID(nationalID) {
		return nil, ErrInvalidDriverNationalID
	}

	licenseClass, err := normalizeLicenseClass(in.LicenseClass)
	if err != nil {
		return nil, err
	}

	hmacIdx := crypto.Index(nationalID, s.cfg.HMACKey)
	cipherText, err := crypto.Encrypt(nationalID, s.cfg.EncryptionKey)
	if err != nil {
		return nil, fmt.Errorf("encrypt driver national id: %w", err)
	}

	d := Driver{
		ID:               uuid.New(),
		Name:             in.Name,
		NameNormalized:   namenorm.Normalize(in.Name),
		NationalIDCipher: cipherText,
		NationalIDHMAC:   hmacIdx,
		NationalIDMasked: crypto.Mask(nationalID),
		Email:            in.Email,
		Region:           strings.TrimSpace(in.Region),
		Status:           "active",

		LicenseClass:           licenseClass,
		LicenseExpiryDate:      in.LicenseExpiryDate,
		Gender:                 in.Gender,
		BirthDate:              in.BirthDate,
		HasProfessionalLicense: in.HasProfessionalLicense,
		EmploymentDate:         in.EmploymentDate,
		HasTransferCert:        in.HasTransferCert,
		InspectionDate:         in.InspectionDate,
		Remarks:                in.Remarks,
	}

	if err := s.store.Create(ctx, &d); err != nil {
		return nil, err
	}
	writeAuditBestEffort(ctx, s.auditRepo, actorOrEmpty(actors), "create", "drivers", d.ID, nil, d.AuditSnapshot())
	return &d, nil
}

// UpdateDriverInput 代表更新司機基本資料所需之輸入，欄位為 nil 表示不變更。
type UpdateDriverInput struct {
	Name                   *string
	Email                  *string
	Region                 *string
	Status                 *string
	LicenseClass           *string
	LicenseExpiryDate      *time.Time
	ClearLicenseExpiryDate bool
	Gender                 *string
	BirthDate              *time.Time
	ClearBirthDate         bool
	HasProfessionalLicense *bool
	EmploymentDate         *time.Time
	ClearEmploymentDate    bool
	HasTransferCert        *bool
	InspectionDate         *time.Time
	ClearInspectionDate    bool
	Remarks                *string
}

// Update 更新司機基本資料。
func (s *DriverService) Update(ctx context.Context, id uuid.UUID, in UpdateDriverInput, actors ...ActorContext) (*Driver, error) {
	existing, err := s.store.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, ErrDriverNotFound) {
			return nil, ErrDriverNotFound
		}
		return nil, fmt.Errorf("failed to get driver: %w", err)
	}
	before := existing.AuditSnapshot()

	if in.Name != nil {
		name := strings.TrimSpace(*in.Name)
		if name == "" {
			return nil, ErrDriverNameRequired
		}
		existing.Name = name
		existing.NameNormalized = namenorm.Normalize(*in.Name)
	}
	if in.Email != nil {
		existing.Email = in.Email
	}
	if in.Region != nil {
		existing.Region = strings.TrimSpace(*in.Region)
	}
	if in.Status != nil {
		if *in.Status != "active" && *in.Status != "inactive" {
			return nil, ErrInvalidStatus
		}
		existing.Status = *in.Status
	}
	if in.LicenseClass != nil {
		licenseClass, err := normalizeLicenseClass(in.LicenseClass)
		if err != nil {
			return nil, err
		}
		existing.LicenseClass = licenseClass
	}
	if in.LicenseExpiryDate != nil {
		existing.LicenseExpiryDate = in.LicenseExpiryDate
	} else if in.ClearLicenseExpiryDate {
		existing.LicenseExpiryDate = nil
	}
	if in.Gender != nil {
		existing.Gender = in.Gender
	}
	if in.BirthDate != nil {
		existing.BirthDate = in.BirthDate
	} else if in.ClearBirthDate {
		existing.BirthDate = nil
	}
	if in.HasProfessionalLicense != nil {
		existing.HasProfessionalLicense = *in.HasProfessionalLicense
	}
	if in.EmploymentDate != nil {
		existing.EmploymentDate = in.EmploymentDate
	} else if in.ClearEmploymentDate {
		existing.EmploymentDate = nil
	}
	if in.HasTransferCert != nil {
		existing.HasTransferCert = *in.HasTransferCert
	}
	if in.InspectionDate != nil {
		existing.InspectionDate = in.InspectionDate
	} else if in.ClearInspectionDate {
		existing.InspectionDate = nil
	}
	if in.Remarks != nil {
		existing.Remarks = in.Remarks
	}

	if err := s.store.Update(ctx, existing); err != nil {
		return nil, err
	}
	writeAuditBestEffort(ctx, s.auditRepo, actorOrEmpty(actors), "update", "drivers", id, before, existing.AuditSnapshot())
	return existing, nil
}

// Reveal 解密司機身分證明碼；高風險揭露必須先成功寫入稽核紀錄。
func (s *DriverService) Reveal(ctx context.Context, id, actorID uuid.UUID, actorRole, ip, ua string) (string, error) {
	d, err := s.store.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, ErrDriverNotFound) {
			return "", ErrDriverNotFound
		}
		return "", fmt.Errorf("failed to get driver: %w", err)
	}
	if d == nil {
		return "", ErrDriverNotFound
	}
	if len(d.NationalIDCipher) == 0 {
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
		EntityType: "drivers",
		EntityID:   &entityIDStr,
		IPAddress:  &ip,
		UserAgent:  &ua,
	}); err != nil {
		return "", fmt.Errorf("%w: %v", ErrRevealAuditUnavailable, err)
	}
	return crypto.Decrypt(d.NationalIDCipher, s.cfg.EncryptionKey)
}

// AssignVehicleInput 代表指派司機車輛所需之輸入。
type AssignVehicleInput struct {
	VehicleID     uuid.UUID
	EffectiveFrom time.Time
	EffectiveTo   *time.Time
}

// AssignVehicle 建立司機與車輛之指派期間。
func (s *DriverService) AssignVehicle(ctx context.Context, driverID uuid.UUID, in AssignVehicleInput, actors ...ActorContext) (*DriverAssignment, error) {
	if driverID == uuid.Nil || in.VehicleID == uuid.Nil || in.EffectiveFrom.IsZero() ||
		(in.EffectiveTo != nil && !in.EffectiveTo.After(in.EffectiveFrom)) {
		return nil, ErrInvalidAssignmentRange
	}
	assignment := &DriverAssignment{
		DriverID:      driverID,
		VehicleID:     in.VehicleID,
		EffectiveFrom: in.EffectiveFrom,
		EffectiveTo:   in.EffectiveTo,
	}
	if err := s.store.AssignVehicle(ctx, assignment); err != nil {
		return nil, err
	}
	writeAuditBestEffort(ctx, s.auditRepo, actorOrEmpty(actors), "assign_vehicle", "driver_assignments", assignment.ID, nil, assignment.AuditSnapshot())
	return assignment, nil
}

// Delete 軟刪除司機並收斂其生效中車輛指派。
func (s *DriverService) Delete(ctx context.Context, id, actorID uuid.UUID, actorRole string, actors ...ActorContext) error {
	before, err := s.store.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, ErrDriverNotFound) {
			return ErrDriverNotFound
		}
		return fmt.Errorf("failed to get driver: %w", err)
	}
	if before == nil {
		return ErrDriverNotFound
	}
	beforeSnapshot := before.AuditSnapshot()

	actor := actorOrEmpty(actors)
	if actor.ActorID == uuid.Nil {
		actor.ActorID = actorID
	}
	if actor.ActorRole == "" {
		actor.ActorRole = actorRole
	}
	deleteFn := func(txCtx context.Context) error {
		ok, err := s.store.SoftDelete(txCtx, id, actorID)
		if err != nil {
			return fmt.Errorf("failed to soft delete driver: %w", err)
		}
		if !ok {
			return ErrDriverNotFound
		}

		if err := s.store.CloseActiveAssignments(txCtx, id); err != nil {
			return fmt.Errorf("failed to close active assignments: %w", err)
		}

		if s.auditRepo != nil {
			entityIDStr := id.String()
			if err := s.auditRepo.Write(txCtx, AuditEntry{
				ActorID:    &actor.ActorID,
				ActorRole:  &actor.ActorRole,
				Action:     "delete",
				EntityType: "drivers",
				EntityID:   &entityIDStr,
				BeforeData: beforeSnapshot,
				IPAddress:  &actor.IPAddress,
				UserAgent:  &actor.UserAgent,
			}); err != nil {
				slog.Error("driver delete audit write failed", "entity_id", id.String(), "error", err)
				return fmt.Errorf("failed to write driver audit: %w", err)
			}
		}
		return nil
	}

	if s.txRunner != nil {
		return s.txRunner.WithTx(ctx, deleteFn)
	}
	return deleteFn(ctx)
}
