# 架构文档 - PracticeHelper

> 状态：已按 2026-03-26 当前工作区收口。
> 这份文档只回答 5 件事：系统怎么分、数据怎么流、状态怎么变、
> 接口怎么分、当前边界是什么。

工程收口专项看
[ARCHITECTURE_CONVERGENCE_PLAN.md](../plans/ARCHITECTURE_CONVERGENCE_PLAN.md)；
sidecar agent runtime 深改看
[AGENT_DEEP_REDESIGN_PLAN.md](../plans/AGENT_DEEP_REDESIGN_PLAN.md)。

## 1. 一张图看全局

```text
Browser
  │
  ▼
Vue Web
  │ HTTP
  ▼
Go API (:8090)
  │                ┌───────────────────────┐
  ├── SQLite ────▶ │ data/practicehelper.db │
  │                └───────────────────────┘
  │ HTTP
  ▼
Python Sidecar (:8000)
  │ HTTP
  ▼
LLM / Embedding / Rerank Provider
```

## 2. 三个进程各负责什么

| 面 | 负责 | 不负责 |
|:---|:-----|:-------|
| `web/` | 页面、流式展示、表单与导航 | 不直接碰数据库，不决定训练状态机 |
| `server/` | 公开 API、状态机、持久化、审计、导出、内部回调 | 不做 LLM 推理 |
| `sidecar/` | 仓库分析、出题、评估、复盘、runtime 工具调用与校验 | 不直接对外暴露，不直接写数据库 |

一句话：**Go 是产品边界和持久化边界，sidecar 是受约束的推理执行层。**

## 3. Go 后端分层

`server/cmd/api/main.go` 的启动顺序是：

1. 读取配置
2. 打开并 bootstrap SQLite
3. 构造 repo / sidecar client / vector store
4. 构造 service
5. 注册 Gin 路由并启动服务

核心包职责如下：

| 包 | 作用 |
|:---|:-----|
| `internal/config` | 读取 `PRACTICEHELPER_SERVER_*` 配置 |
| `internal/controller` | HTTP transport、错误映射、流式输出、internal 路由 |
| `internal/service` | 训练编排、状态机、推荐、导出、审计、internal callback |
| `internal/repo` | SQLite 读写、分页、原子状态切换、检索与索引表访问 |
| `internal/domain` | DTO、状态常量、runtime / command 契约 |
| `internal/infra/sqlite` | SQLite 打开、建表、seed、迁移补丁 |
| `internal/sidecar` | 调 sidecar 的 HTTP 客户端与 NDJSON 解码 |
| `internal/vectorstore` | 向量库接入，目前实现 `qdrant` |

默认调用方向固定为：

```text
controller -> service -> repo
                   └-> sidecar client
```

## 4. 前端入口

当前前端路由就是下面这 9 个页面：

| 路由 | 作用 |
|:-----|:-----|
| `/` | 首页，承接 Diagnose / Today / Recommendation |
| `/profile` | 用户画像 |
| `/job-targets` | 岗位 JD 管理与分析历史 |
| `/projects` | 项目导入、项目列表、项目画像 |
| `/prompt-experiments` | Prompt 版本与实验审计 |
| `/train` | 训练配置与开始训练 |
| `/history` | 历史 session 列表、筛选、批量导出 |
| `/sessions/:id` | 训练过程页 |
| `/reviews/:id` | 复盘页 |

## 5. sidecar runtime 现在是什么

sidecar 不是单纯的 prompt wrapper，当前已经具备：

- FastAPI 入口：`analyze_repo`、`analyze_job_target`、`generate_question`、
  `evaluate_answer`、`generate_review`，其中训练相关接口带 stream 变体
- `AgentRuntime`：agent loop、single-shot fallback、stream fallback
- `runtime_tools`：只读工具、side effect 工具、typed command 工具
- `flows/langgraph.py`：图层保持薄壳，只做最小流程编排
- `prompts/sets/*`：prompt set 版本化管理

当前工具面分成两类动作：

- `side_effects`：如 `record_observation`、`update_knowledge`、
  `set_depth_signal`、`suggest_next_session`
- typed command：`transition_session`、`upsert_review_path`

typed command 会通过 Go internal callback 拿结构化结果，再回流到
`command_results`，不会让 sidecar 直接改数据库。

## 6. 核心数据

SQLite 是当前唯一真相源。关键表可以按这几组理解：

| 数据组 | 主要表 | 作用 |
|:-------|:-------|:-----|
| 用户画像 | `user_profile` | 目标岗位、技术栈、项目、自感弱项 |
| 项目与导入 | `project_profiles`、`project_import_jobs`、`repo_chunks`、`repo_chunks_fts` | 项目画像、导入任务、源码切片与 FTS |
| 岗位 JD | `job_targets`、`job_target_analysis_runs` | JD 原文、分析快照、默认 JD |
| 训练主链路 | `training_sessions`、`training_turns`、`review_cards` | session、turn、review |
| 弱项与复习 | `weakness_tags`、`weakness_snapshots`、`review_schedule` | 弱项记忆、趋势、待复习 |
| Prompt / 审计 | `prompt_preferences`、`evaluation_logs` | prompt 选择、实验、评估日志 |
| 知识与长期记忆 | `knowledge_nodes`、`knowledge_edges`、`knowledge_snapshots`、`agent_observations`、`session_memory_summaries`、`memory_index` | 知识图谱、observation、session summary、统一检索索引 |
| 向量索引 | `memory_embedding_records`、`memory_embedding_jobs`、`repo_chunk_embedding_records`、`repo_chunk_embedding_jobs` | memory / repo chunk 的 embedding 任务和索引状态 |

## 7. 关键状态机

### 7.1 训练 session

当前核心状态是：

```text
draft -> waiting_answer -> evaluating
                           ├-> waiting_answer
                           └-> review_pending -> completed
```

说明：

- `review_pending` 不是失败态，而是“问答已结束，等待复盘生成”。
- 关键轮次决策优先看 typed command `transition_session` 的结果。
- Go 仍保留最终状态迁移和兜底上限。

### 7.2 项目导入任务

```text
queued -> running(analyzing_repository / persisting_project)
           ├-> completed
           └-> failed
```

### 7.3 JD 分析状态

可见状态是：

- `idle`
- `running`
- `succeeded`
- `failed`
- `stale`

只有可绑定的成功快照才能进入新训练；过期或失败的 JD 只能回看，不能绑定新 session。

## 8. 关键链路

### 8.1 项目导入

1. Web 调 `POST /api/projects/import`
2. Go 创建 `project_import_jobs`
3. 后台任务调用 sidecar 做仓库分析
4. Go 持久化 `project_profiles`、`repo_chunks`、FTS
5. 如果向量写入启用，再异步建立 repo chunk embedding

### 8.2 训练

1. Web 调 `POST /api/sessions` 或 `POST /api/sessions/stream`
2. Go 创建 session，预装 `agent_context`
3. sidecar 生成问题
4. Web 提交回答到 `POST /api/sessions/:id/answer` 或 stream 变体
5. Go 抢占 session 状态，sidecar 评估回答
6. Go 根据结果继续追问或进入 `review_pending`

### 8.3 复盘与推荐

1. sidecar 生成 review，输出 `side_effects` / `command_results`
2. Go 归一化推荐结果并落库 `review_cards`
3. Go 更新弱项、待复习、observation、knowledge、session summary
4. 首页、历史页、复盘页消费这些持久化结果

### 8.4 检索

当前是“SQLite 真相源 + 可选向量增强”：

- `memory_index` 负责 observation / session summary 的统一候选池
- repo chunk 当前支持 `Qdrant vector recall + optional rerank + FTS5 fallback`
- review 结果会记录 `retrieval_trace`

## 9. 接口边界

### 9.1 Go 对外 API

按领域可以这样看：

- profile / dashboard
- job targets
- projects / import jobs
- prompt sets / prompt preferences / prompt experiments
- sessions / reviews / weaknesses / exports

### 9.2 Go internal API

只给 sidecar 用：

- `GET /internal/search-chunks`
- `GET /internal/session-detail/:id`
- `POST /internal/agent-commands`

### 9.3 sidecar internal API

Go 调 sidecar 的入口：

- `POST /internal/analyze_repo`
- `POST /internal/analyze_job_target`
- `POST /internal/generate_question`
- `POST /internal/generate_question/stream`
- `POST /internal/evaluate_answer`
- `POST /internal/evaluate_answer/stream`
- `POST /internal/generate_review`
- `POST /internal/generate_review/stream`
- `POST /internal/embed_memory`
- `POST /internal/rerank_memory`

## 10. 配置与运行

### 10.1 Server

Go 端主要读取：

- `PRACTICEHELPER_SERVER_PORT`
- `PRACTICEHELPER_SERVER_DB_PATH`
- `PRACTICEHELPER_SERVER_SIDECAR_URL`
- `PRACTICEHELPER_SERVER_SIDECAR_TIMEOUT_SECONDS`
- `PRACTICEHELPER_SERVER_LOG_PATH`
- `PRACTICEHELPER_SERVER_VECTOR_STORE_*`
- `PRACTICEHELPER_SERVER_VECTOR_*_ENABLED`

### 10.2 Sidecar

sidecar 主要读取：

- `PRACTICEHELPER_SIDECAR_MODEL`
- `PRACTICEHELPER_SIDECAR_OPENAI_BASE_URL`
- `PRACTICEHELPER_SIDECAR_OPENAI_API_KEY`
- `PRACTICEHELPER_SIDECAR_EMBEDDING_*`
- `PRACTICEHELPER_SIDECAR_RERANK_*`
- `PRACTICEHELPER_SIDECAR_SERVER_BASE_URL`
- `PRACTICEHELPER_INTERNAL_TOKEN`
- `PRACTICEHELPER_SIDECAR_LOG_PATH`

## 11. 当前硬边界

- 单用户、单体、SQLite 真相源。
- Go 负责最终状态机、持久化、审计和恢复语义。
- sidecar 可以给出结构化意图，但不能越过 Go 直接改库。
- 多 agent 不是当前训练热路径。
- Prompt 版本可选，但核心 prompt 仍以源码 markdown 为准，不走前端全量编辑。
