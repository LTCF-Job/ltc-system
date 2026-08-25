package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/jackc/pgx/v5/pgxpool"
	"ltc-system/apps/api/internal/config"
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

	if err := pool.Ping(ctx); err != nil {
		slog.Error("Failed to ping database", slog.String("error", err.Error()))
		os.Exit(1)
	}

	action := "up"
	if len(os.Args) > 1 {
		action = os.Args[1]
	}

	migrationsDir := "migrations"
	if _, err := os.Stat(migrationsDir); os.IsNotExist(err) {
		migrationsDir = filepath.Join("apps", "api", "migrations")
	}

	var filename string
	if action == "down" {
		filename = filepath.Join(migrationsDir, "000001_init_schema.down.sql")
	} else {
		filename = filepath.Join(migrationsDir, "000001_init_schema.up.sql")
	}

	sqlBytes, err := os.ReadFile(filename)
	if err != nil {
		slog.Error("Failed to read migration file", slog.String("file", filename), slog.String("error", err.Error()))
		os.Exit(1)
	}

	slog.Info("Running migration", slog.String("action", action), slog.String("file", filename))
	if _, err := pool.Exec(ctx, string(sqlBytes)); err != nil {
		slog.Error("Migration failed", slog.String("error", err.Error()))
		os.Exit(1)
	}

	slog.Info("Migration completed successfully", slog.String("action", action))
	fmt.Println("Migration applied successfully!")
}
