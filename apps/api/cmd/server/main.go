package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"ltc-system/apps/api/internal/adapter"
	"ltc-system/apps/api/internal/adapter/google"
	"ltc-system/apps/api/internal/config"
	"ltc-system/apps/api/internal/handler"
	"ltc-system/apps/api/internal/repository"
	"ltc-system/apps/api/internal/service"

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
	pool, err := connectDatabase(ctx, cfg)
	if err != nil {
		// production 下無資料庫連線代表這台伺服器無法提供任何真實資料，
		// 啟動即失敗比讓所有 repository 帶著 nil pool 悄悄上線更安全。
		// local 下允許離線啟動，方便前端在沒有本機 DB 時開發。
		if cfg.AppEnv == "production" {
			slog.Error("Database connection failed, refusing to start in production", slog.String("error", err.Error()))
			os.Exit(1)
		}
		slog.Warn("Could not connect to database (running in offline mode, local only)", slog.String("error", err.Error()))
	} else {
		defer pool.Close()
		slog.Info("Connected to PostgreSQL database successfully")
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
	siteSvc := service.NewSiteService(siteRepo)
	vehicleSvc := service.NewVehicleService(vehicleRepo)
	driverSvc := service.NewDriverService(driverRepo, cfg)
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
	h := handlers{
		region:       handler.NewRegionHandler(regionSvc),
		kase:         handler.NewCaseHandler(masterSvc, importSvc),
		site:         handler.NewSiteHandler(siteSvc),
		vehicle:      handler.NewVehicleHandler(vehicleSvc),
		driver:       handler.NewDriverHandler(driverSvc),
		ride:         handler.NewRideHandler(rideSvc),
		export:       handler.NewExportHandler(precheckSvc, reportSvc),
		notification: handler.NewNotificationHandler(notificationSvc),
		holiday:      handler.NewHolidayHandler(holidaySvc),
		report:       handler.NewReportHandler(reportSvc),
		audit:        handler.NewAuditHandler(auditSvc),
		task:         handler.NewTaskHandler(taskSvc),
		maintenance:  handler.NewMaintenanceHandler(maintenanceSvc),
		attendance:   handler.NewAttendanceHandler(attendanceSvc),
		fuel:         handler.NewFuelHandler(fuelSvc),
		dashboard:    handler.NewDashboardHandler(dashboardSvc),
		form:         handler.NewFormHandler(formSvc),
	}

	r := newRouter(cfg, pool, h)

	addr := fmt.Sprintf(":%s", cfg.Port)
	slog.Info("Starting LTC API Server", slog.String("addr", addr), slog.String("env", cfg.AppEnv))
	if err := r.Run(addr); err != nil {
		slog.Error("Server terminated unexpectedly", slog.String("error", err.Error()))
		os.Exit(1)
	}
}

// connectDatabase 建立連線池並確認可連通；任何一步失敗都回傳 error，交由呼叫端依環境決定是否啟動。
func connectDatabase(ctx context.Context, cfg *config.Config) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("create database pool: %w", err)
	}
	pingCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}
	return pool, nil
}
