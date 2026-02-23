# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [Unreleased]

### Added

- README API examples: all JSON response fields now use snake\_case matching the OpenAPI spec and oapi-codegen output
- README API examples: health check values corrected from `"up"` to `"ok"` per `HealthChecksKafka`/`HealthChecksPostgres` enums
- README API examples: liveness/readiness curl examples updated (empty body responses, no `jq` piping)
- README API examples: message `segments` field documented as JSONB object (`segment_N` keys) instead of array
- README API examples: all curl URLs updated from `http://` to `https://` (Traefik redirects HTTP→HTTPS)
- README Quick Start: added `make init` prerequisite for hosts/TLS/codegen setup
- README Prerequisites: added `mkcert` dependency
- Dockerfile: added missing `pkg/decorator`, `pkg/logger`, `pkg/metrics` module COPY for Go workspace build
- oapi-codegen config: fixed `include-tags` from non-existent `[Public, System]` to actual `[Interchanges, Messages, System]`
- Router handler: updated type references to match regenerated oapi-codegen output (shorter type names)
- README API examples: `by_type` field corrected from `{"MSCONS": 1}` to `{}` (not yet populated by service layer)
- Docker Compose: Kafka healthcheck updated from `kafka-topics.sh` to `kafka-topics` (Confluent Platform 8.x dropped `.sh` suffix)
- Docker Compose: svc-ingestion healthcheck disabled (distroless image has no `wget` binary)
- `.envrc.dist`: fixed env var names to match `envconfig` struct (`EDIFACT_DB_HOST` etc. instead of `DATABASE_URL`)
- Dedicated `migrate/migrate` Docker service for schema migrations
- `Dockerfile.postgres`: custom PostgreSQL 16 image with `pg_uuidv7` extension compiled from source
