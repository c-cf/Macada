# Cross-platform: works on Linux (sh), Windows (cmd, powershell via GNU Make)
# Requires: Docker, Go 1.21+, Node.js

.PHONY: up down build runtime-build backend-dev backend-test frontend-dev frontend-install clean

# ---------------------------------------------------------------------------
# Docker
# ---------------------------------------------------------------------------
up: runtime-build
	docker compose up --build -d

down:
	docker compose down

runtime-build:
	docker compose --profile build build runtime

# ---------------------------------------------------------------------------
# Backend  (go -C avoids shell-specific `cd`)
# ---------------------------------------------------------------------------
backend-dev:
	go -C backend run ./cmd/server

backend-test:
	go -C backend test ./... -v -race

backend-build:
	go -C backend build -o bin/server ./cmd/server

backend-tidy:
	go -C backend mod tidy

backend-migrate-up:
	go -C backend run ./cmd/server -migrate-up

backend-migrate-down:
	go -C backend run ./cmd/server -migrate-down

# ---------------------------------------------------------------------------
# Frontend  (npm --prefix avoids shell-specific `cd`)
# ---------------------------------------------------------------------------
frontend-install:
	npm --prefix frontend install

frontend-dev:
	npm --prefix frontend run dev

frontend-build:
	npm --prefix frontend run build

frontend-lint:
	npm --prefix frontend run lint

# ---------------------------------------------------------------------------
# Combo
# ---------------------------------------------------------------------------
build: backend-build frontend-build

# ---------------------------------------------------------------------------
# Clean  (OS-aware rm)
# ---------------------------------------------------------------------------
ifeq ($(OS),Windows_NT)
clean:
	if exist backend\bin rmdir /s /q backend\bin
else
clean:
	rm -rf backend/bin
endif
