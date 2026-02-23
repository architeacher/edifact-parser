## ──────────────────────────────────────────────────────────────
## Utilities
## ──────────────────────────────────────────────────────────────

.PHONY: help
help: ## Show this help message
	@printf "${BOLD}${CYAN}%-30s${RESET} %s\n" "Target" "Description"
	@printf "%-30s %s\n" "------" "-----------"
	@grep -hE '^[a-zA-Z_/-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "${CYAN}%-30s${RESET} %s\n", $$1, $$2}'

.PHONY: list
list: ## List all make targets
	@$(MAKE) -pRrq -f $(firstword $(MAKEFILE_LIST)) : 2>/dev/null | \
		awk -v RS= -F: '/^# File/,/^# Finished/ {if ($$1 !~ "^[#.]") {print $$1}}' | \
		sort | grep -Ev '(Makefile|\.mk)'

.PHONY: stats
stats: ## Show service Go stats
	@printf "${BOLD}Go module stats for ${SERVICE_NAME}${RESET}\n"
	@cd ${SERVICE_DIR} && ${GO} list -m all | wc -l | xargs printf "  Dependencies: %s\n"
	@find ${SERVICE_DIR} -name '*.go' | wc -l | xargs printf "  Go files: %s\n"

define printMessage
	@printf "${BOLD}${GREEN}▶ %s${RESET}\n" "$(1)"
endef