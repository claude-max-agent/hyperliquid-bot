.PHONY: all build test lint clean run help

# Build settings
BINARY_NAME := hyperliquid-bot
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS := -ldflags "-X main.version=$(VERSION) -X main.buildTime=$(BUILD_TIME)"

# Go settings
GOBIN := $(shell go env GOPATH)/bin

all: lint test build

## Build the application
build:
	go build $(LDFLAGS) -o bin/$(BINARY_NAME) ./cmd/bot

## Run the application
run: build
	./bin/$(BINARY_NAME) -config config/config.yaml

## Run tests
test:
	go test -v -race -coverprofile=coverage.out ./...

## Run tests with coverage report
test-coverage: test
	go tool cover -html=coverage.out -o coverage.html

## Run linter
lint:
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not installed. Run: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
	fi

## Format code
fmt:
	go fmt ./...
	goimports -w -local github.com/zono819/hyperliquid-bot .

## Download dependencies
deps:
	go mod download
	go mod tidy

## Clean build artifacts
clean:
	rm -rf bin/
	rm -f coverage.out coverage.html

## Install development tools
tools:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install golang.org/x/tools/cmd/goimports@latest

## Show help
help:
	@echo "Available targets:"
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/## /  /'
	@echo ""
	@echo "Usage: make [target]"
