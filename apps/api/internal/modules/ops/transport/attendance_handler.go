package transport

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"ltc-system/apps/api/internal/modules/ops/app"
	"ltc-system/apps/api/internal/platform/auth"
	"ltc-system/apps/api/internal/platform/httpx"
)

// AttendanceHandler 處理司機出勤與請假 API 請求。
type AttendanceHandler struct {
	attendanceSvc *app.AttendanceService
}

// NewAttendanceHandler 建立 AttendanceHandler 實例。
func NewAttendanceHandler(attendanceSvc *app.AttendanceService) *AttendanceHandler {
	return &AttendanceHandler{attendanceSvc: attendanceSvc}
}

// GetMonthAttendance 查詢指定月份出勤矩陣。
func (h *AttendanceHandler) GetMonthAttendance(c *gin.Context) {
	periodYm := c.Query("month")
	var driverID *uuid.UUID
	if dIDStr := c.Query("driverId"); dIDStr != "" {
		if id, err := uuid.Parse(dIDStr); err == nil {
			driverID = &id
		}
	}

	report, err := h.attendanceSvc.GetMonthAttendance(c.Request.Context(), periodYm, driverID)
	if err != nil {
		httpx.RespondErrorCode(c, http.StatusInternalServerError, httpx.CodeInternalError, err, nil)
		return
	}

	httpx.RespondSuccess(c, http.StatusOK, report, nil)
}

// Upsert 登記單日出勤狀態。
func (h *AttendanceHandler) Upsert(c *gin.Context) {
	var req struct {
		DriverID   uuid.UUID `json:"driverId" binding:"required"`
		RecordDate string    `json:"recordDate" binding:"required"`
		Status     string    `json:"status" binding:"required,oneof=work leave sick off"`
		Note       *string   `json:"note"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.RespondErrorCode(c, http.StatusBadRequest, httpx.CodeValidationFailed, err, nil)
		return
	}

	recDate, err := time.Parse("2006-01-02", req.RecordDate)
	if err != nil {
		httpx.RespondError(c, http.StatusBadRequest, httpx.CodeValidationFailed, "日期格式必須為 YYYY-MM-DD", nil)
		return
	}

	actorID := auth.GetActorID(c)
	actorRole := auth.GetActorRole(c)
	item, err := h.attendanceSvc.Upsert(c.Request.Context(), req.DriverID, recDate, req.Status, req.Note, &actorID, &actorRole)
	if err != nil {
		httpx.RespondErrorCode(c, http.StatusInternalServerError, httpx.CodeInternalError, err, nil)
		return
	}

	httpx.RespondSuccess(c, http.StatusOK, item, nil)
}
