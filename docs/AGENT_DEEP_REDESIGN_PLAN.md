# Agent Deep Redesign Plan

> 状态：提案，已按 2026-03-22 当前仓库实现审校。  
> 目标：把 sidecar 从“带只读工具的 LLM pipeline”渐进升级为“带知识图谱记忆、真实工具调用、受约束自主权”的 ReAct agent。  
> 本文不是原始想法直抄，而是已经结合当前 `server + sidecar + web` 代码现实修正过的执行版方案。

---

## 1. 当前现实

在继续设计之前，先把现在到底是什么说清楚。

### 1.1 Sidecar 现在已经不是纯 single-shot

当前 `sidecar/app/agent_runtime.py` 已经具备：

- 非流式 `generate_question / evaluate_answer / generate_review / analyze_repo / analyze_job_target` 会先走 `_run_tool_loop`
- `_run_tool_loop` 已允许模型读取运行时工具，再输出最终 JSON
- tool loop 失败时会降级到 `_run_single_shot`
- 流式链路目前仍走 `_stream_single_shot_task`，没有复用 tool loop

也就是说，当前 sidecar 已经有“受约束的工具调用”，但还不是本文要推进的真正 agent。

### 1.2 当前工具仍然是只读上下文获取器

`sidecar/app/runtime_prompts.py` 里已有的工具，基本都是：

- 读取模板
- 读取项目画像
- 读取 context chunks
- 读取 weakness memory
- 读取 session/turn 摘要

这些工具的特点是：

- 数据都来自请求体预装载
- 不会跨请求持久化新记忆
- 不会触发副作用
- 不会按需回调 Go 获取新信息

所以今天的 sidecar 更准确地说，是“带只读工具上下文的结构化推理器”，不是有长期记忆和可行动能力的 agent。

### 1.3 当前持久记忆只有 weakness / review / evaluation log

Go 端目前已经有这些持久化层：

- `weakness_tags`
- `weakness_snapshots`
- `review_schedule`
- `evaluation_logs`

但还没有：

- 知识图谱式的 topic / concept / skill 图
- 非结构化 observations
- 可被 agent 主动写入的长期策略记忆

### 1.4 当前训练深度仍由 Go 固定控制

`server/internal/service/answer_service.go` 当前逻辑是：

- `turn_index >= max_turns` 就进入 `review_pending`
- 否则一定创建下一条 follow-up turn

也就是说：

- 模型今天没有“跳过追问”或“追加深挖一轮”的决策权
- 训练深度是 Go 侧固定 FSM，不是 agent 根据信号动态决定

---

## 2. 审核结论

原始 Deep Redesign Plan 的方向是对的，但按当前仓库现实，至少有 6 个地方需要先修口径，不然落地时会撞墙。

### 2.1 `knowledge_nodes` 的种子初始化不能全部放在 bootstrap

原始想法里写的是：

- 从 `question_templates.topic` 初始化 topic 层节点
- 从 `user_profile.tech_stacks` 自动生成 topic 层节点

这里要拆开：

- `question_templates.topic` 可以在 bootstrap 或后置 seed 里初始化
- `user_profile.tech_stacks` 不适合放在 bootstrap，因为它依赖用户运行期数据，不是 schema 初始化时就稳定存在

更合理的做法是：

1. bootstrap 只保证表存在
2. service 层提供 `EnsureKnowledgeSeeds(ctx)`：
   - 从 `question_templates` 保证基础 topic 节点
   - 从 `user_profile.tech_stacks` 做增量补种
3. 在保存画像、创建 session、或显式后台修复任务时触发这个 ensure

### 2.2 `depth_signal` 不应该挂在用户可见的 `EvaluationResult`

原始提案把：

```go
type EvaluationResult struct {
    // ... existing fields ...
    DepthSignal string `json:"depth_signal"`
}
```

和

```python
class EvaluateAnswerSideEffects(BaseModel):
    depth_signal: str = "normal"
```

同时存在。

这会带来两个问题：

- 控制信号和用户可见反馈混在一起
- Go 端会出现两份真相源

更合理的做法是：

- `EvaluationResult` 保持用户可见评估内容
- `depth_signal` 只存在于 `side_effects`

也就是：

- 用户看见的是 `headline / suggestion / followup_intent`
- Go 编排层看见的是 `side_effects.depth_signal`

### 2.3 `get_session_detail` 不能只返回 `TrainingSessionSummary`

原始提案里：

```text
GET /internal/session-detail/:id -> TrainingSessionSummary
```

但如果 `generate_review` 要按需回看历史 session，`TrainingSessionSummary` 太薄，只够列表页展示，不够 agent 做分析。

更合理的选择有两个：

1. 返回完整 `TrainingSession`
2. 新增一个专门给 agent 用的 `AgentSessionDetail`

建议用第 2 个，字段比完整 session 小，但必须至少覆盖：

- 基本 session 元信息
- turns
- evaluation 摘要
- review 摘要
- 绑定的 JD / project / prompt set 元信息

### 2.4 Sidecar 回调 Go 需要新增配置和内网保护

当前 sidecar 只有 LLM 配置，没有“如何反向调用 Go API”的配置。

如果要加：

- `search_repo_chunks`
- `get_session_detail`

就必须新增 sidecar -> Go 的配置，例如：

```env
PRACTICEHELPER_SIDECAR_SERVER_BASE_URL=http://127.0.0.1:8090
PRACTICEHELPER_INTERNAL_TOKEN=...
```

同时还要注意一件事：

- 当前 `server/internal/controller/router.go` 里的路由都是公开 HTTP 路由
- 如果直接把 `/internal/search-chunks`、`/internal/session-detail/:id` 挂进去而不做保护，它们会跟 `/api/*` 一样暴露出去

所以这类 internal 端点必须至少满足下面之一：

- 要求 `X-PracticeHelper-Internal-Token`
- 严格限制为 loopback 调用
- 或者单独起内部 router/group 并加统一 middleware

这不是“优化项”，是第一天就该补的边界。

### 2.5 流式链路不适合在 Phase 2 一起硬切到 agent loop

当前稳定性最好的是：

- 非流式：tool loop + single-shot fallback
- 流式：single-shot streaming

如果在同一阶段把 streaming 也切成 agent loop，会让风险成倍放大：

- 工具调用事件如何流式展示
- action tools 的 side_effects 如何在 stream 结束前收口
- 失败重试如何和 NDJSON 协议对齐

更稳的路线是：

- Phase 2 只升级非流式 agent loop
- 流式链路先继续保留现有 single-shot streaming
- 等非流式链路稳定后，再评估是否做 streaming agent loop

### 2.6 不应该让 LLM 直接写库，Go 仍然是唯一副作用入口

原始方案里的思路其实已经接近正确：

- `record_observation`
- `update_knowledge`
- `suggest_next_session`

都不是 sidecar 直写 DB，而是通过响应体 side effects 回到 Go 再持久化。

这个原则必须保留，而且要写死在设计里：

- sidecar 负责“产出结构化意图”
- Go 负责“验证 + 持久化 + 状态机决策”

不要把 knowledge graph 的写入权限直接下放给 sidecar 工具，否则调试、审计和幂等都会变差。

---

## 3. 目标架构

在上面这些修正成立之后，Deep Redesign 的目标态更适合定义为：

- 五层持久 memory
- 一层进程内 working memory
- 一个统一的 memory index / retrieval planner
- 两类工具
- 一个受约束的 ReAct loop

### 3.1 Memory 分层

这里不要做“一个万能 memory 表”。更稳的做法是：

- 真实数据按类型分表存
- 检索时再通过统一 index 和 retrieval planner 聚合

#### A. Identity / Profile Memory（SQLite，持久）

这层回答“这个用户是谁、目标是什么、长期偏好是什么”。

第一版直接复用和扩展现有结构，不单独造新大表：

- `user_profile`
- active job target 绑定
- 长期训练偏好（如果后续补）

特点：

- 体量小
- 高稳定
- 几乎所有任务都可能会用
- 不需要 embedding

加载策略：

- 直接并入 `recall_training_context`
- 不额外给它拆一个独立工具，避免工具数膨胀

#### B. Knowledge Graph（SQLite，持久）

新增三张表，表达“用户对知识域的掌握状态”：

```sql
CREATE TABLE knowledge_nodes (
    id TEXT PRIMARY KEY,
    parent_id TEXT NOT NULL DEFAULT '',
    label TEXT NOT NULL,
    node_type TEXT NOT NULL CHECK(node_type IN ('topic','concept','skill')),
    proficiency REAL NOT NULL DEFAULT 0,
    confidence REAL NOT NULL DEFAULT 0.5,
    hit_count INTEGER NOT NULL DEFAULT 0,
    last_assessed_at TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    UNIQUE(parent_id, label)
);

CREATE TABLE knowledge_edges (
    source_id TEXT NOT NULL,
    target_id TEXT NOT NULL,
    edge_type TEXT NOT NULL CHECK(edge_type IN ('contains','prerequisite','related')),
    created_at TEXT NOT NULL,
    PRIMARY KEY(source_id, target_id, edge_type)
);

CREATE TABLE knowledge_snapshots (
    id TEXT PRIMARY KEY,
    node_id TEXT NOT NULL,
    session_id TEXT NOT NULL,
    proficiency REAL NOT NULL,
    evidence TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL
);
```

语义约束：

- `proficiency`：`0-5`，表示 `unknown / heard / basics / apply / analyze / teach`
- `confidence`：`0-1`，表示系统对这次判断有多确定
- `knowledge_snapshots` 不替代当前表，只负责时间序列

图谱生命周期：

1. 表结构由 bootstrap 创建
2. `question_templates.topic` 生成基础 topic 节点
3. `user_profile.tech_stacks` 在 service 层做增量补种
4. `evaluate_answer` / `generate_review` 的 side effects 推动 proficiency 更新
5. 长期未评估节点做 confidence 衰减

#### C. Episodic Memory（Session Summary，SQLite，持久）

原始 session / turn 明细会越来越重，所以需要一层对 agent 友好的摘要记忆。

建议新增一张派生表：

```sql
CREATE TABLE session_memory_summaries (
    id TEXT PRIMARY KEY,
    session_id TEXT NOT NULL UNIQUE,
    mode TEXT NOT NULL,
    topic TEXT NOT NULL DEFAULT '',
    project_id TEXT NOT NULL DEFAULT '',
    job_target_id TEXT NOT NULL DEFAULT '',
    prompt_set_id TEXT NOT NULL DEFAULT '',
    summary TEXT NOT NULL,
    strengths_json TEXT NOT NULL DEFAULT '[]',
    gaps_json TEXT NOT NULL DEFAULT '[]',
    misconceptions_json TEXT NOT NULL DEFAULT '[]',
    growth_json TEXT NOT NULL DEFAULT '[]',
    recommended_focus_json TEXT NOT NULL DEFAULT '[]',
    salience REAL NOT NULL DEFAULT 0.5,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);
```

语义：

- 原始 `training_sessions / training_turns / review_cards` 仍然是 source of truth
- `session_memory_summaries` 是给 agent 做长期回看、相似召回和轻量检索的派生层

为什么要单独做这一层：

- 直接把历史 `turns` 全塞进 prompt 太重
- review agent 真正需要的是“这轮暴露了什么模式”，不是每轮原文逐字回放

建议生成时机：

- `generate_review` 成功后
- 或 review retry 成功后

#### D. Agent Observations（SQLite，持久）

图谱记录“知道什么”，observations 记录“为什么这样判断”和“对这个人该怎么追问”：

```sql
CREATE TABLE agent_observations (
    id TEXT PRIMARY KEY,
    session_id TEXT NOT NULL DEFAULT '',
    category TEXT NOT NULL CHECK(category IN ('pattern','misconception','growth','strategy_note')),
    content TEXT NOT NULL,
    tags_json TEXT NOT NULL DEFAULT '[]',
    relevance REAL NOT NULL DEFAULT 1.0,
    created_at TEXT NOT NULL,
    archived_at TEXT NOT NULL DEFAULT ''
);
```

推荐类别：

- `pattern`
- `misconception`
- `growth`
- `strategy_note`

约束：

- 活跃 observation 上限 `200`
- 超出后按 `relevance` 归档，不物理删除

#### E. Artifact Memory（现有表复用）

这层不是“人的长期记忆”，而是 agent 在工作时需要调用的上下文资产。

优先复用现有表：

- `project_profiles`
- `repo_chunks`
- `repo_chunks_fts`
- `job_target_analysis_runs`
- `question_templates`
- `evaluation_logs`
- `review_schedule`

特点：

- 任务相关性强
- scope 清晰
- 很多更适合结构化检索，不适合一上来就向量化

#### F. Working Memory（sidecar 进程内）

一次 `_run_agent_loop()` 调用内的临时记忆：

```python
@dataclass
class WorkingMemory:
    task_type: str
    session_id: str
    tool_results: dict[str, Any] = field(default_factory=dict)
    reasoning_trace: list[str] = field(default_factory=list)
    side_effects: dict[str, Any] = field(default_factory=dict)
```

它的生命周期只等于一次任务调用，不持久化。

### 3.2 Unified Memory Index

真正的 memory 仍然分表存，但建议补一个统一索引层，负责：

- 统一 scope
- 统一 salience / confidence / freshness
- 统一 retrieval 候选筛选

注意：

- `memory_index` 不是 source of truth
- 它只是 registry / lookup 层

建议表结构：

```sql
CREATE TABLE memory_index (
    id TEXT PRIMARY KEY,
    memory_type TEXT NOT NULL,
    scope_type TEXT NOT NULL, -- global / project / session / job_target
    scope_id TEXT NOT NULL DEFAULT '',
    topic TEXT NOT NULL DEFAULT '',
    project_id TEXT NOT NULL DEFAULT '',
    session_id TEXT NOT NULL DEFAULT '',
    job_target_id TEXT NOT NULL DEFAULT '',
    tags_json TEXT NOT NULL DEFAULT '[]',
    entities_json TEXT NOT NULL DEFAULT '[]',
    summary TEXT NOT NULL DEFAULT '',
    salience REAL NOT NULL DEFAULT 0.5,
    confidence REAL NOT NULL DEFAULT 0.5,
    freshness REAL NOT NULL DEFAULT 1.0,
    ref_table TEXT NOT NULL,
    ref_id TEXT NOT NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);
```

设计原则：

- 任何 memory 都能通过 `ref_table + ref_id` 回源
- 不在 index 里塞完整正文
- index 负责“先筛候选，再回源 materialize”

### 3.3 Scope 设计

不要把所有 memory 都按 `project_id` 或 `session_id` 粗暴分桶。更合理的是统一 scope 层级：

- `global`
  - profile
  - global knowledge graph
  - 长期 observations
- `project`
  - project-specific concepts
  - repo chunks
  - 项目局部 observations
- `session`
  - session summary
  - 当轮 observations
  - evaluation logs
- `job_target`
  - JD snapshot
  - 岗位导向策略记忆

这意味着：

- Knowledge graph 节点也建议支持 `scope_type + scope_id`
- 默认是 `global`
- 遇到项目特有术语或项目局部概念时，可以挂到 `project`

### 3.4 Retrieval Planner 与加载策略

memory 系统最重要的不是“存进去”，而是“什么时候拿什么”。

第一版建议采用三段式加载：

#### A. Always-on 预装载

每次任务先带最小核心上下文：

- request 本体
- profile snapshot（并入 `recall_training_context`）
- top weaknesses
- 相关 knowledge subgraph
- top observations
- 必要时的 JD snapshot / project summary

#### B. Planner 决定是否补查

如果预装载不够，再按需回调：

- 更多 repo chunks
- 历史 session detail
- 更多 observations
- 相关 session summaries

#### C. 预算裁剪

真正进 prompt 前再做一次压缩：

- top-k
- 去重
- 同类 observation 合并
- graph 限局部子图
- token budget 裁剪

结论：

- 不能一次把所有 memory 都塞进 prompt
- “预装载 + 按需回调混合” 是当前仓库最稳的路径

### 3.5 每个任务的默认 memory 装载

#### `generate_question`

优先读取：

1. 当前 request + profile snapshot
2. top weaknesses
3. 当前 topic 的 knowledge subgraph
4. top observations
5. project 模式下的初始 repo chunks

#### `evaluate_answer`

优先读取：

1. 当前 question / answer / expected points
2. 相关 knowledge nodes
3. top observations
4. 必要时 project chunks
5. 少量相近 session summaries（不是原始 turns）

#### `generate_review`

优先读取：

1. 当前 session summary 草稿上下文
2. knowledge graph 的前后变化
3. 相关 observations
4. 最近 2-3 条相近 session summaries
5. 当前 weakness profile

### 3.6 Embedding / Rerank 策略

这里不建议第一天就“全 memory 全向量化”。

更合理的顺序是：

#### 第一批适合 embedding 的对象

- `agent_observations`
- `session_memory_summaries`

因为这两类数据：

- 自然语言密度高
- 语义表达多样
- 很适合做相似召回

#### 第一批不建议急着 embedding 的对象

- `knowledge_nodes`
- `knowledge_edges`
- `weakness_tags`
- `repo_chunks`

原因：

- graph 更适合结构化关系检索
- weakness 本来就高度结构化
- repo chunks 当前已有 SQLite FTS5，先不要还没证明不够就上 embedding

#### Rerank 的建议

第一版先做规则重排，不上重模型 reranker：

- scope match
- topic match
- project match
- job target match
- recency
- salience
- confidence

如果后续证明 observation / session summary 的召回还不够，再加：

- `embedding similarity`
- hybrid score

但不建议第一版就上 cross-encoder 或 LLM reranker。

### 3.7 工具体系

工具分两类：

#### A. 回忆工具

分成两种来源：

- 预装载：随请求体一起带进 sidecar
- 按需回调：sidecar 再回调 Go

建议工具注册集中到新文件：

- `sidecar/app/agent_tools.py`

第一版回忆工具建议如下：

| Tool | 来源 | 方式 | 用途 |
|---|---|---|---|
| `recall_training_context` | 请求体 | 预装载 | 当前任务上下文 |
| `recall_weakness_profile` | `agent_context` | 预装载 | Top-N 弱项 |
| `recall_knowledge_graph` | `agent_context` | 预装载 | 相关 topic 子图 |
| `recall_observations` | `agent_context` | 预装载 | 相关 observations |
| `recall_session_summaries` | `agent_context` | 预装载 | 最近相近 session 摘要 |
| `search_repo_chunks` | Go internal API | 按需回调 | 追加搜索项目片段 |
| `get_session_detail` | Go internal API | 按需回调 | 获取历史 session 详情 |

#### B. 行动工具

这些工具不直接写 DB，只写入 `WorkingMemory.side_effects`，最后一起回传 Go：

| Tool | 输出位置 | 用途 |
|---|---|---|
| `record_observation` | `side_effects.observations` | 记录观察 |
| `update_knowledge` | `side_effects.knowledge_updates` | 更新知识图谱 |
| `suggest_next_session` | `side_effects.recommended_next` | 建议下一轮训练 |
| `set_depth_signal` | `side_effects.depth_signal` | 控制追问深度 |

### 3.8 任务与工具的映射

建议第一版这样分配：

| 任务 | 回忆工具 | 行动工具 |
|---|---|---|
| `generate_question` | `recall_training_context` `recall_weakness_profile` `recall_knowledge_graph` `recall_observations` `search_repo_chunks` | — |
| `evaluate_answer` | `recall_training_context` `recall_knowledge_graph` `recall_observations` `search_repo_chunks` | `record_observation` `update_knowledge` `set_depth_signal` |
| `generate_review` | `recall_training_context` `recall_weakness_profile` `recall_knowledge_graph` `recall_observations` `recall_session_summaries` `get_session_detail` | `record_observation` `update_knowledge` `suggest_next_session` |

约束：

- 每个任务第一版最多 `6` 个工具
- 超过这个数之前，先压缩职责，不要继续膨胀

---

## 4. 数据契约调整

### 4.1 Go domain types

`server/internal/domain/types.go` 需要补这些类型：

- `KnowledgeNode`
- `KnowledgeEdge`
- `KnowledgeSubgraph`
- `AgentObservation`
- `KnowledgeUpdate`
- `AgentContext`

其中 `AgentContext` 建议至少包括：

- `KnowledgeSubgraph`
- `Observations`
- `WeaknessProfile`
- `SessionHistory`

这里的 `SessionHistory` 建议直接用现有 `TrainingSessionSummary` 或一个更精简的新类型，不要引用仓库里并不存在的 `SessionSummary`。

### 4.2 Sidecar 请求体增加 `agent_context`

第一版建议只在这三个任务上加：

- `GenerateQuestionRequest`
- `EvaluateAnswerRequest`
- `GenerateReviewRequest`

也就是 Go 预装载一份相关上下文给 sidecar，而不是让 sidecar 第一步就全靠回调。

### 4.3 Sidecar 响应体增加 `side_effects`

建议增加任务专属 side effects，而不是把所有控制信号塞进用户可见 result：

```python
class EvaluateAnswerSideEffects(BaseModel):
    observations: list[AgentObservation] = []
    knowledge_updates: list[KnowledgeUpdate] = []
    depth_signal: Literal["skip_followup", "extend", "normal"] = "normal"


class EvaluateAnswerEnvelope(BaseModel):
    result: EvaluationResult
    side_effects: EvaluateAnswerSideEffects = EvaluateAnswerSideEffects()
    raw_output: str = ""
```

`GenerateReviewEnvelope` 也类似：

- `observations`
- `knowledge_updates`
- `recommended_next`

---

## 5. Go 后端集成方式

### 5.1 AgentContext 预装载

`answer_service.go` / `session_creation_service.go` / `review_service.go` 需要新增一层：

- `GetAgentContext(ctx, session/request)` 或等价 helper

它负责：

- 取 profile snapshot
- 取相关 weakness
- 取相关知识子图
- 取 observations
- 取必要的 session summaries / session history

### 5.2 Side effects 统一由 Go 落库

收到 sidecar 响应后：

- `observations` -> repo/service 持久化
- `knowledge_updates` -> repo/service upsert
- `recommended_next` -> 写入 `ReviewCard` 或保留现有推荐入口
- `depth_signal` -> 影响 answer FSM

### 5.3 Dynamic Depth 的 Go 语义

`depth_signal` 第一版建议这样处理：

- `skip_followup`
  - 当前回答质量足够高，直接进入 `review_pending`
- `extend`
  - 仅当未超过系统允许的上限时追加一轮
- `normal`
  - 保持当前 FSM

注意这里不要让模型无限加轮次。Go 端仍然需要硬上限，例如：

- 初始 `max_turns <= 5`
- `extend` 后的绝对上限仍然限制在 `6` 或 `7`

---

## 6. Sidecar -> Go 回调设计

### 6.1 新增 Go internal 端点

建议新增：

- `GET /internal/search-chunks?project_id=&query=&limit=`
- `GET /internal/session-detail/:id`

但必须带 internal 保护：

- `X-PracticeHelper-Internal-Token`

### 6.2 新增 `go_client.py`

新增：

- `sidecar/app/go_client.py`

职责：

- 读取 `PRACTICEHELPER_SIDECAR_SERVER_BASE_URL`
- 读取 `PRACTICEHELPER_INTERNAL_TOKEN`
- 封装 sidecar -> Go 的内部 HTTP 调用

### 6.3 为什么仍然需要“预装载 + 按需回调混合”

这个决策和当前仓库是兼容的：

- 预装载可以降低 agent 第一步就去回调的频率
- 按需回调可以避免一次把所有历史和 chunks 塞进 prompt

这是比“全预装载”或“全远程工具化”更稳的折中。

---

## 7. Agent Loop 升级方案

### 7.1 Constrained ReAct Loop

非流式任务里，把 `_run_tool_loop` 升级为 `_run_agent_loop`：

1. 接收任务与 `agent_context`
2. 初始化 `WorkingMemory`
3. 执行 agent loop，最多 `8` 轮
4. 模型每轮可选择：
   - 调用工具
   - 输出最终 JSON
5. JSON 验证失败时，把校验反馈追加回消息，继续内循环
6. 轮次耗尽后，退回现有 single-shot fallback
7. 从 working memory 收集 side effects，拼到 envelope

### 7.2 Phase 2 不动 streaming loop

为了保留当前稳定链路：

- 非流式先切 `_run_agent_loop`
- 流式仍保留 `_stream_single_shot_task`

如果后续要做 stream agent loop，建议单开一轮设计，不和本次计划绑在一起。

### 7.3 LangGraph 收敛

当前 5 条图的收敛建议是：

1. `analyze_repo`
   - 保持现状：`collect -> rank -> summarize`
2. `analyze_job_target`
   - 保持现状：单节点或轻图
3. `agent_task`
   - `prepare_context -> run_agent_loop -> collect_side_effects`

`generate_question / evaluate_answer / generate_review` 共享统一 `agent_task` 主循环。

---

## 8. 分阶段实施

### Phase 1：Knowledge Graph Foundation + Observations

目标：先把长期记忆基础设施补齐。

涉及文件：

- `server/internal/infra/sqlite/bootstrap.go`
- `server/internal/domain/types.go`
- 新建 `server/internal/repo/knowledge_repo.go`
- 新建 `server/internal/repo/observation_repo.go`
- 新建 `server/internal/repo/session_memory_repo.go`
- 新建 `server/internal/repo/memory_index_repo.go`
- `server/internal/service/*`
- `sidecar/app/schemas.py`

交付物：

- 表结构 ready
- Go 可读写 knowledge graph / observations / session summaries / memory index
- 基础 topic seed ready

### Phase 2：非流式 Agent Loop + 工具注册表

目标：把 sidecar 从“固定只读工具 loop”升级成“带 side effects 的 constrained ReAct loop”，同时接上 retrieval planner。

涉及文件：

- 新建 `sidecar/app/agent_tools.py`
- `sidecar/app/agent_runtime.py`
- `sidecar/app/runtime_prompts.py`
- `sidecar/app/langgraph_flows.py`
- `server/internal/service/answer_service.go`
- `server/internal/service/session_creation_service.go`
- `server/internal/service/review_service.go`
- `server/internal/sidecar/client.go`
- `server/internal/service/*` 中的 retrieval planner / materialize helper

交付物：

- 非流式 `generate_question / evaluate_answer / generate_review` 接入 agent loop
- `record_observation` / `update_knowledge` 能通过 side effects 回写 Go
- `session_memory_summaries` 开始参与 review / answer 的默认装载
- streaming 仍保持现状，不一起动

### Phase 3：Go Callback + Dynamic Depth

目标：让 agent 能按需搜索、按信号控制追问深度。

涉及文件：

- 新建 `sidecar/app/go_client.py`
- `server/internal/controller/router.go`
- 新建 `server/internal/controller/internal_controller.go`
- `sidecar/app/agent_tools.py`
- `server/internal/service/answer_service.go`
- prompt 模板

交付物：

- `search_repo_chunks` / `get_session_detail` 可用
- `depth_signal` 接进 Go FSM
- project 模式的上下文选择不再完全依赖预选 chunks

### Phase 4：Progressive Autonomy

目标：逐步放权，但仍由 Go 保持最终边界，同时只在高价值 memory 上增量引入 embedding / rerank。

涉及方向：

- `suggest_next_session` 深化
- prerequisite edge 推断
- review 中基于知识图谱给出学习路径建议
- 为 `agent_observations` / `session_memory_summaries` 增加 embedding 与 hybrid rerank

交付物：

- 推荐下一轮训练时能解释“为什么是这一轮”
- graph 不只是记录热度，还能表达学习路径
- embedding 只覆盖 observations / session summaries，不扩到 graph 和 repo chunks

---

## 9. 验证方式

### Phase 1

- `cd server && GOCACHE=/tmp/go-build go test -tags sqlite_fts5 ./...`
- 手动检查 `knowledge_nodes` / `agent_observations` / `session_memory_summaries` / `memory_index` 表已创建
- 手动检查基础 topic seed 是否存在

### Phase 2

- `cd sidecar && uv run pytest`
- 启动三进程，创建 session -> 提交答案
- 检查 `agent_observations` 有写入
- 检查 `knowledge_nodes.proficiency` 有更新
- 检查 review 结束后 `session_memory_summaries` 有生成
- 检查 retrieval planner 默认优先使用 summary 而不是回放全量 turns

### Phase 3

- sidecar 日志确认 callback 已调用 Go internal API
- project 模式下确认 agent 会按需搜新的 repo chunks
- 高分答案触发 `skip_followup`
- 低质量答案在允许范围内触发 `extend`

### Phase 4

- 完整跑一轮 session
- 检查 `recommended_next` 是否有知识图谱依据
- 检查 `knowledge_edges` 是否开始沉淀 `prerequisite`
- 检查 observations / session summaries 的 embedding 召回只作用于对应 memory，不影响 graph / repo chunk 的主检索路径

---

## 10. 非目标

这份 redesign plan 明确不做：

- 把 `analyze_repo` 也改成通用 agent loop
- 让 sidecar 直接写数据库
- 在 Phase 2 同时重写 streaming loop
- 为了“更像 agent”继续无限加工具
- 在图谱还没跑稳前就引入全量向量库或把所有 memory 都 embedding 化
- 在还没证明 FTS5 不够前，先把 `repo_chunks` 检索切到 embedding

---

## 11. 结论

这条 Deep Redesign 线路是值得做的，但必须按当前仓库现实改成“渐进升级”，而不是一次性推翻现有链路。

最关键的收口原则有 4 条：

1. 先补记忆基础设施，再补 agent loop
2. sidecar 只产出 side effects，Go 仍然是唯一副作用入口
3. 非流式先升级，流式后置
4. internal callback 从第一天就带鉴权边界

再补 2 条 memory 侧原则：

5. 真实 memory 分表存，统一 index 只做 registry，不做 source of truth
6. embedding 先上 observations / session summaries，不先上 graph / weakness / repo chunks

按这个版本推进，才是在当前 PracticeHelper 仓库里可落地、可验证、可回滚的 Agent Deep Redesign。
