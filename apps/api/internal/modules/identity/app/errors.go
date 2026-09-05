package app

import "errors"

var (
	// ErrIdentityProviderUnconfigured 表示 Supabase Service Role Key 尚未設定。
	ErrIdentityProviderUnconfigured = errors.New("identity provider is not configured")
	ErrAuditUnavailable             = errors.New("security audit is unavailable")
	ErrRoleNotFound                 = errors.New("role not found")
	ErrSystemRoleImmutable          = errors.New("system role cannot be modified or deleted")
	ErrRoleInUse                    = errors.New("role is still assigned to users")
	ErrUnknownRole                  = errors.New("unknown role key")
	// ErrUnknownModuleKey 表示權限矩陣含有未登記於 ModuleKeys 的功能模組 key。
	ErrUnknownModuleKey       = errors.New("unknown permission module key")
	ErrUserNotFound           = errors.New("user not found")
	ErrCannotDeleteSelf       = errors.New("cannot delete your own account")
	ErrInvalidCredentials     = errors.New("invalid credentials")
	ErrCannotResetOwnPassword = errors.New("cannot reset your own password through this endpoint")
)
