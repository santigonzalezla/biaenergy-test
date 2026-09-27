package analysis

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/santigonzalezla/biaenergy-test/backend/internal/db"
	"github.com/santigonzalezla/biaenergy-test/backend/internal/httpserver"
)

func newRouter(t *testing.T, repository *fakeRepository) http.Handler {
	t.Helper()

	lifetime, cancel := context.WithCancel(context.Background())
	service := NewService(lifetime, repository, &fakeAnalyzer{}, "America/Bogota", time.Second)
	t.Cleanup(func() {
		cancel()
		service.Wait()
	})

	router := chi.NewRouter()
	NewHandler(service).RegisterRoutes(router)

	return router
}

func TestHandlerStart(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		active     bool
		wantStatus int
		wantReused bool
	}{
		{name: "Whole fleet without body", body: "", wantStatus: http.StatusAccepted},
		{name: "Selected meters", body: `{"meterIds": ["` + uuid.NewString() + `"]}`, wantStatus: http.StatusAccepted},
		{name: "Analysis already running", body: "", active: true, wantStatus: http.StatusAccepted, wantReused: true},
		{name: "Invalid meter id", body: `{"meterIds": ["abc"]}`, wantStatus: http.StatusBadRequest},
		{name: "Malformed JSON", body: `{"meterIds": `, wantStatus: http.StatusBadRequest},
	}

	for _, tableTest := range tests {
		t.Run(tableTest.name, func(t *testing.T) {
			repository := newFakeRepository()
			if tableTest.active {
				running := db.Analysis{UidAnalysis: uuid.New(), StrStatusAnalysis: db.AnalysisStatusRUNNING}
				repository.active = &running
			}

			request := httptest.NewRequest(http.MethodPost, "/ai/analyze", strings.NewReader(tableTest.body))
			recorder := httptest.NewRecorder()

			newRouter(t, repository).ServeHTTP(recorder, request)

			if recorder.Code != tableTest.wantStatus {
				t.Fatalf("status = %d, want %d (body: %s)", recorder.Code, tableTest.wantStatus, recorder.Body.String())
			}
			if tableTest.wantStatus != http.StatusAccepted {
				return
			}

			var body AnalysisResponse
			if err := json.NewDecoder(recorder.Body).Decode(&body); err != nil {
				t.Fatalf("invalid JSON: %v", err)
			}
			if recorder.Header().Get("Location") != "/api/ai/analysis/"+body.Id.String() {
				t.Fatalf("Location = %q", recorder.Header().Get("Location"))
			}
			if (recorder.Header().Get("X-Analysis-Reused") == "true") != tableTest.wantReused {
				t.Fatalf("X-Analysis-Reused = %q, want reused=%v", recorder.Header().Get("X-Analysis-Reused"), tableTest.wantReused)
			}
		})
	}
}

func TestHandlerQueries(t *testing.T) {
	repository := newFakeRepository()
	stored := db.Analysis{
		UidAnalysis:       uuid.New(),
		StrStatusAnalysis: db.AnalysisStatusCOMPLETED,
		DtmCreatedAt:      time.Now(),
	}
	repository.stored[stored.UidAnalysis] = stored

	tests := []struct {
		name       string
		path       string
		latest     bool
		wantStatus int
		wantCode   string
	}{
		{name: "Existing analysis", path: "/ai/analysis/" + stored.UidAnalysis.String(), wantStatus: http.StatusOK},
		{name: "Unknown analysis", path: "/ai/analysis/" + uuid.NewString(), wantStatus: http.StatusNotFound, wantCode: "ANALYSIS_NOT_FOUND"},
		{name: "Invalid analysis id", path: "/ai/analysis/abc", wantStatus: http.StatusBadRequest, wantCode: "INVALID_ANALYSIS_ID"},
		{name: "Latest without analyses", path: "/ai/analysis/latest", wantStatus: http.StatusNotFound, wantCode: "NO_ANALYSIS_YET"},
		{name: "Latest analysis", path: "/ai/analysis/latest", latest: true, wantStatus: http.StatusOK},
	}

	for _, tableTest := range tests {
		t.Run(tableTest.name, func(t *testing.T) {
			repository.latest = nil
			if tableTest.latest {
				repository.latest = &stored
			}

			request := httptest.NewRequest(http.MethodGet, tableTest.path, nil)
			recorder := httptest.NewRecorder()

			newRouter(t, repository).ServeHTTP(recorder, request)

			if recorder.Code != tableTest.wantStatus {
				t.Fatalf("status = %d, want %d (body: %s)", recorder.Code, tableTest.wantStatus, recorder.Body.String())
			}

			if tableTest.wantCode != "" {
				var body httpserver.ErrorBody
				if err := json.NewDecoder(recorder.Body).Decode(&body); err != nil {
					t.Fatalf("invalid error body: %v", err)
				}
				if body.Error.Code != tableTest.wantCode {
					t.Fatalf("code = %q, want %q", body.Error.Code, tableTest.wantCode)
				}
			}
		})
	}
}
