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

| #   | 改进项            | 优先级 | 当前状态 | 当前判断                                                                                                                                                                                                 |
| --- | ----------------- | ------ | -------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1   | 多轮训练流        | P0     | ✅       | `max_turns`、独立 `turn_index`、循环 FSM 和多轮 UI 都已落地                                                                                                                                              |
| 2   | 题库扩展          | P0     | ✅       | 已外置到 seed 文件，已覆盖 10 个 topic，且 basics 已支持按 weakness 自动选择候选 topic 的 mixed 模式                                                                                                     |
| 3   | 会话历史分页      | P0     | ✅       | `/api/sessions` 分页筛选、`/history` 页面和路由已落地                                                                                                                                                    |
| 4   | 弱项趋势可视化    | P0     | ✅       | `weakness_snapshots`、趋势接口和首页 sparkline 已落地                                                                                                                                                    |
| 5   | LangGraph 多节点  | P1     | 🟡       | `analyze_repo` 已补 `collect_bundle -> rank_chunks -> summarize`，`evaluate_answer` / `generate_review` 也已加校验重试；但整体还不是更复杂的多节点编排                                                 |
| 6   | 间隔重复          | P1     | 🟡       | `review_schedule` 已打通创建 / 到期展示 / 完成推进，并能按 weakness tag 建计划和回显弱项标签；但完成复习仍按 session 总分推进，入口也还没直达对应训练                                                    |
| 7   | 引导式 Onboarding | P1     | 🟡       | 首页已有 3 步引导提示，但不是 stepper，也还没形成完整首次引导流                                                                                                                                          |
| 8   | 自适应难度        | P1     | ✅       | `intensity=auto` 已接入后端策略和前端选择器                                                                                                                                                              |
| 9   | 评估审计追踪      | P1     | 🟡       | `evaluation_logs` 已记录 `flow_name`、`latency_ms`、`model_name`、`prompt_set_id`、`prompt_hash`、`raw_output`，并已开放 session 级详情接口；但前端还没把原始响应用清晰面板展开，也没有在 Review 页内直接查看 |
| 10  | 前端体验增强      | P2     | ✅       | 已补暗色模式、`Ctrl+Enter`、路由过渡、全局页面/列表进入动效、剩余页面移动端收口，并清掉暗色主题下的硬编码高刺激文字色                                                                                    |
| 11  | 数据导出          | P2     | ✅       | 已支持单次 Session 的 Markdown / JSON / PDF 导出，以及 History 跨页勾选后的批量 ZIP 导出                                                                                                                 |
| 12  | Prompt 版本管理   | P2     | 🟡       | sidecar 已支持 prompt set registry、训练页选版本、session 绑定版本、实验页跨 session A/B 对比和审计明细；但还没有在线编辑、显著性分析和更细粒度 flow 级切换                                              |
| 13  | RAG 升级          | P2     | ⬜       | 当前仍是 SQLite FTS5 检索，没有向量检索层                                                                                                                                                                |

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
- `analyze_repo` 已增加 `collect_bundle -> rank_chunks -> summarize`
- `evaluate_answer` 已增加 `retrieve_context -> evaluate -> validate_output -> re_evaluate`
- `generate_review` 已增加校验、重试与超预算失败，不再只是简单单节点调用

剩余缺口：

- 还没有把更多上下文准备、审计检查或失败分流继续拆成独立节点
- 目前的图主要集中在关键推理链路，还没有形成更完整的图级观测与回放能力

### 5.2 间隔重复 🟡 部分实现

当前代码已经具备：

- `review_schedule` 表
- Review 完成后会自动创建或更新复习计划，已有 weakness tag 时会按弱项拆成多条 schedule
- `GET /api/reviews/due` + 首页“今日待复习”卡片，能展示到期复习项
- `GET /api/reviews/due` 已能回显 `weakness_kind / weakness_label`
- `POST /api/reviews/due/:id/complete` 已接上，完成后会按 session 总分走简化版 SM-2 推进下一次复习时间

剩余缺口：

- weakness 级 schedule 虽已落地，但完成复习时仍是用 session 总分推进，而不是按弱项粒度反馈
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
- `generate_question` / `evaluate_answer` / `generate_review` 及其 stream 变体都会记录 `flow_name`、`latency_ms`、`model_name`、`prompt_set_id`、`prompt_hash`、`raw_output`
- 已新增 `GET /api/sessions/:id/evaluation-logs`
- Prompt 实验页已能按 session 展开每个 flow 的 `model_name / prompt_hash / latency_ms`
- repo / service / web helper 测试已经覆盖日志持久化、prompt 版本选择和实验页基础行为

剩余缺口：

- `raw_output` 虽已落库并通过 session 审计接口返回，但前端还没有直接查看原始响应的清晰入口
- Review 页还没有“查看评估详情”的内联展开面板
- 目前的明细查看入口主要在 Prompt 实验页，不是所有页面都能直接展开

---

## 6. P2：锦上添花

### 6.1 前端体验增强 ✅ 已实现

当前代码已经具备：

- 暗色模式
- `Ctrl+Enter` 提交答案
- 页面过渡动画
- 全局页面分段进入动画和列表 stagger 动效
- `AppShell`、`Home`、`Profile`、`Projects`、`JobTargets`、`PromptExperiments`、`History`、`Session`、`Review` 的移动端收口
- 暗色模式色板、阴影和背景层次的第二轮打磨，避免高饱和度和纯黑阴影过于刺眼
- 针对 `text-black/80`、`text-gray-400` 这类旧硬编码文字色的主题化覆盖，避免暗色模式漏出刺眼对比
- AppShell 导航在小屏上改成横向可滚动，避免顶部导航挤压成两列大按钮

当前验收点：

- 不再只有路由切换动画，页面主区块和长列表都会按统一节奏进入
- 暗色模式下主要阅读文本、阴影、表单和卡片不再混入硬编码黑灰色
- 之前没专项收口的几个页面，在窄屏下按钮、列表和编辑区都能正常堆叠，不需要横向滚动才能完成主流程

### 6.2 数据导出 ✅ 已实现

当前代码已经具备：

- `GET /api/sessions/:id/export?format=markdown|json|pdf`
- `POST /api/sessions/export` 批量打包选中的 Session 文件
- Review 页可直接切换导出格式
- History 页可跨页勾选多条训练记录，再按选定格式打包成 ZIP 导出
- 单次 Session 的 Markdown 报告、结构化 JSON 和文本版 PDF 报告生成链路

当前验收点：

- 单次导出不再只有 Markdown
- 批量导出不再局限于当前页临时勾选，而是支持跨页累计选择
- 导出文件名会按格式区分扩展名和批量 ZIP 命名，下载后不需要手工改后缀

### 6.3 Prompt 版本管理 🟡 已落第一版

当前已经补上第一轮可用链路：

- sidecar prompt 文件已改成 `prompt_set` 目录 + `registry.json`
- 训练页已支持直接选择 prompt 版本，默认值来自 registry 的默认版本
- 创建 session 后会把整套 `prompt_set_id` 固定绑定到 session，不会在 question / answer / review 之间漂移
- `evaluation_logs` 已落 `prompt_set_id` 和 `prompt_hash`
- 已新增：
  - `GET /api/prompt-sets`
  - `GET /api/prompt-experiments`
  - `GET /api/sessions/:id/evaluation-logs`
- History / Review 已展示当前 session 的 prompt 版本
- 已新增独立的 `/prompt-experiments` 页面，可做跨 session 的左右版本聚合对比，并展开单次样本的审计明细

当前还没做：

- prompt 在线编辑器
- prompt 删除 / 归档 UI
- 同一 session 内的双跑对比
- 日期窗口、显著性分析、胜率统计等更完整的实验分析能力
- 每个 flow 独立切换不同 prompt 版本

### 6.4 RAG 升级 ⬜ 未开始

当前仍是 SQLite FTS5 检索，尚未接入 embedding、sqlite-vec 或向量召回。

---

## 7. 按当前仓库现实重排后的执行顺序

如果按当前代码现实继续推进，更合理的顺序应该是：

1. 先把 **P1 半成项** 收口
2. 再把 **已经收口完成的 P2** 保持在当前边界，不继续顺手扩
3. 最后再单开 **P2 剩余项**

具体来说：

1. P0 现在已经收口，不需要再围绕 topic 覆盖和 mixed 训练继续反复返工
2. P1 里最值得优先做的是：LangGraph 深化、把当前复习计划补成更完整的 weakness 级闭环、评估审计详情补齐
3. P2 里已经收口完成的是前端体验增强、数据导出和 Prompt 版本管理 v1；下一步应回到 P1 半成项，优先补 LangGraph 深化、weakness 级复习闭环和更完整的审计详情，再决定是否单开 `6.4 RAG 升级`
