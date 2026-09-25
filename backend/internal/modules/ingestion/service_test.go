package ingestion

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/santigonzalezla/biaenergy-test/backend/internal/apperror"
)

// fakeRepository simula que ya existían `existing` filas: solo "inserta" el resto
type fakeRepository struct {
	existing int64
	err      error
	called   bool
}

func (fake *fakeRepository) SaveReadings(ctx context.Context, rows []ReadingRow) (Stats, error) {
	fake.called = true

	return Stats{Inserted: int64(len(rows)) - fake.existing, MetersCreated: 1}, fake.err
}

func (fake *fakeRepository) SaveEvents(ctx context.Context, rows []EventRow) (Stats, error) {
	fake.called = true

	return Stats{Inserted: int64(len(rows)) - fake.existing}, fake.err
}

func TestServiceImportReadings(t *testing.T) {
	validCSV := readingsHeader +
		"M-101,2026-09-01 00:00:00,23.5,221.9,101.28,0.954,OK\n" +
		"M-101,2026-09-01 01:00:00,20.1,221.1,100.49,0.935,OK\n"

	tests := []struct {
		name           string
		csv            string
		existing       int64
		repositoryErr  error
		wantStatus     int
		wantCode       string
		wantRepoCalled bool
		wantResult     ImportResult
	}{
		{
			name:           "First import inserts everything",
			csv:            validCSV,
			wantRepoCalled: true,
			wantResult:     ImportResult{Rows: 2, Inserted: 2, Skipped: 0, MetersCreated: 1},
		},
		{
			name:           "Re-import skips existing rows",
			csv:            validCSV,
			existing:       2,
			wantRepoCalled: true,
			wantResult:     ImportResult{Rows: 2, Inserted: 0, Skipped: 2, MetersCreated: 1},
		},
		{name: "Invalid row rejects whole file", csv: validCSV + "M-101,2026-09-01 02:00:00,abc,220,1,0.9,OK\n", wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR"},
		{name: "Missing columns", csv: "meter_id\nM-101\n", wantStatus: http.StatusBadRequest, wantCode: "INVALID_CSV"},
		{name: "Only header", csv: readingsHeader, wantStatus: http.StatusBadRequest, wantCode: "EMPTY_CSV"},
		{name: "Database error is 500", csv: validCSV, repositoryErr: errors.New("connection refused"), wantStatus: http.StatusInternalServerError, wantRepoCalled: true},
	}

	for _, tableTest := range tests {
		t.Run(tableTest.name, func(t *testing.T) {
			repository := &fakeRepository{existing: tableTest.existing, err: tableTest.repositoryErr}
			service := NewService(repository, time.UTC)

			result, err := service.ImportReadings(context.Background(), strings.NewReader(tableTest.csv))

			if repository.called != tableTest.wantRepoCalled {
				t.Fatalf("repository called = %v, want %v", repository.called, tableTest.wantRepoCalled)
			}

			if tableTest.wantStatus != 0 {
				appErr := apperror.From(err)

				if appErr.Status != tableTest.wantStatus || (tableTest.wantCode != "" && appErr.Code != tableTest.wantCode) {
					t.Fatalf("error = %d %s, want %d %s", appErr.Status, appErr.Code, tableTest.wantStatus, tableTest.wantCode)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result != tableTest.wantResult {
				t.Fatalf("result = %+v, want %+v", result, tableTest.wantResult)
			}
		})
	}
}
