package app

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// ErrVehicleInUse 表示車輛仍有生效中之司機指派或排班趟次綁定，不可刪除。
var ErrVehicleInUse = errors.New("vehicle is still in use")

// VehicleService 封裝車輛主檔業務邏輯，包含車輛目前掛載的司機。
type VehicleService struct {
	store     VehicleStore
	drivers   DriverStore
	auditRepo AuditWriter
}

// NewVehicleService 建立 VehicleService 實例。
func NewVehicleService(store VehicleStore, drivers DriverStore, auditRepo AuditWriter) *VehicleService {
	return &VehicleService{store: store, drivers: drivers, auditRepo: auditRepo}
}

// List 查詢車輛清單，並帶出每台車今日生效的司機。
func (s *VehicleService) List(ctx context.Context, filter VehicleFilter, page, pageSize int) ([]Vehicle, int64, error) {
	list, total, err := s.store.List(ctx, filter, page, pageSize)
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

// VehicleInput 是新增與更新車輛共用的輸入。Region 不在其中：車輛的區域一律由所屬單位帶出。
type VehicleInput struct {
	PlateNo                   string
	DisplayName               string
	SiteID                    *uuid.UUID
	Brand                     string
	Model                     string
	ManufactureYM             string
	CompulsoryInsuranceExpiry *time.Time
	PassengerInsuranceExpiry  *time.Time
	ThirdPartyInsuranceExpiry *time.Time
	LastInspectionDate        *time.Time
	WheelchairAccessible      *bool
	Status                    string
}

func (in VehicleInput) apply(v *Vehicle) {
	v.PlateNo = in.PlateNo
	v.DisplayName = in.DisplayName
	v.SiteID = in.SiteID
	v.Brand = in.Brand
	v.Model = in.Model
	v.ManufactureYM = in.ManufactureYM
	v.CompulsoryInsuranceExpiry = in.CompulsoryInsuranceExpiry
	v.PassengerInsuranceExpiry = in.PassengerInsuranceExpiry
	v.ThirdPartyInsuranceExpiry = in.ThirdPartyInsuranceExpiry
	v.LastInspectionDate = in.LastInspectionDate
	v.WheelchairAccessible = in.WheelchairAccessible
	v.Status = in.Status
}

// Create 新增車輛。
func (s *VehicleService) Create(ctx context.Context, in VehicleInput) (*Vehicle, error) {
	v := Vehicle{ID: uuid.New()}
	in.apply(&v)
	if err := s.store.Create(ctx, &v); err != nil {
		return nil, err
	}
	return &v, nil
}

// Update 更新車輛。
func (s *VehicleService) Update(ctx context.Context, id uuid.UUID, in VehicleInput) (*Vehicle, error) {
	v := Vehicle{ID: id}
	in.apply(&v)
	if err := s.store.Update(ctx, &v); err != nil {
		return nil, err
	}
	return &v, nil
}

// Delete 軟刪除車輛；仍有生效中司機指派或排班趟次綁定時回 ErrVehicleInUse。
func (s *VehicleService) Delete(ctx context.Context, id, actorID uuid.UUID, actorRole string) error {
	assignments, err := s.store.CountActiveDriverAssignments(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to count active driver assignments: %w", err)
	}
	legs, err := s.store.CountScheduleLegs(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to count schedule legs: %w", err)
	}
	if assignments > 0 || legs > 0 {
		return ErrVehicleInUse
	}

	ok, err := s.store.SoftDelete(ctx, id, actorID)
	if err != nil {
		return fmt.Errorf("failed to soft delete vehicle: %w", err)
	}
	if !ok {
		return fmt.Errorf("vehicle not found")
	}

	if s.auditRepo != nil {
		entityIDStr := id.String()
		_ = s.auditRepo.Write(ctx, AuditEntry{
			ActorID:    &actorID,
			ActorRole:  &actorRole,
			Action:     "delete",
			EntityType: "vehicles",
			EntityID:   &entityIDStr,
		})
	}
	return nil
}
