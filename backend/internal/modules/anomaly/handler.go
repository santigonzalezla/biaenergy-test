package anomaly

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
	route.Route("/anomalies", func(anomalies chi.Router) {
		anomalies.Get("/", handler.list)

		anomalies.Route("/{anomalyId}", func(anomaly chi.Router) {
			anomaly.Get("/", handler.get)
			anomaly.Patch("/status", handler.updateStatus)
		})
	})
}

func (handler *Handler) list(writer http.ResponseWriter, request *http.Request) {
	values := request.URL.Query()

	response, err := handler.service.List(request.Context(), ListQuery{
		AnalysisId: values.Get("analysisId"),
		Type:       values.Get("type"),
		Severity:   values.Get("severity"),
		Status:     values.Get("status"),
	})

	if err != nil {
		httpserver.WriteError(writer, request, err)
		return
	}

	httpserver.WriteJSON(writer, http.StatusOK, response)
}

func (handler *Handler) get(writer http.ResponseWriter, request *http.Request) {
	id, err := parseAnomalyId(request)

	if err != nil {
		httpserver.WriteError(writer, request, err)
		return
	}

	anomaly, err := handler.service.Get(request.Context(), id)

	if err != nil {
		httpserver.WriteError(writer, request, err)
		return
	}

	httpserver.WriteJSON(writer, http.StatusOK, anomaly)
}

func (handler *Handler) updateStatus(writer http.ResponseWriter, request *http.Request) {
	id, err := parseAnomalyId(request)

	if err != nil {
		httpserver.WriteError(writer, request, err)
		return
	}

	var body UpdateStatusRequest

	if err := httpserver.DecodeJSON(writer, request, &body); err != nil {
		httpserver.WriteError(writer, request, err)
		return
	}

	anomaly, err := handler.service.UpdateStatus(request.Context(), id, body)

	if err != nil {
		httpserver.WriteError(writer, request, err)
		return
	}

	httpserver.WriteJSON(writer, http.StatusOK, anomaly)
}

func parseAnomalyId(request *http.Request) (uuid.UUID, error) {
	id, err := uuid.Parse(chi.URLParam(request, "anomalyId"))

	if err != nil {
		return uuid.Nil, apperror.BadRequest("INVALID_ANOMALY_ID", "Anomaly ID must be a valid UUID")
	}

	return id, nil
}
