```bash
    ____                      __  _
   /  _/___  ____ ____  _____/ /_(_)___  ____
   / // __ \/ __ `/ _ \/ ___/ __/ / __ \/ __ \
 _/ // / / / /_/ /  __(__  ) /_/ / /_/ / / / /
/___/_/ /_/\__, /\___/____/\__/_/\____/_/ /_/
          /____/

```

EDIFACT ingestion system for the German electricity retail market with hexagonal architecture and CQRS.

## Table of Contents

- [Prerequisites](#prerequisites)
- [Quick Start](#quick-start)
- [Local URLs](#local-urls)
- [Project Structure](#project-structure)
- [Available Commands](#available-commands)
- [Testing](#testing)
  - [Unit Tests](#unit-tests)
  - [Integration Tests](#integration-tests)
- [API Examples](#api-examples)
  - [Ingest an EDIFACT File](#ingest-an-edifact-file)
  - [Get Interchange Details](#get-interchange-details)
  - [Download Raw EDIFACT File](#download-raw-edifact-file)
  - [List Messages](#list-messages)
  - [List Messages with Filtering](#list-messages-with-filtering)
  - [Health Check](#health-check)
- [Configuration](#configuration)
- [Database Schema](#database-schema)
- [Documentation](#documentation)

## Prerequisites

- Go 1.26+
- Docker & Docker Compose
- [mkcert](https://github.com/FiloSottile/mkcert#installation) (for local TLS certificates)

## Quick Start

```bash
# One-time setup (adds /etc/hosts entries, generates TLS certs via mkcert, generates API stubs)
make init

# Start development environment (PostgreSQL, Kafka, migrations, service, observability)
make start

# Ingest a sample EDIFACT file
curl -s -X POST https://api.ingestion.dev/v1/interchanges \
  -F "file=@services/svc-ingestion/testdata/edifact/valid/mscons_single.edi" | jq
```

## Local URLs

| Service | URL |
|---------|-----|
| API | https://api.ingestion.dev |
| API Documentation | https://docs.ingestion.dev |
| Traefik Dashboard | https://traefik.ingestion.dev |
| Jaeger UI | https://jaeger.ingestion.dev |
| Prometheus | https://prometheus.ingestion.dev |

## Project Structure

```
nomos-technical-challenge/
├── services/
│   └── svc-ingestion/             # EDIFACT ingestion service
│       ├── cmd/svc-ingestion/     # Entry point (runtime.NewFromEnv().Run())
│       ├── internal/
│       │   ├── adapters/
│       │   │   ├── inbound/
│       │   │   │   ├── edifact/   # Lexer, parser, tokens, delimiters
│       │   │   │   └── http/      # HTTP handlers, middleware
│       │   │   ├── outbound/      # Kafka event publisher
│       │   │   ├── repos/         # PostgreSQL repositories (pgx, squirrel, scany)
│       │   │   └── services/      # Ingestion service adapter
│       │   ├── config/            # Environment-based configuration
│       │   ├── domain/model/      # Interchange, Message, Subscription, ProcessingVersion
│       │   ├── infrastructure/    # Logger, metrics, tracing decorators
│       │   ├── ports/             # Port interfaces
│       │   ├── runtime/           # Service wiring and lifecycle
│       │   └── usecases/
│       │       ├── commands/      # IngestFile
│       │       └── queries/       # GetInterchange, GetRawFile, ListMessages, Health
│       ├── migrations/            # PostgreSQL migrations (001–006)
│       ├── itest/                 # Integration tests (testcontainers)
│       └── testdata/edifact/      # 10 valid + 5 invalid reference files
├── docs/
│   ├── contracts/openapi/         # OpenAPI 3.0.3 specifications
│   ├── architecture.md            # ADRs and C4 diagrams
│   └── CHANGELOG.md
├── build/
│   ├── mk/                        # Modular Makefiles
│   └── oapi/                      # oapi-codegen config
├── deployment/
│   └── docker/                    # Docker Compose, Traefik, Prometheus
└── specs/
    └── 001-edifact-ingestion/     # Feature spec, plan, tasks, checklists
```

## Available Commands

```bash
make help              # Show all available targets
make init              # One-time setup (hosts, TLS certs, API codegen)
make start             # Start Docker containers
make stop              # Stop containers (keep volumes)
make restart           # Stop and start containers
make destroy           # Remove containers, networks, volumes
make logs              # Tail logs from all services
make build             # Build svc-ingestion binary
make run               # Build and run locally
make dev               # Hot-reload with air
make lint              # Run golangci-lint (18 linters)
make lint-fix          # Run golangci-lint with --fix
make lint-api          # Lint OpenAPI spec (Redocly)
make bundle-api        # Bundle modular OpenAPI spec
make generate-api      # Regenerate HTTP server stubs
make migrate-up        # Apply pending database migrations
make migrate-down      # Rollback last migration
make stats             # Show code statistics
```

## Testing

### Unit Tests

```bash
# Run all unit tests with race detection
make test-unit

# Run specific package tests
cd services/svc-ingestion && go test -v -race ./internal/adapters/inbound/edifact/...
```

### Integration Tests

Integration tests use `testcontainers-go` to spin up real PostgreSQL and Kafka containers.

```bash
# Run integration tests (requires Docker)
make test-integration

# Run all tests (unit + integration)
make test

# View coverage report
make coverage
```

## API Examples

### Ingest an EDIFACT File

```bash
curl -s -X POST https://api.ingestion.dev/v1/interchanges \
  -F "file=@services/svc-ingestion/testdata/edifact/valid/mscons_single.edi" | jq
```

**Response (201 Created):**

```json
{
  "interchange_id": "019529a1-7c3f-7a6b-843b-9cca8343bf08",
  "message_summary": {
    "by_type": {},
    "total": 1
  },
  "status": "completed"
}
```

**Idempotent resubmission** of the same file returns `200 OK` with the original interchange ID.

**Note:** Maximum file size is 10 MB. Files exceeding this limit return `413 Payload Too Large`. The `by_type` breakdown is not yet populated (returns `{}`).

### Get Interchange Details

```bash
curl -s https://api.ingestion.dev/v1/interchanges/019529a1-7c3f-7a6b-843b-9cca8343bf08 | jq
```

**Response (200 OK):**

```json
{
  "active_version": {
    "created_at": "2026-02-19T12:00:00.123456Z",
    "format_version": "",
    "id": "019529a1-8a00-7d3e-b123-4567890abcde",
    "parser_version": "1.0.0",
    "version_number": 1
  },
  "created_at": "2026-02-19T12:00:00.123456Z",
  "id": "019529a1-7c3f-7a6b-843b-9cca8343bf08",
  "message_summary": {
    "by_type": {},
    "total": 1
  },
  "prepared_at": "2026-02-19T08:00:00Z",
  "receiver_id": "9876543210987",
  "reference": "00000001",
  "sender_id": "1234567890123",
  "status": "completed"
}
```

### Download Raw EDIFACT File

```bash
curl -s https://api.ingestion.dev/v1/interchanges/019529a1-7c3f-7a6b-843b-9cca8343bf08/raw \
  -o original.edi
```

**Response (200 OK):** Binary EDIFACT content (`application/octet-stream`).

### List Messages

```bash
curl -s "https://api.ingestion.dev/v1/messages?limit=20" | jq
```

**Response (200 OK):**

```json
{
  "items": [
    {
      "created_at": "2026-02-19T12:00:00.123456Z",
      "id": "019529a2-1e59-7c6a-aaf0-0c60fccccf89",
      "interchange_id": "019529a1-7c3f-7a6b-843b-9cca8343bf08",
      "message_reference": "MSG001",
      "message_type": "MSCONS",
      "processing_version_id": "019529a1-8a00-7d3e-b123-4567890abcde",
      "segments": {
        "segment_0": {"tag": "UNH", "de1": "MSG001", "de2": ["MSCONS", "D", "04B", "UN"]},
        "segment_1": {"tag": "BGM", "de1": "7", "de2": "DOCID001", "de3": "9"},
        "segment_2": {"tag": "DTM", "de1": ["137", "202602190800", "203"]},
        "segment_5": {"tag": "LOC", "de1": "172", "de2": "DE0001234567890000000000000123456"},
        "...": "remaining segments omitted for brevity"
      },
      "subscription_id": "019529a1-9b11-7e4f-c234-567890abcdef"
    }
  ],
  "total": 1
}
```

**Note:** The `segments` field is a JSONB object keyed by `segment_N`. Each segment contains a `tag` and data elements (`de1`, `de2`, ...). Single-component elements are strings; multi-component elements are arrays. The `next_cursor` field appears only when additional pages exist. The `subscription_id` field is omitted when no LOC+172 segment is present.

### List Messages with Filtering

Filter by subscription, message type, or interchange:

```bash
curl -s "https://api.ingestion.dev/v1/messages?message_type=MSCONS&limit=10" | jq
```

**Available Query Parameters:**

| Parameter | Description | Example |
|-----------|-------------|---------|
| `subscription_id` | Filter by metering point subscription | `?subscription_id=019529a1-9b11-...` |
| `message_type` | Filter by EDIFACT message type | `?message_type=MSCONS` |
| `interchange_id` | Filter by parent interchange | `?interchange_id=019529a1-7c3f-...` |
| `processing_version_id` | Filter by processing version | `?processing_version_id=019529a1-8a00-...` |
| `limit` | Items per page (1–100, default 20) | `?limit=50` |
| `cursor` | Cursor for next page (base64url-encoded) | `?cursor=eyJpZCI6...` |

### Health Check

```bash
curl -s https://api.ingestion.dev/v1/health | jq
```

**Response (200 OK):**

```json
{
  "checks": {
    "kafka": "ok",
    "postgres": "ok"
  },
  "status": "healthy"
}
```

**Additional probes:**

```bash
# Liveness (process alive) — returns 200 with empty body
curl -s -o /dev/null -w "%{http_code}" https://api.ingestion.dev/v1/liveness

# Readiness (dependencies connected) — returns 200 with empty body
curl -s -o /dev/null -w "%{http_code}" https://api.ingestion.dev/v1/readiness
```

## Configuration

Environment variables use the `EDIFACT_` prefix:

| Variable | Default | Description |
|----------|---------|-------------|
| `EDIFACT_SERVER_PORT` | `8080` | HTTP server port |
| `EDIFACT_DB_HOST` | — | PostgreSQL host |
| `EDIFACT_DB_PORT` | `5432` | PostgreSQL port |
| `EDIFACT_DB_USER` | — | PostgreSQL user |
| `EDIFACT_DB_PASSWORD` | — | PostgreSQL password |
| `EDIFACT_DB_NAME` | — | Database name |
| `EDIFACT_DB_SSLMODE` | `disable` | SSL mode |
| `EDIFACT_DB_POOL_SIZE` | `20` | Connection pool size |
| `EDIFACT_KAFKA_BROKERS` | — | Kafka broker addresses (comma-separated) |
| `EDIFACT_KAFKA_TOPIC` | `edifact.events` | Kafka topic name |
| `EDIFACT_LOG_LEVEL` | `info` | Log level (debug/info/warn/error) |
| `EDIFACT_OTEL_ENDPOINT` | `localhost:4317` | OpenTelemetry OTLP endpoint |
| `EDIFACT_OTEL_SERVICE_NAME` | `edifact-ingestion` | Service name in traces |
| `EDIFACT_OTEL_ENABLED` | `true` | Enable distributed tracing |

Copy `.envrc.dist` to `.envrc` and adjust for local development.

## Database Schema

Six migrations manage the schema (PostgreSQL 16, UUID v7 primary keys):

| Table | Purpose |
|-------|---------|
| `interchanges` | UNB envelope data, raw content (BYTEA), SHA-256 content hash (idempotency) |
| `processing_versions` | Version tracking per interchange for reprocessing support |
| `subscriptions` | Metering points extracted from LOC+172 segments |
| `messages` | Parsed EDIFACT messages with JSONB segments |
| `domain_events` | Immutable audit log (triggers prevent UPDATE/DELETE) |

## Documentation

- [Architecture](architecture.md) — ADRs and system diagrams
- [API Specification](contracts/openapi/ingestion/v1/public/swagger-pact.json) — OpenAPI 3.0.3 spec
- [Changelog](CHANGELOG.md) — Release history
- [Feature Spec](../specs/001-edifact-ingestion/spec.md) — Requirements and design
- [Quickstart](../specs/001-edifact-ingestion/quickstart.md) — Getting started guide
