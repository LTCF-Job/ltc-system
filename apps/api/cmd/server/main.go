package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"ltc-system/apps/api/internal/config"
	"ltc-system/apps/api/internal/handler"
	"ltc-system/apps/api/internal/middleware"
	"ltc-system/apps/api/internal/repository"
	"ltc-system/apps/api/internal/service"
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
	caseRepo := repository.NewCaseRepository(pool)
	siteRepo := repository.NewSiteRepository(pool)
	vehicleRepo := repository.NewVehicleRepository(pool)
	driverRepo := repository.NewDriverRepository(pool)
	formRepo := repository.NewFormRepository(pool)
	holidayRepo := repository.NewHolidayRepository(pool)
	notificationRepo := repository.NewNotificationRepository(pool)
	auditRepo := repository.NewAuditRepository(pool)


	// 初始化 Services
	masterSvc := service.NewMasterService(cfg, pool, caseRepo, siteRepo, vehicleRepo, driverRepo)
	importSvc := service.NewImportService(masterSvc, siteRepo, vehicleRepo, driverRepo, caseRepo)
	rideSvc := service.NewRideService(pool, formRepo, driverRepo, caseRepo, vehicleRepo)
	precheckSvc := service.NewPrecheckService(pool)
	notificationSvc := service.NewNotificationService(notificationRepo, auditRepo, nil)
	holidaySvc := service.NewHolidayService(holidayRepo, auditRepo)
	reportSvc := service.NewReportService(pool)
	auditSvc := service.NewAuditService(auditRepo)

	taskSvc := service.NewTaskService(pool, caseRepo, holidayRepo, notificationSvc)

	// 初始化 Handlers
	caseH := handler.NewCaseHandler(caseRepo, masterSvc, importSvc)
	siteH := handler.NewSiteHandler(siteRepo)
	vehicleH := handler.NewVehicleHandler(vehicleRepo)
	driverH := handler.NewDriverHandler(cfg, driverRepo)
	rideH := handler.NewRideHandler(rideSvc)
	exportH := handler.NewExportHandler(precheckSvc)
	notificationH := handler.NewNotificationHandler(notificationSvc)
	holidayH := handler.NewHolidayHandler(holidaySvc)
	reportH := handler.NewReportHandler(reportSvc)
	auditH := handler.NewAuditHandler(auditSvc)
	taskH := handler.NewTaskHandler(taskSvc)

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.SlogLoggerMiddleware())

	// CORS 設定
	corsConfig := cors.DefaultConfig()
	corsConfig.AllowAllOrigins = true
	corsConfig.AllowHeaders = []string{"Origin", "Content-Length", "Content-Type", "Authorization", "X-Ingest-Token", "X-Mock-Role", "X-Mock-User-ID"}
	corsConfig.AllowMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}
	r.Use(cors.New(corsConfig))

	// 健康檢查端點 (健康檢查不走 JWT)
	r.GET("/healthz", func(c *gin.Context) {
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
		// 1. 個案主檔
		apiV1.GET("/cases", middleware.RequireRoles("viewer", "staff", "admin"), caseH.List)
		apiV1.POST("/cases", middleware.RequireRoles("staff", "admin"), caseH.Create)
		apiV1.POST("/cases/:id/reveal", middleware.RequireRoles("staff", "admin"), caseH.Reveal)
		apiV1.POST("/cases/schedules", middleware.RequireRoles("staff", "admin"), caseH.CreateSchedule)
		apiV1.POST("/cases/import", middleware.RequireRoles("staff", "admin"), caseH.ImportExcel)

		// 2. 據點主檔
		apiV1.GET("/sites", middleware.RequireRoles("viewer", "staff", "admin"), siteH.List)
		apiV1.POST("/sites", middleware.RequireRoles("staff", "admin"), siteH.Create)

		// 3. 車輛主檔
		apiV1.GET("/vehicles", middleware.RequireRoles("viewer", "staff", "admin"), vehicleH.List)

		// 4. 司機主檔
		apiV1.GET("/drivers", middleware.RequireRoles("viewer", "staff", "admin"), driverH.List)
		apiV1.POST("/drivers", middleware.RequireRoles("staff", "admin"), driverH.Create)

		// 5. 搭乘紀錄更正與未回報清單
		apiV1.PATCH("/rides/:id", middleware.RequireRoles("staff", "admin"), rideH.Correct)
		apiV1.GET("/rides/missing", middleware.RequireRoles("viewer", "staff", "admin"), taskH.GetMissingReports)

		// 6. 匯出前置檢核
		apiV1.GET("/exports/precheck", middleware.RequireRoles("staff", "admin"), exportH.Precheck)

		// 7. 國定假日與行事曆管理 (B5.1)
		apiV1.GET("/holidays", middleware.RequireRoles("viewer", "staff", "admin"), holidayH.List)
		apiV1.POST("/holidays", middleware.RequireRoles("staff", "admin"), holidayH.Create)
		apiV1.POST("/holidays/import", middleware.RequireRoles("staff", "admin"), holidayH.Import)
		apiV1.DELETE("/holidays/:date", middleware.RequireRoles("admin"), holidayH.Delete)

		// 8. 通知收件人管理與通知留痕 (B5.2b)
		apiV1.GET("/settings/notification-recipients", middleware.RequireRoles("viewer", "staff", "admin"), notificationH.ListRecipients)
		apiV1.POST("/settings/notification-recipients", middleware.RequireRoles("admin"), notificationH.CreateRecipient)
		apiV1.PATCH("/settings/notification-recipients/:id", middleware.RequireRoles("admin"), notificationH.UpdateRecipient)
		apiV1.DELETE("/settings/notification-recipients/:id", middleware.RequireRoles("admin"), notificationH.DeleteRecipient)
		apiV1.GET("/notifications/logs", middleware.RequireRoles("viewer", "staff", "admin"), notificationH.ListLogs)

		// 9. 營運報表 - 車輛趟數表 (B5.4)
		apiV1.GET("/reports/trip-summary", middleware.RequireRoles("viewer", "staff", "admin"), reportH.GetTripSummary)
		apiV1.GET("/reports/trip-summary/export", middleware.RequireRoles("viewer", "staff", "admin"), reportH.ExportTripSummaryExcel)

		// 10. 稽核紀錄查詢 (B5.5 - 僅限 admin)
		apiV1.GET("/audit", middleware.RequireRoles("admin"), auditH.List)

		// 11. 排程與維運任務端點 (B5.2 / B5.3)
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

