# 答题反馈 V2

## 现状问题

基于当前代码，有 4 个核心体感问题：

| # | 问题 | 根因 |
|---|------|------|
| 1 | 用户不知道系统是否卡住 | `useProgressSteps` 按 1200ms 定时推进步骤，与后端真实阶段无关 |
| 2 | 反馈像字段列表，不像教练判断 | `SessionView` 右栏只平铺 score → strengths → gaps，无结论、无证据、无改进建议 |
| 3 | 追问缺乏意图说明 | `EvaluationResult.followup_question` 是裸问题，无 intent 字段 |
| 4 | 复盘是终点而非入口 | `ReviewView` 有"继续训练"链接但跳回通用 `/train`，无具体推荐 |

---

## 目标

把训练流程从"能用"升级到"用户知道发生了什么、为什么这么判、下一步该怎么做"。

不做的事：多轮自由对话、复杂 agent 链路、大规模架构重构。

---

## 改动计划

按实现难度和用户感知收益排序，分三层推进。

### Layer 1：消除卡住感（prompt 改动 + 前端调整）

**1.1 真实阶段驱动的处理态**

当前 `StreamEvent.type` 已有 `phase / context / reasoning / content`。问题不在后端缺事件，在于前端没用它们。

改动：
- 前端：`ProgressPanel` 不再按定时器推进，改为消费 `StreamEvent` 中的 `phase` 值映射到用户可读步骤
- 后端（可选增强）：在流式输出中补充 `answer_saved` / `evaluation_started` 等状态事件

`StreamEvent` 扩展：

```go
// 新增 type=status 事件，name 取值：
// answer_saved, evaluation_started, feedback_ready,
// followup_ready, review_started, review_saved
```

前端映射：

```
answer_saved       → "已保存回答"
evaluation_started → "正在评估"
feedback_ready     → "反馈已生成"
followup_ready     → "追问已生成"
review_started     → "正在生成复盘"
review_saved       → "复盘已完成"
```

**1.2 提交确认与草稿保留**

当前 `mutation.onError` 清空了错误提示但未保留草稿。`onSuccess` 清空了 `answer.value`。

改动：
- 提交后立即显示"已收到回答"确认态，将已提交内容固定为只读区块
- 失败时保留 `answer.value`，不清空

**1.3 复盘前收口**

当前 `SessionView` 在 session 状态变为 `completed` 后直接 `router.push` 到 ReviewView，没有过渡。

改动：在跳转前短暂展示收口卡片："本轮问答已完成，正在整理复盘"。

---

### Layer 2：反馈可理解性（prompt schema 扩展 + 前端重排）

**2.1 EvaluationResult 扩展**

```python
class EvaluationResult(BaseModel):
    score: float
    score_breakdown: dict[str, float]
    strengths: list[str]
    gaps: list[str]
    # --- V2 新增 ---
    headline: str = ""           # 一句话结论："答到可过线但深度不足"
    suggestion: str = ""         # 改进示例："可以补充失败场景的兜底策略"
    followup_intent: str = ""    # 追问意图："确认你是否理解方案代价"
    followup_question: str = ""
    followup_expected_points: list[str]
    weakness_hits: list[WeaknessHit]
```

这三个字段（`headline`, `suggestion`, `followup_intent`）只需要改 sidecar prompt + schema，不涉及架构变动。

**2.2 反馈卡重排**

当前 SessionView 右栏信息层级：`score → strengths → gaps`。

改为：

```
headline（一句话结论）
  → strengths（亮点，默认折叠只显示前 2 条）
  → gaps（缺口，默认折叠只显示前 2 条）
  → suggestion（改进建议）
  → score_breakdown（分项得分，折叠）
```

**2.3 追问意图展示**

追问卡片新增 `followup_intent` 展示区，与主问题卡片视觉区分。用户能看到"为什么追问你这个"。

---

### Layer 3：训练闭环（schema 扩展 + 前端新区块）

**3.1 ReviewCard 扩展**

```python
class ReviewCard(BaseModel):
    overall: str
    highlights: list[str]
    gaps: list[str]
    suggested_topics: list[str]
    next_training_focus: list[str]
    score_breakdown: dict[str, float]
    # --- V2 新增 ---
    top_fix: str = ""                    # "最该优先修正：trade-off 表述不够具体"
    top_fix_reason: str = ""             # "这会影响项目题的说服力"
    recommended_next: NextSession | None = None  # 具体推荐
```

```python
class NextSession(BaseModel):
    mode: str        # "basics" | "project"
    topic: str = ""
    project_id: str = ""
    reason: str = ""
```

**3.2 ReviewView 改造**

复盘页顶部改为动作导向：

```
top_fix + top_fix_reason
  → highlights / gaps（现有内容）
  → recommended_next（推荐下一轮 + 原因）
  → "立即开始下一轮"按钮（携带推荐参数跳转到 /train）
```

当前"继续训练"是裸链接跳 `/train`，改为携带 `mode` / `topic` / `project_id` query params。

---

## 不在 V2 范围内

以下能力有价值但实现成本高，建议作为独立后续任务：

| 能力 | 不做的原因 |
|------|-----------|
| 证据绑定（引用用户原文 + repo chunk） | 需要 LLM 做精确引用抽取，prompt 复杂度和解析可靠性都是问题，当前 sidecar 架构不直接支持 |
| 作答结构 checklist | 是锦上添花，不解决核心体感问题 |
| 移动端 tab 布局 | 当前用户场景以桌面端为主 |
| 考察重点 / 回答指南卡片 | 需要出题阶段额外产出结构化元数据，改动链路较长 |

---

## 完成标准

- Layer 1 完成：提交后能看到真实阶段推进，不再依赖定时器假进度
- Layer 2 完成：每次反馈有一句话结论 + 改进建议，追问有意图说明
- Layer 3 完成：复盘页有具体的下一轮推荐，用户可一键开始
