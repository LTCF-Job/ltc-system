package app

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFuelService_ListRejectsInvalidPaginationBeforeRepository(t *testing.T) {
	svc := NewFuelService(nil, nil)

	_, _, err := svc.List(context.Background(), 1, 0, nil, nil, nil, nil, "")

	require.ErrorIs(t, err, ErrInvalidFuelPagination)
}

type stubFuelStore struct{}

func (stubFuelStore) List(context.Context, int, int, *uuid.UUID, *uuid.UUID, *time.Time, *time.Time, string) ([]FuelLog, int, error) {
	return nil, 0, nil
}
func (stubFuelStore) Create(_ context.Context, item *FuelLog) error {
	item.ID = uuid.New()
	item.CreatedAt = time.Now()
	return nil
}
func (stubFuelStore) Update(context.Context, *FuelLog) error { return nil }
func (stubFuelStore) Delete(context.Context, uuid.UUID) error { return nil }

func TestFuelService_ReceiptURL(t *testing.T) {
	tests := []struct {
		name       string
		receiptURL *string
		wantErr    error
		wantStored *string
	}{
		{name: "accepts an https url", receiptURL: strPtr("https://example.com/receipt.pdf"), wantStored: strPtr("https://example.com/receipt.pdf")},
		{name: "treats empty string as unset", receiptURL: strPtr("")},
		{name: "treats omitted value as unset"},
		{name: "rejects javascript scheme", receiptURL: strPtr("javascript:alert(1)"), wantErr: ErrInvalidReceiptURL},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewFuelService(stubFuelStore{}, discardAuditWriter{})
			ctx := context.Background()

			item, err := svc.Create(ctx, FuelLogInput{
				VehicleID:  uuid.New(),
				FuelDate:   time.Now(),
				Liters:     10,
				Cost:       500,
				ReceiptURL: tt.receiptURL,
				CreatedBy:  uuid.New(),
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
