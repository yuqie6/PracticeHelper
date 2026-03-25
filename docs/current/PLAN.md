# 当前执行计划 - PracticeHelper

> 状态：已按 2026-03-26 当前工作区收口。

## 1. 当前结论

- 阶段 A、阶段 B 已完成，当前主线是阶段 C。
- 工程收口专项单独看
  [ARCHITECTURE_CONVERGENCE_PLAN.md](../plans/ARCHITECTURE_CONVERGENCE_PLAN.md)。
- sidecar agent runtime 深改单独看
  [AGENT_DEEP_REDESIGN_PLAN.md](../plans/AGENT_DEEP_REDESIGN_PLAN.md)。
- 这里不再重复讲已完成阶段的大段历史，只保留“现在该做什么”。

## 2. 当前已经具备的底座

当前代码已经不是早期 demo，下面这些底座都已经成立：

- 用户画像、首页 dashboard、项目导入、岗位 JD 管理
- basics / project 多轮训练闭环
- 复盘卡、弱项记忆、待复习和推荐下一轮
- History 页的批量导出与自助删除；删除 session 时会同步清理 review、
  审计日志、session memory、observation、复习计划和对应检索索引，并回滚受影响的
  weakness / knowledge 聚合
- prompt set、prompt 偏好、prompt 实验与审计
- 第一版检索升级，包括 memory embedding、repo chunk embedding、
  optional rerank 和 Qdrant + FTS5 fallback

## 3. 阶段 C 现在该做什么

### 3.1 收口第一版检索 / RAG

重点不是“再讲一遍已经接上了向量检索”，而是把下面几件事补扎实：

- 证据绑定更清楚
- 召回质量更稳定
- review / 推荐里能更可信地引用检索结果

### 3.2 继续做实 Prompt 实验

- 让 prompt set 的差异更容易比较
- 让实验结果更容易回看和复盘
- 保持前端只做受控 overlay，不把完整 prompt 编辑重新扩成主线

### 3.3 强化推荐和复习闭环

- 首页建议、待复习、历史表现要更稳定地互相联动
- 推荐不只是展示一条文案，而是要能推动下一轮训练发生
- 历史删除已经补到“彻底遗忘”语义，后续这一条线不再回头补基础删除能力，
  而是看删除后的推荐、趋势和待复习是否持续稳定

### 3.4 继续补回归验证

- 主链路需要继续做 live / regression 验证
- 避免“功能看起来都在，但组合起来又漂了”

## 4. 当前完成标准

阶段 C 可以认为收口，至少要满足：

- 当前主线在代码、页面和文档里说的是同一件事
- 检索、推荐、复习、prompt 实验不再只是“第一版能跑”
- 连续使用时，用户能感到训练更深、更连续，而不是只是功能更多

## 5. 当前不该继续展开的事

- 多 agent 蓝图落地
- chat-first 首页
- Workspace 级重构
- 再去重写阶段 A / B 已完成的体验问题
- 把主题切换、动效、导出这类已收口项重新包装成当前主线

## 6. 已完成阶段去哪看

- 阶段 A 的体验升级记录：
  [ANSWER_FEEDBACK_UX_V2.md](../records/ANSWER_FEEDBACK_UX_V2.md)
- 阶段 B 的岗位视角记录：
  [JD_TRAINING_STAGE_B.md](../records/JD_TRAINING_STAGE_B.md)
- 阶段 C 的升级盘点：
  [PRODUCT_UPGRADE_PLAN.md](../plans/PRODUCT_UPGRADE_PLAN.md)
