# dotgo Makefile
# Build and development tasks for the dotgo CLI

# Variables
BINARY_NAME=dotgo
MAIN_PATH=.
BUILD_DIR=build
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

# Go variables
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod

# Build flags
LDFLAGS=-ldflags "-X main.Version=$(VERSION) -X main.Commit=$(COMMIT) -X main.BuildTime=$(BUILD_TIME)"

# Platform targets
PLATFORMS=darwin/amd64 darwin/arm64 linux/amd64 linux/arm64 windows/amd64

.PHONY: all build clean test deps lint install uninstall help dev release cross-compile

# Default target
all: clean deps test build

# Build the binary
build:
	@echo "Building $(BINARY_NAME) version $(VERSION)..."
	$(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME) $(MAIN_PATH)
	@echo "Build completed: $(BINARY_NAME)"

# Build for development (with race detector)
dev:
	@echo "Building $(BINARY_NAME) for development..."
	$(GOBUILD) -race $(LDFLAGS) -o $(BINARY_NAME) $(MAIN_PATH)

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	$(GOCLEAN)
	rm -rf $(BUILD_DIR)
	rm -f $(BINARY_NAME)

# Run tests
test:
	@echo "Running tests..."
	$(GOTEST) -v ./...

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	$(GOTEST) -v -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Download dependencies
deps:
	@echo "Downloading dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy

# Lint the code
lint:
	@echo "Running linters..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not found, install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
		exit 1; \
	fi

# Format code
fmt:
	@echo "Formatting code..."
	$(GOCMD) fmt ./...

# Install the binary to GOBIN
install: build
	@echo "Installing $(BINARY_NAME) to $(shell go env GOPATH)/bin..."
	cp $(BINARY_NAME) $(shell go env GOPATH)/bin/

# Uninstall the binary from GOBIN
uninstall:
	@echo "Uninstalling $(BINARY_NAME) from $(shell go env GOPATH)/bin..."
	rm -f $(shell go env GOPATH)/bin/$(BINARY_NAME)

# Cross-compile for multiple platforms
cross-compile: clean deps
	@echo "Cross-compiling for multiple platforms..."
	@mkdir -p $(BUILD_DIR)
	@for platform in $(PLATFORMS); do \
		echo "Building for $$platform..."; \
		GOOS=$$(echo $$platform | cut -d'/' -f1); \
		GOARCH=$$(echo $$platform | cut -d'/' -f2); \
		output_name=$(BUILD_DIR)/$(BINARY_NAME)-$$GOOS-$$GOARCH; \
		if [ $$GOOS = "windows" ]; then output_name="$$output_name.exe"; fi; \
		GOOS=$$GOOS GOARCH=$$GOARCH $(GOBUILD) $(LDFLAGS) -o $$output_name $(MAIN_PATH); \
	done
	@echo "Cross-compilation completed in $(BUILD_DIR)/"

# Create a release
release: clean deps test lint cross-compile
	@echo "Creating release archive..."
	@mkdir -p $(BUILD_DIR)/release
	@for platform in $(PLATFORMS); do \
		GOOS=$$(echo $$platform | cut -d'/' -f1); \
		GOARCH=$$(echo $$platform | cut -d'/' -f2); \
		binary_name=$(BINARY_NAME)-$$GOOS-$$GOARCH; \
		if [ $$GOOS = "windows" ]; then binary_name="$$binary_name.exe"; fi; \
		archive_name=$(BUILD_DIR)/release/$(BINARY_NAME)-$(VERSION)-$$GOOS-$$GOARCH; \
		if [ $$GOOS = "windows" ]; then \
			zip -j $$archive_name.zip $(BUILD_DIR)/$$binary_name README.md LICENSE 2>/dev/null || true; \
		else \
			tar -czf $$archive_name.tar.gz -C $(BUILD_DIR) $$binary_name -C .. README.md LICENSE 2>/dev/null || tar -czf $$archive_name.tar.gz -C $(BUILD_DIR) $$binary_name; \
		fi; \
	done
	@echo "Release archives created in $(BUILD_DIR)/release/"

# Run the binary with test arguments
run: build
	./$(BINARY_NAME) --help

# Run with verbose flag
run-verbose: build
	./$(BINARY_NAME) --verbose --help

# Development watch mode (requires entr)
watch:
	@if command -v entr >/dev/null 2>&1; then \
		echo "Watching for changes... (Press Ctrl+C to stop)"; \
		find . -name '*.go' | entr -r make dev; \
	else \
		echo "entr not found. Install with: brew install entr (macOS) or apt install entr (Ubuntu)"; \
		exit 1; \
	fi

# Show help
help:
	@echo "dotgo Makefile Help"
	@echo "==================="
	@echo ""
	@echo "Available targets:"
	@echo "  build           Build the binary"
	@echo "  dev             Build for development (with race detector)"
	@echo "  clean           Clean build artifacts"
	@echo "  test            Run tests"
	@echo "  test-coverage   Run tests with coverage report"
	@echo "  deps            Download and tidy dependencies"
	@echo "  lint            Run code linters"
	@echo "  fmt             Format code"
	@echo "  install         Install binary to GOPATH/bin"
	@echo "  uninstall       Remove binary from GOPATH/bin"
	@echo "  cross-compile   Build for multiple platforms"
	@echo "  release         Create release builds and archives"
	@echo "  run             Build and run with --help"
	@echo "  run-verbose     Build and run with --verbose --help"
	@echo "  watch           Watch for changes and rebuild (requires entr)"
	@echo "  help            Show this help message"
	@echo ""
	@echo "Variables:"
	@echo "  VERSION         $(VERSION)"
	@echo "  COMMIT          $(COMMIT)"
	@echo "  BUILD_TIME      $(BUILD_TIME)"