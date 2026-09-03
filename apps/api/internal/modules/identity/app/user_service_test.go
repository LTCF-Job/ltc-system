package app

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeAdminProvider struct {
	configured        bool
	users             map[uuid.UUID]*AuthUser
	verifyErr         error
	setPasswordCalled bool
	deleteCalled      bool
}

func (f *fakeAdminProvider) Configured() bool { return f.configured }

func (f *fakeAdminProvider) ListUsers(ctx context.Context) ([]AuthUser, error) {
	var out []AuthUser
	for _, u := range f.users {
		out = append(out, *u)
	}
	return out, nil
}

func (f *fakeAdminProvider) GetUser(ctx context.Context, id uuid.UUID) (*AuthUser, error) {
	if u, ok := f.users[id]; ok {
		return u, nil
	}
	return nil, nil
}

func (f *fakeAdminProvider) CreateUser(ctx context.Context, in CreateAuthUserInput) (*AuthUser, error) {
	u := &AuthUser{ID: uuid.New(), Email: in.Email, DisplayName: in.DisplayName, RoleKey: in.RoleKey}
	if f.users == nil {
		f.users = map[uuid.UUID]*AuthUser{}
	}
	f.users[u.ID] = u
	return u, nil
}

func (f *fakeAdminProvider) UpdateUser(ctx context.Context, id uuid.UUID, in UpdateAuthUserInput) (*AuthUser, error) {
	u := f.users[id]
	return u, nil
}

func (f *fakeAdminProvider) DeleteUser(ctx context.Context, id uuid.UUID) error {
	f.deleteCalled = true
	delete(f.users, id)
	return nil
}

func (f *fakeAdminProvider) SetCustomPermissions(ctx context.Context, id uuid.UUID, perms map[string]ModulePermission) error {
	return nil
}

func (f *fakeAdminProvider) VerifyPassword(ctx context.Context, email, password string) error {
	return f.verifyErr
}

func (f *fakeAdminProvider) SetPassword(ctx context.Context, id uuid.UUID, newPassword string) error {
	f.setPasswordCalled = true
	return nil
}

func TestUserService_UnconfiguredReturns503ForEveryMethod(t *testing.T) {
	admin := &fakeAdminProvider{configured: false}
	svc := NewUserService(admin, newFakeRoleStore(), nil)
	ctx := context.Background()
	actorID := uuid.New()

	_, err := svc.List(ctx, "", "")
	assert.ErrorIs(t, err, ErrIdentityProviderUnconfigured)

	_, err = svc.Get(ctx, uuid.New())
	assert.ErrorIs(t, err, ErrIdentityProviderUnconfigured)

	_, err = svc.Create(ctx, CreateAuthUserInput{}, actorID, "admin")
	assert.ErrorIs(t, err, ErrIdentityProviderUnconfigured)

	_, err = svc.Update(ctx, uuid.New(), UpdateAuthUserInput{}, actorID, "admin")
	assert.ErrorIs(t, err, ErrIdentityProviderUnconfigured)

	err = svc.UpdatePermissions(ctx, uuid.New(), nil, actorID, "admin")
	assert.ErrorIs(t, err, ErrIdentityProviderUnconfigured)

	err = svc.Delete(ctx, uuid.New(), actorID, "admin")
	assert.ErrorIs(t, err, ErrIdentityProviderUnconfigured)

	err = svc.ChangeSelfPassword(ctx, actorID, "a@example.com", "old", "newpass1")
	assert.ErrorIs(t, err, ErrIdentityProviderUnconfigured)

	err = svc.ResetPassword(ctx, uuid.New(), actorID, "admin", "newpass1")
	assert.ErrorIs(t, err, ErrIdentityProviderUnconfigured)

	assert.False(t, admin.setPasswordCalled, "未設定金鑰時不應呼叫 Admin API")
	assert.False(t, admin.deleteCalled)
}

func TestUserService_Delete_CannotDeleteSelf(t *testing.T) {
	admin := &fakeAdminProvider{configured: true, users: map[uuid.UUID]*AuthUser{}}
	svc := NewUserService(admin, newFakeRoleStore(), nil)
	actorID := uuid.New()

	err := svc.Delete(context.Background(), actorID, actorID, "admin")
	assert.ErrorIs(t, err, ErrCannotDeleteSelf)
	assert.False(t, admin.deleteCalled)
}

func TestUserService_Create_RejectsUnknownRole(t *testing.T) {
	admin := &fakeAdminProvider{configured: true, users: map[uuid.UUID]*AuthUser{}}
	roleStore := newFakeRoleStore()
	svc := NewUserService(admin, roleStore, nil)

	_, err := svc.Create(context.Background(), CreateAuthUserInput{Email: "a@example.com", RoleKey: "not_a_role"}, uuid.New(), "admin")
	assert.ErrorIs(t, err, ErrUnknownRole)
}

func TestUserService_ChangeSelfPassword_WrongOldPasswordDoesNotCallSetPassword(t *testing.T) {
	admin := &fakeAdminProvider{configured: true, verifyErr: assert.AnError}
	svc := NewUserService(admin, newFakeRoleStore(), nil)

	err := svc.ChangeSelfPassword(context.Background(), uuid.New(), "a@example.com", "wrong-old-password", "newpass1")
	assert.ErrorIs(t, err, ErrInvalidCredentials)
	assert.False(t, admin.setPasswordCalled, "舊密碼驗證失敗不應呼叫 Admin API 設定新密碼")
}

func TestUserService_ChangeSelfPassword_Success(t *testing.T) {
	admin := &fakeAdminProvider{configured: true}
	audit := &fakeIdentityAuditWriter{}
	svc := NewUserService(admin, newFakeRoleStore(), audit)

	err := svc.ChangeSelfPassword(context.Background(), uuid.New(), "a@example.com", "old-password", "newpass1")
	require.NoError(t, err)
	assert.True(t, admin.setPasswordCalled)
	require.Len(t, audit.entries, 1)
	assert.Equal(t, "change_password", audit.entries[0].Action)
}

func TestUserService_ResetPassword_CannotResetOwnAccount(t *testing.T) {
	admin := &fakeAdminProvider{configured: true}
	svc := NewUserService(admin, newFakeRoleStore(), nil)
	actorID := uuid.New()

	err := svc.ResetPassword(context.Background(), actorID, actorID, "admin", "newpass1")
	assert.ErrorIs(t, err, ErrCannotResetOwnPassword)
	assert.False(t, admin.setPasswordCalled, "重設自己的密碼應被拒絕，不應呼叫 Admin API")
}

func TestUserService_ResetPassword_Success(t *testing.T) {
	admin := &fakeAdminProvider{configured: true}
	audit := &fakeIdentityAuditWriter{}
	svc := NewUserService(admin, newFakeRoleStore(), audit)
	targetID := uuid.New()

	err := svc.ResetPassword(context.Background(), targetID, uuid.New(), "admin", "newpass1")
	require.NoError(t, err)
	assert.True(t, admin.setPasswordCalled, "重設他人密碼不需驗證舊密碼，應直接呼叫 Admin API")
	require.Len(t, audit.entries, 1)
	assert.Equal(t, "reset_password", audit.entries[0].Action)
}
