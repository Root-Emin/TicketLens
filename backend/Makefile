.PHONY: build run test test-all ml-test ml-lint lint migrate migrate-down \
        docker-up docker-down classifier clean help

APP_NAME := masterfabric
BUILD_DIR := bin
MAIN_PATH := ./cmd/server
MIGRATION_DIR := internal/infrastructure/postgres/migrations
DB_DSN ?= postgres://masterfabric:masterfabric@localhost:5432/masterfabric?sslmode=disable

COMPOSE := docker compose -f deployments/docker-compose.yml
# Datastores and the broker. `classifier` is excluded on purpose: it is the only
# service that needs an image build, and it is opt-in (see the classifier target).
INFRA_SERVICES := postgres redis kafka kafka-ui

# The ML workspace lives inside the backend. Go tooling never sees it (there is
# no Go source under ml/), so its tests need their own target.
ML_DIR := ml
ML_PY := $(ML_DIR)/.venv/bin/python

## help: Show this help message
help:
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/## /  /'

## build: Build the application binary
build:
	@echo "Building $(APP_NAME)..."
	go build -o $(BUILD_DIR)/$(APP_NAME) $(MAIN_PATH)

## run: Run the application
run:
	@echo "Running $(APP_NAME)..."
	go run $(MAIN_PATH)

## test: Run all tests
test:
	@echo "Running tests..."
	go test -v -race -count=1 ./...

## test-cover: Run tests with coverage
test-cover:
	@echo "Running tests with coverage..."
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

## ml-test: Run the Python classifier test suite (ml/)
ml-test:
	@echo "Running ml tests..."
	@test -x $(ML_PY) || { \
		echo "ml/.venv not found. Create it with:"; \
		echo "  cd ml && python3 -m venv .venv && .venv/bin/pip install -e '.[dev]'"; \
		exit 1; \
	}
	cd $(ML_DIR) && .venv/bin/python -m pytest -q

## ml-lint: Lint the Python sources (ml/)
ml-lint:
	@echo "Linting ml..."
	cd $(ML_DIR) && .venv/bin/python -m ruff check src tests

## test-all: Run both the Go and the Python suites
test-all: test ml-test

## lint: Run linter
lint:
	@echo "Running linter..."
	golangci-lint run ./...

## migrate: Run database migrations up
migrate:
	@echo "Running migrations..."
	goose -dir $(MIGRATION_DIR) postgres "$(DB_DSN)" up

## migrate-down: Rollback last migration
migrate-down:
	@echo "Rolling back last migration..."
	goose -dir $(MIGRATION_DIR) postgres "$(DB_DSN)" down

## migrate-status: Show migration status
migrate-status:
	@echo "Migration status..."
	goose -dir $(MIGRATION_DIR) postgres "$(DB_DSN)" status

## docker-up: Start infrastructure (Postgres, Redis, Kafka) — no classifier
docker-up:
	@echo "Starting Docker services..."
	$(COMPOSE) up -d $(INFRA_SERVICES)

## docker-down: Stop Docker Compose services
docker-down:
	@echo "Stopping Docker services..."
	$(COMPOSE) down

## classifier: Build and start the ml/ model service (opt-in)
classifier:
	@echo "Starting classifier..."
	$(COMPOSE) up -d --build classifier

## docker-build: Build Docker image
docker-build:
	@echo "Building Docker image..."
	docker build -f deployments/Dockerfile -t $(APP_NAME):latest .

## clean: Clean build artifacts
clean:
	@echo "Cleaning..."
	rm -rf $(BUILD_DIR) coverage.out coverage.html tmp

## seed: Seed the database with the demo tenant (idempotent)
seed:
	@echo "Seeding database..."
	go run ./cmd/seed
