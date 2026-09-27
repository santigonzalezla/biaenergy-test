package dashboard

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/santigonzalezla/biaenergy-test/backend/internal/db"
)

type Repository interface {
	Summary(ctx context.Context) (db.GetDashboardSummaryRow, error)
	LatestAnalysis(ctx context.Context) (db.Analysis, bool, error)
}

type PostgresRepository struct {
	queries *db.Queries
}

var _ Repository = (*PostgresRepository)(nil)

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{queries: db.New(pool)}
}

func (repository *PostgresRepository) Summary(ctx context.Context) (db.GetDashboardSummaryRow, error) {
	summary, err := repository.queries.GetDashboardSummary(ctx)

	if err != nil {
		return db.GetDashboardSummaryRow{}, fmt.Errorf("failed to get dashboard summary: %w", err)
	}

	return summary, nil
}

func (repository *PostgresRepository) LatestAnalysis(ctx context.Context) (db.Analysis, bool, error) {
	analysis, err := repository.queries.GetLatestAnalysis(ctx)

	if errors.Is(err, pgx.ErrNoRows) {
		return db.Analysis{}, false, nil
	}

	if err != nil {
		return db.Analysis{}, false, fmt.Errorf("failed to get latest analysis: %w", err)
	}

	return analysis, true, nil
}
