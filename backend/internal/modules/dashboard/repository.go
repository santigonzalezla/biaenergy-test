package dashboard

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/santigonzalezla/biaenergy-test/backend/internal/db"
)

type Repository interface {
	Summary(ctx context.Context) (db.GetDashboardSummaryRow, error)
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
