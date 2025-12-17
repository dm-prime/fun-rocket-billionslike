.PHONY: help clean build run dev watch test test-verbose fmt lint vet install deps

# Build output in repo root to avoid mkdir on Windows shells
APP_EXE := main.exe

help:
	@echo Available targets:
	@echo   make help        - show this help message
	@echo   make deps        - download and install dependencies
	@echo   make build       - build to .\$(APP_EXE)
	@echo   make run         - build then run .\$(APP_EXE)
	@echo   make dev         - watch files and rebuild on changes (uses compile-daemon)
	@echo   make test        - run tests
	@echo   make test-verbose - run tests with verbose output
	@echo   make fmt         - format code with gofmt
	@echo   make lint        - run golangci-lint (if installed)
	@echo   make vet         - run go vet
	@echo   make clean       - remove build artifacts and caches

deps:
	@echo "Downloading dependencies..."
	go mod download
	go mod tidy

build:
	@echo "Building $(APP_EXE)..."
	go build -o .\$(APP_EXE) .

run: build
	@echo "Running $(APP_EXE)..."
	.\$(APP_EXE)

dev:
	@echo "Starting file watcher with compile-daemon..."
	go run github.com/githubnemo/CompileDaemon -command=".\$(APP_EXE)" -build="go build -o .\$(APP_EXE) ." -include="*.go" -exclude-dir="tmp,vendor"

watch: dev

test:
	@echo "Running tests..."
	go test ./...

test-verbose:
	@echo "Running tests with verbose output..."
	go test -v ./...

fmt:
	@echo "Formatting code..."
	go fmt ./...

lint:
	@where golangci-lint >nul 2>&1 && golangci-lint run || echo "golangci-lint not installed. Install it with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"

vet:
	@echo "Running go vet..."
	go vet ./...

clean:
	@echo "Cleaning build artifacts..."
	go clean -cache -testcache
	@if exist $(APP_EXE) del /F /Q $(APP_EXE) 2>nul || rm -f $(APP_EXE) 2>/dev/null || true
	@echo "Clean complete."

