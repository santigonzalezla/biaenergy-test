package ingestion

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"time"

	"github.com/santigonzalezla/biaenergy-test/backend/internal/apperror"
	"github.com/santigonzalezla/biaenergy-test/backend/internal/db"
)

const (
	defaultListLimit = 20
	maxListLimit     = 100
)

type ImportFunc func(ctx context.Context, upload Upload) (ImportResult, error)

type Service struct {
	repository Repository
	location   *time.Location
}

func NewService(repository Repository, location *time.Location) *Service {
	return &Service{repository: repository, location: location}
}

func (service *Service) ImportReadings(ctx context.Context, upload Upload) (ImportResult, error) {
	content, batch, err := service.prepare(upload, db.ImportKindREADINGS)

	if err != nil {
		return ImportResult{}, err
	}

	rows, rowErrs, err := ParseReadings(bytes.NewReader(content), service.location)

	if err := service.checkParsed(ctx, batch, len(rows), rowErrs, err); err != nil {
		return ImportResult{}, err
	}

	batch.Rows = len(rows)

	return service.save(ctx, batch, func() (Stats, error) {
		return service.repository.SaveReadings(ctx, batch, rows)
	})
}

func (service *Service) ImportEvents(ctx context.Context, upload Upload) (ImportResult, error) {
	content, batch, err := service.prepare(upload, db.ImportKindEVENTS)

	if err != nil {
		return ImportResult{}, err
	}

	rows, rowErrs, err := ParseEvents(bytes.NewReader(content), service.location)

	if err := service.checkParsed(ctx, batch, len(rows), rowErrs, err); err != nil {
		return ImportResult{}, err
	}

	batch.Rows = len(rows)

	return service.save(ctx, batch, func() (Stats, error) {
		return service.repository.SaveEvents(ctx, batch, rows)
	})
}

func (service *Service) List(ctx context.Context, limit int) (ListResponse, error) {
	if limit <= 0 {
		limit = defaultListLimit
	}

	batches, err := service.repository.ListBatches(ctx, int32(min(limit, maxListLimit)))

	if err != nil {
		return ListResponse{}, err
	}

	return ListResponse{Data: toBatchResponses(batches)}, nil
}

func (service *Service) prepare(upload Upload, kind db.ImportKind) ([]byte, NewBatch, error) {
	content, err := io.ReadAll(upload.Content)

	if err != nil {
		return nil, NewBatch{}, apperror.BadRequest("INVALID_FILE", "The file could not be read").WithCause(err)
	}

	checksum := sha256.Sum256(content)

	return content, NewBatch{
		Kind:     kind,
		FileName: upload.FileName,
		FileSize: int64(len(content)),
		Checksum: hex.EncodeToString(checksum[:]),
		UserId:   upload.UserId,
	}, nil
}

func (service *Service) save(ctx context.Context, batch NewBatch, persist func() (Stats, error)) (ImportResult, error) {
	previous, err := service.repository.PreviousImport(ctx, batch.Kind, batch.Checksum)

	if err != nil {
		return ImportResult{}, err
	}

	stats, err := persist()

	if err != nil {
		return ImportResult{}, err
	}

	return ImportResult{
		BatchId:              stats.BatchId,
		Rows:                 batch.Rows,
		Inserted:             stats.Inserted,
		Skipped:              int64(batch.Rows) - stats.Inserted,
		MetersCreated:        stats.MetersCreated,
		MetersRestored:       stats.MetersRestored,
		PreviouslyImportedAt: previous,
	}, nil
}

func (service *Service) checkParsed(ctx context.Context, batch NewBatch, rowCount int, rowErrs []RowError, parseErr error) error {
	var failure *apperror.AppError

	switch {
	case parseErr != nil:
		failure = apperror.BadRequest("INVALID_CSV", parseErr.Error())
	case len(rowErrs) > 0:
		failure = apperror.Validation(rowErrs)
	case rowCount == 0:
		failure = apperror.BadRequest("EMPTY_CSV", "The file has no data rows")
	default:
		return nil
	}

	if err := service.repository.RecordFailure(ctx, batch, Failure{Code: failure.Code, Message: describeFailure(failure, rowErrs)}); err != nil {
		slog.WarnContext(ctx, "import failure could not be recorded", "file", batch.FileName, "error", err)
	}

	return failure
}

func describeFailure(failure *apperror.AppError, rowErrs []RowError) string {
	if len(rowErrs) == 0 {
		return failure.Message
	}

	first := rowErrs[0]
	message := fmt.Sprintf("%d invalid rows; first at line %d", len(rowErrs), first.Line)

	if first.Column != "" {
		message += fmt.Sprintf(", column %s", first.Column)
	}

	return message + ": " + first.Message
}
