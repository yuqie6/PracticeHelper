# 开发计划 - PracticeHelper

## Phase 0 - 立项与技术栈决策 ✅ 已完成

确定产品边界、技术选型和工程约束。

**交付物**：README.md、docs/PRD.md、docs/PLAN.md、docs/ARCHITECTURE.md

---

## Phase 1 - 工程骨架 ✅ 已完成

搭建可持续开发的三端最小骨架。

**交付物**：
- `web/`：Vue 3 + Vite + TypeScript + pnpm，Tailwind + neo-brutalist CSS 设计系统
- `server/`：Gin + SQLite，完整分层结构（config / controller / service / repo / sidecar）
- `sidecar/`：FastAPI + LangGraph + AgentRuntime
- Makefile 统一命令、bootstrap.sh 一键初始化

---

## Phase 2 - 画像与首页 ✅ 已完成

**交付物**：
- 首页（HomeView）：Dashboard 聚合展示——今日建议、投递倒计时、薄弱点热区、训练记录、推荐专项、画像摘要
- 画像页（ProfileView）：完整表单（目标岗位、公司类型、阶段、投递时间、技术栈、主讲项目、自感弱项），支持创建和更新
- Go API：`GET/POST/PATCH /api/profile`、`GET /api/dashboard`
- 前端数据层：vue-query 封装，保存后自动刷新 dashboard

---

## Phase 3 - 项目导入 ✅ 已完成

**交付物**：
- 项目页（ProjectsView）：仓库 URL 导入、项目列表、项目画像编辑（名称、摘要、技术栈、亮点、难点、trade-off、ownership、可追问点）
- sidecar 管线：克隆仓库 → 文件过滤 → chunk 切分 → LLM 生成画像
- Go API：`POST /api/projects/import`、`GET /api/projects`、`GET/PATCH /api/projects/:id`

---

## Phase 4 - 基础知识训练 + 项目训练 ✅ 已完成

**交付物**：
- 训练配置页（TrainView）：模式选择（基础知识 / 项目）、主题选择、项目选择、强度选择
- 训练过程页（SessionView）：问题展示、答案输入、评估反馈（评分、优点、漏洞）、追问自动切换、训练完成后自动跳转复盘
- 复盘页（ReviewView）：总评、分项得分、回答亮点、漏洞、下次训练重点、继续训练入口
- Go service：完整的训练会话编排（出题 → 评分 → 追问 → 复盘 → 薄弱点更新）
- sidecar：generate_question / evaluate_answer / generate_review 三条 AI 链路
- 种子题目：Go / Redis / Kafka 各 1 个模板

---

## Phase 5 - 端到端验证与健壮性 🟡 进行中

三端代码结构已就绪，当前已经补上训练恢复入口、请求级日志、日志落盘和前端阶段化等待提示，但仍缺少真实环境下的完整流程验证和更完善的异常处理。

### 待完成
- 配置 LLM 后跑通完整训练流程（画像 → 导入 → 出题 → 回答 → 追问 → 复盘）
- 前端错误处理（API 失败时的提示，mutation 的 onError 处理）
- 前端加载态（初次进入页面时的 loading 状态）
- 边界情况（导入失败、LLM 超时、会话状态异常等）
- 进一步细化训练阶段反馈（不仅有等待提示，还能展示更准确的服务端阶段）

**完成标准**：一个新用户从零开始，能无障碍地完成画像 → 导入项目 → 做一轮基础训练 → 做一轮项目训练 → 查看复盘

---

## Phase 6 - 推荐与闭环收口 ⬜ 待开始

让历史训练真正反馈到下一轮训练。

### 待完成
- 验证薄弱点 severity 的升降机制是否符合预期
- 验证首页推荐是否与真实弱项绑定
- 增加更多种子题目模板（当前每个主题只有 1 题）

**完成标准**：做 3 轮训练后，首页建议准确反映薄弱环节，且弱项改善后推荐会变化

---

## 执行顺序

Phase 5（端到端验证）→ Phase 6（推荐闭环），核心工作从编码转向验证和打磨。
