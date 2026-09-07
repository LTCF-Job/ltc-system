package transport

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"ltc-system/apps/api/internal/domain/rocdate"
	"ltc-system/apps/api/internal/modules/task/app"
	"ltc-system/apps/api/internal/platform/clock"
	"ltc-system/apps/api/internal/platform/httpx"
)

// TaskHandler 處理排程與後台任務相關之 HTTP 請求。
type TaskHandler struct {
	svc *app.TaskService
}

// NewTaskHandler 建立 TaskHandler 實例。
func NewTaskHandler(svc *app.TaskService) *TaskHandler {
	return &TaskHandler{svc: svc}
}

// CheckMissingReports 觸發未回報檢查並派送通知。
func (h *TaskHandler) CheckMissingReports(c *gin.Context) {
	dateStr := c.DefaultQuery("date", clock.Today().Format("2006-01-02"))
	region := c.Query("region")

	targetDate, err := rocdate.ParseDate(dateStr)
	if err != nil {
		httpx.RespondError(c, http.StatusBadRequest, httpx.CodeValidationFailed, "日期格式錯誤，請使用 YYYY-MM-DD", nil)
		return
	}

	missingList, err := h.svc.CheckMissingReports(c.Request.Context(), targetDate, region)
	if err != nil {
		httpx.RespondErrorCode(c, http.StatusInternalServerError, httpx.CodeInternalError, err, nil)
		return
	}

	httpx.RespondSuccess(c, http.StatusOK, gin.H{
		"date":           dateStr,
		"triggeredCount": len(missingList),
		"missingCount":   len(missingList),
		"items":          missingList,
		"message":        fmt.Sprintf("已成功執行未回報檢核，累計 %d 筆項目並記錄催報通知日誌。", len(missingList)),
	}, nil)
}

// GetMissingReports 供前端頁面查詢特定日期或今日之未回報清單。
func (h *TaskHandler) GetMissingReports(c *gin.Context) {
	dateStr := c.DefaultQuery("date", clock.Today().Format("2006-01-02"))
	region := c.Query("region")

	targetDate, err := rocdate.ParseDate(dateStr)
	if err != nil {
		httpx.RespondError(c, http.StatusBadRequest, httpx.CodeValidationFailed, "日期格式錯誤，請使用 YYYY-MM-DD", nil)
		return
	}

	missingList, err := h.svc.ListMissingReports(c.Request.Context(), targetDate, region)
	if err != nil {
		httpx.RespondErrorCode(c, http.StatusInternalServerError, httpx.CodeInternalError, err, nil)
		return
	}

	httpx.RespondSuccess(c, http.StatusOK, missingList, nil)
}

// MonthEndReminder 觸發每月 26 日申報提醒檢查與發信通知。
func (h *TaskHandler) MonthEndReminder(c *gin.Context) {
	today := clock.Today()
	monthStr := c.DefaultQuery("month", rocdate.FormatROCYearMonth(today.Year(), int(today.Month())))

	year, month, err := rocdate.ParseROCYearMonth(monthStr)
	if err != nil {
		httpx.RespondError(c, http.StatusBadRequest, httpx.CodeValidationFailed, "月份格式錯誤，請使用 RRR-MM（如 115-07）", nil)
		return
	}

	summary, err := h.svc.MonthEndReminder(c.Request.Context(), year, month)
	if err != nil {
		httpx.RespondError(c, http.StatusInternalServerError, httpx.CodeInternalError, "執行月底提醒任務失敗", nil)
		return
	}

	httpx.RespondSuccess(c, http.StatusOK, summary, nil)
}
