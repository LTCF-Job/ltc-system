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

// Resolve 在 Admin API 未設定金鑰時 fail-open 為「沒有個人覆蓋」（回 nil, nil）：這是
// AdminIdentityProvider「未設定必須 fail-loud」規則的刻意例外——customPermissions 是疊加在
// 角色矩陣之上的可選層，未設定時退回純角色矩陣判斷，不能讓它拖垮所有走 RequirePermission 的路由。
// 已設定但查詢本身失敗（網路、逾時等）則正常回傳 error，交由呼叫端視為系統錯誤處理。
func (r userCustomPermissionResolver) Resolve(ctx context.Context, actorID uuid.UUID) (map[string]auth.ModulePermission, error) {
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
