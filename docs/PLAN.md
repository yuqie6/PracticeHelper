# 开发计划 - PracticeHelper

本文档是当前阶段的执行计划。产品方向和阶段划分见 [VISION.md](./VISION.md)。

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

详见 [ANSWER_FEEDBACK_UX_V2.md](./ANSWER_FEEDBACK_UX_V2.md)。

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
- 基础题种子模板已扩到 Go / Redis / Kafka 每个 topic 至少 5 条
- 追问 prompt 已增加“证据不足时保守表达”约束，避免把未证实事实写成既定前提
- 已补充 repo / service / sidecar 测试，覆盖衰减排序、推荐文案、题库覆盖和 prompt 约束

### 完成标准

- 自动化测试能证明衰减排序、推荐绑定、题库覆盖和 prompt 约束都已生效
- 继续做 3 轮 live 训练时，首页建议应能随真实薄弱环节变化而变化

---

## 阶段 B - JD 岗位视角产品级收口 ⬜ 当前主线

目标：让训练从“泛泛练”变成“围绕目标岗位练”，让问题、评分和复盘都能明确回答“这是不是岗位真正看重的内容”。

详见 [JD_TRAINING_STAGE_B.md](./JD_TRAINING_STAGE_B.md)。

### 本阶段要完成

- 新增独立 JD 页面，支持多份 JD 管理，而不是把 JD 文本塞进画像页
- 每份 JD 支持多次分析，并保留完整分析历史；训练默认绑定所选 JD 的最新成功分析快照
- TrainView 在开始训练前明确选择本轮使用的 JD；未选择时保留现有通用训练路径
- `training_sessions` 持久化 `job_target_id` 与 `job_target_analysis_id`，保证一轮训练吃的是固定快照，不受后续切换影响
- basics / project 两条出题链路都引用 JD 要求，避免继续只围绕 topic 或项目素材泛泛出题
- 评分、追问和复盘都引用 JD 关注点，能明确指出“岗位要求了什么、这次回答没体现什么”
- 为 JD 分析、训练绑定和回看链路补齐 API/前端/sidecar 测试闭环

### 完成标准

- 用户可以在独立 JD 页面保存多份 JD，查看每份 JD 的最新分析和分析历史
- 用户开始一轮 basics 或 project 训练前，可以明确选择本轮参考的 JD，并能看懂它当前为什么可用 / 不可用
- 问题、评分和复盘都能引用所选 JD 的能力要求，而不是只给泛化反馈
- 回看 session / review 时，能知道这轮训练绑定的是哪份 JD、哪次分析快照
- JD 状态变成 stale / failed / running 时，页面和 live E2E 都能明确验证“旧快照可回看，但新训练不可继续绑定”

---

## 明确延后

以下方向正确但不属于近期计划：

- 导入任务升级为可恢复的 worker / 队列模型
- 仓库理解沉淀细粒度分析资产
- 多 agent 并行分析仓库
- tool-loop 流式化替代 single-shot streaming
- 评分 rubric 程序化、证据绑定（引用用户原文 + repo chunk）
- 自动抓取 JD / 多岗位对比 / 简历联动
