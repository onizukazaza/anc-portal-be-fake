# =============================================================================
# anc-portal-be — Command Reference
# =============================================================================
# ใช้: make <target>
# ดูคำสั่งทั้งหมด: make help
# =============================================================================

.PHONY: help dev build test test-cover lint ci migrate seed import worker \
        otel-up otel-down otel-down-v otel-logs otel-ps \
        local-up local-down local-down-v local-logs local-ps \
        docker-build docker-build-worker \
        tidy clean

# >> Default
help: ## แสดงคำสั่งทั้งหมด
	@echo ""
	@echo "  anc-portal-be — Available Commands"
	@echo "  ==================================="
	@echo ""
	@echo "  Development:"
	@echo "    make dev           Run API with hot-reload (air)"
	@echo "    make build         Build API binary"
	@echo "    make test          Run all tests"
	@echo "    make test-cover    Run tests with coverage report"
	@echo "    make lint          Run golangci-lint"
	@echo "    make ci            Run full CI pipeline locally (lint→test→vuln→build)"	@echo "    make tidy          go mod tidy"
	@echo "    make clean         Remove build artifacts"
	@echo ""
	@echo "  Database:"
	@echo "    make migrate       Run database migrations"
	@echo "    make seed          Run data seeding"
	@echo "    make import        Import CSV data (requires ENV, PATH, TYPE)"
	@echo ""
	@echo "  Worker:"
	@echo "    make worker        Run background worker"
	@echo ""
	@echo "  Observability (OTel + Grafana):"
	@echo "    make otel-up       Start observability stack"
	@echo "    make otel-down     Stop observability stack"
	@echo "    make otel-down-v   Stop + remove volumes"
	@echo "    make otel-logs     View stack logs"
	@echo "    make otel-ps       Show running containers"
	@echo ""
	@echo "  Local Dev Stack (PostgreSQL + Redis + Kafka):"
	@echo "    make local-up      Start local dependencies"
	@echo "    make local-down    Stop local dependencies"
	@echo "    make local-down-v  Stop + remove volumes"
	@echo "    make local-logs    View stack logs"
	@echo "    make local-ps      Show running containers"
	@echo ""
	@echo "  Docker Build:"
	@echo "    make docker-build         Build API Docker image"
	@echo "    make docker-build-worker  Build Worker Docker image"
	@echo ""

# =============================================================================
# >> Development
# =============================================================================

dev: ## รัน API server ด้วย Air (hot-reload)
	air -c .air.local.toml

GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo dev)
BUILD_TIME := $(shell date -u +%Y-%m-%dT%H:%M:%SZ 2>/dev/null || echo unknown)
LDFLAGS    := -X github.com/onizukazaza/anc-portal-be-fake/pkg/buildinfo.GitCommit=$(GIT_COMMIT) \
              -X github.com/onizukazaza/anc-portal-be-fake/pkg/buildinfo.BuildTime=$(BUILD_TIME)

build: ## Build API binary
	go build -ldflags="$(LDFLAGS)" -o ./tmp/main.exe ./cmd/api

test: ## Run all tests
	go test ./...

COVERAGE_THRESHOLD ?= 70
test-cover: ## Run tests with coverage report (threshold: $(COVERAGE_THRESHOLD)%)
	@echo ">> Running tests with coverage..."
	@go test -coverprofile=coverage.out -covermode=atomic ./...
	@echo ""
	@echo ">> Coverage by package:"
	@go tool cover -func=coverage.out
	@echo ""
	@TOTAL=$$(go tool cover -func=coverage.out | grep total | awk '{print $$NF}' | tr -d '%'); \
	echo ">> Total coverage: $${TOTAL}%"; \
	if [ $$(echo "$${TOTAL} < $(COVERAGE_THRESHOLD)" | bc -l) -eq 1 ]; then \
		echo ">> FAIL: coverage $${TOTAL}% is below threshold $(COVERAGE_THRESHOLD)%"; \
		exit 1; \
	else \
		echo ">> PASS: coverage $${TOTAL}% meets threshold $(COVERAGE_THRESHOLD)%"; \
	fi

lint: ## Run golangci-lint (ต้องติดตั้ง: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest)
	golangci-lint run ./...

ci: lint test ## Run full CI pipeline locally (lint → test → vuln → build)
	@echo ""
	@echo ">> [3/4] Vulnerability check..."
	govulncheck ./...
	@echo ""
	@echo ">> [4/4] Build binaries..."
	go build -ldflags="$(LDFLAGS)" -o ./tmp/main.exe ./cmd/api
	go build -ldflags="$(LDFLAGS)" -o ./tmp/worker.exe ./cmd/worker
	go build -ldflags="$(LDFLAGS)" -o ./tmp/migrate.exe ./cmd/migrate
	go build -ldflags="$(LDFLAGS)" -o ./tmp/seed.exe ./cmd/seed
	go build -ldflags="$(LDFLAGS)" -o ./tmp/import.exe ./cmd/import
	@echo ""
	@echo "=========================================="
	@echo "  CI Pipeline PASSED (all 4 steps)"  
	@echo "=========================================="
	@echo ""

swagger: ## Generate Swagger docs (swag init)
	swag init -g cmd/api/main.go -o docs --parseDependency --parseInternal

tidy: ## go mod tidy
	go mod tidy

clean: ## ลบ build artifacts
	@rm -rf tmp

# =============================================================================
# >> Database
# =============================================================================

migrate: ## Run database migrations
	go run ./cmd/migrate

seed: ## Run data seeding
	go run ./cmd/seed

import: ## Import CSV data (example: make import ENV=.env.local PATH=./base_data/users.csv TYPE=user)
	go run ./cmd/import/main.go --env $(ENV) --path $(PATH) --service_type $(TYPE)

# =============================================================================
# >> Worker
# =============================================================================

worker: ## Run background worker (Kafka consumer)
	go run ./cmd/worker

# =============================================================================
# >> Observability Stack (OTel + Grafana)
# =============================================================================

OTEL_DIR = deployments/observability

otel-up: ## Start observability stack (Grafana + Tempo + Prometheus + OTel Collector)
	docker compose -f $(OTEL_DIR)/docker-compose.yaml --env-file $(OTEL_DIR)/.env up -d

otel-down: ## Stop observability stack
	docker compose -f $(OTEL_DIR)/docker-compose.yaml --env-file $(OTEL_DIR)/.env down

otel-down-v: ## Stop observability stack + remove data volumes
	docker compose -f $(OTEL_DIR)/docker-compose.yaml --env-file $(OTEL_DIR)/.env down -v

otel-logs: ## View observability stack logs (follow)
	docker compose -f $(OTEL_DIR)/docker-compose.yaml logs -f

otel-ps: ## Show observability stack status
	docker compose -f $(OTEL_DIR)/docker-compose.yaml ps

# =============================================================================
# >> Local Dev Stack (PostgreSQL + Redis + Kafka)
# =============================================================================

LOCAL_DIR = deployments/local

local-up: ## Start local dependencies (PostgreSQL + Redis + Kafka + Kafka UI)
	docker compose -f $(LOCAL_DIR)/docker-compose.yaml --env-file $(LOCAL_DIR)/.env up -d

local-down: ## Stop local dependencies
	docker compose -f $(LOCAL_DIR)/docker-compose.yaml --env-file $(LOCAL_DIR)/.env down

local-down-v: ## Stop local dependencies + remove data volumes
	docker compose -f $(LOCAL_DIR)/docker-compose.yaml --env-file $(LOCAL_DIR)/.env down -v

local-logs: ## View local stack logs (follow)
	docker compose -f $(LOCAL_DIR)/docker-compose.yaml logs -f

local-ps: ## Show local stack status
	docker compose -f $(LOCAL_DIR)/docker-compose.yaml ps

# =============================================================================
# >> Docker Build
# =============================================================================

DOCKER_DIR = deployments/docker

docker-build: ## Build API Docker image
	docker build -f $(DOCKER_DIR)/Dockerfile \
		--build-arg GIT_COMMIT=$(GIT_COMMIT) \
		--build-arg BUILD_TIME=$(BUILD_TIME) \
		-t anc-portal-be .

docker-build-worker: ## Build Worker Docker image (from main Dockerfile multi-target)
	docker build -f $(DOCKER_DIR)/Dockerfile --target worker \
		--build-arg GIT_COMMIT=$(GIT_COMMIT) \
		--build-arg BUILD_TIME=$(BUILD_TIME) \
		-t anc-portal-worker .
