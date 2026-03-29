# Agent Deep Redesign Plan

> 状态：已按 2026-03-26 当前工作区重写为结论版。
> 当前判断：Phase A / Phase B 已完成；Phase C 已接第一版 typed command
> path，但还没收口；Phase D 还没开始。

## 1. 这份文档只回答什么

它只回答 4 个问题：

1. 当前 sidecar agent 已经做到哪了
2. 在这个仓库里，“成熟 agent”到底指什么
3. Go 和 sidecar 的边界怎么守住
4. Phase C 还该继续补什么，Phase D 什么时候才值得开始

这不是通用 agent 平台方案，也不是“让模型直接接管数据库”的计划。

另外，这份文档不负责改写产品形态：它不能成为把 PracticeHelper 推成 chat-first 产品、全局教练 chat 主入口或 Workspace 级大重构的理由。当前产品主轴仍然是训练页作为主交易面。

## 2. 当前已经有什么

### 2.1 Runtime 基线已经成立

当前 `sidecar/app/runtime/` 已经具备：

- 非流式 agent loop
- 流式 agent loop
- JSON / 语义校验
- single-shot fallback
- stream fallback
- `runtime_trace`

换句话说，现在的 sidecar 已经不是 prompt wrapper，而是一个受约束的单 agent runtime。

### 2.2 工具分层已经成立

当前工具面大致分三类：

- flow-specific 只读工具
- memory / retrieval 工具
- 动作工具

动作工具又分两种：

- `side_effects`：`record_observation`、`update_knowledge`、
  `set_depth_signal`、`suggest_next_session`
- typed command：`transition_session`、`upsert_review_path`

### 2.3 长期记忆和检索已经接上第一版

当前已经成立的事实：

- `memory_index` 负责 observation / session summary 的统一候选池
- memory retrieval 已支持规则重排、向量召回和 optional rerank
- repo chunk 已支持 `Qdrant vector recall + optional rerank + FTS5 fallback`
- review 已能持久化 `retrieval_trace`

### 2.4 Go 侧仍然保留最终裁决权

当前关键边界已经成立：

- sidecar 可以提出结构化意图
- Go 决定最终状态迁移和持久化
- typed command 通过 `/internal/agent-commands` 回调 Go
- `command_results` 回流后再参与最终校验

这是当前方案最重要的安全边界，不应该被打破。

## 3. 还没收口的点

当前最大的问题不是“没有 agent”，而是“第一版已经接上，但还不够稳”：

- typed command 覆盖面还窄
- retrieval 的证据绑定还不够清楚
- context compaction 还需要更稳定的策略
- 恢复语义、失败分类和回归验证还要继续补
- 多 agent 仍停留在设计层，不该提前落到训练热路径

## 4. 这个仓库里的“成熟 agent”定义

成熟 agent 在这里不是“更自由”，而是下面 6 层更完整：

| 层 | 要解决什么 | 当前判断 |
|:---|:-----------|:---------|
| Task Surface | 任务入口和请求级上下文绑定 | 已成立 |
| Context Engine | 上下文预装载、压缩、检索和补查 | 已成立第一版，仍在收口 |
| Planning Engine | 最小策略选择与流程编排 | 已成立，但仍保持薄壳 |
| Action Engine | `side_effects` + typed command 双轨动作 | 已成立第一版，仍在收口 |
| Validation / Recovery | JSON / 语义校验、重试、fallback、恢复 | 已成立第一版，仍在收口 |
| Observability | trace、prompt 元信息、raw output、检索轨迹 | 已成立 |

## 5. 边界怎么划

| 角色 | 负责什么 | 不负责什么 |
|:-----|:---------|:-----------|
| Go | 产品边界、状态机、持久化、审计、恢复 | 不做模型推理 |
| sidecar | 上下文理解、工具调用、结构化输出、runtime 校验 | 不直写数据库 |
| `side_effects` | cheap / local / batch-friendly 的结构化意图 | 不做关键状态裁决 |
| typed command | 关键动作向 Go 申请结构化裁决 | 不绕过 Go 直接落库 |

一句话：**模型可以建议，Go 负责拍板。**

## 6. Phase C 现在该继续补什么

### 6.1 强化 typed command path

当前第一版已经接上 `transition_session` 和 `upsert_review_path`。
下一步重点不是“再发明更多命令”，而是先把这条路径做稳：

- 结果校验更严格
- 失败 / 延迟 / 拒绝语义更清楚
- trace 和前端展示更一致

### 6.2 强化 retrieval 和 context 利用

下一步重点：

- 检索命中为什么命中，要更容易解释
- review / 推荐引用的证据要更可信
- context compaction 要继续减少无效上下文

### 6.3 强化 validation / recovery

下一步重点：

- 失败分类更清楚
- fallback 何时触发更稳定
- stream / 非 stream 的行为尽量一致

### 6.4 强化 observability

当前已经有 `prompt_set_id`、`prompt_hash`、`raw_output`、
`runtime_trace`、`retrieval_trace`。接下来更重要的是：

- 让这些信息更容易对齐到一次真实训练请求
- 让 command path、persist 阶段和 review 结果更容易串起来看

## 7. Phase D 什么时候才值得开始

多 agent 不是不能做，但必须满足这些前提：

- 单 agent 主路径已经稳定
- typed command 契约已经够清楚
- shared artifact 结构先定义稳定
- 目标任务真的是高价值长任务，而不是当前训练热路径

如果这些条件没满足，先做多 agent 只会把主线搞散。

## 8. 验收方式

继续推进这条线时，至少要同时看这几层：

- sidecar runtime 测试
- server 的 `internal/service/controller` 测试
- web 的 streaming / client 测试
- 必要的 live 主链路验证

不要只看某一层单独通过就说这条线收口了。

## 9. 明确不做

- 不把 Go 的状态机和持久化边界交给模型
- 不把 sidecar 扩成通用 agent 平台
- 不为了“更像 agent”提前铺多 agent
- 不把数据库直接暴露给 LLM
