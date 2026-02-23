## ──────────────────────────────────────────────────────────────
## Linting (Docker-based golangci-lint)
## ──────────────────────────────────────────────────────────────

.PHONY: lint
lint: ## Run golangci-lint in Docker on the service module
	$(call printMessage,Running golangci-lint ${GOLANGCI_VERSION})
	${DOCKER} run --rm \
		-v "${REPO_ROOT}:/workspace:ro" \
		-w /workspace \
		${GOLANGCI_IMAGE} \
		golangci-lint run ./services/svc-ingestion/...

.PHONY: lint-fix
lint-fix: ## Run golangci-lint with auto-fix (modifies source files)
	$(call printMessage,Running golangci-lint --fix)
	${DOCKER} run --rm \
		-v "${REPO_ROOT}:/workspace" \
		-w /workspace \
		${GOLANGCI_IMAGE} \
		golangci-lint run --fix ./services/svc-ingestion/...
