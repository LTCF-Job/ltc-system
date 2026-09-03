package main

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	identityapp "ltc-system/apps/api/internal/modules/identity/app"
)

// fakeAdminProvider 是 identityapp.AdminIdentityProvider 的最小假實作，只用來驗證
// userCustomPermissionResolver 的轉接邏輯，不涉及 identity 模組自己的業務規則測試。
type fakeAdminProvider struct {
	configured bool
	users      map[uuid.UUID]*identityapp.AuthUser
	getUserErr error
}

func (f *fakeAdminProvider) Configured() bool { return f.configured }
func (f *fakeAdminProvider) ListUsers(ctx context.Context) ([]identityapp.AuthUser, error) {
	return nil, nil
}
func (f *fakeAdminProvider) GetUser(ctx context.Context, id uuid.UUID) (*identityapp.AuthUser, error) {
	if f.getUserErr != nil {
		return nil, f.getUserErr
	}
	return f.users[id], nil
}
func (f *fakeAdminProvider) CreateUser(ctx context.Context, in identityapp.CreateAuthUserInput) (*identityapp.AuthUser, error) {
	return nil, nil
}
func (f *fakeAdminProvider) UpdateUser(ctx context.Context, id uuid.UUID, in identityapp.UpdateAuthUserInput) (*identityapp.AuthUser, error) {
	return nil, nil
}
func (f *fakeAdminProvider) DeleteUser(ctx context.Context, id uuid.UUID) error { return nil }
func (f *fakeAdminProvider) SetCustomPermissions(ctx context.Context, id uuid.UUID, perms map[string]identityapp.ModulePermission) error {
	return nil
}
func (f *fakeAdminProvider) VerifyPassword(ctx context.Context, email, password string) error {
	return nil
}
func (f *fakeAdminProvider) SetPassword(ctx context.Context, id uuid.UUID, newPassword string) error {
	return nil
}

func TestUserCustomPermissionResolver_NotConfigured_FailsOpenToNoOverride(t *testing.T) {
	resolver := userCustomPermissionResolver{admin: &fakeAdminProvider{configured: false}}

	perms, err := resolver.Resolve(context.Background(), uuid.New())

	require.NoError(t, err, "Admin API 未設定金鑰時應 fail-open，不得回傳 error 阻斷整條 RequirePermission 鏈路")
	assert.Nil(t, perms)
}

func TestUserCustomPermissionResolver_Configured_ReturnsUserOverride(t *testing.T) {
	actorID := uuid.New()
	resolver := userCustomPermissionResolver{admin: &fakeAdminProvider{
		configured: true,
		users: map[uuid.UUID]*identityapp.AuthUser{
			actorID: {ID: actorID, CustomPermissions: map[string]identityapp.ModulePermission{
				"masters_cases": {View: true, Edit: false, Delete: false},
			}},
		},
	}}

	perms, err := resolver.Resolve(context.Background(), actorID)

	require.NoError(t, err)
	require.NotNil(t, perms)
	assert.True(t, perms["masters_cases"].View)
	assert.False(t, perms["masters_cases"].Edit)
}

func TestUserCustomPermissionResolver_Configured_UserNotFound_ReturnsNoOverride(t *testing.T) {
	resolver := userCustomPermissionResolver{admin: &fakeAdminProvider{configured: true, users: map[uuid.UUID]*identityapp.AuthUser{}}}

	perms, err := resolver.Resolve(context.Background(), uuid.New())

	require.NoError(t, err)
	assert.Nil(t, perms)
}

func TestUserCustomPermissionResolver_Configured_GetUserError_Propagates(t *testing.T) {
	resolver := userCustomPermissionResolver{admin: &fakeAdminProvider{configured: true, getUserErr: assert.AnError}}

	_, err := resolver.Resolve(context.Background(), uuid.New())

	// 已設定金鑰但查詢本身失敗（網路、逾時等）不是 fail-open 範圍，應正常回傳 error
	// 讓 RequirePermission 視為系統錯誤，不能誤判為「沒有個人覆蓋」而放行。
	assert.ErrorIs(t, err, assert.AnError)
}
