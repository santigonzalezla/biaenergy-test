package anomaly

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/santigonzalezla/biaenergy-test/backend/internal/db"
)

var ErrNotFound = errors.New("anomaly not found")

type Repository interface {
	LatestCompletedAnalysisId(ctx context.Context) (uuid.UUID, bool, error)
	List(ctx context.Context, params db.ListAnomaliesParams) ([]db.ListAnomaliesRow, error)
	GetById(ctx context.Context, id uuid.UUID) (db.GetAnomalyRow, error)
	UpdateStatus(ctx context.Context, params db.UpdateAnomalyStatusParams) error
}

type PostgresRepository struct {
	queries *db.Queries
}

var _ Repository = (*PostgresRepository)(nil)

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{queries: db.New(pool)}
}

func (repository *PostgresRepository) LatestCompletedAnalysisId(ctx context.Context) (uuid.UUID, bool, error) {
	id, err := repository.queries.GetLatestCompletedAnalysisID(ctx)

	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, false, nil
	}

	if err != nil {
		return uuid.Nil, false, fmt.Errorf("failed to get latest completed analysis: %w", err)
	}

	return id, true, nil
}

func (repository *PostgresRepository) List(ctx context.Context, params db.ListAnomaliesParams) ([]db.ListAnomaliesRow, error) {
	anomalies, err := repository.queries.ListAnomalies(ctx, params)

	if err != nil {
		return nil, fmt.Errorf("failed to list anomalies: %w", err)
	}

	return anomalies, nil
}

func (repository *PostgresRepository) GetById(ctx context.Context, id uuid.UUID) (db.GetAnomalyRow, error) {
	anomaly, err := repository.queries.GetAnomaly(ctx, id)

	if errors.Is(err, pgx.ErrNoRows) {
		return db.GetAnomalyRow{}, ErrNotFound
	}

	if err != nil {
		return db.GetAnomalyRow{}, fmt.Errorf("failed to get anomaly: %w", err)
	}

	return anomaly, nil
}

func (repository *PostgresRepository) UpdateStatus(ctx context.Context, params db.UpdateAnomalyStatusParams) error {
	_, err := repository.queries.UpdateAnomalyStatus(ctx, params)

	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}

	if err != nil {
		return fmt.Errorf("failed to update anomaly status: %w", err)
	}

	return nil
}
