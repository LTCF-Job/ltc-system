package main

import (
	"net/http"
	"strings"
	"time"

	"ltc-system/apps/api/internal/config"
	"ltc-system/apps/api/internal/handler"
	"ltc-system/apps/api/internal/middleware"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

// handlers 收集所有要註冊到路由的 delivery adapter。
type handlers struct {
	region       *handler.RegionHandler
	kase         *handler.CaseHandler
	site         *handler.SiteHandler
	vehicle      *handler.VehicleHandler
	driver       *handler.DriverHandler
	ride         *handler.RideHandler
	export       *handler.ExportHandler
	notification *handler.NotificationHandler
	holiday      *handler.HolidayHandler
	report       *handler.ReportHandler
	audit        *handler.AuditHandler
	task         *handler.TaskHandler
	maintenance  *handler.MaintenanceHandler
	attendance   *handler.AttendanceHandler
	fuel         *handler.FuelHandler
	dashboard    *handler.DashboardHandler
	form         *handler.FormHandler
}

// newRouter 組裝 gin engine：全域 middleware、CORS、健康檢查與 v1 路由表。
func newRouter(cfg *config.Config, pool *pgxpool.Pool, h handlers) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.SlogLoggerMiddleware())

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

	// Webhook 端點 (X-Ingest-Token 驗證)
	r.POST("/api/v1/ingest/google-form", h.ride.IngestWebhook)

	// 需要 JWT 認證之 API 群組
	apiV1 := r.Group("/api/v1")
	apiV1.Use(middleware.AuthMiddleware(cfg))
	{
		// 0. 區域主檔
		apiV1.GET("/regions", middleware.RequireRoles("viewer", "staff", "admin"), h.region.List)
		apiV1.GET("/regions/:id", middleware.RequireRoles("viewer", "staff", "admin"), h.region.Get)
		apiV1.POST("/regions", middleware.RequireRoles("staff", "admin"), h.region.Create)
		apiV1.PATCH("/regions/:id", middleware.RequireRoles("staff", "admin"), h.region.Update)
		apiV1.DELETE("/regions/:id", middleware.RequireRoles("admin"), h.region.Delete)

		// 1. 個案主檔與排班

		apiV1.GET("/cases", middleware.RequireRoles("viewer", "staff", "admin"), h.kase.List)
		apiV1.POST("/cases", middleware.RequireRoles("staff", "admin"), h.kase.Create)
		apiV1.GET("/cases/template", middleware.RequireRoles("viewer", "staff", "admin"), h.kase.DownloadTemplate)
		apiV1.GET("/cases/export", middleware.RequireRoles("viewer", "staff", "admin"), h.kase.ExportProfileWorkbook)
		apiV1.GET("/cases/:id", middleware.RequireRoles("viewer", "staff", "admin"), h.kase.Get)
		apiV1.PATCH("/cases/:id", middleware.RequireRoles("staff", "admin"), h.kase.Update)
		apiV1.PUT("/cases/:id/transport-preference", middleware.RequireRoles("staff", "admin"), h.kase.UpdateTransportPreference)
		apiV1.POST("/cases/:id/reveal", middleware.RequireRoles("staff", "admin"), h.kase.Reveal)
		apiV1.GET("/cases/:id/schedule", middleware.RequireRoles("viewer", "staff", "admin"), h.kase.GetSchedule)
		apiV1.PUT("/cases/:id/schedule", middleware.RequireRoles("staff", "admin"), h.kase.SaveSchedule)
		apiV1.POST("/cases/schedules", middleware.RequireRoles("staff", "admin"), h.kase.CreateSchedule)
		apiV1.POST("/cases/import", middleware.RequireRoles("staff", "admin"), h.kase.ImportExcel)
		apiV1.POST("/masters/import", middleware.RequireRoles("staff", "admin"), h.kase.ImportExcel)

		// 2. 據點主檔
		apiV1.GET("/sites", middleware.RequireRoles("viewer", "staff", "admin"), h.site.List)
		apiV1.POST("/sites", middleware.RequireRoles("staff", "admin"), h.site.Create)
		apiV1.PATCH("/sites/:id", middleware.RequireRoles("staff", "admin"), h.site.Update)
		apiV1.DELETE("/sites/:id", middleware.RequireRoles("admin"), h.site.Delete)

		// 3. 車輛主檔
		apiV1.GET("/vehicles", middleware.RequireRoles("viewer", "staff", "admin"), h.vehicle.List)
		apiV1.POST("/vehicles", middleware.RequireRoles("staff", "admin"), h.vehicle.Create)
		apiV1.PATCH("/vehicles/:id", middleware.RequireRoles("staff", "admin"), h.vehicle.Update)

		// 4. 司機主檔
		apiV1.GET("/drivers", middleware.RequireRoles("viewer", "staff", "admin"), h.driver.List)
		apiV1.POST("/drivers", middleware.RequireRoles("staff", "admin"), h.driver.Create)
		apiV1.PATCH("/drivers/:id", middleware.RequireRoles("staff", "admin"), h.driver.Update)
		apiV1.POST("/drivers/:id/reveal", middleware.RequireRoles("staff", "admin"), h.driver.Reveal)
		apiV1.POST("/drivers/:id/assignments", middleware.RequireRoles("staff", "admin"), h.driver.AssignVehicle)

		// 5. 表單管理與欄位對應
		apiV1.GET("/forms", middleware.RequireRoles("viewer", "staff", "admin"), h.form.ListForms)
		apiV1.POST("/forms", middleware.RequireRoles("staff", "admin"), h.form.CreateFormAssociation)
		apiV1.DELETE("/forms/:id", middleware.RequireRoles("staff", "admin"), h.form.DeleteFormAssociation)
		apiV1.GET("/forms/google-drive-files", middleware.RequireRoles("staff", "admin"), h.form.ListGoogleDriveFiles)
		apiV1.POST("/forms/inspect-sheet", middleware.RequireRoles("staff", "admin"), h.form.InspectGoogleSheet)
		apiV1.POST("/forms/:id/sync", middleware.RequireRoles("staff", "admin"), h.form.SyncForm)
		apiV1.GET("/forms/columns", middleware.RequireRoles("viewer", "staff", "admin"), h.form.ListColumns)
		apiV1.PATCH("/forms/columns/:id/mapping", middleware.RequireRoles("staff", "admin"), h.form.UpdateColumnMapping)
		apiV1.POST("/forms/columns/batch-mapping", middleware.RequireRoles("staff", "admin"), h.form.BatchMapping)

		// 6. 搭乘月曆、搭乘紀錄更正、異常搭乘與未回報清單
		apiV1.GET("/rides/calendar", middleware.RequireRoles("viewer", "staff", "admin"), h.ride.GetCalendar)
		apiV1.GET("/rides/issues", middleware.RequireRoles("viewer", "staff", "admin"), h.ride.ListIssues)
		apiV1.GET("/rides/missing", middleware.RequireRoles("viewer", "staff", "admin"), h.task.GetMissingReports)
		apiV1.GET("/rides/:id", middleware.RequireRoles("viewer", "staff", "admin"), h.ride.GetRecord)
		apiV1.PATCH("/rides/:id", middleware.RequireRoles("staff", "admin"), h.ride.Correct)
		apiV1.POST("/rides/manual-report", middleware.RequireRoles("staff", "admin"), h.ride.ManualReport)
		apiV1.POST("/rides/:id/resolve-conflict", middleware.RequireRoles("staff", "admin"), h.ride.ResolveConflict)

		// 7. 匯出前置檢核與工作管理 (B4.3)
		apiV1.GET("/exports/precheck", middleware.RequireRoles("staff", "admin"), h.export.Precheck)
		apiV1.POST("/exports/precheck", middleware.RequireRoles("staff", "admin"), h.export.Precheck)
		apiV1.GET("/exports", middleware.RequireRoles("viewer", "staff", "admin"), h.export.List)
		apiV1.POST("/exports", middleware.RequireRoles("staff", "admin"), h.export.Create)
		apiV1.GET("/exports/:id", middleware.RequireRoles("viewer", "staff", "admin"), h.export.Get)
		apiV1.GET("/exports/:id/download", middleware.RequireRoles("viewer", "staff", "admin"), h.export.Download)

		// 8. 國定假日與行事曆管理 (B5.1)
		apiV1.GET("/holidays", middleware.RequireRoles("viewer", "staff", "admin"), h.holiday.List)
		apiV1.POST("/holidays", middleware.RequireRoles("staff", "admin"), h.holiday.Create)
		apiV1.POST("/holidays/import", middleware.RequireRoles("staff", "admin"), h.holiday.Import)
		apiV1.DELETE("/holidays/:date", middleware.RequireRoles("admin"), h.holiday.Delete)

		// 9. 通知收件人管理與通知留痕 (B5.2b)
		apiV1.GET("/settings/notification-recipients", middleware.RequireRoles("viewer", "staff", "admin"), h.notification.ListRecipients)
		apiV1.POST("/settings/notification-recipients", middleware.RequireRoles("admin"), h.notification.CreateRecipient)
		apiV1.PATCH("/settings/notification-recipients/:id", middleware.RequireRoles("admin"), h.notification.UpdateRecipient)
		apiV1.DELETE("/settings/notification-recipients/:id", middleware.RequireRoles("admin"), h.notification.DeleteRecipient)
		apiV1.GET("/notifications/logs", middleware.RequireRoles("viewer", "staff", "admin"), h.notification.ListLogs)

		// 10. 營運報表 - 車輛趟數表 (B5.4) 與 新竹接送時刻表 (B6.1)
		apiV1.GET("/reports/trip-summary", middleware.RequireRoles("viewer", "staff", "admin"), h.report.GetTripSummary)
		apiV1.GET("/reports/trip-summary/export", middleware.RequireRoles("viewer", "staff", "admin"), h.report.ExportTripSummaryExcel)
		apiV1.GET("/reports/hsinchu-schedule", middleware.RequireRoles("viewer", "staff", "admin"), h.report.GetHsinchuSchedule)
		apiV1.GET("/reports/hsinchu-schedule/export", middleware.RequireRoles("viewer", "staff", "admin"), h.report.ExportHsinchuScheduleExcel)

		// 11. 車輛維修保養管理與空白表下載 (B6.2)
		apiV1.GET("/vehicles/maintenance", middleware.RequireRoles("viewer", "staff", "admin"), h.maintenance.List)
		apiV1.POST("/vehicles/maintenance", middleware.RequireRoles("staff", "admin"), h.maintenance.Create)
		apiV1.PATCH("/vehicles/maintenance/:id", middleware.RequireRoles("staff", "admin"), h.maintenance.Update)
		apiV1.DELETE("/vehicles/maintenance/:id", middleware.RequireRoles("staff", "admin"), h.maintenance.Delete)
		apiV1.GET("/vehicles/maintenance/blank-template", middleware.RequireRoles("viewer", "staff", "admin"), h.maintenance.DownloadBlankTemplate)

		// 12. 司機出勤與請假登錄 (B6.3)
		apiV1.GET("/attendance", middleware.RequireRoles("viewer", "staff", "admin"), h.attendance.GetMonthAttendance)
		apiV1.POST("/attendance", middleware.RequireRoles("staff", "admin"), h.attendance.Upsert)

		// 13. 車輛油資管理 (B6.3)
		apiV1.GET("/fuel-logs", middleware.RequireRoles("viewer", "staff", "admin"), h.fuel.List)
		apiV1.POST("/fuel-logs", middleware.RequireRoles("staff", "admin"), h.fuel.Create)
		apiV1.PATCH("/fuel-logs/:id", middleware.RequireRoles("staff", "admin"), h.fuel.Update)
		apiV1.DELETE("/fuel-logs/:id", middleware.RequireRoles("staff", "admin"), h.fuel.Delete)

		// 14. 視覺化儀表板指標 (B6.4)
		apiV1.GET("/dashboard/metrics", middleware.RequireRoles("viewer", "staff", "admin"), h.dashboard.GetMetrics)
		apiV1.GET("/dashboard/stats", middleware.RequireRoles("viewer", "staff", "admin"), h.dashboard.GetStats)

		// 15. 稽核紀錄查詢 (B5.5 - 僅限 admin)
		apiV1.GET("/audit", middleware.RequireRoles("admin"), h.audit.List)

		// 16. 排程與維運任務端點 (B5.2 / B5.3)
		apiV1.POST("/tasks/check-missing-reports", middleware.RequireRoles("staff", "admin"), h.task.CheckMissingReports)
		apiV1.POST("/tasks/month-end-reminder", middleware.RequireRoles("staff", "admin"), h.task.MonthEndReminder)
	}

	return r
}
