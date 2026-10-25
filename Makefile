# GoContact Community Edition — Standard Build & Test Automation
# Copyright (C) 2026 Tolegen Manassov (trollyHUB). Licensed under AGPL-3.0.

.PHONY: all build test test-race lint clean run help

# Variables
APP_NAME := addressbook
CMD_PKG := ./cmd/addressbook
BUILD_FLAGS := -ldflags="-s -w"

all: test build

## build: Build the desktop application executable
build:
	@echo "==> Building $(APP_NAME)..."
	go build $(BUILD_FLAGS) -o $(APP_NAME) $(CMD_PKG)

## test: Run all package tests
test:
	@echo "==> Running tests..."
	go test -v -count=1 ./assets/... ./cmd/... ./internal/...

## test-race: Run core tests with Go data race detector
test-race:
	@echo "==> Running tests with race detector..."
	go test -v -race ./assets/... ./internal/config/... ./internal/domain/... ./internal/infrastructure/... ./internal/repository/... ./internal/service/...

## run: Run the application
run:
	go run $(CMD_PKG)

## clean: Remove build artifacts and temporary files
clean:
	@echo "==> Cleaning..."
	rm -f $(APP_NAME) $(APP_NAME).exe
	go clean

## help: Display this help message
help:
	@echo "Available commands:"
	@sed -n 's/^##//p' $(MAKEFILE_LIST) | column -t -s ':'
