package app

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestMaintenanceService_CRUD(t *testing.T) {
	maintenanceRepo := stubMaintenanceStore{}
	vehicleRepo := emptyVehicleLister{}
	auditRepo := discardAuditWriter{}

	svc := NewMaintenanceService(maintenanceRepo, vehicleRepo, auditRepo, stubTemplateRenderer{})
	ctx := context.Background()

	in := MaintenanceLogInput{
		VehicleID:   uuid.New(),
		ServiceDate: time.Now(),
		Mileage:     52000.5,
		Items:       "更換機油、機油濾清器、檢查胎壓",
		Vendor:      strPtr("順益汽車保養廠"),
		Cost:        3500.0,
		CreatedBy:   uuid.New(),
	}

	item, err := svc.Create(ctx, in, nil, nil)
	assert.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, item.ID)

	item, err = svc.Update(ctx, item.ID, in, nil, nil)
	assert.NoError(t, err)

	err = svc.Delete(ctx, item.ID, nil, nil)
	assert.NoError(t, err)
}
