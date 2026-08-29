package app

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
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

func TestDriverService_Create(t *testing.T) {
	cfg := testConfig()

	t.Run("rejects invalid national id", func(t *testing.T) {
		store := newFakeDriverStore()
		svc := NewDriverService(store, cfg)

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
		svc := NewDriverService(store, cfg)

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
}

func TestDriverService_Update(t *testing.T) {
	cfg := testConfig()

	t.Run("not found", func(t *testing.T) {
		store := newFakeDriverStore()
		svc := NewDriverService(store, cfg)

		_, err := svc.Update(context.Background(), uuid.New(), UpdateDriverInput{})
		assert.ErrorIs(t, err, ErrDriverNotFound)
	})

	t.Run("applies only provided fields", func(t *testing.T) {
		id := uuid.New()
		store := newFakeDriverStore()
		store.byID[id] = &Driver{ID: id, Name: "舊名字", Region: "hsinchu", Status: "active"}
		svc := NewDriverService(store, cfg)

		newName := "新名字"
		d, err := svc.Update(context.Background(), id, UpdateDriverInput{Name: &newName})

		assert.NoError(t, err)
		assert.Equal(t, "新名字", d.Name)
		assert.Equal(t, "hsinchu", d.Region) // 未提供的欄位保持不變
	})
}

func TestDriverService_Reveal(t *testing.T) {
	cfg := testConfig()
	store := newFakeDriverStore()
	svc := NewDriverService(store, cfg)

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
	svc := NewDriverService(store, testConfig())

	driverID := uuid.New()
	vehicleID := uuid.New()
	from := time.Now()

	assignment, err := svc.AssignVehicle(context.Background(), driverID, AssignVehicleInput{
		VehicleID:     vehicleID,
		IsPrimary:     true,
		EffectiveFrom: from,
	})

	assert.NoError(t, err)
	assert.Equal(t, driverID, assignment.DriverID)
	assert.Equal(t, vehicleID, assignment.VehicleID)
	assert.True(t, assignment.IsPrimary)
	assert.Same(t, assignment, store.lastAssign)
}
