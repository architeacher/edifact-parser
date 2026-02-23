# syntax=docker/dockerfile:1

# ── Stage: dev-tools ──────────────────────────────────────────
FROM golang:1.26-alpine AS dev-tools

RUN apk add --no-cache git curl

# Install air for hot-reload and migrate for schema migrations
RUN go install github.com/air-verse/air@latest && \
    go install -tags "postgres" github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# ── Stage: builder ────────────────────────────────────────────
FROM golang:1.26-alpine AS builder

WORKDIR /workspace

# Cache module downloads separately from source
# Copy workspace and all module files referenced in go.work
COPY go.mod go.work ./
COPY services/svc-ingestion/go.mod services/svc-ingestion/go.sum ./services/svc-ingestion/
COPY pkg/decorator/go.mod pkg/decorator/go.sum ./pkg/decorator/
COPY pkg/logger/go.mod pkg/logger/go.sum ./pkg/logger/
COPY pkg/metrics/go.mod pkg/metrics/go.sum ./pkg/metrics/

RUN --mount=type=cache,target=/root/.cache/go \
    go mod download && \
    cd services/svc-ingestion && go mod download

# Copy source
COPY . .

# Build the service binary
RUN --mount=type=cache,target=/root/.cache/go \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w" \
    -o /out/svc-ingestion \
    ./services/svc-ingestion/cmd/svc-ingestion/

# ── Stage: prod ───────────────────────────────────────────────
FROM gcr.io/distroless/static-debian12 AS prod

COPY --from=builder /out/svc-ingestion /usr/local/bin/svc-ingestion

EXPOSE 8080

USER nonroot:nonroot

ENTRYPOINT ["/usr/local/bin/svc-ingestion"]

# ── Stage: dev ────────────────────────────────────────────────
FROM golang:1.26-alpine AS dev

RUN apk add --no-cache git

WORKDIR /app

COPY --from=dev-tools /go/bin/air /usr/local/bin/air
COPY --from=dev-tools /go/bin/migrate /usr/local/bin/migrate

# Mount source at runtime via volume; air config expects root = "/app"
EXPOSE 8080

CMD ["air", "-c", "./services/svc-ingestion/deployment/docker/config/air/.air.toml"]
