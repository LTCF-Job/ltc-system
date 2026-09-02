package app

import (
	"time"

	"github.com/google/uuid"
)

// Role 代表一個角色身分定義，含其功能模組權限矩陣。
type Role struct {
	ID          uuid.UUID
	Key         string
	Name        string
	Description string
	TagType     string
	IsSystem    bool
	BaseRole    string
	Permissions map[string]ModulePermission
	UserCount   int
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// ModulePermission 是單一功能模組的檢視／編輯權限。
type ModulePermission struct {
	View bool `json:"view"`
	Edit bool `json:"edit"`
}

// AuthUser 代表 Supabase Auth 的一個使用者帳號。
type AuthUser struct {
	ID                uuid.UUID
	Email             string
	DisplayName       string
	Phone             string
	Role              string
	RoleKey           string
	CustomPermissions map[string]ModulePermission
	Status            string
	CreatedAt         time.Time
	LastSignInAt      *time.Time
}

// CreateAuthUserInput 是建立使用者所需的欄位。
type CreateAuthUserInput struct {
	Email       string
	Password    string
	DisplayName string
	Phone       string
	RoleKey     string
}

// UpdateAuthUserInput 是更新使用者所需的欄位。
type UpdateAuthUserInput struct {
	DisplayName *string
	Phone       *string
	RoleKey     *string
	Status      *string
}

// AuditEntry 是本模組寫入稽核日誌的內容。
type AuditEntry struct {
	ActorID    *uuid.UUID
	ActorRole  *string
	Action     string
	EntityType string
	EntityID   *string
	BeforeData any
	AfterData  any
}
