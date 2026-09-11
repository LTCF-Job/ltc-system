package auth

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeVersionedUserStateResolver struct {
	state            UserState
	version          string
	callCount        int
	versionCallCount int
}

func (f *fakeVersionedUserStateResolver) Validate(ctx context.Context, actorID uuid.UUID, role string) (UserState, error) {
	f.callCount++
	return f.state, nil
}

func (f *fakeVersionedUserStateResolver) ValidateVersioned(ctx context.Context, actorID uuid.UUID, role string) (UserState, string, error) {
	f.callCount++
	return f.state, f.version, nil
}

func (f *fakeVersionedUserStateResolver) ValidateVersion(ctx context.Context, actorID uuid.UUID) (string, error) {
	f.versionCallCount++
	return f.version, nil
}

func TestCachedUserStateResolver_VersionedSourceDoesNotReloadStateOnCacheHit(t *testing.T) {
	resolver := &fakeVersionedUserStateResolver{state: UserStateActive, version: "v1"}
	cached := NewCachedUserStateResolver(resolver, time.Minute)
	actorID := uuid.New()

	state, err := cached.Validate(context.Background(), actorID, "staff")
	require.NoError(t, err)
	assert.Equal(t, UserStateActive, state)
	state, err = cached.Validate(context.Background(), actorID, "staff")
	require.NoError(t, err)
	assert.Equal(t, UserStateActive, state)

	assert.Equal(t, 1, resolver.callCount, "versioned source 在版本未變更時不應重新載入完整狀態")
	assert.Equal(t, 1, resolver.versionCallCount, "cache hit 只應查詢輕量版本")
}

func TestCachedUserStateResolver_VersionChangeReloadsState(t *testing.T) {
	resolver := &fakeVersionedUserStateResolver{state: UserStateActive, version: "v1"}
	cached := NewCachedUserStateResolver(resolver, time.Minute)
	actorID := uuid.New()

	state, err := cached.Validate(context.Background(), actorID, "staff")
	require.NoError(t, err)
	require.Equal(t, UserStateActive, state)
	resolver.version = "v2"
	resolver.state = UserStateDisabled

	state, err = cached.Validate(context.Background(), actorID, "staff")
	require.NoError(t, err)
	assert.Equal(t, UserStateDisabled, state)
	assert.Equal(t, 2, resolver.callCount, "版本變更後應重新驗證使用者狀態")
}

func TestCachedUserStateResolver_RefetchesVersionedSourceAfterTTL(t *testing.T) {
	resolver := &fakeVersionedUserStateResolver{state: UserStateActive, version: "v1"}
	cached := NewCachedUserStateResolver(resolver, time.Minute)
	actorID := uuid.New()
	cached.cache[actorID] = userStateCacheEntry{
		state:   UserStateActive,
		role:    "staff",
		version: "v1",
		expires: time.Now().Add(-time.Second),
	}

	_, err := cached.Validate(context.Background(), actorID, "staff")
	require.NoError(t, err)
	assert.Equal(t, 1, resolver.callCount)
}

// TestCachedUserStateResolver_DistinguishesDisabledFromRoleMismatch 確認快取會忠實保留
// 來源回傳的三種狀態，而不是把「角色不一致」跟「已停用」都壓成同一個布林值——這是
// 中介層能給出正確訊息的前提。
func TestCachedUserStateResolver_DistinguishesDisabledFromRoleMismatch(t *testing.T) {
	resolver := &fakeVersionedUserStateResolver{state: UserStateRoleMismatch, version: "v1"}
	cached := NewCachedUserStateResolver(resolver, time.Minute)
	actorID := uuid.New()

	state, err := cached.Validate(context.Background(), actorID, "staff")
	require.NoError(t, err)
	assert.Equal(t, UserStateRoleMismatch, state)
	assert.NotEqual(t, UserStateDisabled, state, "角色不一致不應被誤判為已停用")
}
