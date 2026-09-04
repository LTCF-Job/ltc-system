package app

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSeedDefaultAdmin_SkipsWhenEmailOrPasswordMissing(t *testing.T) {
	admin := &fakeAdminProvider{configured: true}
	roles := newFakeRoleStore()

	require.NoError(t, SeedDefaultAdmin(context.Background(), admin, roles, "", "pw"))
	require.NoError(t, SeedDefaultAdmin(context.Background(), admin, roles, "a@b.com", ""))
	assert.Empty(t, admin.users)
}

func TestSeedDefaultAdmin_SkipsWhenProviderUnconfigured(t *testing.T) {
	admin := &fakeAdminProvider{configured: false}
	roles := newFakeRoleStore()

	require.NoError(t, SeedDefaultAdmin(context.Background(), admin, roles, "a@b.com", "pw"))
	assert.Empty(t, admin.users)
}

func TestSeedDefaultAdmin_CreatesUserWithAdminRole(t *testing.T) {
	admin := &fakeAdminProvider{configured: true}
	roles := newFakeRoleStore()
	roles.byKey[defaultAdminRoleKey] = &Role{Key: defaultAdminRoleKey}

	err := SeedDefaultAdmin(context.Background(), admin, roles, "ltcf-admin@ltc.example.com", "ltcf-admin_1234")
	require.NoError(t, err)

	require.Len(t, admin.users, 1)
	for _, u := range admin.users {
		assert.Equal(t, "ltcf-admin@ltc.example.com", u.Email)
		assert.Equal(t, defaultAdminRoleKey, u.RoleKey)
	}
}

func TestSeedDefaultAdmin_IdempotentWhenEmailAlreadyExists(t *testing.T) {
	admin := &fakeAdminProvider{configured: true}
	roles := newFakeRoleStore()
	roles.byKey[defaultAdminRoleKey] = &Role{Key: defaultAdminRoleKey}

	require.NoError(t, SeedDefaultAdmin(context.Background(), admin, roles, "Admin@Example.com", "pw1"))
	require.Len(t, admin.users, 1)

	// 同一 email 大小寫不同再次呼叫，確認不會重複建立第二個帳號。
	require.NoError(t, SeedDefaultAdmin(context.Background(), admin, roles, "admin@example.com", "pw2"))
	assert.Len(t, admin.users, 1)
}

func TestSeedDefaultAdmin_ErrorsWhenAdminRoleMissing(t *testing.T) {
	admin := &fakeAdminProvider{configured: true}
	roles := newFakeRoleStore()

	err := SeedDefaultAdmin(context.Background(), admin, roles, "a@b.com", "pw")
	assert.ErrorContains(t, err, "admin")
	assert.Empty(t, admin.users)
}
