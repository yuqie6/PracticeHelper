# 开发计划 - PracticeHelper

本文档是当前阶段的执行计划。产品方向和阶段划分见 [VISION.md](./VISION.md)。

如果后续要把 sidecar 从“当前这套已具备 agent loop、长期记忆装载、结构化 trace 和受控动作执行的受约束 agent runtime”继续升级为“训练域里的成熟 agent runtime”，当前仓库已经有一份按源码审校过的独立方案，见 [AGENT_DEEP_REDESIGN_PLAN.md](../plans/AGENT_DEEP_REDESIGN_PLAN.md)。

当前这轮工程收口则走独立执行计划，见 [ARCHITECTURE_CONVERGENCE_PLAN.md](../plans/ARCHITECTURE_CONVERGENCE_PLAN.md)。它回答拆分、卫生和文档同步，不替代这里的产品主线。

## 已完成

| Phase | 内容 | 交付物 |
|-------|------|--------|
| 0 | 立项与技术栈决策 | README / PRD / PLAN / ARCHITECTURE |
| 1 | 工程骨架 | web (Vue3+Vite) / server (Gin+SQLite) / sidecar (FastAPI+LangGraph) / Makefile |
| 2 | 画像与首页 | HomeView / ProfileView / dashboard API |
| 3 | 项目导入 | ProjectsView / 仓库克隆→chunk→LLM 画像管线 / 后台任务+进度轮询+失败重试 |
| 4 | 基础+项目训练 | TrainView / SessionView / ReviewView / 出题→评分→追问→复盘→薄弱点更新全链路 |
| 5-patch | 主线修复 | schema 兼容 / sidecar 超时可配 / 导入后台化+通知 / 流式结构化卡片 / 错误提示收口 |

---

## Phase 5 - 端到端验证 ✅ 已完成

目标：配置真实 LLM 后，新用户能无障碍走完画像 → 导入项目 → 基础训练 → 项目训练 → 复盘。

### 本阶段已补上的关键修复

- `review_pending` 会话已支持 `retry-review` 恢复入口，前端可直接重试生成复盘
- 回答提交与复盘重试已增加服务端原子抢占，避免重复点击或多标签页并发导致重复评估 / 重复复盘
- 训练页在处理中会锁定输入区，并在冲突错误后主动刷新到最新 session 状态
- 真实 LLM 主链路可通过 `scripts/e2e_live.py` 持续回放；默认样例已收口到 `scripts/e2e_live.sample.json`
- 复盘生成失败时会保留 `review_pending`，并返回可识别错误码引导前端直接进入恢复入口

### 完成标准

从零开始完成完整流程无阻塞，失败态有明确恢复路径。

---

## Phase 6 - 答题反馈 V2 ✅ 已完成

目标：把训练体感从"能用"升级到"用户知道发生了什么、为什么这么判、下一步该怎么做"。

详见 [ANSWER_FEEDBACK_UX_V2.md](../records/ANSWER_FEEDBACK_UX_V2.md)。

### 本阶段已完成

- 训练页与开始训练页的进度面板已改为消费真实流式事件，不再依赖定时器假进度
- 回答提交流式链路补充 `answer_received` / `answer_saved` / `evaluation_started` / `feedback_ready` / `followup_ready` / `review_started` / `review_saved` 状态事件
- 训练页在提交后会固定展示已提交内容，失败时保留草稿，避免误以为回答丢失
- 训练完成后会先展示复盘收口过渡，再跳转到复盘页
- `EvaluationResult` 已新增 `headline` / `suggestion` / `followup_intent`，训练页反馈区改为结论优先的层级化展示
- `ReviewCard` 已新增 `top_fix` / `top_fix_reason` / `recommended_next`，复盘页可直接启动推荐的下一轮训练
- `TrainView` 已支持从复盘推荐参数预填 mode/topic/project_id

### 完成标准

- 提交后能看到真实阶段推进
- 每次反馈有一句话结论 + 改进建议，追问有意图说明
- 复盘页有具体下一轮推荐，可一键开始

---

## Phase 7 - 推荐质量与智能体增强 ✅ 已完成

目标：让历史训练真正反馈到下一轮推荐，提升 AI 输出的稳定性和可信度。

### 本阶段已完成

- `weakness_tags` 在读取时会按 `last_seen_at` 计算有效 severity，区分近期稳定弱项和陈旧偶发问题
- 旧弱项再次命中时，会先按衰减后的有效热度继续累加，而不是沿着陈旧高点直接叠加
- dashboard 的 `today_focus` / `recommended_track` 已直接绑定当前有效 Top1 弱项，推荐文案会带出具体 label
- 基础题种子模板已经外置，并扩到 10 个 basics topic；`mixed` 也已支持按 weakness 自动跨 topic 选题
- 追问 prompt 已增加“证据不足时保守表达”约束，避免把未证实事实写成既定前提
- 已补充 repo / service / sidecar 测试，覆盖衰减排序、推荐文案、题库覆盖和 prompt 约束

### 完成标准

- 自动化测试能证明衰减排序、推荐绑定、题库覆盖和 prompt 约束都已生效
- 继续做 3 轮 live 训练时，首页建议应能随真实薄弱环节变化而变化

---

## 阶段 B - JD 岗位视角产品级收口 ✅ 已落地

目标：让训练从“泛泛练”变成“围绕目标岗位练”，让问题、评分和复盘都能明确回答“这是不是岗位真正看重的内容”。

详见 [JD_TRAINING_STAGE_B.md](../records/JD_TRAINING_STAGE_B.md)。

### 本阶段已经落地

- 独立 JD 页面已经存在，支持多份 JD 管理、默认 JD 和分析历史回看
- `training_sessions` 已持久化 `job_target_id` 与 `job_target_analysis_id`，一轮训练固定绑定分析快照
- `TrainView` 已支持显式选择本轮参考 JD，未就绪的 JD 会被阻止绑定
- basics / project 两条出题链路都已注入 JD 分析快照
- 评分、追问和复盘都已引用 JD 关注点
- session / review / dashboard 都已能回看当前轮次绑定的 JD

### 验收现状

- `scripts/e2e_live.py` 已覆盖 JD 创建 -> 分析成功 -> 激活 -> basics / project 训练绑定 -> review 回看 -> `stale` 阻断 -> dashboard 从 `job_target` 回退到 `generic`
- `job_target_service` / repo 层测试已覆盖 `running` / `failed` / `stale` 的不可绑定语义，以及 `latest_successful_analysis` 在失败或过期后的保留逻辑
- web 侧 `jobTargetStatus` 与训练入口文案已经把 `idle / running / failed / stale / succeeded` 五种状态拆开处理，阶段 B 现在可以视为产品级收口
- 后续如果继续补 `running / failed` 的 live UI smoke，更适合作为阶段 C 的回归测试，而不是阶段 B 的未完成项

---

## 阶段 C - 产品全面升级计划 🟡 当前主线

目标：把产品从“能练、能记住、能带岗位视角”继续升级成“训练更深、回看更清楚、复习更持续、可审计性更强”的 AI 面试教练。

详见 [PRODUCT_UPGRADE_PLAN.md](../plans/PRODUCT_UPGRADE_PLAN.md)。

### 当前代码已经具备的基础

- 多轮训练流已经落地，`max_turns` 可配，追问按独立 turn 持久化
- 历史页、分页筛选、弱项趋势图和首页待复习卡片已经落地
- 题库已经外置到 seed 文件，并扩到 10 个 topic；`mixed` 会按 weakness 选择候选 topic
- `intensity=auto` 已稳定可用；`review_schedule` 也已打通 review 生成 -> 到期展示 -> 完成推进 的基础链路
- `evaluation_logs` 已覆盖 `generate_question` / `evaluate_answer` / `generate_review` 及其 stream 变体，并且 Review 审计面板与 Prompt 实验页都已有前端承接
- `runtime_trace` 已贯通 sidecar runtime、Go 持久化阶段和前端展示；当前不再是“只有 prompt 元信息，没有统一 trace”
- LangGraph 当前已经收成“`analyze_repo` 多节点 + `generate_question` 策略节点 + `evaluate_answer / generate_review` 薄壳图”的结构；输出校验、重试预算和 `side_effects` 收口都在 `agent_runtime.py`
- 默认 JD、`recommendation_scope` 和 generic fallback 语义已经收口，岗位模式不再是阶段 C 之前的阻塞项
- 关键动作已经进入“`side_effects` + 少量 typed command”双轨：`transition_session` 和 `upsert_review_path` 已有第一版 command path，但 Go 仍保留最终状态机和持久化边界

### 当前 agent 主线定位

- PracticeHelper 当前追求的不是“把系统改写成通用 agent 平台”，而是把现有训练链路的 `sidecar` 继续做成训练域里的成熟 agent runtime
- 这条 agent 主线是阶段 C 的技术底座，不替代“训练深度与留存升级”这条产品主线
- 近期仍然是单 agent 主路径；多 agent 只写成后续高价值长任务的演进方向，不进入当前训练热路径
- Go 继续保留产品边界、状态机、持久化、审计和恢复入口；sidecar 继续负责上下文理解、规划、工具使用、输出校验和结构化意图生成
- 动作模型已经不是“只有 side effects”：当前主路径是 cheap/local 动作继续走 `side_effects`，关键状态决策开始走受控 typed command，但不放权到 sidecar 直写数据库
- 近期优先补的是检索、memory 利用、失败恢复和可观测性，而不是直接铺复杂多 agent 编排

### 本阶段当前更适合继续推进的方向

- 不再把已落地的多轮训练、弱项级待复习入口、评估审计面板重新写成“半成能力”
- 把 Prompt 版本管理从 v1 继续补强：当前已有版本选择、A/B 对比和审计明细，但还没有在线编辑、更细粒度 flow 级切换和更强实验分析
- 把单 agent 成熟化作为当前技术主线：优先补检索升级、memory 利用、恢复语义和观测能力，而不是先上多 agent
- observation / session summary 已落第一版 embedding / hybrid rerank；repo chunk 这一轮已经补上
  `Qdrant vector recall + optional rerank + FTS5 fallback`，但不要顺手把 graph / 更大范围
  的全量 RAG 写成已经在做
- 如果要继续推进 sidecar agent 化，按 [AGENT_DEEP_REDESIGN_PLAN.md](../plans/AGENT_DEEP_REDESIGN_PLAN.md) 的分阶段路线渐进推进，而不是一次性推翻现有 stream / FSM / sidecar client 主链路
- Go 侧继续保留最终落库和状态机边界；sidecar 的动作能力维持 `side_effects` + 少量 typed command 双轨，不把关键状态迁移退化成自由 side effect
- 多 agent 目前只作为后续高价值长任务的实现蓝图，不作为阶段 C 的当前热路径方案

### 完成标准

- 已完成能力不再在文档里被写成“待补缺口”
- 当前真未做项和已实现能力的边界清楚
- `runtime_trace` 和第一版 typed command path 被当成当前事实，而不是继续写成“后续概念”
- 文档、代码和页面对当前主线的描述重新一致

---

## 明确延后

以下方向正确但不属于近期计划：

- 导入任务升级为可恢复的 worker / 队列模型
- 仓库理解沉淀细粒度分析资产
- 多 agent 并行分析仓库
- stream 里的更细粒度 tool-call 可视化与 token 级 agent 输出
- 评分 rubric 程序化、证据绑定（引用用户原文 + repo chunk）
- 自动抓取 JD / 多岗位对比 / 简历联动
