package app

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestMaintenanceService_ReceiptURL(t *testing.T) {
	tests := []struct {
		name       string
		receiptURL *string
		wantErr    error
		wantStored *string
	}{
		{name: "accepts an https url", receiptURL: strPtr("https://example.com/receipt.pdf"), wantStored: strPtr("https://example.com/receipt.pdf")},
		{name: "accepts an http url", receiptURL: strPtr("http://example.com/receipt.pdf"), wantStored: strPtr("http://example.com/receipt.pdf")},
		{name: "treats empty string as unset", receiptURL: strPtr("")},
		{name: "treats omitted value as unset"},
		{name: "rejects javascript scheme", receiptURL: strPtr("javascript:alert(1)"), wantErr: ErrInvalidReceiptURL},
		{name: "rejects data scheme", receiptURL: strPtr("data:text/html,<script>alert(1)</script>"), wantErr: ErrInvalidReceiptURL},
		{name: "rejects value with no scheme", receiptURL: strPtr("example.com/receipt.pdf"), wantErr: ErrInvalidReceiptURL},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			maintenanceRepo := stubMaintenanceStore{}
			svc := NewMaintenanceService(maintenanceRepo, emptyVehicleLister{}, discardAuditWriter{}, stubTemplateRenderer{})
			ctx := context.Background()

			item, err := svc.Create(ctx, MaintenanceLogInput{
				VehicleID:   uuid.New(),
				ServiceDate: time.Now(),
				Mileage:     100,
				Items:       "測試項目",
				Cost:        0,
				ReceiptURL:  tt.receiptURL,
				CreatedBy:   uuid.New(),
			}, nil, nil)

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				return
			}

			assert.NoError(t, err)
			if tt.wantStored == nil {
				assert.Nil(t, item.ReceiptURL)
			} else {
				require.NotNil(t, item.ReceiptURL)
				assert.Equal(t, *tt.wantStored, *item.ReceiptURL)
			}
		})
	}
}
