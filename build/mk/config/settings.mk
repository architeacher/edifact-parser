# Build settings

REPO_ROOT        := $(shell git rev-parse --show-toplevel)
SERVICES_DIR     := ${REPO_ROOT}/services
SERVICE_NAME     := svc-ingestion
SERVICE_DIR      := ${SERVICES_DIR}/${SERVICE_NAME}
DEPLOY_DIR       := ${REPO_ROOT}/deployment/docker
COMPOSE_FILE     := ${DEPLOY_DIR}/compose.yaml

PROJECT_NAME     := ingestion
CERTS_DIR        := .certs
ENV_FILE         := .envrc

GO               := go
GOFLAGS          ?=
GOTEST_FLAGS     ?= -race -count=1

DOCKER           := docker
DOCKER_COMPOSE   := docker compose

API_CONTRACTS_DIR := ${REPO_ROOT}/docs/contracts/openapi
API_SPEC_DIR     := ${API_CONTRACTS_DIR}/ingestion/v1
API_SPEC_FILE    := ${API_SPEC_DIR}/specs.yaml
API_BUNDLED_JSON := ${API_SPEC_DIR}/public/swagger-pact.json

GOLANGCI_VERSION := v2.10.1
GOLANGCI_IMAGE   := golangci/golangci-lint:${GOLANGCI_VERSION}

# Colours for terminal output
RESET   := \033[0m
BOLD    := \033[1m
GREEN   := \033[0;32m
YELLOW  := \033[0;33m
RED     := \033[0;31m
CYAN    := \033[0;36m
