# GoContact Studio — Master Monorepo Makefile
# Copyright (C) 2026 Tolegen Manassov (trollyHUB). See NOTICE.

.PHONY: all build test test-race test-mcp check-docs verify clean run vet fmt help

# Variables
APP_NAME := addressbook
CMD_PKG := ./cmd/addressbook
BUILD_FLAGS := -ldflags="-s -w"

all: test check-docs build

## build: Build the desktop application executable (Windows PE safe)
build:
	@echo "==> Building $(APP_NAME)..."
	go build $(BUILD_FLAGS) -o $(APP_NAME).exe $(CMD_PKG)

## test: Run all package tests including MCP tools
test:
	@echo "==> Running full unit and package tests..."
	go test -v -count=1 ./...

## test-race: Run core tests with Go data race detector
test-race:
	@echo "==> Running core tests with race detector..."
	go test -v -race ./assets/... ./internal/config/... ./internal/domain/... ./internal/infrastructure/... ./internal/repository/... ./internal/service/... ./tools/...

## test-mcp: Run MCP Server tests specifically
test-mcp:
	@echo "==> Running MCP Server tests..."
	go test -v -count=1 ./tools/gocontact-mcp/...

## check-docs: Validate Markdown links and Mermaid diagrams across all 80 roadmap volumes
check-docs:
	@echo "==> Validating roadmap specs and ADRs..."
	powershell -ExecutionPolicy Bypass -File ./.agents/scripts/check-docs.ps1

## verify: Run full cross-repo and clean architecture verification
verify:
	@echo "==> Running full system verification..."
	powershell -ExecutionPolicy Bypass -File ./.agents/scripts/verify-all-repos.ps1

## sync-all: Safely sync changes to Community Edition and Specs Vault
sync-all:
	@echo "==> Syncing all tiers..."
	powershell -ExecutionPolicy Bypass -File ./.agents/scripts/sync-all.ps1

## run: Run the desktop application locally
run:
	go run $(CMD_PKG)

## vet: Run go vet on all packages
vet:
	@echo "==> Running go vet..."
	go vet ./...

## fmt: Format all Go source files
fmt:
	@echo "==> Formatting code..."
	gofmt -s -w .

## clean: Remove build artifacts and temporary files
clean:
	@echo "==> Cleaning..."
	rm -f $(APP_NAME) $(APP_NAME).exe
	go clean

## help: Display this help message
help:
	@echo "Available commands:"
	@sed -n 's/^##//p' $(MAKEFILE_LIST) | column -t -s ':'
