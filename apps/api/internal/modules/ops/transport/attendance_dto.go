package transport

import (
	"github.com/google/uuid"
	"ltc-system/apps/api/internal/modules/ops/app"
)

// AttendanceRecordResponse 是單筆出勤登記對外的 API 契約，欄位與
// apps/web/src/types/api.d.ts 的 AttendanceRecordDTO 對齊。
type AttendanceRecordResponse struct {
	ID         uuid.UUID `json:"id"`
	DriverID   uuid.UUID `json:"driverId"`
	RecordDate string    `json:"recordDate"`
	Status     string    `json:"status"`
	Note       *string   `json:"note"`
	Source     string    `json:"source"`
}

func newAttendanceRecordResponse(a app.AttendanceRecord) AttendanceRecordResponse {
	return AttendanceRecordResponse{
		ID:         a.ID,
		DriverID:   a.DriverID,
		RecordDate: formatDateOnly(a.RecordDate),
		Status:     a.Status,
		Note:       a.Note,
		Source:     a.Source,
	}
}
