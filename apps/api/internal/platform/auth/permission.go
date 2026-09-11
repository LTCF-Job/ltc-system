package auth

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"ltc-system/apps/api/internal/platform/httpx"
)

// ModulePermission 是 identity 模組角色權限矩陣在 platform 層的對應形狀；platform 套件不可
// 匯入業務模組（見 backend-architecture skill 的分層規則），故在此重複定義而非直接引用。
type ModulePermission struct {
	View   bool `json:"view"`
	Edit   bool `json:"edit"`
	Delete bool `json:"delete"`
}

// PermissionResolver 依角色 key 解析其模組權限矩陣；角色不存在時回傳 (nil, nil)，
// 不視為錯誤——RequirePermission 會把「查無此模組權限」與「查無此角色」一律當作拒絕存取。
type PermissionResolver interface {
	Resolve(ctx context.Context, roleKey string) (map[string]ModulePermission, error)
}

// VersionedPermissionResolver 在需要完整回源時回傳權限矩陣與共享版本；版本通常來自
// roles.updated_at 或其他單調遞增的共享版本欄位。
type VersionedPermissionResolver interface {
	ResolveVersioned(ctx context.Context, roleKey string) (map[string]ModulePermission, string, error)
}

// PermissionVersionResolver 提供比完整權限矩陣更輕量的版本查詢，讓 cache hit 仍能確認
// 跨 replica 的權限是否變更，而不必每次重新載入 JSON 權限資料。
type PermissionVersionResolver interface {
	ResolveVersion(ctx context.Context, roleKey string) (string, error)
}

// CustomPermissionResolver 依使用者 ID 解析其個人層級的模組權限覆蓋；沒有設定覆蓋時
// 回傳 (nil, nil)，RequirePermission 會視為「維持角色矩陣原值」而非拒絕存取。
type CustomPermissionResolver interface {
	Resolve(ctx context.Context, actorID uuid.UUID) (map[string]ModulePermission, error)
}

// VersionedCustomPermissionResolver 在需要完整回源時回傳個人權限覆蓋與共享版本。
type VersionedCustomPermissionResolver interface {
	ResolveVersioned(ctx context.Context, actorID uuid.UUID) (map[string]ModulePermission, string, error)
}

// CustomPermissionVersionResolver 提供個人權限覆蓋的輕量版本查詢，避免 cache hit 時重新
// 載入完整的使用者安全狀態投影。
type CustomPermissionVersionResolver interface {
	ResolveVersion(ctx context.Context, actorID uuid.UUID) (string, error)
}

// permissionCacheTTL 是沒有輕量版本來源時的 fallback 上限；正式的 role／custom permission
// source 都提供版本查詢，cache hit 會先比對版本，再決定是否重載完整權限矩陣。
const permissionCacheTTL = 30 * time.Second
const permissionCacheMaxEntries = 1024

type permissionCacheEntry struct {
	perms   map[string]ModulePermission
	version string
	expires time.Time
}

// CachedPermissionResolver 包一層短 TTL 的行程內快取在 PermissionResolver 外面。
type CachedPermissionResolver struct {
	source PermissionResolver
	mu     sync.RWMutex
	cache  map[string]permissionCacheEntry
}

// NewCachedPermissionResolver 建立 CachedPermissionResolver 實例。
func NewCachedPermissionResolver(source PermissionResolver) *CachedPermissionResolver {
	return &CachedPermissionResolver{source: source, cache: make(map[string]permissionCacheEntry)}
}

// Resolve 命中未過期快取時先查輕量版本；版本相同直接回傳，版本變更或 cache miss 才完整回源。
func (c *CachedPermissionResolver) Resolve(ctx context.Context, roleKey string) (map[string]ModulePermission, error) {
	now := time.Now()
	c.mu.RLock()
	entry, cached := c.cache[roleKey]
	c.mu.RUnlock()
	if cached && now.Before(entry.expires) {
		if versionSource, ok := c.source.(PermissionVersionResolver); ok {
			version, err := versionSource.ResolveVersion(ctx, roleKey)
			if err != nil {
				return nil, err
			}
			if version == entry.version {
				return entry.perms, nil
			}
		} else {
			return entry.perms, nil
		}
	}

	if source, ok := c.source.(VersionedPermissionResolver); ok {
		perms, version, err := source.ResolveVersioned(ctx, roleKey)
		if err != nil {
			return nil, err
		}
		c.store(roleKey, perms, version)
		return perms, nil
	}

	perms, err := c.source.Resolve(ctx, roleKey)
	if err != nil {
		return nil, err
	}
	c.store(roleKey, perms, "")
	return perms, nil
}

func (c *CachedPermissionResolver) store(roleKey string, perms map[string]ModulePermission, version string) {
	now := time.Now()
	c.mu.Lock()
	c.pruneExpiredLocked(now)
	c.cache[roleKey] = permissionCacheEntry{perms: perms, version: version, expires: now.Add(permissionCacheTTL)}
	if len(c.cache) > permissionCacheMaxEntries {
		c.evictOneLocked()
	}
	c.mu.Unlock()
}

// InvalidateRole 讓角色或角色權限異動立即失效，不等待 TTL。
func (c *CachedPermissionResolver) InvalidateRole(roleKey string) {
	c.mu.Lock()
	delete(c.cache, roleKey)
	c.mu.Unlock()
}

func (c *CachedPermissionResolver) pruneExpiredLocked(now time.Time) {
	for key, entry := range c.cache {
		if !now.Before(entry.expires) {
			delete(c.cache, key)
		}
	}
}

func (c *CachedPermissionResolver) evictOneLocked() {
	for key := range c.cache {
		delete(c.cache, key)
		return
	}
}

// CachedCustomPermissionResolver 比照 CachedPermissionResolver，對 CustomPermissionResolver
// 包一層相同 TTL 的行程內快取，key 改用 actorID，讓兩種權限來源有一致的更新延遲與快取行為。
type CachedCustomPermissionResolver struct {
	source CustomPermissionResolver
	mu     sync.RWMutex
	cache  map[uuid.UUID]permissionCacheEntry
}

// NewCachedCustomPermissionResolver 建立 CachedCustomPermissionResolver 實例。
func NewCachedCustomPermissionResolver(source CustomPermissionResolver) *CachedCustomPermissionResolver {
	return &CachedCustomPermissionResolver{source: source, cache: make(map[uuid.UUID]permissionCacheEntry)}
}

// Resolve 命中未過期快取時先查輕量版本；版本相同直接回傳，版本變更或 cache miss 才完整回源。
func (c *CachedCustomPermissionResolver) Resolve(ctx context.Context, actorID uuid.UUID) (map[string]ModulePermission, error) {
	now := time.Now()
	c.mu.RLock()
	entry, cached := c.cache[actorID]
	c.mu.RUnlock()
	if cached && now.Before(entry.expires) {
		if versionSource, ok := c.source.(CustomPermissionVersionResolver); ok {
			version, err := versionSource.ResolveVersion(ctx, actorID)
			if err != nil {
				return nil, err
			}
			if version == entry.version {
				return entry.perms, nil
			}
		} else {
			return entry.perms, nil
		}
	}

	if source, ok := c.source.(VersionedCustomPermissionResolver); ok {
		perms, version, err := source.ResolveVersioned(ctx, actorID)
		if err != nil {
			return nil, err
		}
		c.store(actorID, perms, version)
		return perms, nil
	}

	perms, err := c.source.Resolve(ctx, actorID)
	if err != nil {
		return nil, err
	}
	c.mu.Lock()
	c.pruneExpiredLocked(time.Now())
	c.cache[actorID] = permissionCacheEntry{perms: perms, version: "", expires: time.Now().Add(permissionCacheTTL)}
	if len(c.cache) > permissionCacheMaxEntries {
		c.evictOneLocked()
	}
	c.mu.Unlock()
	return perms, nil
}

func (c *CachedCustomPermissionResolver) store(actorID uuid.UUID, perms map[string]ModulePermission, version string) {
	now := time.Now()
	c.mu.Lock()
	c.pruneExpiredLocked(now)
	c.cache[actorID] = permissionCacheEntry{perms: perms, version: version, expires: now.Add(permissionCacheTTL)}
	if len(c.cache) > permissionCacheMaxEntries {
		c.evictOneLocked()
	}
	c.mu.Unlock()
}

// InvalidateUser 讓個人權限或使用者停用立即失效，不等待 TTL。
func (c *CachedCustomPermissionResolver) InvalidateUser(actorID uuid.UUID) {
	c.mu.Lock()
	delete(c.cache, actorID)
	c.mu.Unlock()
}

func (c *CachedCustomPermissionResolver) pruneExpiredLocked(now time.Time) {
	for key, entry := range c.cache {
		if !now.Before(entry.expires) {
			delete(c.cache, key)
		}
	}
}

func (c *CachedCustomPermissionResolver) evictOneLocked() {
	for key := range c.cache {
		delete(c.cache, key)
		return
	}
}

// RequirePermission 依角色的模組權限矩陣驗證請求是否具備指定模組的 view／edit／delete 權限；
// 授權結果會隨「角色身分管理」頁的設定變動，讓自訂角色也能在 API 層拿到與其權限矩陣一致的
// 存取範圍，不需要在路由上寫死角色字面值。個人層級的 customPermissions 覆蓋透過
// customResolver 疊加在角色矩陣之上，兩者採同一套「查詢＋TTL 快取」機制，取捨見
// docs/decisions/custom-permission-admin-api-enforcement.md。
func RequirePermission(resolver PermissionResolver, customResolver CustomPermissionResolver, module, action string) gin.HandlerFunc {
	return func(c *gin.Context) {
		roleVal, exists := c.Get(ContextKeyActorRole)
		if !exists {
			httpx.RespondError(c, http.StatusUnauthorized, httpx.CodeUnauthenticated, "未登入或無法識別角色", nil)
			return
		}
		roleKey, _ := roleVal.(string)

		effective, err := ResolveEffectivePermissions(c.Request.Context(), resolver, customResolver, roleKey, GetActorID(c))
		if err != nil {
			// 與 MiddlewareWithUserState 同理：這是每支受保護 API 的必經路徑，
			// 不記錄就只剩一個沒有成因的 500。
			slog.Error("permission_resolution_failed",
				slog.String("request_id", httpx.RequestID(c)),
				slog.String("path", c.Request.URL.Path),
				slog.String("module", module),
				slog.String("role_key", roleKey),
				slog.String("error_type", fmt.Sprintf("%T", err)),
				slog.String("error_message", err.Error()),
			)
			httpx.RespondError(c, http.StatusInternalServerError, httpx.CodeInternalError, "無法解析權限", nil)
			return
		}

		if !hasAction(effective[module], action) {
			httpx.RespondError(c, http.StatusForbidden, httpx.CodeForbidden, "權限不足，拒絕存取", nil)
			return
		}
		c.Next()
	}
}

// ResolveEffectivePermissions 解析某個使用者最終生效的模組權限矩陣：先取角色矩陣，再疊上
// 個人層級覆蓋。RequirePermission 與 GET /auth/me 共用這一份邏輯，確保前端拿到的權限與
// API 實際放行的範圍一致。
func ResolveEffectivePermissions(ctx context.Context, resolver PermissionResolver, customResolver CustomPermissionResolver, roleKey string, actorID uuid.UUID) (map[string]ModulePermission, error) {
	perms, err := resolver.Resolve(ctx, roleKey)
	if err != nil {
		return nil, err
	}
	custom, err := customResolver.Resolve(ctx, actorID)
	if err != nil {
		return nil, err
	}
	return mergeCustomPermissions(perms, custom), nil
}

// mergeCustomPermissions 用「整個模組物件覆蓋」語意疊加個人權限，對齊前端
// apps/web/src/stores/auth.ts 的 effectivePermissions：custom 裡有該模組 key 就整包
// 取代角色矩陣的值（未給的欄位視為 false），沒有才維持角色矩陣原值。
func mergeCustomPermissions(rolePerms, custom map[string]ModulePermission) map[string]ModulePermission {
	if len(custom) == 0 {
		return rolePerms
	}
	merged := make(map[string]ModulePermission, len(rolePerms)+len(custom))
	for k, v := range rolePerms {
		merged[k] = v
	}
	for k, v := range custom {
		merged[k] = v
	}
	return merged
}

func hasAction(p ModulePermission, action string) bool {
	switch action {
	case "view":
		return p.View
	case "edit":
		return p.Edit
	case "delete":
		return p.Delete
	default:
		return false
	}
}
