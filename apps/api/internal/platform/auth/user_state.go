package auth

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
)

// CachedUserStateResolver 將已驗證 JWT 的帳號狀態以短 TTL 快取；cache hit 只查共享版本，
// 避免每個 API request 都重新載入完整投影或呼叫外部 Admin API。使用者異動時可透過
// InvalidateUser 立即清除本機項目。
type CachedUserStateResolver struct {
	source UserStateResolver
	ttl    time.Duration
	mu     sync.RWMutex
	cache  map[uuid.UUID]userStateCacheEntry
}

type userStateCacheEntry struct {
	active  bool
	role    string
	version string
	expires time.Time
}

// NewCachedUserStateResolver 建立有界的帳號狀態短快取。
func NewCachedUserStateResolver(source UserStateResolver, ttl time.Duration) *CachedUserStateResolver {
	if ttl <= 0 {
		ttl = 5 * time.Second
	}
	return &CachedUserStateResolver{source: source, ttl: ttl, cache: make(map[uuid.UUID]userStateCacheEntry)}
}

// Validate 命中未過期快取時先查輕量版本；版本相同直接沿用狀態，版本變更或 cache miss
// 才回源查詢並更新快取。
func (c *CachedUserStateResolver) Validate(ctx context.Context, actorID uuid.UUID, role string) (bool, error) {
	now := time.Now()
	c.mu.RLock()
	entry, cached := c.cache[actorID]
	c.mu.RUnlock()
	if cached && entry.role == role && now.Before(entry.expires) {
		if versionSource, ok := c.source.(UserStateVersionResolver); ok {
			version, err := versionSource.ValidateVersion(ctx, actorID)
			if err != nil {
				return false, err
			}
			if version == entry.version {
				return entry.active, nil
			}
		} else {
			return entry.active, nil
		}
	}

	if source, ok := c.source.(VersionedUserStateResolver); ok {
		active, version, err := source.ValidateVersioned(ctx, actorID, role)
		if err != nil {
			return false, err
		}
		c.mu.Lock()
		c.cache[actorID] = userStateCacheEntry{active: active, role: role, version: version, expires: now.Add(c.ttl)}
		c.mu.Unlock()
		return active, nil
	}

	active, err := c.source.Validate(ctx, actorID, role)
	if err != nil {
		return false, err
	}
	c.mu.Lock()
	c.cache[actorID] = userStateCacheEntry{active: active, role: role, expires: now.Add(c.ttl)}
	c.mu.Unlock()
	return active, nil
}

// InvalidateUser 讓帳號停用／角色異動在本機不等待 TTL。
func (c *CachedUserStateResolver) InvalidateUser(actorID uuid.UUID) {
	c.mu.Lock()
	delete(c.cache, actorID)
	c.mu.Unlock()
}
