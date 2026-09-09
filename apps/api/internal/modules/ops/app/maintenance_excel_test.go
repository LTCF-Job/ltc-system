package app_test

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/xuri/excelize/v2"
	"ltc-system/apps/api/internal/modules/ops/app"
	"ltc-system/apps/api/internal/modules/ops/infra"
)

// emptyVehicleLister 與 discardAuditWriter 在外部測試套件中重新宣告，因為內部
// 套件的 test double 不對外可見。
type emptyVehicleLister struct{}

func (emptyVehicleLister) List(context.Context, string, string, int, int) ([]app.VehicleRef, int64, error) {
	return nil, 0, nil
}

type discardAuditWriter struct{}

func (discardAuditWriter) Write(context.Context, app.AuditEntry) error { return nil }

type stubMaintenanceStore struct{}

func (stubMaintenanceStore) List(context.Context, int, int, *uuid.UUID, *time.Time, *time.Time, string) ([]app.MaintenanceLog, int, error) {
	return nil, 0, nil
}
func (stubMaintenanceStore) Create(context.Context, *app.MaintenanceLog) error { return nil }
func (stubMaintenanceStore) Update(context.Context, *app.MaintenanceLog) error { return nil }
func (stubMaintenanceStore) Delete(context.Context, uuid.UUID) error           { return nil }

func TestMaintenanceService_GenerateBlankMaintenanceExcel(t *testing.T) {
	maintenanceRepo := stubMaintenanceStore{}
	vehicleRepo := emptyVehicleLister{}
	auditRepo := discardAuditWriter{}

	svc := app.NewMaintenanceService(maintenanceRepo, vehicleRepo, auditRepo, infra.NewExcelRenderer())

	ctx := context.Background()
	excelBytes, err := svc.GenerateBlankMaintenanceExcel(ctx)
	assert.NoError(t, err)
	assert.NotEmpty(t, excelBytes)

	f, err := excelize.OpenReader(bytes.NewReader(excelBytes))
	assert.NoError(t, err)
	defer f.Close()

	sheetList := f.GetSheetList()
	assert.NotEmpty(t, sheetList)
}
