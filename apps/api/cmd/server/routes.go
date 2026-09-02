package main

import (
	"net/http"
	"strings"
	"time"

	audittransport "ltc-system/apps/api/internal/modules/audit/transport"
	caregivertransport "ltc-system/apps/api/internal/modules/caregiver/transport"
	importtransport "ltc-system/apps/api/internal/modules/caseimport/transport"
	casetransport "ltc-system/apps/api/internal/modules/casemgmt/transport"
	demoapp "ltc-system/apps/api/internal/modules/demo/app"
	demotransport "ltc-system/apps/api/internal/modules/demo/transport"
	drtransport "ltc-system/apps/api/internal/modules/driverreport/transport"
	holidaytransport "ltc-system/apps/api/internal/modules/holiday/transport"
	identitytransport "ltc-system/apps/api/internal/modules/identity/transport"
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
	role         *identitytransport.RoleHandler
	identity     *identitytransport.IdentityHandler
	demo         *demotransport.ResetHandler
}

// newRouter 組裝 gin engine：全域 middleware、CORS、健康檢查與 v1 路由表。
func newRouter(cfg *config.Config, pool *pgxpool.Pool, h handlers, demoGuard *demoapp.ConcurrencyGuard, perm auth.PermissionResolver, customPerm auth.CustomPermissionResolver) *gin.Engine {
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

	// demo/reset 本身需要獨佔鎖，必須在 ConcurrencyGuardMiddleware 掛載前註冊，
	// 否則這個請求會先卡在自己持有的共享鎖，永遠等不到獨佔鎖（自我死結）。
	if h.demo != nil {
		apiV1.POST("/demo/reset", auth.RequireRoles("viewer", "staff", "admin"), h.demo.Reset)
	}
	if cfg.DataPlane == "demo" && demoGuard != nil {
		apiV1.Use(demotransport.ConcurrencyGuardMiddleware(demoGuard))
	}
	{
		// 以下路由的授權改依 perm.RequirePermission(module, action) 查角色的模組權限矩陣，
		// 取代逐路由寫死角色字面值；自訂角色在「角色身分管理」頁調整矩陣後，API 存取範圍會
		// 跟著變（見 docs/decisions/role-permission-api-authorization.md）。使用者／角色管理、
		// 排程任務與國定假日（不在模組權限矩陣涵蓋範圍內）維持 RequireRoles 粗粒度白名單。

		// 0. 區域主檔
		apiV1.GET("/regions", auth.RequirePermission(perm, customPerm, "masters_regions", "view"), h.region.List)
		apiV1.GET("/regions/:id", auth.RequirePermission(perm, customPerm, "masters_regions", "view"), h.region.Get)
		apiV1.POST("/regions", auth.RequirePermission(perm, customPerm, "masters_regions", "edit"), h.region.Create)
		apiV1.PATCH("/regions/:id", auth.RequirePermission(perm, customPerm, "masters_regions", "edit"), h.region.Update)
		apiV1.DELETE("/regions/:id", auth.RequirePermission(perm, customPerm, "masters_regions", "delete"), h.region.Delete)

		// 1. 個案主檔與排班

		apiV1.GET("/cases", auth.RequirePermission(perm, customPerm, "masters_cases", "view"), h.kase.List)
		apiV1.POST("/cases", auth.RequirePermission(perm, customPerm, "masters_cases", "edit"), h.kase.Create)
		apiV1.GET("/cases/template", auth.RequirePermission(perm, customPerm, "masters_cases", "view"), h.caseImport.DownloadTemplate)
		apiV1.GET("/cases/export", auth.RequirePermission(perm, customPerm, "masters_cases", "view"), h.kase.ExportProfileWorkbook)
		apiV1.GET("/cases/:id", auth.RequirePermission(perm, customPerm, "masters_cases", "view"), h.kase.Get)
		apiV1.PATCH("/cases/:id", auth.RequirePermission(perm, customPerm, "masters_cases", "edit"), h.kase.Update)
		apiV1.DELETE("/cases/:id", auth.RequirePermission(perm, customPerm, "masters_cases", "delete"), h.kase.Delete)
		apiV1.PUT("/cases/:id/transport-preference", auth.RequirePermission(perm, customPerm, "masters_cases", "edit"), h.kase.UpdateTransportPreference)
		apiV1.POST("/cases/:id/reveal", auth.RequirePermission(perm, customPerm, "masters_cases", "edit"), h.kase.Reveal)
		apiV1.GET("/cases/:id/schedule", auth.RequirePermission(perm, customPerm, "masters_cases", "view"), h.kase.GetSchedule)
		apiV1.PUT("/cases/:id/schedule", auth.RequirePermission(perm, customPerm, "masters_cases", "edit"), h.kase.SaveSchedule)
		apiV1.POST("/cases/schedules", auth.RequirePermission(perm, customPerm, "masters_cases", "edit"), h.kase.CreateSchedule)
		apiV1.POST("/cases/import", auth.RequirePermission(perm, customPerm, "masters_cases", "edit"), h.caseImport.ImportExcel)
		apiV1.POST("/masters/import", auth.RequirePermission(perm, customPerm, "masters_cases", "edit"), h.caseImport.ImportExcel)

		// 2. 單位主檔
		apiV1.GET("/sites", auth.RequirePermission(perm, customPerm, "masters_sites", "view"), h.site.List)
		apiV1.POST("/sites", auth.RequirePermission(perm, customPerm, "masters_sites", "edit"), h.site.Create)
		apiV1.PATCH("/sites/:id", auth.RequirePermission(perm, customPerm, "masters_sites", "edit"), h.site.Update)
		apiV1.DELETE("/sites/:id", auth.RequirePermission(perm, customPerm, "masters_sites", "delete"), h.site.Delete)

		// 3. 車輛主檔
		apiV1.GET("/vehicles", auth.RequirePermission(perm, customPerm, "masters_vehicles", "view"), h.vehicle.List)
		apiV1.POST("/vehicles", auth.RequirePermission(perm, customPerm, "masters_vehicles", "edit"), h.vehicle.Create)
		apiV1.PATCH("/vehicles/:id", auth.RequirePermission(perm, customPerm, "masters_vehicles", "edit"), h.vehicle.Update)
		apiV1.DELETE("/vehicles/:id", auth.RequirePermission(perm, customPerm, "masters_vehicles", "delete"), h.vehicle.Delete)
		apiV1.PUT("/vehicles/:id/drivers", auth.RequirePermission(perm, customPerm, "masters_vehicles", "edit"), h.vehicle.SetDrivers)

		// 4. 司機主檔
		apiV1.GET("/drivers", auth.RequirePermission(perm, customPerm, "masters_drivers", "view"), h.driver.List)
		apiV1.POST("/drivers", auth.RequirePermission(perm, customPerm, "masters_drivers", "edit"), h.driver.Create)
		apiV1.PATCH("/drivers/:id", auth.RequirePermission(perm, customPerm, "masters_drivers", "edit"), h.driver.Update)
		apiV1.DELETE("/drivers/:id", auth.RequirePermission(perm, customPerm, "masters_drivers", "delete"), h.driver.Delete)
		apiV1.POST("/drivers/:id/reveal", auth.RequirePermission(perm, customPerm, "masters_drivers", "edit"), h.driver.Reveal)
		apiV1.POST("/drivers/:id/assignments", auth.RequirePermission(perm, customPerm, "masters_drivers", "edit"), h.driver.AssignVehicle)

		// 5. 司機接送匯報表與欄位對應
		apiV1.GET("/driver-reports", auth.RequirePermission(perm, customPerm, "driver_reports", "view"), h.driverReport.ListForms)
		apiV1.POST("/driver-reports", auth.RequirePermission(perm, customPerm, "driver_reports", "edit"), h.driverReport.CreateForm)
		apiV1.GET("/driver-reports/imported-months", auth.RequirePermission(perm, customPerm, "driver_reports", "view"), h.driverReport.ListImportedMonths)
		apiV1.GET("/driver-reports/columns", auth.RequirePermission(perm, customPerm, "driver_report_mappings", "view"), h.driverReport.ListColumns)
		apiV1.PATCH("/driver-reports/columns/:id/mapping", auth.RequirePermission(perm, customPerm, "driver_report_mappings", "edit"), h.driverReport.UpdateColumnMapping)
		apiV1.POST("/driver-reports/columns/batch-mapping", auth.RequirePermission(perm, customPerm, "driver_report_mappings", "edit"), h.driverReport.BatchMapping)
		apiV1.DELETE("/driver-reports/:id", auth.RequirePermission(perm, customPerm, "driver_reports", "delete"), h.driverReport.DeleteForm)
		apiV1.GET("/driver-reports/:id/template", auth.RequirePermission(perm, customPerm, "driver_reports", "edit"), h.driverReport.DownloadTemplate)
		apiV1.POST("/driver-reports/:id/import", auth.RequirePermission(perm, customPerm, "driver_reports", "edit"), h.driverReport.ImportExcel)

		// 6. 搭乘月曆、搭乘紀錄更正、異常搭乘與未回報清單
		apiV1.GET("/rides/calendar", auth.RequirePermission(perm, customPerm, "rides_calendar", "view"), h.ride.GetCalendar)
		apiV1.GET("/rides/issues", auth.RequirePermission(perm, customPerm, "rides_issues", "view"), h.ride.ListIssues)
		apiV1.GET("/rides/missing", auth.RequirePermission(perm, customPerm, "rides_missing", "view"), h.task.GetMissingReports)
		apiV1.GET("/rides/:id", auth.RequirePermission(perm, customPerm, "rides_issues", "view"), h.ride.GetRecord)
		apiV1.PATCH("/rides/:id", auth.RequirePermission(perm, customPerm, "rides_issues", "edit"), h.ride.Correct)
		apiV1.POST("/rides/manual-report", auth.RequirePermission(perm, customPerm, "rides_issues", "edit"), h.ride.ManualReport)
		apiV1.POST("/rides/:id/resolve-conflict", auth.RequirePermission(perm, customPerm, "rides_issues", "edit"), h.ride.ResolveConflict)

		// 7. 匯出前置檢核與工作管理 (B4.3)
		apiV1.GET("/exports/precheck", auth.RequirePermission(perm, customPerm, "exports", "edit"), h.export.Precheck)
		apiV1.POST("/exports/precheck", auth.RequirePermission(perm, customPerm, "exports", "edit"), h.export.Precheck)
		apiV1.GET("/exports", auth.RequirePermission(perm, customPerm, "exports", "view"), h.export.List)
		apiV1.POST("/exports", auth.RequirePermission(perm, customPerm, "exports", "edit"), h.export.Create)
		apiV1.GET("/exports/:id", auth.RequirePermission(perm, customPerm, "exports", "view"), h.export.Get)
		apiV1.GET("/exports/:id/download", auth.RequirePermission(perm, customPerm, "exports", "view"), h.export.Download)
		apiV1.GET("/exports/:id/files/:caseId/download", auth.RequirePermission(perm, customPerm, "exports", "view"), h.export.DownloadCaseFile)

		// 8. 國定假日與行事曆管理 (B5.1)——不在模組權限矩陣涵蓋範圍內，維持角色白名單
		apiV1.GET("/holidays", auth.RequireRoles("viewer", "staff", "admin"), h.holiday.List)
		apiV1.POST("/holidays", auth.RequireRoles("staff", "admin"), h.holiday.Create)
		apiV1.POST("/holidays/import", auth.RequireRoles("staff", "admin"), h.holiday.Import)
		apiV1.DELETE("/holidays/:date", auth.RequireRoles("admin"), h.holiday.Delete)

		// 9. 通知收件人管理與通知留痕 (B5.2b)
		apiV1.GET("/settings/notification-recipients", auth.RequirePermission(perm, customPerm, "settings_notifications", "view"), h.notification.ListRecipients)
		apiV1.POST("/settings/notification-recipients", auth.RequirePermission(perm, customPerm, "settings_notifications", "edit"), h.notification.CreateRecipient)
		apiV1.POST("/settings/notification-recipients/batch", auth.RequirePermission(perm, customPerm, "settings_notifications", "edit"), h.notification.BatchCreateRecipients)
		apiV1.POST("/settings/notification-recipients/batch-delete", auth.RequirePermission(perm, customPerm, "settings_notifications", "edit"), h.notification.BatchDeleteRecipients)
		apiV1.PATCH("/settings/notification-recipients/:id", auth.RequirePermission(perm, customPerm, "settings_notifications", "edit"), h.notification.UpdateRecipient)
		apiV1.DELETE("/settings/notification-recipients/:id", auth.RequirePermission(perm, customPerm, "settings_notifications", "delete"), h.notification.DeleteRecipient)
		apiV1.GET("/notifications/logs", auth.RequirePermission(perm, customPerm, "settings_notifications", "view"), h.notification.ListLogs)

		// 10. 營運報表 - 車輛趟數表 (B5.4) 與 新竹接送時刻表 (B6.1)
		apiV1.GET("/reports/trip-summary", auth.RequirePermission(perm, customPerm, "reports_trip_summary", "view"), h.report.GetTripSummary)
		apiV1.GET("/reports/trip-summary/export", auth.RequirePermission(perm, customPerm, "reports_trip_summary", "view"), h.report.ExportTripSummaryExcel)
		apiV1.GET("/reports/hsinchu-schedule", auth.RequirePermission(perm, customPerm, "reports_hsinchu_schedule", "view"), h.report.GetHsinchuSchedule)
		apiV1.GET("/reports/hsinchu-schedule/export", auth.RequirePermission(perm, customPerm, "reports_hsinchu_schedule", "view"), h.report.ExportHsinchuScheduleExcel)

		// 11. 車輛維修保養管理與空白表下載 (B6.2)
		apiV1.GET("/vehicles/maintenance", auth.RequirePermission(perm, customPerm, "vehicles_maintenance", "view"), h.maintenance.List)
		apiV1.POST("/vehicles/maintenance", auth.RequirePermission(perm, customPerm, "vehicles_maintenance", "edit"), h.maintenance.Create)
		apiV1.PATCH("/vehicles/maintenance/:id", auth.RequirePermission(perm, customPerm, "vehicles_maintenance", "edit"), h.maintenance.Update)
		apiV1.DELETE("/vehicles/maintenance/:id", auth.RequirePermission(perm, customPerm, "vehicles_maintenance", "delete"), h.maintenance.Delete)
		apiV1.GET("/vehicles/maintenance/blank-template", auth.RequirePermission(perm, customPerm, "vehicles_maintenance", "view"), h.maintenance.DownloadBlankTemplate)

		// 12. 司機出勤與請假登錄 (B6.3)
		apiV1.GET("/attendance", auth.RequirePermission(perm, customPerm, "attendance_fuel", "view"), h.attendance.GetMonthAttendance)
		apiV1.POST("/attendance", auth.RequirePermission(perm, customPerm, "attendance_fuel", "edit"), h.attendance.Upsert)

		// 13. 車輛油資管理 (B6.3)
		apiV1.GET("/fuel-logs", auth.RequirePermission(perm, customPerm, "attendance_fuel", "view"), h.fuel.List)
		apiV1.POST("/fuel-logs", auth.RequirePermission(perm, customPerm, "attendance_fuel", "edit"), h.fuel.Create)
		apiV1.PATCH("/fuel-logs/:id", auth.RequirePermission(perm, customPerm, "attendance_fuel", "edit"), h.fuel.Update)
		apiV1.DELETE("/fuel-logs/:id", auth.RequirePermission(perm, customPerm, "attendance_fuel", "delete"), h.fuel.Delete)

		// 14. 視覺化儀表板指標 (B6.4)
		apiV1.GET("/dashboard/metrics", auth.RequirePermission(perm, customPerm, "dashboard", "view"), h.dashboard.GetMetrics)
		apiV1.GET("/dashboard/stats", auth.RequirePermission(perm, customPerm, "dashboard", "view"), h.dashboard.GetStats)

		// 15. 稽核紀錄查詢 (B5.5)
		apiV1.GET("/audit", auth.RequirePermission(perm, customPerm, "audit_logs", "view"), h.audit.List)

		// 16. 排程與維運任務端點 (B5.2 / B5.3)——維運觸發端點，不對映任何模組，維持角色白名單
		apiV1.POST("/tasks/check-missing-reports", auth.RequireRoles("staff", "admin"), h.task.CheckMissingReports)
		apiV1.POST("/tasks/month-end-reminder", auth.RequireRoles("staff", "admin"), h.task.MonthEndReminder)

		// 17. 照護人員主檔管理
		apiV1.GET("/caregivers", auth.RequirePermission(perm, customPerm, "masters_caregivers", "view"), h.caregiver.List)
		apiV1.POST("/caregivers", auth.RequirePermission(perm, customPerm, "masters_caregivers", "edit"), h.caregiver.Create)
		apiV1.GET("/caregivers/template", auth.RequirePermission(perm, customPerm, "masters_caregivers", "view"), h.caregiver.DownloadTemplate)
		apiV1.POST("/caregivers/import", auth.RequirePermission(perm, customPerm, "masters_caregivers", "edit"), h.caregiver.ImportExcel)
		apiV1.PATCH("/caregivers/:id", auth.RequirePermission(perm, customPerm, "masters_caregivers", "edit"), h.caregiver.Update)
		apiV1.DELETE("/caregivers/:id", auth.RequirePermission(perm, customPerm, "masters_caregivers", "delete"), h.caregiver.Delete)
		apiV1.PUT("/caregivers/:id/site", auth.RequirePermission(perm, customPerm, "masters_caregivers", "edit"), h.caregiver.LinkSite)

		// 18. 角色身分管理 roleH
		apiV1.GET("/roles", auth.RequireRoles("admin"), h.role.ListRoles)
		apiV1.GET("/roles/:id", auth.RequireRoles("admin"), h.role.GetRole)
		apiV1.POST("/roles", auth.RequireRoles("admin"), h.role.CreateRole)
		apiV1.PATCH("/roles/:id", auth.RequireRoles("admin"), h.role.UpdateRole)
		apiV1.DELETE("/roles/:id", auth.RequireRoles("admin"), h.role.DeleteRole)

		// 19. 使用者帳號管理與密碼變更 identityH
		apiV1.GET("/users", auth.RequireRoles("admin"), h.identity.ListUsers)
		apiV1.GET("/users/:id", auth.RequireRoles("admin"), h.identity.GetUser)
		apiV1.POST("/users", auth.RequireRoles("admin"), h.identity.CreateUser)
		apiV1.PATCH("/users/:id", auth.RequireRoles("admin"), h.identity.UpdateUser)
		apiV1.PUT("/users/:id/permissions", auth.RequireRoles("admin"), h.identity.UpdateUserPermissions)
		apiV1.DELETE("/users/:id", auth.RequireRoles("admin"), h.identity.DeleteUser)
		apiV1.POST("/auth/change-password", auth.RequireRoles("viewer", "staff", "admin"), h.identity.ChangeSelfPassword)
	}

	return r
}
