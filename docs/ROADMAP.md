# ROADMAP - PracticeHelper

## 1. 文档目的

这份文档是 PracticeHelper 的实际开发路线图。

它回答 4 个问题：

1. v0 到底要做成什么
2. 为什么要按这个顺序开发
3. 每个阶段具体做什么
4. 什么叫"这一阶段完成了"

如果开发过程中出现分歧，默认以这份文档为推进基线。

---

## 2. 项目目标

PracticeHelper 的 v0 目标是：一个面向后端 / Agent 求职方向的单用户面试训练 Agent。

这个 Agent 至少要具备：

- **LLM**：负责出题、追问、评估、复盘
- **Memory**：记住用户画像、项目画像、训练记录、薄弱点
- **Tools**：导入仓库、检索项目上下文、更新弱点、创建训练 session、生成复盘

v0 不是为了把"agent"这个词贴上去，而是要先长出最基本的 agent 骨架。

---

## 3. v0 成功定义

只要满足下面 5 条，就说明 v0 已经成立：

1. 用户可以完成画像初始化
2. 用户可以导入一个 GitHub 项目并修正项目画像
3. 用户可以完成一轮基础知识训练（主问题 + 追问 + 复盘）
4. 用户可以完成一轮项目训练（围绕真实项目追问 + 复盘）
5. 系统能把训练结果沉淀到 weakness memory，并影响下一轮推荐

如果缺任意一条，v0 都只能算半成品。

---

## 4. 开发总原则

### 4.1 先做纵向闭环，不先摊平模块

正确顺序是：先让系统真的能训练一次 → 再让系统能训练得更准 → 最后再让系统变聪明。

### 4.2 先让它对自己有用

判断一个功能要不要做的标准不是"像不像产品"，而是：明天会不会真的打开它练 15 分钟。

### 4.3 先把 agent 的骨头长出来

v0 不追求多 agent 协作、长链自治、花哨 planning loop。v0 只追求有记忆、有工具、有结构化决策、有真实训练价值。

### 4.4 先做规则可控，再逐步提高 LLM 自由度

先输出结构正确，再优化回答质量。先做到稳定可复现，再追求"像真人教练"。

---

## 5. 范围分层

### v0 必做

- 用户画像
- 首页 dashboard
- GitHub 项目导入
- 项目画像草稿 + 手工编辑
- 基础知识训练闭环
- 项目训练闭环
- 复盘卡
- weakness memory
- 推荐下一轮训练

### v0 明确不做

- 算法在线判题
- 多用户 / 登录 / 权限
- 简历解析 / JD 解析
- 全真模拟面试长流程
- 通用 repo chat
- 向量数据库
- 复杂多 agent 编排
- 移动端原生适配

### v1 候选

- 算法训练模块
- 模拟面试模式（30~45 分钟）
- JD 定制训练
- 简历联动
- 主动推荐训练计划
- 更智能的弱点衰减与强化逻辑

---

## 6. 能力地图

### 6.1 LLM 层（Python sidecar）

4 条 AI 链路，所有输出必须结构化且 schema 可校验：

- `analyze_repo`：仓库分析 → 项目画像草稿
- `generate_question`：基于主题/项目/弱项 → 出题
- `evaluate_answer`：评分 + 追问生成 + 弱项识别
- `generate_review`：整轮训练 → 复盘卡

### 6.2 Memory 层（Go + SQLite）

实际可查询、可更新、可影响下一轮训练的数据：

- user_profile / project_profile / repo_chunks
- training_session / training_turn / review_card
- weakness_tag / question_template

### 6.3 Tool 层

系统内部具备的工具能力：

- import_github_repo / scan_repo_text_files / search_project_chunks
- load_user_profile / load_project_profile
- create_training_session / save_answer
- upsert_weakness / generate_review_card / recommend_next_track

### 6.4 Service 层（Go API）

编排层：管理 session 状态机、决定何时调用 sidecar、决定用哪些 memory 进入问题生成与评估、将结果沉淀为弱点与推荐。

---

## 7. 已完成阶段

以下阶段的代码已全部就绪。

### Phase 0 - 基线稳定化 ✅

- 三端 make lint / make test / make build 可执行
- golangci-lint 版本固定在 `.tools/`，不依赖全局环境
- `.env` 在 `.gitignore` 中，只保留 `.env.example`
- `bootstrap.sh` 一键初始化
- 文档口径统一（README / PRD / ARCHITECTURE / PLAN / ROADMAP）

### Phase 1 - 用户画像闭环 ✅

- Go API：`GET/POST/PATCH /api/profile`，`GET /api/dashboard`
- 前端 ProfileView：完整表单（岗位、公司类型、阶段、投递时间、技术栈、主讲项目、自感弱项）
- 保存 → 刷新回显 → dashboard 聚合画像摘要
- 推荐逻辑：无画像时引导建画像，有画像无训练时引导开第一轮

### Phase 2 - GitHub 项目导入闭环 ✅

- sidecar：`git clone --depth 1` → 文件过滤 → chunk 切分 → LLM 生成画像草稿
- Go API：`POST /api/projects/import`（含重复导入检测）、`GET /api/projects`、`GET/PATCH /api/projects/:id`
- FTS5 全文索引建立
- 前端 ProjectsView：URL 导入、项目列表、画像编辑（名称、摘要、技术栈、亮点、难点、trade-off、ownership、可追问点）

### Phase 3 - 基础知识训练闭环 ✅

- 种子题目模板：Go / Redis / Kafka 各 1 个
- Go service 完整编排：创建 session → 出题 → 用户回答 → 评分 + 追问 → 用户回答追问 → 复盘 → 弱项更新
- sidecar：generate_question / evaluate_answer / generate_review 三条 AI 链路
- 评分维度：准确性、完整性、落地感、表达清晰度、抗追问能力
- 前端 TrainView（模式/主题/强度选择）→ SessionView（问答交互 + 评估反馈）→ ReviewView（复盘卡展示）

### Phase 4 - weakness memory 与 dashboard 推荐 ✅

- weakness upsert：新弱项插入，已有弱项 severity 按 `current + hit × 0.35` 递增（上限 1.5），frequency +1
- weakness relieve：评分 ≥ 75 时相关弱项 severity 下调 0.18
- dashboard 聚合：Top 5 weakness、最近训练记录、今日建议、推荐专项、投递倒计时
- 推荐逻辑：按 severity → frequency → last_seen_at 排序，topic/project/expression 分类推荐

### Phase 5 - 项目训练闭环 ✅

- 训练配置页支持 project 模式 + 项目选择
- 出题时通过 FTS5 检索项目 chunk 作为上下文（默认 6 个）
- 追问围绕项目 followup_points + summary 展开
- 复盘卡覆盖项目表达维度
- project 类 weakness 可沉淀

---

## 8. 待完成阶段

代码结构已就绪，核心差距是：从未在真实 LLM 环境下跑通完整流程。

### Phase 6 - 端到端验证 ⬜ 当前优先

#### 目标

配置真实 LLM，从零跑通完整用户旅程，修复所有阻断问题。

#### 具体任务

1. **LLM 接入验证**：配置 `.env` 中的 MODEL / BASE_URL / API_KEY，确认 sidecar 4 个端点都能正常返回结构化 JSON
2. **画像流程验证**：填写真实画像 → 保存 → 刷新回显 → dashboard 显示画像摘要
3. **项目导入验证**：导入一个真实公开仓库 → 查看生成的画像草稿 → 编辑并保存
4. **基础知识训练验证**：选 Go 主题 → 回答主问题 → 查看评分和追问 → 回答追问 → 查看复盘卡
5. **项目训练验证**：选已导入项目 → 回答主问题 → 回答追问 → 查看复盘卡
6. **记忆闭环验证**：做 2-3 轮训练后，首页推荐是否反映真实弱项

#### 验证用例

使用真实数据：

- 画像：Go 后端 / Agent 方向、实习前、主讲 Mirror / OfferPilot / SneakerFlash
- 项目导入：优先用中小型仓库
- 训练：故意答得一般，观察追问是否咬住薄弱点；故意答得好，观察 severity 是否下降

#### 完成标准

一个新用户从零开始，能无障碍地完成：画像 → 导入项目 → 基础训练 → 项目训练 → 查看复盘 → 首页推荐反映弱项。

---

### Phase 7 - 错误处理与健壮性 ⬜

#### 目标

从"能演示"变成"真的能日常用"。

#### 前端

- API 失败时的错误提示（当前 mutation 没有 onError 处理）
- 页面初次加载的 loading 状态（当前直接显示空）
- 导入中的进度反馈（仓库克隆 + LLM 分析耗时较长）
- 训练提交中的等待状态优化（LLM 评估可能需要数秒）

#### 后端

- sidecar 调用失败时的统一错误包装（当前直接返回 502）
- LLM 超时的友好提示
- session 状态非法流转保护

#### sidecar

- LLM 返回非 JSON 时的重试逻辑（当前只有 tool-loop → single-shot 两级降级）
- 仓库克隆失败（不存在 / 私有 / 网络问题）时的明确错误信息

#### 完成标准

- 任意一次 sidecar 失败不会导致前端空白
- 普通错误能定位到前端 / Go / sidecar 哪一层
- 连续使用一周不想砸掉它

---

### Phase 8 - 训练质量调优 ⬜

#### 目标

让训练真的有用，而不只是结构正确。

#### 具体任务

- 扩充种子题目模板（当前每主题仅 1 个，需要至少 5-8 个覆盖不同知识点）
- 调优 system prompt（出题不能太泛、评分不能太宽、追问要咬住弱点）
- 项目训练的追问质量验证（追问是否围绕项目资产，而不是泛泛而谈）
- 复盘卡内容质量验证（是否具体、可执行，而不是客服式总结）
- weakness 命中逻辑验证（severity 升降幅度是否合理，推荐是否稳定）

#### 验证方法

- 故意答得一般，观察追问是否会咬住薄弱点
- 故意答得较好，观察 weakness 是否不过度放大
- 连续 5 轮训练后，首页推荐是否准确反映弱项变化

#### 完成标准

- 问题不像百科问答，而是会让人卡壳
- 复盘不像客服总结，而是能指出具体改进方向
- weakness 推荐稳定，不会反复抖动

---

## 9. 推荐执行顺序

```
Phase 6（端到端验证）→ Phase 7（错误处理）→ Phase 8（训练质量）
```

不要跳过 Phase 6 直接做 Phase 7 或 8。原因：不先跑通完整流程，就不知道真正的阻断点在哪里，后续修复会变成盲打。

---

## 10. 开发禁止事项

为了避免项目被做散，开发过程中默认禁止：

- 一边做 v0 一边加算法模块
- 一边做训练链路一边做多用户系统
- 为了更像 agent 而先堆多 agent 图
- 为了更像产品先做复杂视觉包装
- 为了更聪明引入向量库 / RAG / 长上下文而不解决核心训练质量

默认优先级永远是：

**真实训练闭环 > 记忆闭环 > 工程稳定 > 新功能扩张**

---

## 11. 文档关系

- `docs/PRD.md`：回答做什么（产品边界）
- `docs/ARCHITECTURE.md`：回答系统怎么分层（技术边界）
- `docs/PLAN.md`：回答当前阶段按什么主线推进
- `docs/ROADMAP.md`：回答详细按什么顺序开发，以及每一步怎么验收

优先级：PRD 的产品边界 > ROADMAP 的开发顺序 > ARCHITECTURE 的技术边界。
