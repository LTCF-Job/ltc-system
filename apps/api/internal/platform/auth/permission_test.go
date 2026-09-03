package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeResolver struct {
	perms       map[string]map[string]ModulePermission
	err         error
	callCount   int
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

// fakeCustomResolver 模擬 CustomPermissionResolver；預設（zero value）回 (nil, nil)，
// 等同 userCustomPermissionResolver 在 Admin API 未設定金鑰時的 fail-open 行為。
type fakeCustomResolver struct {
	perms     map[uuid.UUID]map[string]ModulePermission
	err       error
	callCount int
}

func (f *fakeCustomResolver) Resolve(ctx context.Context, actorID uuid.UUID) (map[string]ModulePermission, error) {
	f.callCount++
	if f.err != nil {
		return nil, f.err
	}
	return f.perms[actorID], nil
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

// performPermissionRequestAs 比照 performPermissionRequest，另外帶入 actorID，供
// CustomPermissionResolver 依使用者查詢個人覆蓋。
func performPermissionRequestAs(t *testing.T, h gin.HandlerFunc, actorID uuid.UUID, role string) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/cases", nil)
	c.Set(ContextKeyActorID, actorID)
	c.Set(ContextKeyActorRole, role)
	h(c)
	return w
}

func TestRequirePermission_NoActorRole(t *testing.T) {
	resolver := &fakeResolver{}
	w := performPermissionRequest(t, RequirePermission(resolver, &fakeCustomResolver{}, "masters_cases", "view"), "", false)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestRequirePermission_GrantedAction(t *testing.T) {
	resolver := &fakeResolver{perms: map[string]map[string]ModulePermission{
		"dispatcher": {"masters_cases": {View: true, Edit: true, Delete: false}},
	}}
	w := performPermissionRequest(t, RequirePermission(resolver, &fakeCustomResolver{}, "masters_cases", "edit"), "dispatcher", true)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRequirePermission_DeniedAction(t *testing.T) {
	resolver := &fakeResolver{perms: map[string]map[string]ModulePermission{
		"dispatcher": {"masters_cases": {View: true, Edit: true, Delete: false}},
	}}
	w := performPermissionRequest(t, RequirePermission(resolver, &fakeCustomResolver{}, "masters_cases", "delete"), "dispatcher", true)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestRequirePermission_UnknownModule(t *testing.T) {
	resolver := &fakeResolver{perms: map[string]map[string]ModulePermission{
		"dispatcher": {"masters_cases": {View: true, Edit: true}},
	}}
	w := performPermissionRequest(t, RequirePermission(resolver, &fakeCustomResolver{}, "masters_vehicles", "view"), "dispatcher", true)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestRequirePermission_UnknownRole(t *testing.T) {
	resolver := &fakeResolver{perms: map[string]map[string]ModulePermission{}}
	w := performPermissionRequest(t, RequirePermission(resolver, &fakeCustomResolver{}, "masters_cases", "view"), "not_a_real_role", true)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestRequirePermission_ResolverError(t *testing.T) {
	resolver := &fakeResolver{err: assert.AnError}
	w := performPermissionRequest(t, RequirePermission(resolver, &fakeCustomResolver{}, "masters_cases", "view"), "dispatcher", true)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestRequirePermission_CustomResolverError(t *testing.T) {
	resolver := &fakeResolver{perms: map[string]map[string]ModulePermission{
		"dispatcher": {"masters_cases": {View: true, Edit: true, Delete: false}},
	}}
	w := performPermissionRequest(t, RequirePermission(resolver, &fakeCustomResolver{err: assert.AnError}, "masters_cases", "view"), "dispatcher", true)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestRequirePermission_CustomPermissionOverridesModule_Granted(t *testing.T) {
	actorID := uuid.New()
	resolver := &fakeResolver{perms: map[string]map[string]ModulePermission{
		"staff": {"masters_cases": {View: true, Edit: false, Delete: false}},
	}}
	custom := &fakeCustomResolver{perms: map[uuid.UUID]map[string]ModulePermission{
		actorID: {"masters_cases": {View: true, Edit: true, Delete: false}},
	}}
	w := performPermissionRequestAs(t, RequirePermission(resolver, custom, "masters_cases", "edit"), actorID, "staff")
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRequirePermission_CustomPermissionOverridesModule_Denied(t *testing.T) {
	actorID := uuid.New()
	resolver := &fakeResolver{perms: map[string]map[string]ModulePermission{
		"staff": {"masters_cases": {View: true, Edit: true, Delete: false}},
	}}
	// 個人覆蓋只給 view，edit 依整包覆蓋語意視為 false，不是沿用角色矩陣的 true。
	custom := &fakeCustomResolver{perms: map[uuid.UUID]map[string]ModulePermission{
		actorID: {"masters_cases": {View: true, Edit: false, Delete: false}},
	}}
	w := performPermissionRequestAs(t, RequirePermission(resolver, custom, "masters_cases", "edit"), actorID, "staff")
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestRequirePermission_CustomPermissionOnlyAffectsOverriddenModule(t *testing.T) {
	actorID := uuid.New()
	resolver := &fakeResolver{perms: map[string]map[string]ModulePermission{
		"staff": {
			"masters_cases":    {View: true, Edit: true, Delete: false},
			"masters_vehicles": {View: true, Edit: true, Delete: false},
		},
	}}
	custom := &fakeCustomResolver{perms: map[uuid.UUID]map[string]ModulePermission{
		actorID: {"masters_cases": {View: true, Edit: false, Delete: false}},
	}}
	w := performPermissionRequestAs(t, RequirePermission(resolver, custom, "masters_vehicles", "edit"), actorID, "staff")
	assert.Equal(t, http.StatusOK, w.Code, "未被個人覆蓋的模組仍應照角色矩陣判斷")
}

func TestRequirePermission_NoCustomPermissions_FallsBackToRoleMatrix(t *testing.T) {
	resolver := &fakeResolver{perms: map[string]map[string]ModulePermission{
		"dispatcher": {"masters_cases": {View: true, Edit: true, Delete: false}},
	}}
	w := performPermissionRequestAs(t, RequirePermission(resolver, &fakeCustomResolver{}, "masters_cases", "edit"), uuid.New(), "dispatcher")
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRequirePermission_CustomPermissionForUnknownModule_NoEffectOnKnownModules(t *testing.T) {
	actorID := uuid.New()
	resolver := &fakeResolver{perms: map[string]map[string]ModulePermission{
		"staff": {"masters_cases": {View: true, Edit: true, Delete: false}},
	}}
	custom := &fakeCustomResolver{perms: map[uuid.UUID]map[string]ModulePermission{
		actorID: {"not_a_real_module": {View: true, Edit: true, Delete: true}},
	}}
	w := performPermissionRequestAs(t, RequirePermission(resolver, custom, "masters_cases", "edit"), actorID, "staff")
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRequirePermission_CustomPermissionDoesNotPolluteSharedRoleCache(t *testing.T) {
	userA, userB := uuid.New(), uuid.New()
	resolver := &fakeResolver{perms: map[string]map[string]ModulePermission{
		"staff": {"masters_cases": {View: true, Edit: true, Delete: false}},
	}}
	cached := NewCachedPermissionResolver(resolver)
	custom := &fakeCustomResolver{perms: map[uuid.UUID]map[string]ModulePermission{
		userA: {"masters_cases": {View: true, Edit: false, Delete: false}},
	}}
	handler := RequirePermission(cached, custom, "masters_cases", "edit")

	wA := performPermissionRequestAs(t, handler, userA, "staff")
	assert.Equal(t, http.StatusForbidden, wA.Code)

	// 使用者 B 同角色、無個人覆蓋，應仍拿到未被污染的角色矩陣原值。
	wB := performPermissionRequestAs(t, handler, userB, "staff")
	assert.Equal(t, http.StatusOK, wB.Code, "使用者 A 的個人覆蓋不應污染 CachedPermissionResolver 快取的共用角色矩陣")
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

func TestCachedCustomPermissionResolver_CachesWithinTTL(t *testing.T) {
	actorID := uuid.New()
	resolver := &fakeCustomResolver{perms: map[uuid.UUID]map[string]ModulePermission{
		actorID: {"masters_cases": {View: true, Edit: true, Delete: true}},
	}}
	cached := NewCachedCustomPermissionResolver(resolver)

	first, err := cached.Resolve(context.Background(), actorID)
	require.NoError(t, err)
	second, err := cached.Resolve(context.Background(), actorID)
	require.NoError(t, err)

	assert.Equal(t, first, second)
	assert.Equal(t, 1, resolver.callCount, "第二次查詢應該命中快取，不再打回 source")
}

func TestCachedCustomPermissionResolver_RefetchesAfterTTL(t *testing.T) {
	actorID := uuid.New()
	resolver := &fakeCustomResolver{perms: map[uuid.UUID]map[string]ModulePermission{
		actorID: {"masters_cases": {View: true, Edit: true, Delete: true}},
	}}
	cached := NewCachedCustomPermissionResolver(resolver)
	cached.cache[actorID] = permissionCacheEntry{
		perms:   resolver.perms[actorID],
		expires: time.Now().Add(-time.Second),
	}

	_, err := cached.Resolve(context.Background(), actorID)
	require.NoError(t, err)
	assert.Equal(t, 1, resolver.callCount, "快取過期後應該回源重新查詢一次")
}
