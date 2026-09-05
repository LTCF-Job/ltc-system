package main

import (
	"github.com/google/uuid"
	"ltc-system/apps/api/internal/platform/auth"
)

// permissionCacheInvalidator 將兩種 platform cache 組合成 identity 模組需要的單一 port。
type permissionCacheInvalidator struct {
	roles *auth.CachedPermissionResolver
	users *auth.CachedCustomPermissionResolver
}

func (i permissionCacheInvalidator) InvalidateRole(roleKey string) {
	if i.roles != nil {
		i.roles.InvalidateRole(roleKey)
	}
}

func (i permissionCacheInvalidator) InvalidateUser(userID uuid.UUID) {
	if i.users != nil {
		i.users.InvalidateUser(userID)
	}
}
