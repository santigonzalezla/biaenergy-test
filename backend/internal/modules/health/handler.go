package health

import (
	"context"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/santigonzalezla/biaenergy-test/backend/internal/httpserver"
)

type Pinger interface {
	Ping(ctx context.Context) error
}

type Handler struct {
	db Pinger
}

type response struct {
	Status   string    `json:"status"`
	Database string    `json:"database"`
	Time     time.Time `json:"time"`
}

func NewHandler(db Pinger) *Handler {
	return &Handler{db: db}
}

func (handler *Handler) RegisterRoutes(route chi.Router) {
	route.Get("/health", handler.check)
}

func (handler *Handler) check(writer http.ResponseWriter, request *http.Request) {
	ctx, cancel := context.WithTimeout(request.Context(), 2*time.Second)
	defer cancel()

	if err := handler.db.Ping(ctx); err != nil {
		httpserver.WriteJSON(writer, http.StatusServiceUnavailable, response{
			Status:   "degraded",
			Database: "down",
			Time:     time.Now().UTC(),
		})
		return
	}

	httpserver.WriteJSON(writer, http.StatusOK, response{
		Status:   "ok",
		Database: "up",
		Time:     time.Now().UTC(),
	})
}
