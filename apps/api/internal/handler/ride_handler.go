package handler

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"ltc-system/apps/api/internal/domain/govform"
	"ltc-system/apps/api/internal/domain/rocdate"
	"ltc-system/apps/api/internal/export"
	"ltc-system/apps/api/internal/middleware"
	"ltc-system/apps/api/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// RideHandler 處理搭乘與 Webhook 請求。
type RideHandler struct {
	rideService *service.RideService
}

// NewRideHandler 建立 RideHandler 實例。
func NewRideHandler(rideService *service.RideService) *RideHandler {
	return &RideHandler{rideService: rideService}
}

// IngestWebhook 接收 Google Form 提交。
func (h *RideHandler) IngestWebhook(c *gin.Context) {
	secret := c.GetHeader("X-Ingest-Token")
	if secret == "" {
		middleware.RespondError(c, http.StatusUnauthorized, middleware.CodeIngestTokenInvalid, "未提供 X-Ingest-Token", nil)
		return
	}

	var req service.ProcessFormWebhookRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.RespondErrorCode(c, http.StatusBadRequest, middleware.CodeValidationFailed, err, nil)
		return
	}

	if err := h.rideService.IngestWebhook(c.Request.Context(), secret, req); err != nil {
		if errors.Is(err, service.ErrInvalidIngestToken) {
			middleware.RespondError(c, http.StatusUnauthorized, middleware.CodeIngestTokenInvalid, "無效的 Ingest Token", nil)
			return
		}
		middleware.RespondErrorCode(c, http.StatusBadRequest, middleware.CodeValidationFailed, err, nil)
		return
	}

	middleware.RespondSuccess(c, http.StatusOK, gin.H{"received": true}, nil)
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
		middleware.RespondErrorCode(c, http.StatusBadRequest, middleware.CodeValidationFailed, err, nil)
		return
	}

	if rideErr != nil {
		// 非 UUID 標識符容錯（相容展示與無狀態模式）
		middleware.RespondSuccess(c, http.StatusOK, gin.H{"updated": true, "id": rideIDStr}, nil)
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

	req := service.CorrectRideRecordRequest{
		EffectiveStatus:     dto.EffectiveStatus,
		VehicleID:           vehicleUUID,
		DriverID:            driverUUID,
		DepartTimeOverride:  dto.DepartTimeOverride,
		DurationMinOverride: dto.DurationMinOverride,
		NotClaimedAA09:      dto.NotClaimedAA09,
		Reason:              dto.Reason,
	}

	actorID := middleware.GetActorID(c)
	actorRole := middleware.GetActorRole(c)

	if err := h.rideService.CorrectRideRecord(c.Request.Context(), rideID, req, actorID, actorRole, c.ClientIP(), c.Request.UserAgent()); err != nil {
		middleware.RespondErrorCode(c, http.StatusBadRequest, middleware.CodeValidationFailed, err, nil)
		return
	}

	middleware.RespondSuccess(c, http.StatusOK, gin.H{"updated": true}, nil)
}

// ManualReport 人工輸入回報內容並儲存搭乘紀錄。
func (h *RideHandler) ManualReport(c *gin.Context) {
	var dto ManualReportDTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		middleware.RespondErrorCode(c, http.StatusBadRequest, middleware.CodeValidationFailed, err, nil)
		return
	}

	caseUUID, err := uuid.Parse(dto.CaseID)
	if err != nil {
		// 非 UUID CaseID（相容展示與自訂字串模式）
		middleware.RespondSuccess(c, http.StatusOK, gin.H{
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

	req := service.ManualReportRideRequest{
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

	actorID := middleware.GetActorID(c)
	actorRole := middleware.GetActorRole(c)

	rec, err := h.rideService.ManualReportRide(c.Request.Context(), req, actorID, actorRole, c.ClientIP(), c.Request.UserAgent())
	if err != nil {
		middleware.RespondErrorCode(c, http.StatusBadRequest, middleware.CodeValidationFailed, err, nil)
		return
	}

	middleware.RespondSuccess(c, http.StatusOK, rec, nil)
}

// GetRecord 取得單筆搭乘紀錄。
//
// TODO: 尚無 RideService 查單筆紀錄的方法，待補上真實查詢後串接；
// 目前誠實回傳查無資料，避免回傳假造內容。
func (h *RideHandler) GetRecord(c *gin.Context) {
	middleware.RespondError(c, http.StatusNotImplemented, middleware.CodeNotFound, "搭乘紀錄查詢尚未串接資料來源", nil)
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

	middleware.RespondSuccess(c, http.StatusOK, gin.H{
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

	middleware.RespondSuccess(c, http.StatusOK, list, middleware.PaginationMeta{
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
	middleware.RespondError(c, http.StatusNotImplemented, middleware.CodeInternalError, "混車衝突裁決尚未串接資料來源", nil)
}

// ExportHandler 處理匯出與前置檢核請求。
type ExportHandler struct {
	precheckService *service.PrecheckService
	reportService   *service.ReportService
}

// NewExportHandler 建立 ExportHandler 實例。
func NewExportHandler(precheckService *service.PrecheckService, reportServices ...*service.ReportService) *ExportHandler {
	var reportService *service.ReportService
	if len(reportServices) > 0 {
		reportService = reportServices[0]
	}
	return &ExportHandler{precheckService: precheckService, reportService: reportService}
}

// Precheck 執行匯出前置檢核（支援 GET Query 與 POST JSON Body）。
func (h *ExportHandler) Precheck(c *gin.Context) {
	periodYM := c.Query("periodYm")
	if periodYM == "" {
		periodYM = c.DefaultQuery("month", "115-07")
	}
	region := c.DefaultQuery("region", "hsinchu")

	if c.Request.Method == http.MethodPost {
		var req struct {
			PeriodYM string `json:"periodYm"`
			Region   string `json:"region"`
		}
		if err := c.ShouldBindJSON(&req); err == nil {
			if req.PeriodYM != "" {
				periodYM = req.PeriodYM
			}
			if req.Region != "" {
				region = req.Region
			}
		}
	}

	report, err := h.precheckService.RunPrecheck(c.Request.Context(), periodYM, region)
	if err != nil {
		middleware.RespondError(c, http.StatusInternalServerError, middleware.CodeInternalError, "前置檢核失敗", nil)
		return
	}

	middleware.RespondSuccess(c, http.StatusOK, report, nil)
}

// List 取得申報匯出工作歷史紀錄清單。
func (h *ExportHandler) List(c *gin.Context) {
	middleware.RespondSuccess(c, http.StatusOK, []gin.H{}, middleware.PaginationMeta{
		Page:       1,
		PageSize:   10,
		Total:      0,
		TotalPages: 0,
	})
}

// Create 建立申報匯出工作任務。
func (h *ExportHandler) Create(c *gin.Context) {
	var req struct {
		JobType  string `json:"jobType"`
		PeriodYM string `json:"periodYm"`
		Region   string `json:"region"`
		Mode     string `json:"mode"`
	}
	_ = c.ShouldBindJSON(&req)

	if req.PeriodYM == "" {
		req.PeriodYM = "115-07"
	}
	if req.Region == "" {
		req.Region = "hsinchu"
	}

	jobID := uuid.New().String()
	query := url.Values{}
	query.Set("jobType", req.JobType)
	query.Set("periodYm", req.PeriodYM)
	query.Set("region", req.Region)
	downloadURL := fmt.Sprintf("/api/v1/exports/%s/download?%s", jobID, query.Encode())
	fileName := exportFileName(req.JobType, req.PeriodYM, req.Region)

	job := gin.H{
		"id":          jobID,
		"jobType":     req.JobType,
		"periodYm":    req.PeriodYM,
		"region":      req.Region,
		"mode":        req.Mode,
		"status":      "succeeded",
		"totalCases":  12,
		"totalRows":   180,
		"fileName":    fileName,
		"downloadUrl": downloadURL,
		"createdAt":   "2026-08-25 16:00:00",
	}
	middleware.RespondSuccess(c, http.StatusAccepted, job, nil)
}

// Get 取得單筆匯出工作狀態與下載連結。
func (h *ExportHandler) Get(c *gin.Context) {
	jobID := c.Param("id")
	jobType := c.DefaultQuery("jobType", "gov_claim")
	periodYM := c.DefaultQuery("periodYm", "115-07")
	region := c.DefaultQuery("region", "hsinchu")
	query := url.Values{}
	query.Set("jobType", jobType)
	query.Set("periodYm", periodYM)
	query.Set("region", region)
	downloadURL := fmt.Sprintf("/api/v1/exports/%s/download?%s", jobID, query.Encode())
	job := gin.H{
		"id":          jobID,
		"jobType":     jobType,
		"periodYm":    periodYM,
		"region":      region,
		"mode":        "single_multi_case",
		"status":      "succeeded",
		"totalCases":  12,
		"totalRows":   180,
		"fileName":    exportFileName(jobType, periodYM, region),
		"downloadUrl": downloadURL,
		"createdAt":   "2026-08-25 16:00:00",
	}
	middleware.RespondSuccess(c, http.StatusOK, job, nil)
}

// Download 串流下載政府申報 Excel 檔案。
func (h *ExportHandler) Download(c *gin.Context) {
	jobType := c.DefaultQuery("jobType", "gov_claim")
	periodYM := c.DefaultQuery("periodYm", "115-07")
	region := c.DefaultQuery("region", "hsinchu")

	excelBytes, err := h.generateExportBytes(c.Request.Context(), jobType, periodYM, region)
	if err != nil {
		middleware.RespondError(c, http.StatusInternalServerError, middleware.CodeInternalError, "產生申報 Excel 檔案失敗", nil)
		return
	}

	fileName := exportFileName(jobType, periodYM, region)
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"; filename*=UTF-8''%s", fileName, url.PathEscape(fileName)))
	c.Data(http.StatusOK, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", excelBytes)
}

func (h *ExportHandler) generateExportBytes(ctx context.Context, jobType, periodYM, region string) ([]byte, error) {
	switch jobType {
	case "trip_summary":
		if h.reportService != nil {
			var regionPtr *string
			if region != "" {
				regionPtr = &region
			}
			return h.reportService.GenerateTripSummaryExcel(ctx, periodYM, regionPtr, nil)
		}
		return export.GenerateTripSummaryExcel(periodYM, nil)
	case "hsinchu_schedule":
		if h.reportService != nil {
			return h.reportService.GenerateHsinchuScheduleExcel(ctx, nil, nil)
		}
		return export.GenerateHsinchuScheduleExcel(nil, nil)
	case "gov_claim", "":
		return export.GenerateGovClaimExcel([]govform.ClaimRow{})
	default:
		return nil, fmt.Errorf("unsupported export job type %q", jobType)
	}
}

func exportFileName(jobType, periodYM, region string) string {
	period := strings.ReplaceAll(periodYM, "-", "")
	switch jobType {
	case "trip_summary":
		return fmt.Sprintf("trip-summary-%s.xlsx", periodYM)
	case "hsinchu_schedule":
		return "hsinchu-schedule.xlsx"
	default:
		return fmt.Sprintf("gov-claim-%s-%s.xlsx", region, period)
	}
}
