FROM golang:1.27-alpine AS builder

WORKDIR /src

COPY backend/go.mod backend/go.sum ./
RUN go mod download

COPY backend/ .

ENV CGO_ENABLED=0 GOOS=linux

RUN go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api \
    && go build -trimpath -ldflags="-s -w" -o /out/migrate ./cmd/migrate \
    && go build -trimpath -ldflags="-s -w" -o /out/seed ./cmd/seed


FROM gcr.io/distroless/static-debian12:nonroot AS runtime

WORKDIR /app

COPY --from=builder /out/api /app/api
COPY --from=builder /out/migrate /app/migrate
COPY --from=builder /out/seed /app/seed
COPY data/readings.csv data/events.csv /data/

EXPOSE 8080

USER nonroot:nonroot

ENTRYPOINT ["/app/api"]
