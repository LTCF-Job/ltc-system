package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"ltc-system/apps/api/internal/middleware"
	"ltc-system/apps/api/internal/service"
)

// DashboardHandler 處理儀表板圖表與指標請求。
type DashboardHandler struct {
	dashboardSvc *service.DashboardService
}

// NewDashboardHandler 建立 DashboardHandler 實例。
func NewDashboardHandler(dashboardSvc *service.DashboardService) *DashboardHandler {
	return &DashboardHandler{dashboardSvc: dashboardSvc}
}

// GetMetrics 取得全方位儀表板圖表與統計指標。
func (h *DashboardHandler) GetMetrics(c *gin.Context) {
	periodYm := c.Query("month")
	metrics, err := h.dashboardSvc.GetMetrics(c.Request.Context(), periodYm)
	if err != nil {
		middleware.RespondError(c, http.StatusInternalServerError, middleware.CodeInternalError, err.Error(), nil)
		return
	}

	middleware.RespondSuccess(c, http.StatusOK, metrics, nil)
}
