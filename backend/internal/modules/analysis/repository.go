package analysis

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/santigonzalezla/biaenergy-test/backend/internal/db"
)

var ErrNotFound = errors.New("analysis not found")

type Dataset struct {
	Meters   []db.ListMetersForAnalysisRow
	Readings []db.ListReadingsForAnalysisRow
	Events   []db.ListEventsForAnalysisRow
}

type Summary struct {
	MetersAnalyzed int32
	Anomalies      int32
	HighPriority   int32
}

type Repository interface {
	Create(ctx context.Context) (db.Analysis, error)
	GetById(ctx context.Context, id uuid.UUID) (db.Analysis, error)
	Latest(ctx context.Context) (db.Analysis, bool, error)
	Active(ctx context.Context) (db.Analysis, bool, error)
	Advance(ctx context.Context, id uuid.UUID, step string, progress int16) error
	Fail(ctx context.Context, id uuid.UUID, message string) error
	FailInterrupted(ctx context.Context) (int64, error)
	LoadDataset(ctx context.Context, meterIds []uuid.UUID) (Dataset, error)
	Complete(ctx context.Context, id uuid.UUID, meterIds []uuid.UUID, anomalies []db.InsertAnomalyParams, summary Summary) error
}

type PostgresRepository struct {
	pool    *pgxpool.Pool
	queries *db.Queries
}

var _ Repository = (*PostgresRepository)(nil)

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool, queries: db.New(pool)}
}

func (repository *PostgresRepository) Create(ctx context.Context) (db.Analysis, error) {
	analysis, err := repository.queries.CreateAnalysis(ctx, nil)

	if err != nil {
		return db.Analysis{}, fmt.Errorf("failed to create analysis: %w", err)
	}

	return analysis, nil
}

func (repository *PostgresRepository) GetById(ctx context.Context, id uuid.UUID) (db.Analysis, error) {
	analysis, err := repository.queries.GetAnalysis(ctx, id)

	if errors.Is(err, pgx.ErrNoRows) {
		return db.Analysis{}, ErrNotFound
	}

	if err != nil {
		return db.Analysis{}, fmt.Errorf("failed to get analysis: %w", err)
	}

	return analysis, nil
}

func (repository *PostgresRepository) Latest(ctx context.Context) (db.Analysis, bool, error) {
	return optional(repository.queries.GetLatestAnalysis(ctx))
}

func (repository *PostgresRepository) Active(ctx context.Context) (db.Analysis, bool, error) {
	return optional(repository.queries.GetActiveAnalysis(ctx))
}

func (repository *PostgresRepository) Advance(ctx context.Context, id uuid.UUID, step string, progress int16) error {
	err := repository.queries.AdvanceAnalysis(ctx, db.AdvanceAnalysisParams{ID: id, Step: step, Progress: progress})

	if err != nil {
		return fmt.Errorf("failed to advance analysis to %s: %w", step, err)
	}

	return nil
}

func (repository *PostgresRepository) Fail(ctx context.Context, id uuid.UUID, message string) error {
	err := repository.queries.FailAnalysis(ctx, db.FailAnalysisParams{ID: id, ErrorMessage: &message})

	if err != nil {
		return fmt.Errorf("failed to mark analysis as failed: %w", err)
	}

	return nil
}

func (repository *PostgresRepository) FailInterrupted(ctx context.Context) (int64, error) {
	affected, err := repository.queries.FailInterruptedAnalyses(ctx)

	if err != nil {
		return 0, fmt.Errorf("failed to close interrupted analyses: %w", err)
	}

	return affected, nil
}

func (repository *PostgresRepository) LoadDataset(ctx context.Context, meterIds []uuid.UUID) (Dataset, error) {
	meters, err := repository.queries.ListMetersForAnalysis(ctx, meterIds)

	if err != nil {
		return Dataset{}, fmt.Errorf("failed to load meters: %w", err)
	}

	readings, err := repository.queries.ListReadingsForAnalysis(ctx, meterIds)

	if err != nil {
		return Dataset{}, fmt.Errorf("failed to load readings: %w", err)
	}

	events, err := repository.queries.ListEventsForAnalysis(ctx, meterIds)

	if err != nil {
		return Dataset{}, fmt.Errorf("failed to load events: %w", err)
	}

	return Dataset{Meters: meters, Readings: readings, Events: events}, nil
}

func (repository *PostgresRepository) Complete(ctx context.Context, id uuid.UUID, meterIds []uuid.UUID, anomalies []db.InsertAnomalyParams, summary Summary) error {
	tx, err := repository.pool.Begin(ctx)

	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	queries := repository.queries.WithTx(tx)

	for _, anomaly := range anomalies {
		if err := queries.InsertAnomaly(ctx, anomaly); err != nil {
			return fmt.Errorf("failed to save anomaly for meter %s: %w", anomaly.MeterID, err)
		}
	}

	err = queries.RefreshMeterStatuses(ctx, db.RefreshMeterStatusesParams{AnalysisID: id, MeterIds: meterIds})

	if err != nil {
		return fmt.Errorf("failed to refresh meter statuses: %w", err)
	}

	err = queries.CompleteAnalysis(ctx, db.CompleteAnalysisParams{
		ID:             id,
		MetersAnalyzed: summary.MetersAnalyzed,
		Anomalies:      summary.Anomalies,
		HighPriority:   summary.HighPriority,
	})

	if err != nil {
		return fmt.Errorf("failed to complete analysis: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit analysis results: %w", err)
	}

	return nil
}

func optional(analysis db.Analysis, err error) (db.Analysis, bool, error) {
	if errors.Is(err, pgx.ErrNoRows) {
		return db.Analysis{}, false, nil
	}

	if err != nil {
		return db.Analysis{}, false, fmt.Errorf("failed to query analysis: %w", err)
	}

	return analysis, true, nil
}
