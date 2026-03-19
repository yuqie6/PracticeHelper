# 架构文档 - PracticeHelper

## 1. 系统总览

PracticeHelper 由三个进程组成，通过 HTTP 通信：

```
浏览器 ──HTTP──▶ Go API（:8090）──HTTP──▶ Python sidecar（:8000）──HTTP──▶ LLM Provider
                    │
                    ▼
                SQLite（data/practicehelper.db）
```

- **Go API** 是唯一对外暴露的后端，负责路由、持久化、状态管理，以及转发 AI 任务给 sidecar。
- **Python sidecar** 是内部服务，不直接对外暴露。它接收 Go 的结构化请求，通过 LLM 完成仓库分析、出题、评估和复盘，返回结构化 JSON。
- **Vue 前端** 通过 Vite 反向代理（开发时）或同源部署（生产时）调用 Go API。

## 2. Go 服务分层

```
server/
  cmd/api/main.go           # 入口：加载配置 → 打开数据库 → 创建 sidecar client → 注册路由 → 启动
  internal/
    config/config.go         # 从环境变量读取 Port / DatabasePath / SidecarURL / LogPath
    controller/router.go     # Gin 路由与 handler：解析请求 → 调用 service → 序列化响应
    service/service.go       # 业务逻辑：训练会话编排、出题/评分调度、薄弱点更新
    domain/types.go          # 所有数据结构定义
    repo/repo.go             # SQLite 存储层：DDL、CRUD、FTS5 检索
    sidecar/client.go        # sidecar HTTP 客户端：序列化请求 → POST → 反序列化响应
    observability/request.go # request_id 生成与透传
```

调用链路：`controller → service → repo` + `service → sidecar/client`。controller 不直接访问数据库，service 不直接处理 HTTP。

## 3. Python sidecar 管线

```
sidecar/app/
  main.py            # FastAPI 入口：4 个 POST 端点，每个调用对应的 LangGraph flow
  config.py          # 从环境变量读取 LLM 配置
  schemas.py         # Pydantic 请求/响应模型（与 Go 侧 domain/types.go 字段对齐）
  langgraph_flows.py # 4 条 LangGraph 图，每条是 START → run → END 的单节点图
  agent_runtime.py   # 核心 AI 逻辑：tool-loop 模式与 single-shot 降级
  llm_client.py      # OpenAI 兼容的 HTTP 客户端（用 urllib 实现，无第三方依赖）
  repo_context.py    # 仓库克隆、文件过滤、chunk 切分、技术栈检测
```

### 3.1 AgentRuntime 的双模式执行

每个 AI 任务（出题、评估等）执行时：

1. **Tool-loop 模式（优先）**：向 LLM 发送系统 prompt + 用户 prompt + 工具定义，LLM 可以主动调用工具读取上下文，最多循环 4 轮，最终输出结构化 JSON。
2. **Single-shot 降级**：如果 tool-loop 失败（网络错误、JSON 解析失败等），将所有工具数据直接拼入 prompt，让 LLM 一次性输出结果。
3. 两种模式都失败时，直接抛出异常，不做启发式兜底。

### 3.2 仓库导入管线（repo_context.py）

`analyze_repo` 调用时：

1. `git clone --depth 1` 到临时目录
2. 递归扫描文件，过滤条件：后缀在白名单内（`.go`/`.py`/`.ts`/`.md` 等）、不在忽略目录中（`node_modules`/`.git` 等）、大小 ≤ 256KB
3. 按路径重要性排序（README/docs 优先，`cmd/`/`internal/`/`app/`/`src/` 次之），取前 80 个文件
4. 文本按 1400 字符切分（220 字符重叠），生成 `RepoChunk` 列表，最多保留 120 个
5. 检测技术栈（基于文件名和后缀的关键词映射）
6. 将以上数据打包为 `RepoAnalysisBundle`，供 AgentRuntime 读取并生成项目画像

### 3.3 LangGraph 的实际用法

当前 4 条 LangGraph 图都是单节点线性图（`START → run → END`），本质上是 `AgentRuntime` 方法的薄壳。保留 LangGraph 是为后续扩展（如多步骤训练流程）预留结构，但 v0 不做复杂多节点编排。

## 3.4 日志与 request_id

- Go API 启动时会把结构化 JSON 日志同时写到 stdout 和 `PRACTICEHELPER_SERVER_LOG_PATH`。
- sidecar 启动时会把日志同时写到 stdout 和 `PRACTICEHELPER_SIDECAR_LOG_PATH`。
- Gin 中间件会为每个请求分配或透传 `X-Request-ID`，并写入响应头。
- Go 调用 sidecar 时会继续透传这个 `X-Request-ID`，方便在 `server.log` 和 `sidecar.log` 里串联同一次训练请求。
- 当前日志重点覆盖：HTTP 请求、sidecar 请求、AI 任务开始/结束与耗时。

## 3.5 流式输出与推理摘要

- 训练创建与回答提交流程新增了流式接口，server 对前端返回 `application/x-ndjson`。
- sidecar 也提供对应的流式内部接口，Go API 负责把事件继续转发给前端。
- 事件类型目前包括：
  - `phase`：阶段切换
  - `context`：已读取的上下文类型
  - `reasoning`：可公开的推理摘要
  - `content`：模型输出片段
  - `result`：最终结构化结果
  - `error`：流式过程中出现的错误
- 这里展示的是“可公开推理摘要”，不是模型原始隐藏思维链。
- 当前流式链路为了稳定性，使用的是 single-shot 上下文灌入 + provider streaming；非流式接口仍保留原有 tool-loop 优先策略。

## 4. 数据库 Schema

SQLite 共 9 张表（含 1 张 FTS5 虚拟表），启动时自动建表：

| 表 | 用途 | 关键字段 |
|---|------|---------|
| `user_profile` | 用户画像（单行，`id` 固定为 1） | target_role, current_stage, tech_stacks_json, self_reported_weaknesses_json |
| `project_profiles` | 导入的项目画像 | repo_url（UNIQUE）, summary, highlights_json, challenges_json, followup_points_json |
| `repo_chunks` | 项目源码文本片段 | project_id（外键）, file_path, content, importance, fts_key |
| `repo_chunks_fts` | FTS5 全文索引（镜像 repo_chunks） | chunk_id, project_id, file_path, file_type, content |
| `question_templates` | 基础知识题目模板（预置种子数据） | mode, topic, prompt, focus_points_json, score_weights_json |
| `training_sessions` | 训练会话 | mode（basics/project）, status, total_score, review_id |
| `training_turns` | 训练回合（主问题 + 追问） | session_id（外键）, question, answer, evaluation_json, followup_* |
| `review_cards` | 复盘卡 | session_id（UNIQUE 外键）, overall, highlights_json, gaps_json |
| `weakness_tags` | 薄弱点标签 | kind, label（联合 UNIQUE）, severity, frequency |

JSON 数组字段（如 `tech_stacks_json`）存储为 `TEXT`，在 Go 侧用 `json.Marshal`/`json.Unmarshal` 序列化。

## 5. 训练状态机

一次训练会话的状态流转：

```
waiting_answer ──用户回答主问题──▶ followup ──用户回答追问──▶ review_pending ──生成复盘卡──▶ completed
```

具体流程：

1. `CreateSession`：Go 向 sidecar 请求生成主问题，创建 session（状态 `waiting_answer`）和第一个 turn。
2. 第一次 `SubmitAnswer`：Go 将用户答案发给 sidecar 评分，sidecar 返回评估结果 + 追问问题。状态变为 `followup`。
3. 第二次 `SubmitAnswer`：Go 将追问答案发给 sidecar 评分，然后请求生成复盘卡。状态变为 `completed`。

每轮训练固定为 2 段式（1 个主问题 + 1 个追问），结束后立即生成复盘卡。

### 薄弱点更新机制

- 每次评估后，sidecar 返回的 `weakness_hits` 会写入 `weakness_tags` 表。
- 新弱项直接插入；已有弱项的 severity 按 `current + hit × 0.35` 递增（上限 1.5），frequency +1。
- 如果某次评分 ≥ 75 分，相关弱项的 severity 会下调 0.18（答好了就降温）。

## 6. API 清单

所有接口前缀 `/api`，响应格式 `{"data": ...}` 或 `{"error": {"message": "..."}}`。

| 方法 | 路径 | 用途 |
|------|------|------|
| GET | `/healthz` | 健康检查 |
| GET | `/api/dashboard` | 首页聚合数据（画像 + 弱项 Top 5 + 最近会话 + 今日建议） |
| GET | `/api/profile` | 获取用户画像 |
| POST | `/api/profile` | 创建/更新用户画像 |
| PATCH | `/api/profile` | 同 POST（兼容两种语义） |
| POST | `/api/projects/import` | 导入 GitHub 仓库（触发 sidecar 分析） |
| GET | `/api/projects` | 列出所有已导入项目 |
| GET | `/api/projects/:id` | 获取单个项目详情 |
| PATCH | `/api/projects/:id` | 编辑项目画像 |
| POST | `/api/sessions` | 创建训练会话（触发 sidecar 出题） |
| POST | `/api/sessions/stream` | 流式创建训练会话 |
| GET | `/api/sessions/:id` | 获取会话详情（含所有回合） |
| POST | `/api/sessions/:id/answer` | 提交回答（触发 sidecar 评分，可能触发复盘生成） |
| POST | `/api/sessions/:id/answer/stream` | 流式提交回答 |
| GET | `/api/reviews/:id` | 获取复盘卡 |
| GET | `/api/weaknesses` | 列出所有薄弱点标签 |

## 7. 检索策略

项目训练时的上下文检索仅使用 SQLite FTS5，不引入向量数据库：

- 检索范围限于当前项目的 `repo_chunks`
- 查询词由项目的 `followup_points` + `summary` 拼接而成
- FTS5 查询构建：过滤掉 < 2 字符的 token，每个 token 加双引号后用 `OR` 连接
- 无 FTS 匹配时回退到按 `importance DESC` 排序取 top N
- 每次检索默认取 6 个 chunk

## 8. 前端页面

| 路由 | 页面 | 对应组件 |
|------|------|---------|
| `/` | 首页（Dashboard） | HomeView.vue |
| `/profile` | 用户画像编辑 | ProfileView.vue |
| `/projects` | 项目列表与导入 | ProjectsView.vue |
| `/train` | 训练配置（选模式/主题/项目） | TrainView.vue |
| `/sessions/:id` | 训练过程（问答交互） | SessionView.vue |
| `/reviews/:id` | 复盘卡展示 | ReviewView.vue |

前端通过 `web/src/api/client.ts` 封装的 fetch 函数与 Go API 通信，开发时由 Vite 代理 `/api` 到 `:8090`。

训练创建与回答提交期间，前端不会只显示一个按钮 loading，而是根据当前动作展示阶段化等待提示、推理摘要和流式输出片段。

## 9. 设计约束

- **单用户单体架构**：v0 只服务一个用户，不做多用户、不做微服务拆分。
- **SQLite 即全部存储**：不引入 Redis、PostgreSQL 或向量数据库。
- **LLM 是硬依赖**：sidecar 核心链路不保留启发式兜底，LLM 不可用时直接报错。
- **LangGraph 保持克制**：当前只用单节点图做调度壳，不做复杂多 agent 编排。
- **前端 neo-brutalist 风格**：粗边框、硬阴影、无圆角、黑白主色配鲜艳强调色。禁止圆角、渐变、灰色边框。
