package docs

import (
	_ "embed"
	"net/http"

	"github.com/go-chi/chi/v5"
)

//go:embed openapi.yaml
var specification []byte

const swaggerPage = `<!doctype html>
<html lang="es">
<head>
    <meta charset="utf-8"/>
    <meta name="viewport" content="width=device-width, initial-scale=1"/>
    <title>BiaEnergy API · Swagger</title>
    <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/swagger-ui-dist@5/swagger-ui.css"/>
</head>
<body>
    <div id="swagger-ui"></div>
    <script src="https://cdn.jsdelivr.net/npm/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
    <script>
        window.ui = SwaggerUIBundle({url: 'openapi.yaml', dom_id: '#swagger-ui', persistAuthorization: true});
    </script>
</body>
</html>`

type Handler struct{}

func NewHandler() *Handler {
	return &Handler{}
}

func (handler *Handler) RegisterRoutes(route chi.Router) {
	route.Get("/docs", handler.page)
	route.Get("/openapi.yaml", handler.specification)
}

func (handler *Handler) page(writer http.ResponseWriter, request *http.Request) {
	writer.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = writer.Write([]byte(swaggerPage))
}

func (handler *Handler) specification(writer http.ResponseWriter, request *http.Request) {
	writer.Header().Set("Content-Type", "application/yaml")
	_, _ = writer.Write(specification)
}
