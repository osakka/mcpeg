# MCPEG Makefile - Delegates to build script (Single Source of Truth)
# All build logic is centralized in scripts/build.sh

BUILD_SCRIPT := ./scripts/build.sh

# Default target
.PHONY: all
all: build

# Core build targets
.PHONY: build build-dev build-prod
build:
	@$(BUILD_SCRIPT) build

build-dev:
	@$(BUILD_SCRIPT) build-dev

build-prod:
	@$(BUILD_SCRIPT) build-prod

# Development and testing
.PHONY: test coverage fmt tidy dev
test:
	@$(BUILD_SCRIPT) test

coverage:
	@$(BUILD_SCRIPT) coverage

fmt:
	@$(BUILD_SCRIPT) fmt

tidy:
	@$(BUILD_SCRIPT) tidy

dev:
	@$(BUILD_SCRIPT) dev

# Code generation and validation
.PHONY: generate validate
generate:
	@$(BUILD_SCRIPT) generate

validate:
	@$(BUILD_SCRIPT) validate

# Release and distribution
.PHONY: release clean
release:
	@$(BUILD_SCRIPT) release

clean:
	@$(BUILD_SCRIPT) clean

# Information and help
.PHONY: info help
info:
	@$(BUILD_SCRIPT) info

help:
	@$(BUILD_SCRIPT) help

# Install to GOPATH (not delegated, as it's a Go-specific pattern)
.PHONY: install
install:
	@echo "Installing MCPEG binary to GOPATH/bin..."
	@go install -trimpath -ldflags "-X main.Version=$$($(BUILD_SCRIPT) info | grep Version | cut -d: -f2 | xargs)" ./cmd/mcpeg

# Development tools (optional, if available)
.PHONY: lint security deps bench
lint:
	@echo "Running linting..."
	@if command -v golangci-lint >/dev/null 2>&1; then golangci-lint run; else echo "golangci-lint not installed, skipping"; fi

security:
	@echo "Running security scan..."
	@if command -v gosec >/dev/null 2>&1; then gosec ./...; else echo "gosec not installed, skipping"; fi

deps:
	@echo "Checking dependencies..."
	@go mod verify
	@go list -m -u all

bench:
	@echo "Running benchmarks..."
	@go test -bench=. -benchmem ./...

# Docker (not delegated, as it's environment-specific)
.PHONY: docker
docker:
	@echo "Building Docker image..."
	@docker build -t mcpeg:latest .
	@docker build -t mcpeg:$$($(BUILD_SCRIPT) info | grep Version | cut -d: -f2 | xargs) .