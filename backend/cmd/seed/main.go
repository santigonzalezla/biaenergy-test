package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	_ "time/tzdata"

	"github.com/joho/godotenv"
	"github.com/santigonzalezla/biaenergy-test/backend/internal/apperror"
	"github.com/santigonzalezla/biaenergy-test/backend/internal/config"
	"github.com/santigonzalezla/biaenergy-test/backend/internal/modules/auth"
	"github.com/santigonzalezla/biaenergy-test/backend/internal/modules/ingestion"
	"github.com/santigonzalezla/biaenergy-test/backend/internal/platform/database"
)

func main() {
	if err := run(); err != nil {
		slog.Error("seed failed", "error", err)
		os.Exit(1)
	}
}

func run() error {
	readingsPath := flag.String("readings", "../data/readings.csv", "path to readings CSV")
	eventsPath := flag.String("events", "../data/events.csv", "path to events CSV")
	flag.Parse()

	_ = godotenv.Load("../.env")

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := database.NewPool(ctx, cfg.DatabaseUrl)
	if err != nil {
		return err
	}
	defer pool.Close()

	if err := seedAdmin(ctx, auth.NewPostgresRepository(pool)); err != nil {
		return err
	}

	service := ingestion.NewService(ingestion.NewPostgresRepository(pool), cfg.Location)

	if err := importFile(ctx, "readings", *readingsPath, service.ImportReadings); err != nil {
		return err
	}

	return importFile(ctx, "events", *eventsPath, service.ImportEvents)
}

func seedAdmin(ctx context.Context, writer auth.UserWriter) error {
	account := auth.AdminAccount{
		Email:    envOrDefault("ADMIN_EMAIL", "admin@bia.app"),
		Password: os.Getenv("ADMIN_PASSWORD"),
		Name:     envOrDefault("ADMIN_NAME", "Admin BIA"),
	}

	if account.Password == "" {
		slog.Warn("admin user skipped", "reason", "ADMIN_PASSWORD is not set")
		return nil
	}

	user, err := auth.UpsertAdmin(ctx, writer, account)

	if err != nil {
		var appErr *apperror.AppError
		if errors.As(err, &appErr) && appErr.Details != nil {
			slog.Error("invalid admin account", "details", appErr.Details)
		}

		return fmt.Errorf("seed admin: %w", err)
	}

	slog.Info("admin user ready", "email", user.Email, "name", user.Name)

	return nil
}

func envOrDefault(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}

	return fallback
}

func importFile(ctx context.Context, label, path string, importFn ingestion.ImportFunc) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open %s: %w", path, err)
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return fmt.Errorf("stat %s: %w", path, err)
	}

	result, err := importFn(ctx, ingestion.Upload{FileName: filepath.Base(path), Size: info.Size(), Content: file})
	if err != nil {
		var appErr *apperror.AppError
		if errors.As(err, &appErr) && appErr.Details != nil {
			slog.Error("invalid rows", "file", path, "details", appErr.Details)
		}

		return fmt.Errorf("import %s: %w", label, err)
	}

	slog.Info("import completed",
		"file", label,
		"rows", result.Rows,
		"inserted", result.Inserted,
		"skipped", result.Skipped,
		"metersCreated", result.MetersCreated,
		"metersRestored", result.MetersRestored,
	)

	return nil
}
