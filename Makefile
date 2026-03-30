VERSION   ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT    ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS   := -X github.com/postulate/api/internal/handler.version=$(VERSION) \
             -X github.com/postulate/api/internal/handler.commit=$(COMMIT) \
             -X github.com/postulate/api/internal/handler.buildTime=$(BUILD_TIME)

MODULES := ./api/... ./cli/... ./sdk/... ./plugins/platform-standards/...

# Default database URL for local development. Override with DB_URL=<url>.
DB_URL ?= postgres://postulate_dev:postulate_dev@localhost:5432/postulate_dev?sslmode=disable

# Path to migration files (relative to repo root).
MIGRATIONS_PATH := api/internal/migrate/migrations

.PHONY: build run test lint tidy docker-build docker-run docker-scan help \
        db-setup db-start db-stop db-status db-reset hooks \
        migrate-up migrate-down migrate-down-all migrate-status migrate-create migrate-version \
        install-tools

## hooks: install git hooks from .githooks/
hooks:
	git config core.hooksPath .githooks
	@echo "✓ git hooks installed"

## build: build all Go modules in the workspace
build:
	go build $(MODULES)
	go build -ldflags "$(LDFLAGS)" -o bin/postulate-api ./api/cmd/api/...

## run: start the API server using api/config.yaml
run: build
	./bin/postulate-api

## test: run all tests across the workspace with race detection
test:
	go test -race $(MODULES)

## lint: run golangci-lint across all modules
lint:
	cd api && golangci-lint run ./...
	cd cli && golangci-lint run ./...
	cd sdk && golangci-lint run ./...
	cd plugins/platform-standards && golangci-lint run ./...

## tidy: run go mod tidy across all modules
tidy:
	cd api && go mod tidy
	cd cli && go mod tidy
	cd sdk && go mod tidy
	cd plugins/platform-standards && go mod tidy

## help: print available targets
help:
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/## //' | column -t -s ':'

## docker-build: build the API container image and tag it as postulate-api:local
docker-build:
	docker build \
		--build-arg VERSION=$(VERSION) \
		--build-arg COMMIT=$(COMMIT) \
		--build-arg BUILD_TIME=$(BUILD_TIME) \
		-t postulate-api:local \
		./api

## docker-run: start the API and dependencies via docker-compose
docker-run:
	docker compose -f api/docker-compose.yml up

## docker-scan: scan postulate-api:local for CRITICAL CVEs using trivy
docker-scan:
	trivy image --exit-code 1 --severity CRITICAL postulate-api:local

## db-setup: create local PostgreSQL roles and databases
db-setup:
	@bash scripts/db-setup.sh
	@echo "✓ db-setup complete"

## db-start: start the local PostgreSQL service via Homebrew
db-start:
	brew services start postgresql@16
	@echo "✓ PostgreSQL started"

## db-stop: stop the local PostgreSQL service via Homebrew
db-stop:
	brew services stop postgresql@16
	@echo "✓ PostgreSQL stopped"

## db-status: print the current PostgreSQL service status
db-status:
	brew services info postgresql@16

## db-reset: drop and recreate postulate_dev and postulate_test databases (prompts for confirmation)
db-reset:
	@read -p "This will DROP postulate_dev and postulate_test. Type 'yes' to confirm: " confirm; \
	if [ "$$confirm" = "yes" ]; then \
		psql postgres -c "DROP DATABASE IF EXISTS postulate_dev;"; \
		psql postgres -c "DROP DATABASE IF EXISTS postulate_test;"; \
		psql postgres -c "CREATE DATABASE postulate_dev OWNER postulate_dev;"; \
		psql postgres -c "CREATE DATABASE postulate_test OWNER postulate_dev;"; \
		echo "✓ Databases recreated"; \
	else \
		echo "Aborted."; \
	fi

## install-tools: install developer tools (golang-migrate CLI)
install-tools:
	go install -tags 'pgx5' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
	@echo "✓ tools installed"

## migrate-up: apply all pending migrations to postulate_dev (override with DB_URL=<url>)
migrate-up:
	migrate -path $(MIGRATIONS_PATH) -database "$(DB_URL)" up

## migrate-down: roll back the most recent migration
migrate-down:
	migrate -path $(MIGRATIONS_PATH) -database "$(DB_URL)" down 1

## migrate-down-all: roll back all applied migrations (prompts for confirmation)
migrate-down-all:
	@read -p "This will roll back ALL migrations. Type 'yes' to confirm: " confirm; \
	if [ "$$confirm" = "yes" ]; then \
		migrate -path $(MIGRATIONS_PATH) -database "$(DB_URL)" down -all; \
	else \
		echo "Aborted."; \
	fi

## migrate-status: show applied and pending migration status
migrate-status:
	migrate -path $(MIGRATIONS_PATH) -database "$(DB_URL)" status

## migrate-version: print the current schema version
migrate-version:
	migrate -path $(MIGRATIONS_PATH) -database "$(DB_URL)" version

## migrate-create name=<description>: create a new numbered migration file pair
migrate-create:
	@if [ -z "$(name)" ]; then \
		echo "Usage: make migrate-create name=<description>"; \
		exit 1; \
	fi
	migrate create -ext sql -dir $(MIGRATIONS_PATH) -seq $(name)

