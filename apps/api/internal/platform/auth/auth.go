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

// setActorFromClaims 將 JWT claims 中的 sub、角色與 data plane 注入 Gin Context；
// sub 不是合法 UUID 時回應 401 並回傳 false。
func setActorFromClaims(c *gin.Context, claims jwt.MapClaims) bool {
	sub, _ := claims.GetSubject()
	actorID, err := uuid.Parse(sub)
	if err != nil {
		// uuid.Nil 會流進稽核紀錄與「不可刪除自己」等以 actor 為準的判斷，靜默放行等同錯誤授權。
		httpx.RespondError(c, http.StatusUnauthorized, httpx.CodeUnauthenticated, "憑證未包含可識別的使用者 ID", nil)
		return false
	}

	email, _ := claims["email"].(string)
	name := ""
	if userMetadata, ok := claims["user_metadata"].(map[string]interface{}); ok {
		if n, ok := userMetadata["display_name"].(string); ok {
			name = n
		}
	}

	// role 與 data_plane 只認 app_metadata：它唯一的寫入路徑是持 service role key 的 Admin API
	// （見 identity/infra/supabase_admin_client.go），使用者無法自行竄改；user_metadata 則可由使用者
	// 呼叫 supabase.auth.updateUser 寫入，頂層 role claim 又只是 Postgres role（authenticated／
	// anon／service_role），兩者都不能當業務角色。
	// 這裡不做角色白名單：管理員可從「角色身分管理」自建角色，合法 key 是動態的（見
	// identity/app/role_service.go 的 Create），寫死清單會讓自訂角色使用者被靜默降級；未知 role
	// 會在 RequirePermission 查不到權限矩陣而以 403 擋下。
	role := "viewer"
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
		name = email
	}

	c.Set(ContextKeyActorID, actorID)
	c.Set(ContextKeyActorRole, role)
	c.Set(ContextKeyActorName, name)
	c.Set(ContextKeyUserEmail, email)
	c.Set(ContextKeyDataPlane, dataPlane)
	return true
}

// Middleware 驗證傳入的 Supabase JWT Token 簽章並將使用者角色與 ID 注入 Gin Context。
func Middleware(cfg *config.Config) gin.HandlerFunc {
	jwks, err := newSupabaseJWKS(cfg.SupabaseJWKSURL)
	if err != nil {
		// 正式環境缺少可用 JWKS 時無法驗證任何憑證，直接 fail fast 避免帶著漏洞啟動
		panic(fmt.Sprintf("無法初始化 Supabase JWKS (%s): %v", cfg.SupabaseJWKSURL, err))
	}

	return func(c *gin.Context) {
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
			c.Set(ContextKeyDataPlane, DataPlaneProduction)
			if !enforceDataPlane(c, cfg) {
				return
			}
			c.Next()
			return
		}

		// 本機且未設定 JWKS 時，安全降級為未驗證解析，僅限本機開發使用
		if cfg.AppEnv == "local" && jwks == nil {
			claims := jwt.MapClaims{}
			token, _, err := new(jwt.Parser).ParseUnverified(tokenStr, claims)
			if err != nil || token == nil {
				httpx.RespondError(c, http.StatusUnauthorized, httpx.CodeUnauthenticated, "無效的 JWT Token", nil)
				return
			}
			if !setActorFromClaims(c, claims) {
				return
			}
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
			jwt.WithExpirationRequired(),
			// 綁定簽發者與受眾，避免其他 Supabase 專案或非使用者流程（如 service_role）的 token 被接受。
			jwt.WithIssuer(cfg.SupabaseJWTIssuer),
			jwt.WithAudience("authenticated"))
		if err != nil || !token.Valid {
			httpx.RespondError(c, http.StatusUnauthorized, httpx.CodeUnauthenticated, "無效的 JWT Token", nil)
			return
		}

		if !setActorFromClaims(c, claims) {
			return
		}
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

// GetActorEmail 從 Context 取出當前使用者的登入信箱。
func GetActorEmail(c *gin.Context) string {
	if val, exists := c.Get(ContextKeyUserEmail); exists {
		if e, ok := val.(string); ok {
			return e
		}
	}
	return ""
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
