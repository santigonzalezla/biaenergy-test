package analysis

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/santigonzalezla/biaenergy-test/backend/internal/apperror"
	"github.com/santigonzalezla/biaenergy-test/backend/internal/httpserver"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (handler *Handler) RegisterRoutes(route chi.Router) {
	route.Route("/ai", func(ai chi.Router) {
		ai.Post("/analyze", handler.start)
		ai.Get("/analysis/latest", handler.latest)
		ai.Get("/analysis/{analysisId}", handler.get)
	})
}

func (handler *Handler) start(writer http.ResponseWriter, request *http.Request) {
	var body StartRequest

	if request.ContentLength > 0 {
		if err := httpserver.DecodeJSON(writer, request, &body); err != nil {
			httpserver.WriteError(writer, request, err)
			return
		}
	}

	analysis, created, err := handler.service.Start(request.Context(), body)

	if err != nil {
		httpserver.WriteError(writer, request, err)
		return
	}

	writer.Header().Set("Location", "/api/ai/analysis/"+analysis.Id.String())
	if !created {
		writer.Header().Set("X-Analysis-Reused", "true")
	}

	httpserver.WriteJSON(writer, http.StatusAccepted, analysis)
}

func (handler *Handler) get(writer http.ResponseWriter, request *http.Request) {
	id, err := uuid.Parse(chi.URLParam(request, "analysisId"))

	if err != nil {
		httpserver.WriteError(writer, request, apperror.BadRequest("INVALID_ANALYSIS_ID", "Analysis ID must be a valid UUID"))
		return
	}

	analysis, err := handler.service.Get(request.Context(), id)

	if err != nil {
		httpserver.WriteError(writer, request, err)
		return
	}

	httpserver.WriteJSON(writer, http.StatusOK, analysis)
}

func (handler *Handler) latest(writer http.ResponseWriter, request *http.Request) {
	analysis, err := handler.service.Latest(request.Context())

	if err != nil {
		httpserver.WriteError(writer, request, err)
		return
	}

	httpserver.WriteJSON(writer, http.StatusOK, analysis)
}
