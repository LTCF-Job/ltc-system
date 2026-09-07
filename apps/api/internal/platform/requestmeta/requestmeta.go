package requestmeta

import "context"

type contextKey uint8

const metadataKey contextKey = iota

// Metadata 是 HTTP 請求可供稽核使用的來源資訊。
type Metadata struct {
	IPAddress string
	UserAgent string
}

// With 將請求來源資訊附加到 context；背景任務未提供時維持空字串。
func With(ctx context.Context, ipAddress, userAgent string) context.Context {
	return context.WithValue(ctx, metadataKey, Metadata{IPAddress: ipAddress, UserAgent: userAgent})
}

// FromContext 讀取請求來源資訊；沒有附加資料時回傳零值。
func FromContext(ctx context.Context) Metadata {
	if metadata, ok := ctx.Value(metadataKey).(Metadata); ok {
		return metadata
	}
	return Metadata{}
}
