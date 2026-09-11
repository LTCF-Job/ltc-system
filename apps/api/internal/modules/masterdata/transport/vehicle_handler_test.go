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
	created   *app.Vehicle
	updated   *app.Vehicle
	updateErr error
	deleteErr error
}

func (f *fakeVehicleStore) List(ctx context.Context, filter app.VehicleFilter, page, pageSize int) ([]app.Vehicle, int64, error) {
	return nil, 0, nil
}

func (f *fakeVehicleStore) Create(ctx context.Context, v *app.Vehicle) error {
	f.created = v
	return nil
}

func (f *fakeVehicleStore) Update(ctx context.Context, v *app.Vehicle) error {
	if f.updateErr != nil {
		return f.updateErr
	}
	f.updated = v
	return nil
}

func (f *fakeVehicleStore) SoftDelete(ctx context.Context, id, actorID uuid.UUID) (bool, error) {
	if f.deleteErr != nil {
		return false, f.deleteErr
	}
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

// TestVehicleHandler_Update_NotFound_Returns404 確認更新不存在的車輛回 404，而不是被籠統的
// 400 驗證失敗掩蓋掉真正原因。
func TestVehicleHandler_Update_NotFound_Returns404(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := &fakeVehicleStore{updateErr: app.ErrVehicleNotFound}
	h := newTestVehicleHandler(store)
	id := uuid.New()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body := `{"plateNo":"BZG-7915","displayName":"竹南2車"}`
	c.Request = httptest.NewRequest(http.MethodPatch, "/api/v1/vehicles/"+id.String(), strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: id.String()}}

	h.Update(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// TestVehicleHandler_Delete_NotFound_Returns404 確認刪除不存在的車輛回 404，而不是籠統的 500
// 「系統發生錯誤」。
func TestVehicleHandler_Delete_NotFound_Returns404(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := &fakeVehicleStore{deleteErr: app.ErrVehicleNotFound}
	h := newTestVehicleHandler(store)
	id := uuid.New()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodDelete, "/api/v1/vehicles/"+id.String(), nil)
	c.Params = gin.Params{{Key: "id", Value: id.String()}}

	h.Delete(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// newTestVehicleHandlerWithDrivers 建立一個帶有真實 DriverStore 假物件的 VehicleHandler，
// 供 SetDrivers 相關測試使用（預設的 newTestVehicleHandler 傳入 nil drivers，呼叫
// ReplaceVehicleDrivers 會直接 panic）。
func newTestVehicleHandlerWithDrivers(store *fakeVehicleStore, drivers *fakeDriverStore) *VehicleHandler {
	return NewVehicleHandler(app.NewVehicleService(store, drivers, nil))
}

// TestVehicleHandler_SetDrivers_ReferenceInvalid_Returns400 確認指派了不存在的司機
// （外鍵違反）會回 400 並附具體原因，而不是籠統的 500「系統發生錯誤」。
func TestVehicleHandler_SetDrivers_ReferenceInvalid_Returns400(t *testing.T) {
	gin.SetMode(gin.TestMode)
	drivers := &fakeDriverStore{replaceErr: app.ErrAssignmentReferenceInvalid}
	h := newTestVehicleHandlerWithDrivers(&fakeVehicleStore{}, drivers)
	id := uuid.New()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body := `{"driverIds":["` + uuid.New().String() + `"]}`
	c.Request = httptest.NewRequest(http.MethodPut, "/api/v1/vehicles/"+id.String()+"/drivers", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: id.String()}}

	h.SetDrivers(c)

	require.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "不存在")
}

// TestVehicleHandler_SetDrivers_Overlap_Returns409 確認指派期間重疊（違反不重疊限制）
// 會回 409 並附具體原因，而不是籠統的 500「系統發生錯誤」。
func TestVehicleHandler_SetDrivers_Overlap_Returns409(t *testing.T) {
	gin.SetMode(gin.TestMode)
	drivers := &fakeDriverStore{replaceErr: app.ErrAssignmentOverlap}
	h := newTestVehicleHandlerWithDrivers(&fakeVehicleStore{}, drivers)
	id := uuid.New()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body := `{"driverIds":["` + uuid.New().String() + `"]}`
	c.Request = httptest.NewRequest(http.MethodPut, "/api/v1/vehicles/"+id.String()+"/drivers", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: id.String()}}

	h.SetDrivers(c)

	require.Equal(t, http.StatusConflict, w.Code)
	assert.Contains(t, w.Body.String(), "已有其他車輛指派")
}
