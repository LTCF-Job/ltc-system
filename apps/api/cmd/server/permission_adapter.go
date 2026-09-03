package main

import (
	"context"

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
	out := make(map[string]auth.ModulePermission, len(role.Permissions))
	for k, v := range role.Permissions {
		out[k] = auth.ModulePermission{View: v.View, Edit: v.Edit, Delete: v.Delete}
	}
	return out, nil
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
