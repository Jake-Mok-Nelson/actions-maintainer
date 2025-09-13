# Makefile for actions-maintainer

.PHONY: build clean test install help

# Binary name
BINARY_NAME=actions-maintainer

# Build directory
BUILD_DIR=./bin

# Version (can be overridden)
VERSION ?= dev

# Build flags
LDFLAGS=-ldflags "-X main.version=$(VERSION)"

help: ## Show this help message
	@echo "Available commands:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-15s\033[0m %s\n", $$1, $$2}'

build: ## Build the binary
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	@go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/actions-maintainer

build-all: ## Build for multiple platforms
	@echo "Building for multiple platforms..."
	@mkdir -p $(BUILD_DIR)
	@GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 ./cmd/actions-maintainer
	@GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 ./cmd/actions-maintainer
	@GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 ./cmd/actions-maintainer
	@GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe ./cmd/actions-maintainer

install: build ## Install the binary to $GOPATH/bin
	@echo "Installing $(BINARY_NAME)..."
	@cp $(BUILD_DIR)/$(BINARY_NAME) $(GOPATH)/bin/

clean: ## Clean build artifacts
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)
	@rm -f $(BINARY_NAME)
	@go clean

test: ## Run tests
	@echo "Running tests..."
	@go test -v ./...

fmt: ## Format code
	@echo "Formatting code..."
	@go fmt ./...

lint: ## Run linters
	@echo "Running linters..."
	@go vet ./...

deps: ## Download dependencies
	@echo "Downloading dependencies..."
	@go mod download
	@go mod tidy

run: build ## Build and run with example args
	@echo "Running $(BINARY_NAME)..."
	@$(BUILD_DIR)/$(BINARY_NAME)

demo: build ## Run a demo scan (requires GITHUB_TOKEN)
	@if [ -z "$(GITHUB_TOKEN)" ]; then \
		echo "Please set GITHUB_TOKEN environment variable"; \
		exit 1; \
	fi
	@echo "Running demo scan..."
	@$(BUILD_DIR)/$(BINARY_NAME) scan --owner actions --output demo-results.json

# Default target
all: clean deps fmt lint test build ## Run all checks and build