package transport

import (
	"github.com/gin-gonic/gin"
	"ltc-system/apps/api/internal/modules/ops/app"
)

func auditContext(c *gin.Context) app.AuditContext {
	ip := c.ClientIP()
	ua := c.Request.UserAgent()
	return app.AuditContext{IPAddress: &ip, UserAgent: &ua}
}
