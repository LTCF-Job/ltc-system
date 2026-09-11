package main

import (
	"net/http"
	"strings"

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
// 實際自動關聯的筆數。四段分開執行，中途某段失敗時，訊息會列出已完成的段落名稱，
// 讓使用者知道還剩哪幾段需要重試，而不是整段訊息重來一次。
func (h *pendingRelinkHandler) Relink(c *gin.Context) {
	ctx := c.Request.Context()
	actorID := auth.GetActorID(c)
	actorRole := auth.GetActorRole(c)
	ip := c.ClientIP()
	ua := c.Request.UserAgent()

	var done []string

	sites, err := h.caseSvc.RelinkAllPendingSites(ctx, actorID, actorRole, ip, ua)
	if err != nil {
		respondRelinkError(c, err, done, "據點")
		return
	}
	done = append(done, "據點")

	caregivers, err := h.caseSvc.RelinkAllPendingCaregivers(ctx, actorID, actorRole, ip, ua)
	if err != nil {
		respondRelinkError(c, err, done, "照護人員")
		return
	}
	done = append(done, "照護人員")

	drivers, err := h.driverReportSvc.RelinkAllPendingDrivers(ctx)
	if err != nil {
		respondRelinkError(c, err, done, "司機")
		return
	}
	done = append(done, "司機")

	caseColumns, err := h.driverReportSvc.RelinkAllPendingCaseColumns(ctx)
	if err != nil {
		respondRelinkError(c, err, done, "匯報表單欄位")
		return
	}

	httpx.RespondSuccess(c, http.StatusOK, gin.H{
		"sites":       sites,
		"caregivers":  caregivers,
		"drivers":     drivers,
		"caseColumns": caseColumns,
	}, nil)
}

// respondRelinkError 回報重新比對失敗，並在 reason 中列出已完成的段落，
// 讓使用者知道還需要重試哪一段，而不必整個流程重來一次；err 只記錄於伺服器端 log。
func respondRelinkError(c *gin.Context, err error, done []string, failedStage string) {
	reason := "重新比對「" + failedStage + "」時失敗"
	if len(done) > 0 {
		reason += "（已完成：" + strings.Join(done, "、") + "），請重新整理後再試一次"
	} else {
		reason += "，請重新整理後再試一次"
	}
	httpx.RespondErrorCodeWithReason(c, http.StatusInternalServerError, httpx.CodeInternalError, err, reason, nil)
}
