# PracticeHelper

PracticeHelper 是一个面向后端 / AI Agent 方向求职者的面试训练系统，围绕用户画像、项目理解和弱点记忆，提供个性化的训练与成长路径。

它不是题库，而是一个能根据你的真实项目、目标岗位和历史弱项持续追问、打分、复盘的训练 Agent。产品方向与未来演化见 [docs/VISION.md](docs/VISION.md)，当前主线见 [docs/PLAN.md](docs/PLAN.md) 和 [docs/PRODUCT_UPGRADE_PLAN.md](docs/PRODUCT_UPGRADE_PLAN.md)，阶段 B 的已完成设计记录见 [docs/JD_TRAINING_STAGE_B.md](docs/JD_TRAINING_STAGE_B.md)。

## 它能做什么

1. **基础知识训练** —— 当前已支持 Go、Redis、Kafka、MySQL、系统设计、分布式、网络、微服务、操作系统、Docker & K8s 共 10 个显式主题，并提供按 weakness 自动跨 topic 选题的 `mixed` 入口；训练支持 `2-5` 轮连续问答和 `intensity=auto`。
2. **项目面试训练** —— 导入 GitHub 仓库后，系统会在后台异步分析代码并持续更新导入进度；导航区会显示全局导入通知，失败任务可以直接重试，完成后再围绕技术选型、架构 trade-off、难点和个人贡献进行深挖训练。
3. **岗位 JD 训练** —— 支持多份 JD 管理、分析历史、默认 JD、训练前显式绑定；JD 变成 `stale` 后会阻止新训练绑定，但旧成功快照仍可回看。
4. **训练复盘与推荐** —— 每轮训练结束后生成复盘卡，给出 `top_fix`、亮点、漏洞和推荐下一轮训练参数。
5. **弱项记忆与历史回看** —— 系统持续记录薄弱点，按有效热度生成 `today_focus` / `recommended_track`，并提供历史页、弱项趋势图和首页待复习入口。

## 目录结构

```
practicehelper/
  web/                  # Vue 3 前端
  server/               # Go API 服务
  sidecar/              # Python AI sidecar
  docs/                 # 产品文档（VISION / ROADMAP / PRD / PLAN / ARCHITECTURE）
  data/                 # SQLite 数据库、seed 与本地日志（本地生成，不入库）
  scripts/              # 开发脚本
  styles/               # 设计资源
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

sidecar 的核心链路（项目导入、岗位分析、出题、评估、复盘）依赖外部 LLM。需要在 `.env` 中配置：

```
PRACTICEHELPER_SIDECAR_MODEL=你的模型名
PRACTICEHELPER_SIDECAR_OPENAI_BASE_URL=http://127.0.0.1:3000/v1
PRACTICEHELPER_SIDECAR_OPENAI_API_KEY=你的密钥
PRACTICEHELPER_SIDECAR_LLM_TIMEOUT_SECONDS=45
PRACTICEHELPER_SERVER_SIDECAR_TIMEOUT_SECONDS=90
PRACTICEHELPER_SIDECAR_GITHUB_TOKEN=
```

不配置时，所有 sidecar 接口会直接返回 503 错误。
如果你接的是响应偏慢的真实 LLM，或者要导入稍大的 GitHub 仓库，建议把
`PRACTICEHELPER_SERVER_SIDECAR_TIMEOUT_SECONDS` 保持在 `90` 以上。虽然仓库导入已经
改成后台任务，不会再阻塞住前端请求，但后台 worker 仍然需要足够长的 sidecar 超时预算。
如果你要分析私有仓库，或者想减少 GitHub 拉取时的匿名限流，可以额外配置
`PRACTICEHELPER_SIDECAR_GITHUB_TOKEN`。

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
| `make dev-server-hot` | 用 `air` 启动 Go API 热重载（端口 8090） |
| `make dev-sidecar` | 启动 Python sidecar（端口 8000） |
| `make lint` | 运行三端 lint 检查 |
| `make format` | 运行三端格式化 |
| `make test` | 运行三端测试 |
| `make coverage` | 运行三端覆盖率统计并执行覆盖率门禁 |
| `make test-gate` | 顺序执行 `lint`、`test`、`coverage` 的测试门禁 |
| `make build` | 构建前端产物和编译 Go 后端 |
| `make check` | 顺序执行 `lint`、`test`、`build` 完整校验 |
| `make e2e-live` | 用内置可回放样例，对当前运行中的 API 跑一轮真实端到端 smoke |

兼容旧习惯时，`make web-dev` / `make server-dev` / `make sidecar-dev` 仍然可用，但推荐统一切到 `dev-*` 这一组命名。

Go 服务如果需要边改边自动重启，可以直接执行：

```bash
make dev-server-hot
```

第一次执行会自动把 `air` 安装到仓库本地的 `.tools/bin/air`，后面保存 `server/` 下的 `.go` 文件时会自动重新编译并重启服务。

### 覆盖率门禁

日常快速回归继续用 `make test`。如果你要确认这次改动是否达到当前测试门禁，用：

```bash
make test-gate
```

它会依次跑：

- `make lint`
- `make test`
- `make coverage`

其中 `make coverage` 会分别调用：

- `pnpm --dir web coverage`
- `cd server && GOCACHE=/tmp/go-build go test -tags sqlite_fts5 -covermode=count -coverpkg=./internal/... -coverprofile=coverage.out ./...`
- `cd sidecar && uv run pytest --cov=app ...`

真实 LLM 的 `make e2e-live` 继续保留为手动 smoke，不纳入默认阻塞门禁。

### 真实 E2E 回放

`make e2e-live` 默认会读取 `scripts/e2e_live.sample.json`，按固定画像、固定两轮回答跑完整主线：

- 保存画像
- 创建、分析并激活一份 JD
- 后台导入项目
- 项目画像编辑并保存后再恢复原值
- basics 训练一轮
- project 训练一轮
- 拉取两张 review
- 校验 session / review 是否固定绑定 JD 分析快照
- 校验 dashboard 的 top weakness、`today_focus`、`recommended_track` 是否绑定到同一条弱项
- 校验 JD 变成 `stale` 后，训练绑定会被阻止，dashboard 推荐会回退到 generic
- 校验回答提交流式状态序列是否覆盖 `answer_received` → `answer_saved` → `review_saved`

如果你只想替换仓库地址，不需要改样例文件，直接传参即可：

```bash
python3 ./scripts/e2e_live.py --repo-url https://github.com/yourname/your-repo --output /tmp/practicehelper-e2e-live-summary.json
```

如果你要回放自己的固定话术，可以复制 `scripts/e2e_live.sample.json` 改成自己的场景，再执行：

```bash
python3 ./scripts/e2e_live.py --scenario ./scripts/e2e_live.sample.json --output /tmp/practicehelper-e2e-live-summary.json
```

手动运行 Go 服务时需要带 FTS5 编译标签：

```bash
cd server && GOCACHE=/tmp/go-build go run -tags sqlite_fts5 ./cmd/api
```

如果你想手动跑热重载，也可以这样执行：

```bash
cd server && ../.tools/bin/air -c .air.toml
```

## 当前进度

- [x] 产品文档与技术栈决策
- [x] 三端骨架搭建（Vue + Gin + FastAPI 均可本地启动）
- [x] 统一开发脚本与 lint / format / test
- [x] 用户画像闭环（表单 + CRUD + dashboard 聚合）
- [x] GitHub 项目导入闭环（后台任务 + 进度轮询 + 克隆分析 + 画像编辑 + FTS5 索引）
- [x] 基础知识训练闭环（出题 + 评分 + 追问 + 复盘）
- [x] 项目训练闭环（基于项目上下文的出题 + 追问 + 复盘）
- [x] weakness memory 与 dashboard 推荐
- [x] 前端 8 个主流程页面 + 1 个 Prompt 实验页（HomeView / ProfileView / JobTargetsView / ProjectsView / PromptExperimentsView / TrainView / SessionView / ReviewView / HistoryView）
- [x] 请求级结构化日志、日志落盘、训练恢复入口、日期倒计时修复
- [x] 训练创建 / 回答提交阶段的可见等待态
- [x] 训练创建 / 回答提交的流式输出与推理摘要展示
- [x] Prompt 版本选择、Prompt 实验对比页与 Review 审计面板
- [x] 单次 Session 的 Markdown / JSON / PDF 导出，以及 History 跨页批量 ZIP 导出
- [x] 可回放的真实端到端 smoke 脚本（`scripts/e2e_live.py` / `make e2e-live` / `scripts/e2e_live.sample.json`）
- [x] 端到端验证（配置真实 LLM 后可用 `make e2e-live` 跑通完整流程）
- [x] 训练体验做稳（答题反馈 V2、推荐质量与弱项衰减、题库扩充、追问保守表达）
- [x] 阶段 B：岗位视角训练收口（独立 JD 页面、多 JD 管理、分析历史、训练前选择 JD、JD 绑定验证）
- [x] 会话历史页、弱项趋势图与首页待复习入口
- [x] 多轮训练（`max_turns=2-5`）、复盘推荐下一轮、`review_pending -> retry-review` 恢复入口

## 当前主线

当前主线已经从“岗位视角接入”切到“训练深度与留存升级”：

- 阶段 A / B 和阶段 C 的首轮闭环已经落地，当前不再把多轮训练、弱项级待复习入口、评估审计面板这些已实现能力写成“待补缺口”
- 如果继续推进，更适合聚焦还没做完的增强项：
  - Prompt 版本管理仍停在 v1：已有版本选择、A/B 对比和审计明细，但还没有在线编辑、更细粒度 flow 级切换和更强实验分析
  - 项目训练当前仍是 SQLite FTS5 检索，RAG 升级还是独立未开始项
  - LangGraph 继续保持薄壳，不为了“更像 agent”继续堆复杂图

详细方案见 [docs/PLAN.md](docs/PLAN.md) 和 [docs/PRODUCT_UPGRADE_PLAN.md](docs/PRODUCT_UPGRADE_PLAN.md)。
