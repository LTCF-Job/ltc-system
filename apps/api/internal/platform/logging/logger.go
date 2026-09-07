package logging

import (
	"log/slog"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// Middleware 以 JSON 結構化日誌記錄傳入請求與執行耗時。
func Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := safeQuerySummary(c.Request.URL.Query())

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()
		clientIP := c.ClientIP()
		method := c.Request.Method

		slog.Info("HTTP Request",
			slog.Int("status", status),
			slog.String("method", method),
			slog.String("path", path),
			slog.String("query", query),
			slog.String("ip", clientIP),
			slog.Duration("latency", latency),
			slog.String("error", c.Errors.ByType(gin.ErrorTypePrivate).String()),
		)
	}
}

// safeQuerySummary 只保留不含個資的篩選與分頁欄位；q、nationalId、phone 等查詢值不進入 log。
func safeQuerySummary(values url.Values) string {
	allowed := []string{"page", "pageSize", "status", "region", "topic", "issueType"}
	parts := make([]string, 0, len(allowed))
	for _, key := range allowed {
		if value := strings.TrimSpace(values.Get(key)); value != "" {
			parts = append(parts, key+"="+url.QueryEscape(value))
		}
	}
	if len(parts) == 0 {
		if len(values) > 0 {
			return "hasQuery=true"
		}
		return ""
	}
	return strings.Join(parts, "&")
}
