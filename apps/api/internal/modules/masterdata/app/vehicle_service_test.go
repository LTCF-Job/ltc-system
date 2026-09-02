package app

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeVehicleStore is a deterministic VehicleStore test double.
type fakeVehicleStore struct {
	list               []Vehicle
	deleted            map[uuid.UUID]bool
	softDeleteErr      error
	activeAssignments  int
	activeScheduleLegs int
	countErr           error
	softDeleteCalled   bool
}

func (f *fakeVehicleStore) List(ctx context.Context, filter VehicleFilter, page, pageSize int) ([]Vehicle, int64, error) {
	return f.list, int64(len(f.list)), nil
}

func (f *fakeVehicleStore) Create(ctx context.Context, v *Vehicle) error { return nil }

func (f *fakeVehicleStore) Update(ctx context.Context, v *Vehicle) error { return nil }

func (f *fakeVehicleStore) CountActiveDriverAssignments(ctx context.Context, vehicleID uuid.UUID) (int, error) {
	return f.activeAssignments, f.countErr
}

func (f *fakeVehicleStore) CountScheduleLegs(ctx context.Context, vehicleID uuid.UUID) (int, error) {
	return f.activeScheduleLegs, f.countErr
}

func (f *fakeVehicleStore) SoftDelete(ctx context.Context, id, actorID uuid.UUID) (bool, error) {
	f.softDeleteCalled = true
	if f.softDeleteErr != nil {
		return false, f.softDeleteErr
	}
	if f.deleted == nil {
		f.deleted = map[uuid.UUID]bool{}
	}
	f.deleted[id] = true
	return true, nil
}

func TestVehicleService_List(t *testing.T) {
	vehicleA, vehicleB := uuid.New(), uuid.New()
	driverStore := newFakeDriverStore()
	driverStore.byVehicle = map[uuid.UUID][]Driver{
		vehicleA: {
			{ID: uuid.New(), Code: "DRV001", Name: "郭澤威"},
			{ID: uuid.New(), Code: "DRV002", Name: "林志豪"},
		},
	}
	svc := NewVehicleService(&fakeVehicleStore{list: []Vehicle{{ID: vehicleA}, {ID: vehicleB}}}, driverStore, nil)

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
	svc := NewVehicleService(&fakeVehicleStore{}, driverStore, nil)

	vehicleID := uuid.New()
	driverIDs := []uuid.UUID{uuid.New(), uuid.New()}
	from := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)

	err := svc.SetDrivers(context.Background(), vehicleID, driverIDs, from)

	assert.NoError(t, err)
	assert.Equal(t, vehicleID, driverStore.lastReplace.vehicleID)
	assert.Equal(t, driverIDs, driverStore.lastReplace.driverIDs)
	assert.Equal(t, from, driverStore.lastReplace.effectiveFrom)
}

type fakeMasterAuditWriter struct {
	entries []AuditEntry
}

func (f *fakeMasterAuditWriter) Write(_ context.Context, e AuditEntry) error {
	f.entries = append(f.entries, e)
	return nil
}

func TestVehicleService_Delete(t *testing.T) {
	t.Run("有生效中司機指派時擋下且不寫入軟刪", func(t *testing.T) {
		store := &fakeVehicleStore{activeAssignments: 1}
		svc := NewVehicleService(store, newFakeDriverStore(), nil)

		err := svc.Delete(context.Background(), uuid.New(), uuid.New(), "admin")
		assert.ErrorIs(t, err, ErrVehicleInUse)
		assert.False(t, store.softDeleteCalled)
	})

	t.Run("有生效中排班趟次時擋下", func(t *testing.T) {
		store := &fakeVehicleStore{activeScheduleLegs: 1}
		svc := NewVehicleService(store, newFakeDriverStore(), nil)

		err := svc.Delete(context.Background(), uuid.New(), uuid.New(), "admin")
		assert.ErrorIs(t, err, ErrVehicleInUse)
		assert.False(t, store.softDeleteCalled)
	})

	t.Run("成功軟刪並寫入稽核", func(t *testing.T) {
		store := &fakeVehicleStore{}
		audit := &fakeMasterAuditWriter{}
		svc := NewVehicleService(store, newFakeDriverStore(), audit)
		vehicleID := uuid.New()

		err := svc.Delete(context.Background(), vehicleID, uuid.New(), "admin")
		require.NoError(t, err)
		assert.True(t, store.deleted[vehicleID])
		require.Len(t, audit.entries, 1)
		assert.Equal(t, "delete", audit.entries[0].Action)
	})
}
