.PHONY: build run test clean help install-deps build-all

# Variables
APP_NAME := scanserver
CMD_PATH := cmd/scanserver
BUILD_DIR := build
GO := go

# Default target
.DEFAULT_GOAL := help

help: ## Show this help message
	@echo "Scanner Service - Makefile commands:"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'

install-deps: ## Install Go dependencies
	@echo "Installing dependencies..."
	$(GO) mod download
	$(GO) mod tidy
	@echo "Done!"

build: ## Build for current platform
	@echo "Building $(APP_NAME) for current platform..."
	$(GO) build -o $(BUILD_DIR)/$(APP_NAME) ./$(CMD_PATH)
	@echo "Binary created: $(BUILD_DIR)/$(APP_NAME)"

build-all: ## Build for all platforms (Windows, Linux, macOS)
	@echo "Building for all platforms..."
	@bash scripts/build.sh

build-windows: ## Build for Windows only
	@echo "Building for Windows..."
	GOOS=windows GOARCH=amd64 $(GO) build -o $(BUILD_DIR)/$(APP_NAME)-windows-amd64.exe ./$(CMD_PATH)

build-linux: ## Build for Linux only
	@echo "Building for Linux..."
	GOOS=linux GOARCH=amd64 $(GO) build -o $(BUILD_DIR)/$(APP_NAME)-linux-amd64 ./$(CMD_PATH)

build-macos: ## Build for macOS only
	@echo "Building for macOS..."
	GOOS=darwin GOARCH=amd64 $(GO) build -o $(BUILD_DIR)/$(APP_NAME)-darwin-amd64 ./$(CMD_PATH)
	GOOS=darwin GOARCH=arm64 $(GO) build -o $(BUILD_DIR)/$(APP_NAME)-darwin-arm64 ./$(CMD_PATH)

run: ## Run the application
	@echo "Running $(APP_NAME)..."
	$(GO) run ./$(CMD_PATH)

run-dev: ## Run with development settings
	@echo "Running $(APP_NAME) in development mode..."
	$(GO) run ./$(CMD_PATH) -host localhost -port 8080

test: ## Run tests
	@echo "Running tests..."
	$(GO) test -v -race -coverprofile=coverage.out ./...
	@echo "Coverage report: coverage.out"

test-coverage: test ## Run tests and show coverage
	$(GO) tool cover -html=coverage.out

clean: ## Clean build artifacts
	@echo "Cleaning build directory..."
	rm -rf $(BUILD_DIR)
	rm -f coverage.out
	@echo "Done!"

fmt: ## Format Go code
	@echo "Formatting code..."
	$(GO) fmt ./...
	@echo "Done!"

lint: ## Run linter (requires golangci-lint)
	@echo "Running linter..."
	golangci-lint run ./...

vet: ## Run go vet
	@echo "Running go vet..."
	$(GO) vet ./...

deps-update: ## Update dependencies
	@echo "Updating dependencies..."
	$(GO) get -u ./...
	$(GO) mod tidy
	@echo "Done!"

docker-build: ## Build Docker image
	@echo "Building Docker image..."
	docker build -t $(APP_NAME):latest .

docker-run: ## Run in Docker container
	@echo "Running Docker container..."
	docker run -p 8080:8080 $(APP_NAME):latest

setup: install-deps ## Setup development environment
	@echo "Setting up development environment..."
	@mkdir -p web/static web/templates scans
	@echo "Done! Run 'make run' to start the server."
