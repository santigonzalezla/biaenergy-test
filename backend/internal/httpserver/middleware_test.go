package httpserver

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5/middleware"
)

func TestMiddlewares(t *testing.T) {
	tests := []struct {
		name       string
		handler    http.HandlerFunc
		wantStatus int
		wantCode   string
	}{
		{
			name: "Successful request",
			handler: func(writer http.ResponseWriter, request *http.Request) {
				WriteJSON(writer, http.StatusOK, map[string]string{"status": "ok"})
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "Panic becomes JSON 500",
			handler: func(writer http.ResponseWriter, request *http.Request) {
				panic("unexpected bug")
			},
			wantStatus: http.StatusInternalServerError,
			wantCode:   "INTERNAL_ERROR",
		},
	}

	for _, tableTest := range tests {
		t.Run(tableTest.name, func(t *testing.T) {
			handler := middleware.RequestID(RequestLogger(Recover(tableTest.handler)))

			request := httptest.NewRequest(http.MethodGet, "/test", nil)
			recorder := httptest.NewRecorder()

			handler.ServeHTTP(recorder, request)

			if recorder.Code != tableTest.wantStatus {
				t.Fatalf("status = %d, want %d", recorder.Code, tableTest.wantStatus)
			}

			if recorder.Header().Get(middleware.RequestIDHeader) == "" {
				t.Fatal("X-Request-Id header is missing")
			}

			if tableTest.wantCode != "" {
				var body ErrorBody

				if err := json.NewDecoder(recorder.Body).Decode(&body); err != nil {
					t.Fatalf("invalid JSON body: %v", err)
				}

				if body.Error.Code != tableTest.wantCode {
					t.Fatalf("code = %q, want %q", body.Error.Code, tableTest.wantCode)
				}
			}
		})
	}
}
