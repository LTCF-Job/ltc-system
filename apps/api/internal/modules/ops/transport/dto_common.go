package transport

import "time"

// 與 request 端接受的格式相同，同一欄位送出與收回的形狀才會一致。
const dateOnlyLayout = "2006-01-02"

func formatDateOnly(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(dateOnlyLayout)
}
