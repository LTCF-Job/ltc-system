package transport

import (
	"github.com/gin-gonic/gin"
	"ltc-system/apps/api/internal/modules/demo/app"
)

// ConcurrencyGuardMiddleware 讓一般 API 請求與 Demo 重置互斥。
func ConcurrencyGuardMiddleware(guard *app.ConcurrencyGuard) gin.HandlerFunc {
	return func(c *gin.Context) {
		release := guard.BeginRequest()
		defer release()
		c.Next()
	}
}
