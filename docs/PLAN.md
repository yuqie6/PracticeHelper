# 开发计划 - PracticeHelper

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

## Phase 5 - 端到端验证 🟡 进行中

目标：配置真实 LLM 后，新用户能无障碍走完画像 → 导入项目 → 基础训练 → 项目训练 → 复盘。

### 待完成

- session 状态机异常路径收口：review 失败后的中间状态与恢复

### 本阶段已补上的关键修复

- `review_pending` 会话已支持 `retry-review` 恢复入口，前端可直接重试生成复盘
- 回答提交与复盘重试已增加服务端原子抢占，避免重复点击或多标签页并发导致重复评估 / 重复复盘
- 训练页在处理中会锁定输入区，并在冲突错误后主动刷新到最新 session 状态
- 真实 LLM 主链路可通过 `scripts/e2e_live.py` 持续回放；默认样例已收口到 `scripts/e2e_live.sample.json`

### 完成标准

从零开始完成完整流程无阻塞，失败态有明确恢复路径。

---

## Phase 6 - 答题反馈 V2 ⬜ 待开始

目标：把训练体感从"能用"升级到"用户知道发生了什么、为什么这么判、下一步该怎么做"。

详见 [ANSWER_FEEDBACK_UX_V2.md](./ANSWER_FEEDBACK_UX_V2.md)。

### Layer 1：消除卡住感

- 前端 `ProgressPanel` 改为消费真实 `StreamEvent.phase`，不再用定时器假进度
- 后端补充 `answer_saved` / `evaluation_started` 等状态事件
- 提交确认态 + 失败保留草稿
- 复盘前收口过渡

### Layer 2：反馈可理解性

- sidecar `EvaluationResult` 新增 `headline` / `suggestion` / `followup_intent`
- 反馈卡信息层级重排：结论优先，分项得分折叠
- 追问卡新增意图展示

### Layer 3：训练闭环

- sidecar `ReviewCard` 新增 `top_fix` / `top_fix_reason` / `recommended_next`
- 复盘页改为动作导向：优先修正项 → 下一轮推荐 → 一键开始
- "继续训练"携带推荐参数跳转

### 完成标准

- 提交后能看到真实阶段推进
- 每次反馈有一句话结论 + 改进建议，追问有意图说明
- 复盘页有具体下一轮推荐，可一键开始

---

## Phase 7 - 推荐质量与智能体增强 ⬜ 待开始

目标：让历史训练真正反馈到下一轮推荐，提升 AI 输出的稳定性和可信度。

### 待完成

- 验证薄弱点 severity 升降机制
- 验证首页推荐与真实弱项的绑定
- 弱项记忆增加时间衰减，区分"偶发卡壳"和"稳定弱项"
- 增加种子题目模板覆盖度
- 追问生成增加"证据不足时保守表达"约束

### 完成标准

做 3 轮训练后，首页建议准确反映薄弱环节，弱项改善后推荐会变化。

---

## 明确延后

以下方向正确但不属于近期计划：

- 导入任务升级为可恢复的 worker / 队列模型
- 仓库理解沉淀细粒度分析资产
- 多 agent 并行分析仓库
- tool-loop 流式化替代 single-shot streaming
- 评分 rubric 程序化、证据绑定（引用用户原文 + repo chunk）
