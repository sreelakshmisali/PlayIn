# PlayHub developer commands.
# Every target here has a plain equivalent documented in README.md, so make is
# a convenience and never a requirement.

SHELL := /bin/sh
COMPOSE := docker compose

.DEFAULT_GOAL := help
.PHONY: help setup up down logs ps rebuild clean \
        migrate-up migrate-down migrate-version migrate-create \
        backend-run backend-test backend-test-db backend-cover backend-lint \
        frontend-install frontend-dev frontend-build frontend-typecheck \
        health

help: ## List available targets
	@grep -hE '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) \
	  | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-22s %s\n", $$1, $$2}'

# --- stack -------------------------------------------------------------------

setup: ## Copy .env.example to .env if it does not exist yet
	@test -f .env || (cp .env.example .env && echo "created .env, edit DB_PASSWORD before use")

up: ## Build and start postgres, run migrations, start api and web
	$(COMPOSE) up --build -d

down: ## Stop everything, keep the database volume
	$(COMPOSE) down

clean: ## Stop everything and delete the database volume
	$(COMPOSE) down -v

logs: ## Follow logs from all services
	$(COMPOSE) logs -f

ps: ## Show container status
	$(COMPOSE) ps

rebuild: ## Rebuild images from scratch
	$(COMPOSE) build --no-cache

health: ## Call the health endpoint
	curl -sS http://localhost:$${API_PORT_HOST:-8080}/api/v1/health

# --- migrations --------------------------------------------------------------

migrate-up: ## Apply all pending migrations
	$(COMPOSE) run --rm migrate up

migrate-down: ## Roll back the most recent migration
	$(COMPOSE) run --rm migrate down 1

migrate-version: ## Print the current schema version
	$(COMPOSE) run --rm migrate version

migrate-create: ## Create a migration pair: make migrate-create name=add_venues
	@test -n "$(name)" || (echo "usage: make migrate-create name=add_venues" && exit 1)
	$(COMPOSE) run --rm migrate create -ext sql -dir /migrations -seq $(name)

# --- backend -----------------------------------------------------------------

backend-run: ## Run the API on the host (needs Go and a running postgres)
	cd backend && go run ./cmd/api

backend-test: ## Run backend tests (repository tests skip without a database)
	cd backend && go test ./... -race

backend-test-db: ## Run backend tests including the repository tests against the compose database
	cd backend && PLAYHUB_TEST_DATABASE_URL="postgres://$${DB_USER:-playhub}:$${DB_PASSWORD}@localhost:$${DB_PORT_HOST:-5432}/$${DB_NAME:-playhub}?sslmode=disable" go test ./... -race

backend-cover: ## Run backend tests and print coverage
	cd backend && go test ./... -coverprofile=coverage.out -covermode=atomic \
	  && go tool cover -func=coverage.out | tail -1

backend-lint: ## Vet the backend
	cd backend && go vet ./...

# --- frontend ----------------------------------------------------------------

frontend-install: ## Install frontend dependencies
	cd frontend && npm ci

frontend-dev: ## Run the Vite dev server on port 5173
	cd frontend && npm run dev

frontend-build: ## Type-check and build the production bundle
	cd frontend && npm run build

frontend-typecheck: ## Type-check without emitting
	cd frontend && npm run typecheck
