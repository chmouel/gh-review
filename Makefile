.PHONY: build install test clean lint fmt help

BINARY_NAME=gh-review
GO=go
GOFLAGS=-v

all: build

build: ## Build the binary
	$(GO) build $(GOFLAGS) -o $(BINARY_NAME) .

help: ## Display this help message
	@echo "Usage: make [target]"
	@echo ""
	@echo "Available targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-15s %s\n", $$1, $$2}'

install: build ## Install the extension to gh
	gh extension install .

uninstall: ## Uninstall the extension from gh
	gh extension remove review

test: ## Run tests
	$(GO) test -v ./...

test-coverage: ## Run tests with coverage
	$(GO) test -v -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out -o coverage.html

fumpt: ## Run gofumpt to format code
	gofumpt -w .

lint: ## Run linter
	golangci-lint run

fmt: ## Format code
	$(GO) fmt ./...
	gofumpt -w .

clean: ## Clean build artifacts
	rm -f $(BINARY_NAME)
	rm -f coverage.out coverage.html
	rm -rf dist/

deps: ## Download dependencies
	$(GO) mod download
	$(GO) mod tidy

dev: build ## Build and run (useful for testing)
	./$(BINARY_NAME)

.DEFAULT_GOAL := help
