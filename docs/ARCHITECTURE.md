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
    controller/router.go      # Gin transport 层：公开路由、handler、错误映射、stream 输出、中间件
    controller/internal_controller.go # 仅供 sidecar 调用的 internal 检索与 session 详情接口
    domain/types.go           # 共享数据结构：领域实体、请求/响应 DTO、stream event
    infra/sqlite/
      open.go                 # SQLite 连接与 DSN 组装
      bootstrap.go            # migration + seed question templates + knowledge seeds
    repo/
      repo.go                 # Store 定义与 repo.New(db)
      profile_repo.go         # 用户画像读写
      job_target_repo.go      # JD 与分析快照读写
      project_repo.go         # 项目画像、repo chunk、FTS 检索
      import_job_repo.go      # 项目导入任务 CRUD
      prompt_set_repo.go      # 历史 prompt set 聚合与实验查询
      question_template_repo.go # basics 模板查询
      session_repo.go         # 训练 session / turn / 原子状态切换与分页查询
      review_repo.go          # review card 落库与查询
      review_schedule_repo.go # 到期复习项读写与推进
      weakness_repo.go        # weakness 聚合与降温
      evaluation_log_repo.go  # 生成/评估耗时审计
      knowledge_repo.go       # knowledge graph / snapshot / graph 查询与更新
      observation_repo.go     # agent observations 写入与检索
      session_memory_repo.go  # session memory summary 写入与检索
      memory_index_repo.go    # 统一 memory index upsert / list
      scan.go                 # DB row -> domain 扫描器
      util.go                 # repo 内部 JSON/时间/ID/FTS 辅助函数
    service/
      service.go              # Service 定义、构造与服务级错误
      profile_service.go      # 画像与 dashboard 聚合
      job_target_service.go   # JD 管理、分析与默认 JD 逻辑
      import_service.go       # 项目导入编排与后台恢复
      prompt_set_service.go   # prompt set 列表、实验报告与评估审计
      session_creation_service.go # 创建训练会话与出题
      answer_service.go       # 回答提交、多轮状态机与复盘生成
      review_service.go       # 复盘查询与到期复习推进
      agent_context_service.go # question / answer / review 的 agent_context 预装载
      internal_service.go     # sidecar internal 接口对应的服务逻辑
      export_service.go       # Session Markdown / JSON / PDF 导出与批量 ZIP 组装
      audit_service.go        # evaluation_logs 记录
      session_guard.go        # session 原子抢占与恢复语义
      basics_topics.go        # basics topic 归一化与 mixed 选题
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
  main.py            # FastAPI 入口：5 类 AI 任务端点，其中出题 / 评估 / 复盘各带 stream 变体
  config.py          # 从环境变量读取 LLM 配置
  schemas.py         # Pydantic 请求/响应模型（与 Go 侧 domain/types.go 字段对齐）
  langgraph_flows.py # 5 条 LangGraph 图：repo 分析 / JD 分析 / 出题 / 评估 / 复盘
  agent_runtime.py   # 核心 AI 逻辑：agent loop、stream agent loop 与 single-shot 降级
  agent_tools.py     # 回忆工具、Go callback 工具与行动工具注册
  go_client.py       # sidecar 回调 Go internal 接口
  prompt_loader.py   # 从 markdown 文件加载长 system prompt
  runtime_prompts.py # 运行时动态拼装 system/user prompt
  runtime_support.py # agent loop / stream / single-shot 共享辅助逻辑
  prompts/*.md       # 外置的长 system prompt 模板
  llm_client.py      # OpenAI 兼容的 HTTP 客户端（用 urllib 实现，无第三方依赖）
  repo_context.py    # 仓库克隆、文件过滤、chunk 切分、技术栈检测
```

### 3.1 AgentRuntime 的双模式执行

每个 AI 任务（出题、评估等）执行时：

1. **Agent loop 模式（优先）**：向 LLM 发送系统 prompt + 用户 prompt + 工具定义，LLM 可以主动调用读取工具和行动工具，最多循环 8 轮；JSON 结构校验和业务语义校验都在 runtime 内循环里完成。
2. **Single-shot 降级**：如果 agent loop 失败（网络错误、工具契约失配、JSON 或语义校验失败等），将只读工具数据直接拼入 prompt，让 LLM 一次性输出结果；流式接口则退回 provider streaming 的 single-shot 路径。
3. 两种模式都失败时，直接抛出异常，不做启发式兜底。

这意味着当前 sidecar 已经不是“只读工具的受约束推理器”了，而是带长期记忆装载、行动工具和 Go 侧副作用回写的受约束 agent；但它仍然不是完全自治、可自由规划的通用 agent。当前更准确的目标，是把它继续升级成“训练域里的成熟 agent runtime”，而不是把系统直接改写成通用 agent 平台。对应的后续升级方案见 [AGENT_DEEP_REDESIGN_PLAN.md](./AGENT_DEEP_REDESIGN_PLAN.md)。

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

当前并不是“所有图都是单节点”：

- `analyze_repo` 已拆成 `collect_bundle -> rank_chunks -> summarize`
- `analyze_job_target` 目前仍是简单 flow，主要承担岗位分析入口
- `generate_question` 已拆成 `select_strategy -> generate`
- `evaluate_answer` 和 `generate_review` 现在都收成了简单 task graph，图层只负责调用 runtime 并封装 envelope
- `evaluate_answer` / `generate_review` 的校验、重试和 side_effects 收口都已经下沉到 `agent_runtime.py` 内部

也就是说，当前仓库已经把仓库分析、出题、评估和复盘这几条核心链路接成了“图层薄壳 + runtime 实质编排”的结构。后续如果继续做产品升级，更合理的方向是：

- 在 `generate_question` 里继续增强策略选择和上下文筛选
- 在 `evaluate_answer` / `generate_review` 里继续提高输出校验与失败分流的可观测性
- 在 `analyze_repo` 里继续深化检索重排，而不是盲目把图铺得更大

目前最重要的仍然不是把图堆复杂，而是让训练质量、检索质量、状态机和可观测性先稳定。

### 3.4 成熟 Agent Runtime 的分层

如果按当前仓库的实际演进方向来定义，“成熟 agent”不是指更多自由度，而是指运行时闭环更完整。对 PracticeHelper 来说，更合适的分层是：

1. **Task Surface**：接收 `generate_question / evaluate_answer / generate_review` 等结构化任务，并绑定 `prompt_set_id`、`job_target_analysis`、`agent_context` 等请求级上下文。
2. **Context Engine**：由 Go 预装 `agent_context`，并在 sidecar 内做材料压缩、按需补查和 token budget 裁剪。当前 observations / session summaries 已走 `memory_index + embedding + optional rerank` 的混合检索，repo chunk 仍以 SQLite FTS5 主路径为主。
3. **Planning Engine**：LangGraph 继续保持薄壳，只承担最小策略节点和步骤编排；任务内真正的决策、校验和 fallback 继续收在 runtime。
4. **Action Engine**：当前以 `side_effects` 为主，sidecar 负责产出结构化动作意图，Go 负责最终状态迁移与持久化。后续如果要继续放权，优先新增 typed command path，而不是让 sidecar 直写 DB。
5. **Validation and Recovery**：`AgentRuntime` 内部负责 JSON 校验、语义校验、loop 内重试、single-shot fallback 与 stream fallback。
6. **Observability**：Go 继续记录 `prompt_set_id`、`prompt_hash`、`model_name`、`raw_output`、`latency` 等审计字段；sidecar 继续输出 `phase/context/reasoning/result` 等可公开阶段事件。

这里的角色边界也要说清：

- **Go**：产品边界、状态机边界、持久化边界、审计边界
- **sidecar**：上下文理解、规划、工具调用、输出校验、结构化意图生成

当前主路径仍然是单 agent runtime。多 agent 只作为后续高价值长任务的演进方向，不属于当前训练热路径的默认结构。

## 3.5 日志与 request_id

- Go API 启动时会把结构化 JSON 日志同时写到 stdout 和 `PRACTICEHELPER_SERVER_LOG_PATH`。
- sidecar 启动时会把日志同时写到 stdout 和 `PRACTICEHELPER_SIDECAR_LOG_PATH`。
- Go 调 sidecar 的 HTTP 超时由 `PRACTICEHELPER_SERVER_SIDECAR_TIMEOUT_SECONDS` 控制，默认 90 秒。
- Gin 中间件会为每个请求分配或透传 `X-Request-ID`，并写入响应头。
- Go 调用 sidecar 时会继续透传这个 `X-Request-ID`，方便在 `server.log` 和 `sidecar.log` 里串联同一次训练请求。
- 当前日志重点覆盖：HTTP 请求、sidecar 请求、AI 任务开始/结束与耗时。

## 3.6 流式输出与推理摘要

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
- 当前流式链路也优先走 agent loop，把工具读取过程转成 `phase/context/reasoning` 事件；只有 agent loop 无法稳定收口时，才退回 single-shot streaming。
- 流式 `result` 事件内部已经可以携带 `raw_output` 和 `side_effects`，供 Go 侧统一解析并落库。

## 3.7 智能体任务环境（PEAS）

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

## 3.8 环境特性判断

从智能体视角看，这不是一个“静态问答器”的环境，而是一个带外部噪声和状态演化的任务环境。

- **部分可观察（Partially Observable）**：用户真实能力、真实项目贡献、仓库是否完整表达关键事实、LLM 内部推理过程都无法直接观测，只能通过用户回答、仓库材料和结构化输出近似判断。
- **随机 / 非确定性（Stochastic）**：相同输入下，LLM 的题目、评分措辞、追问角度和结构化结果都可能波动；网络、流式中断和仓库内容噪音也会带来不确定性。
- **强序列性（Sequential）**：项目分析影响出题，主问题回答影响追问，追问和评分共同影响复盘，复盘又反过来更新弱项记忆并影响后续训练。
- **动态（Dynamic）**：外部仓库、网络、超时、用户操作、session 状态和弱项权重都会在运行期间变化；尤其流式响应意味着生成过程本身就是运行中的状态变化。
- **离散-连续混合（Hybrid）**：训练模式、session 状态、事件类型、弱项类别是离散的；分数、severity、耗时、上下文长度等是连续量。
- **多主体参与（Human-in-the-loop）**：用户、PracticeHelper agent 和外部 LLM Provider 会共同决定结果，系统不是单主体面对被动环境。
- **已知规则 + 未知分布（Mixed Known/Unknown）**：API、状态机、Schema 和持久化规则是确定的，但“什么问题最有效”“什么评分最稳定”“用户真实短板是什么”并不是完全已知的。

这决定了系统不能只依赖“一条 prompt”，而需要靠状态机、记忆、检索、结构化输出约束和日志链路共同保证质量。

## 3.9 设计建议与当前收口状态

结合上面的环境特性，更适合把设计建议拆成“已落地 / 部分落地 / 仍待补齐”，避免继续把已经完成的收口项写成纯 future work。

1. **证据绑定 ⬜ 仍待补齐**：`gaps`、`weakness_hits`、`followup_question` 目前还没有直接引用用户原文或具体 repo chunk；这一点仍是减少“脑补式判断”的关键缺口。
2. **评分 rubric 程序化 🟡 部分落地**：基础题模板已经能提供 `score_weights`，sidecar prompt 也按 rubric 产出 `score_breakdown`；但总分和最终裁决仍主要依赖 LLM 一次性给出，程序侧还没有把多维评分稳定汇总成统一判定。
3. **状态机异常路径 🟡 已补主要护栏，但还没完全收口**：回答提交的原子抢占、`review_pending -> retry-review` 恢复、流式阶段事件、失败回滚，以及 `evaluate_answer` 输出校验/重试预算都已经存在；但 stream / non-stream 一致性和更系统的端到端异常回归还需要继续补。
4. **弱项记忆衰减与复习入口 🟡 已形成闭环，但算法仍偏简化**：`effectiveSeverity`、`weakness_snapshots`、`review_schedule`、首页待复习卡片、弱项标签回显、直达 `/train?mode=basics&topic=...` 的入口都已落地；当前仍是用 session 总分走简化版 SM-2 推进下次复习时间，而不是按弱项粒度独立反馈。
5. **保守表达与置信度意识 ✅ 已落地基础约束**：`analyze_repo`、`analyze_job_target`、`evaluate_answer` prompt 已明确要求“证据不足时保守表达”；但还没有进一步把证据来源显式展示给用户。
6. **LangGraph 薄壳定位 🟡 已稳定当前边界，但仍有深化空间**：`analyze_repo` / `generate_question` / `evaluate_answer` / `generate_review` 都已经有最小多节点编排；当前优先级仍然是训练质量、检索质量和状态安全，而不是继续堆图复杂度。

## 4. 数据库 Schema

SQLite 当前共 21 张表/虚拟表（其中 20 张普通表 + 1 张 FTS5 虚拟表），启动时由
`internal/infra/sqlite.Bootstrap` 自动建表并补 question template / knowledge seed：

| 表 | 用途 | 关键字段 |
|---|------|---------|
| `user_profile` | 用户画像（单行，`id` 固定为 1） | target_role, current_stage, tech_stacks_json, self_reported_weaknesses_json |
| `project_import_jobs` | 项目导入后台任务 | repo_url, status, stage, message, error_message, project_id |
| `job_targets` | 岗位 JD 元数据 | title, company_name, latest_analysis_id, latest_analysis_status |
| `job_target_analysis_runs` | JD 结构化分析历史 | job_target_id, source_text_snapshot, status, must_have_skills_json, evaluation_focus_json |
| `project_profiles` | 导入的项目画像 | repo_url（UNIQUE）, summary, highlights_json, challenges_json, followup_points_json |
| `repo_chunks` | 项目源码文本片段 | project_id（外键）, file_path, content, importance, fts_key |
| `repo_chunks_fts` | FTS5 全文索引（镜像 repo_chunks） | chunk_id, project_id, file_path, file_type, content |
| `question_templates` | 基础知识题目模板（外部 seed 文件导入） | mode, topic, prompt, focus_points_json, score_weights_json |
| `training_sessions` | 训练会话 | mode（basics/project）, job_target_id, prompt_set_id, status, max_turns, total_score, review_id |
| `training_turns` | 训练回合（每一轮独立一条） | session_id（外键）, turn_index, question, answer, evaluation_json |
| `review_cards` | 复盘卡 | session_id（UNIQUE 外键）, overall, top_fix, top_fix_reason, recommended_next_json |
| `weakness_tags` | 薄弱点标签 | kind, label（联合 UNIQUE）, severity, frequency |
| `weakness_snapshots` | 弱项严重度历史点 | weakness_id, session_id, severity, created_at |
| `review_schedule` | 按 session + weakness tag 生成的待复习计划 | session_id, review_card_id, weakness_tag_id, topic, next_review_at, interval_days, ease_factor |
| `evaluation_logs` | 生成/评估耗时审计 | session_id, turn_id, flow_name, model_name, prompt_set_id, prompt_hash, raw_output, latency_ms |
| `knowledge_nodes` | topic / concept / skill 知识图谱节点 | scope_type, scope_id, parent_id, label, node_type, proficiency, confidence, hit_count |
| `knowledge_edges` | 知识图谱边 | source_id, target_id, edge_type |
| `knowledge_snapshots` | 知识节点分阶段快照 | node_id, session_id, proficiency, evidence, created_at |
| `agent_observations` | 模型沉淀的 pattern / misconception / growth / strategy note | session_id, scope_type, scope_id, topic, category, content, tags_json, relevance |
| `session_memory_summaries` | 每轮训练结束后的长期摘要 | session_id, mode, topic, project_id, job_target_id, prompt_set_id, summary, salience |
| `memory_index` | 各类长期记忆的统一检索索引 | memory_type, scope_type, scope_id, topic, ref_table, ref_id, salience, confidence, freshness |

JSON 数组字段（如 `tech_stacks_json`）存储为 `TEXT`，在 Go 侧用 `json.Marshal`/`json.Unmarshal` 序列化。

## 5. 训练状态机

一次训练会话创建时是**可配置多轮**，默认 2 轮，创建入口允许 2-5 轮；运行中如果
sidecar 返回 `depth_signal=extend`，Go 最多会把本轮 session 上调到 6 轮。

状态流转不再是固定的“主问题 + 1 次追问”，而是由 `turn_index`、`max_turns` 和
`depth_signal` 共同决定：

```
waiting_answer ──用户回答当前轮──▶ evaluating ──继续追问──▶ waiting_answer
                                                └──收口复盘──▶ review_pending ──生成复盘卡──▶ completed
```

具体流程：

1. `CreateSession`：Go 向 sidecar 请求生成第 1 轮问题，创建 session（状态 `waiting_answer`）和第一个 turn。
2. 每次 `SubmitAnswer` 都会先原子地把 session 从 `waiting_answer/active` 抢占到 `evaluating`，防止重复提交。
3. sidecar 评估完成后，Go 会先落库本轮回答、评估结果和 `side_effects`，再决定下一步：
   - `depth_signal=skip_followup`：本轮提前收口，直接进入 review
   - `depth_signal=extend`：如果当前已到 `max_turns` 且 `max_turns < 6`，先把 session 上限加 1
   - 其余情况按当前 `turn_index / max_turns` 正常判断
4. 如果最终判断还需要继续追问：
   - 保存本轮回答和评估结果
   - 从评估结果里取下一轮问题
   - 新插入一条 `training_turns`
   - session 状态回到 `waiting_answer`
5. 如果最终判断应当收口复盘：
   - 保存本轮回答和评估结果
   - session 进入 `review_pending`
   - 继续生成复盘，成功后切到 `completed`
6. `RetrySessionReview`：只有 `review_pending` 会话允许重试。服务端会先把状态原子切到 `evaluating`，避免重复点“重试生成复盘”触发并发复盘。

这里有两个实现细节要明确：

- 追问已经不是内嵌字段，而是独立 `turn`
- 多轮训练的默认轮次判断由 `turn_index` 和 `max_turns` 决定，但 sidecar 已能通过 `depth_signal` 提前收口或额外补一轮

### 5.2 Review 推荐与学习路径兜底

当前 review 收口已经不再完全依赖模型自由发挥：

- sidecar 可以通过 `recommended_next` 或 `side_effects.recommended_next` 给出下一轮建议
- Go 在 `persistReview` 时会继续做推荐归一化，确保 `mode / topic / project_id / reason` 至少满足当前 session 语义
- basics 模式下，如果 review 本身太稀疏，Go 会尝试基于当前 topic 的知识图谱回填 `suggested_topics` 和 `next_training_focus`
- 当前工作区已经开始沉淀第一版 `prerequisite` 边，用来表达“推荐 topic -> 当前 topic”的学习先修关系

这层兜底的作用不是替代模型，而是保证：

- 推荐结果不会明显漂移出当前训练语境
- 稀疏 review 仍然能给出最小学习路径
- 后续学习路径与知识图谱可以继续收敛到同一套结构化语义

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

普通 JSON 接口前缀为 `/api`，成功时返回 `{"data": ...}`，失败时返回 `{"error": {"code": "...", "message": "..."}}`。
其中有两个例外：

- `/healthz` 是顶层健康检查接口，不走 `/api`
- stream 接口返回 `application/x-ndjson`，导出接口直接返回附件内容（单次 `markdown/json/pdf`，批量 `zip`），不走统一 JSON envelope

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
| GET | `/api/prompt-sets` | 列出可用 prompt set |
| GET | `/api/prompt-experiments/prompt-sets` | 列出 Prompt 实验页可比较的 prompt set |
| GET | `/api/prompt-experiments` | 读取两个 prompt set 的聚合对比报告 |
| GET | `/api/sessions` | 分页列出训练历史，支持 mode / topic / status 筛选 |
| POST | `/api/sessions` | 创建训练会话（触发 sidecar 出题） |
| POST | `/api/sessions/export` | 批量导出选中的 session（ZIP） |
| POST | `/api/sessions/stream` | 流式创建训练会话 |
| GET | `/api/sessions/:id` | 获取会话详情（含所有回合） |
| GET | `/api/sessions/:id/evaluation-logs` | 获取当前 session 的评估审计日志 |
| GET | `/api/sessions/:id/export?format=markdown\|json\|pdf` | 导出单次 session 报告 |
| POST | `/api/sessions/:id/answer` | 提交回答（触发 sidecar 评分，可能触发复盘生成） |
| POST | `/api/sessions/:id/answer/stream` | 流式提交回答 |
| POST | `/api/sessions/:id/retry-review` | 对 `review_pending` 会话重试生成复盘 |
| GET | `/api/reviews/:id` | 获取复盘卡 |
| GET | `/api/weaknesses` | 列出所有薄弱点标签 |
| GET | `/api/weaknesses/trends` | 获取首页弱项趋势 sparkline 数据 |
| GET | `/api/reviews/due` | 获取当前到期的待复习项 |
| POST | `/api/reviews/due/:id/complete` | 标记一条待复习已完成，并推进下次复习时间 |

## 6.5 阶段 B 扩展（已形成产品级主链路）

下面这部分不再只是“第一版接进去”，而是已经成为当前可依赖的产品级主链路；当前更适合把它当成既成事实，再把剩余注意力放到阶段 C。

### 当前已实现接口

| 方法 | 路径 | 用途 |
|------|------|------|
| GET | `/api/job-targets` | 列出所有 JD 条目 |
| POST | `/api/job-targets` | 创建一份新的 JD |
| POST | `/api/job-targets/clear-active` | 清空当前默认 JD |
| GET | `/api/job-targets/:id` | 获取单个 JD 详情 |
| PATCH | `/api/job-targets/:id` | 编辑 JD 元数据和原文 |
| POST | `/api/job-targets/:id/activate` | 将某份 JD 设为默认 JD |
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
- 如果用户显式选了 JD，但 `latest_analysis_status` 不是 `succeeded` 或没有可用成功快照，则返回 `job_target_not_ready`，不允许开始这轮训练
- JD 原文被修改导致 `stale`，或后续新分析进入 `running` / `failed` 时，`latest_successful_analysis` 仍会保留给 dashboard / history / review 只读回看
- 默认 JD 只会在 `succeeded + latest_successful_analysis` 同时存在时参与推荐；否则 dashboard 仍显示该 JD，但 `recommendation_scope` 会自动回退到 `generic`

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
| `/job-targets` | JD / 岗位页 | JobTargetsView.vue |
| `/projects` | 项目列表与导入 | ProjectsView.vue |
| `/prompt-experiments` | Prompt 实验 / 审计页 | PromptExperimentsView.vue |
| `/train` | 训练配置（选模式/主题/项目） | TrainView.vue |
| `/history` | 训练历史页 | HistoryView.vue |
| `/sessions/:id` | 训练过程（问答交互） | SessionView.vue |
| `/reviews/:id` | 复盘卡展示 | ReviewView.vue |

前端通过 `web/src/api/client.ts` 封装的 fetch 函数与 Go API 通信，开发时由 Vite 代理 `/api` 到 `:8090`。

训练创建与回答提交期间，前端不会只显示一个按钮 loading，而是根据当前动作展示阶段化等待提示、推理摘要和流式输出片段。

## 9. 设计约束

- **单用户单体架构**：v0 只服务一个用户，不做多用户、不做微服务拆分。
- **SQLite 即全部存储**：不引入 Redis、PostgreSQL 或向量数据库。
- **LLM 是硬依赖**：sidecar 核心链路不保留启发式兜底，LLM 不可用时直接报错。
- **LangGraph 保持克制**：当前线上主路径仍是单 agent runtime + 最小节点化调度；多 agent 只保留为后续高价值长任务的演进方向，不进入当前训练热路径。
- **前端 neo-brutalist 风格**：粗边框、硬阴影、无圆角、黑白主色配鲜艳强调色。禁止圆角、渐变、灰色边框。

### 为什么这样设计

上述约束不是偷懒，而是有意为之：

- **单用户单体架构**：让系统先对一个人有用，验证训练价值后再考虑多用户。
- **SQLite 即全部存储**：弱点记忆、项目画像、训练历史都在一个库里，未来接 JD 理解或学习规划时不需要引入新的存储层。
- **LangGraph 薄壳**：保留扩展点但不过早引入复杂编排，先把单 agent 的检索、memory、恢复和观测做成熟；后续如果确实需要多 agent，也只在高价值长任务中渐进进入。
