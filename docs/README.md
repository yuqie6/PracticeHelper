# Docs Guide

> 这份索引用来回答两个问题：
> 1. `docs/` 里每份文档该去哪里找
> 2. 当几份文档说法不完全一样时，先信哪一份

## 目录结构

```text
docs/
├── README.md
├── current/   # 当前事实主链
├── plans/     # 专项计划与执行方案
└── records/   # 已完成阶段的设计记录
```

## 怎么读

优先级默认这样看：

1. `current/`
2. `plans/`
3. `records/`

含义是：

- `current/` 讲的是当前仓库事实，是默认入口。
- `plans/` 讲的是某条专项主线怎么推进，不单独覆盖当前事实。
- `records/` 讲的是当时为什么这么做、边界是什么，主要用于回溯。

## current

| 文档 | 作用 |
|:-----|:-----|
| [`current/VISION.md`](./current/VISION.md) | 产品方向锚点 |
| [`current/PRD.md`](./current/PRD.md) | 当前产品边界 |
| [`current/ROADMAP.md`](./current/ROADMAP.md) | 阶段顺序与推进原则 |
| [`current/PLAN.md`](./current/PLAN.md) | 当前主线与近期任务 |
| [`current/ARCHITECTURE.md`](./current/ARCHITECTURE.md) | 当前技术事实、状态机、Schema、API |

## plans

| 文档 | 作用 |
|:-----|:-----|
| [`plans/PRODUCT_UPGRADE_PLAN.md`](./plans/PRODUCT_UPGRADE_PLAN.md) | 阶段 C 产品升级清单 |
| [`plans/ARCHITECTURE_CONVERGENCE_PLAN.md`](./plans/ARCHITECTURE_CONVERGENCE_PLAN.md) | 当前工程收口专项 |
| [`plans/AGENT_DEEP_REDESIGN_PLAN.md`](./plans/AGENT_DEEP_REDESIGN_PLAN.md) | sidecar agent runtime 深改专项 |

## records

| 文档 | 作用 |
|:-----|:-----|
| [`records/JD_TRAINING_STAGE_B.md`](./records/JD_TRAINING_STAGE_B.md) | 阶段 B 的设计记录与验收口径 |
| [`records/ANSWER_FEEDBACK_UX_V2.md`](./records/ANSWER_FEEDBACK_UX_V2.md) | 阶段 A 的答题反馈 V2 设计记录 |

## 最短阅读路径

如果只是想快速恢复项目上下文，按这个顺序看：

1. [`current/VISION.md`](./current/VISION.md)
2. [`current/PRD.md`](./current/PRD.md)
3. [`current/ROADMAP.md`](./current/ROADMAP.md)
4. [`current/PLAN.md`](./current/PLAN.md)
5. [`current/ARCHITECTURE.md`](./current/ARCHITECTURE.md)
