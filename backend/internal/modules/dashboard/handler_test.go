package dashboard

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestHandlerSummary(t *testing.T) {
	tests := []struct {
		name          string
		method        string
		repositoryErr error
		wantStatus    int
	}{
		{name: "Returns KPIs", method: http.MethodGet, wantStatus: http.StatusOK},
		{name: "Database down", method: http.MethodGet, repositoryErr: errors.New("connection refused"), wantStatus: http.StatusInternalServerError},
		{name: "Wrong method", method: http.MethodPost, wantStatus: http.StatusMethodNotAllowed},
	}

	for _, tableTest := range tests {
		t.Run(tableTest.name, func(t *testing.T) {
			router := chi.NewRouter()
			NewHandler(NewService(&fakeRepository{summary: datasetSummary(), err: tableTest.repositoryErr})).RegisterRoutes(router)

			request := httptest.NewRequest(tableTest.method, "/dashboard/summary", nil)
			recorder := httptest.NewRecorder()

			router.ServeHTTP(recorder, request)

			if recorder.Code != tableTest.wantStatus {
				t.Fatalf("status = %d, want %d (body: %s)", recorder.Code, tableTest.wantStatus, recorder.Body.String())
			}

			if tableTest.wantStatus != http.StatusOK {
				return
			}

			// El contrato JSON: nombres en camelCase que consume el frontend
			var body map[string]any
			if err := json.NewDecoder(recorder.Body).Decode(&body); err != nil {
				t.Fatalf("invalid JSON: %v", err)
			}

			for _, key := range []string{"metersCount", "metersWithAlerts", "readingsCount", "totalKwh", "openAnomalies", "highPriorityAnomalies"} {
				if _, ok := body[key]; !ok {
					t.Fatalf("missing key %q in %v", key, body)
				}
			}
		})
	}
}
