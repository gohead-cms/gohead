include common.mk

LDFLAGS += -X "$(MODULE)/cmd.version=$(VERSION)" -X "$(MODULE)/cmd.commit=$(VERSION_HASH)" -X "$(MODULE)/cmd.buildDate=$(DATE)"

## Build:

.PHONY: build
build: ## Build the gohead binary into dist/ (cgo on, for SQLite)
	$Q CGO_ENABLED=1 $(go) build -trimpath -ldflags '$(LDFLAGS)' -o dist/gohead ./main

.PHONY: run
run: build ## Build and start the API with config.yaml
	$Q ./dist/gohead start --config config.yaml

.PHONY: docker
docker: ## Build the API and admin UI images locally
	$Q docker build -t ghcr.io/gohead-cms/gohead:dev --build-arg VERSION=$(VERSION) --build-arg COMMIT=$(VERSION_HASH) .
	$Q docker build -t ghcr.io/gohead-cms/gohead-admin:dev gui

.PHONY: up
up: ## Start the full stack with Docker Compose (builds from source)
	$Q docker compose up -d --build

.PHONY: down
down: ## Stop the Docker Compose stack
	$Q docker compose down

## Test:

.PHONY: unit-test
unit-test: ## Run the Go test suite
	$Q $(go) test -count=1 ./...

.PHONY: test-race
test-race: ## Run the Go test suite with the race detector
	$Q $(go) test -race -count=1 ./...

.PHONY: test-postgres
test-postgres: ## Run DB tests against PostgreSQL (needs GOHEAD_TEST_DATABASE_URL)
	$Q test -n "$$GOHEAD_TEST_DATABASE_URL" || { echo "set GOHEAD_TEST_DATABASE_URL=postgres://..."; exit 1; }
	$Q $(go) test -count=1 -p 1 ./pkg/storage/... ./internal/api/... ./internal/graphql/...

.PHONY: lint
lint: ## gofmt check and go vet
	$Q test -z "$$(gofmt -l $$(git ls-files '*.go'))" || { gofmt -l $$(git ls-files '*.go'); exit 1; }
	$Q $(go) vet ./...

## Admin UI:

.PHONY: gui
gui: ## Install dependencies and build the admin UI
	$Q cd gui && npm ci && npm run build

.PHONY: gui-dev
gui-dev: ## Run the admin UI dev server
	$Q cd gui && npm run dev

## Docs:

.PHONY: docs
docs: ## Serve the handbook locally on http://localhost:8000
	$Q pip install -q -r docs/requirements.txt && mkdocs serve -f docs/mkdocs.yml

## Release:

.PHONY: bump-version
bump-version: ## Bump app version
	$Q ./scripts/bump_version.sh

## Help:
help: ## Show this help
	@echo ''
	@echo 'Usage:'
	@echo '  ${YELLOW}make${RESET} ${GREEN}<target> [options]${RESET}'
	@echo ''
	@echo 'Options:'
	@$(call global_option, "V [0|1]", "enable verbose mode (default:0)")
	@echo ''
	@echo 'Targets:'
	@awk 'BEGIN {FS = ":.*?## "} { \
		if (/^[a-zA-Z_-]+:.*?##.*$$/) {printf "    ${YELLOW}%-20s${GREEN}%s${RESET}\n", $$1, $$2} \
		else if (/^## .*$$/) {printf "  ${CYAN}%s${RESET}\n", substr($$1,4)} \
		}' $(MAKEFILE_LIST)
