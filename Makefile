.PHONY: all build test test-integration test-race test-race-integration test-coverage lint clean install dev run help deps fmt scrape ship-it release release-all

# Variables
BINARY_NAME=skulto
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
POSTHOG_API_KEY ?= $(SKULTO_POSTHOG_API_KEY)
LDFLAGS=-ldflags "-s -w -X github.com/asteroid-belt/skulto/pkg/version.Version=$(VERSION) -X github.com/asteroid-belt/skulto/pkg/version.Commit=$(COMMIT) -X github.com/asteroid-belt/skulto/pkg/version.BuildDate=$(BUILD_DATE) -X github.com/asteroid-belt/skulto/internal/telemetry.PostHogAPIKey=$(POSTHOG_API_KEY) -X github.com/asteroid-belt/skulto/internal/telemetry.Version=$(VERSION)"

# Go settings
GO=go
GOFLAGS=-v
CGO_ENABLED=0

# Directories
BUILD_DIR=./build
RELEASE_DIR=./release
CMD_DIR=./cmd/skulto

# Default target
all: build

## build: Build the binary
build:
	@echo "💀 Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=$(CGO_ENABLED) $(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(CMD_DIR)
	@echo "✅ Built $(BUILD_DIR)/$(BINARY_NAME)"

## release: Build skulto for specified platform (GOOS=linux|darwin GOARCH=amd64|arm64)
release:
ifndef GOOS
	$(error GOOS is required. Usage: make release GOOS=linux GOARCH=amd64)
endif
ifndef GOARCH
	$(error GOARCH is required. Usage: make release GOOS=linux GOARCH=amd64)
endif
	@GOOS=$(GOOS) GOARCH=$(GOARCH) ./scripts/release.sh skulto

## release-all: Build all artifacts for specified platform (GOOS=linux|darwin GOARCH=amd64|arm64)
release-all:
ifndef GOOS
	$(error GOOS is required. Usage: make release-all GOOS=linux GOARCH=amd64)
endif
ifndef GOARCH
	$(error GOARCH is required. Usage: make release-all GOOS=linux GOARCH=amd64)
endif
	@GOOS=$(GOOS) GOARCH=$(GOARCH) ./scripts/release.sh all

release-easy:
	@GOOS=$(shell uname | tr '[:upper:]' '[:lower:]') GOARCH=$(shell uname -m | sed 's/x86_64/amd64/' | sed 's/aarch64/arm64/') ./scripts/release.sh skulto

## clean-release: Remove release artifacts
clean-release:
	@echo "🧹 Cleaning release artifacts..."
	@rm -rf $(RELEASE_DIR)
	@echo "✅ Release artifacts cleaned"

## dev: Build for development (with race detector, requires CGO)
dev:
	@echo "💀 Building $(BINARY_NAME) for development..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=1 $(GO) build $(GOFLAGS) -race $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(CMD_DIR)

## test: Run all tests including integration tests that require network access
test:
	@echo "🧪 Running tests with coverage..."
	$(GO) test -v -coverprofile=coverage.out ./internal/...
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "📊 Coverage report: coverage.html"

## test-race: Run tests with race detector (requires CGO, excludes integration tests)
test-race:
	@echo "🧪 Running tests with race detector..."
	CGO_ENABLED=1 $(GO) test -v -race ./internal/...
	@echo "✅ Tests passed"

## lint: Run linters
lint:
	@echo "🔍 Running linters..."
	./bin/golangci-lint run --timeout=5m
	@echo "✅ Linting passed"

## fmt: Format code
format:
	@echo "📝 Formatting code..."
	$(GO) fmt ./...
	@echo "✅ Code formatted"

## clean: Remove build artifacts
clean:
	@echo "🧹 Cleaning..."
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html
	$(GO) clean -cache
	@echo "✅ Cleaned"

## install: Install binary to GOPATH/bin
install: build
	@echo "📦 Installing $(BINARY_NAME)..."
	cp $(BUILD_DIR)/$(BINARY_NAME) $(GOPATH)/bin/
	@echo "✅ Installed to $(GOPATH)/bin/$(BINARY_NAME)"

## deps: Download dependencies
deps:
	@echo "📦 Downloading dependencies..."
	$(GO) mod download
	$(GO) mod tidy
	curl -sSfL https://golangci-lint.run/install.sh | sh -s v2.7.2
	@echo "✅ Dependencies ready"

ci: deps build test

## run: Build and run
run: build
	@echo "🏃 Running $(BINARY_NAME)..."
	$(BUILD_DIR)/$(BINARY_NAME)

## ship-it: Push to remote after checking for unstaged files
ship_it: build lint test
	chmod +x ./scripts/ship-it.sh
	@./scripts/ship-it.sh

## help: Show this help
help:
	@echo "💀 SKULTO Makefile"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@sed -n 's/^##//p' $(MAKEFILE_LIST) | column -t -s ':' | sed -e 's/^/ /'

# Default help
.DEFAULT_GOAL := help