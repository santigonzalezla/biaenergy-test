package ingestion

import (
	"net/http"
	"path/filepath"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/santigonzalezla/biaenergy-test/backend/internal/apperror"
	"github.com/santigonzalezla/biaenergy-test/backend/internal/httpserver"
	"github.com/santigonzalezla/biaenergy-test/backend/internal/modules/auth"
)

const maxUploadBytes = 10 << 20

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (handler *Handler) RegisterRoutes(router chi.Router) {
	router.Post("/readings/import", handler.importReadings)
	router.Post("/events/import", handler.importEvents)
	router.Get("/imports", handler.list)
}

func (handler *Handler) importReadings(writer http.ResponseWriter, request *http.Request) {
	handleImport(writer, request, handler.service.ImportReadings)
}

func (handler *Handler) importEvents(writer http.ResponseWriter, request *http.Request) {
	handleImport(writer, request, handler.service.ImportEvents)
}

func (handler *Handler) list(writer http.ResponseWriter, request *http.Request) {
	limit := 0

	if raw := request.URL.Query().Get("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)

		if err != nil || parsed < 1 {
			httpserver.WriteError(writer, request, apperror.Validation(map[string]string{"limit": "Must be a positive integer"}))
			return
		}

		limit = parsed
	}

	response, err := handler.service.List(request.Context(), limit)

	if err != nil {
		httpserver.WriteError(writer, request, err)
		return
	}

	httpserver.WriteJSON(writer, http.StatusOK, response)
}

func handleImport(writer http.ResponseWriter, request *http.Request, importFunc ImportFunc) {
	request.Body = http.MaxBytesReader(writer, request.Body, maxUploadBytes)

	file, header, err := request.FormFile("file")

	if err != nil {
		httpserver.WriteError(writer, request, apperror.BadRequest("INVALID_FILE", "Send the csv in a multipart/form-data field name 'file'").WithCause(err))
		return
	}
	defer file.Close()

	result, err := importFunc(request.Context(), Upload{
		FileName: filepath.Base(header.Filename),
		Size:     header.Size,
		Content:  file,
		UserId:   currentUserId(request),
	})

	if err != nil {
		httpserver.WriteError(writer, request, err)
		return
	}

	httpserver.WriteJSON(writer, http.StatusOK, result)
}

func currentUserId(request *http.Request) *uuid.UUID {
	claims, ok := auth.ClaimsFrom(request.Context())

	if !ok {
		return nil
	}

	return &claims.UserId
}
