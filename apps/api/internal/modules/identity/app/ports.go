package app

import (
	"context"

	"github.com/google/uuid"
)

// RoleStore 定義角色主檔的讀寫邊界。
type RoleStore interface {
	List(ctx context.Context) ([]Role, error)
	GetByID(ctx context.Context, id uuid.UUID) (*Role, error)
	GetByKey(ctx context.Context, key string) (*Role, error)
	Create(ctx context.Context, r *Role) error
	Update(ctx context.Context, r *Role) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// UserCounter 統計某角色目前被幾位使用者採用。
type UserCounter interface {
	CountUsersByRoleKey(ctx context.Context, key string) (int, error)
}

// AuditWriter 定義 identity 模組異動留痕的寫入邊界。
type AuditWriter interface {
	Write(ctx context.Context, e AuditEntry) error
}

// TxRunner 讓角色的建立／更新／刪除與稽核落在同一個資料庫交易內。
type TxRunner interface {
	WithTx(ctx context.Context, fn func(ctx context.Context) error) error
}

// PermissionCacheInvalidator 由 composition root 注入，避免 identity 模組直接依賴 platform/auth。
type PermissionCacheInvalidator interface {
	InvalidateRole(roleKey string)
	InvalidateUser(userID uuid.UUID)
}

// AdminIdentityProvider 是 Supabase Auth Admin API 的邊界；Configured 回 false 時
// 呼叫任何其他方法都必須回 ErrIdentityProviderUnconfigured，不得靜默退化成假資料。
type AdminIdentityProvider interface {
	Configured() bool
	ListUsers(ctx context.Context) ([]AuthUser, error)
	GetUser(ctx context.Context, id uuid.UUID) (*AuthUser, error)
	CreateUser(ctx context.Context, in CreateAuthUserInput) (*AuthUser, error)
	UpdateUser(ctx context.Context, id uuid.UUID, in UpdateAuthUserInput) (*AuthUser, error)
	DeleteUser(ctx context.Context, id uuid.UUID) error
	SetCustomPermissions(ctx context.Context, id uuid.UUID, perms map[string]ModulePermission) error
	VerifyPassword(ctx context.Context, email, password string) error
	SetPassword(ctx context.Context, id uuid.UUID, newPassword string) error
}
