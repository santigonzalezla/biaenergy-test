package app

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/santigonzalezla/biaenergy-test/backend/internal/aiclient"
	"github.com/santigonzalezla/biaenergy-test/backend/internal/config"
	"github.com/santigonzalezla/biaenergy-test/backend/internal/httpserver"
	"github.com/santigonzalezla/biaenergy-test/backend/internal/modules/analysis"
	"github.com/santigonzalezla/biaenergy-test/backend/internal/modules/anomaly"
	"github.com/santigonzalezla/biaenergy-test/backend/internal/modules/auth"
	"github.com/santigonzalezla/biaenergy-test/backend/internal/modules/dashboard"
	"github.com/santigonzalezla/biaenergy-test/backend/internal/modules/health"
	"github.com/santigonzalezla/biaenergy-test/backend/internal/modules/ingestion"
	"github.com/santigonzalezla/biaenergy-test/backend/internal/modules/meter"
	"github.com/santigonzalezla/biaenergy-test/backend/internal/platform/database"
	"github.com/santigonzalezla/biaenergy-test/backend/internal/platform/token"
)

type Builder struct {
	cfg     config.Config
	pool    *pgxpool.Pool
	routes  httpserver.Routes
	workers []Worker
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

func (builder *Builder) WithModules(ctx context.Context) *Builder {
	if builder.err != nil {
		return builder
	}

	if builder.pool == nil {
		builder.err = errors.New("modules: WithDatabase must be called before WithModules")
		return builder
	}

	meterRepository := meter.NewPostgresRepository(builder.pool)

	aiClient := aiclient.New(builder.cfg.AiServiceUrl, aiclient.WithTimeout(builder.cfg.AiTimeout))
	analysisService := analysis.NewService(
		ctx,
		analysis.NewPostgresRepository(builder.pool),
		aiClient,
		builder.cfg.Location.String(),
		builder.cfg.AiTimeout,
	)

	if err := analysisService.RecoverInterrupted(ctx); err != nil {
		builder.err = fmt.Errorf("failed to recover interrupted analyses: %w", err)
		return builder
	}

	tokenIssuer := token.NewIssuer(builder.cfg.JwtSecret, builder.cfg.JwtTtl)
	requireAuth := auth.RequireAuth(tokenIssuer)
	authService := auth.NewService(auth.NewPostgresRepository(builder.pool), tokenIssuer)

	builder.workers = append(builder.workers, analysisService)
	builder.routes.RequireAuth = requireAuth
	builder.routes.Public = append(builder.routes.Public,
		health.NewHandler(builder.pool),
		auth.NewHandler(authService, requireAuth),
	)
	builder.routes.Protected = append(builder.routes.Protected,
		meter.NewHandler(meter.NewService(meterRepository, builder.cfg.Location)),
		ingestion.NewHandler(ingestion.NewService(ingestion.NewPostgresRepository(builder.pool), builder.cfg.Location)),
		dashboard.NewHandler(dashboard.NewService(dashboard.NewPostgresRepository(builder.pool))),
		analysis.NewHandler(analysisService),
		anomaly.NewHandler(anomaly.NewService(anomaly.NewPostgresRepository(builder.pool))),
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
		Handler:           httpserver.NewRouter(builder.cfg.AllowedOrigins, builder.routes),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	return &App{cfg: builder.cfg, server: server, pool: builder.pool, workers: builder.workers}, nil
}
