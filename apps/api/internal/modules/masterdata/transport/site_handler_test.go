package transport

import (
	"context"
	"encoding/json"
	"errors"
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

// fakeSiteStore is a deterministic app.SiteStore test double.
type fakeSiteStore struct {
	created   *app.Site
	updated   *app.Site
	deletedID uuid.UUID
	deleteErr error
}

func (f *fakeSiteStore) List(ctx context.Context, region, q string, page, pageSize int) ([]app.Site, int64, error) {
	return nil, 0, nil
}

func (f *fakeSiteStore) GetByID(ctx context.Context, id uuid.UUID) (*app.Site, error) {
	return nil, errors.New("not found")
}

func (f *fakeSiteStore) Create(ctx context.Context, s *app.Site) error {
	f.created = s
	return nil
}

func (f *fakeSiteStore) Update(ctx context.Context, s *app.Site) error {
	f.updated = s
	return nil
}

func (f *fakeSiteStore) Delete(ctx context.Context, id uuid.UUID) error {
	f.deletedID = id
	return f.deleteErr
}

func newTestSiteHandler(store *fakeSiteStore) *SiteHandler {
	return NewSiteHandler(app.NewSiteService(store))
}

func TestSiteHandler_Create_BindsDTOAndPersists(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := &fakeSiteStore{}
	h := newTestSiteHandler(store)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body := `{"code":"S01","name":"竹北站","address":"竹北市文興路一段1號","region":"hsinchu","status":"active"}`
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/sites", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	h.Create(c)

	require.Equal(t, http.StatusCreated, w.Code)
	require.NotNil(t, store.created, "Create must actually call the store, not just echo the request")
	assert.Equal(t, "S01", store.created.Code)
	assert.Equal(t, "竹北站", store.created.Name)
	assert.Equal(t, "hsinchu", store.created.Region)
}

func TestSiteHandler_Update_PersistsChange(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := &fakeSiteStore{}
	h := newTestSiteHandler(store)
	id := uuid.New()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body := `{"code":"S01","name":"竹北站(更新)","address":"竹北市文興路一段1號","region":"hsinchu","status":"active"}`
	c.Request = httptest.NewRequest(http.MethodPatch, "/api/v1/sites/"+id.String(), strings.NewReader(body))
	c.Params = gin.Params{{Key: "id", Value: id.String()}}

	h.Update(c)

	require.Equal(t, http.StatusOK, w.Code)
	require.NotNil(t, store.updated, "Update must actually call the store, not just echo the request")
	assert.Equal(t, id, store.updated.ID)
	assert.Equal(t, "竹北站(更新)", store.updated.Name)
}

func TestSiteHandler_Delete_CallsStore(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := &fakeSiteStore{}
	h := newTestSiteHandler(store)
	id := uuid.New()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodDelete, "/api/v1/sites/"+id.String(), nil)
	c.Params = gin.Params{{Key: "id", Value: id.String()}}

	h.Delete(c)

	require.Equal(t, http.StatusNoContent, w.Code)
	assert.Equal(t, id, store.deletedID, "Delete must actually call the store, not just return success")
}

func TestSiteHandler_Delete_PropagatesStoreError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := &fakeSiteStore{deleteErr: errors.New("foreign key violation")}
	h := newTestSiteHandler(store)
	id := uuid.New()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodDelete, "/api/v1/sites/"+id.String(), nil)
	c.Params = gin.Params{{Key: "id", Value: id.String()}}

	h.Delete(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestSiteHandler_Create_ResponseShape 鎖定搬遷前後的 JSON 欄位契約：模組化不得
// 改變任何既有回應欄位名稱。
func TestSiteHandler_Create_ResponseShape(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := newTestSiteHandler(&fakeSiteStore{})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body := `{"code":"S01","name":"竹北站","address":"文興路","region":"hsinchu","openDays":[1,2],"status":"active"}`
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/sites", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	h.Create(c)

	var envelope struct {
		Data map[string]json.RawMessage `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &envelope))

	for _, field := range []string{"id", "code", "name", "address", "region", "openDays", "status", "createdAt", "updatedAt"} {
		_, ok := envelope.Data[field]
		assert.Truef(t, ok, "response must keep field %q", field)
	}
	assert.Len(t, envelope.Data, 9, "response must not gain or lose fields")
}
