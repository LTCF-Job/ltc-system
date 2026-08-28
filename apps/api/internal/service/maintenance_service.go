package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"ltc-system/apps/api/internal/export"
	"ltc-system/apps/api/internal/repository"
)

// MaintenanceService 提供車輛維修保養管理與空白表產生服務。
type MaintenanceService struct {
	maintenanceRepo *repository.MaintenanceRepository
	vehicleRepo     *repository.VehicleRepository
	auditRepo       *repository.AuditRepository
}

// NewMaintenanceService 建立 MaintenanceService 實例。
func NewMaintenanceService(
	maintenanceRepo *repository.MaintenanceRepository,
	vehicleRepo *repository.VehicleRepository,
	auditRepo *repository.AuditRepository,
) *MaintenanceService {
	return &MaintenanceService{
		maintenanceRepo: maintenanceRepo,
		vehicleRepo:     vehicleRepo,
		auditRepo:       auditRepo,
	}
}

// List 查詢維修保養紀錄清單。
func (s *MaintenanceService) List(ctx context.Context, page, pageSize int, vehicleID *uuid.UUID, startDate, endDate *time.Time, q string) ([]repository.MaintenanceLogEntity, int, error) {
	return s.maintenanceRepo.List(ctx, page, pageSize, vehicleID, startDate, endDate, q)
}

// Create 新增維修保養紀錄並記錄稽核留痕。
// MaintenanceLogInput 代表新增或修改維修保養紀錄所需之輸入。
type MaintenanceLogInput struct {
	VehicleID   uuid.UUID
	ServiceDate time.Time
	Mileage     float64
	Items       string
	Vendor      *string
	Cost        float64
	ReceiptURL  *string
	Note        *string
	CreatedBy   uuid.UUID
}

func (s *MaintenanceService) Create(ctx context.Context, in MaintenanceLogInput, actorID *uuid.UUID, actorRole *string) (*repository.MaintenanceLogEntity, error) {
	item := &repository.MaintenanceLogEntity{
		VehicleID:   in.VehicleID,
		ServiceDate: in.ServiceDate,
		Mileage:     in.Mileage,
		Items:       in.Items,
		Vendor:      in.Vendor,
		Cost:        in.Cost,
		ReceiptURL:  in.ReceiptURL,
		Note:        in.Note,
		CreatedBy:   in.CreatedBy,
	}
	if err := s.maintenanceRepo.Create(ctx, item); err != nil {
		return nil, err
	}

	if s.auditRepo != nil {
		_ = s.auditRepo.Insert(ctx, &repository.AuditLogEntity{
			ActorID:    actorID,
			ActorRole:  actorRole,
			Action:     "create",
			EntityType: "maintenance_logs",
			EntityID:   strPtr(item.ID.String()),
			AfterData:  item,
			CreatedAt:  time.Now(),
		})
	}
	return item, nil
}

// Update 修改維修保養紀錄。
func (s *MaintenanceService) Update(ctx context.Context, id uuid.UUID, in MaintenanceLogInput, actorID *uuid.UUID, actorRole *string) (*repository.MaintenanceLogEntity, error) {
	item := &repository.MaintenanceLogEntity{
		ID:          id,
		VehicleID:   in.VehicleID,
		ServiceDate: in.ServiceDate,
		Mileage:     in.Mileage,
		Items:       in.Items,
		Vendor:      in.Vendor,
		Cost:        in.Cost,
		ReceiptURL:  in.ReceiptURL,
		Note:        in.Note,
	}
	if err := s.maintenanceRepo.Update(ctx, item); err != nil {
		return nil, err
	}

	if s.auditRepo != nil {
		_ = s.auditRepo.Insert(ctx, &repository.AuditLogEntity{
			ActorID:    actorID,
			ActorRole:  actorRole,
			Action:     "update",
			EntityType: "maintenance_logs",
			EntityID:   strPtr(item.ID.String()),
			AfterData:  item,
			CreatedAt:  time.Now(),
		})
	}
	return item, nil
}

// Delete 刪除維修保養紀錄。
func (s *MaintenanceService) Delete(ctx context.Context, id uuid.UUID, actorID *uuid.UUID, actorRole *string) error {
	if err := s.maintenanceRepo.Delete(ctx, id); err != nil {
		return err
	}

	if s.auditRepo != nil {
		_ = s.auditRepo.Insert(ctx, &repository.AuditLogEntity{
			ActorID:    actorID,
			ActorRole:  actorRole,
			Action:     "delete",
			EntityType: "maintenance_logs",
			EntityID:   strPtr(id.String()),
			CreatedAt:  time.Now(),
		})
	}
	return nil
}

// GenerateBlankMaintenanceExcel 產生符合規格書 §8.3 的 15 車空白維修保養檢查表格。
func (s *MaintenanceService) GenerateBlankMaintenanceExcel(ctx context.Context) ([]byte, error) {
	var vehicles []repository.VehicleEntity
	if s.vehicleRepo != nil {
		vList, _, _ := s.vehicleRepo.List(ctx, "", "", 1, 100)
		vehicles = vList
	}
	if len(vehicles) == 0 {
		vehicles = []repository.VehicleEntity{
			{DisplayName: "竹北一車", PlateNo: "BZG-7915"},
			{DisplayName: "竹北二車", PlateNo: "BZG-7916"},
			{DisplayName: "竹北三車", PlateNo: "BZG-7917"},
			{DisplayName: "竹南一車", PlateNo: "BZG-8801"},
			{DisplayName: "竹南二車", PlateNo: "BZG-8802"},
		}
	}

	labels := make([]export.MaintenanceVehicleLabel, len(vehicles))
	for i, v := range vehicles {
		labels[i] = export.MaintenanceVehicleLabel{DisplayName: v.DisplayName, PlateNo: v.PlateNo}
	}

	return export.GenerateBlankMaintenanceExcel(labels)
}

func strPtr(s string) *string {
	return &s
}
