# Agent Deep Redesign Plan

> 状态：审校更新版（2026-03-23）。
> 真实完成度：Phase A / Phase B 已完成；
> Phase C 已落第一版 command path，但仍未收口；Phase D 仍未开始。
> 目标：在保留 Go 产品边界和执行边界的前提下，把 PracticeHelper 的
> `sidecar` 从“基础版 constrained agent runtime”逐步升级为
> “训练域里的成熟 agent runtime”。
> 本文讨论的是训练域成熟 agent，不是通用 agent 平台，也不是
> “让模型直接接管数据库”。

---

## 1. 文档定位

这份文档回答 4 个问题：

1. PracticeHelper 当前的 agent 到底已经做到哪了
2. “成熟 agent”在这个仓库里具体是什么意思
3. Go、sidecar、memory、tools、状态机之间的边界怎么划
4. 后续怎么从成熟单 agent 走到多 agent，但又不把当前主链路搞散

这里先把主线关系说清楚：

- 当前产品主线仍然是“训练深度与留存升级”
- agent 成熟化是这条主线的核心技术底座
- 近期优先级是把单 agent 做成熟
- 多 agent 不是当前热路径方案，而是后续高价值任务的演进阶段

所以本文不是“另开一条和产品无关的 AI 支线”，而是：

> 用成熟 agent runtime 支撑训练体验、长期记忆、复盘质量和持续留存。

### 1.1 审校结论（2026-03-23）

这次按当前工作区代码和测试逐条核对后，结论可以直接收成 3 句：

- **已经落地**：`AgentRuntime` 的 tool loop / retry / single-shot fallback /
  stream fallback、`agent_context` 五类预装载、`memory_index` + rule rerank +
  vector similarity + optional rerank、`retrieval_trace`、Go 侧
  `depth_signal` 状态机接线、review 推荐归一化与知识图谱兜底、
  `prerequisite` 第一版边推断、以及 `prompt_set / prompt_hash / model /
  raw_output` 审计日志、统一 `runtime_trace` / 失败分类、`context compaction`
  第一版，以及关键 side effects 的落库可见性。
- **已收口**：observability 现在已经能同时看到 prompt 元信息、原始输出、
  review 检索轨迹、stream 工具路径、统一结构化 trace，以及 Go 侧
  `persist_*` 持久化轨迹；observations 已做会话内去重，knowledge update
  继续走 upsert。
- **已起步但未收口**：typed command path 第一版已经接上
  `transition_session / upsert_review_path`、Go internal callback
  `/internal/agent-commands`、`command_results` 回流和结果校验；多 agent
  coordinator、shared artifacts 和多角色协作仍然停留在设计层。

本次审校还做了最基本实测：

- `cd sidecar && uv run --group dev pytest tests/test_agent_runtime.py tests/test_prompt_versioning.py tests/test_main_streaming.py -q` 通过，`65 passed`
- `cd server && go test -tags sqlite_fts5 ./internal/sidecar ./internal/service ./internal/controller -count=1` 通过
- `pnpm --dir web test -- --run src/lib/streaming.spec.ts src/api/client.spec.ts` 通过，`14` 个 test files、`46` 个 tests 通过

---

## 2. 当前现实

在继续设计之前，先把现在到底有什么说清楚。

### 2.1 当前不是 prompt app，而是基础版 constrained agent

当前 `sidecar/app/agent_runtime.py` 已经具备：

- 非流式 `generate_question / evaluate_answer / generate_review / analyze_repo / analyze_job_target`
  都优先走 `_run_agent_loop`
- `_run_agent_loop` 允许模型读取工具、调用行动工具，再输出最终 JSON
- JSON 结构校验和业务语义校验都在 runtime 内部完成
- agent loop 收口不稳时会退回 `_run_single_shot`
- 流式 `generate_question / evaluate_answer / generate_review`
  也优先走 `_stream_agent_loop`
- stream `result` 事件已经能携带 `raw_output`、`side_effects`、`command_results`
  和 `trace`

也就是说，当前 sidecar 已经不是“带几个只读工具的 prompt wrapper”，而是：

- 有 loop
- 有 fallback
- 有行动工具
- 有 side effects
- 有 stream 回退语义
- 有第一版 retrieval planner

但它还不是成熟 agent。当前更准确的说法是：

> 一个带长期记忆预装载、少量行动工具和 Go 边界约束的训练域单 agent。

### 2.2 当前工具分层已经存在，但能力面还偏窄

当前工具大致分成 3 类：

#### A. flow-specific 只读工具

来自 `runtime_prompts.py`，例如：

- `read_question_templates`
- `read_project_brief`
- `read_context_chunks`
- `read_evaluation_context`
- `read_turn_history`

它们负责把当前任务的直接材料喂给模型。

#### B. agent memory 工具

来自 `agent_tools.py`，例如：

- `recall_training_context`
- `recall_weakness_profile`
- `recall_knowledge_graph`
- `recall_observations`
- `recall_session_summaries`

它们负责让模型读取 Go 预装载的长期上下文。

#### C. 动作工具

当前已经有：

- `record_observation`
- `update_knowledge`
- `suggest_next_session`
- `set_depth_signal`
- `transition_session`
- `upsert_review_path`

其中前四个仍然先写入 `side_effects`，再由 Go 统一落库；后两个则通过
Go internal callback 走第一版 typed command path，拿到结构化 command result
后再继续完成校验和收口。

这条边界是当前设计里最重要的安全原则之一，必须保留。

### 2.3 当前持久 memory 已落地，但利用方式仍偏保守

Go 端目前已经具备这些持久层：

- `weakness_tags`
- `weakness_snapshots`
- `review_schedule`
- `evaluation_logs`
- `knowledge_nodes`
- `knowledge_edges`
- `knowledge_snapshots`
- `agent_observations`
- `session_memory_summaries`
- `memory_index`

已经落地的事实包括：

- 保存画像或装载 `agent_context` 时会补 `knowledge_nodes` 基础种子
- `evaluate_answer / generate_review` 的 `side_effects`
  已能回写 observations 和 knowledge updates
- review 落库后会生成 `session_memory_summaries`
- observations / knowledge / session summaries 都会同步进 `memory_index`
- `agent_context` 预装载 observations / session summaries 时，
  已经优先走 `memory_index`，再按 `ref_id` 回源 materialize
- review 持久化时已经会带上 `retrieval_trace`，导出链路也能看到命中来源、
  分数和 fallback 情况

但当前 memory 的主要问题不是“没有存”，而是“还不会更聪明地拿”：

- observations / session summaries 已经不是纯 Top-N，而是
  `memory_index` 候选池上的规则重排 + vector similarity + optional rerank
- project 场景仍以预选 repo chunks 为主，按需回调只是补充
- `retrieval_trace` 已经能解释为什么命中、分数怎么来的、是否走了 fallback
- 还没有通用 context compaction，也还没有扩成更激进的全局 semantic recall

### 2.4 当前训练深度已经能受 sidecar 影响，但最终边界仍在 Go

`answer_service.go` 当前逻辑已经不是固定 2 轮写死：

- 默认按 `turn_index / max_turns` 驱动多轮 FSM
- `transition_session` command result 会优先裁决 `skip_followup / extend`
- 没有 command result 时，才回退 `side_effects.depth_signal`
- 当最终决策是 `extend` 且当前已到上限但 `max_turns < 6` 时，Go 仍会额外补一轮
- 真正创建 follow-up turn、调整 `max_turns`、进入 `review_pending`
  的最终决定仍由 Go 执行

也就是说：

- 模型已经可以先向 Go 申请关键轮次决策，再提出“跳过追问”或“追加一轮”的结构化意图
- 旧 `side_effects.depth_signal` 仍保留为兼容 fallback，但不再是热路径首选
- 但它没有直接改 session 状态的权限
- Go 仍然保留最终状态机边界和兜底上限

### 2.5 当前 Go 侧已经承担第一版知识图谱兜底推荐

这部分不能再写成 future work，因为当前工作区已经落了第一版。

Go 侧当前已经在做：

- 对 `recommended_next` 做 basics/project 归一化
- 当 review 太稀疏时，基于知识图谱回填 `suggested_topics`
  与 `next_training_focus`
- 基于 review 推荐 topic 与当前 topic，沉淀第一版 `prerequisite` 边
- 为 basics 推荐 reason 生成知识图谱兜底文案

这意味着当前系统已经从：

> 完全依赖模型自由发挥推荐下一轮

走到了：

> 模型给建议，Go 再结合知识图谱和 session 语义做归一化与最小学习路径兜底。

这个方向应当继续强化，而不是回退。

---

## 3. 什么叫“训练域里的成熟 agent”

这里先把“成熟”两个字讲清楚。

成熟 agent 不是：

- 给模型更多自由
- 上更多 agent
- 放开更多写权限
- 把数据库直接暴露给 LLM

成熟 agent 应该同时具备 6 层能力。

### 3.1 Context Engine

回答：

> 它到底看到了什么。

它不只看当前请求，还要系统地看到：

- 当前训练任务
- 用户画像
- 弱项画像
- 知识图谱子图
- 历史 observations
- 相近 session summaries
- 必要时回调更多 repo chunks 或 session detail

### 3.2 Planning Engine

回答：

> 它下一步准备怎么干。

它要能决定：

- 先读哪些工具
- 当前上下文够不够
- 什么时候补检索
- 什么时候追问
- 什么时候收口
- 什么时候该退回保守路径

### 3.3 Action Engine

回答：

> 它能不能真的推进任务。

成熟 agent 必须能通过工具改变系统状态，但动作必须：

- 语义明确
- 可验证
- 可审计
- 可回放
- 可幂等

它能“改世界”，但不能获得原始 SQL 自由。

### 3.4 Validation and Recovery

回答：

> 它错了以后怎么办。

成熟 agent 的关键不是“永不出错”，而是：

- 结构不对会重试
- 语义不对会修正
- loop 不稳会降级
- 流式收口失败会 fallback
- 关键失败有恢复入口

### 3.5 Memory Engine

回答：

> 它能不能越练越像认识这个用户。

成熟 agent 的 memory 不是聊天记录，而是结构化长期记忆：

- weakness
- knowledge graph
- observations
- session summaries
- retrieval index

重点不是“存进去”，而是“什么时候拿什么”。

### 3.6 Observability

回答：

> 你怎么知道它到底为什么这么做。

成熟 agent 必须可观察、可解释、可调试。

至少要能看到：

- prompt set / prompt hash / model
- raw output
- tool 使用路径
- fallback 原因
- validation/retry 轨迹
- session 状态迁移
- 关键副作用落库结果

一句人话总结：

> 成熟 agent = 能稳定把训练任务做完，而不是只是看起来很聪明。

---

## 4. 目标架构

在这个仓库里，更合适的目标态不是“通用 agent 平台”，而是：

- Go 继续做产品边界、状态机边界和执行边界
- sidecar 逐步升级为训练域成熟 agent runtime
- 多 agent 只在高价值长任务进入，不进入当前训练热路径

### 4.1 系统角色划分

#### Go：世界状态层

Go 负责：

- 对外 API
- session 状态机
- 持久化
- 审计
- 并发一致性
- 恢复入口
- 最终副作用落库

Go 不是 agent 的大脑，但它是 agent 所处世界的裁判和骨架。

#### sidecar：认知决策层

sidecar 负责：

- 上下文理解
- 任务内规划
- 工具调用
- 输出校验
- 结构化意图生成
- 流式阶段反馈
- 后续多 agent 协调

一句话：

> Go 负责“稳”，sidecar 负责“强”。

### 4.2 六层运行时分层

成熟单 agent 的目标结构建议固定成下面 6 层。

#### Layer 1：Task Surface

职责：

- 接收 `GenerateQuestion / EvaluateAnswer / GenerateReview`
  等结构化任务
- 持有请求级 metadata，例如 `prompt_set_id`、`request_id`、
  `job_target_analysis`、`agent_context`

当前实现位置：

- `sidecar/app/main.py`
- `sidecar/app/schemas.py`
- `server/internal/sidecar/client.go`

#### Layer 2：Context Engine

职责：

- 预装载 `agent_context`
- 按 scope、topic、project、job target 做 memory 候选筛选
- 需要时回调 Go internal API 补材料
- 做 token budget 裁剪与材料压缩

当前已落地的基本原则：

- 预装载 + 按需回调混合
- memory_index 先筛，再回源 materialize
- repo chunks 继续以 SQLite FTS5 主路径为主

当前已经补上的能力：

- observation / session summary 的更强排序
- retrieval trace

近期还要继续强化的方向：

- 更明确的 context compaction
- 更广的 semantic recall 范围与压缩策略

#### Layer 3：Planning Engine

职责：

- 决定当前任务的执行策略
- 决定要不要继续补查
- 决定当前使用哪些工具
- 决定是否需要保守收口

当前已落第一版：

- `analyze_repo`: `collect_bundle -> rank_chunks -> summarize`
- `generate_question`: `select_strategy -> generate`

近期目标：

- 不把所有复杂度堆到 LangGraph 图里
- 继续保持“薄图 + runtime 实质编排”

#### Layer 4：Action Engine

职责：

- 接受模型动作意图
- 将动作分流到 cheap/local path 或 critical/long path
- 保证动作语义清晰、边界稳定

这一层采用混合双轨。

##### Track A：`side_effects`

适用场景：

- cheap
- local
- batch-friendly
- 不需要立即改变 Go 世界状态才能继续思考

当前继续保留这些动作：

| Tool | 输出位置 | 用途 |
|---|---|---|
| `record_observation` | `side_effects.observations` | 记录观察 |
| `update_knowledge` | `side_effects.knowledge_updates` | 更新知识图谱 |
| `suggest_next_session` | `side_effects.recommended_next` | 建议下一轮训练 |
| `set_depth_signal` | `side_effects.depth_signal` | 控制追问深度 |

##### Track B：typed command API（已落第一版，仍在扩展）

适用场景：

- 关键状态迁移
- 需要立即看到 Go 执行结果
- 长动作或重动作
- 必须保证幂等与审计

当前已经落地的是最小 command baseline，而不是完整 command 平台。

建议第一版 command 只覆盖少量高价值动作：

| Command Type | 用途 |
|---|---|
| `transition_session` | 请求关键 FSM 迁移，如准备额外 follow-up 或确认提前收口 |
| `upsert_review_path` | 请求 Go 生成或确认推荐学习路径，并返回规范化结果 |
| `enqueue_long_job` | 预留命令类型，后续可请求 Go 启动长任务，如离线分析、批量证据整理、异步索引补算 |

typed command 的核心原则：

- sidecar 只能发结构化命令
- Go 决定是否执行、如何执行、返回什么结果
- sidecar 不获得原始 DB 写权限

#### Layer 5：Validation and Recovery

职责：

- JSON 结构校验
- 业务语义校验
- loop 内重试
- single-shot fallback
- stream fallback
- 恢复理由与失败分类

当前已落地：

- `used_any_tool` 护栏
- JSON 提取与校验
- 语义校验回灌
- loop 轮次上限
- 非流式/流式 fallback

近期要补的不是“有没有 fallback”，而是：

- 更细的失败分类
- 更系统的恢复语义
- command / decision 轨迹的进一步细化

#### Layer 6：Observability

职责：

- 记录 prompt 版本、模型版本、原始输出
- 记录工具轨迹与 fallback
- 记录状态迁移与落库结果
- 向 UI 暴露可公开阶段事件

当前对前端公开的流式事件仍保持：

- `status`
- `phase`
- `context`
- `trace`
- `reasoning`
- `content`
- `result`
- `error`

当前已经落地的可观测面：

- sidecar header + Go 审计日志里已记录 `prompt_set / prompt_hash / model / raw_output`
- `generate_review` 已会把 `retrieval_trace` 带进 review 持久化和 export
- stream 的 `context` 事件已经能反映工具使用路径
- `trace` 事件已经覆盖 `prepare_context / tool_call / command / validate /
  fallback / finalize / persist / error`
- `runtime_trace` 已能同时记录 sidecar runtime、`context_compaction`
  摘要和 Go 侧 `persist_*` 持久化结果

当前仍未进入的可观测深水区只有两类：

- `command` 轨迹已经落第一版：`command` phase trace 与流式
  `command_requested / command_applied / command_deferred / command_rejected`
  已可见；后续只继续细化粒度和多 agent 预算轨迹
- 多 agent 共享工件和 coordinator 预算轨迹

---

## 5. 数据契约与接口边界

### 5.1 当前真实契约

当前 3 类核心训练请求继续保留 `agent_context`：

```python
class GenerateQuestionRequest(BaseModel):
    ...
    agent_context: AgentContext | None = None


class EvaluateAnswerRequest(BaseModel):
    ...
    agent_context: AgentContext | None = None


class GenerateReviewRequest(BaseModel):
    ...
    agent_context: AgentContext | None = None
```

当前 2 类核心 envelope 已经升级为
`result + side_effects + command_results + raw_output + trace`：

```python
class EvaluateAnswerEnvelope(BaseModel):
    result: EvaluationResult
    side_effects: EvaluateAnswerSideEffects
    command_results: list[AgentCommandResult] = []
    raw_output: str = ""
    trace: RuntimeTrace | None = None


class GenerateReviewEnvelope(BaseModel):
    result: ReviewCard
    side_effects: GenerateReviewSideEffects
    command_results: list[AgentCommandResult] = []
    raw_output: str = ""
    trace: RuntimeTrace | None = None
```

这部分已经是当前训练热路径的真实协议，不需要回退到旧的三段式 envelope。

### 5.2 当前 memory 读取契约

当前 `AgentContext` 的最小稳定形状固定为：

```python
class AgentContext(BaseModel):
    profile: ProfileSnapshot | None = None
    knowledge_subgraph: KnowledgeSubgraph | None = None
    observations: list[AgentObservation] = []
    weakness_profile: list[WeaknessTag] = []
    session_summaries: list[SessionMemorySummary] = []
```

近期不再把 memory 重新写成“万能大表”，而是继续坚持：

- source of truth 分表存
- retrieval 通过 `memory_index` 聚合
- materialize 时按 `ref_table + ref_id` 回源

### 5.3 当前 command 契约（后续继续扩展）

当前第一版 command path 已经采用下面的接口形状；后续新增命令也沿这套结构扩展。

#### AgentCommandEnvelope

```python
class AgentCommandEnvelope(BaseModel):
    command_id: str
    command_type: Literal[
        "transition_session",
        "upsert_review_path",
        "enqueue_long_job",
    ]
    session_id: str = ""
    idempotency_key: str
    reason: str = ""
    payload: dict[str, Any] = {}
```

#### AgentCommandResult

```python
class AgentCommandResult(BaseModel):
    command_id: str
    status: Literal["accepted", "rejected", "applied", "deferred"]
    applied: bool = False
    data: dict[str, Any] = {}
    error_code: str = ""
    error_message: str = ""
```

语义固定为：

- sidecar 只能提交 typed command
- Go 返回结构化执行结果
- command 必须带 `idempotency_key`
- command 只用于关键状态迁移和长动作

### 5.4 多 agent 共享工件契约

多 agent 阶段默认使用下面的共享工件，而不是 agent 之间互传自由文本。

#### TaskSpec

```python
class TaskSpec(BaseModel):
    task_id: str
    role: Literal["planner", "researcher", "executor", "reviewer"]
    goal: str
    input_refs: list[str] = []
    tool_budget: int = 0
    stop_condition: str = ""
```

#### AgentResult

```python
class AgentResult(BaseModel):
    task_id: str
    status: Literal["completed", "retry", "blocked", "failed"]
    summary: str = ""
    artifacts: list[dict[str, Any]] = []
    action_proposals: list[dict[str, Any]] = []
    open_questions: list[str] = []
```

#### ReviewVerdict

```python
class ReviewVerdict(BaseModel):
    task_id: str
    accepted: bool
    issues: list[str] = []
    retry_hint: str = ""
```

这些类型的目标不是“让系统更像论文”，而是：

- 让 agent 间 handoff 可验证
- 让 reviewer 可以明确指出哪里不通过
- 让 coordinator 能控制预算与重试

---

## 6. 工具体系

### 6.1 当前单 agent 工具分配

当前训练域工具继续按下面映射推进：

| 任务 | 回忆工具 | 动作工具 |
|---|---|---|
| `generate_question` | `recall_training_context` `recall_knowledge_graph` `recall_observations` `search_repo_chunks` | — |
| `evaluate_answer` | `recall_training_context` `recall_knowledge_graph` `recall_observations` `search_repo_chunks` | `record_observation` `update_knowledge` `set_depth_signal` `transition_session` |
| `generate_review` | `recall_training_context` `recall_weakness_profile` `recall_knowledge_graph` `recall_observations` `recall_session_summaries` `get_session_detail` | `record_observation` `update_knowledge` `suggest_next_session` `upsert_review_path` |

补充说明：

- 上表主要列 agent memory / callback / action 工具
- 每个 flow 仍然额外带有各自的 `read_*` prompt tools，例如
  `read_evaluation_context`、`read_turn_history`、`read_question_templates`
- `search_repo_chunks` 只在绑定项目且 Go callback 可用时注入
- 当前 `stable-v1` 和 `candidate-v1` prompt 已默认优先走
  `transition_session / upsert_review_path`；`set_depth_signal /
  suggest_next_session` 继续保留为兼容 fallback
- `get_session_detail` 只在 review 场景且有 session id 时注入

当前约束继续保留：

- 热路径任务不应该继续无限膨胀工具面
- 当前实现里各 flow 的工具数实际已经偏厚，大多在 `7~10` 个量级
- 新增工具前先压缩职责，不顺手膨胀

### 6.2 retrieval 的近期设计

近期 retrieval 主线固定为：

1. Go 先组 `agent_context`
2. `memory_index` 先筛 observation / session summary 候选
3. 按 `ref_table + ref_id` 回源 materialize
4. sidecar 必要时按需回调 `search_repo_chunks / get_session_detail`
5. 最终再做 token budget 裁剪

补充说明：

- 当前这一步主要还是依赖固定 limit 和 `compact_chunks` 的轻量裁剪
- 它还不是通用 context compaction，这也是 Phase B 还没收口的原因之一

几个明确边界：

- 不一次把所有 memory 都塞进 prompt
- repo chunk 已升级成 `Qdrant vector recall + optional rerank + FTS5 fallback`
- 不把 graph 或更大范围的 repo 语义层急着全 embedding 化
- observation / session summary 仍然是第一批已经收口完成的 embedding 对象

### 6.3 embedding / rerank 的近期边界

第一批适合引入 embedding 的对象：

- `agent_observations`
- `session_memory_summaries`
- `repo_chunks`（仅限当前项目范围，且保留 FTS5 fallback）

第一批不急着 embedding 的对象：

- `knowledge_nodes`
- `knowledge_edges`
- `weakness_tags`

当前已落地的排序链路：

- scope match
- topic match
- project match
- job target match
- recency
- salience
- confidence
- observation / session summary 的 embedding similarity
- 可选 rerank model 二次排序

当前还没做的是把 semantic recall 范围继续放大，而不是“有没有 embedding”。

---

## 7. 单 agent 演进阶段

### Phase A：当前基线（已落第一版）

目标：

- 有可用的 constrained runtime
- 有长期记忆预装载
- 有 side effects 回写
- 有最小 retrieval planner
- 有流式 fallback

当前已成立的事实：

- `AgentRuntime` 已是训练主脑
- `agent_context` 已接进 question / answer / review
- `memory_index` 已参与 observations / session summaries 检索
- Go callback 已能补 repo chunk 与 session detail
- `depth_signal` 已接进 Go FSM
- review 推荐链路已开始做 Go 侧知识图谱兜底

### Phase B：成熟单 agent 基线（已收口）

目标：

> 先把单 agent 做成熟，而不是先把多 agent 立起来。

近期优先级固定为：

1. 检索更稳
2. memory 利用更准
3. 恢复更完整
4. 可观测性更强

本阶段已经确认落地的部分：

- retrieval trace
- observation / session summary 的更强排序
- `memory_index + vector + optional rerank` 的混合召回
- review 推荐归一化、学习路径兜底和 `prerequisite` 第一版边推断
- context compaction
- validation / fallback 的失败分类
- stream / non-stream 一致性观测
- 关键动作的幂等与落库可见性

本阶段现在可以视为收口完成，后续不再继续把这些项当作主施工点。

本阶段完成标准：

- 单 agent 在 question / answer / review 主链路上更稳
- 出错后能解释为什么降级、为什么重试
- memory 不只排序更准，还能把取材与压缩过程解释清楚
- review 推荐和学习路径更少依赖模型自由发挥

### Phase C：受控放权（已落第一版，进行中）

目标：

- 在单 agent 稳定之后，再增强 planner 与动作执行能力
- 为关键动作引入 typed command path

本阶段已经落地的部分：

- `transition_session` command：评估阶段可让 sidecar 向 Go FSM 申请
  `skip_followup / extend` 的结构化决策
- `upsert_review_path` command：复盘阶段可让 sidecar 先向 Go 请求规范化
  `recommended_next / suggested_topics / next_training_focus`
- `command_results` 已进入 envelope，并参与 review / evaluation 的结果校验
- Go internal callback `/internal/agent-commands` 已成为 sidecar command 主链路
- `evaluate_answer / generate_review` 的默认 prompt 已切到新命令语义
- runtime 已有单次 run 内的 command budget 与 idempotency cache
- stream 与 trace 已能暴露 `command_requested / command_applied / command_deferred / command_rejected`

本阶段还要补：

- 更强 planner
- 更细的 action budget
- 更清楚的 action result 回流
- 更多高价值 command 类型

本阶段完成标准：

- sidecar 不只会产出 side effects，还能对关键动作发起 typed command
- Go 仍然保留最终状态迁移与持久化边界
- 新能力不破坏当前训练热路径的稳定性

### Phase D：多 agent 进入高价值长任务（尚未开始）

目标：

- 不是把所有任务都改成多 agent
- 而是把多 agent 用在真正值得的任务上

优先适用任务：

- 更深的 `analyze_repo`
- 复杂 `generate_review`
- 长材料证据整理
- prompt audit / experiment analysis

明确不优先进入的任务：

- basics `generate_question`
- 常规 `evaluate_answer`
- 低时延训练热路径

---

## 8. 多 agent 蓝图

多 agent 的价值不在“更像 agent”，而在：

- 把复杂长任务拆开
- 把读、做、校验分层
- 在不放弃 Go 边界的前提下增加任务完成度

### 8.1 角色设计

#### Planner

职责：

- 判断当前任务是否值得进入多 agent
- 切分子任务
- 分配工具预算
- 定义停止条件

输入：

- `TaskSpec`
- `ContextBundle`

输出：

- 一组子任务 `TaskSpec`

限制：

- Planner 自己不直接写状态
- Planner 只能产生计划和 action proposal

#### Researcher

职责：

- 补证据
- 补检索
- 归纳上下文

可见工具：

- 只读工具
- Go internal read callback

限制：

- 不可调用动作工具
- 不可发 typed command

#### Executor

职责：

- 根据 planner 和 researcher 结果推进动作
- 调用 action tool
- 必要时发 typed command

限制：

- 不可直接写 DB
- 动作必须受 tool budget 与 command budget 限制

#### Reviewer

职责：

- 检查结果是否过约束
- 检查证据是否足够
- 检查动作是否越权
- 决定接受、重试或降级

输出：

- `ReviewVerdict`

### 8.2 协调器设计

多 agent 的 coordinator 仍放在 sidecar，不放到 Go。

职责：

- 接收顶层任务
- 判断单 agent 还是多 agent
- 管理 shared artifacts
- 控制 agent budget
- 收敛最终结果
- 必要时退回单 agent fallback

协调器必须强制的规则：

- 任一子 agent 不得绕过 coordinator 直接对外部系统产生未审计写动作
- 关键写动作仍经 Go command gate 或最终 side effects 回写
- reviewer 不通过时，不允许 executor 结果直接收口

### 8.3 shared artifacts

多 agent 之间默认只交换结构化工件，不交换任意自然语言长上下文。

最小共享工件：

- `TaskSpec`
- `ContextBundle`
- `EvidenceBundle`
- `AgentResult`
- `ReviewVerdict`

共享原则：

- 每个工件都要可序列化
- 每个工件都要能落审计日志
- 每个工件都要能标注来源和 budget 消耗

### 8.4 准入条件

只有满足下面条件的任务才允许进入多 agent：

- 单 agent 方案已明显过载
- 任务价值足够高
- 上下文切分收益明显
- 写冲突可控
- 时延预算允许

如果不满足，默认继续走成熟单 agent。

---

## 9. 验证方式

### 9.1 当前单 agent 基线验证

- `make test-sidecar`
- `make test-server`

本次审校实测结果：

- `cd sidecar && uv run --group dev pytest tests/test_agent_runtime.py tests/test_prompt_versioning.py tests/test_main_streaming.py -q`：`65 passed`
- `cd server && go test -tags sqlite_fts5 ./internal/sidecar ./internal/service ./internal/controller -count=1`：通过
- `pnpm --dir web test -- --run src/lib/streaming.spec.ts src/api/client.spec.ts`：`14` 个 test files、`46` 个 tests 通过

重点场景：

- 模型跳过工具直接输出时，能退回 single-shot fallback
- 结构校验失败时，能在 loop 内重试
- `skip_followup` / `extend` 能被 Go FSM 正确解释
- `review_pending -> retry-review` 仍可恢复
- review 推荐稀疏时，Go 能给出知识图谱兜底的推荐与学习路径
- `prerequisite` 第一版边能在知识图谱里回看
- observations / session summaries 已走 `memory_index + embedding + optional rerank`
  的混合召回；向量链路失败时会退回现有规则重排

### 9.2 近期成熟化验收

验收点应聚焦在：

- retrieval 是否更稳
- memory 是否更会用
- fallback 是否更清晰
- 观测是否足够解释失败

当前已经可以验到的场景：

- 第一版 retrieval trace 已能随 review/export 暴露 observations / session summaries
  的命中原因、分数和 fallback 情况
- recommendation / learning path 在模型稀疏输出下仍然可解释

当前仍建议补的验收场景：

- stream / non-stream 在核心结果上保持一致，并且有统一 trace 对照
- fallback 原因可以被日志明确定位，并沉淀成更稳定的失败分类
- side effects 落库结果在调试路径中可见，而不只是在 repo 状态里间接体现

### 9.3 后续 command path 验收

在 typed command 引入后，至少验证：

- command 带 `idempotency_key`
- Go 会返回结构化 command result
- rejected / deferred / applied 三类结果可区分
- command 失败不会直接污染训练热路径状态

### 9.4 后续多 agent 验收

在多 agent 真正进入前，至少验证：

- planner 能正确决定是否进入多 agent
- researcher 不越权写状态
- reviewer 可阻止错误结果直接收口
- coordinator 可在多 agent 失败时退回单 agent

---

## 10. 非目标

当前明确不做：

- 把 PracticeHelper 写成通用 agent 平台
- 让 sidecar 获得数据库直写权限
- 让 LLM 直接写 SQL
- 为了“更像 agent”把当前训练热路径改成默认多 agent
- 在 graph / repo chunk 上立即全量 embedding 化
- 在单 agent 还没稳住前就优先铺复杂长链自治

---

## 11. 结论

这份方案的核心不是：

> 让模型更自由。

而是：

> 在保留 Go 边界的前提下，把 sidecar 逐步做成训练域里的成熟 agent runtime。

对当前仓库来说，最合理的路线是：

1. 保留 Go 做产品边界、状态机和执行边界
2. 把 sidecar 继续做强，先把单 agent 做成熟
3. 用混合双轨动作模型扩展 agent 能力，但不放弃 Go 最终落库
4. 只在高价值长任务引入多 agent

一句话总结：

> 近期先把单 agent 做成熟，后续再把多 agent 做正确。

按当前工作区的真实完成度，近期不该再把 `retrieval_trace`、
`memory_index` 混合召回或 Go 侧推荐兜底当成主施工点；这些已经落了第一版。
当前真正还没收口的，是 `Phase C` 的 command 范围扩张、action budget
和 command 结果观测；`Phase D` 的多 agent coordinator / shared artifacts
则仍然没有开工。
