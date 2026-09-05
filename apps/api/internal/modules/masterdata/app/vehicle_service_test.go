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
			{ID: uuid.New(), Name: "郭澤威"},
			{ID: uuid.New(), Name: "林志豪"},
		},
	}
	svc := NewVehicleService(&fakeVehicleStore{list: []Vehicle{{ID: vehicleA}, {ID: vehicleB}}}, driverStore, nil)

	list, total, err := svc.List(context.Background(), VehicleFilter{}, 1, 20)

	assert.NoError(t, err)
	assert.Equal(t, int64(2), total)
	assert.Equal(t, []VehicleDriver{
		{ID: driverStore.byVehicle[vehicleA][0].ID, Name: "郭澤威"},
		{ID: driverStore.byVehicle[vehicleA][1].ID, Name: "林志豪"},
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

func TestVehicleService_Create_NormalizesStatus(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "空字串預設 active", input: "", want: "active"},
		{name: "接受 active", input: "active", want: "active"},
		{name: "接受 inactive", input: "inactive", want: "inactive"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewVehicleService(&fakeVehicleStore{}, newFakeDriverStore(), nil)
			v, err := svc.Create(context.Background(), VehicleInput{Status: tt.input})
			require.NoError(t, err)
			assert.Equal(t, tt.want, v.Status)
		})
	}
	t.Run("拒絕非法狀態", func(t *testing.T) {
		svc := NewVehicleService(&fakeVehicleStore{}, newFakeDriverStore(), nil)
		_, err := svc.Create(context.Background(), VehicleInput{Status: "maintenance"})
		assert.ErrorIs(t, err, ErrInvalidStatus)
	})
}

func TestVehicleService_Update_NormalizesStatus(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "空字串預設 active", input: "", want: "active"},
		{name: "接受 inactive", input: "inactive", want: "inactive"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewVehicleService(&fakeVehicleStore{}, newFakeDriverStore(), nil)
			v, err := svc.Update(context.Background(), uuid.New(), VehicleInput{Status: tt.input})
			require.NoError(t, err)
			assert.Equal(t, tt.want, v.Status)
		})
	}
	t.Run("拒絕非法狀態", func(t *testing.T) {
		svc := NewVehicleService(&fakeVehicleStore{}, newFakeDriverStore(), nil)
		_, err := svc.Update(context.Background(), uuid.New(), VehicleInput{Status: "retired"})
		assert.ErrorIs(t, err, ErrInvalidStatus)
	})
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

	t.Run("查無車輛時回傳 NotFound sentinel", func(t *testing.T) {
		store := &fakeVehicleStore{}
		store.softDeleteErr = nil
		// 以 fake 的 false 回傳模擬資料庫沒有更新任何列。
		storeWithNotFound := &notFoundVehicleStore{fakeVehicleStore: store}
		svc := NewVehicleService(storeWithNotFound, newFakeDriverStore(), nil)

		err := svc.Delete(context.Background(), uuid.New(), uuid.New(), "admin")
		assert.ErrorIs(t, err, ErrVehicleNotFound)
	})
}

type notFoundVehicleStore struct {
	*fakeVehicleStore
}

func (s *notFoundVehicleStore) SoftDelete(context.Context, uuid.UUID, uuid.UUID) (bool, error) {
	return false, nil
}
