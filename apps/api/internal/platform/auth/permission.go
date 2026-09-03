package auth

import (
	"context"
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

// CustomPermissionResolver 依使用者 ID 解析其個人層級的模組權限覆蓋；沒有設定覆蓋時
// 回傳 (nil, nil)，RequirePermission 會視為「維持角色矩陣原值」而非拒絕存取。
type CustomPermissionResolver interface {
	Resolve(ctx context.Context, actorID uuid.UUID) (map[string]ModulePermission, error)
}

// permissionCacheTTL 讓「角色身分管理」頁改權限後，API 授權在這個時間內就會反映新設定，
// 不需要使用者重新登入換發 JWT；同時避免每個受保護請求都直接查一次 roles 表。個人層級的
// customPermissions 也採同一 TTL，取捨見 docs/decisions/custom-permission-admin-api-enforcement.md。
const permissionCacheTTL = 30 * time.Second

type permissionCacheEntry struct {
	perms   map[string]ModulePermission
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

// Resolve 命中未過期快取就直接回傳，否則回源查詢並刷新快取。
func (c *CachedPermissionResolver) Resolve(ctx context.Context, roleKey string) (map[string]ModulePermission, error) {
	c.mu.RLock()
	entry, ok := c.cache[roleKey]
	c.mu.RUnlock()
	if ok && time.Now().Before(entry.expires) {
		return entry.perms, nil
	}

	perms, err := c.source.Resolve(ctx, roleKey)
	if err != nil {
		return nil, err
	}
	c.mu.Lock()
	c.cache[roleKey] = permissionCacheEntry{perms: perms, expires: time.Now().Add(permissionCacheTTL)}
	c.mu.Unlock()
	return perms, nil
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

// Resolve 命中未過期快取就直接回傳，否則回源查詢並刷新快取。
func (c *CachedCustomPermissionResolver) Resolve(ctx context.Context, actorID uuid.UUID) (map[string]ModulePermission, error) {
	c.mu.RLock()
	entry, ok := c.cache[actorID]
	c.mu.RUnlock()
	if ok && time.Now().Before(entry.expires) {
		return entry.perms, nil
	}

	perms, err := c.source.Resolve(ctx, actorID)
	if err != nil {
		return nil, err
	}
	c.mu.Lock()
	c.cache[actorID] = permissionCacheEntry{perms: perms, expires: time.Now().Add(permissionCacheTTL)}
	c.mu.Unlock()
	return perms, nil
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
