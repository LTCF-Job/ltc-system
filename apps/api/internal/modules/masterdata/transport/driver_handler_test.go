package transport

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"ltc-system/apps/api/internal/modules/masterdata/app"
	"ltc-system/apps/api/internal/platform/config"
)

func testDriverConfig() *config.Config {
	return &config.Config{
		EncryptionKey: bytes.Repeat([]byte("a"), 32),
		HMACKey:       bytes.Repeat([]byte("b"), 32),
	}
}

// fakeDriverStore 只實作測試需要用到的行為，其餘回傳零值即可。
type fakeDriverStore struct {
	createErr  error
	created    *app.Driver
	assignErr  error
	replaceErr error
}

func (f *fakeDriverStore) List(ctx context.Context, q, status string, page, pageSize int) ([]app.Driver, int64, error) {
	return nil, 0, nil
}
func (f *fakeDriverStore) GetByID(ctx context.Context, id uuid.UUID) (*app.Driver, error) {
	return nil, app.ErrDriverNotFound
}
func (f *fakeDriverStore) Create(ctx context.Context, d *app.Driver) error {
	if f.createErr != nil {
		return f.createErr
	}
	f.created = d
	return nil
}
func (f *fakeDriverStore) Update(ctx context.Context, d *app.Driver) error { return nil }
func (f *fakeDriverStore) AssignVehicle(ctx context.Context, a *app.DriverAssignment) error {
	if f.assignErr != nil {
		return f.assignErr
	}
	return nil
}
func (f *fakeDriverStore) ListByVehicleIDsOnDate(ctx context.Context, vehicleIDs []uuid.UUID, on time.Time) (map[uuid.UUID][]app.Driver, error) {
	return nil, nil
}
func (f *fakeDriverStore) ReplaceVehicleDrivers(ctx context.Context, vehicleID uuid.UUID, driverIDs []uuid.UUID, effectiveFrom time.Time) error {
	if f.replaceErr != nil {
		return f.replaceErr
	}
	return nil
}
func (f *fakeDriverStore) SoftDelete(ctx context.Context, id, actorID uuid.UUID) (bool, error) {
	return true, nil
}
func (f *fakeDriverStore) CloseActiveAssignments(ctx context.Context, driverID uuid.UUID) error {
	return nil
}

func newTestDriverHandler(store *fakeDriverStore) *DriverHandler {
	return NewDriverHandler(app.NewDriverService(store, testDriverConfig(), nil))
}

func TestDriverHandler_Create_DuplicateNationalID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	store := &fakeDriverStore{createErr: app.ErrDuplicateNationalID}
	h := newTestDriverHandler(store)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body := `{"name":"測試司機","nationalId":"A202559750"}`
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/drivers", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	h.Create(c)

	require.Equal(t, http.StatusConflict, w.Code)
	assert.Contains(t, w.Body.String(), "已被其他司機使用")
}

func TestDriverHandler_Create_InvalidEmailFormat(t *testing.T) {
	gin.SetMode(gin.TestMode)

	store := &fakeDriverStore{}
	h := newTestDriverHandler(store)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body := `{"name":"測試司機","nationalId":"A202559750","email":"not-an-email"}`
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/drivers", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	h.Create(c)

	require.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "電子信箱格式不正確")
	assert.Nil(t, store.created)
}

func TestDriverHandler_Create_BlankEmailAllowed(t *testing.T) {
	gin.SetMode(gin.TestMode)

	store := &fakeDriverStore{}
	h := newTestDriverHandler(store)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body := `{"name":"測試司機","nationalId":"A202559750","email":""}`
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/drivers", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	h.Create(c)

	require.Equal(t, http.StatusCreated, w.Code)
	require.NotNil(t, store.created)
}

// TestDriverHandler_AssignVehicle_ReferenceInvalid_Returns400 確認指派了不存在的車輛
// （外鍵違反）會回 400 並附具體原因，而不是籠統的 500「系統發生錯誤」。
func TestDriverHandler_AssignVehicle_ReferenceInvalid_Returns400(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := &fakeDriverStore{assignErr: app.ErrAssignmentReferenceInvalid}
	h := newTestDriverHandler(store)
	driverID := uuid.New()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body := `{"vehicleId":"` + uuid.New().String() + `"}`
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/drivers/"+driverID.String()+"/vehicle", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: driverID.String()}}

	h.AssignVehicle(c)

	require.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "不存在")
}

// TestDriverHandler_AssignVehicle_Overlap_Returns409 確認指派期間重疊（違反不重疊限制）
// 會回 409 並附具體原因，而不是籠統的 500「系統發生錯誤」。
func TestDriverHandler_AssignVehicle_Overlap_Returns409(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := &fakeDriverStore{assignErr: app.ErrAssignmentOverlap}
	h := newTestDriverHandler(store)
	driverID := uuid.New()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body := `{"vehicleId":"` + uuid.New().String() + `"}`
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/drivers/"+driverID.String()+"/vehicle", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: driverID.String()}}

	h.AssignVehicle(c)

	require.Equal(t, http.StatusConflict, w.Code)
	assert.Contains(t, w.Body.String(), "已有其他車輛指派")
}
