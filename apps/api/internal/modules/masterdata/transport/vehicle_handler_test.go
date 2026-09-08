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

	t.Run("未提供車別時驗證失敗回傳 400", func(t *testing.T) {
		store := &fakeVehicleStore{}
		h := newTestVehicleHandler(store)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		body := `{"plateNo":"BZG-7915","siteName":"竹南日照據點"}`
		c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/vehicles", strings.NewReader(body))
		c.Request.Header.Set("Content-Type", "application/json")

		h.Create(c)

		require.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "displayName")
		assert.Nil(t, store.created)
	})

	t.Run("車別為空字串時驗證失敗回傳 400", func(t *testing.T) {
		store := &fakeVehicleStore{}
		h := newTestVehicleHandler(store)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		body := `{"plateNo":"BZG-7915","displayName":"","siteName":"竹南日照據點"}`
		c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/vehicles", strings.NewReader(body))
		c.Request.Header.Set("Content-Type", "application/json")

		h.Create(c)

		require.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "displayName")
		assert.Nil(t, store.created)
	})

	t.Run("提供車別時建立成功並寫入車別與據點", func(t *testing.T) {
		store := &fakeVehicleStore{}
		h := newTestVehicleHandler(store)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		body := `{"plateNo":"BZG-7915","displayName":"竹南2車","siteName":"竹南日照據點"}`
		c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/vehicles", strings.NewReader(body))
		c.Request.Header.Set("Content-Type", "application/json")

		h.Create(c)

		require.Equal(t, http.StatusCreated, w.Code)
		require.NotNil(t, store.created)
		assert.Equal(t, "BZG-7915", store.created.PlateNo)
		assert.Equal(t, "竹南2車", store.created.DisplayName)
		assert.Equal(t, "竹南日照據點", store.created.SiteName)
	})

	t.Run("不提供據點時可正常建立", func(t *testing.T) {
		store := &fakeVehicleStore{}
		h := newTestVehicleHandler(store)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		body := `{"plateNo":"BZG-7916","displayName":"竹南3車"}`
		c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/vehicles", strings.NewReader(body))
		c.Request.Header.Set("Content-Type", "application/json")

		h.Create(c)

		require.Equal(t, http.StatusCreated, w.Code)
		require.NotNil(t, store.created)
		assert.Empty(t, store.created.SiteName)
	})
}

func TestVehicleHandler_Update_RequiresDisplayName(t *testing.T) {
	gin.SetMode(gin.TestMode)
	vehicleID := uuid.New()

	t.Run("更新時未提供車別回傳 400", func(t *testing.T) {
		store := &fakeVehicleStore{}
		h := newTestVehicleHandler(store)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		body := `{"plateNo":"BZG-7915","siteName":"竹南日照據點"}`
		c.Request = httptest.NewRequest(http.MethodPatch, "/api/v1/vehicles/"+vehicleID.String(), strings.NewReader(body))
		c.Request.Header.Set("Content-Type", "application/json")
		c.Params = gin.Params{{Key: "id", Value: vehicleID.String()}}

		h.Update(c)

		require.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "displayName")
		assert.Nil(t, store.updated)
	})

	t.Run("更新時提供車別更新成功", func(t *testing.T) {
		store := &fakeVehicleStore{}
		h := newTestVehicleHandler(store)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		body := `{"plateNo":"BZG-7915","displayName":"竹南2車(改)","siteName":"竹南日照據點"}`
		c.Request = httptest.NewRequest(http.MethodPatch, "/api/v1/vehicles/"+vehicleID.String(), strings.NewReader(body))
		c.Request.Header.Set("Content-Type", "application/json")
		c.Params = gin.Params{{Key: "id", Value: vehicleID.String()}}

		h.Update(c)

		require.Equal(t, http.StatusOK, w.Code)
		require.NotNil(t, store.updated)
		assert.Equal(t, "竹南2車(改)", store.updated.DisplayName)
	})
}
