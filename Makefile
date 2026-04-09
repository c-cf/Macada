# Cross-platform: works on Linux (sh), Windows (cmd, powershell via GNU Make)
# Requires: Docker, Go 1.21+, Node.js

.PHONY: help up down build runtime-build \
        backend-dev backend-test backend-build backend-tidy backend-migrate-up backend-migrate-down \
        frontend-install frontend-dev frontend-build frontend-lint \
        prod-up prod-down prod-init \
        clean

# Default target
.DEFAULT_GOAL := help

# ---------------------------------------------------------------------------
# Help  — print all targets that have a ## description
# ---------------------------------------------------------------------------
help: ## Show available make targets
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-24s\033[0m %s\n", $$1, $$2}'

# ---------------------------------------------------------------------------
# Docker (development)
# ---------------------------------------------------------------------------
up: runtime-build ## Start all dev services (builds images first)
	docker compose up --build -d

down: ## Stop all dev services
	docker compose down

runtime-build: ## Build the agent runtime Docker image
	docker compose --profile build build runtime

# ---------------------------------------------------------------------------
# Backend  (go -C avoids shell-specific `cd`)
# ---------------------------------------------------------------------------
backend-dev: ## Run backend server in development mode
	go -C backend run ./cmd/server

backend-test: ## Run backend tests with race detection
	go -C backend test ./... -v -race

backend-build: ## Compile backend binary to backend/bin/server
	go -C backend build -o bin/server ./cmd/server

backend-tidy: ## Tidy Go module dependencies
	go -C backend mod tidy

backend-migrate-up: ## Apply all pending database migrations
	go -C backend run ./cmd/server -migrate-up

backend-migrate-down: ## Roll back the last database migration
	go -C backend run ./cmd/server -migrate-down

# ---------------------------------------------------------------------------
# Frontend  (npm --prefix avoids shell-specific `cd`)
# ---------------------------------------------------------------------------
frontend-install: ## Install frontend Node.js dependencies
	npm --prefix frontend install

frontend-dev: ## Start frontend dev server (hot-reload on :3000)
	npm --prefix frontend run dev

frontend-build: ## Build frontend for production
	npm --prefix frontend run build

frontend-lint: ## Run ESLint + TypeScript type-checker
	npm --prefix frontend run lint

# ---------------------------------------------------------------------------
# Production
# ---------------------------------------------------------------------------
prod-init: ## Pull runtime image before first prod start (run once)
	docker compose -f docker-compose.prod.yml --env-file .env.prod \
		--profile init up runtime

prod-up: ## Start all production services (requires .env.prod)
	docker compose -f docker-compose.prod.yml --env-file .env.prod up -d --build

prod-down: ## Stop all production services
	docker compose -f docker-compose.prod.yml --env-file .env.prod down

# ---------------------------------------------------------------------------
# Combo
# ---------------------------------------------------------------------------
build: backend-build frontend-build ## Build both backend and frontend

# ---------------------------------------------------------------------------
# Clean  (OS-aware rm)
# ---------------------------------------------------------------------------
ifeq ($(OS),Windows_NT)
clean: ## Remove compiled artifacts
	if exist backend\bin rmdir /s /q backend\bin
else
clean:
	rm -rf backend/bin
endif
