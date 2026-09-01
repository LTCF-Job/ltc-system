package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	auditapp "ltc-system/apps/api/internal/modules/audit/app"
	auditinfra "ltc-system/apps/api/internal/modules/audit/infra"
	audittransport "ltc-system/apps/api/internal/modules/audit/transport"
	caregiverapp "ltc-system/apps/api/internal/modules/caregiver/app"
	caregiverinfra "ltc-system/apps/api/internal/modules/caregiver/infra"
	caregivertransport "ltc-system/apps/api/internal/modules/caregiver/transport"
	importapp "ltc-system/apps/api/internal/modules/caseimport/app"
	importinfra "ltc-system/apps/api/internal/modules/caseimport/infra"
	importtransport "ltc-system/apps/api/internal/modules/caseimport/transport"
	caseapp "ltc-system/apps/api/internal/modules/casemgmt/app"
	caseinfra "ltc-system/apps/api/internal/modules/casemgmt/infra"
	casetransport "ltc-system/apps/api/internal/modules/casemgmt/transport"
	drapp "ltc-system/apps/api/internal/modules/driverreport/app"
	drinfra "ltc-system/apps/api/internal/modules/driverreport/infra"
	drtransport "ltc-system/apps/api/internal/modules/driverreport/transport"
	holidayapp "ltc-system/apps/api/internal/modules/holiday/app"
	holidayinfra "ltc-system/apps/api/internal/modules/holiday/infra"
	holidaytransport "ltc-system/apps/api/internal/modules/holiday/transport"
	masterapp "ltc-system/apps/api/internal/modules/masterdata/app"
	masterinfra "ltc-system/apps/api/internal/modules/masterdata/infra"
	mastertransport "ltc-system/apps/api/internal/modules/masterdata/transport"
	notifyapp "ltc-system/apps/api/internal/modules/notification/app"
	notifyinfra "ltc-system/apps/api/internal/modules/notification/infra"
	notifytransport "ltc-system/apps/api/internal/modules/notification/transport"
	opsapp "ltc-system/apps/api/internal/modules/ops/app"
	opsinfra "ltc-system/apps/api/internal/modules/ops/infra"
	opstransport "ltc-system/apps/api/internal/modules/ops/transport"
	reportapp "ltc-system/apps/api/internal/modules/reporting/app"
	reportinfra "ltc-system/apps/api/internal/modules/reporting/infra"
	reporttransport "ltc-system/apps/api/internal/modules/reporting/transport"
	rideapp "ltc-system/apps/api/internal/modules/ride/app"
	rideinfra "ltc-system/apps/api/internal/modules/ride/infra"
	ridetransport "ltc-system/apps/api/internal/modules/ride/transport"
	taskapp "ltc-system/apps/api/internal/modules/task/app"
	taskinfra "ltc-system/apps/api/internal/modules/task/infra"
	tasktransport "ltc-system/apps/api/internal/modules/task/transport"
	"ltc-system/apps/api/internal/platform/config"
	"ltc-system/apps/api/internal/platform/pgxdb"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
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
	caseRepo := caseinfra.NewCaseRepository(pool)
	rideRepo := rideinfra.NewRideRepository(pool)
	holidayRepo := holidayinfra.NewHolidayRepository(pool)
	notificationRepo := notifyinfra.NewNotificationRepository(pool)
	auditSvc := auditapp.NewService(auditinfra.NewAuditRepository(pool))

	// masterdata 模組自有的 repository，legacy service 透過本檔的 adapter 取用
	mdRegionRepo := masterinfra.NewRegionRepository(pool)
	mdSiteRepo := masterinfra.NewSiteRepository(pool)
	mdVehicleRepo := masterinfra.NewVehicleRepository(pool)
	mdDriverRepo := masterinfra.NewDriverRepository(pool)
	maintenanceRepo := opsinfra.NewMaintenanceRepository(pool)
	attendanceRepo := opsinfra.NewAttendanceRepository(pool)
	fuelRepo := opsinfra.NewFuelRepository(pool)
	reportRepo := reportinfra.NewReportRepository(pool)
	dashboardRepo := reportinfra.NewDashboardRepository(pool)
	precheckRepo := reportinfra.NewPrecheckRepository(pool)
	govClaimRepo := reportinfra.NewGovClaimRepository(pool)
	exportJobRepo := reportinfra.NewExportJobRepository(pool)
	taskRepo := taskinfra.NewTaskRepository(pool)
	caregiverRepo := caregiverinfra.NewCaregiverRepository(pool)

	// 初始化 Services
	mdAudit := masterdataAuditWriter{svc: auditSvc}
	regionSvc := masterapp.NewRegionService(mdRegionRepo, mdAudit)
	siteSvc := masterapp.NewSiteService(mdSiteRepo)
	vehicleSvc := masterapp.NewVehicleService(mdVehicleRepo, mdDriverRepo)
	driverSvc := masterapp.NewDriverService(mdDriverRepo, cfg)
	txRunner := pgxdb.NewTxRunner(pool)
	caseSvc := caseapp.NewCaseService(cfg, caseRepo, caseSiteFinder{repo: mdSiteRepo}, caseAuditWriter{svc: auditSvc}, caseinfra.NewExcelRenderer())
	excelAdapter := importinfra.NewExcelAdapter()
	importSvc := importapp.NewImportService(
		caseRegistrar{svc: caseSvc},
		caseDuplicateFinder{svc: caseSvc},
		importSiteLookup{repo: mdSiteRepo},
		importVehicleLookup{repo: mdVehicleRepo},
		caseRepo,
		excelAdapter,
		excelAdapter,
		txRunner,
	)
	rideSvc := rideapp.NewRideService(rideRepo, rideDriverResolver{repo: mdDriverRepo}, rideScheduleReader{repo: caseRepo}, rideAuditWriter{svc: auditSvc})
	driverReportExcel := drinfra.NewExcelAdapter()
	driverReportSvc := drapp.NewDriverReportService(
		drinfra.NewDriverReportRepository(pool),
		driverReportExcel,
		driverReportExcel,
		driverReportCaseLookup{repo: caseRepo},
		driverReportDriverResolver{repo: mdDriverRepo},
		driverReportRideIngestor{svc: rideSvc},
		driverReportAuditWriter{svc: auditSvc},
		txRunner,
	)
	excelRenderer := reportinfra.NewExcelRenderer()
	precheckSvc := reportapp.NewPrecheckService(precheckRepo)
	govClaimSvc := reportapp.NewGovClaimService(cfg, govClaimRepo, exportJobRepo, excelRenderer, reportinfra.NewZipArchiver(), precheckSvc, reportingAuditWriter{svc: auditSvc})
	notificationSvc := notifyapp.NewNotificationService(notificationRepo, notificationAuditWriter{svc: auditSvc}, nil)
	holidayProvider := holidayapp.GovernmentHolidayProvider(&holidayinfra.GovernmentHolidayHTTPClient{
		Endpoint: holidayinfra.GovernmentHolidayCSVEndpoint,
		Client:   &http.Client{Timeout: cfg.GovernmentHolidayAPITimeout},
	})
	holidaySvc := holidayapp.NewHolidaySyncService(holidayRepo, holidayAuditWriter{svc: auditSvc}, holidayProvider)
	reportSvc := reportapp.NewReportService(reportRepo, excelRenderer)
	opsAudit := opsAuditWriter{svc: auditSvc}
	maintenanceSvc := opsapp.NewMaintenanceService(maintenanceRepo, opsVehicleLister{repo: mdVehicleRepo}, opsAudit, opsinfra.NewExcelRenderer())
	attendanceSvc := opsapp.NewAttendanceService(attendanceRepo, opsDriverLister{repo: mdDriverRepo}, opsAudit, holidayRepo)
	fuelSvc := opsapp.NewFuelService(fuelRepo, opsAudit)
	dashboardSvc := reportapp.NewDashboardService(dashboardRepo)
	taskSvc := taskapp.NewTaskService(taskRepo, taskScheduleReader{repo: caseRepo}, holidayRepo, notificationSvc)
	caregiverExcelAdapter := caregiverinfra.NewExcelAdapter()
	caregiverSvc := caregiverapp.NewCaregiverService(caregiverRepo, caregiverSiteLookup{repo: mdSiteRepo}, caregiverExcelAdapter, caregiverExcelAdapter)

	// 初始化 Handlers
	h := handlers{
		region:       mastertransport.NewRegionHandler(regionSvc),
		kase:         casetransport.NewCaseHandler(caseSvc),
		caseImport:   importtransport.NewImportHandler(importSvc),
		site:         mastertransport.NewSiteHandler(siteSvc),
		vehicle:      mastertransport.NewVehicleHandler(vehicleSvc),
		driver:       mastertransport.NewDriverHandler(driverSvc),
		ride:         ridetransport.NewRideHandler(rideSvc),
		export:       reporttransport.NewExportHandler(precheckSvc, govClaimSvc),
		notification: notifytransport.NewNotificationHandler(notificationSvc),
		holiday:      holidaytransport.NewHolidayHandler(holidaySvc),
		report:       reporttransport.NewReportHandler(reportSvc),
		audit:        audittransport.NewAuditHandler(auditSvc),
		task:         tasktransport.NewTaskHandler(taskSvc),
		maintenance:  opstransport.NewMaintenanceHandler(maintenanceSvc),
		attendance:   opstransport.NewAttendanceHandler(attendanceSvc),
		fuel:         opstransport.NewFuelHandler(fuelSvc),
		dashboard:    reporttransport.NewDashboardHandler(dashboardSvc),
		driverReport: drtransport.NewDriverReportHandler(driverReportSvc),
		caregiver:    caregivertransport.NewCaregiverHandler(caregiverSvc),
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
	// Supabase 的連線池網址走 pgbouncer transaction pooling，同一個 pgxpool 連線
	// 在不同請求間可能被路由到不同後端連線；pgx 預設會快取 prepared statement 名稱，
	// 在這種環境下會不定期撞名回傳 "prepared statement already exists"，需改用
	// simple protocol（見 cmd/migrate/main.go 同樣的修法）。
	poolCfg, err := pgxpool.ParseConfig(cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse database config: %w", err)
	}
	poolCfg.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
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
