package main

import (
	"context"
	"crypto/sha256"
	"fmt"
	"log/slog"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"ltc-system/apps/api/internal/config"
	"ltc-system/apps/api/internal/domain/govform"
	"ltc-system/apps/api/internal/export"
	"ltc-system/apps/api/internal/repository"
	"ltc-system/apps/api/internal/service"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	cfg, err := config.LoadFromEnv()
	if err != nil {
		slog.Error("Failed to load config", slog.String("error", err.Error()))
		os.Exit(1)
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("Failed to connect to database", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer pool.Close()

	periodYM := "11507"
	if len(os.Args) > 1 {
		periodYM = os.Args[1]
	}
	region := "hsinchu"
	if len(os.Args) > 2 {
		region = os.Args[2]
	}

	slog.Info("Starting Export Job", slog.String("periodYM", periodYM), slog.String("region", region))

	precheckRepo := repository.NewPrecheckRepository(pool)
	precheckSvc := service.NewPrecheckService(precheckRepo)
	report, err := precheckSvc.RunPrecheck(ctx, periodYM, region)
	if err != nil || !report.Passed {
		slog.Error("Precheck failed, aborting export", slog.Int("errors", report.TotalErrors))
		os.Exit(1)
	}

	// 產生範例申報行
	var sampleRows []govform.ClaimRow
	govExcelBytes, err := export.GenerateGovClaimExcel(sampleRows)
	if err != nil {
		slog.Error("Failed to generate excel", slog.String("error", err.Error()))
		os.Exit(1)
	}

	checksum := fmt.Sprintf("%x", sha256.Sum256(govExcelBytes))
	slog.Info("Export completed successfully",
		slog.String("checksum", checksum),
		slog.Int("bytes", len(govExcelBytes)),
	)
}
