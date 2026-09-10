package app

import (
	"bytes"
	"context"
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
	listResult []Driver
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

func (f *fakeDriverStore) List(ctx context.Context, q, status string, page, pageSize int) ([]Driver, int64, error) {
	return f.listResult, int64(len(f.listResult)), nil
}

func (f *fakeDriverStore) GetByID(ctx context.Context, id uuid.UUID) (*Driver, error) {
	d, ok := f.byID[id]
	if !ok {
		return nil, ErrDriverNotFound
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

	t.Run("rejects blank name", func(t *testing.T) {
		store := newFakeDriverStore()
		svc := NewDriverService(store, cfg, nil)

		_, err := svc.Create(context.Background(), CreateDriverInput{
			Name:       "  ",
			NationalID: "A123456789",
		})

		assert.ErrorIs(t, err, ErrDriverNameRequired)
		assert.Nil(t, store.lastCreate)
	})

	t.Run("rejects invalid national id", func(t *testing.T) {
		store := newFakeDriverStore()
		svc := NewDriverService(store, cfg, nil)

		_, err := svc.Create(context.Background(), CreateDriverInput{
			Name:       "測試司機",
			NationalID: "NOT-VALID",
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
		})

		assert.NoError(t, err)
		assert.NotNil(t, d)
		assert.NotEmpty(t, d.NationalIDCipher)
		assert.NotEmpty(t, d.NationalIDHMAC)
		assert.Equal(t, "A12***6789", d.NationalIDMasked)
		assert.Equal(t, "A123456789", d.NationalID)
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

	t.Run("email", func(t *testing.T) {
		tests := []struct {
			name       string
			email      *string
			wantErr    error
			wantStored *string
		}{
			{name: "accepts a valid email", email: strPtr("driver@example.com"), wantStored: strPtr("driver@example.com")},
			{name: "trims surrounding spaces", email: strPtr("  driver@example.com  "), wantStored: strPtr("driver@example.com")},
			{name: "treats empty string as unset", email: strPtr("")},
			{name: "treats blank spaces as unset", email: strPtr("   ")},
			{name: "treats omitted value as unset"},
			{name: "rejects an invalid format", email: strPtr("not-an-email"), wantErr: ErrInvalidDriverEmail},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				store := newFakeDriverStore()
				svc := NewDriverService(store, cfg, nil)

				d, err := svc.Create(context.Background(), CreateDriverInput{
					Name:       "測試司機",
					NationalID: "A123456789",
					Email:      tt.email,
				})

				if tt.wantErr != nil {
					assert.ErrorIs(t, err, tt.wantErr)
					assert.Nil(t, store.lastCreate)
					return
				}

				assert.NoError(t, err)
				if tt.wantStored == nil {
					assert.Nil(t, d.Email)
				} else {
					require.NotNil(t, d.Email)
					assert.Equal(t, *tt.wantStored, *d.Email)
				}
			})
		}
	})

	t.Run("creates driver and assigns vehicle when vehicleID provided", func(t *testing.T) {
		store := newFakeDriverStore()
		svc := NewDriverService(store, cfg, nil)
		vehicleID := uuid.New()

		d, err := svc.Create(context.Background(), CreateDriverInput{
			Name:       "指派司機",
			NationalID: "A123456789",
			VehicleID:  &vehicleID,
		})

		require.NoError(t, err)
		assert.NotNil(t, d)
		assert.Equal(t, "指派司機", d.Name)
		assert.NotNil(t, store.lastAssign)
		assert.Equal(t, d.ID, store.lastAssign.DriverID)
		assert.Equal(t, vehicleID, store.lastAssign.VehicleID)
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
		store.byID[id] = &Driver{ID: id, Name: "舊名字", Status: "active"}
		svc := NewDriverService(store, cfg, nil)

		newName := "新名字"
		d, err := svc.Update(context.Background(), id, UpdateDriverInput{Name: &newName})

		assert.NoError(t, err)
		assert.Equal(t, "新名字", d.Name)
	})

	t.Run("updates license class and expiry date", func(t *testing.T) {
		id := uuid.New()
		oldExpiry := time.Date(2027, 5, 21, 0, 0, 0, 0, time.UTC)
		store := newFakeDriverStore()
		store.byID[id] = &Driver{ID: id, Name: "舊名字", Status: "active",
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

	t.Run("email 未提供時保留原值", func(t *testing.T) {
		id := uuid.New()
		store := newFakeDriverStore()
		store.byID[id] = &Driver{ID: id, Name: "舊名字", Status: "active", Email: strPtr("old@example.com")}
		svc := NewDriverService(store, cfg, nil)

		d, err := svc.Update(context.Background(), id, UpdateDriverInput{Name: strPtr("新名字")})

		assert.NoError(t, err)
		require.NotNil(t, d.Email)
		assert.Equal(t, "old@example.com", *d.Email)
	})

	t.Run("email 送空字串會清空原值", func(t *testing.T) {
		id := uuid.New()
		store := newFakeDriverStore()
		store.byID[id] = &Driver{ID: id, Name: "舊名字", Status: "active", Email: strPtr("old@example.com")}
		svc := NewDriverService(store, cfg, nil)

		d, err := svc.Update(context.Background(), id, UpdateDriverInput{Email: strPtr("")})

		assert.NoError(t, err)
		assert.Nil(t, d.Email)
	})

	t.Run("email 格式錯誤時拒絕更新", func(t *testing.T) {
		id := uuid.New()
		store := newFakeDriverStore()
		store.byID[id] = &Driver{ID: id, Name: "舊名字", Status: "active", Email: strPtr("old@example.com")}
		svc := NewDriverService(store, cfg, nil)

		_, err := svc.Update(context.Background(), id, UpdateDriverInput{Email: strPtr("not-an-email")})

		assert.ErrorIs(t, err, ErrInvalidDriverEmail)
		assert.Nil(t, store.lastUpdate)
	})

	t.Run("接受合法的狀態值", func(t *testing.T) {
		id := uuid.New()
		store := newFakeDriverStore()
		store.byID[id] = &Driver{ID: id, Name: "舊名字", Status: "active"}
		svc := NewDriverService(store, cfg, nil)

		d, err := svc.Update(context.Background(), id, UpdateDriverInput{Status: strPtr("inactive")})

		assert.NoError(t, err)
		assert.Equal(t, "inactive", d.Status)
	})

	t.Run("拒絕非法狀態值", func(t *testing.T) {
		id := uuid.New()
		store := newFakeDriverStore()
		store.byID[id] = &Driver{ID: id, Name: "舊名字", Status: "active"}
		svc := NewDriverService(store, cfg, nil)

		_, err := svc.Update(context.Background(), id, UpdateDriverInput{Status: strPtr("resigned")})

		assert.ErrorIs(t, err, ErrInvalidStatus)
	})

	t.Run("rejects an unknown license class", func(t *testing.T) {
		id := uuid.New()
		store := newFakeDriverStore()
		store.byID[id] = &Driver{ID: id, Name: "舊名字", Status: "active"}
		svc := NewDriverService(store, cfg, nil)

		_, err := svc.Update(context.Background(), id, UpdateDriverInput{LicenseClass: strPtr("motorcycle")})

		assert.ErrorIs(t, err, ErrInvalidDriverLicenseClass)
		assert.Nil(t, store.lastUpdate)
	})

	t.Run("clears the expiry date only when asked", func(t *testing.T) {
		id := uuid.New()
		expiry := time.Date(2027, 5, 21, 0, 0, 0, 0, time.UTC)
		store := newFakeDriverStore()
		store.byID[id] = &Driver{ID: id, Name: "舊名字", Status: "active", LicenseExpiryDate: &expiry}
		svc := NewDriverService(store, cfg, nil)

		d, err := svc.Update(context.Background(), id, UpdateDriverInput{})
		assert.NoError(t, err)
		assert.Equal(t, expiry, *d.LicenseExpiryDate)

		d, err = svc.Update(context.Background(), id, UpdateDriverInput{ClearLicenseExpiryDate: true})
		assert.NoError(t, err)
		assert.Nil(t, d.LicenseExpiryDate)
	})
}

func TestDriverService_List_DecryptsNationalID(t *testing.T) {
	cfg := testConfig()
	store := newFakeDriverStore()
	cipher, err := crypto.Encrypt("A123456789", cfg.EncryptionKey)
	assert.NoError(t, err)
	store.listResult = []Driver{
		{ID: uuid.New(), NationalIDCipher: cipher},
		{ID: uuid.New(), Name: "無身分證司機"},
	}
	svc := NewDriverService(store, cfg, nil)

	drivers, _, err := svc.List(context.Background(), "", "", 1, 20)

	require.NoError(t, err)
	require.Len(t, drivers, 2)
	assert.Equal(t, "A123456789", drivers[0].NationalID)
	assert.Empty(t, drivers[1].NationalID)
}

func TestDriverService_AssignVehicle(t *testing.T) {
	store := newFakeDriverStore()
	svc := NewDriverService(store, testConfig(), nil)

	driverID := uuid.New()
	vehicleID := uuid.New()

	assignment, err := svc.AssignVehicle(context.Background(), driverID, AssignVehicleInput{
		VehicleID: vehicleID,
	})

	assert.NoError(t, err)
	assert.Equal(t, driverID, assignment.DriverID)
	assert.Equal(t, vehicleID, assignment.VehicleID)
	assert.Same(t, assignment, store.lastAssign)
}

func TestDriverService_AssignVehicleRejectsInvalidDateRange(t *testing.T) {
	store := newFakeDriverStore()
	svc := NewDriverService(store, testConfig(), nil)

	_, err := svc.AssignVehicle(context.Background(), uuid.New(), AssignVehicleInput{})

	assert.ErrorIs(t, err, ErrInvalidAssignmentRange)
	assert.Nil(t, store.lastAssign)
}

func TestDriverService_UnassignVehicle(t *testing.T) {
	t.Run("成功解除車輛指派", func(t *testing.T) {
		store := newFakeDriverStore()
		svc := NewDriverService(store, testConfig(), nil)

		driverID := uuid.New()
		err := svc.UnassignVehicle(context.Background(), driverID)

		assert.NoError(t, err)
		assert.Equal(t, driverID, store.closedAssignments)
	})

	t.Run("無效的司機ID拒絕解除", func(t *testing.T) {
		store := newFakeDriverStore()
		svc := NewDriverService(store, testConfig(), nil)

		err := svc.UnassignVehicle(context.Background(), uuid.Nil)
		assert.ErrorIs(t, err, ErrInvalidAssignmentRange)
	})
}

func TestDriverService_Delete(t *testing.T) {
	t.Run("成功刪除並收斂車輛指派", func(t *testing.T) {
		store := newFakeDriverStore()
		driverID := uuid.New()
		store.byID[driverID] = &Driver{ID: driverID, Name: "待刪除司機", Status: "active"}
		svc := NewDriverService(store, testConfig(), nil)

		err := svc.Delete(context.Background(), driverID, uuid.New(), "admin")
		require.NoError(t, err)
		assert.True(t, store.deleted[driverID])
		assert.Equal(t, driverID, store.closedAssignments)
	})

	t.Run("已刪除再次刪除回錯誤", func(t *testing.T) {
		store := newFakeDriverStore()
		driverID := uuid.New()
		store.byID[driverID] = &Driver{ID: driverID, Name: "待刪除司機", Status: "active"}
		svc := NewDriverService(store, testConfig(), nil)

		require.NoError(t, svc.Delete(context.Background(), driverID, uuid.New(), "admin"))
		err := svc.Delete(context.Background(), driverID, uuid.New(), "admin")
		assert.ErrorIs(t, err, ErrDriverNotFound)
	})
}

func TestDriverService_CreateAndUpdate_ExtendedFields(t *testing.T) {
	cfg := testConfig()
	store := newFakeDriverStore()
	svc := NewDriverService(store, cfg, nil)

	birthDate := time.Date(1998, 8, 29, 0, 0, 0, 0, time.UTC)
	employmentDate := time.Date(2024, 11, 1, 0, 0, 0, 0, time.UTC)
	gender := "男"
	remarks := "優良駕駛"

	t.Run("成功新增包含擴充欄位且無區域之司機", func(t *testing.T) {
		d, err := svc.Create(context.Background(), CreateDriverInput{
			Name:                   "黃駿凱",
			NationalID:             "A123456789",
			Gender:                 &gender,
			BirthDate:              &birthDate,
			HasProfessionalLicense: true,
			EmploymentDate:         &employmentDate,
			HasTransferCert:        true,
			Remarks:                &remarks,
		})

		require.NoError(t, err)
		require.NotNil(t, d)
		assert.Equal(t, "黃駿凱", d.Name)
		assert.Equal(t, &gender, d.Gender)
		assert.Equal(t, &birthDate, d.BirthDate)
		assert.True(t, d.HasProfessionalLicense)
		assert.Equal(t, &employmentDate, d.EmploymentDate)
		assert.True(t, d.HasTransferCert)
		assert.Equal(t, &remarks, d.Remarks)
	})

	t.Run("成功更新擴充欄位與清空日期", func(t *testing.T) {
		driverID := uuid.New()
		store.byID[driverID] = &Driver{
			ID:                     driverID,
			Name:                   "鄧運政",
			Status:                 "active",
			BirthDate:              &birthDate,
			EmploymentDate:         &employmentDate,
			HasProfessionalLicense: true,
			HasTransferCert:        true,
		}

		newGender := "女"
		newRemarks := "轉調後勤"
		hasProf := false
		hasTrans := false

		updated, err := svc.Update(context.Background(), driverID, UpdateDriverInput{
			Gender:                 &newGender,
			ClearBirthDate:         true,
			HasProfessionalLicense: &hasProf,
			ClearEmploymentDate:    true,
			HasTransferCert:        &hasTrans,
			Remarks:                &newRemarks,
		})

		require.NoError(t, err)
		assert.Equal(t, &newGender, updated.Gender)
		assert.Nil(t, updated.BirthDate)
		assert.False(t, updated.HasProfessionalLicense)
		assert.Nil(t, updated.EmploymentDate)
		assert.False(t, updated.HasTransferCert)
		assert.Equal(t, &newRemarks, updated.Remarks)
	})
}

func TestDriverService_UpdateNationalID(t *testing.T) {
	cfg := testConfig()
	// 建立一位已登記身分證的司機，之後用同一組金鑰驗證密文、HMAC 與遮罩值。
	newDriver := func() (*fakeDriverStore, uuid.UUID) {
		store := newFakeDriverStore()
		id := uuid.New()
		cipher, err := crypto.Encrypt("A123456789", cfg.EncryptionKey)
		require.NoError(t, err)
		store.byID[id] = &Driver{
			ID:               id,
			Name:             "王小明",
			Status:           "active",
			NationalIDCipher: cipher,
			NationalIDHMAC:   crypto.Index("A123456789", cfg.HMACKey),
			NationalIDMasked: crypto.Mask("A123456789"),
			NationalID:       "A123456789",
		}
		return store, id
	}

	t.Run("未提供身分證時三個欄位原封不動", func(t *testing.T) {
		store, id := newDriver()
		before := *store.byID[id]
		name := "王大明"

		d, err := NewDriverService(store, cfg, nil).Update(context.Background(), id, UpdateDriverInput{Name: &name})

		require.NoError(t, err)
		assert.Equal(t, before.NationalIDCipher, d.NationalIDCipher)
		assert.Equal(t, before.NationalIDHMAC, d.NationalIDHMAC)
		assert.Equal(t, before.NationalIDMasked, d.NationalIDMasked)
		assert.Equal(t, before.NationalID, d.NationalID)
	})

	t.Run("提供新身分證時同步換掉密文、HMAC、遮罩值與明碼", func(t *testing.T) {
		store, id := newDriver()
		before := *store.byID[id]
		newID := "B234567894"

		d, err := NewDriverService(store, cfg, nil).Update(context.Background(), id, UpdateDriverInput{NationalID: &newID})

		require.NoError(t, err)
		assert.NotEqual(t, before.NationalIDCipher, d.NationalIDCipher)
		assert.Equal(t, crypto.Index(newID, cfg.HMACKey), d.NationalIDHMAC)
		assert.Equal(t, crypto.Mask(newID), d.NationalIDMasked)
		assert.Equal(t, newID, d.NationalID)

		plain, err := crypto.Decrypt(d.NationalIDCipher, cfg.EncryptionKey)
		require.NoError(t, err)
		assert.Equal(t, newID, plain, "解密後的明碼必須是新輸入的身分證")
	})

	t.Run("提供相同身分證時不重新加密", func(t *testing.T) {
		store, id := newDriver()
		before := *store.byID[id]
		sameID := "a123456789" // 大小寫與空白不影響判定

		d, err := NewDriverService(store, cfg, nil).Update(context.Background(), id, UpdateDriverInput{NationalID: &sameID})

		require.NoError(t, err)
		assert.Equal(t, before.NationalIDCipher, d.NationalIDCipher, "同一組身分證不應產生新密文")
	})

	t.Run("檢查碼錯誤時拒絕且不寫入", func(t *testing.T) {
		store, id := newDriver()
		badID := "A123456780"

		_, err := NewDriverService(store, cfg, nil).Update(context.Background(), id, UpdateDriverInput{NationalID: &badID})

		assert.ErrorIs(t, err, ErrInvalidDriverNationalID)
		assert.Nil(t, store.lastUpdate, "驗證失敗不得呼叫 store")
	})
}
