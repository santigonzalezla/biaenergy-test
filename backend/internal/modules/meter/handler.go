package meter

import (
	"net/http"
	"strconv"
	"time"

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
	route.Route("/meters", func(meters chi.Router) {
		meters.Get("/", handler.list)
		meters.Post("/", handler.create)

		meters.Route("/{meterId}", func(meter chi.Router) {
			meter.Get("/", handler.get)
			meter.Patch("/", handler.update)
			meter.Delete("/", handler.delete)
			meter.Get("/readings", handler.listReadings)
			meter.Get("/events", handler.listEvents)
		})
	})
}

func (handler *Handler) list(writer http.ResponseWriter, request *http.Request) {
	query, err := parseListQuery(request)

	if err != nil {
		httpserver.WriteError(writer, request, err)
		return
	}

	page, err := handler.service.List(request.Context(), query)

	if err != nil {
		httpserver.WriteError(writer, request, err)
		return
	}

	httpserver.WriteJSON(writer, http.StatusOK, page)
}

func (handler *Handler) get(writer http.ResponseWriter, request *http.Request) {
	id, err := parseMeterID(request)

	if err != nil {
		httpserver.WriteError(writer, request, err)
		return
	}

	meter, err := handler.service.Get(request.Context(), id)

	if err != nil {
		httpserver.WriteError(writer, request, err)
		return
	}

	httpserver.WriteJSON(writer, http.StatusOK, meter)
}

func (handler *Handler) create(writer http.ResponseWriter, request *http.Request) {
	var body CreateMeterRequest

	if err := httpserver.DecodeJSON(writer, request, &body); err != nil {
		httpserver.WriteError(writer, request, err)
		return
	}

	meter, err := handler.service.Create(request.Context(), body)
	if err != nil {
		httpserver.WriteError(writer, request, err)
		return
	}

	writer.Header().Set("Location", "/api/meters/"+meter.ID.String())
	httpserver.WriteJSON(writer, http.StatusCreated, meter)
}

func (handler *Handler) update(writer http.ResponseWriter, request *http.Request) {
	id, err := parseMeterID(request)
	if err != nil {
		httpserver.WriteError(writer, request, err)
		return
	}

	var body UpdateMeterRequest

	if err := httpserver.DecodeJSON(writer, request, &body); err != nil {
		httpserver.WriteError(writer, request, err)
		return
	}

	meter, err := handler.service.Update(request.Context(), id, body)
	if err != nil {
		httpserver.WriteError(writer, request, err)
		return
	}

	httpserver.WriteJSON(writer, http.StatusOK, meter)
}

func (handler *Handler) delete(writer http.ResponseWriter, request *http.Request) {
	id, err := parseMeterID(request)
	if err != nil {
		httpserver.WriteError(writer, request, err)
		return
	}

	if err := handler.service.Delete(request.Context(), id); err != nil {
		httpserver.WriteError(writer, request, err)
		return
	}

	writer.WriteHeader(http.StatusNoContent)
}

func (handler *Handler) listReadings(writer http.ResponseWriter, request *http.Request) {
	id, err := parseMeterID(request)

	if err != nil {
		httpserver.WriteError(writer, request, err)
		return
	}

	query, err := parseSeriesQuery(request)

	if err != nil {
		httpserver.WriteError(writer, request, err)
		return
	}

	series, err := handler.service.ListReadings(request.Context(), id, query)

	if err != nil {
		httpserver.WriteError(writer, request, err)
		return
	}

	httpserver.WriteJSON(writer, http.StatusOK, series)
}

func (handler *Handler) listEvents(writer http.ResponseWriter, request *http.Request) {
	id, err := parseMeterID(request)

	if err != nil {
		httpserver.WriteError(writer, request, err)
		return
	}

	query, err := parseSeriesQuery(request)

	if err != nil {
		httpserver.WriteError(writer, request, err)
		return
	}

	series, err := handler.service.ListEvents(request.Context(), id, query)

	if err != nil {
		httpserver.WriteError(writer, request, err)
		return
	}

	httpserver.WriteJSON(writer, http.StatusOK, series)
}

func parseMeterID(request *http.Request) (uuid.UUID, error) {
	id, err := uuid.Parse(chi.URLParam(request, "meterId"))

	if err != nil {
		return uuid.Nil, apperror.BadRequest("INVALID_METER_ID", "Must be a valid UUID")
	}

	return id, nil
}

func parseListQuery(request *http.Request) (ListQuery, error) {
	values := request.URL.Query()
	errs := map[string]string{}

	query := ListQuery{
		Status: values.Get("status"),
		Search: values.Get("search"),
		SortBy: values.Get("sortBy"),
	}

	if raw := values.Get("page"); raw != "" {
		page, err := strconv.Atoi(raw)

		if err != nil {
			errs["page"] = "Must be an Integer"
		}

		query.Page = page
	}

	if raw := values.Get("limit"); raw != "" {
		limit, err := strconv.Atoi(raw)
		if err != nil {
			errs["limit"] = "must be an integer"
		}
		query.Limit = limit
	}

	switch values.Get("sortDir") {
	case "", "asc":
		query.SortDesc = false
	case "desc":
		query.SortDesc = true
	default:
		errs["sortDir"] = "must be asc or desc"
	}

	if len(errs) > 0 {
		return ListQuery{}, apperror.Validation(errs)
	}

	return query, nil
}

func parseSeriesQuery(request *http.Request) (SeriesQuery, error) {
	values := request.URL.Query()
	errs := map[string]string{}

	query := SeriesQuery{
		From: parseOptionalTime(values.Get("from"), "from", errs),
		To:   parseOptionalTime(values.Get("to"), "to", errs),
	}

	if len(errs) > 0 {
		return SeriesQuery{}, apperror.Validation(errs)
	}

	return query, nil
}

func parseOptionalTime(raw, field string, errs map[string]string) *time.Time {
	if raw == "" {
		return nil
	}

	value, err := time.Parse(time.RFC3339, raw)

	if err != nil {
		errs[field] = "must be an RFC339 date as YYYY-MM-DDTHH:MM:SSZ"
		return nil
	}

	return &value
}
