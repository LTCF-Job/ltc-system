package app

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// VehicleService 封裝車輛主檔業務邏輯，包含車輛目前掛載的司機。
type VehicleService struct {
	store   VehicleStore
	drivers DriverStore
}

// NewVehicleService 建立 VehicleService 實例。
func NewVehicleService(store VehicleStore, drivers DriverStore) *VehicleService {
	return &VehicleService{store: store, drivers: drivers}
}

// List 查詢車輛清單，並帶出每台車今日生效的司機。
func (s *VehicleService) List(ctx context.Context, region, q string, page, pageSize int) ([]Vehicle, int64, error) {
	list, total, err := s.store.List(ctx, region, q, page, pageSize)
	if err != nil {
		return nil, 0, err
	}

	ids := make([]uuid.UUID, 0, len(list))
	for _, v := range list {
		ids = append(ids, v.ID)
	}
	byVehicle, err := s.drivers.ListByVehicleIDsOnDate(ctx, ids, time.Now())
	if err != nil {
		return nil, 0, err
	}
	for i := range list {
		for _, d := range byVehicle[list[i].ID] {
			list[i].Drivers = append(list[i].Drivers, VehicleDriver{ID: d.ID, Code: d.Code, Name: d.Name})
		}
	}
	return list, total, nil
}

// SetDrivers 以 effectiveFrom 為界，將車輛的司機集合整批換成 driverIDs。
func (s *VehicleService) SetDrivers(ctx context.Context, vehicleID uuid.UUID, driverIDs []uuid.UUID, effectiveFrom time.Time) error {
	return s.drivers.ReplaceVehicleDrivers(ctx, vehicleID, driverIDs, effectiveFrom)
}

// CreateVehicleInput 代表新增車輛所需之輸入。
type CreateVehicleInput struct {
	PlateNo     string
	DisplayName string
	Region      string
	Status      string
}

// Create 新增車輛。
func (s *VehicleService) Create(ctx context.Context, in CreateVehicleInput) (*Vehicle, error) {
	v := Vehicle{
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
func (s *VehicleService) Update(ctx context.Context, id uuid.UUID, in UpdateVehicleInput) (*Vehicle, error) {
	v := Vehicle{
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
