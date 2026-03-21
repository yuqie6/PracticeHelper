# PracticeHelper 产品全面升级计划

## 1. 文档目的

这份文档做两件事：

1. 把下一阶段的产品升级计划写实，作为后续推进基线
2. 对照 2026-03-21 的代码现实，标注每一项当前到底做到哪了

状态说明：

- ✅ 已实现：主链路已经在代码里落地
- 🟡 部分实现：主链路已经能跑，也有基础测试或页面承接，但还没达到计划目标
- ⬜ 未开始：仓库里还没有对应实现

如果计划和仓库现实冲突，默认先以这份文档里的“当前进度”判断范围，再决定下一步。

---

## 2. 当前判断

PracticeHelper 现在已经不是“只有 3 个 topic 的可用 demo”了。代码里已经具备：

- 训练闭环
- 薄弱点记忆与推荐
- 岗位 JD 绑定训练
- 多轮训练基础能力
- 历史回看、弱项趋势和复习计划基础闭环

所以这份升级计划不是从 0 开始搭，而是在已有产品主链路上继续收口训练深度、训练留存和可审计性。

产品定位也应该写得更明确：

- PracticeHelper 不是静态题库
- 它是围绕**真实项目 + 目标岗位 JD**的 AI 面试教练

当前最有价值的三条差异化壁垒仍然成立：

1. 能从 Git 仓库抽取项目上下文，围绕用户真实经历出题
2. 能把岗位 JD 绑定进训练和复盘，形成岗位视角
3. 评分不只看答对没答对，还看落地感、表达力和抗追问能力

---

## 3. 优先级矩阵（含当前进度）

| # | 改进项 | 优先级 | 当前状态 | 当前判断 |
|---|--------|--------|----------|----------|
| 1 | 多轮训练流 | P0 | ✅ | `max_turns`、独立 `turn_index`、循环 FSM 和多轮 UI 都已落地 |
| 2 | 题库扩展 | P0 | ✅ | 已外置到 seed 文件，已覆盖 10 个 topic，且 basics 已支持按 weakness 自动选择候选 topic 的 mixed 模式 |
| 3 | 会话历史分页 | P0 | ✅ | `/api/sessions` 分页筛选、`/history` 页面和路由已落地 |
| 4 | 弱项趋势可视化 | P0 | ✅ | `weakness_snapshots`、趋势接口和首页 sparkline 已落地 |
| 5 | LangGraph 多节点 | P1 | 🟡 | `generate_question` 和 `evaluate_answer` 已不是纯单节点；`analyze_repo` 仍未做 rerank |
| 6 | 间隔重复 | P1 | 🟡 | `review_schedule` 已打通创建 / 到期展示 / 完成推进，首页待复习卡片可直接完成复习；但仍是 session 级，不是 weakness 级闭环 |
| 7 | 引导式 Onboarding | P1 | 🟡 | 首页已有 3 步引导提示，但不是 stepper，也还没形成完整首次引导流 |
| 8 | 自适应难度 | P1 | ✅ | `intensity=auto` 已接入后端策略和前端选择器 |
| 9 | 评估审计追踪 | P1 | 🟡 | `evaluation_logs` 已记录 `generate_question` / `evaluate_answer` / `generate_review` 及其 stream 变体的基础耗时，但还没有 prompt hash、model_name 真值和前端详情面板 |
| 10 | 前端体验增强 | P2 | 🟡 | 已补暗色模式、`Ctrl+Enter`、路由过渡和关键页面的移动端收口；但还没做更完整的动画体系和全页面专项适配 |
| 11 | 数据导出 | P2 | 🟡 | 已新增单次 Session 的 Markdown 导出接口和 Review 页导出入口；但还没做批量导出或更多格式 |
| 12 | Prompt 版本管理 | P2 | ⬜ | prompt 文件还没有版本号和 A/B 对比链路 |
| 13 | RAG 升级 | P2 | ⬜ | 当前仍是 SQLite FTS5 检索，没有向量检索层 |

---

## 4. P0：必须做

### 4.1 多轮训练流 ✅ 已实现

当前代码已经具备：

- `training_sessions.max_turns` 已支持 2-5 轮
- `training_turns` 已按独立 turn 落库，靠 `turn_index` 递增，不再把 followup 塞在同一条记录里
- `answer_service.go` 的状态流转已经是循环式：非最后一轮继续插入下一题，最后一轮进入 `review_pending`
- `TrainView` 可配置轮次数，`SessionView` 已展示当前轮次和多轮问答 UI
- sidecar 的评估 prompt 已带 `turn_index` / `max_turns`，并区分最后一轮是否需要继续追问

剩余缺口：

- 还需要补一次真实 3 轮 live 验证，确保文档、提示词和页面体验完全一致

### 4.2 题库扩展 ✅ 已实现

当前代码已经具备：

- 基础题模板已从硬编码迁到 `data/seed/question_templates.json`
- 启动时 seed 已改为从外部文件读取
- 当前 topic 已覆盖：
  - `go`
  - `redis`
  - `kafka`
  - `mysql`
  - `system_design`
  - `distributed`
  - `network`
  - `microservice`
  - `os`
  - `docker_k8s`
- `basics` 训练配置页已支持 `mixed`，后端会按当前 weakness 自动挑出候选 topic，再只把这些 topic 的模板交给 sidecar
- 每个 topic 当前都是 8 道模板题

当前验收点：

- `os` 已成为独立 topic，不再借 `docker_k8s` 代替系统层训练
- `mixed` 模式会跨 topic 选题，但不会把全部模板直接灌给模型，而是先按 weakness 选出候选 topic
- 前端训练配置、历史筛选、标签翻译和 session 展示都已支持 `os` / `mixed`

### 4.3 会话历史页 + 分页 ✅ 已实现

当前代码已经具备：

- 后端 `GET /api/sessions` 已支持 `page` / `per_page` / `mode` / `topic` / `status`
- 前端已有 `HistoryView`
- 路由 `/history` 已接入导航
- 页面支持分页、模式筛选、主题筛选和状态筛选

剩余缺口：

- 这项主线已经实现，后续只需要补真实使用验收，不需要再当作新需求重做

### 4.4 弱项趋势可视化 ✅ 已实现

当前代码已经具备：

- `weakness_snapshots` 表已存在
- 弱项增减时会写 snapshot
- 后端已有 `/api/weaknesses/trends`
- Dashboard 已展示 top weakness 趋势的小型折线图

当前实现和原计划的差异：

- 现在用的是首页内嵌 SVG sparkline，不是外部 chart 库
- 但“能看到弱项趋势”这个产品目标已经实现

---

## 5. P1：高价值

### 5.1 LangGraph 多节点 🟡 部分实现

当前代码已经具备：

- `generate_question` 已增加 `select_strategy -> generate`
- `evaluate_answer` 已增加输出校验、重试与超预算失败，不再是纯 `START -> run -> END`

剩余缺口：

- `analyze_repo` 还没有 `rank_chunks`
- `evaluate_answer` 还没拆成更明确的 `retrieve_context -> evaluate -> validate_output -> re_evaluate`
- `generate_review` 仍然是简单调用

### 5.2 间隔重复 🟡 部分实现

当前代码已经具备：

- `review_schedule` 表
- Review 完成后会自动创建或更新同一 session 的 schedule，避免重复堆积多条计划
- `GET /api/reviews/due` + 首页“今日待复习”卡片，能展示到期复习项
- `POST /api/reviews/due/:id/complete` 已接上，完成后会按 session 总分走简化版 SM-2 推进下一次复习时间

剩余缺口：

- 当前 schedule 还偏 session 级，不是完整的 weakness tag 级复习闭环
- 待复习入口还没有做到“点一下直接进入对应 topic 训练”

### 5.3 引导式 Onboarding 🟡 部分实现

当前代码已经具备：

- 首页会在没有 profile 时展示 3 步引导提示

剩余缺口：

- 还没有 stepper
- 还没有把“填画像 -> 导入项目 -> 首次训练”收成一个连续首访流程

### 5.4 自适应难度 ✅ 已实现

当前代码已经具备：

- `intensity` 已支持 `auto`
- 后端会按最近 5 个已完成 session 的平均分自动落到 `light` / `standard` / `pressure`
- 前端训练配置页已开放该选项

### 5.5 评估审计追踪 🟡 部分实现

当前代码已经具备：

- `evaluation_logs` 表
- `generate_question` / `evaluate_answer` / `generate_review` 及其 stream 变体都会记录 `flow_name` 和 `latency_ms`
- repo / service 测试已经覆盖日志持久化和创建 session 时写入审计记录

剩余缺口：

- 还没有 `prompt_hash`
- 还没有原始 LLM 响应持久化
- `model_name` 目前没有真正填充
- Review 页也没有“查看评估详情”的展开面板

---

## 6. P2：锦上添花

### 6.1 前端体验增强 🟡 已落第一版

当前已经补上第一轮可直接感知的体验项：

- 暗色模式
- `Ctrl+Enter` 提交答案
- 页面过渡动画
- `AppShell`、`History`、`Session`、`Review` 这几个关键页面的移动端收口

说明：

- 当前已经不再只是“基础响应式布局”
- 但还没有做更完整的动效体系、全页面移动端专项适配和更细的主题打磨

### 6.2 数据导出 🟡 已落第一版

当前已新增：

- `GET /api/sessions/:id/export?format=markdown`
- Review 页“导出 Markdown”入口
- 单次 Session 全量 Markdown 报告生成链路

当前还没做：

- 历史列表批量导出
- JSON / PDF 等其他格式

### 6.3 Prompt 版本管理 ⬜ 未开始

当前 prompt 文件是直接按文件名读取，没有显式版本号、版本记录或 A/B 对比能力。

### 6.4 RAG 升级 ⬜ 未开始

当前仍是 SQLite FTS5 检索，尚未接入 embedding、sqlite-vec 或向量召回。

---

## 7. 按当前仓库现实重排后的执行顺序

如果按当前代码现实继续推进，更合理的顺序应该是：

1. 先把 **P1 半成项** 收口
2. 再把 **已完成第一轮的 P2** 保持在当前边界，不继续顺手扩
3. 最后再单开 **P2 剩余项**

具体来说：

1. P0 现在已经收口，不需要再围绕 topic 覆盖和 mixed 训练继续反复返工
2. P1 里最值得优先做的是：LangGraph 深化、复习计划从 session 级补到 weakness 级、评估审计详情补齐
3. P2 里已经落了一轮的是前端体验增强和单次 Markdown 导出；下一步仍应优先回到 P1，而不是继续扩到 Prompt 版本管理和 RAG
