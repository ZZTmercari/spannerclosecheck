.PHONY: build test install clean lint test-verbose test-coverage help

# Version information
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "v0.1.0")
BUILD_DATE ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
GIT_COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
LDFLAGS := -ldflags "-X main.Version=$(VERSION) -X main.BuildDate=$(BUILD_DATE) -X main.GitCommit=$(GIT_COMMIT)"

# Default target
.DEFAULT_GOAL := help

# Build the analyzer binary
build:
	@echo "Building spannerclosecheck $(VERSION)..."
	go build $(LDFLAGS) -o spannerclosecheck .

# Run all tests
test:
	@echo "Running all tests..."
	go test ./...

# Run tests with verbose output
test-verbose:
	@echo "Running all tests (verbose)..."
	go test -v ./...

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Install the analyzer
install:
	@echo "Installing spannerclosecheck..."
	go install $(LDFLAGS) .

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -f spannerclosecheck
	rm -rf bin/
	rm -f coverage.out coverage.html

# Run linter
lint:
	@echo "Running linter..."
	golangci-lint run

# Run the analyzer on a specific package
run:
	@echo "Running analyzer..."
	go vet -vettool=$$(which spannerclosecheck) ./...

# Help target
help:
	@echo "Available targets:"
	@echo "  make build          - Build the analyzer binary to ./spannerclosecheck"
	@echo "  make test           - Run all tests"
	@echo "  make test-verbose   - Run all tests with verbose output"
	@echo "  make test-coverage  - Run tests with coverage report"
	@echo "  make install        - Install the analyzer to GOPATH/bin"
	@echo "  make clean          - Remove build artifacts"
	@echo "  make lint           - Run golangci-lint"
	@echo "  make run            - Run the analyzer on the current project"
	@echo "  make help           - Show this help message"
