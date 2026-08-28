package service

import (
	"context"

	"github.com/google/uuid"
	"ltc-system/apps/api/internal/repository"
)

// VehicleStore 定義車輛主檔存取邊界。
type VehicleStore interface {
	List(ctx context.Context, region, q string, page, pageSize int) ([]repository.VehicleEntity, int64, error)
	Create(ctx context.Context, v *repository.VehicleEntity) error
	Update(ctx context.Context, v *repository.VehicleEntity) error
}

// VehicleService 封裝車輛主檔業務邏輯。
type VehicleService struct {
	store VehicleStore
}

// NewVehicleService 建立 VehicleService 實例。
func NewVehicleService(store VehicleStore) *VehicleService {
	return &VehicleService{store: store}
}

// List 查詢車輛清單。
func (s *VehicleService) List(ctx context.Context, region, q string, page, pageSize int) ([]repository.VehicleEntity, int64, error) {
	return s.store.List(ctx, region, q, page, pageSize)
}

// CreateVehicleInput 代表新增車輛所需之輸入。
type CreateVehicleInput struct {
	PlateNo     string
	DisplayName string
	Region      string
	Status      string
}

// Create 新增車輛。
func (s *VehicleService) Create(ctx context.Context, in CreateVehicleInput) (*repository.VehicleEntity, error) {
	v := repository.VehicleEntity{
		ID:          uuid.New(),
		PlateNo:     in.PlateNo,
		DisplayName: in.DisplayName,
		Region:      in.Region,
		Status:      in.Status,
	}
	if err := s.store.Create(ctx, &v); err != nil {
		return nil, err
	}
	return &v, nil
}

// UpdateVehicleInput 代表更新車輛所需之輸入。
type UpdateVehicleInput struct {
	PlateNo     string
	DisplayName string
	Region      string
	Status      string
}

// Update 更新車輛。
func (s *VehicleService) Update(ctx context.Context, id uuid.UUID, in UpdateVehicleInput) (*repository.VehicleEntity, error) {
	v := repository.VehicleEntity{
		ID:          id,
		PlateNo:     in.PlateNo,
		DisplayName: in.DisplayName,
		Region:      in.Region,
		Status:      in.Status,
	}
	if err := s.store.Update(ctx, &v); err != nil {
		return nil, err
	}
	return &v, nil
}
