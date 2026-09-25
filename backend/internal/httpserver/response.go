package httpserver

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/santigonzalezla/biaenergy-test/backend/internal/apperror"
)

const maxBodyBytes = 1 << 20

type ErrorDetail struct {
	Code      string    `json:"code"`
	Message   string    `json:"message"`
	Details   any       `json:"details,omitempty"`
	Path      string    `json:"path"`
	Timestamp time.Time `json:"timestamp"`
}

type ErrorBody struct {
	Error ErrorDetail `json:"error"`
}

func WriteJSON(writer http.ResponseWriter, status int, data any) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(status)

	if err := json.NewEncoder(writer).Encode(data); err != nil {
		slog.Error("failed to write JSON response", "error", err)
	}
}

func WriteError(writer http.ResponseWriter, request *http.Request, err error) {
	appErr := apperror.From(err)

	if appErr.Status >= http.StatusInternalServerError {
		slog.ErrorContext(request.Context(), "internal server error",
			"method", request.Method, "path", request.URL.Path, "code", appErr.Code, "error", appErr,
		)
	}

	WriteJSON(writer, appErr.Status, ErrorBody{
		Error: ErrorDetail{
			Code:      appErr.Code,
			Message:   appErr.Message,
			Details:   appErr.Details,
			Path:      request.URL.Path,
			Timestamp: time.Now().UTC(),
		},
	})
}

func DecodeJSON(writer http.ResponseWriter, request *http.Request, destination any) error {
	decoder := json.NewDecoder(http.MaxBytesReader(writer, request.Body, maxBodyBytes))

	decoder.DisallowUnknownFields()

	if err := decoder.Decode(destination); err != nil {
		return apperror.BadRequest("INVALID_BODY", "Invalid JSON body").WithCause(err)
	}

	return nil
}
