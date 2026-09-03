package app

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRoleService_Create_RejectsUnknownModuleKey(t *testing.T) {
	svc := NewRoleService(newFakeRoleStore(), nil, nil, nil)
	_, err := svc.Create(context.Background(), CreateRoleInput{
		Name: "調度員",
		Permissions: map[string]ModulePermission{
			"masters_cases": {View: true},
			"not_a_module":  {View: true},
			"also_bogus":    {View: true},
		},
	}, uuid.New(), "admin")

	require.ErrorIs(t, err, ErrUnknownModuleKey)
	assert.Contains(t, err.Error(), "also_bogus, not_a_module")
}

func TestRoleService_Create_AcceptsRegisteredModuleKeys(t *testing.T) {
	svc := NewRoleService(newFakeRoleStore(), nil, nil, nil)
	perms := map[string]ModulePermission{}
	for _, k := range ModuleKeys {
		perms[k] = ModulePermission{View: true}
	}
	role, err := svc.Create(context.Background(), CreateRoleInput{Name: "調度員", Permissions: perms}, uuid.New(), "admin")

	require.NoError(t, err)
	assert.Len(t, role.Permissions, len(ModuleKeys))
}

func TestRoleService_Update_RejectsUnknownModuleKey(t *testing.T) {
	store := newFakeRoleStore()
	id := uuid.New()
	store.roles[id] = &Role{ID: id, Key: "dispatcher_1"}
	svc := NewRoleService(store, nil, nil, nil)

	_, err := svc.Update(context.Background(), id, UpdateRoleInput{
		Permissions: map[string]ModulePermission{"not_a_module": {View: true}},
	}, uuid.New(), "admin")

	require.ErrorIs(t, err, ErrUnknownModuleKey)
}

func TestUserService_UpdatePermissions_RejectsUnknownModuleKey(t *testing.T) {
	admin := &fakeAdminProvider{configured: true, users: map[uuid.UUID]*AuthUser{}}
	svc := NewUserService(admin, newFakeRoleStore(), nil)

	err := svc.UpdatePermissions(context.Background(), uuid.New(), map[string]ModulePermission{
		"settings_users": {View: true},
		"not_a_module":   {View: true},
	}, uuid.New(), "admin")

	require.ErrorIs(t, err, ErrUnknownModuleKey)
	assert.Contains(t, err.Error(), "not_a_module")
}

func TestUserService_UpdatePermissions_AcceptsRegisteredModuleKeys(t *testing.T) {
	admin := &fakeAdminProvider{configured: true, users: map[uuid.UUID]*AuthUser{}}
	svc := NewUserService(admin, newFakeRoleStore(), nil)

	err := svc.UpdatePermissions(context.Background(), uuid.New(), map[string]ModulePermission{
		"settings_users": {View: true, Edit: true, Delete: true},
	}, uuid.New(), "admin")

	require.NoError(t, err)
}
