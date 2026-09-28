package ingestion

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/santigonzalezla/biaenergy-test/backend/internal/modules/auth"
	"github.com/santigonzalezla/biaenergy-test/backend/internal/platform/token"
)

func TestHandlerListImports(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		wantStatus int
	}{
		{name: "Default limit", path: "/imports", wantStatus: http.StatusOK},
		{name: "Explicit limit", path: "/imports?limit=5", wantStatus: http.StatusOK},
		{name: "Non numeric limit", path: "/imports?limit=all", wantStatus: http.StatusBadRequest},
		{name: "Zero limit", path: "/imports?limit=0", wantStatus: http.StatusBadRequest},
	}

	for _, tableTest := range tests {
		t.Run(tableTest.name, func(t *testing.T) {
			router := chi.NewRouter()
			NewHandler(NewService(&fakeRepository{}, time.UTC)).RegisterRoutes(router)

			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, tableTest.path, nil))

			if recorder.Code != tableTest.wantStatus {
				t.Fatalf("status = %d, want %d (body: %s)", recorder.Code, tableTest.wantStatus, recorder.Body.String())
			}
		})
	}
}

// multipartBody construye un body multipart/form-data con un archivo en el campo indicado
func multipartBody(t *testing.T, field, content string) (*bytes.Buffer, string) {
	t.Helper()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile(field, "data.csv")
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}

	if _, err := part.Write([]byte(content)); err != nil {
		t.Fatalf("write form file: %v", err)
	}

	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	return body, writer.FormDataContentType()
}

func TestHandlerImport(t *testing.T) {
	validReadings := readingsHeader + "M-101,2026-09-01 00:00:00,23.5,221.9,101.28,0.954,OK\n"
	validEvents := "meter_id,event_timestamp,event_type,description\nM-104,2026-09-11 00:00,OPERATIONAL_CHANGE,New line\n"

	tests := []struct {
		name       string
		path       string
		field      string
		content    string
		wantStatus int
	}{
		{name: "Import readings", path: "/readings/import", field: "file", content: validReadings, wantStatus: http.StatusOK},
		{name: "Import events", path: "/events/import", field: "file", content: validEvents, wantStatus: http.StatusOK},
		{name: "Wrong form field", path: "/readings/import", field: "document", content: validReadings, wantStatus: http.StatusBadRequest},
		{name: "Invalid CSV content", path: "/readings/import", field: "file", content: "hello", wantStatus: http.StatusBadRequest},
	}

	for _, tableTest := range tests {
		t.Run(tableTest.name, func(t *testing.T) {
			router := chi.NewRouter()
			NewHandler(NewService(&fakeRepository{}, time.UTC)).RegisterRoutes(router)

			body, contentType := multipartBody(t, tableTest.field, tableTest.content)
			request := httptest.NewRequest(http.MethodPost, tableTest.path, body)
			request.Header.Set("Content-Type", contentType)
			recorder := httptest.NewRecorder()

			router.ServeHTTP(recorder, request)

			if recorder.Code != tableTest.wantStatus {
				t.Fatalf("status = %d, want %d (body: %s)", recorder.Code, tableTest.wantStatus, recorder.Body.String())
			}
		})
	}

	t.Run("Records the uploaded file name and the session user", func(t *testing.T) {
		repository := &fakeRepository{}
		router := chi.NewRouter()
		NewHandler(NewService(repository, time.UTC)).RegisterRoutes(router)

		userId := uuid.New()
		body, contentType := multipartBody(t, "file", validReadings)
		request := httptest.NewRequest(http.MethodPost, "/readings/import", body)
		request.Header.Set("Content-Type", contentType)
		request = request.WithContext(auth.WithClaims(request.Context(), token.Claims{UserId: userId}))

		router.ServeHTTP(httptest.NewRecorder(), request)

		if repository.savedBatch == nil || repository.savedBatch.FileName != "data.csv" {
			t.Fatalf("batch = %+v, want the uploaded file name", repository.savedBatch)
		}

		if repository.savedBatch.UserId == nil || *repository.savedBatch.UserId != userId {
			t.Fatalf("user = %v, want %s", repository.savedBatch.UserId, userId)
		}
	})

	t.Run("Request without multipart body", func(t *testing.T) {
		router := chi.NewRouter()
		NewHandler(NewService(&fakeRepository{}, time.UTC)).RegisterRoutes(router)

		request := httptest.NewRequest(http.MethodPost, "/readings/import", strings.NewReader(validReadings))
		recorder := httptest.NewRecorder()

		router.ServeHTTP(recorder, request)

		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", recorder.Code)
		}
	})
}
