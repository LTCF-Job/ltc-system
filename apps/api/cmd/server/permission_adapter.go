package main

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
	identityapp "ltc-system/apps/api/internal/modules/identity/app"
	"ltc-system/apps/api/internal/platform/auth"
)

// rolePermissionResolver 讓 auth.RequirePermission 透過 identity 模組的 RoleStore 解析角色的
// 模組權限矩陣，只存在於 composition root，platform 套件不需要直接依賴 identity 模組型別。
type rolePermissionResolver struct{ store identityapp.RoleStore }

func (r rolePermissionResolver) Resolve(ctx context.Context, roleKey string) (map[string]auth.ModulePermission, error) {
	role, err := r.store.GetByKey(ctx, roleKey)
	if err != nil {
		return nil, err
	}
	if role == nil {
		return nil, nil
	}
	return rolePermissions(role), nil
}

func rolePermissions(role *identityapp.Role) map[string]auth.ModulePermission {
	out := make(map[string]auth.ModulePermission, len(role.Permissions))
	for k, v := range role.Permissions {
		out[k] = auth.ModulePermission{View: v.View, Edit: v.Edit, Delete: v.Delete}
	}
	return out
}

// ResolveVersioned 使用 roles.updated_at 作為完整回源時保存的共享版本。
func (r rolePermissionResolver) ResolveVersioned(ctx context.Context, roleKey string) (map[string]auth.ModulePermission, string, error) {
	role, err := r.store.GetByKey(ctx, roleKey)
	if err != nil {
		return nil, "", err
	}
	if role == nil {
		return nil, "missing", nil
	}
	return rolePermissions(role), role.UpdatedAt.UTC().Format(time.RFC3339Nano), nil
}

// ResolveVersion 只讀取 roles.updated_at，讓 permission cache 在命中時仍能確認共享版本，
// 不必重新載入整份 permissions JSONB。RoleRepository 提供專用輕量查詢；舊的 RoleStore
// 實作則退回 GetByKey，維持測試替身與其他 adapter 的相容性。
func (r rolePermissionResolver) ResolveVersion(ctx context.Context, roleKey string) (string, error) {
	if versionStore, ok := r.store.(interface {
		GetPermissionVersion(context.Context, string) (string, error)
	}); ok {
		return versionStore.GetPermissionVersion(ctx, roleKey)
	}
	role, err := r.store.GetByKey(ctx, roleKey)
	if err != nil {
		return "", err
	}
	if role == nil {
		return "missing", nil
	}
	return role.UpdatedAt.UTC().Format(time.RFC3339Nano), nil
}

// userCustomPermissionResolver 讓 auth.RequirePermission 透過 identity 模組的
// AdminIdentityProvider 解析使用者個人層級的模組權限覆蓋，只存在於 composition root。
type userCustomPermissionResolver struct {
	admin identityapp.AdminIdentityProvider
}

// Resolve 解析使用者個人層級的模組權限覆蓋，未設定 Admin API 金鑰時回傳空覆蓋。
func (r userCustomPermissionResolver) Resolve(ctx context.Context, actorID uuid.UUID) (map[string]auth.ModulePermission, error) {
	// 只會在非 production 走到（production 缺 key 已由 config.LoadFromEnv 於啟動時擋下）：此時個人層級的 deny 覆蓋會消失，被降權的使用者回復為角色矩陣的完整權限
	if !r.admin.Configured() {
		return nil, nil
	}
	user, err := r.admin.GetUser(ctx, actorID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, nil
	}
	out := make(map[string]auth.ModulePermission, len(user.CustomPermissions))
	for k, v := range user.CustomPermissions {
		out[k] = auth.ModulePermission{View: v.View, Edit: v.Edit, Delete: v.Delete}
	}
	return out, nil
}

// userSecurityStateResolver 以 PostgreSQL 本地投影作為一般 request 的授權來源；只有尚未同步的
// 舊帳號才回源 Supabase 一次，避免每支 API 都依賴 Admin API。
type userSecurityStateResolver struct {
	store identityapp.UserSecurityStateStore
	admin identityapp.AdminIdentityProvider
}

func (r userSecurityStateResolver) load(ctx context.Context, actorID uuid.UUID) (*identityapp.UserSecurityState, error) {
	if state, loaded := securityStateFromRequest(ctx, actorID); loaded {
		return state, nil
	}

	if r.store != nil {
		state, err := r.store.GetSecurityState(ctx, actorID)
		if err != nil {
			return nil, err
		}
		if state != nil {
			rememberSecurityState(ctx, state)
			return state, nil
		}
	}
	if r.admin == nil || !r.admin.Configured() {
		rememberSecurityState(ctx, nil)
		return nil, nil
	}
	user, err := r.admin.GetUser(ctx, actorID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		rememberSecurityState(ctx, nil)
		return nil, nil
	}
	state := identityapp.UserSecurityStateFromAuthUser(*user)
	if state.Status == "" {
		state.Status = "active"
	}
	if r.store != nil {
		if err := r.store.UpsertSecurityState(ctx, state); err != nil {
			return nil, fmt.Errorf("sync user security state: %w", err)
		}
		stored, err := r.store.GetSecurityState(ctx, actorID)
		if err != nil {
			return nil, err
		}
		if stored != nil {
			rememberSecurityState(ctx, stored)
			return stored, nil
		}
	}
	rememberSecurityState(ctx, &state)
	return &state, nil
}

func securityStateFromRequest(ctx context.Context, actorID uuid.UUID) (*identityapp.UserSecurityState, bool) {
	requestState := auth.RequestSecurityStateFromContext(ctx)
	if requestState == nil || !requestState.Loaded {
		return nil, false
	}
	if !requestState.Found {
		return nil, true
	}
	version, _ := strconv.ParseInt(requestState.PermissionVersion, 10, 64)
	return &identityapp.UserSecurityState{
		UserID:            actorID,
		Status:            requestState.Status,
		RoleKey:           requestState.RoleKey,
		PermissionVersion: version,
		CustomPermissions: fromPlatformPermissions(requestState.CustomPermissions),
	}, true
}

func rememberSecurityState(ctx context.Context, state *identityapp.UserSecurityState) {
	requestState := auth.RequestSecurityStateFromContext(ctx)
	if requestState == nil {
		return
	}
	requestState.Loaded = true
	requestState.Found = state != nil
	if state == nil {
		requestState.Status = ""
		requestState.RoleKey = ""
		requestState.PermissionVersion = ""
		requestState.CustomPermissions = nil
		return
	}
	requestState.Status = state.Status
	requestState.RoleKey = state.RoleKey
	requestState.PermissionVersion = strconv.FormatInt(state.PermissionVersion, 10)
	requestState.CustomPermissions = toPlatformPermissions(state.CustomPermissions)
}

func (r userSecurityStateResolver) Resolve(ctx context.Context, actorID uuid.UUID) (map[string]auth.ModulePermission, error) {
	state, err := r.load(ctx, actorID)
	if err != nil || state == nil {
		return nil, err
	}
	return toPlatformPermissions(state.CustomPermissions), nil
}

func (r userSecurityStateResolver) ResolveVersioned(ctx context.Context, actorID uuid.UUID) (map[string]auth.ModulePermission, string, error) {
	state, err := r.load(ctx, actorID)
	if err != nil || state == nil {
		return nil, "missing", err
	}
	return toPlatformPermissions(state.CustomPermissions), fmt.Sprintf("%d", state.PermissionVersion), nil
}

// ResolveVersion 只讀取使用者安全狀態的 permission_version，讓 custom permission cache
// 在命中時不必重新反序列化整份個人覆蓋資料。
func (r userSecurityStateResolver) ResolveVersion(ctx context.Context, actorID uuid.UUID) (string, error) {
	if versionStore, ok := r.store.(interface {
		GetSecurityStateVersion(context.Context, uuid.UUID) (string, error)
	}); ok {
		return versionStore.GetSecurityStateVersion(ctx, actorID)
	}
	state, err := r.load(ctx, actorID)
	if err != nil {
		return "", err
	}
	if state == nil {
		return "missing", nil
	}
	return fmt.Sprintf("%d", state.PermissionVersion), nil
}

func (r userSecurityStateResolver) Validate(ctx context.Context, actorID uuid.UUID, role string) (bool, error) {
	active, _, err := r.ValidateVersioned(ctx, actorID, role)
	return active, err
}

func (r userSecurityStateResolver) ValidateVersioned(ctx context.Context, actorID uuid.UUID, role string) (bool, string, error) {
	state, err := r.load(ctx, actorID)
	if err != nil {
		return false, "error", err
	}
	if state == nil {
		return false, "missing", nil
	}
	roleMatches := state.RoleKey == "" || state.RoleKey == role
	return state.Status == "active" && roleMatches, fmt.Sprintf("%d", state.PermissionVersion), nil
}

// ValidateVersion 只讀取使用者安全狀態的 permission_version；狀態本身仍由 cache entry
// 保存，版本變更時 ValidateVersioned 才會重新載入並檢查 active／role。
func (r userSecurityStateResolver) ValidateVersion(ctx context.Context, actorID uuid.UUID) (string, error) {
	if versionStore, ok := r.store.(interface {
		GetSecurityStateVersion(context.Context, uuid.UUID) (string, error)
	}); ok {
		return versionStore.GetSecurityStateVersion(ctx, actorID)
	}
	state, err := r.load(ctx, actorID)
	if err != nil {
		return "", err
	}
	if state == nil {
		return "missing", nil
	}
	return fmt.Sprintf("%d", state.PermissionVersion), nil
}

func toPlatformPermissions(perms map[string]identityapp.ModulePermission) map[string]auth.ModulePermission {
	out := make(map[string]auth.ModulePermission, len(perms))
	for key, permission := range perms {
		out[key] = auth.ModulePermission{View: permission.View, Edit: permission.Edit, Delete: permission.Delete}
	}
	return out
}

func fromPlatformPermissions(perms map[string]auth.ModulePermission) map[string]identityapp.ModulePermission {
	out := make(map[string]identityapp.ModulePermission, len(perms))
	for key, permission := range perms {
		out[key] = identityapp.ModulePermission{View: permission.View, Edit: permission.Edit, Delete: permission.Delete}
	}
	return out
}
