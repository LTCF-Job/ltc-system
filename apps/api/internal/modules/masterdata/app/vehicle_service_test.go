package app

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// fakeVehicleStore is a deterministic VehicleStore test double.
type fakeVehicleStore struct {
	list []Vehicle
}

func (f *fakeVehicleStore) List(ctx context.Context, filter VehicleFilter, page, pageSize int) ([]Vehicle, int64, error) {
	return f.list, int64(len(f.list)), nil
}

func (f *fakeVehicleStore) Create(ctx context.Context, v *Vehicle) error { return nil }

func (f *fakeVehicleStore) Update(ctx context.Context, v *Vehicle) error { return nil }

func TestVehicleService_List(t *testing.T) {
	vehicleA, vehicleB := uuid.New(), uuid.New()
	driverStore := newFakeDriverStore()
	driverStore.byVehicle = map[uuid.UUID][]Driver{
		vehicleA: {
			{ID: uuid.New(), Code: "DRV001", Name: "郭澤威"},
			{ID: uuid.New(), Code: "DRV002", Name: "林志豪"},
		},
	}
	svc := NewVehicleService(&fakeVehicleStore{list: []Vehicle{{ID: vehicleA}, {ID: vehicleB}}}, driverStore)

	list, total, err := svc.List(context.Background(), VehicleFilter{}, 1, 20)

	assert.NoError(t, err)
	assert.Equal(t, int64(2), total)
	assert.Equal(t, []VehicleDriver{
		{ID: driverStore.byVehicle[vehicleA][0].ID, Code: "DRV001", Name: "郭澤威"},
		{ID: driverStore.byVehicle[vehicleA][1].ID, Code: "DRV002", Name: "林志豪"},
	}, list[0].Drivers)
	assert.Empty(t, list[1].Drivers)
}

func TestVehicleService_SetDrivers(t *testing.T) {
	driverStore := newFakeDriverStore()
	svc := NewVehicleService(&fakeVehicleStore{}, driverStore)

	vehicleID := uuid.New()
	driverIDs := []uuid.UUID{uuid.New(), uuid.New()}
	from := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)

	err := svc.SetDrivers(context.Background(), vehicleID, driverIDs, from)

	assert.NoError(t, err)
	assert.Equal(t, vehicleID, driverStore.lastReplace.vehicleID)
	assert.Equal(t, driverIDs, driverStore.lastReplace.driverIDs)
	assert.Equal(t, from, driverStore.lastReplace.effectiveFrom)
}
