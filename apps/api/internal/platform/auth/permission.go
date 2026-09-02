package auth

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"ltc-system/apps/api/internal/platform/httpx"
)

// ModulePermission 是 identity 模組角色權限矩陣在 platform 層的對應形狀；platform 套件不可
// 匯入業務模組（見 backend-architecture skill 的分層規則），故在此重複定義而非直接引用。
type ModulePermission struct {
	View   bool
	Edit   bool
	Delete bool
}

// PermissionResolver 依角色 key 解析其模組權限矩陣；角色不存在時回傳 (nil, nil)，
// 不視為錯誤——RequirePermission 會把「查無此模組權限」與「查無此角色」一律當作拒絕存取。
type PermissionResolver interface {
	Resolve(ctx context.Context, roleKey string) (map[string]ModulePermission, error)
}

// permissionCacheTTL 讓「角色身分管理」頁改權限後，API 授權在這個時間內就會反映新設定，
// 不需要使用者重新登入換發 JWT；同時避免每個受保護請求都直接查一次 roles 表。
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

// RequirePermission 依角色的模組權限矩陣驗證請求是否具備指定模組的 view／edit／delete 權限；
// 與 RequireRoles 的路由層級粗粒度白名單不同，這裡的授權結果會隨「角色身分管理」頁的設定變動，
// 讓自訂角色也能在 API 層拿到與其權限矩陣一致的存取範圍。
func RequirePermission(resolver PermissionResolver, module, action string) gin.HandlerFunc {
	return func(c *gin.Context) {
		roleVal, exists := c.Get(ContextKeyActorRole)
		if !exists {
			httpx.RespondError(c, http.StatusUnauthorized, httpx.CodeUnauthenticated, "未登入或無法識別角色", nil)
			return
		}
		roleKey, _ := roleVal.(string)

		perms, err := resolver.Resolve(c.Request.Context(), roleKey)
		if err != nil {
			httpx.RespondError(c, http.StatusInternalServerError, httpx.CodeInternalError, "無法解析角色權限", nil)
			return
		}

		if !hasAction(perms[module], action) {
			httpx.RespondError(c, http.StatusForbidden, httpx.CodeForbidden, "權限不足，拒絕存取", nil)
			return
		}
		c.Next()
	}
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
