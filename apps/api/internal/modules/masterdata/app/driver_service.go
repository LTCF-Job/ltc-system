package app

import (
	"context"
	"fmt"
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
}

// NewDriverService 建立 DriverService 實例。
func NewDriverService(store DriverStore, cfg *config.Config, auditRepo AuditWriter) *DriverService {
	return &DriverService{store: store, cfg: cfg, auditRepo: auditRepo}
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
	Code              string
	Name              string
	NationalID        string
	Email             *string
	Region            string
	LicenseClass      *string
	LicenseExpiryDate *time.Time
}

// Create 新增司機：驗證身分證檢查碼，寫入加密密文與 HMAC 索引。
func (s *DriverService) Create(ctx context.Context, in CreateDriverInput) (*Driver, error) {
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
		Code:             in.Code,
		Name:             in.Name,
		NameNormalized:   namenorm.Normalize(in.Name),
		NationalIDCipher: cipherText,
		NationalIDHMAC:   hmacIdx,
		NationalIDMasked: crypto.Mask(nationalID),
		Email:            in.Email,
		Region:           in.Region,
		Status:           "active",

		LicenseClass:      licenseClass,
		LicenseExpiryDate: in.LicenseExpiryDate,
	}

	if err := s.store.Create(ctx, &d); err != nil {
		return nil, err
	}
	return &d, nil
}

// UpdateDriverInput 代表更新司機基本資料所需之輸入，欄位為 nil 表示不變更。
type UpdateDriverInput struct {
	Name              *string
	Email             *string
	Region            *string
	Status            *string
	LicenseClass      *string
	LicenseExpiryDate *time.Time
	// ClearLicenseExpiryDate 為 true 時把駕照有效日期清空；沒有這個旗標無法區分「不變更」與「清空」。
	ClearLicenseExpiryDate bool
}

// Update 更新司機基本資料。
func (s *DriverService) Update(ctx context.Context, id uuid.UUID, in UpdateDriverInput) (*Driver, error) {
	existing, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, ErrDriverNotFound
	}

	if in.Name != nil {
		existing.Name = *in.Name
		existing.NameNormalized = namenorm.Normalize(*in.Name)
	}
	if in.Email != nil {
		existing.Email = in.Email
	}
	if in.Region != nil {
		existing.Region = *in.Region
	}
	// 只接受兩種狀態，非法值一律保留原本的值不變更
	if in.Status != nil && (*in.Status == "active" || *in.Status == "inactive") {
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

	if err := s.store.Update(ctx, existing); err != nil {
		return nil, err
	}
	return existing, nil
}

// Reveal 解密司機身分證明碼。呼叫端負責寫入稽核紀錄。
func (s *DriverService) Reveal(ctx context.Context, id uuid.UUID) (string, error) {
	d, err := s.store.GetByID(ctx, id)
	if err != nil {
		return "", ErrDriverNotFound
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
func (s *DriverService) AssignVehicle(ctx context.Context, driverID uuid.UUID, in AssignVehicleInput) (*DriverAssignment, error) {
	assignment := &DriverAssignment{
		DriverID:      driverID,
		VehicleID:     in.VehicleID,
		EffectiveFrom: in.EffectiveFrom,
		EffectiveTo:   in.EffectiveTo,
	}
	if err := s.store.AssignVehicle(ctx, assignment); err != nil {
		return nil, err
	}
	return assignment, nil
}

// Delete 軟刪除司機並收斂其生效中車輛指派。
func (s *DriverService) Delete(ctx context.Context, id, actorID uuid.UUID, actorRole string) error {
	ok, err := s.store.SoftDelete(ctx, id, actorID)
	if err != nil {
		return fmt.Errorf("failed to soft delete driver: %w", err)
	}
	if !ok {
		return ErrDriverNotFound
	}

	if err := s.store.CloseActiveAssignments(ctx, id); err != nil {
		return fmt.Errorf("failed to close active assignments: %w", err)
	}

	if s.auditRepo != nil {
		entityIDStr := id.String()
		_ = s.auditRepo.Write(ctx, AuditEntry{
			ActorID:    &actorID,
			ActorRole:  &actorRole,
			Action:     "delete",
			EntityType: "drivers",
			EntityID:   &entityIDStr,
		})
	}
	return nil
}
