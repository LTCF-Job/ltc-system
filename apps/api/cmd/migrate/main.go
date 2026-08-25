package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"ltc-system/apps/api/internal/config"
)

var versionPattern = regexp.MustCompile(`^(\d+)_(.+)$`)

type migrationFile struct {
	version int
	name    string
	path    string
}

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

	if err := ensureMigrationsTable(ctx, pool); err != nil {
		slog.Error("Failed to prepare schema_migrations table", slog.String("error", err.Error()))
		os.Exit(1)
	}

	switch action {
	case "up":
		err = runUp(ctx, pool, migrationsDir)
	case "down":
		err = runDown(ctx, pool, migrationsDir)
	default:
		slog.Error("Unknown action", slog.String("action", action))
		os.Exit(1)
	}

	if err != nil {
		slog.Error("Migration failed", slog.String("action", action), slog.String("error", err.Error()))
		os.Exit(1)
	}

	slog.Info("Migration completed successfully", slog.String("action", action))
}

func ensureMigrationsTable(ctx context.Context, pool *pgxpool.Pool) error {
	_, err := pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version BIGINT PRIMARY KEY,
			name TEXT NOT NULL,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
		)
	`)
	return err
}

func appliedVersions(ctx context.Context, pool *pgxpool.Pool) (map[int]bool, error) {
	rows, err := pool.Query(ctx, `SELECT version FROM schema_migrations`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	applied := make(map[int]bool)
	for rows.Next() {
		var version int
		if err := rows.Scan(&version); err != nil {
			return nil, err
		}
		applied[version] = true
	}
	return applied, rows.Err()
}

// runUp 依版本號由小到大套用所有尚未執行過的 *.up.sql。
func runUp(ctx context.Context, pool *pgxpool.Pool, dir string) error {
	files, err := listMigrationFiles(dir, ".up.sql")
	if err != nil {
		return err
	}

	applied, err := appliedVersions(ctx, pool)
	if err != nil {
		return err
	}

	for _, f := range files {
		if applied[f.version] {
			continue
		}

		sqlBytes, err := os.ReadFile(f.path)
		if err != nil {
			return err
		}

		slog.Info("Applying migration", slog.Int("version", f.version), slog.String("file", f.path))

		tx, err := pool.Begin(ctx)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, string(sqlBytes)); err != nil {
			tx.Rollback(ctx)
			return err
		}
		if _, err := tx.Exec(ctx, `INSERT INTO schema_migrations (version, name) VALUES ($1, $2)`, f.version, f.name); err != nil {
			tx.Rollback(ctx)
			return err
		}
		if err := tx.Commit(ctx); err != nil {
			return err
		}
	}

	return nil
}

// runDown 回滾最後一筆已套用的 migration，對應該版本的 *.down.sql。
func runDown(ctx context.Context, pool *pgxpool.Pool, dir string) error {
	files, err := listMigrationFiles(dir, ".down.sql")
	if err != nil {
		return err
	}

	applied, err := appliedVersions(ctx, pool)
	if err != nil {
		return err
	}

	latest := -1
	for v := range applied {
		if v > latest {
			latest = v
		}
	}
	if latest == -1 {
		slog.Info("No applied migrations to roll back")
		return nil
	}

	var target *migrationFile
	for i := range files {
		if files[i].version == latest {
			target = &files[i]
			break
		}
	}
	if target == nil {
		return fmt.Errorf("down migration file not found for version %d", latest)
	}

	sqlBytes, err := os.ReadFile(target.path)
	if err != nil {
		return err
	}

	slog.Info("Rolling back migration", slog.Int("version", target.version), slog.String("file", target.path))

	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, string(sqlBytes)); err != nil {
		tx.Rollback(ctx)
		return err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM schema_migrations WHERE version = $1`, target.version); err != nil {
		tx.Rollback(ctx)
		return err
	}
	return tx.Commit(ctx)
}

func listMigrationFiles(dir, suffix string) ([]migrationFile, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var files []migrationFile
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), suffix) {
			continue
		}

		base := strings.TrimSuffix(entry.Name(), suffix)
		matches := versionPattern.FindStringSubmatch(base)
		if matches == nil {
			continue
		}

		version, err := strconv.Atoi(matches[1])
		if err != nil {
			continue
		}

		files = append(files, migrationFile{
			version: version,
			name:    matches[2],
			path:    filepath.Join(dir, entry.Name()),
		})
	}

	sort.Slice(files, func(i, j int) bool { return files[i].version < files[j].version })
	return files, nil
}
