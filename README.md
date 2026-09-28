# Bia Energy · Anomaly Center

Plataforma que detecta, explica y prioriza anomalías de consumo energético en medidores industriales.

**Datos → Análisis → Anomalía → Explicación → Priorización → Acción**

## Enlaces

| Servicio | URL |
|---|---|
| Frontend | https://biaenergy-test.vercel.app |
| Backend (API) | https://backend-production-1c6d0.up.railway.app/api/health |
| Backend · Swagger | https://backend-production-1c6d0.up.railway.app/api/docs |
| Servicio de IA · Swagger | https://ai-service-production-d553.up.railway.app/docs |

Las credenciales de acceso a la demo se envían por correo.

**Recorrido de la demo:** Login → Dashboard → **✨ Ejecutar análisis IA** → M-109 → Anomalía → Explicación → Acción recomendada.

## Arquitectura

```
Frontend (React · Vercel)
        │  HTTPS + JWT
        ▼
Backend (Go · Railway) ──── PostgreSQL (Railway)
        │  red privada + X-Service-Token
        ▼
Servicio de IA (Python · Railway) ──── Claude API (redacta la explicación)
```

| Carpeta | Stack | Qué hace |
|---|---|---|
| `backend/` | Go, chi, PostgreSQL, sqlc, goose | API REST, autenticación JWT, medidores, importación de CSV, análisis asíncrono, anomalías y dashboard |
| `ai-service/` | Python 3.12, FastAPI, uv | Detecta anomalías con estadística robusta, las correlaciona con eventos, las clasifica con reglas, las prioriza y genera la explicación |
| `frontend/` | React 19, TypeScript, Vite, CSS Modules | Dashboard, medidores, detalle con gráficas, anomalías, investigación e importación |
| `data/` | CSV | Dataset del reto (`readings.csv`, `events.csv`) |

La metodología del motor de anomalías, con los cálculos detrás de cada decisión, está en [`ai-service/docs/METHODOLOGY.md`](ai-service/docs/METHODOLOGY.md).

## Ejecutar en local

### Requisitos

- Docker
- Go 1.27 y [goose](https://github.com/pressly/goose) (`go install github.com/pressly/goose/v3/cmd/goose@latest`)
- Python 3.12 y [uv](https://docs.astral.sh/uv/)
- Node 24 y pnpm
- Opcional: [air](https://github.com/air-verse/air) para recarga en caliente del backend

### 1. Variables de entorno

```bash
cp .env.example .env
```

Completa en `.env`:

- `ADMIN_PASSWORD`: contraseña del usuario `admin@bia.app` (mínimo 12 caracteres; evita `$`, `#` y comillas).
- `LLM_API_KEY`: opcional. Sin ella, las explicaciones se generan con plantillas en lugar de Claude.

### 2. Base de datos, migraciones y datos

```bash
make db-up        # PostgreSQL en Docker (puerto 5433)
make migrate-up   # crea el esquema
make seed         # usuario admin + 4.032 lecturas + 4 eventos
```

### 3. Levantar los servicios (una terminal para cada uno)

```bash
# Servicio de IA → http://localhost:8000/docs
cd ai-service && uv sync && uv run python -m app

# Backend → http://localhost:8080/api/docs
make dev                          # o: cd backend && go run ./cmd/api

# Frontend → http://localhost:5173
cd frontend && pnpm install && pnpm dev
```

Entra a http://localhost:5173 con `admin@bia.app` y tu `ADMIN_PASSWORD`.

### Alternativa: todo en Docker

```bash
make up            # postgres + migraciones + servicio de IA + backend (http://localhost:8080)
make seed-docker   # carga el admin y los datos
cd frontend && pnpm install && pnpm dev
```

## Variables de entorno

| Variable | Servicio | Descripción |
|---|---|---|
| `DATABASE_URL` | backend | Conexión a PostgreSQL |
| `APP_ENV` | backend, IA | `development` o `production` |
| `APP_TIMEZONE` | backend, IA | Zona horaria de los datos (`America/Bogota`) |
| `API_PORT` / `PORT` | backend | Puerto HTTP (por defecto 8080) |
| `ALLOWED_ORIGINS` | backend | Orígenes permitidos por CORS (URL del frontend) |
| `JWT_SECRET` | backend | Secreto del JWT (obligatorio en producción, mínimo 32 caracteres) |
| `JWT_TTL` | backend | Duración de la sesión (por defecto `12h`) |
| `AI_SERVICE_URL` | backend | URL del servicio de IA |
| `AI_TIMEOUT` | backend | Tiempo máximo del análisis (por defecto `2m`) |
| `AI_SERVICE_TOKEN` | backend, IA | Token compartido que protege `POST /analyze` |
| `ADMIN_EMAIL` / `ADMIN_PASSWORD` / `ADMIN_NAME` | seed | Usuario administrador que crea la seed |
| `HOST` / `PORT` | IA | Interfaz y puerto (por defecto `0.0.0.0:8000`) |
| `LLM_API_KEY` / `LLM_MODEL` | IA | Clave y modelo de Claude (por defecto `claude-sonnet-5`) |
| `VITE_API_BASE_URL` | frontend | URL del backend (por defecto `http://localhost:8080`) |

## Tests y calidad

```bash
cd backend    && go test ./... && go vet ./...          # 293 casos
cd ai-service && uv run pytest && uv run ruff check .   # 181 tests
cd frontend   && pnpm test --run && pnpm lint           # 90 tests
```

## Despliegue

| Servicio | Plataforma | Configuración |
|---|---|---|
| PostgreSQL | Railway | Plugin de base de datos |
| Backend | Railway | Root `/`, Dockerfile `backend/Dockerfile`, pre-deploy `/app/migrate`, healthcheck `/api/health` |
| Servicio de IA | Railway | Root `/ai-service`, healthcheck `/health`, accesible por la red privada |
| Frontend | Vercel | Root `frontend`, variable `VITE_API_BASE_URL` |

Cada `git push` a `main` redespliega los servicios afectados. Las migraciones se aplican automáticamente antes de cada despliegue del backend.

La carga inicial de datos en producción se ejecuta una sola vez desde local:

```bash
cd backend && DATABASE_URL='<DATABASE_PUBLIC_URL de Railway>' ADMIN_PASSWORD='<contraseña>' go run ./cmd/seed
```
