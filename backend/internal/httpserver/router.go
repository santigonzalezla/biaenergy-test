package httpserver

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/santigonzalezla/biaenergy-test/backend/internal/apperror"
)

type RouteRegister interface {
	RegisterRoutes(route chi.Router)
}

type Routes struct {
	Public      []RouteRegister
	Protected   []RouteRegister
	RequireAuth func(http.Handler) http.Handler
}

func NewRouter(allowedOrigins []string, routes Routes) http.Handler {
	if len(routes.Protected) > 0 && routes.RequireAuth == nil {
		panic("httpserver: protected routes require an auth middleware")
	}

	route := chi.NewRouter()

	route.Use(middleware.RequestID)
	route.Use(middleware.ClientIPFromHeader("X-Real-IP"))
	route.Use(RequestLogger)
	route.Use(Recover)
	route.Use(cors.Handler(cors.Options{
		AllowedOrigins: allowedOrigins,
		AllowedMethods: []string{http.MethodGet, http.MethodPost, http.MethodPatch, http.MethodDelete, http.MethodOptions},
		AllowedHeaders: []string{"Accept", "Authorization", "Content-Type", "X-Request-Id"},
		ExposedHeaders: []string{middleware.RequestIDHeader},
		MaxAge:         300,
	}))

	route.NotFound(func(writer http.ResponseWriter, request *http.Request) {
		WriteError(writer, request, apperror.NotFound("ROUTE_NOT_FOUND", "The requested resource was not found"))
	})

	route.MethodNotAllowed(func(writer http.ResponseWriter, request *http.Request) {
		WriteError(writer, request, apperror.MethodNotAllowed())
	})

	route.Route("/api", func(api chi.Router) {
		for _, module := range routes.Public {
			module.RegisterRoutes(api)
		}

		if len(routes.Protected) == 0 {
			return
		}

		api.Group(func(protected chi.Router) {
			protected.Use(routes.RequireAuth)

			for _, module := range routes.Protected {
				module.RegisterRoutes(protected)
			}
		})
	})

	return route
}
