package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeResolver struct {
	perms      map[string]map[string]ModulePermission
	err        error
	callCount  int
	lastQueried string
}

func (f *fakeResolver) Resolve(ctx context.Context, roleKey string) (map[string]ModulePermission, error) {
	f.callCount++
	f.lastQueried = roleKey
	if f.err != nil {
		return nil, f.err
	}
	return f.perms[roleKey], nil
}

func performPermissionRequest(t *testing.T, h gin.HandlerFunc, role string, setRole bool) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/cases", nil)
	if setRole {
		c.Set(ContextKeyActorRole, role)
	}
	h(c)
	return w
}

func TestRequirePermission_NoActorRole(t *testing.T) {
	resolver := &fakeResolver{}
	w := performPermissionRequest(t, RequirePermission(resolver, "masters_cases", "view"), "", false)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestRequirePermission_GrantedAction(t *testing.T) {
	resolver := &fakeResolver{perms: map[string]map[string]ModulePermission{
		"dispatcher": {"masters_cases": {View: true, Edit: true, Delete: false}},
	}}
	w := performPermissionRequest(t, RequirePermission(resolver, "masters_cases", "edit"), "dispatcher", true)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRequirePermission_DeniedAction(t *testing.T) {
	resolver := &fakeResolver{perms: map[string]map[string]ModulePermission{
		"dispatcher": {"masters_cases": {View: true, Edit: true, Delete: false}},
	}}
	w := performPermissionRequest(t, RequirePermission(resolver, "masters_cases", "delete"), "dispatcher", true)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestRequirePermission_UnknownModule(t *testing.T) {
	resolver := &fakeResolver{perms: map[string]map[string]ModulePermission{
		"dispatcher": {"masters_cases": {View: true, Edit: true}},
	}}
	w := performPermissionRequest(t, RequirePermission(resolver, "masters_vehicles", "view"), "dispatcher", true)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestRequirePermission_UnknownRole(t *testing.T) {
	resolver := &fakeResolver{perms: map[string]map[string]ModulePermission{}}
	w := performPermissionRequest(t, RequirePermission(resolver, "masters_cases", "view"), "not_a_real_role", true)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestRequirePermission_ResolverError(t *testing.T) {
	resolver := &fakeResolver{err: assert.AnError}
	w := performPermissionRequest(t, RequirePermission(resolver, "masters_cases", "view"), "dispatcher", true)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestCachedPermissionResolver_CachesWithinTTL(t *testing.T) {
	resolver := &fakeResolver{perms: map[string]map[string]ModulePermission{
		"admin": {"masters_cases": {View: true, Edit: true, Delete: true}},
	}}
	cached := NewCachedPermissionResolver(resolver)

	first, err := cached.Resolve(context.Background(), "admin")
	require.NoError(t, err)
	second, err := cached.Resolve(context.Background(), "admin")
	require.NoError(t, err)

	assert.Equal(t, first, second)
	assert.Equal(t, 1, resolver.callCount, "第二次查詢應該命中快取，不再打回 source")
}

func TestCachedPermissionResolver_RefetchesAfterTTL(t *testing.T) {
	resolver := &fakeResolver{perms: map[string]map[string]ModulePermission{
		"admin": {"masters_cases": {View: true, Edit: true, Delete: true}},
	}}
	cached := NewCachedPermissionResolver(resolver)
	cached.cache["admin"] = permissionCacheEntry{
		perms:   resolver.perms["admin"],
		expires: time.Now().Add(-time.Second),
	}

	_, err := cached.Resolve(context.Background(), "admin")
	require.NoError(t, err)
	assert.Equal(t, 1, resolver.callCount, "快取過期後應該回源重新查詢一次")
}
