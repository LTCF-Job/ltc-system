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
	ErrInvalidUserStatus      = errors.New("invalid user status")
	// ErrEmailAlreadyExists／ErrWeakPassword 讓 SupabaseAdminClient 能把常見的 Auth Admin
	// API 非 2xx 情境辨識出來，而不是全部折成一個看不出原因的內部錯誤；帳號不存在則直接沿用
	// 既有的 ErrUserNotFound，維持與其他「查無使用者」情境相同的 404 映射。
	ErrEmailAlreadyExists = errors.New("email already exists")
	ErrWeakPassword       = errors.New("password does not meet strength requirements")
)
