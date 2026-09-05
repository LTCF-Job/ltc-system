package app

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeRoleStore struct {
	roles     map[uuid.UUID]*Role
	byKey     map[string]*Role
	createErr error
}

func newFakeRoleStore() *fakeRoleStore {
	return &fakeRoleStore{roles: map[uuid.UUID]*Role{}, byKey: map[string]*Role{}}
}

func (f *fakeRoleStore) List(ctx context.Context) ([]Role, error) {
	var out []Role
	for _, r := range f.roles {
		out = append(out, *r)
	}
	return out, nil
}

func (f *fakeRoleStore) GetByID(ctx context.Context, id uuid.UUID) (*Role, error) {
	if r, ok := f.roles[id]; ok {
		return r, nil
	}
	return nil, nil
}

func (f *fakeRoleStore) GetByKey(ctx context.Context, key string) (*Role, error) {
	if r, ok := f.byKey[key]; ok {
		return r, nil
	}
	return nil, nil
}

func (f *fakeRoleStore) Create(ctx context.Context, r *Role) error {
	if f.createErr != nil {
		return f.createErr
	}
	cp := *r
	f.roles[r.ID] = &cp
	f.byKey[r.Key] = &cp
	return nil
}

func (f *fakeRoleStore) Update(ctx context.Context, r *Role) error {
	cp := *r
	f.roles[r.ID] = &cp
	return nil
}

func (f *fakeRoleStore) Delete(ctx context.Context, id uuid.UUID) error {
	if r, ok := f.roles[id]; ok {
		delete(f.byKey, r.Key)
	}
	delete(f.roles, id)
	return nil
}

type fakeUserCounter struct {
	counts map[string]int
}

func (f *fakeUserCounter) CountUsersByRoleKey(ctx context.Context, key string) (int, error) {
	return f.counts[key], nil
}

type fakeIdentityAuditWriter struct {
	entries []AuditEntry
}

func (f *fakeIdentityAuditWriter) Write(_ context.Context, e AuditEntry) error {
	f.entries = append(f.entries, e)
	return nil
}

func TestRoleService_Update_SystemRoleImmutable(t *testing.T) {
	store := newFakeRoleStore()
	id := uuid.New()
	store.roles[id] = &Role{ID: id, Key: "admin", IsSystem: true}
	svc := NewRoleService(store, nil, nil, nil)

	name := "改名"
	_, err := svc.Update(context.Background(), id, UpdateRoleInput{Name: &name}, uuid.New(), "admin")
	assert.ErrorIs(t, err, ErrSystemRoleImmutable)
}

func TestRoleService_Delete_SystemRoleImmutable(t *testing.T) {
	store := newFakeRoleStore()
	id := uuid.New()
	store.roles[id] = &Role{ID: id, Key: "admin", IsSystem: true}
	svc := NewRoleService(store, nil, nil, nil)

	err := svc.Delete(context.Background(), id, uuid.New(), "admin")
	assert.ErrorIs(t, err, ErrSystemRoleImmutable)
}

func TestRoleService_Delete_RoleInUse(t *testing.T) {
	store := newFakeRoleStore()
	id := uuid.New()
	store.roles[id] = &Role{ID: id, Key: "dispatcher", IsSystem: false}
	counter := &fakeUserCounter{counts: map[string]int{"dispatcher": 2}}
	svc := NewRoleService(store, counter, nil, nil)

	err := svc.Delete(context.Background(), id, uuid.New(), "admin")
	assert.ErrorIs(t, err, ErrRoleInUse)
}

func TestRoleService_Delete_Success(t *testing.T) {
	store := newFakeRoleStore()
	id := uuid.New()
	store.roles[id] = &Role{ID: id, Key: "custom_role", IsSystem: false}
	counter := &fakeUserCounter{counts: map[string]int{}}
	audit := &fakeIdentityAuditWriter{}
	svc := NewRoleService(store, counter, audit, nil)

	err := svc.Delete(context.Background(), id, uuid.New(), "admin")
	require.NoError(t, err)
	_, exists := store.roles[id]
	assert.False(t, exists)
	require.Len(t, audit.entries, 1)
	assert.Equal(t, "delete", audit.entries[0].Action)
}

func TestRoleService_Create_SlugConflictAppendsSuffix(t *testing.T) {
	store := newFakeRoleStore()
	store.byKey["dispatcher"] = &Role{Key: "dispatcher"}
	svc := NewRoleService(store, nil, &fakeIdentityAuditWriter{}, nil)

	role, err := svc.Create(context.Background(), CreateRoleInput{Name: "dispatcher"}, uuid.New(), "admin")
	require.NoError(t, err)
	assert.Equal(t, "dispatcher_1", role.Key)
}

func TestRoleService_Create_DefaultsBaseRoleToViewer(t *testing.T) {
	store := newFakeRoleStore()
	svc := NewRoleService(store, nil, &fakeIdentityAuditWriter{}, nil)

	role, err := svc.Create(context.Background(), CreateRoleInput{Name: "custom"}, uuid.New(), "admin")
	require.NoError(t, err)
	assert.Equal(t, "viewer", role.BaseRole)
}

func TestRoleService_List_FillsUserCounts(t *testing.T) {
	store := newFakeRoleStore()
	id := uuid.New()
	store.roles[id] = &Role{ID: id, Key: "dispatcher"}
	counter := &fakeUserCounter{counts: map[string]int{"dispatcher": 3}}
	svc := NewRoleService(store, counter, nil, nil)

	roles, err := svc.List(context.Background())
	require.NoError(t, err)
	require.Len(t, roles, 1)
	assert.Equal(t, 3, roles[0].UserCount)
}
