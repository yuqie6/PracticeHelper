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
  cmd/api/main.go           # 入口装配：加载配置 → sqlite.Open/Bootstrap → repo.New → service.New → 注册路由 → 启动
  internal/
    config/config.go          # 从环境变量读取 Port / DatabasePath / SidecarURL / SidecarTimeout / LogPath
    controller/router.go      # Gin transport 层：路由、handler、错误映射、stream 输出、中间件
    domain/types.go           # 共享数据结构：领域实体、请求/响应 DTO、stream event
    infra/sqlite/
      open.go                 # SQLite 连接与 DSN 组装
      bootstrap.go            # migration + seed question templates
    repo/
      repo.go                 # Store 定义与 repo.New(db)
      profile_repo.go         # 用户画像读写
      project_repo.go         # 项目画像、repo chunk、FTS 检索
      import_job_repo.go      # 项目导入任务 CRUD
      question_template_repo.go # basics 模板查询
      session_repo.go         # 训练 session / turn / 原子状态切换
      review_repo.go          # review card 落库与查询
      weakness_repo.go        # weakness 聚合与降温
      scan.go                 # DB row -> domain 扫描器
      util.go                 # repo 内部 JSON/时间/ID/FTS 辅助函数
    service/
      service.go              # Service 定义、构造与服务级错误
      profile_service.go      # 画像与 dashboard 聚合
      import_service.go       # 项目导入编排与后台恢复
      session_service.go      # 会话创建、答题状态机、review 收口
      weakness_service.go     # weakness 辅助逻辑与 dashboard 文案 helper
    sidecar/client.go         # sidecar HTTP 客户端：序列化请求 → POST → 反序列化响应/流
    observability/request.go  # request_id 生成与透传
```

调用链路：`cmd/api(main)` 先完成 `sqlite.Open + sqlite.Bootstrap + repo.New` 装配，然后进入
`controller → service → repo` 和 `service → sidecar/client`。controller 不直接访问数据库，
service 不直接处理 HTTP，repo 也不再负责数据库连接与 migration。

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

`analyze_repo` 被后台导入任务调用时：

1. `git clone --depth 1` 到临时目录
2. 递归扫描文件，过滤条件：后缀在白名单内（`.go`/`.py`/`.ts`/`.md` 等）、不在忽略目录中（`node_modules`/`.git` 等）、大小 ≤ 256KB
3. 按路径重要性排序（README/docs 优先，`cmd/`/`internal/`/`app/`/`src/` 次之），取前 80 个文件
4. 文本按 1400 字符切分（220 字符重叠），生成 `RepoChunk` 列表，最多保留 120 个
5. 检测技术栈（基于文件名和后缀的关键词映射）
6. 将以上数据打包为 `RepoAnalysisBundle`，供 AgentRuntime 读取并生成项目画像

Go API 不再同步等待整个导入结束，而是先创建 `project_import_jobs` 记录，再由后台 goroutine 执行：

1. `queued`：任务已创建，等待后台启动
2. `running + analyzing_repository`：调用 sidecar 完成克隆、筛选、切分和项目画像草稿生成
3. `running + persisting_project`：把 `project_profiles`、`repo_chunks` 和 `repo_chunks_fts` 一起落库
4. `completed` / `failed`：前端通过轮询 `GET /api/import-jobs` 感知结果，并在项目页展示通知；导航区会对 `queued/running` 任务显示全局导入提示，失败任务可直接触发 retry

### 3.3 LangGraph 的实际用法

当前 4 条 LangGraph 图都是单节点线性图（`START → run → END`），本质上是 `AgentRuntime` 方法的薄壳。保留 LangGraph 是为后续扩展（如多步骤训练流程）预留结构，但 v0 不做复杂多节点编排。

## 3.4 日志与 request_id

- Go API 启动时会把结构化 JSON 日志同时写到 stdout 和 `PRACTICEHELPER_SERVER_LOG_PATH`。
- sidecar 启动时会把日志同时写到 stdout 和 `PRACTICEHELPER_SIDECAR_LOG_PATH`。
- Go 调 sidecar 的 HTTP 超时由 `PRACTICEHELPER_SERVER_SIDECAR_TIMEOUT_SECONDS` 控制，默认 90 秒。
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
- 前端不会再直接把 `content` 当原始 JSON 文本整段展示，而是按任务片段拆分，再尽量解析成“问题草稿 / 评估结果 / 复盘草稿”三类结构化卡片。
- 当前流式链路为了稳定性，使用的是 single-shot 上下文灌入 + provider streaming；非流式接口仍保留原有 tool-loop 优先策略。

## 3.6 智能体任务环境（PEAS）

这里的“智能体”特指 `sidecar/app/agent_runtime.py` 驱动的 PracticeHelper 面试训练 agent。`server/` 负责编排、状态管理和持久化，`web/` 负责交互展示；真正承担“分析仓库、出题、评分、追问、复盘”任务的是 sidecar 里的 AgentRuntime。

### Performance measure（绩效指标）

- **训练质量**：问题是否贴合用户目标、项目背景和历史弱项，而不是泛泛而谈。
- **评分准确性**：评分、优点、缺口、追问是否稳定、可解释、可复核。
- **复盘价值**：复盘卡是否能明确指出短板，并给出下一轮可执行的训练方向。
- **个性化程度**：是否持续利用用户画像、项目画像和弱项记忆做针对性训练。
- **交互体验**：是否具备可见等待态、流式阶段反馈、超时后的合理报错。
- **系统一致性**：训练状态流转、弱项更新、项目画像写入是否前后一致。
- **成本与鲁棒性**：在控制 token/上下文开销的同时，对网络波动、LLM 输出漂移和仓库噪音保持稳定。

### Environment（环境）

- **用户环境**：准备后端 / AI Agent 面试的求职者，会填写画像、导入项目并进行多轮问答。
- **项目环境**：用户导入的 GitHub 仓库，包括 README、源码、文档、目录结构和技术栈痕迹。
- **系统环境**：SQLite 中的用户画像、项目画像、训练会话、训练回合、复盘卡和薄弱点标签。
- **外部依赖环境**：Git 仓库拉取、LLM Provider、HTTP 网络、运行时配置和超时约束。
- **交互环境**：浏览器 -> Go API -> Python sidecar -> LLM Provider 的串行调用链。
- **时间环境**：任务不是单次问答，而是“出题 -> 回答 -> 评分 -> 追问 -> 复盘 -> 记忆更新”的连续过程。

### Actuators（执行器）

- 生成项目画像：输出 `summary`、`highlights`、`challenges`、`tradeoffs`、`ownership_points`、`followup_points`。
- 生成训练问题：根据 basics / project 模式和当前上下文输出主问题与 `expected_points`。
- 评估用户答案：输出 `score`、`score_breakdown`、`strengths`、`gaps`、`followup_question`、`weakness_hits`。
- 生成复盘卡：输出 `overall`、`highlights`、`gaps`、`suggested_topics`、`next_training_focus`。
- 发出流式事件：对前端持续输出 `phase`、`context`、`reasoning`、`content`、`result`、`error`。
- 触发长期状态变化：通过 Go API 写入 session 状态、review 卡和 `weakness_tags`。

### Sensors（传感器）

- **用户输入**：画像表单、训练模式、主题、项目选择和文本回答。
- **项目上下文**：仓库概览、repo chunks、技术栈检测结果和检索命中的代码片段。
- **历史记忆**：已有薄弱点标签、历史训练记录和项目画像。
- **系统状态**：当前 session 所处阶段、上一次评估结果、是否需要追问或生成复盘。
- **模型与运行反馈**：LLM 返回的结构化 JSON、流式片段、工具调用结果、超时和解析错误。
- **可观测链路**：`X-Request-ID`、server/sidecar 日志、接口响应耗时和失败事件。

## 3.7 环境特性判断

从智能体视角看，这不是一个“静态问答器”的环境，而是一个带外部噪声和状态演化的任务环境。

- **部分可观察（Partially Observable）**：用户真实能力、真实项目贡献、仓库是否完整表达关键事实、LLM 内部推理过程都无法直接观测，只能通过用户回答、仓库材料和结构化输出近似判断。
- **随机 / 非确定性（Stochastic）**：相同输入下，LLM 的题目、评分措辞、追问角度和结构化结果都可能波动；网络、流式中断和仓库内容噪音也会带来不确定性。
- **强序列性（Sequential）**：项目分析影响出题，主问题回答影响追问，追问和评分共同影响复盘，复盘又反过来更新弱项记忆并影响后续训练。
- **动态（Dynamic）**：外部仓库、网络、超时、用户操作、session 状态和弱项权重都会在运行期间变化；尤其流式响应意味着生成过程本身就是运行中的状态变化。
- **离散-连续混合（Hybrid）**：训练模式、session 状态、事件类型、弱项类别是离散的；分数、severity、耗时、上下文长度等是连续量。
- **多主体参与（Human-in-the-loop）**：用户、PracticeHelper agent 和外部 LLM Provider 会共同决定结果，系统不是单主体面对被动环境。
- **已知规则 + 未知分布（Mixed Known/Unknown）**：API、状态机、Schema 和持久化规则是确定的，但“什么问题最有效”“什么评分最稳定”“用户真实短板是什么”并不是完全已知的。

这决定了系统不能只依赖“一条 prompt”，而需要靠状态机、记忆、检索、结构化输出约束和日志链路共同保证质量。

## 3.8 设计建议

结合上面的环境特性，当前版本最值得优先加强的是以下几点：

1. **给评分和追问增加证据绑定**：让 `gaps`、`weakness_hits`、`followup_question` 尽量对应到用户回答片段或具体 repo chunk，减少“脑补式判断”。
2. **把评分 rubric 程序化收口**：把总分拆成正确性、结构性、深度、trade-off、表达清晰度等稳定维度，再由程序侧汇总，降低 LLM 随机波动。
3. **补强状态机异常路径**：明确流式与非流式的一致性、重复提交保护、超时后的恢复策略、review 失败时的中间状态处理。
4. **给弱项记忆增加衰减和阈值**：区分“偶发卡壳”和“稳定弱项”，避免因为单次误判长期污染 `weakness_tags`。
5. **在项目分析中保留置信度意识**：证据不足时宁可保守表达，也不要把不确定推断写成确定事实。
6. **继续保持 LangGraph 的薄壳定位**：当前最重要的是检索质量、评分稳定性和状态安全，而不是引入更复杂的多 agent 编排。

## 4. 数据库 Schema

SQLite 共 10 张表（含 1 张 FTS5 虚拟表），启动时由 `internal/infra/sqlite.Bootstrap`
自动建表并补种子数据：

| 表 | 用途 | 关键字段 |
|---|------|---------|
| `user_profile` | 用户画像（单行，`id` 固定为 1） | target_role, current_stage, tech_stacks_json, self_reported_weaknesses_json |
| `project_import_jobs` | 项目导入后台任务 | repo_url, status, stage, message, error_message, project_id |
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
2. 第一次 `SubmitAnswer`：服务端先原子地把 session 从 `waiting_answer/active` 抢占到 `evaluating`，防止并发重复提交；主问题评估成功后状态变为 `followup`。
3. 第二次 `SubmitAnswer`：服务端先把 session 从 `followup` 抢占到 `evaluating`，追问评估完成后切到 `review_pending`，再继续生成复盘；成功后状态变为 `completed`。
4. `RetrySessionReview`：只有 `review_pending` 会话允许重试。服务端会先把状态原子切到 `evaluating`，避免重复点“重试生成复盘”触发并发复盘。

每轮训练固定为 2 段式（1 个主问题 + 1 个追问），结束后立即生成复盘卡。

### 5.1 重复提交保护与恢复语义

- `training_sessions.status` 现在不仅是展示字段，也是提交抢占锁：
  - 回答提交前，必须先完成 `source_status -> evaluating` 的原子状态切换。
  - 如果切换失败，会按最新状态返回明确冲突语义，例如 `session_busy`、`session_review_pending`、`session_completed`。
- 这意味着：
  - 用户在同一页狂点提交，或多个标签页同时提交时，只有第一个请求会真正进入评估。
  - `review_pending` 不再接受新的答案提交，只允许走 `POST /api/sessions/:id/retry-review`。
  - 如果评估阶段失败，服务端会把 `evaluating` 回滚回上一个可恢复状态；如果复盘生成失败，则保留 `review_pending` 供后续重试，并返回 `review_generation_retry` 错误码提示前端切到恢复入口。
- 前端训练页会在 `evaluating` 期间锁定输入区，并在收到冲突错误后主动刷新 session，避免用户停留在过期状态。

### 薄弱点更新机制

- 每次评估后，sidecar 返回的 `weakness_hits` 会写入 `weakness_tags` 表。
- 当前稳定支持的薄弱点分类为 `topic`、`project`、`expression`、`followup_breakdown`、`depth`、`detail`。
- 新弱项直接插入；已有弱项的 severity 按 `current + hit × 0.35` 递增（上限 1.5），frequency +1。
- 如果某次评分 ≥ 75 分，相关弱项的 severity 会下调 0.18（答好了就降温）。
- 读取弱项列表时，不直接按数据库里的原始 severity 排序，而是按 `effectiveSeverity = severity / (1 + stale_days / 21)` 计算“当前热度”；排序依据变为 `effectiveSeverity -> frequency -> last_seen_at`。
- 旧弱项再次命中时，也会先按这个“当前热度”作为新的累加基线，避免很久没出现的问题继续沿着陈旧高点叠加。

## 6. API 清单

所有接口前缀 `/api`，响应格式 `{"data": ...}` 或 `{"error": {"message": "..."}}`。

| 方法 | 路径 | 用途 |
|------|------|------|
| GET | `/healthz` | 健康检查 |
| GET | `/api/dashboard` | 首页聚合数据（画像 + 弱项 Top 5 + 最近会话 + 今日建议） |
| GET | `/api/profile` | 获取用户画像 |
| POST | `/api/profile` | 创建/更新用户画像 |
| PATCH | `/api/profile` | 同 POST（兼容两种语义） |
| POST | `/api/projects/import` | 创建项目导入后台任务（202 返回 job） |
| GET | `/api/import-jobs` | 列出最近的项目导入任务 |
| GET | `/api/import-jobs/:id` | 查询单个导入任务状态 |
| POST | `/api/import-jobs/:id/retry` | 重试失败的导入任务 |
| GET | `/api/projects` | 列出所有已导入项目 |
| GET | `/api/projects/:id` | 获取单个项目详情 |
| PATCH | `/api/projects/:id` | 编辑项目画像 |
| POST | `/api/sessions` | 创建训练会话（触发 sidecar 出题） |
| POST | `/api/sessions/stream` | 流式创建训练会话 |
| GET | `/api/sessions/:id` | 获取会话详情（含所有回合） |
| POST | `/api/sessions/:id/answer` | 提交回答（触发 sidecar 评分，可能触发复盘生成） |
| POST | `/api/sessions/:id/answer/stream` | 流式提交回答 |
| POST | `/api/sessions/:id/retry-review` | 对 `review_pending` 会话重试生成复盘 |
| GET | `/api/reviews/:id` | 获取复盘卡 |
| GET | `/api/weaknesses` | 列出所有薄弱点标签 |

## 6.5 阶段 B 扩展（已落地第一版）

下面这部分已经接入当前代码主链路，但仍处于阶段 B 的 MVP 收口期。

### 当前已实现接口

| 方法 | 路径 | 用途 |
|------|------|------|
| GET | `/api/job-targets` | 列出所有 JD 条目 |
| POST | `/api/job-targets` | 创建一份新的 JD |
| GET | `/api/job-targets/:id` | 获取单个 JD 详情 |
| PATCH | `/api/job-targets/:id` | 编辑 JD 元数据和原文 |
| POST | `/api/job-targets/:id/analyze` | 对当前 JD 原文创建一次新的分析快照 |
| GET | `/api/job-targets/:id/analysis-runs` | 查看该 JD 的分析历史 |
| GET | `/api/job-targets/analysis-runs/:id` | 查看某次分析快照详情 |

### 当前已实现数据对象

- `job_targets`
  - 一份 JD 的当前实体
  - 至少包含：`title`、`company_name`、`source_text`、`latest_analysis_id`、`latest_analysis_status`、`last_used_at`
- `job_target_analysis_runs`
  - 某份 JD 的一次分析快照
  - 至少包含：`source_text_snapshot`、`summary`、`must_have_skills`、`bonus_skills`、`responsibilities`、`evaluation_focus`、`status`、`error_message`

### 当前训练绑定规则

- TrainView 开始训练前，用户手动选择本轮参考的 JD
- `CreateSession` 请求新增可选 `job_target_id`
- server 在创建 session 时解析出该 JD **最新成功分析快照**，并持久化到 session 的 `job_target_id` / `job_target_analysis_id`
- 一轮训练创建后，出题、评分、复盘都只吃这次绑定的 JD 快照，不跟随后续切换漂移
- 如果用户没选 JD，则保持当前通用训练路径
- 如果用户选了 JD 但没有可用的成功分析快照，则不允许开始这轮训练

## 7. 检索策略

项目训练时的上下文检索仅使用 SQLite FTS5，不引入向量数据库：

- 检索范围限于当前项目的 `repo_chunks`
- 查询词由项目的 `followup_points` + `summary` 拼接而成
- FTS5 查询构建：过滤掉 < 2 字符的 token，每个 token 加双引号后用 `OR` 连接
- 无 FTS 匹配时回退到按 `importance DESC` 排序取 top N
- 每次检索默认取 6 个 chunk

## 8. 前端页面

当前已经实现的页面如下：

| 路由 | 页面 | 对应组件 |
|------|------|---------|
| `/` | 首页（Dashboard） | HomeView.vue |
| `/profile` | 用户画像编辑 | ProfileView.vue |
| `/projects` | 项目列表与导入 | ProjectsView.vue |
| `/train` | 训练配置（选模式/主题/项目） | TrainView.vue |
| `/sessions/:id` | 训练过程（问答交互） | SessionView.vue |
| `/reviews/:id` | 复盘卡展示 | ReviewView.vue |

阶段 B 已新增：

| 路由 | 页面 | 对应职责 |
|------|------|---------|
| `/job-targets` | JD / 岗位页 | 管理多份 JD、查看最新分析结果与分析历史 |

前端通过 `web/src/api/client.ts` 封装的 fetch 函数与 Go API 通信，开发时由 Vite 代理 `/api` 到 `:8090`。

训练创建与回答提交期间，前端不会只显示一个按钮 loading，而是根据当前动作展示阶段化等待提示、推理摘要和流式输出片段。

## 9. 设计约束

- **单用户单体架构**：v0 只服务一个用户，不做多用户、不做微服务拆分。
- **SQLite 即全部存储**：不引入 Redis、PostgreSQL 或向量数据库。
- **LLM 是硬依赖**：sidecar 核心链路不保留启发式兜底，LLM 不可用时直接报错。
- **LangGraph 保持克制**：当前只用单节点图做调度壳，不做复杂多 agent 编排。
- **前端 neo-brutalist 风格**：粗边框、硬阴影、无圆角、黑白主色配鲜艳强调色。禁止圆角、渐变、灰色边框。

### 为什么这样设计

上述约束不是偷懒，而是有意为之：

- **单用户单体架构**：让系统先对一个人有用，验证训练价值后再考虑多用户。
- **SQLite 即全部存储**：弱点记忆、项目画像、训练历史都在一个库里，未来接 JD 理解或学习规划时不需要引入新的存储层。
- **LangGraph 薄壳**：保留扩展点但不过早引入复杂编排，未来需要多步骤训练流程时可以在现有结构上自然生长。
