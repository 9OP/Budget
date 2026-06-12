-include .env

BINARY := budget

SHELL := /bin/bash

# Colors
RESET  := \033[0m
BOLD   := \033[1m
GREEN  := \033[0;32m
YELLOW := \033[0;33m
BLUE   := \033[0;34m
RED    := \033[0;31m
GREY   := \033[0;90m

define step
	@printf "%b\n" "$(BOLD)$(BLUE)[$(1)]$(RESET) $(2)"
endef

define ok
	@printf "%b\n" "$(BOLD)$(GREEN)[Success]$(RESET) $(1)"
endef

define warn
	@printf "%b\n" "$(BOLD)$(YELLOW)[Warning]$(RESET) $(1)"
endef

define fail
	@printf "%b\n" "$(BOLD)$(RED)[Error]$(RESET) $(1)"
endef

.PHONY: help all build test lint fmt clean \
        _check-go _mod-tidy _vet _golangci-lint

help: ## Show available commands
	@printf "\n%b\n\n" "$(BOLD)Budget$(RESET) — available commands:"
	@awk 'BEGIN {FS=":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-20s %s\n", $$1, $$2}' $(MAKEFILE_LIST)
	@printf "\n"

all: clean fmt lint test build ## Run everything

build: _check-go ## Build binary
	$(call step,Build,Compiling project...)
	@go build -o $(BINARY) ./cmd/...
	$(call ok,Binary built: $(BINARY))

test: _check-go ## Run tests
	$(call step,Test,Running tests...)
	@go test ./...
	$(call ok,All tests passed)

lint: _check-go _mod-tidy _vet _golangci-lint ## Run linters

fmt: _check-go ## Format code
	$(call step,Format,Formatting source...)
	@gofmt -w .
	$(call ok,Code formatted)

clean: ## Remove artifacts
	$(call step,Clean,Removing build artifacts...)
	@go clean
	@rm -f $(BINARY)
	$(call ok,Cleaned)

_check-go:
	@command -v go >/dev/null 2>&1 || { \
		printf "%b\n" "$(BOLD)$(RED)[Error]$(RESET) Go is not installed"; \
		exit 1; \
	}

_mod-tidy:
	$(call step,Lint,Tidying modules...)
	@go mod tidy
	$(call ok,Modules are up to date)

_vet:
	$(call step,Lint,Running go vet...)
	@go vet ./...
	$(call ok,go vet passed)

_golangci-lint:
	@if command -v golangci-lint >/dev/null 2>&1; then \
		printf "%b\n" "$(BOLD)$(BLUE)[Lint]$(RESET) Running golangci-lint..."; \
		golangci-lint run ./...; \
	else \
		printf "%b\n" "$(BOLD)$(YELLOW)[Warning]$(RESET) golangci-lint not installed, skipping"; \
	fi
