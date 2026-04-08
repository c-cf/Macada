# Cross-platform: works on Linux (sh), Windows (cmd, powershell via GNU Make)
# Run from the backend/ directory.

.PHONY: build run dev migrate-up migrate-down test tidy clean

build:
	go build -o bin/server ./cmd/server

run: build
	./bin/server

dev:
	go run ./cmd/server

migrate-up:
	go run ./cmd/server -migrate-up

migrate-down:
	go run ./cmd/server -migrate-down

test:
	go test ./... -v -race

tidy:
	go mod tidy

# OS-aware clean
ifeq ($(OS),Windows_NT)
clean:
	if exist bin rmdir /s /q bin
else
clean:
	rm -rf bin/
endif
