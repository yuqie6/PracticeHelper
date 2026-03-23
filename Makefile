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
AIR_BIN := $(TOOLS_BIN)/air
SERVER_COVERAGE_MIN := 50
SIDECAR_COVERAGE_MIN := 70

.PHONY: help bootstrap setup install-tools deps deps-web deps-sidecar pytest \
	clean-sidecar-coverage \
	check \
	dev-web dev-server dev-server-hot dev-sidecar dev-qdrant web-dev server-dev server-dev-hot sidecar-dev \
	lint lint-web lint-server lint-sidecar \
	format format-web format-server format-sidecar \
	test test-web test-server test-sidecar \
	coverage coverage-web coverage-server coverage-sidecar test-gate \
	build build-web build-server \
	e2e-live

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
	$(ENV_LOADER); uv sync --project "$(SIDECAR_DIR)"

pytest: ## 从仓库根运行 sidecar pytest（可用 PYTEST_ARGS 透传参数）
	$(ENV_LOADER); uv run --project "$(SIDECAR_DIR)" pytest $(PYTEST_ARGS)

clean-sidecar-coverage: ## 清理 sidecar 覆盖率产物（.coverage、coverage.xml、htmlcov、*.cover）
	find "$(SIDECAR_DIR)" \
		\( -path "$(SIDECAR_DIR)/.venv" -o -path "$(SIDECAR_DIR)/.pytest_cache" -o -path "$(SIDECAR_DIR)/.ruff_cache" \) -prune \
		-o \( -name '.coverage' -o -name '.coverage.*' -o -name 'coverage.xml' -o -name 'htmlcov' -o -name '*.cover' \) \
		-print0 | xargs -0r rm -rf

$(GOLANGCI_LINT_BIN):
	mkdir -p "$(TOOLS_BIN)"
	GOBIN="$(TOOLS_BIN)" go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)

$(AIR_BIN):
	mkdir -p "$(TOOLS_BIN)"
	GOBIN="$(TOOLS_BIN)" go install github.com/air-verse/air@latest

install-tools: $(GOLANGCI_LINT_BIN) $(AIR_BIN) ## 安装仓库本地开发工具

check: lint test build ## 运行完整校验流水线

##@ 开发

dev-web: ## 启动前端开发服务器 (5173)
	$(ENV_LOADER); pnpm --dir "$(WEB_DIR)" dev

dev-server: ## 启动 Go API 服务 (8090)
	$(ENV_LOADER); cd "$(SERVER_DIR)" && GOCACHE=/tmp/go-build go run -tags "$(GO_SQLITE_TAGS)" ./cmd/api

dev-server-hot: $(AIR_BIN) ## 启动 Go API 服务热重载 (8090)
	$(ENV_LOADER); cd "$(SERVER_DIR)" && "$(AIR_BIN)" -c .air.toml

dev-sidecar: ## 启动 Python sidecar (8000)
	$(ENV_LOADER); cd "$(SIDECAR_DIR)" && uv run uvicorn app.main:app --reload --host 127.0.0.1 --port 8000

dev-qdrant: ## 启动本地 Qdrant (6333)
	./scripts/dev_qdrant.sh

web-dev: dev-web

server-dev: dev-server

server-dev-hot: dev-server-hot

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
	$(ENV_LOADER); uv run --project "$(SIDECAR_DIR)" pytest

coverage: coverage-web coverage-server coverage-sidecar ## 运行三端覆盖率统计

coverage-web: ## 运行前端覆盖率统计
	$(ENV_LOADER); pnpm --dir "$(WEB_DIR)" coverage

coverage-server: ## 运行 Go 覆盖率统计
	$(ENV_LOADER); cd "$(SERVER_DIR)" && GOCACHE=/tmp/go-build go test -tags "$(GO_SQLITE_TAGS)" -covermode=count -coverpkg=./internal/... -coverprofile=coverage.out ./...
	$(ENV_LOADER); cd "$(SERVER_DIR)" && go tool cover -func=coverage.out | tee coverage.txt
	@awk '/^total:/ { gsub("%","",$$3); if ($$3 + 0 < $(SERVER_COVERAGE_MIN)) { printf("server coverage %.1f%% below %s%%\n", $$3 + 0, "$(SERVER_COVERAGE_MIN)"); exit 1 } }' "$(SERVER_DIR)/coverage.txt"

coverage-sidecar: clean-sidecar-coverage ## 运行 Python 覆盖率统计
	$(ENV_LOADER); cd "$(SIDECAR_DIR)" && uv run pytest --cov=app --cov-report=term-missing --cov-report=xml:coverage.xml --cov-report=html:htmlcov --cov-fail-under=$(SIDECAR_COVERAGE_MIN)

test-gate: lint test coverage ## 运行带覆盖率门禁的完整测试校验

build: build-web build-server ## 构建前端与 Go 后端

build-web: ## 构建前端产物
	$(ENV_LOADER); pnpm --dir "$(WEB_DIR)" build

build-server: ## 编译 Go 后端
	$(ENV_LOADER); cd "$(SERVER_DIR)" && GOCACHE=/tmp/go-build go build -tags "$(GO_SQLITE_TAGS)" ./...

e2e-live: ## 用真实运行中的 API 跑一轮端到端 smoke（需先启动 server/sidecar）
	$(ENV_LOADER); python3 ./scripts/e2e_live.py
