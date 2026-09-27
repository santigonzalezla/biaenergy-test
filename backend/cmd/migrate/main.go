package main

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/joho/godotenv"
	"github.com/pressly/goose/v3"
	schema "github.com/santigonzalezla/biaenergy-test/backend/db"
)

func main() {
	if err := run(); err != nil {
		slog.Error("migration failed", "error", err)
		os.Exit(1)
	}
}

func run() error {
	_ = godotenv.Load("../.env")

	databaseUrl := os.Getenv("DATABASE_URL")
	if databaseUrl == "" {
		return fmt.Errorf("DATABASE_URL is required")
	}

	command := "up"
	if len(os.Args) > 1 {
		command = os.Args[1]
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	database, err := sql.Open("pgx", databaseUrl)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer database.Close()

	if err := database.PingContext(ctx); err != nil {
		return fmt.Errorf("failed to reach database: %w", err)
	}

	goose.SetBaseFS(schema.Migrations)

	if err := goose.SetDialect("postgres"); err != nil {
		return err
	}

	switch command {
	case "up":
		return goose.UpContext(ctx, database, schema.MigrationsDir)
	case "down":
		return goose.DownContext(ctx, database, schema.MigrationsDir)
	case "status":
		return goose.StatusContext(ctx, database, schema.MigrationsDir)
	default:
		return fmt.Errorf("unknown command %q: use up, down or status", command)
	}
}
