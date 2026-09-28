package ingestion

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/santigonzalezla/biaenergy-test/backend/internal/apperror"
	"github.com/santigonzalezla/biaenergy-test/backend/internal/db"
)

type fakeRepository struct {
	existing    int64
	restored    int
	err         error
	failureErr  error
	previous    *time.Time
	called      bool
	savedBatch  *NewBatch
	failures    []Failure
	failedBatch *NewBatch
	batches     []db.ListImportBatchesRow
	listLimit   int32
}

func (fake *fakeRepository) SaveReadings(ctx context.Context, batch NewBatch, rows []ReadingRow) (Stats, error) {
	fake.called = true
	fake.savedBatch = &batch

	return Stats{BatchId: uuid.New(), Inserted: int64(len(rows)) - fake.existing, MetersCreated: 1, MetersRestored: fake.restored}, fake.err
}

func (fake *fakeRepository) SaveEvents(ctx context.Context, batch NewBatch, rows []EventRow) (Stats, error) {
	fake.called = true
	fake.savedBatch = &batch

	return Stats{BatchId: uuid.New(), Inserted: int64(len(rows)) - fake.existing, MetersRestored: fake.restored}, fake.err
}

func (fake *fakeRepository) RecordFailure(ctx context.Context, batch NewBatch, failure Failure) error {
	fake.failedBatch = &batch
	fake.failures = append(fake.failures, failure)

	return fake.failureErr
}

func (fake *fakeRepository) PreviousImport(ctx context.Context, kind db.ImportKind, checksum string) (*time.Time, error) {
	return fake.previous, nil
}

func (fake *fakeRepository) ListBatches(ctx context.Context, limit int32) ([]db.ListImportBatchesRow, error) {
	fake.listLimit = limit

	return fake.batches, fake.err
}

func uploadOf(content string) Upload {
	return Upload{FileName: "data.csv", Size: int64(len(content)), Content: strings.NewReader(content)}
}

func checksumOf(content string) string {
	sum := sha256.Sum256([]byte(content))

	return hex.EncodeToString(sum[:])
}

func assertAppError(t *testing.T, err error, status int, code string) {
	t.Helper()

	appErr := apperror.From(err)

	if appErr.Status != status || appErr.Code != code {
		t.Fatalf("error = %d %s (%v), want %d %s", appErr.Status, appErr.Code, err, status, code)
	}
}

func TestServiceImportReadings(t *testing.T) {
	validCSV := readingsHeader +
		"M-101,2026-09-01 00:00:00,23.5,221.9,101.28,0.954,OK\n" +
		"M-101,2026-09-01 01:00:00,20.1,221.1,100.49,0.935,OK\n"

	tests := []struct {
		name           string
		csv            string
		existing       int64
		restored       int
		repositoryErr  error
		wantStatus     int
		wantCode       string
		wantRepoCalled bool
		wantFailure    bool
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
		{
			name:           "Restored meters are reported",
			csv:            validCSV,
			restored:       1,
			wantRepoCalled: true,
			wantResult:     ImportResult{Rows: 2, Inserted: 2, MetersCreated: 1, MetersRestored: 1},
		},
		{name: "Invalid row rejects whole file", csv: validCSV + "M-101,2026-09-01 02:00:00,abc,220,1,0.9,OK\n", wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR", wantFailure: true},
		{name: "Missing columns", csv: "meter_id\nM-101\n", wantStatus: http.StatusBadRequest, wantCode: "INVALID_CSV", wantFailure: true},
		{name: "Only header", csv: readingsHeader, wantStatus: http.StatusBadRequest, wantCode: "EMPTY_CSV", wantFailure: true},
		{name: "Database error is 500", csv: validCSV, repositoryErr: errors.New("connection refused"), wantStatus: http.StatusInternalServerError, wantCode: "INTERNAL_ERROR", wantRepoCalled: true},
	}

	for _, tableTest := range tests {
		t.Run(tableTest.name, func(t *testing.T) {
			repository := &fakeRepository{existing: tableTest.existing, restored: tableTest.restored, err: tableTest.repositoryErr}
			service := NewService(repository, time.UTC)

			result, err := service.ImportReadings(context.Background(), uploadOf(tableTest.csv))

			if repository.called != tableTest.wantRepoCalled {
				t.Fatalf("repository called = %v, want %v", repository.called, tableTest.wantRepoCalled)
			}

			if (len(repository.failures) > 0) != tableTest.wantFailure {
				t.Fatalf("failures recorded = %v, want %v", repository.failures, tableTest.wantFailure)
			}

			if tableTest.wantStatus != 0 {
				assertAppError(t, err, tableTest.wantStatus, tableTest.wantCode)
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			result.BatchId = uuid.Nil

			if result != tableTest.wantResult {
				t.Fatalf("result = %+v, want %+v", result, tableTest.wantResult)
			}
		})
	}
}

func TestServiceImportEvents(t *testing.T) {
	validCSV := "meter_id,event_timestamp,event_type,description\n" +
		"M-104,2026-09-11 00:00,OPERATIONAL_CHANGE,New production line activated\n" +
		"M-106,2026-09-08 00:00,SCHEDULED_OUTAGE,Scheduled maintenance outage for 12 hours\n"

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
			name:           "First import inserts every event",
			csv:            validCSV,
			wantRepoCalled: true,
			wantResult:     ImportResult{Rows: 2, Inserted: 2, Skipped: 0},
		},
		{
			name:           "Re-import skips existing events",
			csv:            validCSV,
			existing:       2,
			wantRepoCalled: true,
			wantResult:     ImportResult{Rows: 2, Inserted: 0, Skipped: 2},
		},
		{name: "Unknown event type rejects whole file", csv: validCSV + "M-109,2026-09-12 14:00,EARTHQUAKE,?\n", wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR"},
		{name: "Missing columns", csv: "meter_id,event_type\nM-104,OPERATIONAL_CHANGE\n", wantStatus: http.StatusBadRequest, wantCode: "INVALID_CSV"},
		{name: "Database error is 500", csv: validCSV, repositoryErr: errors.New("connection refused"), wantStatus: http.StatusInternalServerError, wantCode: "INTERNAL_ERROR", wantRepoCalled: true},
	}

	for _, tableTest := range tests {
		t.Run(tableTest.name, func(t *testing.T) {
			repository := &fakeRepository{existing: tableTest.existing, err: tableTest.repositoryErr}
			service := NewService(repository, time.UTC)

			result, err := service.ImportEvents(context.Background(), uploadOf(tableTest.csv))

			if repository.called != tableTest.wantRepoCalled {
				t.Fatalf("repository called = %v, want %v", repository.called, tableTest.wantRepoCalled)
			}

			if tableTest.wantStatus != 0 {
				assertAppError(t, err, tableTest.wantStatus, tableTest.wantCode)
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if repository.savedBatch.Kind != db.ImportKindEVENTS {
				t.Fatalf("batch kind = %s, want EVENTS", repository.savedBatch.Kind)
			}

			result.BatchId = uuid.Nil

			if result != tableTest.wantResult {
				t.Fatalf("result = %+v, want %+v", result, tableTest.wantResult)
			}
		})
	}
}

func TestServiceDescribesTheBatch(t *testing.T) {
	content := readingsHeader + "M-101,2026-09-01 00:00:00,23.5,221.9,101.28,0.954,OK\n"
	userId := uuid.New()
	repository := &fakeRepository{}

	_, err := NewService(repository, time.UTC).ImportReadings(context.Background(), Upload{
		FileName: "septiembre.csv",
		Content:  strings.NewReader(content),
		UserId:   &userId,
	})
	if err != nil {
		t.Fatalf("ImportReadings() error = %v", err)
	}

	batch := repository.savedBatch

	if batch.Kind != db.ImportKindREADINGS || batch.FileName != "septiembre.csv" || batch.Rows != 1 {
		t.Fatalf("batch = %+v", batch)
	}

	if batch.Checksum != checksumOf(content) || batch.FileSize != int64(len(content)) {
		t.Fatalf("checksum/size = %s/%d, want the SHA-256 and size of the content", batch.Checksum, batch.FileSize)
	}

	if batch.UserId == nil || *batch.UserId != userId {
		t.Fatalf("user = %v, want %s", batch.UserId, userId)
	}
}

func TestServiceWarnsAboutAPreviousImport(t *testing.T) {
	importedAt := time.Date(2026, 9, 27, 22, 0, 0, 0, time.UTC)
	repository := &fakeRepository{previous: &importedAt}
	content := readingsHeader + "M-101,2026-09-01 00:00:00,23.5,221.9,101.28,0.954,OK\n"

	result, err := NewService(repository, time.UTC).ImportReadings(context.Background(), uploadOf(content))
	if err != nil {
		t.Fatalf("ImportReadings() error = %v", err)
	}

	if result.PreviouslyImportedAt == nil || !result.PreviouslyImportedAt.Equal(importedAt) {
		t.Fatalf("previouslyImportedAt = %v, want %v", result.PreviouslyImportedAt, importedAt)
	}

	if !repository.called {
		t.Fatal("a previous import must not block re-importing: it is idempotent")
	}
}

func TestServiceRecordsFailedImports(t *testing.T) {
	t.Run("Invalid rows are summarized", func(t *testing.T) {
		repository := &fakeRepository{}
		content := readingsHeader + "M-101,2026-09-01 00:00:00,23.5,221.9,101.28,1.4,OK\n"

		_, err := NewService(repository, time.UTC).ImportReadings(context.Background(), uploadOf(content))

		assertAppError(t, err, http.StatusBadRequest, "VALIDATION_ERROR")

		failure := repository.failures[0]

		if failure.Code != "VALIDATION_ERROR" || !strings.Contains(failure.Message, "1 invalid rows; first at line 2, column power_factor") {
			t.Fatalf("failure = %+v", failure)
		}

		if repository.failedBatch.Checksum != checksumOf(content) {
			t.Fatal("the failed batch must keep the checksum of the rejected file")
		}
	})

	t.Run("The original error wins if recording the failure fails", func(t *testing.T) {
		repository := &fakeRepository{failureErr: errors.New("connection refused")}

		_, err := NewService(repository, time.UTC).ImportReadings(context.Background(), uploadOf(readingsHeader))

		assertAppError(t, err, http.StatusBadRequest, "EMPTY_CSV")
	})
}

func TestServiceList(t *testing.T) {
	tests := []struct {
		name      string
		limit     int
		wantLimit int32
	}{
		{name: "Default limit", limit: 0, wantLimit: 20},
		{name: "Explicit limit", limit: 5, wantLimit: 5},
		{name: "Limit is capped", limit: 500, wantLimit: 100},
	}

	for _, tableTest := range tests {
		t.Run(tableTest.name, func(t *testing.T) {
			repository := &fakeRepository{batches: []db.ListImportBatchesRow{{UidImportBatch: uuid.New(), StrFileNameImportBatch: "readings.csv"}}}

			response, err := NewService(repository, time.UTC).List(context.Background(), tableTest.limit)
			if err != nil {
				t.Fatalf("List() error = %v", err)
			}

			if repository.listLimit != tableTest.wantLimit || len(response.Data) != 1 || response.Data[0].FileName != "readings.csv" {
				t.Fatalf("limit = %d, response = %+v", repository.listLimit, response)
			}
		})
	}
}
