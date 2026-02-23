## ──────────────────────────────────────────────────────────────
## Testing
## ──────────────────────────────────────────────────────────────

.PHONY: test-unit
test-unit: ## Run unit tests (no external dependencies)
	$(call printMessage,Running unit tests)
	cd ${SERVICE_DIR} && ${GO} test ${GOTEST_FLAGS} \
		$(shell cd ${SERVICE_DIR} && ${GO} list ./... | grep -v '/itest') \
		-coverprofile=coverage/unit.cover

.PHONY: test-integration
test-integration: ## Run integration tests (requires Docker — starts testcontainers)
	$(call printMessage,Running integration tests with testcontainers)
	cd ${SERVICE_DIR} && ${GO} test ${GOTEST_FLAGS} \
		-timeout 300s \
		./itest/... \
		-coverprofile=coverage/integration.cover

.PHONY: test
test: test-unit test-integration ## Run all tests

.PHONY: coverage
coverage: ## Show coverage report (run test-unit first)
	${GO} tool cover -html=${SERVICE_DIR}/coverage/unit.cover
