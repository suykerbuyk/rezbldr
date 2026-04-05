# Copyright (c) 2026 John Suykerbuyk and SykeTech LTD
# SPDX-License-Identifier: MIT OR Apache-2.0

BINARY  ?= rezbldr
PREFIX  ?= $(HOME)/.local
GOFLAGS ?=

.DEFAULT_GOAL := help

##@ General
.PHONY: help
help: ## Show this help
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) }' $(MAKEFILE_LIST)
	@echo ""
	@echo "Quick start:  make build && make test"

##@ Build
.PHONY: build
build: ## Build the rezbldr binary
	@if [ -z "$$(ls cmd/rezbldr/*.go 2>/dev/null)" ]; then \
		echo "cmd/rezbldr/ has no Go files yet (Phase 3). Compiling packages only."; \
		go build $(GOFLAGS) ./...; \
	else \
		go build $(GOFLAGS) -o $(BINARY) ./cmd/rezbldr; \
	fi

##@ Test
.PHONY: test
test: ## Run unit tests
	go test $(GOFLAGS) ./...

.PHONY: integration
integration: ## Run integration tests (build tag)
	go test $(GOFLAGS) -tags=integration ./...

.PHONY: cover
cover: ## Run tests with coverage report
	go test $(GOFLAGS) -cover ./...

.PHONY: vet
vet: ## Run go vet
	go vet ./...

##@ Install
.PHONY: install
install: build ## Build and install to PREFIX
	install -d $(PREFIX)/bin
	install -m 755 $(BINARY) $(PREFIX)/bin/

##@ Clean
.PHONY: clean
clean: ## Remove build artifacts
	go clean
	rm -f $(BINARY)
