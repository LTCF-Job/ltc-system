package auth

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/MicahParks/keyfunc/v3"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"ltc-system/apps/api/internal/platform/config"
	"ltc-system/apps/api/internal/platform/httpx"
	"ltc-system/apps/api/internal/platform/requestmeta"
)

const (
	ContextKeyActorID   = "actor_id"
	ContextKeyActorRole = "actor_role"
	ContextKeyUserEmail = "user_email"
	ContextKeyActorName = "actor_name"
)

// UserState 是 UserStateResolver 查詢後端使用者狀態的結果。分成三種而非單純的
// active／inactive 布林值，是為了讓中介層能區分「帳號真的被停用」與「JWT 裡的角色
// 宣告已經跟資料庫不一致（例如管理員剛改過角色，使用者還沒重新登入）」——後者不是帳號
// 出了問題，只是需要重新登入換發新 JWT，兩者顯示給使用者的訊息不應該一樣。
type UserState int

const (
	// UserStateActive 代表帳號啟用中且 JWT 角色與資料庫一致，可放行請求。
	UserStateActive UserState = iota
	// UserStateDisabled 代表帳號已被停用或查無此帳號。
	UserStateDisabled
	// UserStateRoleMismatch 代表帳號本身仍是啟用狀態，但 JWT 內的角色宣告與資料庫
	// 目前的角色不同，需要重新登入才能取得新角色。
	UserStateRoleMismatch
)

// UserStateResolver 讓高風險 API 在 JWT 尚未過期時仍能即時拒絕已停用帳號，或要求
// 角色已變更的使用者重新登入。
type UserStateResolver interface {
	Validate(ctx context.Context, actorID uuid.UUID, role string) (UserState, error)
}

// VersionedUserStateResolver 以共享資料來源版本標記回源取得的帳號狀態；未過期的
// process-local 項目會搭配 UserStateVersionResolver 先做輕量版本比對。
type VersionedUserStateResolver interface {
	ValidateVersioned(ctx context.Context, actorID uuid.UUID, role string) (UserState, string, error)
}

// UserStateVersionResolver 只查詢帳號安全狀態的共享版本，避免 cache hit 時重新載入完整投影。
type UserStateVersionResolver interface {
	ValidateVersion(ctx context.Context, actorID uuid.UUID) (string, error)
}

// newSupabaseJWKS 建立向 Supabase JWKS 端點取金鑰並自動輪替的 Keyfunc；未設定 URL 時回傳 nil。
func newSupabaseJWKS(jwksURL string) (keyfunc.Keyfunc, error) {
	if jwksURL == "" {
		return nil, nil
	}
	// ctx 的存續期間同時控制背景自動刷新 goroutine，不可在此提前取消，需與 process 生命週期一致。
	return keyfunc.NewDefaultCtx(context.Background(), []string{jwksURL})
}

// setActorFromClaims 將 JWT claims 中的 sub 與角色注入 Gin Context；
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

	// role 只認 app_metadata：它唯一的寫入路徑是持 service role key 的 Admin API
	// （見 identity/infra/supabase_admin_client.go），使用者無法自行竄改；user_metadata 則可由使用者
	// 呼叫 supabase.auth.updateUser 寫入，頂層 role claim 又只是 Postgres role（authenticated／
	// anon／service_role），兩者都不能當業務角色。
	// 這裡不做角色白名單：管理員可從「角色身分管理」自建角色，合法 key 是動態的（見
	// identity/app/role_service.go 的 Create），寫死清單會讓自訂角色使用者被靜默降級；未知 role
	// 會在 RequirePermission 查不到權限矩陣而以 403 擋下。
	role := "viewer"
	if appMetadata, ok := claims["app_metadata"].(map[string]interface{}); ok {
		if r, ok := appMetadata["role"].(string); ok {
			role = r
		}
		if n, ok := appMetadata["display_name"].(string); ok {
			name = n
		}
	}
	if name == "" {
		name = email
	}

	c.Set(ContextKeyActorID, actorID)
	c.Set(ContextKeyActorRole, role)
	c.Set(ContextKeyActorName, name)
	c.Set(ContextKeyUserEmail, email)
	return true
}

// Middleware 驗證傳入的 Supabase JWT Token 簽章並將使用者角色與 ID 注入 Gin Context。
func Middleware(cfg *config.Config) gin.HandlerFunc {
	return MiddlewareWithUserState(cfg, nil)
}

// MiddlewareWithUserState 驗證 JWT，並可選擇查詢目前使用者狀態。
func MiddlewareWithUserState(cfg *config.Config, userState UserStateResolver) gin.HandlerFunc {
	jwks, err := newSupabaseJWKS(cfg.SupabaseJWKSURL)
	if err != nil {
		// 正式環境缺少可用 JWKS 時無法驗證任何憑證，直接 fail fast 避免帶著漏洞啟動
		panic(fmt.Sprintf("無法初始化 Supabase JWKS (%s): %v", cfg.SupabaseJWKSURL, err))
	}

	return func(c *gin.Context) {
		requestCtx := requestmeta.With(c.Request.Context(), c.ClientIP(), c.Request.UserAgent())
		requestCtx, _ = WithRequestSecurityState(requestCtx)
		c.Request = c.Request.WithContext(requestCtx)
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
		if cfg.AppEnv == "local" && cfg.AllowInsecureMockAuth && strings.HasPrefix(tokenStr, "mock_jwt_") {
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
		if cfg.AppEnv == "local" && cfg.AllowInsecureMockAuth && jwks == nil {
			claims := jwt.MapClaims{}
			token, _, err := new(jwt.Parser).ParseUnverified(tokenStr, claims)
			if err != nil || token == nil {
				httpx.RespondError(c, http.StatusUnauthorized, httpx.CodeUnauthenticated, "無效的 JWT Token", nil)
				return
			}
			if !setActorFromClaims(c, claims) {
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
		if userState != nil {
			state, err := userState.Validate(c.Request.Context(), GetActorID(c), GetActorRole(c))
			if err != nil {
				// 這裡是所有已驗證請求的必經路徑，錯誤不記錄就只剩一個沒有成因的 503；
				// 訊息只進伺服器日誌，回應本身仍維持非技術性字串。
				slog.Error("user_state_validation_failed",
					slog.String("request_id", httpx.RequestID(c)),
					slog.String("path", c.Request.URL.Path),
					slog.String("actor_id", GetActorID(c).String()),
					slog.String("error_type", fmt.Sprintf("%T", err)),
					slog.String("error_message", err.Error()),
				)
				httpx.RespondError(c, http.StatusServiceUnavailable, httpx.CodeServiceUnavailable, "無法確認使用者狀態", nil)
				return
			}
			switch state {
			case UserStateDisabled:
				httpx.RespondError(c, http.StatusUnauthorized, httpx.CodeUnauthenticated, "使用者帳號已停用", nil)
				return
			case UserStateRoleMismatch:
				// 帳號本身沒問題，只是角色已被管理員變更、JWT 裡的舊角色宣告過期，
				// 不應該說成「帳號已停用」誤導使用者；狀態碼仍是 401，前端既有的
				// 401 分支會導向重新登入，效果等同要求換發新 JWT。
				httpx.RespondError(c, http.StatusUnauthorized, httpx.CodeUnauthenticated, "您的權限已更新，請重新登入", nil)
				return
			}
		}
		c.Next()
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
