package transport

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"ltc-system/apps/api/internal/modules/masterdata/app"
)

type fakeVehicleStore struct {
	created *app.Vehicle
	updated *app.Vehicle
}

func (f *fakeVehicleStore) List(ctx context.Context, filter app.VehicleFilter, page, pageSize int) ([]app.Vehicle, int64, error) {
	return nil, 0, nil
}

func (f *fakeVehicleStore) Create(ctx context.Context, v *app.Vehicle) error {
	f.created = v
	return nil
}

func (f *fakeVehicleStore) Update(ctx context.Context, v *app.Vehicle) error {
	f.updated = v
	return nil
}

func (f *fakeVehicleStore) SoftDelete(ctx context.Context, id, actorID uuid.UUID) (bool, error) {
	return true, nil
}

func (f *fakeVehicleStore) CountActiveDriverAssignments(ctx context.Context, vehicleID uuid.UUID) (int, error) {
	return 0, nil
}

func (f *fakeVehicleStore) CountScheduleLegs(ctx context.Context, vehicleID uuid.UUID) (int, error) {
	return 0, nil
}

func newTestVehicleHandler(store *fakeVehicleStore) *VehicleHandler {
	return NewVehicleHandler(app.NewVehicleService(store, nil, nil))
}

func TestVehicleHandler_Create_RequiresDisplayName(t *testing.T) {
	gin.SetMode(gin.TestMode)
	siteID := uuid.New()

	t.Run("未提供代稱時驗證失敗回傳 400", func(t *testing.T) {
		store := &fakeVehicleStore{}
		h := newTestVehicleHandler(store)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		body := `{"plateNo":"BZG-7915","siteId":"` + siteID.String() + `"}`
		c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/vehicles", strings.NewReader(body))
		c.Request.Header.Set("Content-Type", "application/json")

		h.Create(c)

		require.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "displayName")
		assert.Nil(t, store.created)
	})

	t.Run("代稱為空字串時驗證失敗回傳 400", func(t *testing.T) {
		store := &fakeVehicleStore{}
		h := newTestVehicleHandler(store)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		body := `{"plateNo":"BZG-7915","displayName":"","siteId":"` + siteID.String() + `"}`
		c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/vehicles", strings.NewReader(body))
		c.Request.Header.Set("Content-Type", "application/json")

		h.Create(c)

		require.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "displayName")
		assert.Nil(t, store.created)
	})

	t.Run("提供代稱時建立成功並寫入代稱", func(t *testing.T) {
		store := &fakeVehicleStore{}
		h := newTestVehicleHandler(store)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		body := `{"plateNo":"BZG-7915","displayName":"竹南2車","siteId":"` + siteID.String() + `"}`
		c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/vehicles", strings.NewReader(body))
		c.Request.Header.Set("Content-Type", "application/json")

		h.Create(c)

		require.Equal(t, http.StatusCreated, w.Code)
		require.NotNil(t, store.created)
		assert.Equal(t, "BZG-7915", store.created.PlateNo)
		assert.Equal(t, "竹南2車", store.created.DisplayName)
		assert.Equal(t, siteID, *store.created.SiteID)
	})
}

func TestVehicleHandler_Update_RequiresDisplayName(t *testing.T) {
	gin.SetMode(gin.TestMode)
	vehicleID := uuid.New()
	siteID := uuid.New()

	t.Run("更新時未提供代稱回傳 400", func(t *testing.T) {
		store := &fakeVehicleStore{}
		h := newTestVehicleHandler(store)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		body := `{"plateNo":"BZG-7915","siteId":"` + siteID.String() + `"}`
		c.Request = httptest.NewRequest(http.MethodPatch, "/api/v1/vehicles/"+vehicleID.String(), strings.NewReader(body))
		c.Request.Header.Set("Content-Type", "application/json")
		c.Params = gin.Params{{Key: "id", Value: vehicleID.String()}}

		h.Update(c)

		require.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "displayName")
		assert.Nil(t, store.updated)
	})

	t.Run("更新時提供代稱更新成功", func(t *testing.T) {
		store := &fakeVehicleStore{}
		h := newTestVehicleHandler(store)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		body := `{"plateNo":"BZG-7915","displayName":"竹南2車(改)","siteId":"` + siteID.String() + `"}`
		c.Request = httptest.NewRequest(http.MethodPatch, "/api/v1/vehicles/"+vehicleID.String(), strings.NewReader(body))
		c.Request.Header.Set("Content-Type", "application/json")
		c.Params = gin.Params{{Key: "id", Value: vehicleID.String()}}

		h.Update(c)

		require.Equal(t, http.StatusOK, w.Code)
		require.NotNil(t, store.updated)
		assert.Equal(t, "竹南2車(改)", store.updated.DisplayName)
	})
}
