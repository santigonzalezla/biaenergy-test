package auth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/santigonzalezla/biaenergy-test/backend/internal/apperror"
	"github.com/santigonzalezla/biaenergy-test/backend/internal/db"
	"github.com/santigonzalezla/biaenergy-test/backend/internal/httpserver"
	"github.com/santigonzalezla/biaenergy-test/backend/internal/platform/token"
)

type errorBody struct {
	Error struct {
		Code string `json:"code"`
	} `json:"error"`
}

func fakeRequireAuth(user db.AppUser) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			if request.Header.Get("Authorization") != "Bearer valid" {
				httpserver.WriteError(writer, request, apperror.Unauthorized("MISSING_TOKEN", "Authentication is required"))
				return
			}

			claims := token.Claims{UserId: user.UidUser, Email: user.StrEmailUser}
			next.ServeHTTP(writer, request.WithContext(WithClaims(request.Context(), claims)))
		})
	}
}

func passThrough(next http.Handler) http.Handler {
	return next
}

func newTestRouter(t *testing.T, requireAuth func(db.AppUser) Middleware) (http.Handler, db.AppUser) {
	t.Helper()

	admin := newAdmin(t)
	service := NewService(&fakeRepository{users: map[string]db.AppUser{admin.StrEmailUser: admin}}, &fakeIssuer{})

	router := chi.NewRouter()
	NewHandler(service, requireAuth(admin)).RegisterRoutes(router)

	return router, admin
}

func serve(router http.Handler, method, path, body, authorization string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")

	if authorization != "" {
		request.Header.Set("Authorization", authorization)
	}

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	return recorder
}

func assertErrorCode(t *testing.T, recorder *httptest.ResponseRecorder, status int, code string) {
	t.Helper()

	var body errorBody

	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("body is not JSON: %s", recorder.Body.String())
	}

	if recorder.Code != status || body.Error.Code != code {
		t.Fatalf("response = %d %s, want %d %s", recorder.Code, body.Error.Code, status, code)
	}
}

func TestLoginEndpoint(t *testing.T) {
	router, admin := newTestRouter(t, fakeRequireAuth)

	t.Run("Valid credentials return 200 with a non cacheable token", func(t *testing.T) {
		recorder := serve(router, http.MethodPost, "/auth/login", `{"email":"admin@bia.app","password":"`+testPassword+`"}`, "")

		if recorder.Code != http.StatusOK {
			t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
		}

		if got := recorder.Header().Get("Cache-Control"); got != "no-store" {
			t.Fatalf("Cache-Control = %q, want no-store", got)
		}

		var response LoginResponse

		if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
			t.Fatalf("body is not a LoginResponse: %v", err)
		}

		if response.Token == "" || response.TokenType != "Bearer" || response.User.Id != admin.UidUser {
			t.Fatalf("response = %+v", response)
		}

		if strings.Contains(recorder.Body.String(), admin.StrPasswordUser) {
			t.Fatal("response leaks the password hash")
		}
	})

	t.Run("Wrong password returns 401", func(t *testing.T) {
		recorder := serve(router, http.MethodPost, "/auth/login", `{"email":"admin@bia.app","password":"wrong"}`, "")

		assertErrorCode(t, recorder, http.StatusUnauthorized, "INVALID_CREDENTIALS")
	})

	t.Run("Unknown JSON fields return 400", func(t *testing.T) {
		recorder := serve(router, http.MethodPost, "/auth/login", `{"email":"admin@bia.app","password":"x","role":"admin"}`, "")

		assertErrorCode(t, recorder, http.StatusBadRequest, "INVALID_BODY")
	})
}

func TestMeEndpoint(t *testing.T) {
	t.Run("Authenticated request returns the current user", func(t *testing.T) {
		router, admin := newTestRouter(t, fakeRequireAuth)

		recorder := serve(router, http.MethodGet, "/auth/me", "", "Bearer valid")

		if recorder.Code != http.StatusOK {
			t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
		}

		var user UserResponse

		if err := json.Unmarshal(recorder.Body.Bytes(), &user); err != nil || user.Id != admin.UidUser {
			t.Fatalf("user = %+v, err = %v", user, err)
		}
	})

	t.Run("The middleware protects the route", func(t *testing.T) {
		router, _ := newTestRouter(t, fakeRequireAuth)

		recorder := serve(router, http.MethodGet, "/auth/me", "", "")

		assertErrorCode(t, recorder, http.StatusUnauthorized, "MISSING_TOKEN")
	})

	t.Run("Missing claims in the context return 401 instead of panicking", func(t *testing.T) {
		router, _ := newTestRouter(t, func(db.AppUser) Middleware { return passThrough })

		recorder := serve(router, http.MethodGet, "/auth/me", "", "")

		assertErrorCode(t, recorder, http.StatusUnauthorized, "MISSING_TOKEN")
	})
}
