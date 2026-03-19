SHELL := /bin/bash

WEB_DIR := web
SERVER_DIR := server
SIDECAR_DIR := sidecar

.PHONY: web-dev server-dev sidecar-dev lint format test build

web-dev:
	pnpm --dir $(WEB_DIR) dev

server-dev:
	cd $(SERVER_DIR) && go run ./cmd/api

sidecar-dev:
	cd $(SIDECAR_DIR) && uv run uvicorn app.main:app --reload --host 127.0.0.1 --port 8000

lint:
	pnpm --dir $(WEB_DIR) lint
	cd $(SERVER_DIR) && GOLANGCI_LINT_CACHE=/tmp/golangci-lint golangci-lint run ./...
	cd $(SIDECAR_DIR) && uv run ruff check .

format:
	pnpm --dir $(WEB_DIR) format
	cd $(SERVER_DIR) && gofmt -w $$(find . -name '*.go')
	cd $(SIDECAR_DIR) && uv run ruff format .

test:
	pnpm --dir $(WEB_DIR) test -- --run
	cd $(SERVER_DIR) && GOCACHE=/tmp/go-build go test ./...
	cd $(SIDECAR_DIR) && uv run pytest

build:
	pnpm --dir $(WEB_DIR) build
	cd $(SERVER_DIR) && GOCACHE=/tmp/go-build go build ./...
