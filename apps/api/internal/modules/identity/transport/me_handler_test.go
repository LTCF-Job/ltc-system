package transport

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"ltc-system/apps/api/internal/platform/auth"
)

type stubRoleResolver struct{ perms map[string]map[string]auth.ModulePermission }

func (s stubRoleResolver) Resolve(_ context.Context, roleKey string) (map[string]auth.ModulePermission, error) {
	return s.perms[roleKey], nil
}

type stubCustomResolver struct{ perms map[uuid.UUID]map[string]auth.ModulePermission }

func (s stubCustomResolver) Resolve(_ context.Context, actorID uuid.UUID) (map[string]auth.ModulePermission, error) {
	return s.perms[actorID], nil
}

// newMeTestRouter 把 /auth/me 與一組受 RequirePermission 保護的探針路由掛在同一個 engine 上，
// 兩邊共用同一組 resolver，才能驗證前端讀到的權限與 API 實際放行的範圍一致。
func newMeTestRouter(actorID uuid.UUID, roleKey string, perm auth.PermissionResolver, custom auth.CustomPermissionResolver, modules []string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(auth.ContextKeyActorID, actorID)
		c.Set(auth.ContextKeyActorRole, roleKey)
		c.Set(auth.ContextKeyUserEmail, "dispatcher@example.com")
		c.Set(auth.ContextKeyActorName, "調度員")
		c.Next()
	})
	r.GET("/auth/me", NewMeHandler(perm, custom).Me)
	ok := func(c *gin.Context) { c.Status(http.StatusOK) }
	for _, m := range modules {
		for _, a := range []string{"view", "edit", "delete"} {
			r.GET(fmt.Sprintf("/probe/%s/%s", m, a), auth.RequirePermission(perm, custom, m, a), ok)
		}
	}
	return r
}

func getMePermissions(t *testing.T, r *gin.Engine) map[string]modulePermissionDTO {
	t.Helper()
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/auth/me", nil))
	require.Equal(t, http.StatusOK, w.Code)

	var body struct {
		Data meResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	return body.Data.Permissions
}

func TestMeHandler_PermissionsMatchRequirePermission(t *testing.T) {
	actorID := uuid.New()
	modules := []string{"settings_users", "settings_roles", "masters_cases"}
	perm := stubRoleResolver{perms: map[string]map[string]auth.ModulePermission{
		"dispatcher_1": {
			"settings_users": {View: true, Edit: true, Delete: false},
			"settings_roles": {View: true},
			"masters_cases":  {View: true, Edit: true, Delete: true},
		},
	}}
	// 個人覆蓋整包取代該模組，用來確認 /auth/me 也套用了同一份 merge 語意
	custom := stubCustomResolver{perms: map[uuid.UUID]map[string]auth.ModulePermission{
		actorID: {"settings_roles": {View: false, Edit: false, Delete: false}},
	}}
	r := newMeTestRouter(actorID, "dispatcher_1", perm, custom, modules)

	got := getMePermissions(t, r)

	for _, m := range modules {
		for _, a := range []string{"view", "edit", "delete"} {
			w := httptest.NewRecorder()
			r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, fmt.Sprintf("/probe/%s/%s", m, a), nil))

			allowed := map[string]bool{"view": got[m].View, "edit": got[m].Edit, "delete": got[m].Delete}[a]
			wantCode := http.StatusForbidden
			if allowed {
				wantCode = http.StatusOK
			}
			assert.Equal(t, wantCode, w.Code, "module=%s action=%s", m, a)
		}
	}
}

func TestMeHandler_ReturnsActorIdentity(t *testing.T) {
	actorID := uuid.New()
	r := newMeTestRouter(actorID, "dispatcher_1", stubRoleResolver{}, stubCustomResolver{}, nil)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/auth/me", nil))
	require.Equal(t, http.StatusOK, w.Code)

	var body struct {
		Data meResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, actorID.String(), body.Data.ID)
	assert.Equal(t, "dispatcher@example.com", body.Data.Email)
	assert.Equal(t, "調度員", body.Data.DisplayName)
	assert.Equal(t, "dispatcher_1", body.Data.Role)
	assert.NotNil(t, body.Data.Permissions)
}

// 自訂角色的矩陣沒有給 settings_users 時，/users 路由必須 403——遷移前 RequireRoles("admin")
// 對自訂角色是無條件拒絕，改走矩陣後拒絕的依據要來自資料而非角色字面值。
func TestRequirePermission_CustomRoleWithoutSettingsUsers_Forbidden(t *testing.T) {
	actorID := uuid.New()
	perm := stubRoleResolver{perms: map[string]map[string]auth.ModulePermission{
		"dispatcher_1": {"masters_cases": {View: true}},
	}}
	r := newMeTestRouter(actorID, "dispatcher_1", perm, stubCustomResolver{}, []string{"settings_users"})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/probe/settings_users/view", nil))
	assert.Equal(t, http.StatusForbidden, w.Code)
}
