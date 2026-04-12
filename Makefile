.PHONY: lint fix clean vulncheck ci deps deps-update run help

# Variables
BINARY_NAME=kmir
GO=go
GOFLAGS=-v
LINTER=golangci-lint

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: ## Build the binary
	@echo "Building $(BINARY_NAME)..."
	$(GO) build $(GOFLAGS) -o $(BINARY_NAME) .

test: ## Run tests
	@echo "Running tests..."
	$(GO) test -v -race -coverprofile=coverage.out -covermode=atomic ./...

test-short: ## Run short tests only
	@echo "Running short tests..."
	$(GO) test -v -short ./...

lint: ## Run linters
	@echo "Running linters..."
	$(LINTER) run --timeout 5m ./...

fix: ## Auto-fix linting issues
	@echo "Fixing linting issues..."
	$(LINTER) run --fix --timeout 5m ./...

coverage: ## Generate coverage report
	@echo "Generating coverage report..."
	$(GO) test -coverprofile=coverage.out -covermode=atomic ./...
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

coverage-func: ## Show coverage by function
	@echo "Coverage by function:"
	$(GO) test -coverprofile=coverage.out -covermode=atomic ./...
	$(GO) tool cover -func=coverage.out

clean: ## Clean build artifacts
	@echo "Cleaning..."
	rm -f $(BINARY_NAME)
	rm -f coverage.out coverage.html
	rm -f *.test

vulncheck: ## Check for security vulnerabilities
	@echo "Checking for vulnerabilities..."
	$(GO) tool govulncheck ./...

ci: lint vulncheck test ## Run CI checks (lint + vulncheck + test)

deps: ## Download dependencies
	@echo "Downloading dependencies..."
	$(GO) mod download
	$(GO) mod tidy

deps-update: ## Update dependencies
	@echo "Updating dependencies..."
	$(GO) get -u ./...
	$(GO) mod tidy

run: build ## Build and run the binary
	@echo "Running $(BINARY_NAME)..."
	./$(BINARY_NAME)
