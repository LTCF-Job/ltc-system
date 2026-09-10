package app

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

var ErrInvalidFuelPagination = errors.New("fuel pagination must be positive")

// FuelService 提供車輛油資登記與管理服務。
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
	if page < 1 || pageSize < 1 {
		return nil, 0, ErrInvalidFuelPagination
	}
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
func (s *FuelService) Create(ctx context.Context, in FuelLogInput, actorID *uuid.UUID, actorRole *string, auditContexts ...AuditContext) (*FuelLog, error) {
	receiptURL, err := normalizeReceiptURL(in.ReceiptURL)
	if err != nil {
		return nil, err
	}
	item := &FuelLog{
		VehicleID:  in.VehicleID,
		DriverID:   in.DriverID,
		FuelDate:   in.FuelDate,
		Liters:     in.Liters,
		Cost:       in.Cost,
		ReceiptURL: receiptURL,
		CreatedBy:  in.CreatedBy,
	}
	if err := s.fuelRepo.Create(ctx, item); err != nil {
		return nil, err
	}

	writeAuditBestEffort(ctx, s.auditRepo, actorID, actorRole, auditContextOrEmpty(auditContexts), "create", "fuel_logs", item.ID, nil, item.AuditSnapshot())
	return item, nil
}

// Update 修改油資紀錄。
func (s *FuelService) Update(ctx context.Context, id uuid.UUID, in FuelLogInput, actorID *uuid.UUID, actorRole *string, auditContexts ...AuditContext) (*FuelLog, error) {
	var before interface{}
	if s.auditRepo != nil {
		var err error
		before, err = loadFuelAuditSnapshot(ctx, s.fuelRepo, id)
		if err != nil {
			return nil, err
		}
	}
	receiptURL, err := normalizeReceiptURL(in.ReceiptURL)
	if err != nil {
		return nil, err
	}
	item := &FuelLog{
		ID:         id,
		VehicleID:  in.VehicleID,
		DriverID:   in.DriverID,
		FuelDate:   in.FuelDate,
		Liters:     in.Liters,
		Cost:       in.Cost,
		ReceiptURL: receiptURL,
	}
	if err := s.fuelRepo.Update(ctx, item); err != nil {
		return nil, err
	}

	writeAuditBestEffort(ctx, s.auditRepo, actorID, actorRole, auditContextOrEmpty(auditContexts), "update", "fuel_logs", item.ID, before, item.AuditSnapshot())
	return item, nil
}

// Delete 刪除油資紀錄。
func (s *FuelService) Delete(ctx context.Context, id uuid.UUID, actorID *uuid.UUID, actorRole *string, auditContexts ...AuditContext) error {
	var before interface{}
	if s.auditRepo != nil {
		var err error
		before, err = loadFuelAuditSnapshot(ctx, s.fuelRepo, id)
		if err != nil {
			return err
		}
	}
	if err := s.fuelRepo.Delete(ctx, id); err != nil {
		return err
	}

	writeAuditBestEffort(ctx, s.auditRepo, actorID, actorRole, auditContextOrEmpty(auditContexts), "delete", "fuel_logs", id, before, nil)
	return nil
}
