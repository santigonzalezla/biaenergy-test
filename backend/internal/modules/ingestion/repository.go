package ingestion

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/santigonzalezla/biaenergy-test/backend/internal/db"
)

const batchSize = 5000

const (
	defaultSector         = "INDUSTRIAL"
	defaultNominalVoltage = 220
)

type Stats struct {
	Inserted      int64
	MetersCreated int
}

type Repository interface {
	SaveReadings(ctx context.Context, rows []ReadingRow) (Stats, error)
	SaveEvents(ctx context.Context, rows []EventRow) (Stats, error)
}

type PostgresRepository struct {
	pool    *pgxpool.Pool
	queries *db.Queries
}

var _ Repository = (*PostgresRepository)(nil)

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool, queries: db.New(pool)}
}

func (repository *PostgresRepository) SaveReadings(ctx context.Context, rows []ReadingRow) (Stats, error) {
	var stats Stats

	err := repository.withTx(ctx, func(queries *db.Queries) error {
		codes := make([]string, len(rows))

		for i, row := range rows {
			codes[i] = row.MeterCode
		}

		metersIds, created, err := ensureMeters(ctx, queries, codes)

		if err != nil {
			return err
		}

		stats.MetersCreated = created

		for start := 0; start < len(rows); start += batchSize {
			batch := rows[start:min(start+batchSize, len(rows))]

			params := db.InsertReadingsParams{
				MeterIds:     make([]uuid.UUID, len(batch)),
				Timestamps:   make([]time.Time, len(batch)),
				Consumptions: make([]float64, len(batch)),
				Voltages:     make([]float64, len(batch)),
				Currents:     make([]float64, len(batch)),
				PowerFactors: make([]float64, len(batch)),
				Statuses:     make([]string, len(batch)),
			}

			for i, row := range batch {
				params.MeterIds[i] = metersIds[row.MeterCode]
				params.Timestamps[i] = row.Timestamp
				params.Consumptions[i] = row.ConsumptionKwh
				params.Voltages[i] = row.Voltage
				params.Currents[i] = row.Current
				params.PowerFactors[i] = row.PowerFactor
				params.Statuses[i] = row.Status
			}

			inserted, err := queries.InsertReadings(ctx, params)

			if err != nil {
				return fmt.Errorf("insert readings batch at row %d: %w", start, err)
			}

			stats.Inserted += inserted
		}

		return nil
	})

	return stats, err
}

func (repository PostgresRepository) SaveEvents(ctx context.Context, rows []EventRow) (Stats, error) {
	var stats Stats

	err := repository.withTx(ctx, func(queries *db.Queries) error {
		codes := make([]string, len(rows))

		for i, row := range rows {
			codes[i] = row.MeterCode
		}

		meterIds, created, err := ensureMeters(ctx, queries, codes)

		if err != nil {
			return err
		}

		stats.MetersCreated = created

		params := db.InsertEventsParams{
			MeterIds:     make([]uuid.UUID, len(rows)),
			Timestamps:   make([]time.Time, len(rows)),
			Types:        make([]string, len(rows)),
			Descriptions: make([]string, len(rows)),
		}

		for i, row := range rows {
			params.MeterIds[i] = meterIds[row.MeterCode]
			params.Timestamps[i] = row.Timestamp
			params.Types[i] = row.Type
			params.Descriptions[i] = row.Description
		}

		inserted, err := queries.InsertEvents(ctx, params)

		if err != nil {
			return fmt.Errorf("insert events: %w", err)
		}

		stats.Inserted = inserted

		return nil
	})

	return stats, err
}

func (repository *PostgresRepository) withTx(ctx context.Context, fn func(queries *db.Queries) error) error {
	tx, err := repository.pool.Begin(ctx)

	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}

	defer tx.Rollback(ctx)

	if err := fn(repository.queries.WithTx(tx)); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

func ensureMeters(ctx context.Context, queries *db.Queries, codes []string) (map[string]uuid.UUID, int, error) {
	metersIds := make(map[string]uuid.UUID)
	created := 0

	for _, code := range codes {
		if _, seen := metersIds[code]; seen {
			continue
		}

		existing, err := queries.GetMeterByCode(ctx, code)

		if err == nil {
			metersIds[code] = existing.UidMeter
			continue
		}

		if !errors.Is(err, pgx.ErrNoRows) {
			return nil, 0, fmt.Errorf("find meter %s: %w", code, err)
		}

		newMeter, err := queries.CreateMeter(ctx, db.CreateMeterParams{
			Code:           code,
			Name:           "Medidor " + code,
			Sector:         defaultSector,
			NominalVoltage: defaultNominalVoltage,
		})

		if err != nil {
			return nil, 0, fmt.Errorf("create meter %s: %w", code, err)
		}

		metersIds[code] = newMeter.UidMeter
		created++
	}

	return metersIds, created, nil
}
