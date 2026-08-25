package middleware

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"ltc-system/apps/api/internal/config"
)

const (
	ContextKeyActorID   = "actor_id"
	ContextKeyActorRole = "actor_role"
	ContextKeyUserEmail = "user_email"
)

var (
	ErrInvalidToken = errors.New("invalid ingest token")
)

// AuthMiddleware 驗證傳入的 Supabase JWT Token 並將使用者角色與 ID 注入 Gin Context。
func AuthMiddleware(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 開發模式支援 Mock Header 方便本地測試與端點驗收
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
				c.Next()
				return
			}
		}

		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			RespondError(c, http.StatusUnauthorized, CodeUnauthenticated, "未提供認證憑證", nil)
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			RespondError(c, http.StatusUnauthorized, CodeUnauthenticated, "憑證格式錯誤，必須為 Bearer Token", nil)
			return
		}

		tokenStr := parts[1]

		// 解析 Token Claims（支援本地未配置 JWKS 實體連線時之安全降級）
		claims := jwt.MapClaims{}
		token, _, err := new(jwt.Parser).ParseUnverified(tokenStr, claims)
		if err != nil || token == nil {
			RespondError(c, http.StatusUnauthorized, CodeUnauthenticated, "無效的 JWT Token", nil)
			return
		}

		// 取得 sub 與 role
		sub, _ := claims.GetSubject()
		actorID, err := uuid.Parse(sub)
		if err != nil {
			actorID = uuid.Nil
		}

		role := "viewer"
		if userMetadata, ok := claims["user_metadata"].(map[string]interface{}); ok {
			if r, ok := userMetadata["role"].(string); ok {
				role = r
			}
		} else if appMetadata, ok := claims["app_metadata"].(map[string]interface{}); ok {
			if r, ok := appMetadata["role"].(string); ok {
				role = r
			}
		} else if r, ok := claims["role"].(string); ok {
			role = r
		}

		c.Set(ContextKeyActorID, actorID)
		c.Set(ContextKeyActorRole, role)
		c.Next()
	}
}

// RequireRoles 依據角色權限矩陣驗證當前請求是否具有執行權限。
func RequireRoles(allowedRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		roleVal, exists := c.Get(ContextKeyActorRole)
		if !exists {
			RespondError(c, http.StatusUnauthorized, CodeUnauthenticated, "未登入或無法識別角色", nil)
			return
		}

		currentRole := roleVal.(string)
		for _, allowed := range allowedRoles {
			if currentRole == allowed {
				c.Next()
				return
			}
		}

		RespondError(c, http.StatusForbidden, CodeForbidden, "權限不足，拒絕訪問", nil)
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
