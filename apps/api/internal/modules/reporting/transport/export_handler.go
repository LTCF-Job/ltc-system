package transport

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"ltc-system/apps/api/internal/domain/govform"
	"ltc-system/apps/api/internal/modules/reporting/app"
	"ltc-system/apps/api/internal/platform/httpx"
)

// ExportHandler 處理匯出與前置檢核請求。
type ExportHandler struct {
	precheckService *app.PrecheckService
	reportService   *app.ReportService
}

// NewExportHandler 建立 ExportHandler 實例。
func NewExportHandler(precheckService *app.PrecheckService, reportService *app.ReportService) *ExportHandler {
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
		httpx.RespondError(c, http.StatusInternalServerError, httpx.CodeInternalError, "前置檢核失敗", nil)
		return
	}

	httpx.RespondSuccess(c, http.StatusOK, report, nil)
}

// List 取得申報匯出工作歷史紀錄清單。
func (h *ExportHandler) List(c *gin.Context) {
	httpx.RespondSuccess(c, http.StatusOK, []gin.H{}, httpx.PaginationMeta{
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
	httpx.RespondSuccess(c, http.StatusAccepted, job, nil)
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
	httpx.RespondSuccess(c, http.StatusOK, job, nil)
}

// Download 串流下載政府申報 Excel 檔案。
func (h *ExportHandler) Download(c *gin.Context) {
	jobType := c.DefaultQuery("jobType", "gov_claim")
	periodYM := c.DefaultQuery("periodYm", "115-07")
	region := c.DefaultQuery("region", "hsinchu")

	excelBytes, err := h.generateExportBytes(c.Request.Context(), jobType, periodYM, region)
	if err != nil {
		httpx.RespondError(c, http.StatusInternalServerError, httpx.CodeInternalError, "產生申報 Excel 檔案失敗", nil)
		return
	}

	fileName := exportFileName(jobType, periodYM, region)
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"; filename*=UTF-8''%s", fileName, url.PathEscape(fileName)))
	c.Data(http.StatusOK, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", excelBytes)
}

func (h *ExportHandler) generateExportBytes(ctx context.Context, jobType, periodYM, region string) ([]byte, error) {
	switch jobType {
	case "trip_summary":
		var regionPtr *string
		if region != "" {
			regionPtr = &region
		}
		return h.reportService.GenerateTripSummaryExcel(ctx, periodYM, regionPtr, nil)
	case "hsinchu_schedule":
		return h.reportService.GenerateHsinchuScheduleExcel(ctx, nil, nil)
	case "gov_claim", "":
		return h.reportService.GenerateGovClaimExcel([]govform.ClaimRow{})
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
