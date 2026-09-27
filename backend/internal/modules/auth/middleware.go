package auth

import (
	"errors"
	"net/http"
	"strings"

	"github.com/santigonzalezla/biaenergy-test/backend/internal/apperror"
	"github.com/santigonzalezla/biaenergy-test/backend/internal/httpserver"
	"github.com/santigonzalezla/biaenergy-test/backend/internal/platform/token"
)

type Verifier interface {
	Verify(raw string) (token.Claims, error)
}

func RequireAuth(verifier Verifier) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			raw, ok := bearerToken(request.Header.Get("Authorization"))

			if !ok {
				rejectRequest(writer, request, "", apperror.Unauthorized("MISSING_TOKEN", "Authentication is required"))
				return
			}

			claims, err := verifier.Verify(raw)

			if errors.Is(err, token.ErrExpiredToken) {
				rejectRequest(writer, request, "invalid_token", apperror.Unauthorized("TOKEN_EXPIRED", "The session has expired"))
				return
			}

			if err != nil {
				rejectRequest(writer, request, "invalid_token", apperror.Unauthorized("INVALID_TOKEN", "The session token is invalid").WithCause(err))
				return
			}

			next.ServeHTTP(writer, request.WithContext(WithClaims(request.Context(), claims)))
		})
	}
}

func bearerToken(header string) (string, bool) {
	scheme, raw, found := strings.Cut(strings.TrimSpace(header), " ")

	if !found || !strings.EqualFold(scheme, "Bearer") {
		return "", false
	}

	raw = strings.TrimSpace(raw)

	return raw, raw != ""
}

func rejectRequest(writer http.ResponseWriter, request *http.Request, reason string, err *apperror.AppError) {
	challenge := `Bearer realm="biaenergy"`

	if reason != "" {
		challenge += `, error="` + reason + `"`
	}

	writer.Header().Set("WWW-Authenticate", challenge)
	httpserver.WriteError(writer, request, err)
}
