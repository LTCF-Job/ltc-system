package main

import (
	"net/http"
	"strings"
	"time"

	audittransport "ltc-system/apps/api/internal/modules/audit/transport"
	caregivertransport "ltc-system/apps/api/internal/modules/caregiver/transport"
	importtransport "ltc-system/apps/api/internal/modules/caseimport/transport"
	casetransport "ltc-system/apps/api/internal/modules/casemgmt/transport"
	drtransport "ltc-system/apps/api/internal/modules/driverreport/transport"
	holidaytransport "ltc-system/apps/api/internal/modules/holiday/transport"
	mastertransport "ltc-system/apps/api/internal/modules/masterdata/transport"
	notifytransport "ltc-system/apps/api/internal/modules/notification/transport"
	opstransport "ltc-system/apps/api/internal/modules/ops/transport"
	reporttransport "ltc-system/apps/api/internal/modules/reporting/transport"
	ridetransport "ltc-system/apps/api/internal/modules/ride/transport"
	tasktransport "ltc-system/apps/api/internal/modules/task/transport"
	"ltc-system/apps/api/internal/platform/auth"
	"ltc-system/apps/api/internal/platform/config"
	"ltc-system/apps/api/internal/platform/logging"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

// handlers 收集所有要註冊到路由的 delivery adapter。
type handlers struct {
	region       *mastertransport.RegionHandler
	kase         *casetransport.CaseHandler
	caseImport   *importtransport.ImportHandler
	site         *mastertransport.SiteHandler
	vehicle      *mastertransport.VehicleHandler
	driver       *mastertransport.DriverHandler
	ride         *ridetransport.RideHandler
	export       *reporttransport.ExportHandler
	notification *notifytransport.NotificationHandler
	holiday      *holidaytransport.HolidayHandler
	report       *reporttransport.ReportHandler
	audit        *audittransport.AuditHandler
	task         *tasktransport.TaskHandler
	maintenance  *opstransport.MaintenanceHandler
	attendance   *opstransport.AttendanceHandler
	fuel         *opstransport.FuelHandler
	dashboard    *reporttransport.DashboardHandler
	driverReport *drtransport.DriverReportHandler
	caregiver    *caregivertransport.CaregiverHandler
}

// newRouter 組裝 gin engine：全域 middleware、CORS、健康檢查與 v1 路由表。
func newRouter(cfg *config.Config, pool *pgxpool.Pool, h handlers) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(logging.Middleware())

	// CORS 設定：正式環境限制為白名單網域，本機開發維持全放行以配合任意 port 測試
	corsConfig := cors.DefaultConfig()
	if cfg.AppEnv == "production" {
		corsConfig.AllowOrigins = strings.Split(cfg.AllowedOrigins, ",")
	} else {
		corsConfig.AllowAllOrigins = true
	}
	corsConfig.AllowHeaders = []string{"Origin", "Content-Length", "Content-Type", "Authorization", "X-Ingest-Token", "X-Mock-Role", "X-Mock-User-ID"}
	corsConfig.AllowMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}
	r.Use(cors.New(corsConfig))

	// 健康檢查端點 (健康檢查不走 JWT)。避免使用 /healthz：Cloud Run 預設網域的 Google 前端會保留攔截此精確路徑，導致外部請求收不到回應。
	r.GET("/api/health", func(c *gin.Context) {
		dbStatus := "connected"
		if pool == nil || pool.Ping(c.Request.Context()) != nil {
			dbStatus = "disconnected"
		}
		c.JSON(http.StatusOK, gin.H{
			"status":   "ok",
			"env":      cfg.AppEnv,
			"database": dbStatus,
			"time":     time.Now().UTC().Format(time.RFC3339),
		})
	})

	// 需要 JWT 認證之 API 群組
	apiV1 := r.Group("/api/v1")
	apiV1.Use(auth.Middleware(cfg))
	{
		// 0. 區域主檔
		apiV1.GET("/regions", auth.RequireRoles("viewer", "staff", "admin"), h.region.List)
		apiV1.GET("/regions/:id", auth.RequireRoles("viewer", "staff", "admin"), h.region.Get)
		apiV1.POST("/regions", auth.RequireRoles("staff", "admin"), h.region.Create)
		apiV1.PATCH("/regions/:id", auth.RequireRoles("staff", "admin"), h.region.Update)
		apiV1.DELETE("/regions/:id", auth.RequireRoles("admin"), h.region.Delete)

		// 1. 個案主檔與排班

		apiV1.GET("/cases", auth.RequireRoles("viewer", "staff", "admin"), h.kase.List)
		apiV1.POST("/cases", auth.RequireRoles("staff", "admin"), h.kase.Create)
		apiV1.GET("/cases/template", auth.RequireRoles("viewer", "staff", "admin"), h.caseImport.DownloadTemplate)
		apiV1.GET("/cases/export", auth.RequireRoles("viewer", "staff", "admin"), h.kase.ExportProfileWorkbook)
		apiV1.GET("/cases/:id", auth.RequireRoles("viewer", "staff", "admin"), h.kase.Get)
		apiV1.PATCH("/cases/:id", auth.RequireRoles("staff", "admin"), h.kase.Update)
		apiV1.PUT("/cases/:id/transport-preference", auth.RequireRoles("staff", "admin"), h.kase.UpdateTransportPreference)
		apiV1.POST("/cases/:id/reveal", auth.RequireRoles("staff", "admin"), h.kase.Reveal)
		apiV1.GET("/cases/:id/schedule", auth.RequireRoles("viewer", "staff", "admin"), h.kase.GetSchedule)
		apiV1.PUT("/cases/:id/schedule", auth.RequireRoles("staff", "admin"), h.kase.SaveSchedule)
		apiV1.POST("/cases/schedules", auth.RequireRoles("staff", "admin"), h.kase.CreateSchedule)
		apiV1.POST("/cases/import", auth.RequireRoles("staff", "admin"), h.caseImport.ImportExcel)
		apiV1.POST("/masters/import", auth.RequireRoles("staff", "admin"), h.caseImport.ImportExcel)

		// 2. 單位主檔
		apiV1.GET("/sites", auth.RequireRoles("viewer", "staff", "admin"), h.site.List)
		apiV1.POST("/sites", auth.RequireRoles("staff", "admin"), h.site.Create)
		apiV1.PATCH("/sites/:id", auth.RequireRoles("staff", "admin"), h.site.Update)
		apiV1.DELETE("/sites/:id", auth.RequireRoles("admin"), h.site.Delete)

		// 3. 車輛主檔
		apiV1.GET("/vehicles", auth.RequireRoles("viewer", "staff", "admin"), h.vehicle.List)
		apiV1.POST("/vehicles", auth.RequireRoles("staff", "admin"), h.vehicle.Create)
		apiV1.PATCH("/vehicles/:id", auth.RequireRoles("staff", "admin"), h.vehicle.Update)
		apiV1.PUT("/vehicles/:id/drivers", auth.RequireRoles("staff", "admin"), h.vehicle.SetDrivers)

		// 4. 司機主檔
		apiV1.GET("/drivers", auth.RequireRoles("viewer", "staff", "admin"), h.driver.List)
		apiV1.POST("/drivers", auth.RequireRoles("staff", "admin"), h.driver.Create)
		apiV1.PATCH("/drivers/:id", auth.RequireRoles("staff", "admin"), h.driver.Update)
		apiV1.POST("/drivers/:id/reveal", auth.RequireRoles("staff", "admin"), h.driver.Reveal)
		apiV1.POST("/drivers/:id/assignments", auth.RequireRoles("staff", "admin"), h.driver.AssignVehicle)

		// 5. 司機接送匯報表與欄位對應
		apiV1.GET("/driver-reports", auth.RequireRoles("viewer", "staff", "admin"), h.driverReport.ListForms)
		apiV1.POST("/driver-reports", auth.RequireRoles("staff", "admin"), h.driverReport.CreateForm)
		apiV1.GET("/driver-reports/imported-months", auth.RequireRoles("viewer", "staff", "admin"), h.driverReport.ListImportedMonths)
		apiV1.GET("/driver-reports/columns", auth.RequireRoles("viewer", "staff", "admin"), h.driverReport.ListColumns)
		apiV1.PATCH("/driver-reports/columns/:id/mapping", auth.RequireRoles("staff", "admin"), h.driverReport.UpdateColumnMapping)
		apiV1.POST("/driver-reports/columns/batch-mapping", auth.RequireRoles("staff", "admin"), h.driverReport.BatchMapping)
		apiV1.DELETE("/driver-reports/:id", auth.RequireRoles("staff", "admin"), h.driverReport.DeleteForm)
		apiV1.GET("/driver-reports/:id/template", auth.RequireRoles("staff", "admin"), h.driverReport.DownloadTemplate)
		apiV1.POST("/driver-reports/:id/import", auth.RequireRoles("staff", "admin"), h.driverReport.ImportExcel)

		// 6. 搭乘月曆、搭乘紀錄更正、異常搭乘與未回報清單
		apiV1.GET("/rides/calendar", auth.RequireRoles("viewer", "staff", "admin"), h.ride.GetCalendar)
		apiV1.GET("/rides/issues", auth.RequireRoles("viewer", "staff", "admin"), h.ride.ListIssues)
		apiV1.GET("/rides/missing", auth.RequireRoles("viewer", "staff", "admin"), h.task.GetMissingReports)
		apiV1.GET("/rides/:id", auth.RequireRoles("viewer", "staff", "admin"), h.ride.GetRecord)
		apiV1.PATCH("/rides/:id", auth.RequireRoles("staff", "admin"), h.ride.Correct)
		apiV1.POST("/rides/manual-report", auth.RequireRoles("staff", "admin"), h.ride.ManualReport)
		apiV1.POST("/rides/:id/resolve-conflict", auth.RequireRoles("staff", "admin"), h.ride.ResolveConflict)

		// 7. 匯出前置檢核與工作管理 (B4.3)
		apiV1.GET("/exports/precheck", auth.RequireRoles("staff", "admin"), h.export.Precheck)
		apiV1.POST("/exports/precheck", auth.RequireRoles("staff", "admin"), h.export.Precheck)
		apiV1.GET("/exports", auth.RequireRoles("viewer", "staff", "admin"), h.export.List)
		apiV1.POST("/exports", auth.RequireRoles("staff", "admin"), h.export.Create)
		apiV1.GET("/exports/:id", auth.RequireRoles("viewer", "staff", "admin"), h.export.Get)
		apiV1.GET("/exports/:id/download", auth.RequireRoles("viewer", "staff", "admin"), h.export.Download)
		apiV1.GET("/exports/:id/files/:caseId/download", auth.RequireRoles("viewer", "staff", "admin"), h.export.DownloadCaseFile)

		// 8. 國定假日與行事曆管理 (B5.1)
		apiV1.GET("/holidays", auth.RequireRoles("viewer", "staff", "admin"), h.holiday.List)
		apiV1.POST("/holidays", auth.RequireRoles("staff", "admin"), h.holiday.Create)
		apiV1.POST("/holidays/import", auth.RequireRoles("staff", "admin"), h.holiday.Import)
		apiV1.DELETE("/holidays/:date", auth.RequireRoles("admin"), h.holiday.Delete)

		// 9. 通知收件人管理與通知留痕 (B5.2b)
		apiV1.GET("/settings/notification-recipients", auth.RequireRoles("viewer", "staff", "admin"), h.notification.ListRecipients)
		apiV1.POST("/settings/notification-recipients", auth.RequireRoles("admin"), h.notification.CreateRecipient)
		apiV1.PATCH("/settings/notification-recipients/:id", auth.RequireRoles("admin"), h.notification.UpdateRecipient)
		apiV1.DELETE("/settings/notification-recipients/:id", auth.RequireRoles("admin"), h.notification.DeleteRecipient)
		apiV1.GET("/notifications/logs", auth.RequireRoles("viewer", "staff", "admin"), h.notification.ListLogs)

		// 10. 營運報表 - 車輛趟數表 (B5.4) 與 新竹接送時刻表 (B6.1)
		apiV1.GET("/reports/trip-summary", auth.RequireRoles("viewer", "staff", "admin"), h.report.GetTripSummary)
		apiV1.GET("/reports/trip-summary/export", auth.RequireRoles("viewer", "staff", "admin"), h.report.ExportTripSummaryExcel)
		apiV1.GET("/reports/hsinchu-schedule", auth.RequireRoles("viewer", "staff", "admin"), h.report.GetHsinchuSchedule)
		apiV1.GET("/reports/hsinchu-schedule/export", auth.RequireRoles("viewer", "staff", "admin"), h.report.ExportHsinchuScheduleExcel)

		// 11. 車輛維修保養管理與空白表下載 (B6.2)
		apiV1.GET("/vehicles/maintenance", auth.RequireRoles("viewer", "staff", "admin"), h.maintenance.List)
		apiV1.POST("/vehicles/maintenance", auth.RequireRoles("staff", "admin"), h.maintenance.Create)
		apiV1.PATCH("/vehicles/maintenance/:id", auth.RequireRoles("staff", "admin"), h.maintenance.Update)
		apiV1.DELETE("/vehicles/maintenance/:id", auth.RequireRoles("staff", "admin"), h.maintenance.Delete)
		apiV1.GET("/vehicles/maintenance/blank-template", auth.RequireRoles("viewer", "staff", "admin"), h.maintenance.DownloadBlankTemplate)

		// 12. 司機出勤與請假登錄 (B6.3)
		apiV1.GET("/attendance", auth.RequireRoles("viewer", "staff", "admin"), h.attendance.GetMonthAttendance)
		apiV1.POST("/attendance", auth.RequireRoles("staff", "admin"), h.attendance.Upsert)

		// 13. 車輛油資管理 (B6.3)
		apiV1.GET("/fuel-logs", auth.RequireRoles("viewer", "staff", "admin"), h.fuel.List)
		apiV1.POST("/fuel-logs", auth.RequireRoles("staff", "admin"), h.fuel.Create)
		apiV1.PATCH("/fuel-logs/:id", auth.RequireRoles("staff", "admin"), h.fuel.Update)
		apiV1.DELETE("/fuel-logs/:id", auth.RequireRoles("staff", "admin"), h.fuel.Delete)

		// 14. 視覺化儀表板指標 (B6.4)
		apiV1.GET("/dashboard/metrics", auth.RequireRoles("viewer", "staff", "admin"), h.dashboard.GetMetrics)
		apiV1.GET("/dashboard/stats", auth.RequireRoles("viewer", "staff", "admin"), h.dashboard.GetStats)

		// 15. 稽核紀錄查詢 (B5.5 - 僅限 admin)
		apiV1.GET("/audit", auth.RequireRoles("admin"), h.audit.List)

		// 16. 排程與維運任務端點 (B5.2 / B5.3)
		apiV1.POST("/tasks/check-missing-reports", auth.RequireRoles("staff", "admin"), h.task.CheckMissingReports)
		apiV1.POST("/tasks/month-end-reminder", auth.RequireRoles("staff", "admin"), h.task.MonthEndReminder)

		// 17. 照護人員主檔管理
		apiV1.GET("/caregivers", auth.RequireRoles("viewer", "staff", "admin"), h.caregiver.List)
		apiV1.POST("/caregivers", auth.RequireRoles("staff", "admin"), h.caregiver.Create)
		apiV1.GET("/caregivers/template", auth.RequireRoles("viewer", "staff", "admin"), h.caregiver.DownloadTemplate)
		apiV1.POST("/caregivers/import", auth.RequireRoles("staff", "admin"), h.caregiver.ImportExcel)
		apiV1.PATCH("/caregivers/:id", auth.RequireRoles("staff", "admin"), h.caregiver.Update)
		apiV1.DELETE("/caregivers/:id", auth.RequireRoles("admin"), h.caregiver.Delete)
		apiV1.PUT("/caregivers/:id/site", auth.RequireRoles("staff", "admin"), h.caregiver.LinkSite)
	}

	return r
}
