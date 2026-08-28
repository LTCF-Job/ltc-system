package handler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"ltc-system/apps/api/internal/repository"
	"ltc-system/apps/api/internal/service"
)

// fakeSiteStore is a deterministic service.SiteStore test double.
type fakeSiteStore struct {
	created   *repository.SiteEntity
	updated   *repository.SiteEntity
	deletedID uuid.UUID
	deleteErr error
}

func (f *fakeSiteStore) List(ctx context.Context, region, q string, page, pageSize int) ([]repository.SiteEntity, int64, error) {
	return nil, 0, nil
}

func (f *fakeSiteStore) Create(ctx context.Context, s *repository.SiteEntity) error {
	f.created = s
	return nil
}

func (f *fakeSiteStore) Update(ctx context.Context, s *repository.SiteEntity) error {
	f.updated = s
	return nil
}

func (f *fakeSiteStore) Delete(ctx context.Context, id uuid.UUID) error {
	f.deletedID = id
	return f.deleteErr
}

func newTestSiteHandler(store *fakeSiteStore) *SiteHandler {
	return NewSiteHandler(service.NewSiteService(store))
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
