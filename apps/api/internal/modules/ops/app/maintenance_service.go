package app

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// MaintenanceService 提供車輛維修保養管理與空白表產生服務。
type MaintenanceService struct {
	maintenanceRepo MaintenanceStore
	vehicleRepo     VehicleLister
	auditRepo       AuditWriter
	renderer        MaintenanceTemplateRenderer
}

// NewMaintenanceService 建立 MaintenanceService 實例。
func NewMaintenanceService(
	maintenanceRepo MaintenanceStore,
	vehicleRepo VehicleLister,
	auditRepo AuditWriter,
	renderer MaintenanceTemplateRenderer,
) *MaintenanceService {
	return &MaintenanceService{
		maintenanceRepo: maintenanceRepo,
		vehicleRepo:     vehicleRepo,
		auditRepo:       auditRepo,
		renderer:        renderer,
	}
}

// List 查詢維修保養紀錄清單。
func (s *MaintenanceService) List(ctx context.Context, page, pageSize int, vehicleID *uuid.UUID, startDate, endDate *time.Time, q string) ([]MaintenanceLog, int, error) {
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

func (s *MaintenanceService) Create(ctx context.Context, in MaintenanceLogInput, actorID *uuid.UUID, actorRole *string) (*MaintenanceLog, error) {
	item := &MaintenanceLog{
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
		_ = s.auditRepo.Write(ctx, AuditEntry{
			ActorID:    actorID,
			ActorRole:  actorRole,
			Action:     "create",
			EntityType: "maintenance_logs",
			EntityID:   strPtr(item.ID.String()),
			AfterData:  item,
		})
	}
	return item, nil
}

// Update 修改維修保養紀錄。
func (s *MaintenanceService) Update(ctx context.Context, id uuid.UUID, in MaintenanceLogInput, actorID *uuid.UUID, actorRole *string) (*MaintenanceLog, error) {
	item := &MaintenanceLog{
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
		_ = s.auditRepo.Write(ctx, AuditEntry{
			ActorID:    actorID,
			ActorRole:  actorRole,
			Action:     "update",
			EntityType: "maintenance_logs",
			EntityID:   strPtr(item.ID.String()),
			AfterData:  item,
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
		_ = s.auditRepo.Write(ctx, AuditEntry{
			ActorID:    actorID,
			ActorRole:  actorRole,
			Action:     "delete",
			EntityType: "maintenance_logs",
			EntityID:   strPtr(id.String()),
		})
	}
	return nil
}

// GenerateBlankMaintenanceExcel 產生符合規格書 §8.3 的 15 車空白維修保養檢查表格。
func (s *MaintenanceService) GenerateBlankMaintenanceExcel(ctx context.Context) ([]byte, error) {
	// 空白表格上的車輛一律取自車輛主檔。先前在查無車輛時填入五組寫死的車號，
	// 會讓人拿到一份印著不存在車輛的正式表單；查無資料時就產生沒有車輛列的空表。
	vehicles, _, err := s.vehicleRepo.List(ctx, "", "", 1, 100)
	if err != nil {
		return nil, err
	}

	labels := make([]VehicleLabel, len(vehicles))
	for i, v := range vehicles {
		labels[i] = VehicleLabel{DisplayName: v.DisplayName, PlateNo: v.PlateNo}
	}

	return s.renderer.RenderBlankMaintenanceTemplate(labels)
}

func strPtr(s string) *string {
	return &s
}
