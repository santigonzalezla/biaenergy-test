package dashboard

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/santigonzalezla/biaenergy-test/backend/internal/apperror"
	"github.com/santigonzalezla/biaenergy-test/backend/internal/db"
)

type fakeRepository struct {
	summary db.GetDashboardSummaryRow
	latest  *db.Analysis
	err     error
}

func (fake *fakeRepository) Summary(ctx context.Context) (db.GetDashboardSummaryRow, error) {
	return fake.summary, fake.err
}

func (fake *fakeRepository) LatestAnalysis(ctx context.Context) (db.Analysis, bool, error) {
	if fake.latest == nil {
		return db.Analysis{}, false, nil
	}

	return *fake.latest, true, nil
}

func datasetSummary() db.GetDashboardSummaryRow {
	return db.GetDashboardSummaryRow{
		MetersCount:           12,
		MetersWithAlerts:      2,
		ReadingsCount:         4032,
		TotalKwh:              155250.85,
		OpenAnomalies:         4,
		HighPriorityAnomalies: 2,
	}
}

func TestServiceSummary(t *testing.T) {
	t.Run("Maps every column to the response", func(t *testing.T) {
		service := NewService(&fakeRepository{summary: datasetSummary()})

		response, err := service.Summary(context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		want := SummaryResponse{
			MetersCount:           12,
			MetersWithAlerts:      2,
			ReadingsCount:         4032,
			TotalKwh:              155250.85,
			OpenAnomalies:         4,
			HighPriorityAnomalies: 2,
		}

		// Comparar el struct completo detecta un campo olvidado en toSummaryResponse (quedaría en 0)
		if response != want {
			t.Fatalf("response = %+v, want %+v", response, want)
		}
	})

	t.Run("Database error becomes 500", func(t *testing.T) {
		service := NewService(&fakeRepository{err: errors.New("connection refused")})

		_, err := service.Summary(context.Background())

		if got := apperror.From(err).Status; got != http.StatusInternalServerError {
			t.Fatalf("status = %d, want 500", got)
		}
	})
}

func TestSummaryIncludesTheLastAnalysis(t *testing.T) {
	finished := time.Date(2026, 9, 27, 20, 17, 31, 0, time.UTC)
	step := "completed"
	latest := db.Analysis{
		UidAnalysis:             uuid.New(),
		StrStatusAnalysis:       db.AnalysisStatusCOMPLETED,
		StrCurrentStepAnalysis:  &step,
		NumProgressAnalysis:     100,
		NumAnomaliesAnalysis:    4,
		NumHighPriorityAnalysis: 2,
		DtmFinishedAtAnalysis:   &finished,
	}
	service := NewService(&fakeRepository{summary: datasetSummary(), latest: &latest})

	response, err := service.Summary(context.Background())
	if err != nil {
		t.Fatalf("Summary() error = %v", err)
	}

	last := response.LastAnalysis
	if last == nil || last.Id != latest.UidAnalysis || last.Status != db.AnalysisStatusCOMPLETED {
		t.Fatalf("lastAnalysis = %+v", last)
	}
	if last.Anomalies != 4 || last.HighPriority != 2 || !last.FinishedAt.Equal(finished) {
		t.Fatalf("lastAnalysis = %+v, want 4 anomalies, 2 high priority and its finish time", last)
	}
}

func TestSummaryWithoutAnalysesHasNoLastAnalysis(t *testing.T) {
	response, err := NewService(&fakeRepository{summary: datasetSummary()}).Summary(context.Background())
	if err != nil {
		t.Fatalf("Summary() error = %v", err)
	}

	if response.LastAnalysis != nil {
		t.Fatalf("lastAnalysis = %+v, want nil before the first analysis", response.LastAnalysis)
	}
}
