package auth

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/santigonzalezla/biaenergy-test/backend/internal/platform/token"
)

type fakeVerifier struct {
	claims token.Claims
	err    error
	raw    string
}

func (fake *fakeVerifier) Verify(raw string) (token.Claims, error) {
	fake.raw = raw

	return fake.claims, fake.err
}

func protectedEndpoint(reached *token.Claims) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		claims, _ := ClaimsFrom(request.Context())
		*reached = claims
		writer.WriteHeader(http.StatusNoContent)
	})
}

func callProtected(middleware Middleware, authorization string) (*httptest.ResponseRecorder, token.Claims) {
	var reached token.Claims

	request := httptest.NewRequest(http.MethodGet, "/meters", nil)

	if authorization != "" {
		request.Header.Set("Authorization", authorization)
	}

	recorder := httptest.NewRecorder()
	middleware(protectedEndpoint(&reached)).ServeHTTP(recorder, request)

	return recorder, reached
}

func TestRequireAuthRejectsMissingCredentials(t *testing.T) {
	tests := []struct {
		name          string
		authorization string
	}{
		{name: "No Authorization header", authorization: ""},
		{name: "Another scheme", authorization: "Basic YWRtaW46c2VjcmV0"},
		{name: "Scheme without token", authorization: "Bearer"},
		{name: "Scheme with blank token", authorization: "Bearer    "},
		{name: "Token without scheme", authorization: "eyJhbGciOiJIUzI1NiJ9.e30.sig"},
	}

	for _, tableTest := range tests {
		t.Run(tableTest.name, func(t *testing.T) {
			verifier := &fakeVerifier{}

			recorder, reached := callProtected(RequireAuth(verifier), tableTest.authorization)

			assertErrorCode(t, recorder, http.StatusUnauthorized, "MISSING_TOKEN")

			if got := recorder.Header().Get("WWW-Authenticate"); got != `Bearer realm="biaenergy"` {
				t.Fatalf("WWW-Authenticate = %q", got)
			}

			if verifier.raw != "" || reached.UserId != uuid.Nil {
				t.Fatal("the request went past the middleware")
			}
		})
	}
}

func TestRequireAuthRejectsBadTokens(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		wantCode string
	}{
		{name: "Expired token", err: token.ErrExpiredToken, wantCode: "TOKEN_EXPIRED"},
		{name: "Invalid token", err: token.ErrInvalidToken, wantCode: "INVALID_TOKEN"},
		{name: "Unexpected verifier error", err: errors.New("boom"), wantCode: "INVALID_TOKEN"},
	}

	for _, tableTest := range tests {
		t.Run(tableTest.name, func(t *testing.T) {
			recorder, reached := callProtected(RequireAuth(&fakeVerifier{err: tableTest.err}), "Bearer some.jwt.token")

			assertErrorCode(t, recorder, http.StatusUnauthorized, tableTest.wantCode)

			if got := recorder.Header().Get("WWW-Authenticate"); !strings.Contains(got, `error="invalid_token"`) {
				t.Fatalf("WWW-Authenticate = %q, want an invalid_token error", got)
			}

			if reached.UserId != uuid.Nil {
				t.Fatal("the request went past the middleware")
			}
		})
	}
}

func TestRequireAuthPassesClaimsToTheHandler(t *testing.T) {
	claims := token.Claims{UserId: uuid.New(), Email: "admin@bia.app"}
	verifier := &fakeVerifier{claims: claims}

	recorder, reached := callProtected(RequireAuth(verifier), "  bearer   some.jwt.token  ")

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}

	if verifier.raw != "some.jwt.token" {
		t.Fatalf("verified %q, want the trimmed token", verifier.raw)
	}

	if reached != claims {
		t.Fatalf("handler claims = %+v, want %+v", reached, claims)
	}
}

func TestRequireAuthWithRealTokens(t *testing.T) {
	issuer := token.NewIssuer("a-test-secret-with-at-least-32-characters", time.Hour)
	userId := uuid.New()

	signed, _, err := issuer.Issue(userId, "admin@bia.app")
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}

	recorder, reached := callProtected(RequireAuth(issuer), "Bearer "+signed)

	if recorder.Code != http.StatusNoContent || reached.UserId != userId {
		t.Fatalf("status = %d, claims = %+v", recorder.Code, reached)
	}

	tampered := signed[:len(signed)-2] + "xx"
	recorder, _ = callProtected(RequireAuth(issuer), "Bearer "+tampered)

	assertErrorCode(t, recorder, http.StatusUnauthorized, "INVALID_TOKEN")
}
