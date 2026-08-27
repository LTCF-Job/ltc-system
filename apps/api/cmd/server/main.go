package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"ltc-system/apps/api/internal/adapter"
	"ltc-system/apps/api/internal/adapter/google"
	"ltc-system/apps/api/internal/config"
	"ltc-system/apps/api/internal/handler"
	"ltc-system/apps/api/internal/middleware"
	"ltc-system/apps/api/internal/repository"
	"ltc-system/apps/api/internal/service"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	// 結構化 JSON 日誌初始化
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	cfg, err := config.LoadFromEnv()
	if err != nil {
		slog.Error("Configuration error", slog.String("error", err.Error()))
		os.Exit(1)
	}

	if cfg.AppEnv == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	ctx := context.Background()
	var pool *pgxpool.Pool

	pool, err = pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Warn("Could not create database pool (running in offline mode)", slog.String("error", err.Error()))
	} else {
		defer pool.Close()
		pingCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
		defer cancel()
		if err := pool.Ping(pingCtx); err != nil {
			slog.Warn("Database ping failed (continuing startup)", slog.String("error", err.Error()))
		} else {
			slog.Info("Connected to PostgreSQL database successfully")
		}
	}

	// 初始化 Repositories
	regionRepo := repository.NewRegionRepository(pool)
	caseRepo := repository.NewCaseRepository(pool)
	siteRepo := repository.NewSiteRepository(pool)
	vehicleRepo := repository.NewVehicleRepository(pool)
	driverRepo := repository.NewDriverRepository(pool)
	formRepo := repository.NewFormRepository(pool)
	holidayRepo := repository.NewHolidayRepository(pool)
	notificationRepo := repository.NewNotificationRepository(pool)
	auditRepo := repository.NewAuditRepository(pool)
	maintenanceRepo := repository.NewMaintenanceRepository(pool)
	attendanceRepo := repository.NewAttendanceRepository(pool)
	fuelRepo := repository.NewFuelRepository(pool)
	reportRepo := repository.NewReportRepository(pool)
	dashboardRepo := repository.NewDashboardRepository(pool)
	precheckRepo := repository.NewPrecheckRepository(pool)
	taskRepo := repository.NewTaskRepository(pool)

	// 初始化 Services
	googleCli, err := google.NewClient(ctx, cfg.GoogleSAJSON)
	if err != nil {
		slog.Warn("Failed to initialize Google API client (falling back to offline mode)", slog.String("error", err.Error()))
	}
	regionSvc := service.NewRegionService(regionRepo, auditRepo)
	masterSvc := service.NewMasterService(cfg, caseRepo, siteRepo, vehicleRepo, driverRepo, auditRepo)
	importSvc := service.NewImportService(masterSvc, siteRepo, vehicleRepo, driverRepo, caseRepo)
	rideSvc := service.NewRideService(formRepo, driverRepo, caseRepo, vehicleRepo, auditRepo)
	formSvc := service.NewFormService(formRepo, googleCli)
	precheckSvc := service.NewPrecheckService(precheckRepo)
	notificationSvc := service.NewNotificationService(notificationRepo, auditRepo, nil)
	holidayProvider := service.GovernmentHolidayProvider(&adapter.GovernmentHolidayHTTPClient{
		Endpoint: adapter.GovernmentHolidayCSVEndpoint,
		Client:   &http.Client{Timeout: cfg.GovernmentHolidayAPITimeout},
	})
	holidaySvc := service.NewHolidaySyncService(holidayRepo, auditRepo, holidayProvider)
	reportSvc := service.NewReportService(reportRepo)
	auditSvc := service.NewAuditService(auditRepo)
	maintenanceSvc := service.NewMaintenanceService(maintenanceRepo, vehicleRepo, auditRepo)
	attendanceSvc := service.NewAttendanceService(attendanceRepo, driverRepo, auditRepo)
	fuelSvc := service.NewFuelService(fuelRepo, auditRepo)
	dashboardSvc := service.NewDashboardService(dashboardRepo)
	taskSvc := service.NewTaskService(taskRepo, caseRepo, holidayRepo, notificationSvc)

	// 初始化 Handlers
	regionH := handler.NewRegionHandler(regionSvc)
	caseH := handler.NewCaseHandler(caseRepo, masterSvc, importSvc)
	siteH := handler.NewSiteHandler(siteRepo)
	vehicleH := handler.NewVehicleHandler(vehicleRepo)
	driverH := handler.NewDriverHandler(cfg, driverRepo)
	rideH := handler.NewRideHandler(rideSvc)
	exportH := handler.NewExportHandler(precheckSvc, reportSvc)
	notificationH := handler.NewNotificationHandler(notificationSvc)
	holidayH := handler.NewHolidayHandler(holidaySvc)
	reportH := handler.NewReportHandler(reportSvc)
	auditH := handler.NewAuditHandler(auditSvc)
	taskH := handler.NewTaskHandler(taskSvc)
	maintenanceH := handler.NewMaintenanceHandler(maintenanceSvc)
	attendanceH := handler.NewAttendanceHandler(attendanceSvc)
	fuelH := handler.NewFuelHandler(fuelSvc)
	dashboardH := handler.NewDashboardHandler(dashboardSvc)
	formH := handler.NewFormHandler(formSvc)

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
	r.POST("/api/v1/ingest/google-form", rideH.IngestWebhook)

	// 需要 JWT 認證之 API 群組
	apiV1 := r.Group("/api/v1")
	apiV1.Use(middleware.AuthMiddleware(cfg))
	{
		// 0. 區域主檔
		apiV1.GET("/regions", middleware.RequireRoles("viewer", "staff", "admin"), regionH.List)
		apiV1.GET("/regions/:id", middleware.RequireRoles("viewer", "staff", "admin"), regionH.Get)
		apiV1.POST("/regions", middleware.RequireRoles("staff", "admin"), regionH.Create)
		apiV1.PATCH("/regions/:id", middleware.RequireRoles("staff", "admin"), regionH.Update)
		apiV1.DELETE("/regions/:id", middleware.RequireRoles("admin"), regionH.Delete)

		// 1. 個案主檔與排班

		apiV1.GET("/cases", middleware.RequireRoles("viewer", "staff", "admin"), caseH.List)
		apiV1.POST("/cases", middleware.RequireRoles("staff", "admin"), caseH.Create)
		apiV1.GET("/cases/template", middleware.RequireRoles("viewer", "staff", "admin"), caseH.DownloadTemplate)
		apiV1.GET("/cases/:id", middleware.RequireRoles("viewer", "staff", "admin"), caseH.Get)
		apiV1.PATCH("/cases/:id", middleware.RequireRoles("staff", "admin"), caseH.Update)
		apiV1.POST("/cases/:id/reveal", middleware.RequireRoles("staff", "admin"), caseH.Reveal)
		apiV1.GET("/cases/:id/schedule", middleware.RequireRoles("viewer", "staff", "admin"), caseH.GetSchedule)
		apiV1.PUT("/cases/:id/schedule", middleware.RequireRoles("staff", "admin"), caseH.SaveSchedule)
		apiV1.POST("/cases/schedules", middleware.RequireRoles("staff", "admin"), caseH.CreateSchedule)
		apiV1.POST("/cases/import", middleware.RequireRoles("staff", "admin"), caseH.ImportExcel)
		apiV1.POST("/masters/import", middleware.RequireRoles("staff", "admin"), caseH.ImportExcel)

		// 2. 據點主檔
		apiV1.GET("/sites", middleware.RequireRoles("viewer", "staff", "admin"), siteH.List)
		apiV1.POST("/sites", middleware.RequireRoles("staff", "admin"), siteH.Create)
		apiV1.PATCH("/sites/:id", middleware.RequireRoles("staff", "admin"), siteH.Update)
		apiV1.DELETE("/sites/:id", middleware.RequireRoles("admin"), siteH.Delete)

		// 3. 車輛主檔
		apiV1.GET("/vehicles", middleware.RequireRoles("viewer", "staff", "admin"), vehicleH.List)
		apiV1.POST("/vehicles", middleware.RequireRoles("staff", "admin"), vehicleH.Create)
		apiV1.PATCH("/vehicles/:id", middleware.RequireRoles("staff", "admin"), vehicleH.Update)

		// 4. 司機主檔
		apiV1.GET("/drivers", middleware.RequireRoles("viewer", "staff", "admin"), driverH.List)
		apiV1.POST("/drivers", middleware.RequireRoles("staff", "admin"), driverH.Create)
		apiV1.PATCH("/drivers/:id", middleware.RequireRoles("staff", "admin"), driverH.Update)
		apiV1.POST("/drivers/:id/reveal", middleware.RequireRoles("staff", "admin"), driverH.Reveal)
		apiV1.POST("/drivers/:id/assignments", middleware.RequireRoles("staff", "admin"), driverH.AssignVehicle)

		// 5. 表單管理與欄位對應
		apiV1.GET("/forms", middleware.RequireRoles("viewer", "staff", "admin"), formH.ListForms)
		apiV1.POST("/forms", middleware.RequireRoles("staff", "admin"), formH.CreateFormAssociation)
		apiV1.DELETE("/forms/:id", middleware.RequireRoles("staff", "admin"), formH.DeleteFormAssociation)
		apiV1.GET("/forms/google-drive-files", middleware.RequireRoles("staff", "admin"), formH.ListGoogleDriveFiles)
		apiV1.POST("/forms/inspect-sheet", middleware.RequireRoles("staff", "admin"), formH.InspectGoogleSheet)
		apiV1.POST("/forms/:id/sync", middleware.RequireRoles("staff", "admin"), formH.SyncForm)
		apiV1.GET("/forms/columns", middleware.RequireRoles("viewer", "staff", "admin"), formH.ListColumns)
		apiV1.PATCH("/forms/columns/:id/mapping", middleware.RequireRoles("staff", "admin"), formH.UpdateColumnMapping)
		apiV1.POST("/forms/columns/batch-mapping", middleware.RequireRoles("staff", "admin"), formH.BatchMapping)

		// 6. 搭乘月曆、搭乘紀錄更正、異常搭乘與未回報清單
		apiV1.GET("/rides/calendar", middleware.RequireRoles("viewer", "staff", "admin"), rideH.GetCalendar)
		apiV1.GET("/rides/issues", middleware.RequireRoles("viewer", "staff", "admin"), rideH.ListIssues)
		apiV1.GET("/rides/missing", middleware.RequireRoles("viewer", "staff", "admin"), taskH.GetMissingReports)
		apiV1.GET("/rides/:id", middleware.RequireRoles("viewer", "staff", "admin"), rideH.GetRecord)
		apiV1.PATCH("/rides/:id", middleware.RequireRoles("staff", "admin"), rideH.Correct)
		apiV1.POST("/rides/manual-report", middleware.RequireRoles("staff", "admin"), rideH.ManualReport)
		apiV1.POST("/rides/:id/resolve-conflict", middleware.RequireRoles("staff", "admin"), rideH.ResolveConflict)

		// 7. 匯出前置檢核與工作管理 (B4.3)
		apiV1.GET("/exports/precheck", middleware.RequireRoles("staff", "admin"), exportH.Precheck)
		apiV1.POST("/exports/precheck", middleware.RequireRoles("staff", "admin"), exportH.Precheck)
		apiV1.GET("/exports", middleware.RequireRoles("viewer", "staff", "admin"), exportH.List)
		apiV1.POST("/exports", middleware.RequireRoles("staff", "admin"), exportH.Create)
		apiV1.GET("/exports/:id", middleware.RequireRoles("viewer", "staff", "admin"), exportH.Get)
		apiV1.GET("/exports/:id/download", middleware.RequireRoles("viewer", "staff", "admin"), exportH.Download)

		// 8. 國定假日與行事曆管理 (B5.1)
		apiV1.GET("/holidays", middleware.RequireRoles("viewer", "staff", "admin"), holidayH.List)
		apiV1.POST("/holidays", middleware.RequireRoles("staff", "admin"), holidayH.Create)
		apiV1.POST("/holidays/import", middleware.RequireRoles("staff", "admin"), holidayH.Import)
		apiV1.DELETE("/holidays/:date", middleware.RequireRoles("admin"), holidayH.Delete)

		// 9. 通知收件人管理與通知留痕 (B5.2b)
		apiV1.GET("/settings/notification-recipients", middleware.RequireRoles("viewer", "staff", "admin"), notificationH.ListRecipients)
		apiV1.POST("/settings/notification-recipients", middleware.RequireRoles("admin"), notificationH.CreateRecipient)
		apiV1.PATCH("/settings/notification-recipients/:id", middleware.RequireRoles("admin"), notificationH.UpdateRecipient)
		apiV1.DELETE("/settings/notification-recipients/:id", middleware.RequireRoles("admin"), notificationH.DeleteRecipient)
		apiV1.GET("/notifications/logs", middleware.RequireRoles("viewer", "staff", "admin"), notificationH.ListLogs)

		// 10. 營運報表 - 車輛趟數表 (B5.4) 與 新竹接送時刻表 (B6.1)
		apiV1.GET("/reports/trip-summary", middleware.RequireRoles("viewer", "staff", "admin"), reportH.GetTripSummary)
		apiV1.GET("/reports/trip-summary/export", middleware.RequireRoles("viewer", "staff", "admin"), reportH.ExportTripSummaryExcel)
		apiV1.GET("/reports/hsinchu-schedule", middleware.RequireRoles("viewer", "staff", "admin"), reportH.GetHsinchuSchedule)
		apiV1.GET("/reports/hsinchu-schedule/export", middleware.RequireRoles("viewer", "staff", "admin"), reportH.ExportHsinchuScheduleExcel)

		// 11. 車輛維修保養管理與空白表下載 (B6.2)
		apiV1.GET("/vehicles/maintenance", middleware.RequireRoles("viewer", "staff", "admin"), maintenanceH.List)
		apiV1.POST("/vehicles/maintenance", middleware.RequireRoles("staff", "admin"), maintenanceH.Create)
		apiV1.PATCH("/vehicles/maintenance/:id", middleware.RequireRoles("staff", "admin"), maintenanceH.Update)
		apiV1.DELETE("/vehicles/maintenance/:id", middleware.RequireRoles("staff", "admin"), maintenanceH.Delete)
		apiV1.GET("/vehicles/maintenance/blank-template", middleware.RequireRoles("viewer", "staff", "admin"), maintenanceH.DownloadBlankTemplate)

		// 12. 司機出勤與請假登錄 (B6.3)
		apiV1.GET("/attendance", middleware.RequireRoles("viewer", "staff", "admin"), attendanceH.GetMonthAttendance)
		apiV1.POST("/attendance", middleware.RequireRoles("staff", "admin"), attendanceH.Upsert)

		// 13. 車輛油資管理 (B6.3)
		apiV1.GET("/fuel-logs", middleware.RequireRoles("viewer", "staff", "admin"), fuelH.List)
		apiV1.POST("/fuel-logs", middleware.RequireRoles("staff", "admin"), fuelH.Create)
		apiV1.PATCH("/fuel-logs/:id", middleware.RequireRoles("staff", "admin"), fuelH.Update)
		apiV1.DELETE("/fuel-logs/:id", middleware.RequireRoles("staff", "admin"), fuelH.Delete)

		// 14. 視覺化儀表板指標 (B6.4)
		apiV1.GET("/dashboard/metrics", middleware.RequireRoles("viewer", "staff", "admin"), dashboardH.GetMetrics)
		apiV1.GET("/dashboard/stats", middleware.RequireRoles("viewer", "staff", "admin"), dashboardH.GetStats)

		// 15. 稽核紀錄查詢 (B5.5 - 僅限 admin)
		apiV1.GET("/audit", middleware.RequireRoles("admin"), auditH.List)

		// 16. 排程與維運任務端點 (B5.2 / B5.3)
		apiV1.POST("/tasks/check-missing-reports", middleware.RequireRoles("staff", "admin"), taskH.CheckMissingReports)
		apiV1.POST("/tasks/month-end-reminder", middleware.RequireRoles("staff", "admin"), taskH.MonthEndReminder)
	}

	addr := fmt.Sprintf(":%s", cfg.Port)
	slog.Info("Starting LTC API Server", slog.String("addr", addr), slog.String("env", cfg.AppEnv))
	if err := r.Run(addr); err != nil {
		slog.Error("Server terminated unexpectedly", slog.String("error", err.Error()))
		os.Exit(1)
	}
}
