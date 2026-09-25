package ingestion

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/santigonzalezla/biaenergy-test/backend/internal/apperror"
	"github.com/santigonzalezla/biaenergy-test/backend/internal/httpserver"
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
}

func (handler *Handler) importReadings(writer http.ResponseWriter, request *http.Request) {
	handleImport(writer, request, handler.service.ImportReadings)
}

func (handler *Handler) importEvents(writer http.ResponseWriter, request *http.Request) {
	handleImport(writer, request, handler.service.ImportEvents)
}

func handleImport(writer http.ResponseWriter, request *http.Request, importFunc ImportFunc) {
	request.Body = http.MaxBytesReader(writer, request.Body, maxUploadBytes)

	file, _, err := request.FormFile("file")

	if err != nil {
		httpserver.WriteError(writer, request, apperror.BadRequest("INVALID_FILE", "Send the csv in a multipart/form-data field name 'file'").WithCause(err))
		return
	}

	defer file.Close()

	result, err := importFunc(request.Context(), file)

	if err != nil {
		httpserver.WriteError(writer, request, err)
		return
	}

	httpserver.WriteJSON(writer, http.StatusOK, result)
}
