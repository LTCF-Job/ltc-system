package transport

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"ltc-system/apps/api/internal/domain/rocdate"
	"ltc-system/apps/api/internal/modules/reporting/app"
	"ltc-system/apps/api/internal/platform/clock"
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
	periodYm := c.DefaultQuery("periodYm", currentPeriodYmDash())
	if !validatePeriod(c, periodYm) {
		return
	}

	vehID, ok := parseOptionalUUID(c, "vehicleId")
	if !ok {
		return
	}

	report, err := h.reportSvc.GetTripSummary(c.Request.Context(), periodYm, vehID)
	if err != nil {
		httpx.RespondErrorCode(c, http.StatusInternalServerError, httpx.CodeInternalError, err, nil)
		return
	}

	httpx.RespondSuccess(c, http.StatusOK, report, nil)
}

// ExportTripSummaryExcel 匯出車輛趟數表 Excel 檔案。
func (h *ReportHandler) ExportTripSummaryExcel(c *gin.Context) {
	periodYm := c.DefaultQuery("periodYm", currentPeriodYmDash())
	if !validatePeriod(c, periodYm) {
		return
	}

	vehID, ok := parseOptionalUUID(c, "vehicleId")
	if !ok {
		return
	}

	excelBytes, err := h.reportSvc.GenerateTripSummaryExcel(c.Request.Context(), periodYm, vehID)
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
	asOfDate, ok := parseAsOfDate(c)
	if !ok {
		return
	}
	siteID, ok := parseOptionalUUID(c, "siteId")
	if !ok {
		return
	}
	vehID, ok := parseOptionalUUID(c, "vehicleId")
	if !ok {
		return
	}

	report, err := h.reportSvc.GetHsinchuSchedule(c.Request.Context(), siteID, vehID, asOfDate)
	if err != nil {
		httpx.RespondErrorCode(c, http.StatusInternalServerError, httpx.CodeInternalError, err, nil)
		return
	}

	httpx.RespondSuccess(c, http.StatusOK, report, nil)
}

// ExportHsinchuScheduleExcel 匯出新竹接送時刻表 Excel 檔案。
func (h *ReportHandler) ExportHsinchuScheduleExcel(c *gin.Context) {
	asOfDate, ok := parseAsOfDate(c)
	if !ok {
		return
	}
	siteID, ok := parseOptionalUUID(c, "siteId")
	if !ok {
		return
	}
	vehID, ok := parseOptionalUUID(c, "vehicleId")
	if !ok {
		return
	}

	excelBytes, err := h.reportSvc.GenerateHsinchuScheduleExcel(c.Request.Context(), siteID, vehID, asOfDate)
	if err != nil {
		httpx.RespondErrorCode(c, http.StatusInternalServerError, httpx.CodeInternalError, err, nil)
		return
	}

	filename := "hsinchu-schedule.xlsx"
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	c.Data(http.StatusOK, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", excelBytes)
}

// currentPeriodYmDash 以系統目前日期推導 "RRR-MM" 格式的申報月份，
// 取代先前寫死的過期月份字串（原本固定回傳 "115-07"，與實際月份脫節）。
func currentPeriodYmDash() string {
	today := clock.Today()
	return rocdate.FormatROCYearMonth(today.Year(), int(today.Month()))
}

func parseAsOfDate(c *gin.Context) (time.Time, bool) {
	raw := c.Query("asOfDate")
	if raw == "" {
		return clock.Today(), true
	}
	date, err := rocdate.ParseDate(raw)
	if err != nil {
		httpx.RespondError(c, http.StatusBadRequest, httpx.CodeValidationFailed, "查詢日期格式不正確，請重新輸入", nil)
		return time.Time{}, false
	}
	return date, true
}

func validatePeriod(c *gin.Context, period string) bool {
	if _, _, err := rocdate.ParseYearMonth(period); err != nil {
		httpx.RespondError(c, http.StatusBadRequest, httpx.CodeValidationFailed, "月份格式錯誤，請使用 RRR-MM 或 YYYY-MM", nil)
		return false
	}
	return true
}

// uuidQueryParamLabels 讓 UUID 格式錯誤訊息以中文欄位語意呈現，
// 不直接外洩內部查詢參數名稱（如 vehicleId）與「UUID」這類技術術語。
var uuidQueryParamLabels = map[string]string{
	"vehicleId": "車輛",
	"siteId":    "據點",
}

func parseOptionalUUID(c *gin.Context, key string) (*uuid.UUID, bool) {
	raw := c.Query(key)
	if raw == "" {
		return nil, true
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		label := uuidQueryParamLabels[key]
		if label == "" {
			label = "查詢條件"
		}
		httpx.RespondError(c, http.StatusBadRequest, httpx.CodeValidationFailed, label+"格式不正確，請重新選擇", nil)
		return nil, false
	}
	return &id, true
}
