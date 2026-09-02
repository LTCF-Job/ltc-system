package app

import "errors"

var (
	// ErrIdentityProviderUnconfigured 表示 Supabase Service Role Key 尚未設定。
	ErrIdentityProviderUnconfigured = errors.New("identity provider is not configured")
	ErrRoleNotFound                 = errors.New("role not found")
	ErrSystemRoleImmutable          = errors.New("system role cannot be modified or deleted")
	ErrRoleInUse                    = errors.New("role is still assigned to users")
	ErrUnknownRole                  = errors.New("unknown role key")
	ErrUserNotFound                 = errors.New("user not found")
	ErrCannotDeleteSelf             = errors.New("cannot delete your own account")
	ErrInvalidCredentials           = errors.New("invalid credentials")
)
