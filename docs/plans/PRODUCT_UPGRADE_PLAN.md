# PracticeHelper 产品全面升级计划

> 状态：已按 2026-03-26 当前工作区重写为执行清单。
> 这份文档只看阶段 C，不重复讲阶段 A / B 的历史。

## 1. 怎么看这份文档

状态说明：

- `✅ 已实现`：主链路已经在代码里成立
- `🟡 收口中`：第一版已接上，但还没到可以完全放心的程度
- `⬜ 未开始`：当前仓库还没有对应实现

如果文档和代码表述冲突，默认先按这里的当前进度判断范围。

## 2. 当前判断

PracticeHelper 现在缺的不是“有没有功能”，而是“已有闭环够不够深、够不够连续、够不够可审计”。

阶段 C 的目标因此固定为：

- 训练更深
- 留存更强
- 检索更准
- 审计更清楚

## 3. 当前状态总表

| 项目 | 优先级 | 状态 | 当前判断 |
|:-----|:-------|:-----|:---------|
| 多轮训练流 | P0 | ✅ | `max_turns`、独立 turn 和循环 FSM 已成立 |
| 题库扩展 | P0 | ✅ | 10 个 topic + `mixed` 已成立 |
| 会话历史分页 | P0 | ✅ | `/history`、分页、筛选已成立 |
| 弱项趋势可视化 | P0 | ✅ | snapshot、趋势接口、首页 sparkline 已成立 |
| LangGraph 多节点 | P1 | ✅ | 图层已薄壳化，runtime 承担主要编排 |
| 间隔重复 | P1 | ✅ | `review_schedule`、到期展示和完成推进已成立 |
| 引导式 onboarding | P1 | ✅ | 首访 stepper 已接通 |
| 自适应难度 | P1 | ✅ | `intensity=auto` 已接上前后端 |
| 评估审计追踪 | P1 | ✅ | `evaluation_logs`、Prompt 实验页、Review 审计已成立 |
| 训练历史自助删除 | P1 | ✅ | History 页已支持单条 / 批量删除，且会同步清理 review、审计、记忆、复习计划与索引，并回滚受影响聚合 |
| 前端体验增强 | P2 | ✅ | 页面过渡、快捷提交、暗色主题等已收口 |
| 数据导出 | P2 | ✅ | 单次导出与 History 批量 ZIP 已成立 |
| Prompt 版本管理 | P2 | ✅ | prompt set、版本绑定、实验对比已成立 |
| RAG 升级 | P2 | 🟡 | 第一版已落地，但证据绑定和召回质量还没收口 |

## 4. 现在最该继续推进的项

### 4.1 RAG 升级收口

已经有的：

- observations / session summaries 已进入 embedding + 混合召回链路
- repo chunk 已支持 `Qdrant vector recall + optional rerank + FTS5 fallback`
- 新项目导入后会进入 repo chunk embedding 队列
- sidecar 的 LLM / embedding / rerank 配置面已经统一

还差的：

- 结果里的证据绑定更清楚
- 更大范围召回是否值得引入，要基于真实效果而不是先拍脑袋
- review / 推荐对检索结果的引用还要更稳定、更可解释

### 4.2 Prompt 实验深化

已经有的：

- prompt set registry
- 训练页选版本
- session 绑定 prompt set
- Prompt 实验页的基础对比与审计明细

还差的：

- 更容易比较不同 prompt set 的真实效果
- 更清楚地区分“版本差异”和“数据波动”
- 保持前端只做受控 overlay，不把完整 prompt 编辑重新扩成产品主线

### 4.3 推荐与复习闭环继续做实

已经有的：

- 弱项衰减、趋势、待复习入口、推荐下一轮
- 首页 `today_focus` / `recommended_track`
- review 完成后推进下一次复习时间
- History 页删除已经补到“彻底遗忘”语义：删除 session 时会一起清理 review、
  `evaluation_logs`、session memory、observation、复习计划、memory index / embedding，
  并按剩余快照回滚 weakness / knowledge 聚合

还差的：

- 推荐理由更稳定
- 推荐真正带动下一轮训练，而不是只停留在展示层
- 连续使用场景下的回归验证继续补齐

## 5. 当前不建议回头重做的项

这些能力已经有了，不该再被写成“本轮主线”：

- 多轮训练流
- 历史分页
- 训练历史自助删除
- 弱项趋势可视化
- onboarding
- 自适应难度
- 单次 / 批量导出
- Prompt 版本管理当前阶段目标

## 6. 明确不在当前计划内

- chat-first 首页
- 多 agent 协作
- 通用向量数据库平台化能力
- 前端完整编辑核心 prompt
- 简历联动、JD 抓取、多岗位对比
- 通用 repo chat
