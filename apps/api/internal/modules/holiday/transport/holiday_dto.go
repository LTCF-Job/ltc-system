package transport

import (
	"time"

	"ltc-system/apps/api/internal/modules/holiday/app"
)

// holidayDate 同時是前端假日 map 的 key 與 DELETE /holidays/:date 的路徑片段，
// 後端只解析 YYYY-MM-DD，因此不能輸出 RFC3339。
const holidayDateLayout = "2006-01-02"

// HolidayResponse 是假日設定對外的 API 契約，欄位與
// apps/web/src/api/holidays.ts 的 HolidayItem 對齊。
type HolidayResponse struct {
	HolidayDate string    `json:"holidayDate"`
	Name        string    `json:"name"`
	Source      string    `json:"source"`
	IsDayOff    bool      `json:"isDayOff"`
	CreatedAt   time.Time `json:"createdAt"`
}

func newHolidayResponse(h app.Holiday) HolidayResponse {
	return HolidayResponse{
		HolidayDate: h.HolidayDate.Format(holidayDateLayout),
		Name:        h.Name,
		Source:      h.Source,
		IsDayOff:    h.IsDayOff,
		CreatedAt:   h.CreatedAt,
	}
}

func newHolidayResponses(list []app.Holiday) []HolidayResponse {
	out := make([]HolidayResponse, 0, len(list))
	for _, h := range list {
		out = append(out, newHolidayResponse(h))
	}
	return out
}
