package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/santigonzalezla/biaenergy-test/backend/internal/config"
)

const shutdownTimeout = 10 * time.Second

type App struct {
	cfg    config.Config
	server *http.Server
	pool   *pgxpool.Pool
}

func (app *App) Run(ctx context.Context) error {
	defer app.pool.Close()

	serverErr := make(chan error, 1)

	go func() {
		slog.Info("server started", "port", app.cfg.Port, "env", app.cfg.Env)

		if err := app.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	select {
		case err := <- serverErr:
			return fmt.Errorf("server failed: %w", err)
		case <- ctx.Done():
			slog.Info("shutdown signal received")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := app.server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("graceful shutdown failed: %w", err)
	}

	slog.Info("server stopped gracefully")

	return nil
}
