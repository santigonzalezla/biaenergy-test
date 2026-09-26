package dashboard

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/santigonzalezla/biaenergy-test/backend/internal/apperror"
	"github.com/santigonzalezla/biaenergy-test/backend/internal/db"
)

// fakeRepository devuelve la fila y el error que configure cada test
type fakeRepository struct {
	summary db.GetDashboardSummaryRow
	err     error
}

func (fake *fakeRepository) Summary(ctx context.Context) (db.GetDashboardSummaryRow, error) {
	return fake.summary, fake.err
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
