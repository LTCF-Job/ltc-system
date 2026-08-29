package app

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// FuelService 提供車輛油資登錄與管理服務。
type FuelService struct {
	fuelRepo  FuelStore
	auditRepo AuditWriter
}

// NewFuelService 建立 FuelService 實例。
func NewFuelService(
	fuelRepo FuelStore,
	auditRepo AuditWriter,
) *FuelService {
	return &FuelService{
		fuelRepo:  fuelRepo,
		auditRepo: auditRepo,
	}
}

// List 查詢油資紀錄清單。
func (s *FuelService) List(ctx context.Context, page, pageSize int, vehicleID, driverID *uuid.UUID, startDate, endDate *time.Time, q string) ([]FuelLog, int, error) {
	return s.fuelRepo.List(ctx, page, pageSize, vehicleID, driverID, startDate, endDate, q)
}

// FuelLogInput 代表新增或修改油資紀錄所需之輸入。
type FuelLogInput struct {
	VehicleID  uuid.UUID
	DriverID   *uuid.UUID
	FuelDate   time.Time
	Liters     float64
	Cost       float64
	ReceiptURL *string
	CreatedBy  uuid.UUID
}

// Create 新增油資紀錄並寫入稽核日誌。
func (s *FuelService) Create(ctx context.Context, in FuelLogInput, actorID *uuid.UUID, actorRole *string) (*FuelLog, error) {
	item := &FuelLog{
		VehicleID:  in.VehicleID,
		DriverID:   in.DriverID,
		FuelDate:   in.FuelDate,
		Liters:     in.Liters,
		Cost:       in.Cost,
		ReceiptURL: in.ReceiptURL,
		CreatedBy:  in.CreatedBy,
	}
	if err := s.fuelRepo.Create(ctx, item); err != nil {
		return nil, err
	}

	if s.auditRepo != nil {
		_ = s.auditRepo.Write(ctx, AuditEntry{
			ActorID:    actorID,
			ActorRole:  actorRole,
			Action:     "create",
			EntityType: "fuel_logs",
			EntityID:   strPtr(item.ID.String()),
			AfterData:  item,
		})
	}
	return item, nil
}

// Update 修改油資紀錄。
func (s *FuelService) Update(ctx context.Context, id uuid.UUID, in FuelLogInput, actorID *uuid.UUID, actorRole *string) (*FuelLog, error) {
	item := &FuelLog{
		ID:         id,
		VehicleID:  in.VehicleID,
		DriverID:   in.DriverID,
		FuelDate:   in.FuelDate,
		Liters:     in.Liters,
		Cost:       in.Cost,
		ReceiptURL: in.ReceiptURL,
	}
	if err := s.fuelRepo.Update(ctx, item); err != nil {
		return nil, err
	}

	if s.auditRepo != nil {
		_ = s.auditRepo.Write(ctx, AuditEntry{
			ActorID:    actorID,
			ActorRole:  actorRole,
			Action:     "update",
			EntityType: "fuel_logs",
			EntityID:   strPtr(item.ID.String()),
			AfterData:  item,
		})
	}
	return item, nil
}

// Delete 刪除油資紀錄。
func (s *FuelService) Delete(ctx context.Context, id uuid.UUID, actorID *uuid.UUID, actorRole *string) error {
	if err := s.fuelRepo.Delete(ctx, id); err != nil {
		return err
	}

	if s.auditRepo != nil {
		_ = s.auditRepo.Write(ctx, AuditEntry{
			ActorID:    actorID,
			ActorRole:  actorRole,
			Action:     "delete",
			EntityType: "fuel_logs",
			EntityID:   strPtr(id.String()),
		})
	}
	return nil
}
