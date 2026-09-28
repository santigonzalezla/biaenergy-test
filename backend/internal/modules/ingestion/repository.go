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

type NewBatch struct {
	Kind     db.ImportKind
	FileName string
	FileSize int64
	Checksum string
	UserId   *uuid.UUID
	Rows     int
}

type Stats struct {
	BatchId        uuid.UUID
	Inserted       int64
	MetersCreated  int
	MetersRestored int
}

type Failure struct {
	Code    string
	Message string
}

type Repository interface {
	SaveReadings(ctx context.Context, batch NewBatch, rows []ReadingRow) (Stats, error)
	SaveEvents(ctx context.Context, batch NewBatch, rows []EventRow) (Stats, error)
	RecordFailure(ctx context.Context, batch NewBatch, failure Failure) error
	PreviousImport(ctx context.Context, kind db.ImportKind, checksum string) (*time.Time, error)
	ListBatches(ctx context.Context, limit int32) ([]db.ListImportBatchesRow, error)
}

type PostgresRepository struct {
	pool    *pgxpool.Pool
	queries *db.Queries
}

var _ Repository = (*PostgresRepository)(nil)

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool, queries: db.New(pool)}
}

func (repository *PostgresRepository) SaveReadings(ctx context.Context, batch NewBatch, rows []ReadingRow) (Stats, error) {
	codes := make([]string, len(rows))

	for i, row := range rows {
		codes[i] = row.MeterCode
	}

	return repository.saveBatch(ctx, batch, codes, func(queries *db.Queries, batchId uuid.UUID, meterIds map[string]uuid.UUID) (int64, error) {
		var inserted int64

		for start := 0; start < len(rows); start += batchSize {
			chunk := rows[start:min(start+batchSize, len(rows))]

			params := db.InsertReadingsParams{
				MeterIds:      make([]uuid.UUID, len(chunk)),
				Timestamps:    make([]time.Time, len(chunk)),
				Consumptions:  make([]float64, len(chunk)),
				Voltages:      make([]float64, len(chunk)),
				Currents:      make([]float64, len(chunk)),
				PowerFactors:  make([]float64, len(chunk)),
				Statuses:      make([]string, len(chunk)),
				ImportBatchID: &batchId,
			}

			for i, row := range chunk {
				params.MeterIds[i] = meterIds[row.MeterCode]
				params.Timestamps[i] = row.Timestamp
				params.Consumptions[i] = row.ConsumptionKwh
				params.Voltages[i] = row.Voltage
				params.Currents[i] = row.Current
				params.PowerFactors[i] = row.PowerFactor
				params.Statuses[i] = row.Status
			}

			count, err := queries.InsertReadings(ctx, params)

			if err != nil {
				return 0, fmt.Errorf("insert readings batch at row %d: %w", start, err)
			}

			inserted += count
		}

		return inserted, nil
	})
}

func (repository *PostgresRepository) SaveEvents(ctx context.Context, batch NewBatch, rows []EventRow) (Stats, error) {
	codes := make([]string, len(rows))

	for i, row := range rows {
		codes[i] = row.MeterCode
	}

	return repository.saveBatch(ctx, batch, codes, func(queries *db.Queries, batchId uuid.UUID, meterIds map[string]uuid.UUID) (int64, error) {
		params := db.InsertEventsParams{
			MeterIds:      make([]uuid.UUID, len(rows)),
			Timestamps:    make([]time.Time, len(rows)),
			Types:         make([]string, len(rows)),
			Descriptions:  make([]string, len(rows)),
			ImportBatchID: &batchId,
		}

		for i, row := range rows {
			params.MeterIds[i] = meterIds[row.MeterCode]
			params.Timestamps[i] = row.Timestamp
			params.Types[i] = row.Type
			params.Descriptions[i] = row.Description
		}

		inserted, err := queries.InsertEvents(ctx, params)

		if err != nil {
			return 0, fmt.Errorf("insert events: %w", err)
		}

		return inserted, nil
	})
}

func (repository *PostgresRepository) RecordFailure(ctx context.Context, batch NewBatch, failure Failure) error {
	_, err := repository.queries.CreateImportBatch(ctx, db.CreateImportBatchParams{
		Kind:      batch.Kind,
		Status:    db.ImportStatusFAILED,
		FileName:  batch.FileName,
		FileSize:  batch.FileSize,
		Checksum:  batch.Checksum,
		Rows:      int32(batch.Rows),
		ErrorCode: &failure.Code,
		Error:     &failure.Message,
		UserID:    batch.UserId,
	})

	if err != nil {
		return fmt.Errorf("failed to record import failure: %w", err)
	}

	return nil
}

func (repository *PostgresRepository) PreviousImport(ctx context.Context, kind db.ImportKind, checksum string) (*time.Time, error) {
	previous, err := repository.queries.FindCompletedImportByChecksum(ctx, db.FindCompletedImportByChecksumParams{Kind: kind, Checksum: checksum})

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf("failed to find previous import: %w", err)
	}

	importedAt := previous.DtmCreatedAt.UTC()

	return &importedAt, nil
}

func (repository *PostgresRepository) ListBatches(ctx context.Context, limit int32) ([]db.ListImportBatchesRow, error) {
	batches, err := repository.queries.ListImportBatches(ctx, limit)

	if err != nil {
		return nil, fmt.Errorf("failed to list import batches: %w", err)
	}

	return batches, nil
}

type insertFunc func(queries *db.Queries, batchId uuid.UUID, meterIds map[string]uuid.UUID) (int64, error)

func (repository *PostgresRepository) saveBatch(ctx context.Context, batch NewBatch, codes []string, insert insertFunc) (Stats, error) {
	var stats Stats

	err := repository.withTx(ctx, func(queries *db.Queries) error {
		created, err := queries.CreateImportBatch(ctx, db.CreateImportBatchParams{
			Kind:     batch.Kind,
			Status:   db.ImportStatusCOMPLETED,
			FileName: batch.FileName,
			FileSize: batch.FileSize,
			Checksum: batch.Checksum,
			Rows:     int32(batch.Rows),
			UserID:   batch.UserId,
		})

		if err != nil {
			return fmt.Errorf("create import batch: %w", err)
		}

		meterIds, meters, err := ensureMeters(ctx, queries, codes)

		if err != nil {
			return err
		}

		inserted, err := insert(queries, created.UidImportBatch, meterIds)

		if err != nil {
			return err
		}

		err = queries.CompleteImportBatch(ctx, db.CompleteImportBatchParams{
			ID:             created.UidImportBatch,
			Inserted:       int32(inserted),
			Skipped:        int32(int64(batch.Rows) - inserted),
			MetersCreated:  int32(meters.created),
			MetersRestored: int32(meters.restored),
		})

		if err != nil {
			return fmt.Errorf("complete import batch: %w", err)
		}

		stats = Stats{BatchId: created.UidImportBatch, Inserted: inserted, MetersCreated: meters.created, MetersRestored: meters.restored}

		return nil
	})

	return stats, err
}

type meterChanges struct {
	created  int
	restored int
}

func ensureMeters(ctx context.Context, queries *db.Queries, codes []string) (map[string]uuid.UUID, meterChanges, error) {
	meterIds := make(map[string]uuid.UUID)
	var changes meterChanges

	for _, code := range codes {
		if _, seen := meterIds[code]; seen {
			continue
		}

		existing, err := queries.GetMeterByCode(ctx, code)

		if err == nil {
			meterIds[code] = existing.UidMeter
			continue
		}

		if !errors.Is(err, pgx.ErrNoRows) {
			return nil, changes, fmt.Errorf("find meter %s: %w", code, err)
		}

		restored, err := queries.RestoreMeterByCode(ctx, code)

		if err == nil {
			meterIds[code] = restored.UidMeter
			changes.restored++
			continue
		}

		if !errors.Is(err, pgx.ErrNoRows) {
			return nil, changes, fmt.Errorf("restore meter %s: %w", code, err)
		}

		newMeter, err := queries.CreateMeter(ctx, db.CreateMeterParams{
			Code:           code,
			Name:           "Medidor " + code,
			Sector:         defaultSector,
			NominalVoltage: defaultNominalVoltage,
		})

		if err != nil {
			return nil, changes, fmt.Errorf("create meter %s: %w", code, err)
		}

		meterIds[code] = newMeter.UidMeter
		changes.created++
	}

	return meterIds, changes, nil
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
