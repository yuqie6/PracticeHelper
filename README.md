<p align="center">
  <strong>PracticeHelper</strong>
</p>

<p align="center">
  <em>面向后端 / AI 工程师的 AI 面试训练系统 —— 诊断弱项、围绕真实项目深挖、每一轮训练都在成长。</em>
</p>

<p align="center">
  <a href="#核心功能">核心功能</a> •
  <a href="#系统架构">系统架构</a> •
  <a href="#快速开始">快速开始</a> •
  <a href="#开发指南">开发指南</a> •
  <a href="#文档索引">文档索引</a>
</p>

---

## 这是什么？

PracticeHelper 是一个自托管的面试训练系统，专为后端和 AI 方向的求职者设计。它不是题库，而是一个围绕你的**真实项目、目标岗位和历史弱项**持续追问、打分、复盘的训练 Agent。

系统的核心循环：

```
┌─────────────┐     ┌──────────────┐     ┌───────────────┐
│  诊断弱项   │────▶│  深度训练    │────▶│  记忆进展     │
└─────────────┘     └──────────────┘     └───────────────┘
       ▲                                          │
       │            ┌──────────────┐              │
       └────────────│  推荐下一步  │◀─────────────┘
                    └──────────────┘
```

- **诊断** —— 通过多轮问答发现你的真实短板
- **训练** —— 基于你的 GitHub 项目和目标岗位 JD 出题
- **记忆** —— 跨会话追踪弱项，严重度随时间衰减
- **推荐** —— 根据当前弱项排序，告诉你接下来该练什么

> **定位**：单用户、本地自托管工具，不是 SaaS 平台。你的数据和 LLM 完全由自己掌控。

## 核心功能

### 🎯 基础知识训练

内置 10 个主题（Go、Redis、Kafka、MySQL、系统设计、分布式、网络、微服务、操作系统、Docker & K8s），以及一个按弱项自动跨主题选题的 **mixed** 模式。每轮训练支持 2–5 轮可配置的连续问答。

### 🔬 项目面试训练

导入任意 GitHub 仓库，系统会异步克隆、分析代码，生成结构化项目画像（摘要、亮点、难点、Trade-off、可追问点）。训练时围绕你**真实的技术选型、架构决策和个人贡献**深挖。

### 📋 岗位 JD 对齐

支持管理多份 JD 文本，每份带版本化的分析历史。训练前绑定特定 JD，出题、评分、追问、复盘都会参考该岗位的能力要求。JD 原文修改后变为 `stale`，阻止新训练绑定，但旧快照仍可回看。

### 📊 训练复盘

每轮训练结束后生成复盘卡：总评、优先修正项（`top_fix`）、亮点、漏洞、各维度评分明细、建议下一轮训练参数。一键开始推荐的下一轮训练。

### 🧠 弱项记忆

弱项以标签形式持续追踪，带严重度（severity）和频率（frequency）。答差了上升，答好了下降，长期未出现会随时间衰减（有效严重度 = `severity / (1 + 闲置天数 / 21)`）。首页展示 `today_focus`、`recommended_track` 和弱项趋势 sparkline。

### 🔄 间隔复习

基于简化版 SM-2 算法，每轮训练结束后生成待复习条目。到期的复习项会出现在首页 Dashboard，直达对应主题的训练入口。

### 📡 流式输出与推理透明

训练创建和回答提交全程流式推送阶段进展；除了 `phase / context / reasoning / content / result` 这条主链路外，还会把可公开的 `trace` 和 Go 侧 `persist` 结果一起暴露出来，前端不再只剩一个空白 loading。

### 📦 导出

单次训练可导出为 Markdown / JSON / PDF，历史页可批量选择导出为 ZIP 压缩包。

## 系统架构

PracticeHelper 由 **三个本地进程 + 一个可选向量服务** 通过 HTTP 通信组成：

```
浏览器 ──HTTP──▶ Go API (:8090) ──HTTP──▶ Python Sidecar (:8000) ──HTTP──▶ LLM Provider
                    │
          ┌─────────┴─────────┐
          ▼                   ▼
  SQLite (data/practicehelper.db)   Qdrant (:6333, 可选)
```

| 层 | 职责 | 技术选型 |
|:---|:-----|:---------|
| **前端** | 9 个页面，neo-brutalist 风格 | Vue 3、TypeScript、Vite、Tailwind CSS、TanStack Query、Vue Router |
| **API 服务** | 路由、状态机、持久化、sidecar 编排 | Go (Gin)、SQLite + FTS5 |
| **AI Sidecar** | 仓库分析、出题、评估、复盘 | Python (FastAPI)、LangGraph、Pydantic |
| **业务存储** | 训练数据、画像、历史、审计 | SQLite |
| **向量检索** | repo chunk / memory embedding 与向量召回 | Qdrant（可选，当前唯一已实现 provider，可本地或云端） |

### 关键设计决策

- **单用户单体架构** —— 先让系统对一个人真正有用，验证训练价值后再考虑多用户
- **SQLite 仍是系统真相源** —— 画像、项目、会话、弱项、知识图谱和审计结果都在 SQLite；Qdrant 只承载可重建的向量索引
- **LangGraph 保持薄壳** —— 图层只做最小编排，核心逻辑在 `AgentRuntime`（agent loop + single-shot 降级）
- **Go 掌控状态机** —— 会话状态流转、原子锁、复习调度、弱项更新全由服务端强制执行
- **Sidecar 掌控智能** —— 上下文装配、规划、工具调用、输出校验、结构化意图生成

完整架构文档（当前数据库表、训练状态机、Agent Runtime PEAS 模型、检索策略、API 清单）见 [`docs/current/ARCHITECTURE.md`](docs/current/ARCHITECTURE.md)。

## 快速开始

### 前置依赖

| 依赖 | 版本 |
|:-----|:-----|
| Node.js | LTS |
| pnpm | 10+ |
| Go | 1.21+ |
| Python | 3.13+ |
| uv | latest |

### 1. 克隆并初始化

```bash
git clone https://github.com/yourname/practicehelper.git
cd practicehelper
./scripts/bootstrap.sh
```

bootstrap 脚本会依次执行：复制 `.env.example` → `.env`、安装前端依赖、同步 Python 虚拟环境、运行 Go 测试、运行 Python 测试、构建前端。

### 2. 配置 LLM

编辑 `.env`，填入你的 LLM 服务信息：

```bash
PRACTICEHELPER_SIDECAR_MODEL=你的模型名
PRACTICEHELPER_SIDECAR_OPENAI_BASE_URL=http://127.0.0.1:3000/v1
PRACTICEHELPER_SIDECAR_OPENAI_API_KEY=你的密钥
```

Sidecar 使用 OpenAI 兼容 API 格式，任何暴露该接口的服务（OpenAI、Anthropic 代理、vLLM / Ollama 本地模型等）都可以直接接入。
如果本地服务没有鉴权，`PRACTICEHELPER_SIDECAR_OPENAI_API_KEY` 可以留空。

> **不配置 LLM 时，所有 AI 功能返回 503。** 系统不做启发式兜底。

<details>
<summary>可选：GitHub Token、Qdrant 与超时调优</summary>

```bash
# 减少 GitHub 匿名限流（导入私有仓库时必须配置）
PRACTICEHELPER_SIDECAR_GITHUB_TOKEN=ghp_xxx

# 启用 repo chunk / memory 向量召回
# 当前 vector store provider 只实现 qdrant；本地 Docker 和 Qdrant Cloud 共用这一组配置。
PRACTICEHELPER_SERVER_VECTOR_STORE_PROVIDER=qdrant
PRACTICEHELPER_SERVER_VECTOR_STORE_URL=http://127.0.0.1:6333
PRACTICEHELPER_SERVER_VECTOR_STORE_API_KEY=
PRACTICEHELPER_SERVER_VECTOR_WRITE_ENABLED=true
PRACTICEHELPER_SERVER_VECTOR_READ_ENABLED=true
PRACTICEHELPER_SERVER_VECTOR_RERANK_ENABLED=true

# embedding / rerank 可以指向云端 API，也可以指向本地 OpenAI-compatible 服务；
# 本地服务没有鉴权时，API key 可以留空。
PRACTICEHELPER_SIDECAR_EMBEDDING_MODEL=你的 embedding 模型
PRACTICEHELPER_SIDECAR_EMBEDDING_BASE_URL=http://127.0.0.1:3000/v1
PRACTICEHELPER_SIDECAR_EMBEDDING_API_KEY=你的密钥
PRACTICEHELPER_SIDECAR_RERANK_MODEL=你的 rerank 模型
PRACTICEHELPER_SIDECAR_RERANK_BASE_URL=http://127.0.0.1:3000
PRACTICEHELPER_SIDECAR_RERANK_API_KEY=你的密钥

# 接慢速 LLM 或导入大仓库时适当提高超时
PRACTICEHELPER_SERVER_SIDECAR_TIMEOUT_SECONDS=90
PRACTICEHELPER_SIDECAR_LLM_TIMEOUT_SECONDS=45
```

</details>

### 3. 启动服务

如果要验证 6.4 的 repo chunk 向量召回，先启动本地 Qdrant：

```bash
make dev-qdrant
```

然后打开三个终端窗口：

```bash
# 终端 1 —— 前端（端口 5173）
make dev-web

# 终端 2 —— Go API（端口 8090）
make dev-server

# 终端 3 —— Python sidecar（端口 8000）
make dev-sidecar
```

打开 **http://localhost:5173** 即可开始训练。

## 开发指南

### 常用命令

所有命令通过统一的 `Makefile` 管理，运行 `make help` 可查看完整列表。

| 命令 | 说明 |
|:-----|:-----|
| `make bootstrap` | 一键本地初始化 |
| `make setup` | 安装前端、Python 依赖和本地工具 |
| `make dev-web` | 启动前端开发服务器（5173） |
| `make dev-server` | 启动 Go API（8090） |
| `make dev-server-hot` | 启动 Go API 热重载（air） |
| `make dev-sidecar` | 启动 Python sidecar（8000） |
| `make dev-qdrant` | 启动本地 Qdrant（6333） |
| `make lint` | 三端 lint 检查 |
| `make format` | 三端代码格式化 |
| `make test` | 三端测试 |
| `make test-gate` | 完整门禁：lint → test → coverage（含最低阈值） |
| `make build` | 构建前端产物 + 编译 Go 二进制 |
| `make check` | 完整流水线：lint → test → build |
| `make e2e-live` | 端到端 smoke 测试（需先启动服务 + 配置 LLM） |

### 项目结构

```
practicehelper/
├── web/                          # Vue 3 + TypeScript 前端
│   ├── src/views/                #   页面组件
│   ├── src/components/           #   共享 UI 组件
│   └── src/lib/                  #   API 客户端、工具函数
├── server/                       # Go API 服务
│   ├── cmd/api/                  #   入口
│   └── internal/                 #   分层架构
│       ├── config/               #     环境配置
│       ├── controller/           #     HTTP handler & 中间件
│       ├── domain/               #     共享类型 & DTO
│       ├── infra/sqlite/         #     数据库连接、迁移、种子数据
│       ├── repo/                 #     数据访问层
│       ├── service/              #     业务逻辑层
│       ├── sidecar/              #     Sidecar HTTP 客户端
│       └── observability/        #     Request ID & 日志
├── sidecar/                      # Python AI Sidecar
│   ├── app/                      #   FastAPI 应用、AgentRuntime、LLM 客户端
│   │   └── prompts/              #     外置 system prompt 模板
│   └── tests/                    #   Pytest 测试
├── docs/                         # 产品文档 & 架构文档
├── scripts/                      # 初始化脚本、E2E 测试
├── data/                         # SQLite 数据库 & 日志（gitignored）
└── Makefile                      # 统一命令入口
```

### 代码风格

| 层 | 规范 |
|:---|:-----|
| 前端 | TypeScript，2 空格缩进，分号，单引号（Prettier） |
| Go | `gofmt`，导出 CamelCase / 未导出 mixedCase，`golangci-lint` |
| Python | Ruff（100 字符行宽），Pydantic 类型模型，snake_case |

### 测试

测试文件与对应层放在一起：

- `web/src/**/*.spec.ts` — Vitest + Vue Test Utils
- `server/**/*_test.go` — Go testing（需 `sqlite_fts5` build tag）
- `sidecar/tests/test_*.py` — Pytest + coverage

覆盖率门禁：**Server ≥ 50%**，**Sidecar ≥ 70%**。

### 端到端测试

项目附带可回放的端到端 smoke 脚本，覆盖完整主链路：

```bash
# 使用默认样例场景
make e2e-live

# 指定自定义仓库
python3 ./scripts/e2e_live.py --repo-url https://github.com/you/your-repo

# 使用自定义场景文件
python3 ./scripts/e2e_live.py --scenario ./scripts/e2e_live.sample.json
```

测试覆盖：画像保存 → JD 创建/分析/激活 → 项目导入 → basics + project 训练 → 复盘生成 → 弱项追踪 → stale JD 拦截 → 流式状态验证。

### 日志与排障

Go API 和 sidecar 均同时输出结构化 JSON 日志到控制台和本地文件：

```bash
PRACTICEHELPER_SERVER_LOG_PATH=../data/logs/server.log
PRACTICEHELPER_SIDECAR_LOG_PATH=../data/logs/sidecar.log
```

每个请求生成 `X-Request-ID` 并在 Go → sidecar 之间透传，方便在两端日志中串联同一次训练请求。

## 页面一览

| 路由 | 页面 | 说明 |
|:-----|:-----|:-----|
| `/` | Dashboard | 弱项 Top 5、最近训练、今日建议、待复习、倒计时 |
| `/profile` | 用户画像 | 编辑求职背景 |
| `/job-targets` | 岗位管理 | 管理 JD、查看结构化分析与历史 |
| `/projects` | 项目管理 | 导入 GitHub 仓库、查看/编辑项目画像 |
| `/train` | 训练配置 | 选择模式、主题、强度、轮次、JD |
| `/sessions/:id` | 训练过程 | 多轮问答交互，流式反馈 |
| `/reviews/:id` | 复盘 | 复盘卡、评分明细、下一步建议 |
| `/history` | 训练历史 | 分页列表，支持模式/主题/状态筛选 |
| `/prompt-experiments` | Prompt 实验 | Prompt 版本对比、评估审计元数据 |

## 演化路线

项目按阶段渐进演化，每个阶段可独立交付：

| 阶段 | 状态 | 聚焦 |
|:-----|:-----|:-----|
| **A** 训练体验做稳 | ✅ 已完成 | 反馈 UX、弱项衰减、可行动的复盘 |
| **B** 接入岗位视角 | ✅ 已完成 | JD 管理、分析历史、训练绑定 |
| **C** 训练深度与留存 | 🔄 进行中 | 多轮训练、间隔复习、Prompt 实验 |
| **D** 项目证据映射 | 📋 规划中 | 能力标签、证据片段、项目支撑的评估 |
| **E** 学习规划闭环 | 📋 规划中 | 阶段性学习建议、能力雷达图、每周重点 |

完整产品方向见 [`docs/current/VISION.md`](docs/current/VISION.md)，当前阶段任务见 [`docs/current/PLAN.md`](docs/current/PLAN.md)。

## 文档索引

如果几份文档读起来像在打架，默认这样理解：

`README / docs/current/*` 是当前事实主链；
专项计划和阶段设计文档要结合各自的“状态”字段阅读，不单独覆盖主链事实。

| 文档 | 角色 | 当前状态 | 什么时候看 |
|:-----|:-----|:---------|:-----------|
| [`docs/README.md`](docs/README.md) | 文档总索引 | 当前有效 | 想先搞清楚 `docs/` 怎么分层 |
| [`VISION.md`](docs/current/VISION.md) | 方向锚点 | 当前有效 | 想确认产品最终要长成什么 |
| [`PRD.md`](docs/current/PRD.md) | 产品边界 | 当前有效 | 想确认现在做什么、不做什么 |
| [`ROADMAP.md`](docs/current/ROADMAP.md) | 阶段顺序 | 当前有效 | 想确认为什么按这个顺序推进 |
| [`PLAN.md`](docs/current/PLAN.md) | 当前主线 | 当前有效 | 想确认这一轮优先做什么 |
| [`ARCHITECTURE.md`](docs/current/ARCHITECTURE.md) | 技术事实 | 当前有效 | 想确认系统分层、状态机、Schema、API |
| [`PRODUCT_UPGRADE_PLAN.md`](docs/plans/PRODUCT_UPGRADE_PLAN.md) | 阶段 C 产品升级清单 | 专项计划 | 想看训练深度与留存升级的拆解项 |
| [`ARCHITECTURE_CONVERGENCE_PLAN.md`](docs/plans/ARCHITECTURE_CONVERGENCE_PLAN.md) | 工程收口专项 | 专项计划 | 想看这轮拆分、卫生和文档收口怎么做 |
| [`AGENT_DEEP_REDESIGN_PLAN.md`](docs/plans/AGENT_DEEP_REDESIGN_PLAN.md) | agent runtime 深改专项 | 专项计划 | 想看 sidecar agent 能力怎么继续演进 |
| [`JD_TRAINING_STAGE_B.md`](docs/records/JD_TRAINING_STAGE_B.md) | 阶段 B 设计记录 | 已完成记录 | 想回看 JD 主线当时的约束和落地口径 |
| [`ANSWER_FEEDBACK_UX_V2.md`](docs/records/ANSWER_FEEDBACK_UX_V2.md) | 阶段 A 设计记录 | 已完成记录 | 想回看答题反馈 V2 当时为什么这么改 |

## License

本项目用于个人学习和面试准备，详见仓库 License 文件。
