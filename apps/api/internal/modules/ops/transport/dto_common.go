package transport

import (
	"strings"
	"time"
)

// 與 request 端接受的格式相同，同一欄位送出與收回的形狀才會一致。
const dateOnlyLayout = "2006-01-02"

func formatDateOnly(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(dateOnlyLayout)
}

// isValidReceiptURL 允許空白或以 http(s):// 開頭的網址；拒絕 javascript: 等其他協定，
// 避免存入的值被前端直接渲染成 <a href> 時觸發儲存型 XSS。
func isValidReceiptURL(url *string) bool {
	if url == nil || *url == "" {
		return true
	}
	return strings.HasPrefix(*url, "http://") || strings.HasPrefix(*url, "https://")
}
