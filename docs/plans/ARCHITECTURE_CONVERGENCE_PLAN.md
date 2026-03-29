# Architecture Convergence Plan

> 状态：持续收口中（Phase A-D 已落地，Phase E 还在收尾验证）
> 更新时间：2026-03-23
> 目标：在不改公开接口和产品主行为的前提下，收口 PracticeHelper
> 当前工程复杂度，解决中心文件过大、三端契约分散、重复 helper 和低风险
> 工程卫生问题。

---

## 1. 这份文档解决什么

这不是产品路线图，也不是 agent 深改方案。

它也不是仓库当前事实的最高来源。当前事实仍以 `docs/current/*`
为准；这份文档只回答“这轮工程收口专项怎么拆、做到哪了”。

这份文档只回答一个问题：

> 当前仓库已经能跑、能测，但开始变胖了，接下来要怎样有秩序地收口。

本轮只做 5 类工作：

1. 拆分后端中心文件
2. 拆分前端 API 类型与重复 helper
3. 拆分 sidecar schema 聚合文件
4. 清理低风险死代码、无效赋值和格式问题
5. 把主线文档和当前真实实现重新对齐

---

## 2. 硬边界

为了避免“边收口边扩范围”，本轮固定遵守这些边界：

- 不改公开 REST 路径
- 不改 NDJSON 流式事件字段名
- 不改现有 JSON 字段名
- 不新增产品功能
- 允许为了工程收口补低风险内部接口、内部字段和幂等索引，但不改变对外契约
- 不引入 codegen、共享 schema 生成器或新的基础设施
- 只允许纳入和本轮收口直接相关的低风险小修复

---

## 3. 执行顺序

### Phase A：文档与入口收口

- 当前状态：已落地
- 新增本计划文档，作为本轮唯一执行入口
- 在 `docs/current/PLAN.md` 标注本轮工程收口走独立计划
- 在 `docs/current/ARCHITECTURE.md` 增加交叉引用，但不重写现有架构正文

### Phase B：Go backend 中心文件拆分

- 当前状态：已落主干
- `controller/router.go`
  收回到“router 初始化 + 中间件 + route group 注册”角色
- 外部 handler 按领域拆分：
  - profile/dashboard
  - job target
  - project/import
  - prompt
  - session/review/export
- `domain/types.go` 按领域拆分，保留现有 JSON 字段和状态语义不变
- `service_test.go` 按主题拆分，避免一个测试文件跨多个服务面同时膨胀

### Phase C：web 收口

- 当前状态：已落主干
- `web/src/api/client.ts` 拆成“请求执行层 + 契约类型层”
- 保留 `client.ts` 作为聚合出口，避免页面侧大面积改 import
- 抽出跨页面重复的 session helper，例如：
  - `formatSessionName`
  - `buildSessionTarget`
- 页面只做最小逻辑收口，不做视觉重排

### Phase D：sidecar schema 收口

- 当前状态：已落主干
- 将 `sidecar/app/schemas.py` 拆成多个领域模块
- `schemas.py` 保留聚合 re-export，保持调用侧兼容
- 优先收口跨三端都重复出现的训练、review、job target、prompt set
  等模型

### Phase E：卫生与验证

- 当前状态：收尾中
- 修复 lint 已暴露的问题
- 清理拆分过程中出现的未使用赋值、死分支和失效 helper
- 跑 lint / test，确认公开行为不变

---

## 4. 重点收口对象

### 4.1 后端

- `server/internal/controller/router.go`
- `server/internal/domain/types.go`
- `server/internal/service/service_test.go`

### 4.2 前端

- `web/src/api/client.ts`
- Home / Train 等页面中的重复 session helper
- 页面内重复错误提示的映射和轻量辅助逻辑

### 4.3 Sidecar

- `sidecar/app/schemas.py`

---

## 5. 验收标准

完成本轮收口时，需要同时满足：

- 公开 API、JSON 字段、流式事件字段保持不变
- 后端中心文件不再跨多个领域同时担责
- 前端 API 类型不再全部堆在单个 `client.ts`
- sidecar schema 不再全部堆在单个 `schemas.py`
- lint / test 至少有一轮收尾验证
- 文档能明确解释“为什么现在做这轮收口、做到哪算完成”

---

## 6. 不在本轮里做

以下方向正确，但不属于这次收口：

- 新产品功能
- DB migration 或表结构调整
- 多 agent 编排升级
- 全仓跨语言 schema 自动生成
- UI 重设计或体验大改
- worker / queue / 异步基础设施升级
