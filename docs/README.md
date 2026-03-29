# Docs Guide

> 这套文档现在只做一件事：让人能在几分钟内搞清楚
> “项目现在是什么、当前主线是什么、细节该去哪找”。

## 先看哪几份

如果你只想快速恢复当前开发上下文，先看这 3 份：

1. [`current/VISION.md`](./current/VISION.md)：这个产品现在到底是什么。
2. [`current/PLAN.md`](./current/PLAN.md)：当前主线到底在做什么。
3. [`current/ARCHITECTURE.md`](./current/ARCHITECTURE.md)：代码和进程边界怎么分。

如果你还想确认“半年后值得长成什么”，再补一份：

4. [`current/NORTH_STAR.md`](./current/NORTH_STAR.md)：长期 6-12 个月演化锚点。

## 先信谁

默认优先级：

1. `current/`
2. 仓库根 `README.md`
3. `plans/`
4. `records/`

解释很简单：

- `current/` 是当前事实主链，冲突时默认先信它。
- 根 `README.md` 负责项目介绍、启动方式和文档导航，不单独定义当前主线。
- `plans/` 是专项执行文档，只回答某条线怎么推进。
- `records/` 是历史设计记录，用来追溯，不用来判断当前待办。

## 每类文档管什么

| 目录 | 用途 | 什么时候看 |
|:-----|:-----|:-----------|
| `current/` | 当前产品、当前主线、当前架构 | 日常开发、恢复上下文、判断边界 |
| `plans/` | 某条专项怎么继续推进 | 你准备继续那条专项时 |
| `records/` | 已完成阶段当时为什么这么做 | 你要追历史决策时 |

## current

| 文档 | 作用 |
|:-----|:-----|
| [`current/VISION.md`](./current/VISION.md) | 产品定位、已形成价值闭环、当前产品形态约束 |
| [`current/NORTH_STAR.md`](./current/NORTH_STAR.md) | 长期 6-12 个月演化锚点，不回答本周待办 |
| [`current/ROADMAP.md`](./current/ROADMAP.md) | 阶段顺序、为什么按这个顺序做、哪些明确后置 |
| [`current/PLAN.md`](./current/PLAN.md) | 当前执行主线、这轮收口什么、什么不该扩出去 |
| [`current/ARCHITECTURE.md`](./current/ARCHITECTURE.md) | 三端边界、关键数据、状态机和接口面 |

## plans

| 文档 | 作用 |
|:-----|:-----|
| [`plans/PRODUCT_UPGRADE_PLAN.md`](./plans/PRODUCT_UPGRADE_PLAN.md) | 阶段 C 的产品升级清单与当前进度 |
| [`plans/ARCHITECTURE_CONVERGENCE_PLAN.md`](./plans/ARCHITECTURE_CONVERGENCE_PLAN.md) | 当前工程收口专项 |
| [`plans/AGENT_DEEP_REDESIGN_PLAN.md`](./plans/AGENT_DEEP_REDESIGN_PLAN.md) | sidecar agent runtime 深改专项 |

## records

| 文档 | 作用 |
|:-----|:-----|
| [`records/JD_TRAINING_STAGE_B.md`](./records/JD_TRAINING_STAGE_B.md) | 阶段 B 的方案与验收口径 |
| [`records/ANSWER_FEEDBACK_UX_V2.md`](./records/ANSWER_FEEDBACK_UX_V2.md) | 阶段 A 的答题反馈 V2 记录 |

## 不建议的读法

- 不要一上来先读最长的计划文档。
- 不要把根 `README.md` 当成当前主线的最高来源。
- 不要把 `records/` 当成当前事实。
- 如果只是要继续开发当前主线，没必要先翻完全部 `docs/`。
