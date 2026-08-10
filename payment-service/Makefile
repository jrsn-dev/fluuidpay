.PHONY: dev test lint migrate build docker-up docker-down clean tidy

# ── Development ──────────────────────────────────────────────────────────────

dev: ## Start the service locally
	go run ./cmd/api

tidy: ## Download and tidy Go modules
	go mod tidy

# ── Build ────────────────────────────────────────────────────────────────────

build: ## Build the binary
	CGO_ENABLED=0 go build -ldflags="-s -w" -o bin/payment-service ./cmd/api

# ── Testing ──────────────────────────────────────────────────────────────────

test: ## Run all tests
	go test -v -race -count=1 ./...

test-cover: ## Run tests with coverage
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# ── Linting ──────────────────────────────────────────────────────────────────

lint: ## Run linters
	golangci-lint run ./...

vet: ## Run go vet
	go vet ./...

# ── Database ─────────────────────────────────────────────────────────────────

DB_DSN ?= postgres://payment:payment@localhost:5432/payment_service?sslmode=disable

migrate-up: ## Run all pending migrations
	@for file in $(sort $(wildcard migrations/*.up.sql)); do \
		echo "Applying $$file..."; \
		psql "$(DB_DSN)" -f "$$file"; \
	done

migrate-create: ## Create a new migration (usage: make migrate-create name=create_foo)
	@touch migrations/$$(date +%Y%m%d%H%M%S)_$(name).up.sql
	@touch migrations/$$(date +%Y%m%d%H%M%S)_$(name).down.sql
	@echo "Created migration files for $(name)"

# ── Docker ───────────────────────────────────────────────────────────────────

docker-up: ## Start all services with Docker Compose
	docker compose up -d

docker-down: ## Stop all services
	docker compose down

docker-build: ## Build Docker image
	docker compose build

docker-logs: ## Follow logs
	docker compose logs -f

# ── Clean ────────────────────────────────────────────────────────────────────

clean: ## Remove build artifacts
	rm -rf bin/ coverage.out coverage.html

# ── Help ─────────────────────────────────────────────────────────────────────

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'
