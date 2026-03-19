# ARCHITECTURE - PracticeHelper

## 1. 设计目标

PracticeHelper 的架构优先支持：

- 快速迭代 MVP
- 单用户自用场景
- 训练记录与薄弱点可持续积累
- 后续逐步扩展到模拟面试 / 算法训练 / JD 定制训练

## 2. 总体技术栈

### 2.1 前端

- `Vue 3`
- `Vite`
- `TypeScript`
- `pnpm`
- `Vue Router`
- `@tanstack/vue-query`
- `Tailwind CSS`

### 2.2 Go 服务

- `Gin`
- `database/sql`
- `SQLite`
- `FTS5`
- `golangci-lint`

### 2.3 Python sidecar

- `FastAPI`
- `LangGraph`
- `Pydantic`
- `Ruff`

## 3. 架构分层

### 3.1 Web

负责：

- 页面路由
- 用户输入
- 训练过程展示
- 首页与复盘展示

### 3.2 Go API

负责：

- 提供前端 API
- 管理 SQLite 持久化
- 管理训练会话状态机
- 调用 Python sidecar
- 汇总弱项与推荐结果

### 3.3 Python sidecar

负责：

- GitHub 仓库分析
- 训练问题生成
- 回答评估
- 复盘生成

sidecar 对外只暴露 4 个内部接口：

- `analyze_repo`
- `generate_question`
- `evaluate_answer`
- `generate_review`

## 4. 模块草图

### 4.1 Profile 模块

负责：

- 用户画像初始化
- 求职方向、项目列表、目标岗位维护

### 4.2 Project Import 模块

负责：

- 仓库抓取
- 文本文件过滤
- chunk 切分
- 项目画像草稿生成

### 4.3 Training Engine 模块

负责：

- 生成训练流程
- 提问、追问
- 控制一轮训练的状态流转

### 4.4 Evaluation 模块

负责：

- 对回答进行评分
- 识别回答中的漏洞
- 提炼薄弱点标签
- 输出训练复盘卡

### 4.5 Memory / Review 模块

负责：

- 存储历史训练记录
- 汇总薄弱点
- 给出下一轮训练建议

## 5. 数据对象

核心对象至少包括：

- `user_profile`
- `project_profile`
- `repo_chunk`
- `question_template`
- `training_session`
- `training_turn`
- `review_card`
- `weakness_tag`

## 6. API 边界

最小 API 集合固定为：

- `POST /api/profile`
- `GET /api/profile`
- `PATCH /api/profile`
- `POST /api/projects/import`
- `GET /api/projects`
- `GET /api/projects/:id`
- `PATCH /api/projects/:id`
- `POST /api/sessions`
- `POST /api/sessions/:id/answer`
- `GET /api/sessions/:id`
- `GET /api/reviews/:id`
- `GET /api/weaknesses`

## 7. 核心流程

### 7.1 八股训练流程

1. 选择训练主题
2. Go 服务准备上下文与题库模板
3. sidecar 生成主问题
4. 用户回答
5. sidecar 评分并生成追问
6. sidecar 输出复盘卡
7. Go 服务更新弱项

### 7.2 项目训练流程

1. 选择项目
2. Go 服务检索 `project_profile` 和 `repo_chunks`
3. sidecar 基于检索上下文生成问题
4. 用户作答
5. sidecar 围绕技术与 trade-off 追问
6. sidecar 输出项目表达复盘
7. Go 服务更新项目维度弱项

### 7.3 项目导入流程

1. 用户输入 GitHub 仓库 URL
2. Go 服务校验并调用 `analyze_repo`
3. Python sidecar 拉取仓库并扫描白名单文本
4. sidecar 提取项目摘要、亮点、风险点和可追问点
5. Go 服务持久化 `project_profile` 和 `repo_chunks`
6. 用户确认或编辑项目画像

## 8. 检索策略

训练时实时检索只从已导入的 `repo_chunks` 和 `project_profile` 中取上下文：

- 使用 `SQLite FTS5`
- 对文档文件、架构文件、核心源码路径给予更高权重
- 不引入单独向量库
- 不提供独立 repo chat 能力

## 9. LangGraph 使用边界

v0 使用 LangGraph，但保持克制：

- 不做复杂多 agent 协作
- 不做长链路自治
- 只把 LangGraph 用在可控的训练编排与仓库分析流程

建议拆成 4 条受控 graph：

- `analyze_repo_graph`
- `generate_question_graph`
- `evaluate_answer_graph`
- `generate_review_graph`

每条 graph 都以结构化状态输入输出，并由 Pydantic 做 schema 校验。

## 10. 训练状态机

训练状态机固定为：

`draft -> active -> waiting_answer -> evaluating -> followup -> review_pending -> completed`

每轮训练默认 2 段式：

- 主问题 1 个
- 追问 1 到 2 个

结束后立刻生成复盘卡，不做无限对话。

## 11. 前端设计约束

前端统一采用 `neo-brutalist` 风格：

- 粗边框
- 硬阴影
- 无圆角
- 高对比
- 黑白主色加亮色强调

明确避免：

- 圆角
- 渐变
- 灰色边框

## 12. 工程原则

- **先单体，后拆分**：v0 不需要微服务
- **先本地存储，后远程化**：单用户场景优先 SQLite
- **先结构清晰，后功能扩张**：模块边界清楚，但不过度抽象
- **先把 LLM 用在最值钱的地方**：提问、追问、评估、复盘
- **先让 fallback 可用**：LLM 不可用时也要有基础兜底结果
