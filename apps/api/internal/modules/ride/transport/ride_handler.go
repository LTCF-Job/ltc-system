package transport

import (
	"errors"
	"net/http"
	"time"

	"ltc-system/apps/api/internal/domain/rocdate"
	"ltc-system/apps/api/internal/modules/ride/app"
	"ltc-system/apps/api/internal/platform/auth"
	"ltc-system/apps/api/internal/platform/clock"
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

// CorrectDTO 用於寬容接收搭乘更正請求。
type CorrectDTO struct {
	EffectiveStatus     *string `json:"effectiveStatus"`
	VehicleID           *string `json:"vehicleId"`
	DriverID            *string `json:"driverId"`
	DepartTimeOverride  *string `json:"departTimeOverride"`
	DurationMinOverride *int16  `json:"durationMinOverride"`
	NotClaimedAA09      *bool   `json:"notClaimedAa09"`
	Reason              *string `json:"reason"`
	BasedOnFingerprint  *string `json:"basedOnFingerprint" binding:"required"`
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
	if err := httpx.BindJSONStrict(c, &dto); err != nil {
		httpx.RespondErrorCode(c, http.StatusBadRequest, httpx.CodeValidationFailed, err, nil)
		return
	}

	if rideErr != nil {
		httpx.RespondError(c, http.StatusBadRequest, httpx.CodeValidationFailed, "無效的搭乘紀錄 ID", nil)
		return
	}

	var vehicleUUID *uuid.UUID
	if dto.VehicleID != nil && *dto.VehicleID != "" {
		v, err := uuid.Parse(*dto.VehicleID)
		if err != nil {
			httpx.RespondError(c, http.StatusBadRequest, httpx.CodeValidationFailed, "無效的車輛 ID", nil)
			return
		}
		vehicleUUID = &v
	}

	var driverUUID *uuid.UUID
	if dto.DriverID != nil && *dto.DriverID != "" {
		d, err := uuid.Parse(*dto.DriverID)
		if err != nil {
			httpx.RespondError(c, http.StatusBadRequest, httpx.CodeValidationFailed, "無效的司機 ID", nil)
			return
		}
		driverUUID = &d
	}

	req := app.CorrectRideRecordRequest{
		EffectiveStatus:     dto.EffectiveStatus,
		VehicleID:           vehicleUUID,
		DriverID:            driverUUID,
		DepartTimeOverride:  dto.DepartTimeOverride,
		DurationMinOverride: dto.DurationMinOverride,
		NotClaimedAA09:      dto.NotClaimedAA09,
		Reason:              dto.Reason,
		BasedOnFingerprint:  dto.BasedOnFingerprint,
	}

	actorID := auth.GetActorID(c)
	actorRole := auth.GetActorRole(c)

	if err := h.rideService.CorrectRideRecord(c.Request.Context(), rideID, req, actorID, actorRole, c.ClientIP(), c.Request.UserAgent()); err != nil {
		if errors.Is(err, app.ErrRideNotFound) {
			httpx.RespondErrorCode(c, http.StatusNotFound, httpx.CodeNotFound, err, nil)
			return
		}
		if errors.Is(err, app.ErrStaleCorrection) {
			httpx.RespondErrorCode(c, http.StatusConflict, httpx.CodeResourceInUse, err, nil)
			return
		}
		httpx.RespondErrorCode(c, http.StatusBadRequest, httpx.CodeValidationFailed, err, nil)
		return
	}

	httpx.RespondSuccess(c, http.StatusOK, gin.H{"updated": true}, nil)
}

// ManualReport 人工輸入回報內容並儲存搭乘紀錄。
func (h *RideHandler) ManualReport(c *gin.Context) {
	var dto ManualReportDTO
	if err := httpx.BindJSONStrict(c, &dto); err != nil {
		httpx.RespondErrorCode(c, http.StatusBadRequest, httpx.CodeValidationFailed, err, nil)
		return
	}

	caseUUID, err := uuid.Parse(dto.CaseID)
	if err != nil {
		httpx.RespondError(c, http.StatusBadRequest, httpx.CodeValidationFailed, "無效的個案 ID", nil)
		return
	}

	var vehicleUUID *uuid.UUID
	if dto.VehicleID != nil && *dto.VehicleID != "" {
		v, err := uuid.Parse(*dto.VehicleID)
		if err != nil {
			httpx.RespondError(c, http.StatusBadRequest, httpx.CodeValidationFailed, "無效的車輛 ID", nil)
			return
		}
		vehicleUUID = &v
	}

	var driverUUID *uuid.UUID
	if dto.DriverID != nil && *dto.DriverID != "" {
		d, err := uuid.Parse(*dto.DriverID)
		if err != nil {
			httpx.RespondError(c, http.StatusBadRequest, httpx.CodeValidationFailed, "無效的司機 ID", nil)
			return
		}
		driverUUID = &d
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

// rideRecordResponse 是搭乘紀錄的對外回應形狀。
type rideRecordResponse struct {
	ID                     string  `json:"id"`
	CaseID                 string  `json:"caseId"`
	CaseName               string  `json:"caseName"`
	ServiceDate            string  `json:"serviceDate"`
	LegSeq                 int16   `json:"legSeq"`
	MergedStatus           string  `json:"mergedStatus"`
	EffectiveStatus        string  `json:"effectiveStatus"`
	VehicleID              string  `json:"vehicleId"`
	VehicleName            string  `json:"vehicleName"`
	DriverID               *string `json:"driverId"`
	DriverName             string  `json:"driverName"`
	HasConflict            bool    `json:"hasConflict"`
	ConflictResolvedAt     *string `json:"conflictResolvedAt"`
	ConflictResolutionNote *string `json:"conflictResolutionNote"`
	BasedOnFingerprint     string  `json:"basedOnFingerprint"`
	DepartTimeOverride     *string `json:"departTimeOverride"`
	DurationMinOverride    *int16  `json:"durationMinOverride"`
	NotClaimedAA09         bool    `json:"notClaimedAa09"`
	CorrectedBy            *string `json:"correctedBy"`
	CorrectedAt            *string `json:"correctedAt"`
	CorrectionReason       *string `json:"correctionReason"`
}

func toRideRecordResponse(rec *app.RideRecord) rideRecordResponse {
	resp := rideRecordResponse{
		ID:                     rec.ID.String(),
		CaseID:                 rec.CaseID.String(),
		CaseName:               rec.CaseName,
		ServiceDate:            rec.ServiceDate.Format("2006-01-02"),
		LegSeq:                 rec.LegSeq,
		MergedStatus:           rec.MergedStatus,
		EffectiveStatus:        rec.EffectiveStatus,
		VehicleID:              rec.VehicleID.String(),
		VehicleName:            rec.VehicleName,
		DriverName:             rec.DriverName,
		HasConflict:            rec.HasConflict,
		ConflictResolutionNote: rec.ConflictResolutionNote,
		BasedOnFingerprint:     rec.BasedOnFingerprint,
		DepartTimeOverride:     rec.DepartTimeOverride,
		DurationMinOverride:    rec.DurationMinOverride,
		NotClaimedAA09:         rec.NotClaimedAA09,
		CorrectionReason:       rec.CorrectionReason,
	}
	if rec.DriverID != nil {
		s := rec.DriverID.String()
		resp.DriverID = &s
	}
	if rec.CorrectedBy != nil {
		s := rec.CorrectedBy.String()
		resp.CorrectedBy = &s
	}
	if rec.CorrectedAt != nil {
		s := rec.CorrectedAt.Format(time.RFC3339)
		resp.CorrectedAt = &s
	}
	if rec.ConflictResolvedAt != nil {
		s := rec.ConflictResolvedAt.Format(time.RFC3339)
		resp.ConflictResolvedAt = &s
	}
	return resp
}

// GetRecord 取得單筆搭乘紀錄。
func (h *RideHandler) GetRecord(c *gin.Context) {
	rideID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		httpx.RespondError(c, http.StatusBadRequest, httpx.CodeValidationFailed, "無效的搭乘紀錄 ID", nil)
		return
	}

	rec, err := h.rideService.GetRecord(c.Request.Context(), rideID)
	if err != nil {
		if errors.Is(err, app.ErrRideNotFound) {
			httpx.RespondErrorCode(c, http.StatusNotFound, httpx.CodeNotFound, err, nil)
			return
		}
		httpx.RespondErrorCode(c, http.StatusInternalServerError, httpx.CodeInternalError, err, nil)
		return
	}

	httpx.RespondSuccess(c, http.StatusOK, toRideRecordResponse(rec), nil)
}

// GetCalendar 取得搭乘月曆矩陣資料。月份接受民國（115-07）與西元（2026-07）兩種寫法。
func (h *RideHandler) GetCalendar(c *gin.Context) {
	now := clock.Now()
	monthStr := c.DefaultQuery("month", rocdate.FormatROCYearMonth(now.Year(), int(now.Month())))

	start, _, _, err := rocdate.MonthRangeStrict(monthStr)
	if err != nil {
		httpx.RespondError(c, http.StatusBadRequest, httpx.CodeValidationFailed, "月份格式錯誤，請使用 RRR-MM 或 YYYY-MM", nil)
		return
	}
	matrix, err := h.rideService.GetCalendar(c.Request.Context(), start.Year(), int(start.Month()), c.Query("region"), c.Query("q"))
	if err != nil {
		httpx.RespondErrorCode(c, http.StatusInternalServerError, httpx.CodeInternalError, err, nil)
		return
	}

	httpx.RespondSuccess(c, http.StatusOK, matrix, nil)
}

// issueTypeWhitelist 是 issueType 查詢參數的合法值。
var issueTypeWhitelist = map[string]bool{"conflict": true, "unreported": true, "import_error": true}

// issueRideResponse 是「異常集中處理」清單單一列的對外回應形狀。
type issueRideResponse struct {
	ID          string   `json:"id"`
	CaseID      string   `json:"caseId"`
	CaseName    string   `json:"caseName"`
	ServiceDate string   `json:"serviceDate"`
	LegSeq      int16    `json:"legSeq"`
	Description string   `json:"description"`
	Vehicles    []string `json:"vehicles,omitempty"`
	RawPayload  string   `json:"rawPayload,omitempty"`
}

func toIssueRideResponse(item app.IssueRide) issueRideResponse {
	return issueRideResponse{
		ID:          item.ID,
		CaseID:      item.CaseID,
		CaseName:    item.CaseName,
		ServiceDate: item.ServiceDate.Format("2006-01-02"),
		LegSeq:      item.LegSeq,
		Description: item.Description,
		Vehicles:    item.Vehicles,
		RawPayload:  item.RawPayload,
	}
}

// ListIssues 取得異常集中處理清單；issueType 為 conflict/unreported/import_error 三擇一。
func (h *RideHandler) ListIssues(c *gin.Context) {
	issueType := c.Query("issueType")
	if !issueTypeWhitelist[issueType] {
		httpx.RespondError(c, http.StatusBadRequest, httpx.CodeValidationFailed, "無效的 issueType", nil)
		return
	}

	now := clock.Now()
	monthStr := c.DefaultQuery("month", rocdate.FormatROCYearMonth(now.Year(), int(now.Month())))
	start, _, _, err := rocdate.MonthRangeStrict(monthStr)
	if err != nil {
		httpx.RespondError(c, http.StatusBadRequest, httpx.CodeValidationFailed, "月份格式錯誤，請使用 RRR-MM 或 YYYY-MM", nil)
		return
	}

	page, pageSize, err := httpx.ParsePagination(c)
	if err != nil {
		httpx.RespondError(c, http.StatusBadRequest, httpx.CodeValidationFailed, "分頁參數格式錯誤", nil)
		return
	}

	items, total, err := h.rideService.ListIssues(c.Request.Context(), issueType, start.Year(), int(start.Month()), c.Query("region"), c.Query("keyword"), page, pageSize)
	if err != nil {
		httpx.RespondErrorCode(c, http.StatusInternalServerError, httpx.CodeInternalError, err, nil)
		return
	}

	list := make([]issueRideResponse, 0, len(items))
	for _, item := range items {
		list = append(list, toIssueRideResponse(item))
	}

	totalPages := int(total) / pageSize
	if int(total)%pageSize != 0 {
		totalPages++
	}
	httpx.RespondSuccess(c, http.StatusOK, list, httpx.PaginationMeta{
		Page:       page,
		PageSize:   pageSize,
		Total:      total,
		TotalPages: totalPages,
	})
}

// ResolveConflictDTO 用於接收混車衝突裁決請求。
type ResolveConflictDTO struct {
	VehicleID string  `json:"vehicleId" binding:"required"`
	DriverID  *string `json:"driverId"`
	Reason    *string `json:"reason"`
}

// ResolveConflict 人工裁決混車衝突。
func (h *RideHandler) ResolveConflict(c *gin.Context) {
	rideID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		httpx.RespondError(c, http.StatusBadRequest, httpx.CodeValidationFailed, "無效的搭乘紀錄 ID", nil)
		return
	}

	var dto ResolveConflictDTO
	if err := httpx.BindJSONStrict(c, &dto); err != nil {
		httpx.RespondErrorCode(c, http.StatusBadRequest, httpx.CodeValidationFailed, err, nil)
		return
	}

	vehicleID, err := uuid.Parse(dto.VehicleID)
	if err != nil {
		httpx.RespondError(c, http.StatusBadRequest, httpx.CodeValidationFailed, "無效的車輛 ID", nil)
		return
	}

	var driverID *uuid.UUID
	if dto.DriverID != nil && *dto.DriverID != "" {
		if d, err := uuid.Parse(*dto.DriverID); err == nil {
			driverID = &d
		}
	}

	actorID := auth.GetActorID(c)
	actorRole := auth.GetActorRole(c)

	err = h.rideService.ResolveConflict(c.Request.Context(), rideID, app.ResolveConflictInput{
		VehicleID: vehicleID,
		DriverID:  driverID,
		Reason:    dto.Reason,
	}, actorID, actorRole)
	if err != nil {
		if errors.Is(err, app.ErrRideNotFound) {
			httpx.RespondErrorCode(c, http.StatusNotFound, httpx.CodeNotFound, err, nil)
			return
		}
		if errors.Is(err, app.ErrConflictAlreadyResolved) {
			httpx.RespondErrorCode(c, http.StatusConflict, httpx.CodeResourceInUse, err, nil)
			return
		}
		httpx.RespondErrorCode(c, http.StatusInternalServerError, httpx.CodeInternalError, err, nil)
		return
	}

	httpx.RespondSuccess(c, http.StatusOK, gin.H{"resolved": true}, nil)
}
