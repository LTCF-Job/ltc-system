package auth

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/MicahParks/keyfunc/v3"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"ltc-system/apps/api/internal/platform/config"
	"ltc-system/apps/api/internal/platform/httpx"
)

const (
	ContextKeyActorID   = "actor_id"
	ContextKeyActorRole = "actor_role"
	ContextKeyUserEmail = "user_email"
	ContextKeyActorName = "actor_name"
	ContextKeyDataPlane = "actor_data_plane"
	DataPlaneProduction = "production"
	DataPlaneDemo       = "demo"
)

// newSupabaseJWKS 建立向 Supabase JWKS 端點取金鑰並自動輪替的 Keyfunc；未設定 URL 時回傳 nil。
func newSupabaseJWKS(jwksURL string) (keyfunc.Keyfunc, error) {
	if jwksURL == "" {
		return nil, nil
	}
	// ctx 的存續期間同時控制背景自動刷新 goroutine，不可在此提前取消，需與 process 生命週期一致。
	return keyfunc.NewDefaultCtx(context.Background(), []string{jwksURL})
}

// setActorFromClaims 將 JWT claims 中的 sub 與角色資訊注入 Gin Context。
func setActorFromClaims(c *gin.Context, claims jwt.MapClaims) {
	sub, _ := claims.GetSubject()
	actorID, err := uuid.Parse(sub)
	if err != nil {
		actorID = uuid.Nil
	}

	// role 一律以 app_metadata 為準：這是使用者無法自行編輯的伺服器端欄位，符合 Supabase RBAC 慣例，
	// 也與 migrations/000002_seed_reference_data.up.sql 寫入 raw_app_meta_data.role 的方式一致；
	// user_metadata 僅補齊 app_metadata 未提供時的角色與顯示名稱。
	role := "viewer"
	name := ""
	if r, ok := claims["role"].(string); ok {
		role = r
	}
	if userMetadata, ok := claims["user_metadata"].(map[string]interface{}); ok {
		if r, ok := userMetadata["role"].(string); ok {
			role = r
		}
		if n, ok := userMetadata["display_name"].(string); ok {
			name = n
		}
	}
	// data_plane 只信任 app_metadata，理由與上方 role 相同：user_metadata 使用者可自行竄改。
	dataPlane := DataPlaneProduction
	if appMetadata, ok := claims["app_metadata"].(map[string]interface{}); ok {
		if r, ok := appMetadata["role"].(string); ok {
			role = r
		}
		if n, ok := appMetadata["display_name"].(string); ok {
			name = n
		}
		if dp, ok := appMetadata["data_plane"].(string); ok {
			dataPlane = dp
		}
	}
	if name == "" {
		if email, ok := claims["email"].(string); ok {
			name = email
		}
	}

	c.Set(ContextKeyActorID, actorID)
	c.Set(ContextKeyActorRole, role)
	c.Set(ContextKeyActorName, name)
	c.Set(ContextKeyDataPlane, dataPlane)
}

// Middleware 驗證傳入的 Supabase JWT Token 簽章並將使用者角色與 ID 注入 Gin Context。
func Middleware(cfg *config.Config) gin.HandlerFunc {
	jwks, err := newSupabaseJWKS(cfg.SupabaseJWKSURL)
	if err != nil {
		// 正式環境缺少可用 JWKS 時無法驗證任何憑證，直接 fail fast 避免帶著漏洞啟動
		panic(fmt.Sprintf("無法初始化 Supabase JWKS (%s): %v", cfg.SupabaseJWKSURL, err))
	}

	return func(c *gin.Context) {
		// 開發模式支援 Mock Header 方便本機測試與端點驗收
		if cfg.AppEnv == "local" {
			if mockRole := c.GetHeader("X-Mock-Role"); mockRole != "" {
				actorIDStr := c.GetHeader("X-Mock-User-ID")
				actorID, err := uuid.Parse(actorIDStr)
				if err != nil {
					actorID = uuid.New()
				}
				c.Set(ContextKeyActorID, actorID)
				c.Set(ContextKeyActorRole, mockRole)
				c.Set(ContextKeyUserEmail, "dev@example.com")
				c.Set(ContextKeyActorName, "dev@example.com")
				c.Next()
				return
			}
		}

		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			httpx.RespondError(c, http.StatusUnauthorized, httpx.CodeUnauthenticated, "未提供認證憑證", nil)
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			httpx.RespondError(c, http.StatusUnauthorized, httpx.CodeUnauthenticated, "憑證格式錯誤，必須為 Bearer Token", nil)
			return
		}

		tokenStr := parts[1]

		// 本機開發支援 mock_jwt_ 形式之憑證快速解析
		if cfg.AppEnv == "local" && strings.HasPrefix(tokenStr, "mock_jwt_") {
			role := "staff"
			if strings.Contains(tokenStr, "admin") {
				role = "admin"
			} else if strings.Contains(tokenStr, "dispatcher") {
				role = "dispatcher"
			} else if strings.Contains(tokenStr, "driver") {
				role = "driver"
			} else if strings.Contains(tokenStr, "viewer") {
				role = "viewer"
			}
			c.Set(ContextKeyActorID, uuid.MustParse("00000000-0000-0000-0000-000000000001"))
			c.Set(ContextKeyActorRole, role)
			c.Set(ContextKeyUserEmail, role+"@example.com")
			c.Set(ContextKeyActorName, role+"@example.com")
			c.Next()
			return
		}

		// 本機且未設定 JWKS 時，安全降級為未驗證解析，僅限本機開發使用
		if cfg.AppEnv == "local" && jwks == nil {
			claims := jwt.MapClaims{}
			token, _, err := new(jwt.Parser).ParseUnverified(tokenStr, claims)
			if err != nil || token == nil {
				c.Set(ContextKeyActorID, uuid.MustParse("00000000-0000-0000-0000-000000000001"))
				c.Set(ContextKeyActorRole, "admin")
				c.Set(ContextKeyUserEmail, "admin@example.com")
				c.Set(ContextKeyActorName, "admin@example.com")
				c.Next()
				return
			}
			setActorFromClaims(c, claims)
			if !enforceDataPlane(c, cfg) {
				return
			}
			c.Next()
			return
		}

		if jwks == nil {
			httpx.RespondError(c, http.StatusInternalServerError, httpx.CodeInternalError, "伺服器未設定 JWKS，無法驗證身分", nil)
			return
		}

		claims := jwt.MapClaims{}
		token, err := jwt.ParseWithClaims(tokenStr, claims, jwks.Keyfunc,
			jwt.WithValidMethods([]string{"RS256", "ES256"}),
			jwt.WithExpirationRequired())
		if err != nil || !token.Valid {
			httpx.RespondError(c, http.StatusUnauthorized, httpx.CodeUnauthenticated, "無效的 JWT Token", nil)
			return
		}

		setActorFromClaims(c, claims)
		if !enforceDataPlane(c, cfg) {
			return
		}
		c.Next()
	}
}

// enforceDataPlane 驗證 JWT 的 data_plane 是否符合目前服務環境，不符則回應 401 並中止請求。
func enforceDataPlane(c *gin.Context, cfg *config.Config) bool {
	servicePlane := cfg.DataPlane
	if servicePlane == "" {
		servicePlane = DataPlaneProduction
	}
	tokenPlane, _ := c.Get(ContextKeyDataPlane)
	if tokenPlane != servicePlane {
		httpx.RespondError(c, http.StatusUnauthorized, httpx.CodeUnauthenticated, "此憑證不適用於目前環境", nil)
		return false
	}
	return true
}

// RequireRoles 依據角色權限矩陣驗證當前請求是否具有執行權限。
func RequireRoles(allowedRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		roleVal, exists := c.Get(ContextKeyActorRole)
		if !exists {
			httpx.RespondError(c, http.StatusUnauthorized, httpx.CodeUnauthenticated, "未登入或無法識別角色", nil)
			return
		}

		currentRole := roleVal.(string)
		for _, allowed := range allowedRoles {
			if currentRole == allowed {
				c.Next()
				return
			}
		}

		httpx.RespondError(c, http.StatusForbidden, httpx.CodeForbidden, "權限不足，拒絕存取", nil)
	}
}

// GetActorID 從 Context 安全取出當前使用者 UUID。
func GetActorID(c *gin.Context) uuid.UUID {
	if val, exists := c.Get(ContextKeyActorID); exists {
		if id, ok := val.(uuid.UUID); ok {
			return id
		}
	}
	return uuid.Nil
}

// GetActorRole 從 Context 安全取出當前使用者角色字串。
func GetActorRole(c *gin.Context) string {
	if val, exists := c.Get(ContextKeyActorRole); exists {
		if r, ok := val.(string); ok {
			return r
		}
	}
	return "viewer"
}

// GetActorName 從 Context 取出當前使用者的顯示名稱（來源為 JWT user_metadata.display_name，
// 缺漏時退回 email）；兩者皆無時回傳空字串，由呼叫端決定顯示預設值。
func GetActorName(c *gin.Context) string {
	if val, exists := c.Get(ContextKeyActorName); exists {
		if n, ok := val.(string); ok {
			return n
		}
	}
	return ""
}

// GetActorDataPlane 從 Context 安全取出當前憑證所屬的 data plane。
func GetActorDataPlane(c *gin.Context) string {
	if val, exists := c.Get(ContextKeyDataPlane); exists {
		if dp, ok := val.(string); ok {
			return dp
		}
	}
	return DataPlaneProduction
}
