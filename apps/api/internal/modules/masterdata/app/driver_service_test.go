package app

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"ltc-system/apps/api/internal/domain/crypto"
	"ltc-system/apps/api/internal/platform/config"
)

func testConfig() *config.Config {
	return &config.Config{
		EncryptionKey: bytes.Repeat([]byte("a"), 32),
		HMACKey:       bytes.Repeat([]byte("b"), 32),
	}
}

// fakeDriverStore is a deterministic DriverStore test double.
type fakeDriverStore struct {
	byID       map[uuid.UUID]*Driver
	createErr  error
	updateErr  error
	assignErr  error
	lastCreate *Driver
	lastUpdate *Driver
	lastAssign *DriverAssignment

	byVehicle   map[uuid.UUID][]Driver
	lastReplace *replacedVehicleDrivers

	deleted           map[uuid.UUID]bool
	softDeleteErr     error
	closedAssignments uuid.UUID
	closeAssignErr    error
}

// replacedVehicleDrivers 記錄一次 ReplaceVehicleDrivers 呼叫的參數。
type replacedVehicleDrivers struct {
	vehicleID     uuid.UUID
	driverIDs     []uuid.UUID
	effectiveFrom time.Time
}

func newFakeDriverStore() *fakeDriverStore {
	return &fakeDriverStore{byID: map[uuid.UUID]*Driver{}}
}

func (f *fakeDriverStore) List(ctx context.Context, region, q string, page, pageSize int) ([]Driver, int64, error) {
	return nil, 0, nil
}

func (f *fakeDriverStore) GetByID(ctx context.Context, id uuid.UUID) (*Driver, error) {
	d, ok := f.byID[id]
	if !ok {
		return nil, errors.New("not found")
	}
	return d, nil
}

func (f *fakeDriverStore) Create(ctx context.Context, d *Driver) error {
	f.lastCreate = d
	if f.createErr != nil {
		return f.createErr
	}
	f.byID[d.ID] = d
	return nil
}

func (f *fakeDriverStore) Update(ctx context.Context, d *Driver) error {
	f.lastUpdate = d
	if f.updateErr != nil {
		return f.updateErr
	}
	f.byID[d.ID] = d
	return nil
}

func (f *fakeDriverStore) AssignVehicle(ctx context.Context, a *DriverAssignment) error {
	f.lastAssign = a
	return f.assignErr
}

func (f *fakeDriverStore) ListByVehicleIDsOnDate(ctx context.Context, vehicleIDs []uuid.UUID, on time.Time) (map[uuid.UUID][]Driver, error) {
	out := map[uuid.UUID][]Driver{}
	for _, id := range vehicleIDs {
		if drivers, ok := f.byVehicle[id]; ok {
			out[id] = drivers
		}
	}
	return out, nil
}

func (f *fakeDriverStore) ReplaceVehicleDrivers(ctx context.Context, vehicleID uuid.UUID, driverIDs []uuid.UUID, effectiveFrom time.Time) error {
	f.lastReplace = &replacedVehicleDrivers{vehicleID: vehicleID, driverIDs: driverIDs, effectiveFrom: effectiveFrom}
	return nil
}

func (f *fakeDriverStore) SoftDelete(ctx context.Context, id, actorID uuid.UUID) (bool, error) {
	if f.softDeleteErr != nil {
		return false, f.softDeleteErr
	}
	if f.deleted == nil {
		f.deleted = map[uuid.UUID]bool{}
	}
	if f.deleted[id] {
		return false, nil
	}
	f.deleted[id] = true
	return true, nil
}

func (f *fakeDriverStore) CloseActiveAssignments(ctx context.Context, driverID uuid.UUID) error {
	f.closedAssignments = driverID
	return f.closeAssignErr
}

func TestDriverService_Create(t *testing.T) {
	cfg := testConfig()

	t.Run("rejects invalid national id", func(t *testing.T) {
		store := newFakeDriverStore()
		svc := NewDriverService(store, cfg, nil)

		_, err := svc.Create(context.Background(), CreateDriverInput{
			Name:       "測試司機",
			NationalID: "NOT-VALID",
			Region:     "hsinchu",
		})

		assert.ErrorIs(t, err, ErrInvalidDriverNationalID)
		assert.Nil(t, store.lastCreate)
	})

	t.Run("encrypts and stores a valid national id", func(t *testing.T) {
		store := newFakeDriverStore()
		svc := NewDriverService(store, cfg, nil)

		d, err := svc.Create(context.Background(), CreateDriverInput{
			Name:       "測試司機",
			NationalID: "A123456789",
			Region:     "hsinchu",
		})

		assert.NoError(t, err)
		assert.NotNil(t, d)
		assert.NotEmpty(t, d.NationalIDCipher)
		assert.NotEmpty(t, d.NationalIDHMAC)
		assert.Equal(t, "A12***6789", d.NationalIDMasked)
		assert.Equal(t, "active", d.Status)
		assert.Same(t, d, store.lastCreate)

		plain, err := crypto.Decrypt(d.NationalIDCipher, cfg.EncryptionKey)
		assert.NoError(t, err)
		assert.Equal(t, "A123456789", plain)
	})

	t.Run("license class", func(t *testing.T) {
		expiry := time.Date(2031, 4, 22, 0, 0, 0, 0, time.UTC)
		tests := []struct {
			name         string
			licenseClass *string
			wantErr      error
			wantStored   *string
		}{
			{name: "accepts a known class", licenseClass: strPtr("truck"), wantStored: strPtr("truck")},
			{name: "trims surrounding spaces", licenseClass: strPtr(" bus "), wantStored: strPtr("bus")},
			{name: "treats empty string as unset", licenseClass: strPtr("  ")},
			{name: "treats omitted value as unset"},
			{name: "rejects an unknown class", licenseClass: strPtr("motorcycle"), wantErr: ErrInvalidDriverLicenseClass},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				store := newFakeDriverStore()
				svc := NewDriverService(store, cfg, nil)

				d, err := svc.Create(context.Background(), CreateDriverInput{
					Name:              "測試司機",
					NationalID:        "A123456789",
					Region:            "hsinchu",
					LicenseClass:      tt.licenseClass,
					LicenseExpiryDate: &expiry,
				})

				if tt.wantErr != nil {
					assert.ErrorIs(t, err, tt.wantErr)
					assert.Nil(t, store.lastCreate)
					return
				}

				assert.NoError(t, err)
				if tt.wantStored == nil {
					assert.Nil(t, d.LicenseClass)
				} else {
					assert.Equal(t, *tt.wantStored, *d.LicenseClass)
				}
				assert.Equal(t, expiry, *d.LicenseExpiryDate)
			})
		}
	})
}

func strPtr(v string) *string { return &v }

func TestDriverService_Update(t *testing.T) {
	cfg := testConfig()

	t.Run("not found", func(t *testing.T) {
		store := newFakeDriverStore()
		svc := NewDriverService(store, cfg, nil)

		_, err := svc.Update(context.Background(), uuid.New(), UpdateDriverInput{})
		assert.ErrorIs(t, err, ErrDriverNotFound)
	})

	t.Run("applies only provided fields", func(t *testing.T) {
		id := uuid.New()
		store := newFakeDriverStore()
		store.byID[id] = &Driver{ID: id, Name: "舊名字", Region: "hsinchu", Status: "active"}
		svc := NewDriverService(store, cfg, nil)

		newName := "新名字"
		d, err := svc.Update(context.Background(), id, UpdateDriverInput{Name: &newName})

		assert.NoError(t, err)
		assert.Equal(t, "新名字", d.Name)
		assert.Equal(t, "hsinchu", d.Region) // 未提供的欄位保持不變
	})

	t.Run("updates license class and expiry date", func(t *testing.T) {
		id := uuid.New()
		oldExpiry := time.Date(2027, 5, 21, 0, 0, 0, 0, time.UTC)
		store := newFakeDriverStore()
		store.byID[id] = &Driver{ID: id, Name: "舊名字", Region: "hsinchu", Status: "active",
			LicenseClass: strPtr("sedan"), LicenseExpiryDate: &oldExpiry}
		svc := NewDriverService(store, cfg, nil)

		newExpiry := time.Date(2031, 3, 24, 0, 0, 0, 0, time.UTC)
		d, err := svc.Update(context.Background(), id, UpdateDriverInput{
			LicenseClass:      strPtr("trailer"),
			LicenseExpiryDate: &newExpiry,
		})

		assert.NoError(t, err)
		assert.Equal(t, "trailer", *d.LicenseClass)
		assert.Equal(t, newExpiry, *d.LicenseExpiryDate)
	})

	t.Run("rejects an unknown license class", func(t *testing.T) {
		id := uuid.New()
		store := newFakeDriverStore()
		store.byID[id] = &Driver{ID: id, Name: "舊名字", Region: "hsinchu", Status: "active"}
		svc := NewDriverService(store, cfg, nil)

		_, err := svc.Update(context.Background(), id, UpdateDriverInput{LicenseClass: strPtr("motorcycle")})

		assert.ErrorIs(t, err, ErrInvalidDriverLicenseClass)
		assert.Nil(t, store.lastUpdate)
	})

	t.Run("clears the expiry date only when asked", func(t *testing.T) {
		id := uuid.New()
		expiry := time.Date(2027, 5, 21, 0, 0, 0, 0, time.UTC)
		store := newFakeDriverStore()
		store.byID[id] = &Driver{ID: id, Name: "舊名字", Region: "hsinchu", Status: "active", LicenseExpiryDate: &expiry}
		svc := NewDriverService(store, cfg, nil)

		d, err := svc.Update(context.Background(), id, UpdateDriverInput{})
		assert.NoError(t, err)
		assert.Equal(t, expiry, *d.LicenseExpiryDate)

		d, err = svc.Update(context.Background(), id, UpdateDriverInput{ClearLicenseExpiryDate: true})
		assert.NoError(t, err)
		assert.Nil(t, d.LicenseExpiryDate)
	})
}

func TestDriverService_Reveal(t *testing.T) {
	cfg := testConfig()
	store := newFakeDriverStore()
	svc := NewDriverService(store, cfg, nil)

	cipher, err := crypto.Encrypt("A123456789", cfg.EncryptionKey)
	assert.NoError(t, err)
	id := uuid.New()
	store.byID[id] = &Driver{ID: id, NationalIDCipher: cipher}

	plain, err := svc.Reveal(context.Background(), id)
	assert.NoError(t, err)
	assert.Equal(t, "A123456789", plain)

	_, err = svc.Reveal(context.Background(), uuid.New())
	assert.ErrorIs(t, err, ErrDriverNotFound)
}

func TestDriverService_AssignVehicle(t *testing.T) {
	store := newFakeDriverStore()
	svc := NewDriverService(store, testConfig(), nil)

	driverID := uuid.New()
	vehicleID := uuid.New()
	from := time.Now()

	assignment, err := svc.AssignVehicle(context.Background(), driverID, AssignVehicleInput{
		VehicleID:     vehicleID,
		EffectiveFrom: from,
	})

	assert.NoError(t, err)
	assert.Equal(t, driverID, assignment.DriverID)
	assert.Equal(t, vehicleID, assignment.VehicleID)
	assert.Same(t, assignment, store.lastAssign)
}

func TestDriverService_Delete(t *testing.T) {
	t.Run("成功刪除並收斂車輛指派", func(t *testing.T) {
		store := newFakeDriverStore()
		driverID := uuid.New()
		svc := NewDriverService(store, testConfig(), nil)

		err := svc.Delete(context.Background(), driverID, uuid.New(), "admin")
		require.NoError(t, err)
		assert.True(t, store.deleted[driverID])
		assert.Equal(t, driverID, store.closedAssignments)
	})

	t.Run("已刪除再次刪除回錯誤", func(t *testing.T) {
		store := newFakeDriverStore()
		driverID := uuid.New()
		svc := NewDriverService(store, testConfig(), nil)

		require.NoError(t, svc.Delete(context.Background(), driverID, uuid.New(), "admin"))
		err := svc.Delete(context.Background(), driverID, uuid.New(), "admin")
		assert.ErrorIs(t, err, ErrDriverNotFound)
	})
}
