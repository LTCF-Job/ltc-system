package handler

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"ltc-system/apps/api/internal/service"
)

// ReportHandler 處理趟數表等報表相關 HTTP 請求。
type ReportHandler struct {
	reportSvc *service.ReportService
}

// NewReportHandler 建立 ReportHandler 實例。
func NewReportHandler(reportSvc *service.ReportService) *ReportHandler {
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": err.Error()}})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": err.Error()}})
		return
	}

	filename := fmt.Sprintf("trip-summary-%s.xlsx", periodYm)
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	c.Data(http.StatusOK, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", excelBytes)
}
