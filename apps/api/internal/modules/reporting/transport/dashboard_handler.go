package transport

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"ltc-system/apps/api/internal/modules/reporting/app"
	"ltc-system/apps/api/internal/platform/httpx"
)

// DashboardHandler 處理儀表板圖表與指標請求。
type DashboardHandler struct {
	dashboardSvc *app.DashboardService
}

// NewDashboardHandler 建立 DashboardHandler 實例。
func NewDashboardHandler(dashboardSvc *app.DashboardService) *DashboardHandler {
	return &DashboardHandler{dashboardSvc: dashboardSvc}
}

// GetMetrics 取得全方位儀表板圖表與統計指標。
func (h *DashboardHandler) GetMetrics(c *gin.Context) {
	periodYm := c.Query("month")
	metrics, err := h.dashboardSvc.GetMetrics(c.Request.Context(), periodYm)
	if err != nil {
		httpx.RespondErrorCode(c, http.StatusInternalServerError, httpx.CodeInternalError, err, nil)
		return
	}

	httpx.RespondSuccess(c, http.StatusOK, metrics, nil)
}

// GetStats 取得儀表板統計摘要與近期申報匯出紀錄清單。
func (h *DashboardHandler) GetStats(c *gin.Context) {
	stats := gin.H{
		"recentExports": []gin.H{},
	}
	httpx.RespondSuccess(c, http.StatusOK, stats, nil)
}
