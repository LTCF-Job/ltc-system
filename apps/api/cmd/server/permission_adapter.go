package main

import (
	"context"

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
