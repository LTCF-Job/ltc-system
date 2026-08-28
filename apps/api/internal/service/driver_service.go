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
	// ErrDriverNotFound 代表查無司機資料。
	ErrDriverNotFound = errors.New("driver not found")
	// ErrInvalidDriverNationalID 代表司機身分證檢查碼錯誤。
	ErrInvalidDriverNationalID = errors.New("invalid driver national id format")
)

// DriverStore 定義司機主檔存取邊界。
type DriverStore interface {
	List(ctx context.Context, region, q string, page, pageSize int) ([]repository.DriverEntity, int64, error)
	GetByID(ctx context.Context, id uuid.UUID) (*repository.DriverEntity, error)
	Create(ctx context.Context, d *repository.DriverEntity) error
	Update(ctx context.Context, d *repository.DriverEntity) error
	AssignVehicle(ctx context.Context, a *repository.DriverAssignmentEntity) error
}

// DriverService 封裝司機主檔業務邏輯：身分證加密、HMAC 索引、姓名正規化與車輛指派。
type DriverService struct {
	store DriverStore
	cfg   *config.Config
}

// NewDriverService 建立 DriverService 實例。
func NewDriverService(store DriverStore, cfg *config.Config) *DriverService {
	return &DriverService{store: store, cfg: cfg}
}

// List 查詢司機清單。
func (s *DriverService) List(ctx context.Context, region, q string, page, pageSize int) ([]repository.DriverEntity, int64, error) {
	return s.store.List(ctx, region, q, page, pageSize)
}

// CreateDriverInput 代表新增司機所需之輸入。
type CreateDriverInput struct {
	Code       string
	Name       string
	NationalID string
	Email      *string
	Region     string
}

// Create 新增司機：驗證身分證檢查碼，寫入加密密文與 HMAC 索引。
func (s *DriverService) Create(ctx context.Context, in CreateDriverInput) (*repository.DriverEntity, error) {
	nationalID := strings.ToUpper(strings.TrimSpace(in.NationalID))
	if !crypto.ValidateNationalID(nationalID) {
		return nil, ErrInvalidDriverNationalID
	}

	hmacIdx := crypto.Index(nationalID, s.cfg.HMACKey)
	cipherText, err := crypto.Encrypt(nationalID, s.cfg.EncryptionKey)
	if err != nil {
		return nil, fmt.Errorf("encrypt driver national id: %w", err)
	}

	d := repository.DriverEntity{
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
	}

	if err := s.store.Create(ctx, &d); err != nil {
		return nil, err
	}
	return &d, nil
}

// UpdateDriverInput 代表更新司機基本資料所需之輸入，欄位為 nil 表示不變更。
type UpdateDriverInput struct {
	Name   *string
	Email  *string
	Region *string
	Status *string
}

// Update 更新司機基本資料。
func (s *DriverService) Update(ctx context.Context, id uuid.UUID, in UpdateDriverInput) (*repository.DriverEntity, error) {
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
	if in.Status != nil {
		existing.Status = *in.Status
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
	IsPrimary     bool
	EffectiveFrom time.Time
	EffectiveTo   *time.Time
}

// AssignVehicle 建立司機與車輛之指派期間。
func (s *DriverService) AssignVehicle(ctx context.Context, driverID uuid.UUID, in AssignVehicleInput) (*repository.DriverAssignmentEntity, error) {
	assignment := &repository.DriverAssignmentEntity{
		DriverID:      driverID,
		VehicleID:     in.VehicleID,
		IsPrimary:     in.IsPrimary,
		EffectiveFrom: in.EffectiveFrom,
		EffectiveTo:   in.EffectiveTo,
	}
	if err := s.store.AssignVehicle(ctx, assignment); err != nil {
		return nil, err
	}
	return assignment, nil
}
