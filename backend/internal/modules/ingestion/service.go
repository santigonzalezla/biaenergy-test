package ingestion

import (
	"context"
	"io"
	"time"

	"github.com/santigonzalezla/biaenergy-test/backend/internal/apperror"
)

type ImportResult struct {
	Rows          int   `json:"rows"`
	Inserted      int64 `json:"inserted"`
	Skipped       int64 `json:"skipped"`
	MetersCreated int   `json:"metersCreated"`
}

type ImportFunc func(ctx context.Context, source io.Reader) (ImportResult, error)

type Service struct {
	repository Repository
	location   *time.Location
}

func NewService(repository Repository, location *time.Location) *Service {
	return &Service{repository: repository, location: location}
}

func (service *Service) ImportReadings(ctx context.Context, source io.Reader) (ImportResult, error) {
	rows, rowErrs, err := ParseReadings(source, service.location)

	if err := checkParsed(len(rows), rowErrs, err); err != nil {
		return ImportResult{}, err
	}

	stats, err := service.repository.SaveReadings(ctx, rows)

	if err != nil {
		return ImportResult{}, err
	}

	return newResult(len(rows), stats), nil
}

func (service *Service) ImportEvents(ctx context.Context, source io.Reader) (ImportResult, error) {
	rows, rowErrs, err := ParseEvents(source, service.location)

	if err := checkParsed(len(rows), rowErrs, err); err != nil {
		return ImportResult{}, err
	}

	stats, err := service.repository.SaveEvents(ctx, rows)

	if err != nil {
		return ImportResult{}, err
	}

	return newResult(len(rows), stats), nil
}

func checkParsed(rowCount int, rowErrs []RowError, err error) error {
	if err != nil {
		return apperror.BadRequest("INVALID_CSV", err.Error())
	}

	if len(rowErrs) > 0 {
		return apperror.Validation(rowErrs)
	}

	if rowCount == 0 {
		return apperror.BadRequest("EMPTY_CSV", "The file has no data rows")
	}

	return nil
}

func newResult(rowCount int, stats Stats) ImportResult {
	return ImportResult{
		Rows:          rowCount,
		Inserted:      stats.Inserted,
		Skipped:       int64(rowCount) - stats.Inserted,
		MetersCreated: stats.MetersCreated,
	}
}
