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
	active           bool
	version          string
	callCount        int
	versionCallCount int
}

func (f *fakeVersionedUserStateResolver) Validate(ctx context.Context, actorID uuid.UUID, role string) (bool, error) {
	f.callCount++
	return f.active, nil
}

func (f *fakeVersionedUserStateResolver) ValidateVersioned(ctx context.Context, actorID uuid.UUID, role string) (bool, string, error) {
	f.callCount++
	return f.active, f.version, nil
}

func (f *fakeVersionedUserStateResolver) ValidateVersion(ctx context.Context, actorID uuid.UUID) (string, error) {
	f.versionCallCount++
	return f.version, nil
}

func TestCachedUserStateResolver_VersionedSourceDoesNotReloadStateOnCacheHit(t *testing.T) {
	resolver := &fakeVersionedUserStateResolver{active: true, version: "v1"}
	cached := NewCachedUserStateResolver(resolver, time.Minute)
	actorID := uuid.New()

	active, err := cached.Validate(context.Background(), actorID, "staff")
	require.NoError(t, err)
	assert.True(t, active)
	active, err = cached.Validate(context.Background(), actorID, "staff")
	require.NoError(t, err)
	assert.True(t, active)

	assert.Equal(t, 1, resolver.callCount, "versioned source 在版本未變更時不應重新載入完整狀態")
	assert.Equal(t, 1, resolver.versionCallCount, "cache hit 只應查詢輕量版本")
}

func TestCachedUserStateResolver_VersionChangeReloadsState(t *testing.T) {
	resolver := &fakeVersionedUserStateResolver{active: true, version: "v1"}
	cached := NewCachedUserStateResolver(resolver, time.Minute)
	actorID := uuid.New()

	active, err := cached.Validate(context.Background(), actorID, "staff")
	require.NoError(t, err)
	require.True(t, active)
	resolver.version = "v2"
	resolver.active = false

	active, err = cached.Validate(context.Background(), actorID, "staff")
	require.NoError(t, err)
	assert.False(t, active)
	assert.Equal(t, 2, resolver.callCount, "版本變更後應重新驗證使用者狀態")
}

func TestCachedUserStateResolver_RefetchesVersionedSourceAfterTTL(t *testing.T) {
	resolver := &fakeVersionedUserStateResolver{active: true, version: "v1"}
	cached := NewCachedUserStateResolver(resolver, time.Minute)
	actorID := uuid.New()
	cached.cache[actorID] = userStateCacheEntry{
		active:  true,
		role:    "staff",
		version: "v1",
		expires: time.Now().Add(-time.Second),
	}

	_, err := cached.Validate(context.Background(), actorID, "staff")
	require.NoError(t, err)
	assert.Equal(t, 1, resolver.callCount)
}
