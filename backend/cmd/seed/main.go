package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	_ "time/tzdata"

	"github.com/joho/godotenv"
	"github.com/santigonzalezla/biaenergy-test/backend/internal/apperror"
	"github.com/santigonzalezla/biaenergy-test/backend/internal/config"
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

	service := ingestion.NewService(ingestion.NewPostgresRepository(pool), cfg.Location)

	if err := importFile(ctx, "readings", *readingsPath, service.ImportReadings); err != nil {
		return err
	}

	return importFile(ctx, "events", *eventsPath, service.ImportEvents)
}

func importFile(ctx context.Context, label, path string, importFn ingestion.ImportFunc) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open %s: %w", path, err)
	}
	defer file.Close()

	result, err := importFn(ctx, file)
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
	)

	return nil
}
