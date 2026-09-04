package app

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

const defaultAdminRoleKey = "admin"

// SeedDefaultAdmin 確保 email 對應的管理員帳號存在，供部署時自動建立初始管理員使用。
// email 或 password 任一為空、或 admin provider 未設定時視為跳過；已存在同 email 帳號時
// 視為已完成（冪等），不覆寫既有密碼或角色，可安全地在每次部署時重複呼叫。
func SeedDefaultAdmin(ctx context.Context, admin AdminIdentityProvider, roles RoleStore, email, password string) error {
	if email == "" || password == "" {
		return nil
	}
	if admin == nil || !admin.Configured() {
		return nil
	}

	users, err := admin.ListUsers(ctx)
	if err != nil {
		return fmt.Errorf("failed to list users: %w", err)
	}
	for _, u := range users {
		if strings.EqualFold(u.Email, email) {
			return nil
		}
	}

	role, err := roles.GetByKey(ctx, defaultAdminRoleKey)
	if err != nil {
		return fmt.Errorf("failed to look up %q role: %w", defaultAdminRoleKey, err)
	}
	if role == nil {
		return errors.New("system role \"admin\" not found; run migrations before seeding the default admin")
	}

	if _, err := admin.CreateUser(ctx, CreateAuthUserInput{
		Email:       email,
		Password:    password,
		DisplayName: "系統管理員",
		RoleKey:     role.Key,
	}); err != nil {
		return fmt.Errorf("failed to create default admin user: %w", err)
	}
	return nil
}
