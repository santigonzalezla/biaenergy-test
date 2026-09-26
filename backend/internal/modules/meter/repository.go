package meter

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/santigonzalezla/biaenergy-test/backend/internal/db"
)

const uniqueViolationCode = "23505"

var (
	ErrNotFound      = errors.New("meter not found")
	ErrDuplicateCode = errors.New("meter code already exists")
)

type Repository interface {
	List(ctx context.Context, params db.ListMetersParams) ([]db.Meter, int64, error)
	GetById(ctx context.Context, id uuid.UUID) (db.Meter, error)
	Create(ctx context.Context, params db.CreateMeterParams) (db.Meter, error)
	Update(ctx context.Context, params db.UpdateMeterParams) (db.Meter, error)
	SoftDelete(ctx context.Context, id uuid.UUID) error
	ListReadings(ctx context.Context, params db.ListReadingsByMeterParams) ([]db.ListReadingsByMeterRow, error)
	ListEvents(ctx context.Context, params db.ListEventsByMeterParams) ([]db.ListEventsByMeterRow, error)
	LatestReadingTime(ctx context.Context, id uuid.UUID) (time.Time, bool, error)
}

type PostgresRepository struct {
	queries *db.Queries
}

var _ Repository = (*PostgresRepository)(nil)

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{queries: db.New(pool)}
}

func (repository *PostgresRepository) List(ctx context.Context, params db.ListMetersParams) ([]db.Meter, int64, error) {
	meters, err := repository.queries.ListMeters(ctx, params)

	if err != nil {
		return nil, 0, fmt.Errorf("failed to list meters: %w", err)
	}

	total, err := repository.queries.CountMeters(ctx, db.CountMetersParams{
		Status: params.Status,
		Search: params.Search,
	})

	if err != nil {
		return nil, 0, fmt.Errorf("failed to count meters: %w", err)
	}

	return meters, total, nil
}

func (repository *PostgresRepository) GetById(ctx context.Context, id uuid.UUID) (db.Meter, error) {
	meter, err := repository.queries.GetMeterByID(ctx, id)

	if errors.Is(err, pgx.ErrNoRows) {
		return db.Meter{}, ErrNotFound
	}

	if err != nil {
		return db.Meter{}, fmt.Errorf("failed to get meter by ID: %w", err)
	}

	return meter, nil
}

func (repository *PostgresRepository) Create(ctx context.Context, params db.CreateMeterParams) (db.Meter, error) {
	meter, err := repository.queries.CreateMeter(ctx, params)

	if isUniqueViolation(err) {
		return db.Meter{}, ErrDuplicateCode
	}

	if err != nil {
		return db.Meter{}, fmt.Errorf("failed to create meter: %w", err)
	}

	return meter, nil
}

func (repository *PostgresRepository) Update(ctx context.Context, params db.UpdateMeterParams) (db.Meter, error) {
	meter, err := repository.queries.UpdateMeter(ctx, params)

	if errors.Is(err, pgx.ErrNoRows) {
		return db.Meter{}, ErrNotFound
	}

	if err != nil {
		return db.Meter{}, fmt.Errorf("failed to update meter: %w", err)
	}

	return meter, nil
}

func (repository *PostgresRepository) SoftDelete(ctx context.Context, id uuid.UUID) error {
	affected, err := repository.queries.SoftDeleteMeter(ctx, id)

	if err != nil {
		return fmt.Errorf("failed to soft delete meter: %w", err)
	}

	if affected == 0 {
		return ErrNotFound
	}

	return nil
}

func (repository *PostgresRepository) ListReadings(ctx context.Context, params db.ListReadingsByMeterParams) ([]db.ListReadingsByMeterRow, error) {
	readings, err := repository.queries.ListReadingsByMeter(ctx, params)

	if err != nil {
		return nil, fmt.Errorf("failed to list readings: %w", err)
	}

	return readings, nil
}

func (repository *PostgresRepository) ListEvents(ctx context.Context, params db.ListEventsByMeterParams) ([]db.ListEventsByMeterRow, error) {
	events, err := repository.queries.ListEventsByMeter(ctx, params)

	if err != nil {
		return nil, fmt.Errorf("failed to list events: %w", err)
	}

	return events, nil
}

func (repository *PostgresRepository) LatestReadingTime(ctx context.Context, id uuid.UUID) (time.Time, bool, error) {
	latest, err := repository.queries.GetLatestReadingTime(ctx, id)

	if errors.Is(err, pgx.ErrNoRows) {
		return time.Time{}, false, nil
	}

	if err != nil {
		return time.Time{}, false, fmt.Errorf("failed to get latest reading time: %w", err)
	}

	return latest, true, nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError

	return errors.As(err, &pgErr) && pgErr.Code == uniqueViolationCode
}
