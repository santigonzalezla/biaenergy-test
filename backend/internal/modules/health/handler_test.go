package health

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

type fakePinger struct {
	err error
}

func (f fakePinger) Ping(ctx context.Context) error {
	return f.err
}

func TestHealthCheck(t *testing.T) {
	tests := []struct {
		name       string
		pingErr    error
		wantStatus int
	}{
		{name: "available database", pingErr: nil, wantStatus: http.StatusOK},
		{name: "down database", pingErr: errors.New("connection refused"), wantStatus: http.StatusServiceUnavailable},
	}

	for _, tableTest := range tests {
		t.Run(tableTest.name, func(t *testing.T) {
			router := chi.NewRouter()
			NewHandler(fakePinger{err: tableTest.pingErr}).RegisterRoutes(router)

			request := httptest.NewRequest(http.MethodGet, "/health", nil)
			recorder := httptest.NewRecorder()

			router.ServeHTTP(recorder, request)

			if recorder.Code != tableTest.wantStatus {
				t.Fatalf("status = %d, want %d", recorder.Code, tableTest.wantStatus)
			}
		})
	}
}
