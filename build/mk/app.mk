## ──────────────────────────────────────────────────────────────
## Project initialisation & TLS
## ──────────────────────────────────────────────────────────────

.PHONY: ${ENV_FILE}
${ENV_FILE}:
	cat .envrc.dist | tee "${ENV_FILE}" > /dev/null

.PHONY: init-services
init-services: ## Generate per-service .envrc files from templates
	$(call printMessage,Generating per-service .envrc files)
	@for dir in ${SERVICES_DIR}/*/; do \
		if [ -f "$${dir}.envrc.dist" ]; then \
			cp "$${dir}.envrc.dist" "$${dir}.envrc"; \
			echo "Created $${dir}.envrc"; \
		fi \
	done

${CERTS_DIR}:
	mkdir -p "${CERTS_DIR}"

.PHONY: set-hosts
set-hosts: ## Update /etc/hosts with *.ingestion.dev entries
	$(call printMessage,Updating local hosts)
	echo "\n# Ingestion Hosts\n\
====================\n\
127.0.0.1 api.${PROJECT_NAME}.dev docs.${PROJECT_NAME}.dev traefik.${PROJECT_NAME}.dev\n\
127.0.0.1 jaeger.${PROJECT_NAME}.dev prometheus.${PROJECT_NAME}.dev" | sudo tee -a /etc/hosts

.PHONY: study
study: ${CERTS_DIR} ## Check mkcert is installed and initialise local CA
	$(call printMessage,Checking mkcert installation)
ifeq (, $(shell which "mkcert"))
 $(error "Command mkcert not found in $$PATH, please install https://github.com/FiloSottile/mkcert#installation")
endif
	mkcert -install

.PHONY: certify
certify: study ## Generate TLS certs for *.ingestion.dev
	$(call printMessage,Generating TLS certificates)
	mkcert -cert-file "${CERTS_DIR}/star-${PROJECT_NAME}-dev.crt" \
		-key-file "${CERTS_DIR}/star-${PROJECT_NAME}-dev.key" \
		"${PROJECT_NAME}.dev" "*.${PROJECT_NAME}.dev"
	cp "$$(mkcert -CAROOT)/rootCA.pem" "${CERTS_DIR}/"

.PHONY: init
init: ${ENV_FILE} init-services set-hosts certify ## One-time project setup (hosts, certs, API codegen)
	${MAKE} generate-api

## ──────────────────────────────────────────────────────────────
## Application lifecycle
## ──────────────────────────────────────────────────────────────

.PHONY: start
start: ## Start all services via Docker Compose
	$(call printMessage,Starting all services)
	${DOCKER_COMPOSE} -f ${COMPOSE_FILE} up -d --build --wait

.PHONY: stop
stop: ## Stop all services (keep volumes)
	$(call printMessage,Stopping all services)
	${DOCKER_COMPOSE} -f ${COMPOSE_FILE} stop

.PHONY: destroy
destroy: ## Stop and remove all containers, networks, and volumes
	$(call printMessage,Destroying all services and volumes)
	${DOCKER_COMPOSE} -f ${COMPOSE_FILE} down --volumes --remove-orphans

.PHONY: restart
restart: stop start ## Restart all services

.PHONY: logs
logs: ## Tail logs from all services
	${DOCKER_COMPOSE} -f ${COMPOSE_FILE} logs -f

.PHONY: dev
dev: ## Start hot-reload with air (requires Docker Compose running)
	$(call printMessage,Starting hot-reload for ${SERVICE_NAME})
	${DOCKER_COMPOSE} -f ${COMPOSE_FILE} exec ${SERVICE_NAME} air \
		-c /app/deployment/docker/config/air/.air.toml

.PHONY: lint-api
lint-api: ## Lint OpenAPI spec with Redocly
	$(call printMessage,Linting OpenAPI spec)
	${DOCKER} run --rm \
		-v "${API_CONTRACTS_DIR}":/spec \
		-w /spec \
		redocly/cli:latest lint \
		ingestion/v1/specs.yaml \
		--config .redocly.yaml

.PHONY: bundle-api
bundle-api: lint-api ## Bundle modular OpenAPI spec into single file
	$(call printMessage,Bundling OpenAPI spec)
	${DOCKER} run --rm \
		-v "${API_CONTRACTS_DIR}":/spec \
		-w /spec \
		redocly/cli:latest bundle \
		ingestion/v1/specs.yaml \
		--output ingestion/v1/public/swagger-pact.json \
		--ext json \
		--config .redocly.yaml

.PHONY: generate-api
generate-api: bundle-api ## Regenerate HTTP server stubs from bundled OpenAPI spec
	$(call printMessage,Generating OpenAPI server stubs)
	cd ${SERVICE_DIR}/internal/tools && GOWORK=off ${GO} generate .

.PHONY: migrate-up
migrate-up: ## Apply all pending database migrations
	$(call printMessage,Running migrations up)
	${DOCKER_COMPOSE} -f ${COMPOSE_FILE} run --rm migrate \
		-path /migrations -database "postgres://$${POSTGRES_USER:-ingestion}:$${POSTGRES_PASSWORD:-ingestion}@postgres:5432/$${POSTGRES_DB:-ingestion}?sslmode=disable" up

.PHONY: migrate-down
migrate-down: ## Roll back the last database migration
	$(call printMessage,Rolling back last migration)
	${DOCKER_COMPOSE} -f ${COMPOSE_FILE} run --rm migrate \
		-path /migrations -database "postgres://$${POSTGRES_USER:-ingestion}:$${POSTGRES_PASSWORD:-ingestion}@postgres:5432/$${POSTGRES_DB:-ingestion}?sslmode=disable" down 1
