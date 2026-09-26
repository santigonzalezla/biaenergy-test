package dashboard

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/santigonzalezla/biaenergy-test/backend/internal/httpserver"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (handler *Handler) RegisterRoutes(router chi.Router) {
	router.Route("/dashboard", func(dashboard chi.Router) {
		dashboard.Get("/summary", handler.summary)
	})
}

func (handler *Handler) summary(writer http.ResponseWriter, request *http.Request) {
	summary, err := handler.service.Summary(request.Context())

	if err != nil {
		httpserver.WriteError(writer, request, err)
		return
	}

	httpserver.WriteJSON(writer, http.StatusOK, summary)
}
