SHELL := /bin/bash
.DEFAULT_GOAL := help
MAKEFLAGS += --no-builtin-rules

WEB_DIR := web
SERVER_DIR := server
SIDECAR_DIR := sidecar
GO_SQLITE_TAGS := sqlite_fts5
ENV_LOADER := set -a; [ -f ".env" ] && . ".env"; set +a
TOOLS_BIN := $(CURDIR)/.tools/bin
GOLANGCI_LINT_BIN := $(TOOLS_BIN)/golangci-lint
GOLANGCI_LINT_VERSION := v2.11.3

.PHONY: help bootstrap setup install-tools deps deps-web deps-sidecar \
	check \
	dev-web dev-server dev-sidecar web-dev server-dev sidecar-dev \
	lint lint-web lint-server lint-sidecar \
	format format-web format-server format-sidecar \
	test test-web test-server test-sidecar \
	build build-web build-server

##@ 通用

help: ## 显示可用命令
	@awk 'BEGIN { FS = ":.*## "; printf "\nPracticeHelper make targets\n" } \
		/^##@/ { printf "\n%s\n", substr($$0, 5) } \
		/^[a-zA-Z0-9_.-]+:.*## / { printf "  %-18s %s\n", $$1, $$2 }' $(MAKEFILE_LIST)

bootstrap: ## 执行 scripts/bootstrap.sh 完成本地初始化
	./scripts/bootstrap.sh

setup: deps install-tools ## 安装前端、Python 依赖和本地工具

deps: deps-web deps-sidecar ## 安装业务依赖

deps-web: ## 安装前端依赖
	$(ENV_LOADER); pnpm --dir "$(WEB_DIR)" install

deps-sidecar: ## 同步 sidecar Python 依赖
	$(ENV_LOADER); cd "$(SIDECAR_DIR)" && uv sync

$(GOLANGCI_LINT_BIN):
	mkdir -p "$(TOOLS_BIN)"
	GOBIN="$(TOOLS_BIN)" go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)

install-tools: $(GOLANGCI_LINT_BIN) ## 安装仓库本地开发工具

check: lint test build ## 运行完整校验流水线

##@ 开发

dev-web: ## 启动前端开发服务器 (5173)
	$(ENV_LOADER); pnpm --dir "$(WEB_DIR)" dev

dev-server: ## 启动 Go API 服务 (8080)
	$(ENV_LOADER); cd "$(SERVER_DIR)" && GOCACHE=/tmp/go-build go run -tags "$(GO_SQLITE_TAGS)" ./cmd/api

dev-sidecar: ## 启动 Python sidecar (8000)
	$(ENV_LOADER); cd "$(SIDECAR_DIR)" && uv run uvicorn app.main:app --reload --host 127.0.0.1 --port 8000

web-dev: dev-web

server-dev: dev-server

sidecar-dev: dev-sidecar

##@ 质量

lint: lint-web lint-server lint-sidecar ## 运行三端 lint 检查

lint-web: ## 运行前端类型检查
	$(ENV_LOADER); pnpm --dir "$(WEB_DIR)" lint

lint-server: install-tools ## 运行 Go lint
	$(ENV_LOADER); cd "$(SERVER_DIR)" && GOLANGCI_LINT_CACHE=/tmp/golangci-lint "$(GOLANGCI_LINT_BIN)" run ./...

lint-sidecar: ## 运行 Python lint
	$(ENV_LOADER); cd "$(SIDECAR_DIR)" && uv run ruff check .

format: format-web format-server format-sidecar ## 格式化三端代码

format-web: ## 格式化前端代码
	$(ENV_LOADER); pnpm --dir "$(WEB_DIR)" format

format-server: ## 格式化 Go 代码
	$(ENV_LOADER); cd "$(SERVER_DIR)" && gofmt -w $$(find . -name '*.go')

format-sidecar: ## 格式化 Python 代码
	$(ENV_LOADER); cd "$(SIDECAR_DIR)" && uv run ruff format .

test: test-web test-server test-sidecar ## 运行三端测试

test-web: ## 运行前端测试
	$(ENV_LOADER); pnpm --dir "$(WEB_DIR)" test -- --run

test-server: ## 运行 Go 测试
	$(ENV_LOADER); cd "$(SERVER_DIR)" && GOCACHE=/tmp/go-build go test -tags "$(GO_SQLITE_TAGS)" ./...

test-sidecar: ## 运行 Python 测试
	$(ENV_LOADER); cd "$(SIDECAR_DIR)" && uv run pytest

build: build-web build-server ## 构建前端与 Go 后端

build-web: ## 构建前端产物
	$(ENV_LOADER); pnpm --dir "$(WEB_DIR)" build

build-server: ## 编译 Go 后端
	$(ENV_LOADER); cd "$(SERVER_DIR)" && GOCACHE=/tmp/go-build go build -tags "$(GO_SQLITE_TAGS)" ./...
