package auth

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/santigonzalezla/biaenergy-test/backend/internal/apperror"
	"github.com/santigonzalezla/biaenergy-test/backend/internal/httpserver"
)

type Middleware func(http.Handler) http.Handler

type Handler struct {
	service     *Service
	requireAuth Middleware
}

func NewHandler(service *Service, requireAuth Middleware) *Handler {
	return &Handler{service: service, requireAuth: requireAuth}
}

func (handler *Handler) RegisterRoutes(route chi.Router) {
	route.Route("/auth", func(auth chi.Router) {
		auth.Post("/login", handler.login)
		auth.With(handler.requireAuth).Get("/me", handler.me)
	})
}

func (handler *Handler) login(writer http.ResponseWriter, request *http.Request) {
	var body LoginRequest

	if err := httpserver.DecodeJSON(writer, request, &body); err != nil {
		httpserver.WriteError(writer, request, err)
		return
	}

	response, err := handler.service.Login(request.Context(), body)

	if err != nil {
		httpserver.WriteError(writer, request, err)
		return
	}

	writer.Header().Set("Cache-Control", "no-store")
	httpserver.WriteJSON(writer, http.StatusOK, response)
}

func (handler *Handler) me(writer http.ResponseWriter, request *http.Request) {
	claims, ok := ClaimsFrom(request.Context())

	if !ok {
		httpserver.WriteError(writer, request, apperror.Unauthorized("MISSING_TOKEN", "Authentication is required"))
		return
	}

	user, err := handler.service.Me(request.Context(), claims.UserId)

	if err != nil {
		httpserver.WriteError(writer, request, err)
		return
	}

	httpserver.WriteJSON(writer, http.StatusOK, user)
}
