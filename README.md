# PracticeHelper

PracticeHelper 是一个面向后端 / AI Agent 方向求职者的面试训练工具。

它不是题库，而是一个能根据你的真实项目和薄弱环节持续追问、打分、复盘的训练 Agent。

## 它能做什么

1. **基础知识训练** —— 围绕 Go、Redis、Kafka、网络等后端核心主题，由 Agent 出题、评分、追问，模拟真实面试官的连续追问节奏。
2. **项目面试训练** —— 导入你的 GitHub 仓库，Agent 自动分析代码后，围绕技术选型、架构 trade-off、难点和个人贡献进行深挖训练。
3. **训练复盘** —— 每轮训练结束后生成复盘卡，指出亮点、漏洞和下一步建议。
4. **薄弱点记忆** —— 系统持续记录你答得差的主题和表达卡壳点，并在后续训练中优先针对这些弱项出题。

## 目录结构

```
practicehelper/
  web/                  # Vue 3 前端
  server/               # Go API 服务
  sidecar/              # Python AI sidecar
  docs/                 # 产品文档（PRD / ARCHITECTURE / PLAN）
  data/                 # SQLite 数据库（本地生成，不入库）
  scripts/              # 开发脚本
  styles/               # 设计资源
  tests/                # 跨模块测试
  .tools/               # 本地开发工具（golangci-lint 等）
  Makefile              # 统一开发命令入口
```

## 技术栈

| 层 | 选型 |
|---|------|
| 前端 | Vue 3 + Vite + TypeScript + pnpm + Vue Router + @tanstack/vue-query + Tailwind CSS |
| Go 服务 | Gin + SQLite + FTS5 + golangci-lint |
| Python sidecar | FastAPI + LangGraph + Pydantic + Ruff |
| 前端风格 | neo-brutalist（粗边框、硬阴影、无圆角、高对比） |

## 环境搭建

### 前置依赖

- Node.js + pnpm
- Go 1.21+
- Python 3.13+ + uv

### 一键初始化

```bash
./scripts/bootstrap.sh
```

该脚本会依次执行：复制 `.env.example` 为 `.env`、安装前端依赖、同步 Python 虚拟环境、运行 Go 测试、运行 Python 测试、构建前端。

### 手动初始化

```bash
cp .env.example .env
pnpm install
cd sidecar && uv sync && cd ..
cd server && GOCACHE=/tmp/go-build go test -tags sqlite_fts5 ./... && cd ..
```

### 配置 LLM

sidecar 的核心链路（项目导入、出题、评估、复盘）依赖外部 LLM。需要在 `.env` 中配置：

```
PRACTICEHELPER_SIDECAR_MODEL=你的模型名
PRACTICEHELPER_SIDECAR_OPENAI_BASE_URL=http://127.0.0.1:3000/v1
PRACTICEHELPER_SIDECAR_OPENAI_API_KEY=你的密钥
```

不配置时，所有 sidecar 接口会直接返回 503 错误。

### 日志与排障

当前 Go API 和 sidecar 都会同时输出结构化日志到控制台和本地文件，默认路径如下：

```bash
PRACTICEHELPER_SERVER_LOG_PATH=../data/logs/server.log
PRACTICEHELPER_SIDECAR_LOG_PATH=../data/logs/sidecar.log
```

前后端之间会透传 `X-Request-ID`，便于串联一次训练请求在 API 和 sidecar 两侧的日志。

## 开发命令

先执行下面命令可以查看所有分组后的入口：

```bash
make help
```

| 命令 | 作用 |
|------|------|
| `make bootstrap` | 执行 `scripts/bootstrap.sh` 完成本地初始化 |
| `make setup` | 安装前端、Python 依赖和本地工具 |
| `make dev-web` | 启动前端开发服务器（端口 5173） |
| `make dev-server` | 启动 Go API 服务（端口 8090） |
| `make dev-sidecar` | 启动 Python sidecar（端口 8000） |
| `make lint` | 运行三端 lint 检查 |
| `make format` | 运行三端格式化 |
| `make test` | 运行三端测试 |
| `make build` | 构建前端产物和编译 Go 后端 |
| `make check` | 顺序执行 `lint`、`test`、`build` 完整校验 |

兼容旧习惯时，`make web-dev` / `make server-dev` / `make sidecar-dev` 仍然可用，但推荐统一切到 `dev-*` 这一组命名。

手动运行 Go 服务时需要带 FTS5 编译标签：

```bash
cd server && GOCACHE=/tmp/go-build go run -tags sqlite_fts5 ./cmd/api
```

## 当前进度

- [x] 产品文档与技术栈决策
- [x] 三端骨架搭建（Vue + Gin + FastAPI 均可本地启动）
- [x] 统一开发脚本与 lint / format / test
- [x] 用户画像闭环（表单 + CRUD + dashboard 聚合）
- [x] GitHub 项目导入闭环（克隆 + 分析 + 画像编辑 + FTS5 索引）
- [x] 基础知识训练闭环（出题 + 评分 + 追问 + 复盘）
- [x] 项目训练闭环（基于项目上下文的出题 + 追问 + 复盘）
- [x] weakness memory 与 dashboard 推荐
- [x] 前端 6 个页面完整交互（HomeView / ProfileView / ProjectsView / TrainView / SessionView / ReviewView）
- [x] 请求级结构化日志、日志落盘、训练恢复入口、日期倒计时修复
- [x] 训练创建 / 回答提交阶段的可见等待态
- [x] 训练创建 / 回答提交的流式输出与推理摘要展示
- [ ] 端到端验证（配置真实 LLM 后跑通完整流程）
- [ ] 错误处理与健壮性
- [ ] 训练质量调优（prompt 优化、种子题目扩充）
