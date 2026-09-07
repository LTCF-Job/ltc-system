package clock

import "time"

// Clock 是業務邏輯使用的時間來源，避免各模組自行取用不同時區的系統時間。
type Clock interface {
	Now() time.Time
	Today() time.Time
}

// SystemClock 以 Asia/Taipei 作為業務時間所在時區。
type SystemClock struct {
	Location *time.Location
}

// NewAsiaTaipei 建立正式環境使用的 Business Clock。
func NewAsiaTaipei() SystemClock {
	location, err := time.LoadLocation("Asia/Taipei")
	if err != nil {
		// Asia/Taipei 是 Go 時區資料的標準項目；若執行環境缺少 tzdata，
		// 固定 UTC+8 仍比退回伺服器本機時區安全。
		location = time.FixedZone("Asia/Taipei", 8*60*60)
	}
	return SystemClock{Location: location}
}

func (c SystemClock) location() *time.Location {
	if c.Location != nil {
		return c.Location
	}
	return NewAsiaTaipei().Location
}

// Now 回傳目前的 Asia/Taipei 業務時間。
func (c SystemClock) Now() time.Time {
	return time.Now().In(c.location())
}

// Today 回傳 Asia/Taipei 當地曆法日期，正規化為 UTC 午夜。
// 傳給 PostgreSQL 與 daterange 欄位比對（如 effective_range @> $1::date）時，
// 若改用 Asia/Taipei 午夜，pgx 編碼成 timestamptz 後經 UTC session timezone
// 轉換取日期會回退一天；固定用 UTC 表示同一個曆法日期可避免此問題，
// 不論 DB session timezone 為何都會落在正確日期。
func (c SystemClock) Today() time.Time {
	now := c.Now()
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
}

var system = NewAsiaTaipei()

// Now 是不需要注入測試時鐘之基礎 adapter 使用的共用 Business Clock。
func Now() time.Time { return system.Now() }

// Today 回傳目前 Asia/Taipei 的業務日期。
func Today() time.Time { return system.Today() }
