package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
	caseapp "ltc-system/apps/api/internal/modules/casemgmt/app"
	drapp "ltc-system/apps/api/internal/modules/driverreport/app"
	"ltc-system/apps/api/internal/platform/auth"
	"ltc-system/apps/api/internal/platform/httpx"
)

// pendingRelinkHandler 讓使用者在待維護頁手動重跑一次待維護資料重新比對，
// 用於處理上線前的舊資料，或補救單次自動關聯的偶發失敗。
type pendingRelinkHandler struct {
	caseSvc         *caseapp.CaseService
	driverReportSvc *drapp.DriverReportService
}

func newPendingRelinkHandler(caseSvc *caseapp.CaseService, driverReportSvc *drapp.DriverReportService) *pendingRelinkHandler {
	return &pendingRelinkHandler{caseSvc: caseSvc, driverReportSvc: driverReportSvc}
}

// Relink 依序重新比對據點、照護人員、司機與匯報表單欄位四類待維護資料，回傳各類
// 實際自動關聯的筆數。
func (h *pendingRelinkHandler) Relink(c *gin.Context) {
	ctx := c.Request.Context()
	actorID := auth.GetActorID(c)
	actorRole := auth.GetActorRole(c)
	ip := c.ClientIP()
	ua := c.Request.UserAgent()

	sites, err := h.caseSvc.RelinkAllPendingSites(ctx, actorID, actorRole, ip, ua)
	if err != nil {
		httpx.RespondError(c, http.StatusInternalServerError, httpx.CodeInternalError, "重新比對待維護資料失敗", nil)
		return
	}
	caregivers, err := h.caseSvc.RelinkAllPendingCaregivers(ctx, actorID, actorRole, ip, ua)
	if err != nil {
		httpx.RespondError(c, http.StatusInternalServerError, httpx.CodeInternalError, "重新比對待維護資料失敗", nil)
		return
	}
	drivers, err := h.driverReportSvc.RelinkAllPendingDrivers(ctx)
	if err != nil {
		httpx.RespondError(c, http.StatusInternalServerError, httpx.CodeInternalError, "重新比對待維護資料失敗", nil)
		return
	}
	caseColumns, err := h.driverReportSvc.RelinkAllPendingCaseColumns(ctx)
	if err != nil {
		httpx.RespondError(c, http.StatusInternalServerError, httpx.CodeInternalError, "重新比對待維護資料失敗", nil)
		return
	}

	httpx.RespondSuccess(c, http.StatusOK, gin.H{
		"sites":       sites,
		"caregivers":  caregivers,
		"drivers":     drivers,
		"caseColumns": caseColumns,
	}, nil)
}
