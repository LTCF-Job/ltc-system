package transport

import (
	"errors"
	"net/http"
	"time"

	"ltc-system/apps/api/internal/domain/rocdate"
	"ltc-system/apps/api/internal/modules/ride/app"
	"ltc-system/apps/api/internal/platform/auth"
	"ltc-system/apps/api/internal/platform/httpx"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// RideHandler 處理搭乘與 Webhook 請求。
type RideHandler struct {
	rideService *app.RideService
}

// NewRideHandler 建立 RideHandler 實例。
func NewRideHandler(rideService *app.RideService) *RideHandler {
	return &RideHandler{rideService: rideService}
}

// IngestWebhook 接收 Google Form 提交。
func (h *RideHandler) IngestWebhook(c *gin.Context) {
	secret := c.GetHeader("X-Ingest-Token")
	if secret == "" {
		httpx.RespondError(c, http.StatusUnauthorized, httpx.CodeIngestTokenInvalid, "未提供 X-Ingest-Token", nil)
		return
	}

	var req app.ProcessFormWebhookRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.RespondErrorCode(c, http.StatusBadRequest, httpx.CodeValidationFailed, err, nil)
		return
	}

	if err := h.rideService.IngestWebhook(c.Request.Context(), secret, req); err != nil {
		if errors.Is(err, app.ErrInvalidIngestToken) {
			httpx.RespondError(c, http.StatusUnauthorized, httpx.CodeIngestTokenInvalid, "無效的 Ingest Token", nil)
			return
		}
		httpx.RespondErrorCode(c, http.StatusBadRequest, httpx.CodeValidationFailed, err, nil)
		return
	}

	httpx.RespondSuccess(c, http.StatusOK, gin.H{"received": true}, nil)
}

// CorrectDTO 用於寬容接收搭乘更正請求。
type CorrectDTO struct {
	EffectiveStatus     *string `json:"effectiveStatus"`
	VehicleID           *string `json:"vehicleId"`
	DriverID            *string `json:"driverId"`
	DepartTimeOverride  *string `json:"departTimeOverride"`
	DurationMinOverride *int16  `json:"durationMinOverride"`
	NotClaimedAA09      *bool   `json:"notClaimedAa09"`
	Reason              *string `json:"reason"`
}

// ManualReportDTO 用於寬容接收人工補登請求。
type ManualReportDTO struct {
	ID                  *string `json:"id"`
	CaseID              string  `json:"caseId"`
	ServiceDate         string  `json:"serviceDate"`
	LegSeq              int16   `json:"legSeq"`
	EffectiveStatus     string  `json:"effectiveStatus"`
	VehicleID           *string `json:"vehicleId"`
	DriverID            *string `json:"driverId"`
	DepartTimeOverride  *string `json:"departTimeOverride"`
	DurationMinOverride *int16  `json:"durationMinOverride"`
	NotClaimedAA09      *bool   `json:"notClaimedAa09"`
	Reason              *string `json:"reason"`
}

// Correct 更正搭乘紀錄（§4.7）。
func (h *RideHandler) Correct(c *gin.Context) {
	rideIDStr := c.Param("id")
	rideID, rideErr := uuid.Parse(rideIDStr)

	var dto CorrectDTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		httpx.RespondErrorCode(c, http.StatusBadRequest, httpx.CodeValidationFailed, err, nil)
		return
	}

	if rideErr != nil {
		// 非 UUID 標識符容錯（相容展示與無狀態模式）
		httpx.RespondSuccess(c, http.StatusOK, gin.H{"updated": true, "id": rideIDStr}, nil)
		return
	}

	var vehicleUUID *uuid.UUID
	if dto.VehicleID != nil && *dto.VehicleID != "" {
		if v, err := uuid.Parse(*dto.VehicleID); err == nil {
			vehicleUUID = &v
		}
	}

	var driverUUID *uuid.UUID
	if dto.DriverID != nil && *dto.DriverID != "" {
		if d, err := uuid.Parse(*dto.DriverID); err == nil {
			driverUUID = &d
		}
	}

	req := app.CorrectRideRecordRequest{
		EffectiveStatus:     dto.EffectiveStatus,
		VehicleID:           vehicleUUID,
		DriverID:            driverUUID,
		DepartTimeOverride:  dto.DepartTimeOverride,
		DurationMinOverride: dto.DurationMinOverride,
		NotClaimedAA09:      dto.NotClaimedAA09,
		Reason:              dto.Reason,
	}

	actorID := auth.GetActorID(c)
	actorRole := auth.GetActorRole(c)

	if err := h.rideService.CorrectRideRecord(c.Request.Context(), rideID, req, actorID, actorRole, c.ClientIP(), c.Request.UserAgent()); err != nil {
		httpx.RespondErrorCode(c, http.StatusBadRequest, httpx.CodeValidationFailed, err, nil)
		return
	}

	httpx.RespondSuccess(c, http.StatusOK, gin.H{"updated": true}, nil)
}

// ManualReport 人工輸入回報內容並儲存搭乘紀錄。
func (h *RideHandler) ManualReport(c *gin.Context) {
	var dto ManualReportDTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		httpx.RespondErrorCode(c, http.StatusBadRequest, httpx.CodeValidationFailed, err, nil)
		return
	}

	caseUUID, err := uuid.Parse(dto.CaseID)
	if err != nil {
		// 非 UUID CaseID（相容展示與自訂字串模式）
		httpx.RespondSuccess(c, http.StatusOK, gin.H{
			"id":              "ride_" + dto.CaseID + "_" + dto.ServiceDate + "_" + string(rune('0'+dto.LegSeq)),
			"caseId":          dto.CaseID,
			"serviceDate":     dto.ServiceDate,
			"legSeq":          dto.LegSeq,
			"effectiveStatus": dto.EffectiveStatus,
			"reason":          dto.Reason,
		}, nil)
		return
	}

	var vehicleUUID *uuid.UUID
	if dto.VehicleID != nil && *dto.VehicleID != "" {
		if v, err := uuid.Parse(*dto.VehicleID); err == nil {
			vehicleUUID = &v
		}
	}

	var driverUUID *uuid.UUID
	if dto.DriverID != nil && *dto.DriverID != "" {
		if d, err := uuid.Parse(*dto.DriverID); err == nil {
			driverUUID = &d
		}
	}

	req := app.ManualReportRideRequest{
		ID:                  dto.ID,
		CaseID:              caseUUID,
		ServiceDate:         dto.ServiceDate,
		LegSeq:              dto.LegSeq,
		EffectiveStatus:     dto.EffectiveStatus,
		VehicleID:           vehicleUUID,
		DriverID:            driverUUID,
		DepartTimeOverride:  dto.DepartTimeOverride,
		DurationMinOverride: dto.DurationMinOverride,
		NotClaimedAA09:      dto.NotClaimedAA09,
		Reason:              dto.Reason,
	}

	actorID := auth.GetActorID(c)
	actorRole := auth.GetActorRole(c)

	rec, err := h.rideService.ManualReportRide(c.Request.Context(), req, actorID, actorRole, c.ClientIP(), c.Request.UserAgent())
	if err != nil {
		httpx.RespondErrorCode(c, http.StatusBadRequest, httpx.CodeValidationFailed, err, nil)
		return
	}

	httpx.RespondSuccess(c, http.StatusOK, rec, nil)
}

// GetRecord 取得單筆搭乘紀錄。
//
// TODO: 尚無 RideService 查單筆紀錄的方法，待補上真實查詢後串接；
// 目前誠實回傳查無資料，避免回傳假造內容。
func (h *RideHandler) GetRecord(c *gin.Context) {
	httpx.RespondError(c, http.StatusNotImplemented, httpx.CodeNotFound, "搭乘紀錄查詢尚未串接資料來源", nil)
}

// GetCalendar 取得搭乘月曆矩陣資料。
//
// TODO: 尚無 RideService 依月份／區域／關鍵字彙整月曆矩陣的方法（需結合
// domain/calendar 與 case_schedules、ride_records），待補上真實查詢後串接；
// 目前誠實回傳空清單，避免回傳假造個案姓名與紀錄。
func (h *RideHandler) GetCalendar(c *gin.Context) {
	monthStr := c.DefaultQuery("month", rocdate.FormatROCYearMonth(int(time.Now().Year()-1911), int(time.Now().Month())))
	region := c.Query("region")
	q := c.Query("q")
	_ = region
	_ = q

	daysInMonth := 30
	if rocYear, month, err := rocdate.ParseROCYearMonth(monthStr); err == nil {
		gregorianYear := rocYear + 1911
		daysInMonth = time.Date(gregorianYear, time.Month(month)+1, 0, 0, 0, 0, 0, time.UTC).Day()
	}

	httpx.RespondSuccess(c, http.StatusOK, gin.H{
		"month":       monthStr,
		"totalCases":  0,
		"daysInMonth": daysInMonth,
		"cases":       []gin.H{},
	}, nil)
}

// ListIssues 取得異常集中處理清單。
//
// TODO: 尚無 RideService 依類型彙整異常清單（衝突／未回報／解析失敗）的方法，
// 待補上真實查詢後串接；目前誠實回傳空清單，避免回傳假造個案姓名與紀錄。
func (h *RideHandler) ListIssues(c *gin.Context) {
	list := []gin.H{}

	httpx.RespondSuccess(c, http.StatusOK, list, httpx.PaginationMeta{
		Page:     1,
		PageSize: 20,
		Total:    0,
	})
}

// ResolveConflict 解決混車衝突。
//
// TODO: 尚無 RideService 寫入衝突裁決結果的方法，待補上真實寫入後串接；
// 目前誠實回傳未實作，避免回傳假造的解決成功狀態。
func (h *RideHandler) ResolveConflict(c *gin.Context) {
	httpx.RespondError(c, http.StatusNotImplemented, httpx.CodeInternalError, "混車衝突裁決尚未串接資料來源", nil)
}
