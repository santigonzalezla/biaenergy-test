package httpserver

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/santigonzalezla/biaenergy-test/backend/internal/apperror"
)

const frontendOrigin = "http://localhost:5173"

type fakeModule struct {
	path string
}

func (module fakeModule) RegisterRoutes(route chi.Router) {
	route.Get(module.path, func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(http.StatusNoContent)
	})
}

func fakeRequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Header.Get("Authorization") != "Bearer valid" {
			WriteError(writer, request, apperror.Unauthorized("MISSING_TOKEN", "Authentication is required"))
			return
		}

		next.ServeHTTP(writer, request)
	})
}

func newTestRouter() http.Handler {
	return NewRouter([]string{frontendOrigin}, Routes{
		Public:      []RouteRegister{fakeModule{path: "/health"}},
		Protected:   []RouteRegister{fakeModule{path: "/meters"}},
		RequireAuth: fakeRequireAuth,
	})
}

func TestRouterSeparatesPublicAndProtectedRoutes(t *testing.T) {
	tests := []struct {
		name          string
		method        string
		path          string
		authorization string
		wantStatus    int
	}{
		{name: "Public route without token", method: http.MethodGet, path: "/api/health", wantStatus: http.StatusNoContent},
		{name: "Protected route without token", method: http.MethodGet, path: "/api/meters", wantStatus: http.StatusUnauthorized},
		{name: "Protected route with token", method: http.MethodGet, path: "/api/meters", authorization: "Bearer valid", wantStatus: http.StatusNoContent},
		{name: "Unknown route", method: http.MethodGet, path: "/api/nope", authorization: "Bearer valid", wantStatus: http.StatusNotFound},
	}

	router := newTestRouter()

	for _, tableTest := range tests {
		t.Run(tableTest.name, func(t *testing.T) {
			request := httptest.NewRequest(tableTest.method, tableTest.path, nil)

			if tableTest.authorization != "" {
				request.Header.Set("Authorization", tableTest.authorization)
			}

			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, request)

			if recorder.Code != tableTest.wantStatus {
				t.Fatalf("status = %d, want %d (body %s)", recorder.Code, tableTest.wantStatus, recorder.Body.String())
			}
		})
	}
}

func TestRouterAnswersPreflightWithoutToken(t *testing.T) {
	request := httptest.NewRequest(http.MethodOptions, "/api/meters", nil)
	request.Header.Set("Origin", frontendOrigin)
	request.Header.Set("Access-Control-Request-Method", http.MethodGet)
	request.Header.Set("Access-Control-Request-Headers", "Authorization")

	recorder := httptest.NewRecorder()
	newTestRouter().ServeHTTP(recorder, request)

	if recorder.Code == http.StatusUnauthorized {
		t.Fatal("preflight was blocked by the auth middleware")
	}

	if got := recorder.Header().Get("Access-Control-Allow-Origin"); got != frontendOrigin {
		t.Fatalf("Access-Control-Allow-Origin = %q, want %q", got, frontendOrigin)
	}
}

func TestRouterRefusesProtectedRoutesWithoutMiddleware(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("NewRouter() did not panic, want protected routes to fail closed")
		}
	}()

	NewRouter(nil, Routes{Protected: []RouteRegister{fakeModule{path: "/meters"}}})
}
