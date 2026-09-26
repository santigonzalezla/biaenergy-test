package app

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/santigonzalezla/biaenergy-test/backend/internal/config"
	"github.com/santigonzalezla/biaenergy-test/backend/internal/httpserver"
	"github.com/santigonzalezla/biaenergy-test/backend/internal/modules/dashboard"
	"github.com/santigonzalezla/biaenergy-test/backend/internal/modules/health"
	"github.com/santigonzalezla/biaenergy-test/backend/internal/modules/ingestion"
	"github.com/santigonzalezla/biaenergy-test/backend/internal/modules/meter"
	"github.com/santigonzalezla/biaenergy-test/backend/internal/platform/database"
)

type Builder struct {
	cfg     config.Config
	pool    *pgxpool.Pool
	modules []httpserver.RouteRegister
	err     error
}

func NewBuilder(cfg config.Config) *Builder {
	return &Builder{cfg: cfg}
}

func (builder *Builder) WithDatabase(ctx context.Context) *Builder {
	if builder.err != nil {
		return builder
	}

	pool, err := database.NewPool(ctx, builder.cfg.DatabaseUrl)

	if err != nil {
		builder.err = fmt.Errorf("failed to initialize database: %w", err)
		return builder
	}

	builder.pool = pool

	return builder
}

func (builder *Builder) WithModules() *Builder {
	if builder.err != nil {
		return builder
	}

	if builder.pool == nil {

		builder.err = errors.New("modules: WithDatabase must be called before WithModules")
		return builder
	}

	meterRepository := meter.NewPostgresRepository(builder.pool)

	builder.modules = append(builder.modules,
		health.NewHandler(builder.pool),
		meter.NewHandler(meter.NewService(meterRepository)),
		ingestion.NewHandler(ingestion.NewService(ingestion.NewPostgresRepository(builder.pool), builder.cfg.Location)),
		dashboard.NewHandler(dashboard.NewService(dashboard.NewPostgresRepository(builder.pool))),
	)

	return builder
}

func (builder *Builder) Build() (*App, error) {
	if builder.err != nil {
		if builder.pool != nil {
			builder.pool.Close()
		}

		return nil, builder.err
	}

	server := &http.Server{
		Addr:              ":" + builder.cfg.Port,
		Handler:           httpserver.NewRouter(builder.cfg.AllowedOrigins, builder.modules...),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	return &App{cfg: builder.cfg, server: server, pool: builder.pool}, nil
}
