package meter

import (
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

func TestHandler(t *testing.T) {
	existingID := uuid.New()

	tests := []struct {
		name          string
		method        string
		path          string
		body          string
		repositoryErr error
		wantStatus    int
		wantCode      string
	}{
		{name: "List meters", method: http.MethodGet, path: "/meters?page=1&limit=10&sortDir=desc", wantStatus: http.StatusOK},
		{name: "List with non numeric page", method: http.MethodGet, path: "/meters?page=abc", wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR"},
		{name: "List with invalid sortDir", method: http.MethodGet, path: "/meters?sortDir=up", wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR"},
		{name: "Get meter", method: http.MethodGet, path: "/meters/" + existingID.String(), wantStatus: http.StatusOK},
		{name: "Get with invalid UUID", method: http.MethodGet, path: "/meters/not-a-uuid", wantStatus: http.StatusBadRequest, wantCode: "INVALID_METER_ID"},
		{name: "Get missing meter", method: http.MethodGet, path: "/meters/" + uuid.NewString(), repositoryErr: ErrNotFound, wantStatus: http.StatusNotFound, wantCode: "METER_NOT_FOUND"},
		{name: "Create meter", method: http.MethodPost, path: "/meters", body: `{"code":"M-113","name":"Compresor 3"}`, wantStatus: http.StatusCreated},
		{name: "Create with unknown field", method: http.MethodPost, path: "/meters", body: `{"code":"M-113","name":"X","hack":true}`, wantStatus: http.StatusBadRequest, wantCode: "INVALID_BODY"},
		{name: "Create with malformed JSON", method: http.MethodPost, path: "/meters", body: `{"code":`, wantStatus: http.StatusBadRequest, wantCode: "INVALID_BODY"},
		{name: "Update meter", method: http.MethodPatch, path: "/meters/" + existingID.String(), body: `{"status":"ALERT"}`, wantStatus: http.StatusOK},
		{name: "Update with invalid status", method: http.MethodPatch, path: "/meters/" + existingID.String(), body: `{"status":"BROKEN"}`, wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR"},
		{name: "Delete meter", method: http.MethodDelete, path: "/meters/" + existingID.String(), wantStatus: http.StatusNoContent},
		{name: "Delete missing meter", method: http.MethodDelete, path: "/meters/" + uuid.NewString(), repositoryErr: ErrNotFound, wantStatus: http.StatusNotFound, wantCode: "METER_NOT_FOUND"},
	}

	for _, tableTest := range tests {
		t.Run(tableTest.name, func(t *testing.T) {
			repository := &fakeRepository{
				err:   tableTest.repositoryErr,
				meter: db.Meter{UidMeter: existingID, StrCodeMeter: "M-113", StrStatusMeter: db.MeterStatusOK},
			}

			router := chi.NewRouter()
			NewHandler(NewService(repository)).RegisterRoutes(router)

			request := httptest.NewRequest(tableTest.method, tableTest.path, strings.NewReader(tableTest.body))
			recorder := httptest.NewRecorder()

			router.ServeHTTP(recorder, request)

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

			if tableTest.wantStatus == http.StatusCreated && recorder.Header().Get("Location") == "" {
				t.Fatal("Location header is missing on 201")
			}
		})
	}
}

func TestHandlerSeries(t *testing.T) {
	meterID := uuid.NewString()

	tests := []struct {
		name          string
		path          string
		repositoryErr error
		wantStatus    int
		wantCode      string
	}{
		{name: "Readings with default range", path: "/meters/" + meterID + "/readings", wantStatus: http.StatusOK},
		{name: "Readings with explicit range", path: "/meters/" + meterID + "/readings?from=2026-09-12T00:00:00-05:00&to=2026-09-13T00:00:00-05:00", wantStatus: http.StatusOK},
		{name: "Readings with UTC dates", path: "/meters/" + meterID + "/readings?from=2026-09-12T05:00:00Z", wantStatus: http.StatusOK},
		{name: "Readings with non RFC3339 date", path: "/meters/" + meterID + "/readings?from=2026-09-12", wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR"},
		{name: "Readings with inverted range", path: "/meters/" + meterID + "/readings?from=2026-09-13T00:00:00Z&to=2026-09-12T00:00:00Z", wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR"},
		{name: "Readings with invalid UUID", path: "/meters/abc/readings", wantStatus: http.StatusBadRequest, wantCode: "INVALID_METER_ID"},
		{name: "Readings of missing meter", path: "/meters/" + meterID + "/readings", repositoryErr: ErrNotFound, wantStatus: http.StatusNotFound, wantCode: "METER_NOT_FOUND"},
		{name: "Events with default range", path: "/meters/" + meterID + "/events", wantStatus: http.StatusOK},
		{name: "Events of missing meter", path: "/meters/" + meterID + "/events", repositoryErr: ErrNotFound, wantStatus: http.StatusNotFound, wantCode: "METER_NOT_FOUND"},
	}

	for _, tableTest := range tests {
		t.Run(tableTest.name, func(t *testing.T) {
			repository := &fakeRepository{err: tableTest.repositoryErr, hasReadings: true, latestReading: time.Date(2026, 9, 15, 4, 0, 0, 0, time.UTC)}

			router := chi.NewRouter()
			NewHandler(NewService(repository)).RegisterRoutes(router)

			request := httptest.NewRequest(http.MethodGet, tableTest.path, nil)
			recorder := httptest.NewRecorder()

			router.ServeHTTP(recorder, request)

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
				return
			}

			var body struct {
				From time.Time        `json:"from"`
				To   time.Time        `json:"to"`
				Data *json.RawMessage `json:"data"`
			}

			if err := json.NewDecoder(recorder.Body).Decode(&body); err != nil {
				t.Fatalf("invalid series body: %v", err)
			}

			if body.From.IsZero() || body.To.IsZero() || body.Data == nil || string(*body.Data) == "null" {
				t.Fatalf("series response must include from, to and data array: %+v", body)
			}
		})
	}
}
