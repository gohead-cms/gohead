include common.mk

LDFLAGS += -X "$(MODULE)/cmd.version=$(VERSION)" -X "$(MODULE)/cmd.commit=$(VERSION_HASH)" -X "$(MODULE)/cmd.buildDate=$(DATE)"

## Build:

.PHONY: build
build: ## Build the gohead binary into dist/ (pure Go; run `make ui` first to embed the admin UI)
	$Q CGO_ENABLED=0 $(go) build -trimpath -ldflags '$(LDFLAGS)' -o dist/gohead ./main

.PHONY: ui
ui: ## Build the admin UI into internal/webui/static, to embed it in the binary
	$Q cd gui && npm ci --no-audit --no-fund && npm run build
	$Q find internal/webui/static -mindepth 1 ! -name .keep -exec rm -rf {} +
	$Q cp -R gui/dist/. internal/webui/static/
	$Q rm -f internal/webui/static/config.js
	$Q find internal/webui/static -type f \( -name '*.js' -o -name '*.css' -o -name '*.html' -o -name '*.svg' \) -size +1k -exec gzip -9 -k -f {} \;

.PHONY: standalone
standalone: ui build ## Build the binary with the admin UI, then run `gohead standalone` (data in ./data)
	$Q ./dist/gohead standalone --data data

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

.PHONY: test-db
test-db: ## Run DB tests against PostgreSQL or MySQL (needs GOHEAD_TEST_DATABASE_URL; wipes that database)
	$Q test -n "$$GOHEAD_TEST_DATABASE_URL" || { echo "set GOHEAD_TEST_DATABASE_URL=postgres://... or mysql://..."; exit 1; }
	$Q $(go) test -count=1 -p 1 ./pkg/... ./internal/...

.PHONY: test-postgres
test-postgres: test-db

.PHONY: test-e2e
test-e2e: ## Run the admin UI smoke tests (needs an API on :8080 with an empty database)
	$Q cd gui && npx playwright test

.PHONY: api-contract
api-contract: ## Regenerate the public API snapshots in api/ after an intended API change
	$Q UPDATE_API_CONTRACT=1 $(go) test -count=1 -run TestAPIContract ./internal/openapi ./internal/graphql

.PHONY: helm-lint
helm-lint: ## Lint the Helm chart with every CI values file
	$Q for v in charts/gohead/ci/*-values.yaml; do helm lint --strict charts/gohead -f $$v || exit 1; done

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
