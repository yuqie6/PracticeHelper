# PracticeHelper

PracticeHelper 是一个面向 **后端 / Agent 方向求职者** 的面试训练 Agent。

它的目标不是做一个“会背答案的八股题库”，而是做一个 **真的能把人练到更会面试** 的训练搭子。

## 项目定位

PracticeHelper 聚焦三件事：

1. **八股训练**：Go / MySQL / Redis / Kafka / 网络 / 操作系统 / Agent / RAG 等。
2. **项目面试训练**：围绕真实项目经历做追问、纠偏和表达打磨。
3. **模拟面试与复盘**：记录薄弱点，形成持续训练闭环。

## 核心理念

- 不做单纯题库 bot
- 不做只会给标准答案的问答壳
- 要做“会诊断你哪里虚、然后继续追问”的训练 Agent
- 默认服务单人自用（先把自己练强），后续再考虑泛化

## v0 范围

第一版只做最关键的训练闭环：

- 用户画像初始化
- 八股训练
- 项目训练
- 训练后复盘卡
- 薄弱点记录

v0 明确不做：

- 在线代码执行 / 判题系统
- 多用户 SaaS
- 复杂权限系统
- 通用型 Repo Chat
- 全真模拟面试
- 过度工程化的多 agent 编排

## 技术栈

当前已拍板的技术栈如下：

- 前端：`Vue 3 + Vite + TypeScript + pnpm + Vue Router + @tanstack/vue-query + Tailwind CSS`
- 前端风格：`新野兽派（neo-brutalist）`
- Go 后端：`Gin + SQLite + FTS5 + golangci-lint`
- Python sidecar：`FastAPI + LangGraph + Pydantic + Ruff`
- 训练链路：Go 负责 API / 数据 / 状态机，Python 负责仓库分析、出题、评估、复盘
- 检索方案：只基于 `project_profile` 和 `repo_chunks` 做受控检索，不单独引入向量库

## 前端风格

前端统一采用 `neo-brutalist` 风格：

- 粗边框
- 硬阴影
- 无圆角
- 高对比
- 功能主义
- expressive

设计约束：

- 优先使用纯黑边框
- 使用硬边缘阴影
- 统一保持直角
- 以黑白为主，配鲜艳强调色
- 不用圆角
- 不用渐变
- 不用灰色边框

## 推荐目录结构

```text
practicehelper/
  README.md
  PRD.md
  PLAN.md
  ARCHITECTURE.md
  Makefile
  .gitignore
  pnpm-workspace.yaml
  styles/
    neo-brutalist/
  web/
  server/
  sidecar/
  docs/
  scripts/
```

## 当前状态

- [x] 项目目录已创建
- [x] 项目文档初始化
- [x] 技术栈拍板
- [x] MVP 边界确认
- [ ] 项目骨架初始化
- [ ] 三端最小联通

## 本地开发

计划提供统一入口：

- `make web-dev`
- `make server-dev`
- `make sidecar-dev`
- `make lint`
- `make test`

## 下一步

1. 起 Vue / Gin / FastAPI 三端骨架
2. 接入 SQLite 与基础数据模型
3. 跑通画像初始化和首页空态
4. 跑通项目导入与第一轮训练
