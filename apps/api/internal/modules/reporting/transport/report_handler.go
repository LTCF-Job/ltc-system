package transport

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"ltc-system/apps/api/internal/domain/rocdate"
	"ltc-system/apps/api/internal/modules/reporting/app"
	"ltc-system/apps/api/internal/platform/httpx"
)

// ReportHandler 處理趟數表與新竹接送時刻表等報表相關 HTTP 請求。
type ReportHandler struct {
	reportSvc *app.ReportService
}

// NewReportHandler 建立 ReportHandler 實例。
func NewReportHandler(reportSvc *app.ReportService) *ReportHandler {
	return &ReportHandler{reportSvc: reportSvc}
}

// GetTripSummary 查詢車輛趟數表。
func (h *ReportHandler) GetTripSummary(c *gin.Context) {
	periodYm := c.DefaultQuery("periodYm", "115-07")
	if !validatePeriod(c, periodYm) {
		return
	}
	region := c.Query("region")
	var regionPtr *string
	if region != "" {
		regionPtr = &region
	}

	vehID, ok := parseOptionalUUID(c, "vehicleId")
	if !ok {
		return
	}

	report, err := h.reportSvc.GetTripSummary(c.Request.Context(), periodYm, regionPtr, vehID)
	if err != nil {
		httpx.RespondErrorCode(c, http.StatusInternalServerError, httpx.CodeInternalError, err, nil)
		return
	}

	c.JSON(http.StatusOK, report)
}

// ExportTripSummaryExcel 匯出車輛趟數表 Excel 檔案。
func (h *ReportHandler) ExportTripSummaryExcel(c *gin.Context) {
	periodYm := c.DefaultQuery("periodYm", "115-07")
	if !validatePeriod(c, periodYm) {
		return
	}
	region := c.Query("region")
	var regionPtr *string
	if region != "" {
		regionPtr = &region
	}

	vehID, ok := parseOptionalUUID(c, "vehicleId")
	if !ok {
		return
	}

	excelBytes, err := h.reportSvc.GenerateTripSummaryExcel(c.Request.Context(), periodYm, regionPtr, vehID)
	if err != nil {
		httpx.RespondErrorCode(c, http.StatusInternalServerError, httpx.CodeInternalError, err, nil)
		return
	}

	filename := fmt.Sprintf("trip-summary-%s.xlsx", periodYm)
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	c.Data(http.StatusOK, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", excelBytes)
}

// GetHsinchuSchedule 查詢新竹接送時刻表。
func (h *ReportHandler) GetHsinchuSchedule(c *gin.Context) {
	siteID, ok := parseOptionalUUID(c, "siteId")
	if !ok {
		return
	}
	vehID, ok := parseOptionalUUID(c, "vehicleId")
	if !ok {
		return
	}

	report, err := h.reportSvc.GetHsinchuSchedule(c.Request.Context(), siteID, vehID)
	if err != nil {
		httpx.RespondErrorCode(c, http.StatusInternalServerError, httpx.CodeInternalError, err, nil)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": report})
}

// ExportHsinchuScheduleExcel 匯出新竹接送時刻表 Excel 檔案。
func (h *ReportHandler) ExportHsinchuScheduleExcel(c *gin.Context) {
	siteID, ok := parseOptionalUUID(c, "siteId")
	if !ok {
		return
	}
	vehID, ok := parseOptionalUUID(c, "vehicleId")
	if !ok {
		return
	}

	excelBytes, err := h.reportSvc.GenerateHsinchuScheduleExcel(c.Request.Context(), siteID, vehID)
	if err != nil {
		httpx.RespondErrorCode(c, http.StatusInternalServerError, httpx.CodeInternalError, err, nil)
		return
	}

	filename := "hsinchu-schedule.xlsx"
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	c.Data(http.StatusOK, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", excelBytes)
}

func validatePeriod(c *gin.Context, period string) bool {
	if _, _, err := rocdate.ParseYearMonth(period); err != nil {
		httpx.RespondError(c, http.StatusBadRequest, httpx.CodeValidationFailed, "月份格式錯誤，請使用 RRR-MM 或 YYYY-MM", nil)
		return false
	}
	return true
}

func parseOptionalUUID(c *gin.Context, key string) (*uuid.UUID, bool) {
	raw := c.Query(key)
	if raw == "" {
		return nil, true
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		httpx.RespondError(c, http.StatusBadRequest, httpx.CodeValidationFailed, key+" 必須為有效 UUID", nil)
		return nil, false
	}
	return &id, true
}
