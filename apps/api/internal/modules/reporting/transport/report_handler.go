package transport

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
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
	region := c.Query("region")
	var regionPtr *string
	if region != "" {
		regionPtr = &region
	}

	var vehID *uuid.UUID
	if vStr := c.Query("vehicleId"); vStr != "" {
		if uid, err := uuid.Parse(vStr); err == nil {
			vehID = &uid
		}
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
	region := c.Query("region")
	var regionPtr *string
	if region != "" {
		regionPtr = &region
	}

	var vehID *uuid.UUID
	if vStr := c.Query("vehicleId"); vStr != "" {
		if uid, err := uuid.Parse(vStr); err == nil {
			vehID = &uid
		}
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
	var siteID, vehID *uuid.UUID
	if sStr := c.Query("siteId"); sStr != "" {
		if uid, err := uuid.Parse(sStr); err == nil {
			siteID = &uid
		}
	}
	if vStr := c.Query("vehicleId"); vStr != "" {
		if uid, err := uuid.Parse(vStr); err == nil {
			vehID = &uid
		}
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
	var siteID, vehID *uuid.UUID
	if sStr := c.Query("siteId"); sStr != "" {
		if uid, err := uuid.Parse(sStr); err == nil {
			siteID = &uid
		}
	}
	if vStr := c.Query("vehicleId"); vStr != "" {
		if uid, err := uuid.Parse(vStr); err == nil {
			vehID = &uid
		}
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
