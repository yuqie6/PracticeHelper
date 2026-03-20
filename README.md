# PracticeHelper

PracticeHelper 是一个面向后端 / AI Agent 方向求职者的面试训练系统，围绕用户画像、项目理解和弱点记忆，提供个性化的训练与成长路径。

它不是题库，而是一个能根据你的真实项目和薄弱环节持续追问、打分、复盘的训练 Agent。产品方向与未来演化见 [docs/VISION.md](docs/VISION.md)，当前主线与下一阶段方案见 [docs/PLAN.md](docs/PLAN.md) 和 [docs/JD_TRAINING_STAGE_B.md](docs/JD_TRAINING_STAGE_B.md)。

## 它能做什么

1. **基础知识训练** —— 围绕 Go、Redis、Kafka、网络等后端核心主题，由 Agent 出题、评分、追问，模拟真实面试官的连续追问节奏。
2. **项目面试训练** —— 导入你的 GitHub 仓库后，系统会在后台异步分析代码并持续更新导入进度；导航区会显示全局导入通知，失败任务也可以直接重试，完成后再围绕技术选型、架构 trade-off、难点和个人贡献进行深挖训练。
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
PRACTICEHELPER_SERVER_SIDECAR_TIMEOUT_SECONDS=90
```

不配置时，所有 sidecar 接口会直接返回 503 错误。
如果你接的是响应偏慢的真实 LLM，或者要导入稍大的 GitHub 仓库，建议把
`PRACTICEHELPER_SERVER_SIDECAR_TIMEOUT_SECONDS` 保持在 `90` 以上。虽然仓库导入已经
改成后台任务，不会再阻塞住前端请求，但后台 worker 仍然需要足够长的 sidecar 超时预算。

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
| `make build` | 构建前端产物和编译 Go 后端 |
| `make check` | 顺序执行 `lint`、`test`、`build` 完整校验 |
| `make e2e-live` | 用内置可回放样例，对当前运行中的 API 跑一轮真实端到端 smoke |

兼容旧习惯时，`make web-dev` / `make server-dev` / `make sidecar-dev` 仍然可用，但推荐统一切到 `dev-*` 这一组命名。

Go 服务如果需要边改边自动重启，可以直接执行：

```bash
make dev-server-hot
```

第一次执行会自动把 `air` 安装到仓库本地的 `.tools/bin/air`，后面保存 `server/` 下的 `.go` 文件时会自动重新编译并重启服务。

### 真实 E2E 回放

`make e2e-live` 默认会读取 `scripts/e2e_live.sample.json`，按固定画像、固定两轮回答跑完整主线：

- 保存画像
- 后台导入项目
- 项目画像编辑并保存后再恢复原值
- basics 训练一轮
- project 训练一轮
- 拉取两张 review
- 校验 dashboard 的 top weakness、`today_focus`、`recommended_track` 是否绑定到同一条弱项
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
- [x] 前端 6 个页面完整交互（HomeView / ProfileView / ProjectsView / TrainView / SessionView / ReviewView）
- [x] 请求级结构化日志、日志落盘、训练恢复入口、日期倒计时修复
- [x] 训练创建 / 回答提交阶段的可见等待态
- [x] 训练创建 / 回答提交的流式输出与推理摘要展示
- [x] 可回放的真实端到端 smoke 脚本（`scripts/e2e_live.py` / `make e2e-live` / `scripts/e2e_live.sample.json`）
- [x] 端到端验证（配置真实 LLM 后可用 `make e2e-live` 跑通完整流程）
- [x] 训练体验做稳（答题反馈 V2、推荐质量与弱项衰减、题库扩充、追问保守表达）
- [ ] 阶段 B：岗位视角训练（独立 JD 页面、多 JD 管理、分析历史、训练前选择 JD）

## 当前主线

当前主线已经从“训练体验做稳”切到“岗位视角接入”：

- 阶段 A 已完成，重点问题已经从“能不能用”转到“练的是不是目标岗位要的内容”
- 下一步不再优先继续打磨 Phase 7 小边角，而是推进阶段 B 的最小闭环：
  - 独立 JD 页面
  - 多 JD 管理与分析历史
  - 训练前手动选择 JD
  - basics / project 的出题和评分都引用所选 JD

详细方案见 [docs/JD_TRAINING_STAGE_B.md](docs/JD_TRAINING_STAGE_B.md)。
