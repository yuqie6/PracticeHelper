# 阶段 B - JD 岗位视角训练方案

> 状态：已完成。本文保留为阶段 B 的设计记录，同时同步 2026-03-21 的实现口径。

## 1. 背景

阶段 A 已经解决了“这套训练能不能用、用起来顺不顺”的问题。

当前最大的主线问题已经变成：

> 训练结果是不是和目标岗位真正看重的能力对齐。

如果继续只围绕 `topic / project / weakness` 出题和评分，用户能得到“我哪里答得差”，但还得不到“为什么这个岗位会觉得你不够好”。阶段 B 的目标就是把这层岗位视角真正接进训练主链路。

---

## 2. 第一版目标

第一版只做一个最小闭环：

1. 用户可以在独立 JD 页面维护多份 JD
2. 系统可以把每份 JD 分析成结构化岗位要求，并保留完整分析历史
3. 开始训练前，用户明确选择本轮参考的 JD
4. 一轮训练固定绑定这份 JD 的最新成功分析快照
5. basics / project 的出题和评分都引用这个岗位要求
6. review 也开始从岗位视角指出缺口和下一轮建议

### 第一版不做

- JD 自动抓取
- 多岗位对比推荐
- 简历联动
- 一轮训练同时绑定多份 JD
- 训练时手动选择某次历史分析快照
- 岗位要求可视化大盘

### 当前已经落地

- 独立 `job-targets` 页面、多 JD 列表、原文编辑、分析历史和默认 JD 激活已经可用
- `job_targets` / `job_target_analysis_runs` / `training_sessions.job_target_id` / `job_target_analysis_id` 都已经落库
- `TrainView` 已支持显式选择 JD；`idle / running / failed / stale` 都会阻止新训练绑定并显示原因
- 未显式选择 JD 时，只有 `succeeded` 的默认 JD 才会自动带入训练和 dashboard 推荐；不可用时会回退到 generic
- basics / project 的出题、评分、追问和复盘都已读取 JD 分析快照
- session / review / dashboard 都能回看绑定的 JD；`latest_successful_analysis` 会在 `stale` / `failed` 后保留只读回看
- `scripts/e2e_live.py` 已覆盖 success -> bind -> stale fallback 的真实主链路

---

## 3. 用户流程

### 3.1 JD 页面

1. 用户进入独立岗位页 `/job-targets`
2. 新建一份 JD，填写：
   - `title`：用户自定义标题，例如“后端工程师 - 某创业公司”
   - `company_name`：可选
   - `source_text`：原始 JD 文本
3. 点击保存，只保存原文，不自动分析
4. 点击“开始分析”或“重新分析”，系统基于当前原文创建一次新的分析快照
5. 页面展示：
   - 左侧 JD 列表
   - 右侧当前 JD 原文
   - 最新成功分析结果
   - 历史分析记录（成功 / 失败都保留）

### 3.2 训练入口

1. 用户进入 `/train`
2. 先选训练模式、强度、主题 / 项目
3. 再选本轮参考的 JD
4. 允许“不选 JD”直接开始，保持当前通用训练路径
5. 如果选择了 JD，则必须满足：
   - 该 JD 有至少一次成功分析
   - 且当前原文没有在最近一次成功分析后继续被修改
6. 满足后才能开始训练

### 3.3 训练过程

1. `CreateSession` 时把 `job_target_id` 带给后端
2. 后端解析出该 JD **最新成功分析快照**
3. session 固定写入：
   - `job_target_id`
   - `job_target_analysis_id`
4. 后续这一轮里的：
   - 出题
   - 评分
   - 追问
   - 复盘
   都只吃这个绑定快照
5. 中途就算用户回 JD 页面重新分析别的内容，也不会影响已经开始的训练

---

## 4. 数据模型

### 4.1 `job_targets`

表示“一份 JD 的当前实体”。

建议字段：

- `id`
- `title`
- `company_name`
- `source_text`
- `latest_analysis_id`
- `latest_analysis_status`
  - `idle`
  - `running`
  - `succeeded`
  - `failed`
  - `stale`
- `last_used_at`
- `created_at`
- `updated_at`

### 语义约束

- 保存原文不会自动触发分析
- 只要 `source_text` 在最近一次成功分析后被修改，`latest_analysis_status` 就必须变成 `stale`
- `stale` 状态下，这份 JD 不能再用于新训练，必须重新分析

### 4.2 `job_target_analysis_runs`

表示“某份 JD 的一次分析快照”。

建议字段：

- `id`
- `job_target_id`
- `source_text_snapshot`
- `status`
  - `running`
  - `succeeded`
  - `failed`
- `error_message`
- `summary`
- `must_have_skills_json`
- `bonus_skills_json`
- `responsibilities_json`
- `evaluation_focus_json`
- `created_at`
- `finished_at`

### 语义约束

- 每次点击“开始分析 / 重新分析”都创建一条新记录
- 历史记录不覆盖，失败记录也保留
- 训练默认吃“最新成功快照”，不是最新一次分析尝试
- 但如果当前 `job_targets.source_text` 已经修改过且状态为 `stale`，则不能继续复用旧成功快照开始训练

### 4.3 `training_sessions`

新增字段：

- `job_target_id`
- `job_target_analysis_id`

### 绑定规则

- 如果本轮没选 JD，这两个字段都为空
- 如果选了 JD，则两个字段在 `CreateSession` 成功时就固定写入
- 后续 `GetSession` / `GetReview` / recent session summary 都应该能回看这两个字段对应的信息

---

## 5. 页面与交互

### 5.1 新页面：`/job-targets`

页面结构：

- 左栏：JD 列表
  - 标题
  - 公司名
  - 最新状态
  - 最近分析时间
  - 最近使用时间
- 右栏：当前 JD 详情
  - 原文编辑区
  - 保存按钮
  - 分析 / 重新分析按钮
  - 最新成功分析结果
  - 历史分析列表

交互规则：

- 保存原文成功后，只更新 JD 本体，不自动分析
- 点击分析后，按钮进入 loading，历史列表新增一条 `running`
- 成功后右栏切到最新成功分析结果
- 失败后保留错误信息，但不删旧成功快照

### 5.2 训练配置页：`/train`

新增一个 JD 选择区，位于模式 / 主题 / 项目选择之后、开始按钮之前。

交互规则：

- `basics` 和 `project` 都可以选 JD
- 默认选项是“不使用 JD（通用训练）”
- 如果用户选中一个 `idle / failed / stale / running` 的 JD：
  - 开始按钮禁用
  - 文案明确提示“这份 JD 还没有可用分析结果”
- 如果用户选中一个 `succeeded` 的 JD：
  - 可开始训练

### 5.3 Session / Review 页面

第一版需要在页面头部显示：

- 本轮是否绑定 JD
- 绑定的是哪份 JD 标题

不需要第一版就做复杂的“岗位要求对照表”，但至少要让用户知道：

- 这轮是按哪份 JD 练的
- 当前反馈为什么会出现岗位相关措辞

---

## 6. API 方案

### 6.1 Job Target API

### `GET /api/job-targets`

返回列表项，至少包含：

- `id`
- `title`
- `company_name`
- `latest_analysis_id`
- `latest_analysis_status`
- `updated_at`
- `last_used_at`

### `POST /api/job-targets`

创建 JD。

请求体至少包含：

- `title`
- `company_name`
- `source_text`

### `GET /api/job-targets/:id`

返回单个 JD 详情，至少包含：

- `id`
- `title`
- `company_name`
- `source_text`
- `latest_analysis_id`
- `latest_analysis_status`
- `latest_successful_analysis`（可直接内嵌，减少页面二次请求）

### `PATCH /api/job-targets/:id`

更新 JD 原文或元数据。

规则：

- 原文发生变化时，把 `latest_analysis_status` 置为 `stale`
- 不自动触发分析

### `POST /api/job-targets/:id/analyze`

创建一次新的分析快照。

规则：

1. 先创建 `job_target_analysis_runs` 记录，状态为 `running`
2. 调 sidecar 的 `analyze_job_target`
3. 成功则写入结构化字段并标记 `succeeded`
4. 失败则写入 `error_message` 并标记 `failed`
5. 同步更新 `job_targets.latest_analysis_id` 与 `latest_analysis_status`

### `GET /api/job-targets/:id/analysis-runs`

返回该 JD 的分析历史，按时间倒序。

### `GET /api/job-targets/analysis-runs/:id`

返回某次分析快照详情。

### 6.2 训练接口变更

### `POST /api/sessions`

### `POST /api/sessions/stream`

请求体新增可选字段：

- `job_target_id`

服务端行为：

- 为空：保持当前通用训练逻辑
- 非空：
  - 校验 JD 是否存在
  - 校验是否有可用成功快照
  - 解析出最新成功快照
  - 写入 session 的 `job_target_id` / `job_target_analysis_id`

### 错误语义

- `job_target_not_found`
- `job_target_not_ready`
  - 包含 `idle / running / failed / stale / 无成功快照` 这几类场景

---

## 7. Sidecar 扩展

新增一条内部能力：

- `analyze_job_target`

输入：

- JD 原文

输出结构：

- `summary`
- `must_have_skills`
- `bonus_skills`
- `responsibilities`
- `evaluation_focus`

### Prompt 目标

- 不要把 JD 改写成简历建议
- 只提取“岗位实际看重的能力要求”
- 证据不足时保守表达，不要脑补团队规模、技术栈细节或真实业务背景

---

## 8. 训练链路变化

### 8.1 出题

### basics

问题生成优先考虑：

1. 当前 topic
2. 历史弱项
3. 所选 JD 的 `must_have_skills / evaluation_focus`

预期效果：

- 不再只是“随机 Redis 题”
- 而是“这个岗位要求缓存一致性 / 高并发排障 / 分布式经验时，Redis 该怎么练”

### project

问题生成优先考虑：

1. 项目画像
2. 项目上下文 chunk
3. 所选 JD 的职责和能力要求

预期效果：

- 不再只问“你项目里怎么做”
- 而是“这个岗位看重什么，而你的项目里哪一段能证明这一点”

### 8.2 评分与追问

评分时必须能回答两件事：

1. 你的回答本身是否成立
2. 这个岗位要求的关键能力，你有没有体现出来

追问时必须围绕：

- 这次回答暴露的缺口
- JD 真正在意的能力要求

不能继续只给泛化反馈。

### 8.3 复盘

`top_fix` / `gaps` / `recommended_next` 都要能带岗位视角。

期望出现的表达方式是：

- “这个岗位看重故障排查闭环，但你的回答只停在结论”
- “你的项目经历可以支撑缓存一致性这个点，但当前表达还没落到具体取舍”

而不是只说：

- “回答不够完整”

---

## 9. 当前验收现状

当前已经能跑通的主链路是：

1. 新建 JD -> 分析成功 -> 激活为默认 JD
2. 在 TrainView 显式或隐式绑定这份 JD 开启 basics / project 训练
3. session / review 固定回看 `job_target_id` / `job_target_analysis_id`
4. dashboard 的 `recommendation_scope` 在 JD ready 时切到 `job_target`
5. 修改原文后 JD 变成 `stale`，新训练会被 `job_target_not_ready` 阻止，但旧成功快照仍可回看
6. `scripts/e2e_live.py` 已覆盖这条成功链路和 `stale` 回退语义

仍需继续补的主要是回归层验证，不再是阶段 B 的功能缺口：

- `running / failed` 的 live UI smoke 还没有打包成同一条真实 E2E 脚本
- 这两类状态的不可绑定语义、快照保留和提示文案，目前主要由 repo / service / web tests 保证

---

## 10. 默认实现假设

这份方案里的以下决策，已经成为当前实现边界：

- JD 走独立页面，不进画像页
- 支持多 JD
- 保留完整分析历史
- 训练前手动选择 JD
- 一轮训练默认绑定所选 JD 的最新成功分析快照
- 如果当前原文已改但未重新分析，状态为 `stale`，不允许继续开这份 JD 的训练
- 没有选择 JD 时，保持当前通用训练链路
