package app

import (
	"context"
	"sort"
	"strings"
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
	listCalls         int
}

type fakeUserDirectoryStore struct {
	users     map[uuid.UUID]AuthUser
	listCalls int
}

func (f *fakeUserDirectoryStore) ListDirectoryUsers(ctx context.Context, filter UserDirectoryFilter) ([]AuthUser, int, error) {
	f.listCalls++
	users := make([]AuthUser, 0, len(f.users))
	keyword := strings.ToLower(strings.TrimSpace(filter.Keyword))
	for _, user := range f.users {
		roleKey := user.RoleKey
		if roleKey == "" {
			roleKey = user.Role
		}
		if filter.RoleKey != "" && roleKey != filter.RoleKey {
			continue
		}
		if keyword != "" && !strings.Contains(strings.ToLower(user.Email), keyword) && !strings.Contains(strings.ToLower(user.DisplayName), keyword) {
			continue
		}
		user.RoleKey = roleKey
		user.Role = roleKey
		users = append(users, user)
	}
	sort.Slice(users, func(i, j int) bool { return users[i].Email < users[j].Email })
	total := len(users)
	start := (filter.Page - 1) * filter.PageSize
	if start >= total {
		return []AuthUser{}, total, nil
	}
	end := start + filter.PageSize
	if end > total {
		end = total
	}
	return users[start:end], total, nil
}

func (f *fakeUserDirectoryStore) UpsertDirectoryUser(ctx context.Context, user AuthUser) error {
	if f.users == nil {
		f.users = map[uuid.UUID]AuthUser{}
	}
	f.users[user.ID] = user
	return nil
}

func (f *fakeUserDirectoryStore) DeleteDirectoryUser(ctx context.Context, id uuid.UUID) error {
	delete(f.users, id)
	return nil
}

type fakeSecurityStateStore struct {
	states      map[uuid.UUID]UserSecurityState
	updateCalls int
	upsertCalls int
}

func (f *fakeSecurityStateStore) GetSecurityState(ctx context.Context, id uuid.UUID) (*UserSecurityState, error) {
	state, ok := f.states[id]
	if !ok {
		return nil, nil
	}
	return &state, nil
}

func (f *fakeSecurityStateStore) UpsertSecurityState(ctx context.Context, state UserSecurityState) error {
	if f.states == nil {
		f.states = map[uuid.UUID]UserSecurityState{}
	}
	f.states[state.UserID] = state
	f.upsertCalls++
	return nil
}

func (f *fakeSecurityStateStore) UpdateCustomPermissions(ctx context.Context, id uuid.UUID, perms map[string]ModulePermission) error {
	state, ok := f.states[id]
	if !ok {
		return assert.AnError
	}
	state.CustomPermissions = perms
	f.states[id] = state
	f.updateCalls++
	return nil
}

func (f *fakeSecurityStateStore) DeleteSecurityState(ctx context.Context, id uuid.UUID) error {
	delete(f.states, id)
	return nil
}

func (f *fakeAdminProvider) Configured() bool { return f.configured }

func (f *fakeAdminProvider) ListUsers(ctx context.Context) ([]AuthUser, error) {
	f.listCalls++
	var out []AuthUser
	for _, u := range f.users {
		out = append(out, *u)
	}
	return out, nil
}

func TestUserService_List_UsesLocalDirectoryProjectionAfterBootstrap(t *testing.T) {
	firstID := uuid.New()
	secondID := uuid.New()
	admin := &fakeAdminProvider{
		configured: true,
		users: map[uuid.UUID]*AuthUser{
			firstID: {ID: firstID, Email: "first@example.com", DisplayName: "First", RoleKey: "staff", Status: "active"},
		},
	}
	directory := &fakeUserDirectoryStore{}
	svc := NewUserService(admin, newFakeRoleStore(), &fakeIdentityAuditWriter{})
	svc.SetUserDirectoryStore(directory)

	users, total, err := svc.List(context.Background(), "", "", 1, 20)
	require.NoError(t, err)
	require.Len(t, users, 1)
	assert.Equal(t, 1, total)
	assert.Equal(t, 1, admin.listCalls, "首次查詢只應用外部 Admin API 做投影初始化")

	admin.users[secondID] = &AuthUser{ID: secondID, Email: "second@example.com", DisplayName: "Second", RoleKey: "staff", Status: "active"}
	users, total, err = svc.List(context.Background(), "", "", 1, 20)
	require.NoError(t, err)
	assert.Len(t, users, 1, "後續清單查詢應讀本地投影，不應重新掃描外部帳號")
	assert.Equal(t, 1, total)
	assert.Equal(t, 1, admin.listCalls)
	assert.Equal(t, 2, directory.listCalls)
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

	_, _, err := svc.List(ctx, "", "", 1, 20)
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

func TestUserService_Create_AuditSnapshotRedactsPersonalFields(t *testing.T) {
	admin := &fakeAdminProvider{configured: true, users: map[uuid.UUID]*AuthUser{}}
	audit := &fakeIdentityAuditWriter{}
	svc := NewUserService(admin, newFakeRoleStore(), audit)

	user, err := svc.Create(context.Background(), CreateAuthUserInput{
		Email:       "sensitive@example.com",
		Password:    "not-recorded",
		DisplayName: "敏感姓名",
		Phone:       "0912345678",
	}, uuid.New(), "admin")

	require.NoError(t, err)
	require.Len(t, audit.entries, 1)
	snapshot, ok := audit.entries[0].AfterData.(userAuditSnapshot)
	require.True(t, ok)
	assert.Equal(t, user.ID, snapshot.ID)
	assert.Equal(t, "[REDACTED]", snapshot.EmailMasked)
	assert.Empty(t, snapshot.RoleKey)
	assert.Equal(t, "active", snapshot.Status)
}

func TestUserService_Create_AuditFailureDoesNotReportCompletedExternalMutationAsFailed(t *testing.T) {
	admin := &fakeAdminProvider{configured: true, users: map[uuid.UUID]*AuthUser{}}
	audit := &fakeIdentityAuditWriter{err: assert.AnError}
	svc := NewUserService(admin, newFakeRoleStore(), audit)

	user, err := svc.Create(context.Background(), CreateAuthUserInput{Email: "created@example.com"}, uuid.New(), "admin")

	require.NoError(t, err)
	require.NotNil(t, user)
	_, exists := admin.users[user.ID]
	assert.True(t, exists, "外部帳號已建立時，稽核失敗不得讓呼叫端以為 mutation 未完成")
	assert.Len(t, audit.entries, 1)
}

func TestUserService_Update_RequiresAuditBeforeMutation(t *testing.T) {
	targetID := uuid.New()
	admin := &fakeAdminProvider{
		configured: true,
		users:      map[uuid.UUID]*AuthUser{targetID: {ID: targetID, Email: "a@example.com", RoleKey: "viewer"}},
	}
	svc := NewUserService(admin, newFakeRoleStore(), nil)

	_, err := svc.Update(context.Background(), targetID, UpdateAuthUserInput{}, uuid.New(), "admin")

	assert.ErrorIs(t, err, ErrAuditUnavailable)
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

func TestUserService_UpdatePermissions_MissingProjectionPreservesExternalSecurityState(t *testing.T) {
	targetID := uuid.New()
	admin := &fakeAdminProvider{
		configured: true,
		users: map[uuid.UUID]*AuthUser{
			targetID: {ID: targetID, Status: "inactive", RoleKey: "dispatcher"},
		},
	}
	stateStore := &fakeSecurityStateStore{states: map[uuid.UUID]UserSecurityState{}}
	svc := NewUserService(admin, newFakeRoleStore(), &fakeIdentityAuditWriter{})
	svc.SetUserSecurityStateStore(stateStore)

	err := svc.UpdatePermissions(context.Background(), targetID, map[string]ModulePermission{
		"settings_users": {View: true},
	}, uuid.New(), "admin")

	require.NoError(t, err)
	state, ok := stateStore.states[targetID]
	require.True(t, ok)
	assert.Equal(t, "inactive", state.Status, "缺少舊投影時不得以 repository 預設值把停用帳號變成啟用")
	assert.Equal(t, "dispatcher", state.RoleKey)
	assert.Equal(t, 0, stateStore.updateCalls)
	assert.Equal(t, 1, stateStore.upsertCalls)
}
