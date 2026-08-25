package service

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/xuri/excelize/v2"
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
func (s *MaintenanceService) List(ctx context.Context, page, pageSize int, vehicleID *uuid.UUID, startDate, endDate *time.Time) ([]repository.MaintenanceLogEntity, int, error) {
	return s.maintenanceRepo.List(ctx, page, pageSize, vehicleID, startDate, endDate)
}

// Create 新增維修保養紀錄並記錄稽核留痕。
func (s *MaintenanceService) Create(ctx context.Context, item *repository.MaintenanceLogEntity, actorID *uuid.UUID, actorRole *string) error {
	if err := s.maintenanceRepo.Create(ctx, item); err != nil {
		return err
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
	return nil
}

// Update 修改維修保養紀錄。
func (s *MaintenanceService) Update(ctx context.Context, item *repository.MaintenanceLogEntity, actorID *uuid.UUID, actorRole *string) error {
	if err := s.maintenanceRepo.Update(ctx, item); err != nil {
		return err
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
	return nil
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

	f := excelize.NewFile()
	defer f.Close()

	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "#FFFFFF"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#1D5B79"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "#B0C4DE", Style: 1},
			{Type: "top", Color: "#B0C4DE", Style: 1},
			{Type: "bottom", Color: "#B0C4DE", Style: 1},
			{Type: "right", Color: "#B0C4DE", Style: 1},
		},
	})

	gridStyle, _ := f.NewStyle(&excelize.Style{
		Alignment: &excelize.Alignment{Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "#CCCCCC", Style: 1},
			{Type: "top", Color: "#CCCCCC", Style: 1},
			{Type: "bottom", Color: "#CCCCCC", Style: 1},
			{Type: "right", Color: "#CCCCCC", Style: 1},
		},
	})

	isFirst := true
	defaultSheet := f.GetSheetName(0)

	headers := []string{"日期", "車號", "里程數", "保養項目", "廠商", "金額", "備註", "簽名"}

	for _, v := range vehicles {
		sheetName := v.DisplayName
		if sheetName == "" {
			sheetName = v.PlateNo
		}

		if isFirst {
			f.SetSheetName(defaultSheet, sheetName)
			isFirst = false
		} else {
			f.NewSheet(sheetName)
		}

		// 表頭資訊
		f.SetCellValue(sheetName, "A1", fmt.Sprintf("長照交通接送 車輛定期維修保養紀錄表 (%s)", v.DisplayName))
		f.SetCellValue(sheetName, "A2", fmt.Sprintf("車牌號碼：%s", v.PlateNo))

		// 欄位標題
		for colIdx, h := range headers {
			cell, _ := excelize.CoordinatesToCellName(colIdx+1, 4)
			f.SetCellValue(sheetName, cell, h)
			f.SetCellStyle(sheetName, cell, cell, headerStyle)
		}

		// 預留 12 列空白手寫列（規格書 §8.3）
		for r := 5; r <= 16; r++ {
			f.SetCellValue(sheetName, fmt.Sprintf("B%d", r), v.PlateNo)
			for c := 1; c <= len(headers); c++ {
				cell, _ := excelize.CoordinatesToCellName(c, r)
				f.SetCellStyle(sheetName, cell, cell, gridStyle)
			}
			f.SetRowHeight(sheetName, r, 24)
		}

		f.SetColWidth(sheetName, "A", "A", 14)
		f.SetColWidth(sheetName, "B", "B", 14)
		f.SetColWidth(sheetName, "C", "C", 14)
		f.SetColWidth(sheetName, "D", "D", 26)
		f.SetColWidth(sheetName, "E", "E", 18)
		f.SetColWidth(sheetName, "F", "F", 12)
		f.SetColWidth(sheetName, "G", "G", 22)
		f.SetColWidth(sheetName, "H", "H", 14)
	}

	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		return nil, fmt.Errorf("failed to write blank maintenance excel: %w", err)
	}

	return buf.Bytes(), nil
}

func strPtr(s string) *string {
	return &s
}
