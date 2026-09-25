package httpserver

import (
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/santigonzalezla/biaenergy-test/backend/internal/apperror"
)

func RequestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		start := time.Now()
		requestID := middleware.GetReqID(request.Context())
		writer.Header().Set(middleware.RequestIDHeader, requestID)
		wrapped := middleware.NewWrapResponseWriter(writer, request.ProtoMajor)

		next.ServeHTTP(wrapped, request)

		slog.InfoContext(request.Context(), "request completed",
			"method", request.Method,
			"path", request.URL.Path,
			"status", wrapped.Status(),
			"duration_ms", time.Since(start).Milliseconds(),
			"request_id", requestID,
			"client_ip", middleware.GetClientIP(request.Context()),
		)
	})
}

func Recover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				if rec == http.ErrAbortHandler {
					panic(rec)
				}

				slog.ErrorContext(request.Context(), "panic recovered",
					"panic", rec,
					"stack", string(debug.Stack()),
				)

				WriteError(writer, request, apperror.Internal(fmt.Errorf("panic: %v", rec)))
			}
		}()

		next.ServeHTTP(writer, request)
	})
}
