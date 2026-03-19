SHELL := /bin/bash

WEB_DIR := web
SERVER_DIR := server
SIDECAR_DIR := sidecar
GO_SQLITE_TAGS := sqlite_fts5
ENV_LOADER := set -a; [ -f ".env" ] && . ".env"; set +a
TOOLS_BIN := $(CURDIR)/.tools/bin
GOLANGCI_LINT_BIN := $(TOOLS_BIN)/golangci-lint
GOLANGCI_LINT_VERSION := v2.11.3

.PHONY: web-dev server-dev sidecar-dev install-tools lint format test build

$(GOLANGCI_LINT_BIN):
	mkdir -p "$(TOOLS_BIN)"
	GOBIN="$(TOOLS_BIN)" go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)

install-tools: $(GOLANGCI_LINT_BIN)

web-dev:
	$(ENV_LOADER); pnpm --dir "$(WEB_DIR)" dev

server-dev:
	$(ENV_LOADER); cd "$(SERVER_DIR)" && GOCACHE=/tmp/go-build go run -tags "$(GO_SQLITE_TAGS)" ./cmd/api

sidecar-dev:
	$(ENV_LOADER); cd "$(SIDECAR_DIR)" && uv run uvicorn app.main:app --reload --host 127.0.0.1 --port 8000

lint: install-tools
	$(ENV_LOADER); pnpm --dir "$(WEB_DIR)" lint
	$(ENV_LOADER); cd "$(SERVER_DIR)" && GOLANGCI_LINT_CACHE=/tmp/golangci-lint "$(GOLANGCI_LINT_BIN)" run ./...
	$(ENV_LOADER); cd "$(SIDECAR_DIR)" && uv run ruff check .

format:
	$(ENV_LOADER); pnpm --dir "$(WEB_DIR)" format
	$(ENV_LOADER); cd "$(SERVER_DIR)" && gofmt -w $$(find . -name '*.go')
	$(ENV_LOADER); cd "$(SIDECAR_DIR)" && uv run ruff format .

test:
	$(ENV_LOADER); pnpm --dir "$(WEB_DIR)" test -- --run
	$(ENV_LOADER); cd "$(SERVER_DIR)" && GOCACHE=/tmp/go-build go test -tags "$(GO_SQLITE_TAGS)" ./...
	$(ENV_LOADER); cd "$(SIDECAR_DIR)" && uv run pytest

build:
	$(ENV_LOADER); pnpm --dir "$(WEB_DIR)" build
	$(ENV_LOADER); cd "$(SERVER_DIR)" && GOCACHE=/tmp/go-build go build -tags "$(GO_SQLITE_TAGS)" ./...
