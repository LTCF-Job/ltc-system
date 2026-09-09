package app

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"ltc-system/apps/api/internal/domain/crypto"
	"ltc-system/apps/api/internal/domain/namenorm"
	"ltc-system/apps/api/internal/platform/clock"
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
func (s *DriverService) List(ctx context.Context, q, status string, page, pageSize int) ([]Driver, int64, error) {
	return s.store.List(ctx, q, status, page, pageSize)
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
	LicenseClass           *string
	LicenseExpiryDate      *time.Time
	Gender                 *string
	BirthDate              *time.Time
	HasProfessionalLicense bool
	EmploymentDate         *time.Time
	HasTransferCert        bool
	Remarks                *string
	VehicleID              *uuid.UUID
}

// Create 新增司機：驗證身分證檢查碼，寫入加密密文與 HMAC 索引。若有指定指派車輛，
// 於同一交易中建立指派紀錄。actors 是可選的稽核來源資訊。
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
		Status:           "active",

		LicenseClass:           licenseClass,
		LicenseExpiryDate:      in.LicenseExpiryDate,
		Gender:                 in.Gender,
		BirthDate:              in.BirthDate,
		HasProfessionalLicense: in.HasProfessionalLicense,
		EmploymentDate:         in.EmploymentDate,
		HasTransferCert:        in.HasTransferCert,
		Remarks:                in.Remarks,
	}

	var assignment *DriverAssignment
	if in.VehicleID != nil && *in.VehicleID != uuid.Nil {
		assignment = &DriverAssignment{
			DriverID:      d.ID,
			VehicleID:     *in.VehicleID,
			EffectiveFrom: clock.Today(),
		}
	}

	createFn := func(txCtx context.Context) error {
		if err := s.store.Create(txCtx, &d); err != nil {
			return err
		}
		if assignment != nil {
			if err := s.store.AssignVehicle(txCtx, assignment); err != nil {
				return err
			}
		}
		return nil
	}

	if s.txRunner != nil {
		err = s.txRunner.WithTx(ctx, createFn)
	} else {
		err = createFn(ctx)
	}
	if err != nil {
		return nil, err
	}

	writeAuditBestEffort(ctx, s.auditRepo, actorOrEmpty(actors), "create", "drivers", d.ID, nil, d.AuditSnapshot())
	if assignment != nil {
		writeAuditBestEffort(ctx, s.auditRepo, actorOrEmpty(actors), "assign_vehicle", "driver_assignments", assignment.ID, nil, assignment.AuditSnapshot())
	}
	return &d, nil
}

// UpdateDriverInput 代表更新司機基本資料所需之輸入，欄位為 nil 表示不變更。
type UpdateDriverInput struct {
	Name                   *string
	NationalID             *string
	Email                  *string
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
	if in.NationalID != nil {
		// 身分證改動要同步重算密文、HMAC 索引與遮罩值，三者必須一致，
		// 否則 Reveal 解出來的明碼會跟畫面顯示的遮罩對不上。
		nationalID := strings.ToUpper(strings.TrimSpace(*in.NationalID))
		if !crypto.ValidateNationalID(nationalID) {
			return nil, ErrInvalidDriverNationalID
		}
		if !bytes.Equal(crypto.Index(nationalID, s.cfg.HMACKey), existing.NationalIDHMAC) {
			cipherText, err := crypto.Encrypt(nationalID, s.cfg.EncryptionKey)
			if err != nil {
				return nil, fmt.Errorf("encrypt driver national id: %w", err)
			}
			existing.NationalIDCipher = cipherText
			existing.NationalIDHMAC = crypto.Index(nationalID, s.cfg.HMACKey)
			existing.NationalIDMasked = crypto.Mask(nationalID)
		}
	}
	if in.Email != nil {
		existing.Email = in.Email
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

// AssignVehicleInput 代表指派司機車輛所需之輸入。指派不再由使用者輸入期間：
// 一律自今日起生效、不設結束日，期間只作為 driver_assignments 的內部表示。
type AssignVehicleInput struct {
	VehicleID uuid.UUID
}

// AssignVehicle 指派司機目前的車輛。一位司機同期只會有一台車，因此改派前先把
// 生效中的舊指派收斂到今天，否則會撞上 driver_assignments 的不重疊排除約束。
func (s *DriverService) AssignVehicle(ctx context.Context, driverID uuid.UUID, in AssignVehicleInput, actors ...ActorContext) (*DriverAssignment, error) {
	if driverID == uuid.Nil || in.VehicleID == uuid.Nil {
		return nil, ErrInvalidAssignmentRange
	}
	assignment := &DriverAssignment{
		DriverID:      driverID,
		VehicleID:     in.VehicleID,
		EffectiveFrom: clock.Today(),
	}
	assignFn := func(txCtx context.Context) error {
		if err := s.store.CloseActiveAssignments(txCtx, driverID); err != nil {
			return fmt.Errorf("failed to close active assignments: %w", err)
		}
		return s.store.AssignVehicle(txCtx, assignment)
	}
	var err error
	if s.txRunner != nil {
		err = s.txRunner.WithTx(ctx, assignFn)
	} else {
		err = assignFn(ctx)
	}
	if err != nil {
		return nil, err
	}
	writeAuditBestEffort(ctx, s.auditRepo, actorOrEmpty(actors), "assign_vehicle", "driver_assignments", assignment.ID, nil, assignment.AuditSnapshot())
	return assignment, nil
}

// UnassignVehicle 解除司機目前生效中之車輛指派。
func (s *DriverService) UnassignVehicle(ctx context.Context, driverID uuid.UUID, actors ...ActorContext) error {
	if driverID == uuid.Nil {
		return ErrInvalidAssignmentRange
	}
	unassignFn := func(txCtx context.Context) error {
		return s.store.CloseActiveAssignments(txCtx, driverID)
	}
	var err error
	if s.txRunner != nil {
		err = s.txRunner.WithTx(ctx, unassignFn)
	} else {
		err = unassignFn(ctx)
	}
	if err != nil {
		return err
	}
	writeAuditBestEffort(ctx, s.auditRepo, actorOrEmpty(actors), "unassign_vehicle", "driver_assignments", driverID, nil, map[string]interface{}{"driverId": driverID})
	return nil
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
