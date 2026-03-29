# NORTH STAR - PracticeHelper

> 状态：2026-03-26 起作为长期演化锚点。
> 这份文档不回答“这周做什么”，只回答“未来 6-12 个月它应该长成什么”。

## 1. 一句话

PracticeHelper 的长期目标，不是做一个会出题的 AI 页面，
而是做成一个 **单用户的 Interview Growth System**：
它能持续理解用户、组织训练、引用真实证据、追踪成长，并把一次次训练沉淀成长期复利。

## 2. 最终想回答的 5 个问题

当系统成熟以后，它应该能稳定回答：

1. **我现在最该练什么？**
2. **为什么是这个，不是别的？**
3. **我刚才到底哪里答得不行？**
4. **我有哪些真实项目证据能证明自己的能力？**
5. **我这一个月到底有没有变强？**

如果这 5 个问题都回答不好，就算功能很多，也不是成熟的面试训练系统。

## 3. 长期形态

长期看，PracticeHelper 应该是一个：

- 以训练为中枢
- 以证据为支撑
- 以记忆为积累
- 以成长为目标

的个人面试训练工作台。

它不是：
- chat-first 陪聊产品
- 通用求职平台
- 静态题库网站
- 为了炫技而拼 agent 概念的实验场

## 4. 四层能力栈

### 4.1 训练引擎层（永远的心脏）

这一层负责“怎么练”。

长期必须稳定成熟的能力包括：
- session / turn / review 主链路
- 问题生成与追问策略
- 回答评估与结构化反馈
- weakness 更新
- 推荐下一轮训练

成熟标志：
- 训练链路稳定、可解释、可恢复
- 不同模式下（basics / project / JD）都能形成连续训练
- 训练结束后总能落到可行动的 review，而不是停在一段模型输出上

### 4.2 证据与上下文层（把“会说”变成“有凭据地说”）

这一层负责“为什么这么问、为什么这么评”。

长期必须成熟的能力包括：
- JD 理解与岗位要求抽取
- 项目画像与 repo chunk 检索
- observation / session summary / knowledge graph
- retrieval trace
- evidence binding

成熟标志：
- review / 推荐 / 追问里的判断能引用更清楚的证据来源
- 用户能看到“为什么问这个、为什么这么判、证据来自哪里”
- 项目训练不只依赖摘要，而能逐步借助真实代码与项目证据支撑表达训练

### 4.3 成长与规划层（让训练形成长期复利）

这一层负责“练完之后怎么继续变强”。

长期必须成熟的能力包括：
- weakness trend
- spaced repetition
- capability map
- weekly focus / learning path
- progress narrative

成熟标志：
- 系统不只知道用户这轮答得差，还知道他最近一直卡在哪
- 推荐不只是展示文案，而能组织阶段性的训练路径
- 用户能从系统里看到自己在一段时间内的真实变化

### 4.4 交互工作台层（把前三层组织成可持续使用的产品）

这一层负责“用户怎么用前面三层”。

长期必须成熟的工作面包括：
- Diagnose / Today / Recommendation
- Prep（岗位 / 项目准备）
- Train
- Review
- Memory / Growth
- Prompt / Audit（偏内部和高级用户）
- 受控辅助 chat

成熟标志：
- 训练页仍然是主交易面，但其他工作面都已经围绕训练组织得足够清楚
- chat 是解释层和辅助层，而不是主入口
- 用户不会把系统当成单页工具，而会把它当成自己的训练工作台

## 5. 演化阶段（长期版）

### α. Training Engine 成立

标志：
- 训练主链路稳定
- 多轮问答、review、weakness、推荐跑顺
- basics / project / JD 三条训练线都成立

### β. Evidence System 成立

标志：
- 检索与证据绑定更可信
- 项目证据不只是摘要，而能更稳定支撑追问与 review
- why-this-question / why-this-feedback 有解释基础

### γ. Growth System 成立

标志：
- memory 从“记录过什么”升级到“理解这个人怎么在成长”
- 推荐从“下一轮练什么”升级到“下一阶段该怎么练”
- 弱项、趋势、复习、规划形成长期闭环

### δ. Workspace 成熟

标志：
- Home / Diagnose / Review / Memory / Prep 都形成稳定工作面
- 训练仍是主交易面，但不再是唯一有价值的入口
- chat 成为贯穿式辅助层，而不是主入口

### ε. 高级 Agent 化

标志：
- typed command、retrieval orchestration、context compaction 足够成熟
- 某些高价值长任务才开始考虑多 agent
- agent 复杂度服务训练价值，而不是反过来主导产品方向

## 6. 长期不变的边界

哪怕以后能力变强，这些边界也尽量不变：

- 单用户优先，不先按 SaaS 思路设计
- 训练仍然是系统心脏，不退回 chat-first
- Go 保留产品边界与持久化边界
- sidecar 负责受约束的智能，不直接接管数据库
- 不为了“更大”而把系统做成泛求职平台

## 7. 和当前文档的关系

- [VISION.md](./VISION.md)：说当前系统现在是什么、当前形态约束是什么
- [ROADMAP.md](./ROADMAP.md)：说阶段顺序和为什么按这个顺序推进
- [PLAN.md](./PLAN.md)：说这轮执行具体做什么
- [ARCHITECTURE.md](./ARCHITECTURE.md)：说当前技术边界
- `NORTH_STAR.md`：说长期 6-12 个月应该长成什么

如果当前执行和长期蓝图冲突：
- 当前是否要做，以 [PLAN.md](./PLAN.md) 为准
- 长期是否值得往那里演化，以 `NORTH_STAR.md` 为准
