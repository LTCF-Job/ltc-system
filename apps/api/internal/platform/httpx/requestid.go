package httpx

import (
	"crypto/rand"
	"encoding/hex"
	"strings"

	"github.com/gin-gonic/gin"
)

// RequestIDHeader 是回應中攜帶請求識別碼的 header 名稱。
const RequestIDHeader = "X-Request-Id"

// requestIDContextKey 是請求識別碼在 gin context 中的鍵。
const requestIDContextKey = "httpx.requestID"

// maxInboundRequestIDLength 限制沿用外部傳入識別碼的長度，避免呼叫端把任意長字串
// 灌進 log 與回應 header。
const maxInboundRequestIDLength = 64

// RequestIDMiddleware 為每個請求產生識別碼，寫入 gin context 與回應 header。
// 錯誤回應會一併帶回這個識別碼，讓使用者回報問題時後端能直接以它查 log。
func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := sanitizeRequestID(c.GetHeader(RequestIDHeader))
		if id == "" {
			id = newRequestID()
		}
		c.Set(requestIDContextKey, id)
		c.Writer.Header().Set(RequestIDHeader, id)
		c.Next()
	}
}

// RequestID 取出目前請求的識別碼；middleware 未掛載時回傳空字串。
func RequestID(c *gin.Context) string {
	if c == nil {
		return ""
	}
	if id, ok := c.Get(requestIDContextKey); ok {
		if s, ok := id.(string); ok {
			return s
		}
	}
	return ""
}

// sanitizeRequestID 只沿用由英數、連字號與底線組成的識別碼，其餘一律重新產生，
// 避免上游傳入的控制字元污染 log 或 response header。
func sanitizeRequestID(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" || len(raw) > maxInboundRequestIDLength {
		return ""
	}
	for _, r := range raw {
		isAllowed := (r >= '0' && r <= '9') ||
			(r >= 'a' && r <= 'z') ||
			(r >= 'A' && r <= 'Z') ||
			r == '-' || r == '_'
		if !isAllowed {
			return ""
		}
	}
	return raw
}

// newRequestID 產生 12 碼大寫十六進位識別碼，短到使用者能從畫面唸出來回報。
func newRequestID() string {
	buf := make([]byte, 6)
	if _, err := rand.Read(buf); err != nil {
		// 亂數不可用時仍要有識別碼可寫入 log，退回固定前綴而非讓請求失敗。
		return "NORANDOMID"
	}
	return strings.ToUpper(hex.EncodeToString(buf))
}
