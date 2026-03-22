export const messages = {
  'zh-CN': {
    app: {
      name: 'PracticeHelper',
      title: '面试训练助手',
      language: '语言',
      theme: '主题',
      importNotice: '后台正在导入项目：{stage}',
      locales: {
        zhCN: '中文',
        en: 'English',
      },
      themes: {
        light: '浅色',
        dark: '暗色',
      },
      nav: {
        home: '首页',
        profile: '画像',
        jobs: '岗位',
        projects: '项目',
        promptExperiments: 'Prompt 实验',
        train: '训练',
        history: '历史',
      },
    },
    common: {
      loading: '加载中...',
      requestFailed: '请求失败，请稍后重试。',
      notProvided: '暂未填写',
      unnamedSession: '未命名训练',
      unknownSession: '未命名内容',
      save: '保存',
      saving: '保存中...',
      start: '开始训练',
      starting: '启动中...',
      retry: '重试',
      submit: '提交回答',
      submitting: '提交中...',
      continue: '继续下一轮',
      resume: '继续训练',
      score: '得分',
      status: '当前状态',
      severity: '严重度 {value}',
      daysRemainingLabel: '距离目标投递还有天数',
      lastUpdated: '最近更新于 {value}',
      firstTrainingHint: '完成基础画像后，就可以开始第一轮训练。',
      nextRecommendationHint: '完善画像后，系统会给出下一步建议。',
      noRecommendation: '暂无推荐，先完成第一轮训练。',
      setDeadlineHint: '在画像页设置目标投递时间后，这里会显示剩余天数。',
      exportFormatLabel: '导出格式',
      exportFormats: {
        markdown: 'Markdown',
        json: 'JSON',
        pdf: 'PDF',
      },
      exportSuccess: '导出成功',
      operationSuccess: '操作成功',
      apiErrors: {
        toolContextMissing:
          '系统在拿到必要上下文前就中断了本次生成，请直接重试。',
        toolCallFailed: '系统在调用内部工具时失败了，请稍后重试。',
        toolLoopExhausted: '系统来回推理过多次仍没收口，本次生成已终止。',
        jsonParseFailed: '系统生成了不可解析的结果格式，本次结果已中止。',
        semanticValidationFailed:
          '系统生成的结果没有通过业务校验，请直接重试。',
        singleShotFailed: '回退到单轮生成后仍未成功，请稍后再试。',
        backendCallbackFailed: '系统补取训练材料时失败了，请稍后重试。',
        timeout: '这次请求处理超时了，请直接重试。',
        llmRequired: '当前环境没有可用模型，暂时不能继续生成。',
        unknownError: '这次生成失败了，请稍后重试。',
      },
    },
    enums: {
      mode: {
        basics: '基础训练',
        project: '项目训练',
      },
      intensity: {
        auto: '自动',
        light: '轻量',
        standard: '标准',
        pressure: '强化',
      },
      topic: {
        mixed: '混合专项',
        go: 'Go',
        redis: 'Redis',
        kafka: 'Kafka',
        mysql: 'MySQL',
        system_design: '系统设计',
        distributed: '分布式',
        network: '网络',
        microservice: '微服务',
        os: '操作系统',
        docker_k8s: 'Docker & K8s',
      },
      status: {
        draft: '草稿',
        active: '进行中',
        generating_question: '正在生成问题',
        waiting_answer: '等待回答',
        evaluating: '评估中',
        followup: '追问中',
        review_pending: '生成复盘中',
        completed: '已完成',
      },
      importStatus: {
        ready: '已就绪',
      },
      importJobStatus: {
        queued: '排队中',
        running: '导入中',
        completed: '已完成',
        failed: '失败',
      },
      importJobStage: {
        queued: '等待后台开始',
        analyzing_repository: '分析仓库内容',
        persisting_project: '写入项目材料',
        completed: '导入完成',
        failed: '导入失败',
      },
      jobTargetAnalysisStatus: {
        idle: '待分析',
        running: '分析中',
        succeeded: '可用于训练',
        failed: '分析失败',
        stale: '原文已变更',
      },
      weaknessKind: {
        topic: '知识点',
        project: '项目表达',
        expression: '表达方式',
        followup_breakdown: '追问应对',
        depth: '展开深度',
        detail: '细节支撑',
      },
    },
    jobTargetStatus: {
      homeActive: {
        idle: '当前默认 JD 是 {name}，但还未完成分析；首页推荐和训练仍按通用模式运行。',
        running:
          '当前默认 JD 是 {name}，正在分析中；分析完成前，首页推荐和训练不会自动使用它。',
        succeeded: '当前默认 JD 是 {name}，首页推荐和训练已按岗位要求调整。',
        failed:
          '当前默认 JD 是 {name}，但最近一次分析失败；修正内容或重新分析后才会恢复。',
        stale:
          '当前默认 JD 是 {name}，但原文已变更；旧分析仅供回看，新训练不会使用。',
        unknown:
          '当前默认 JD 暂时无法确认是否可用，首页推荐和训练暂按通用模式运行。',
      },
      trainSelection: {
        idle: '这份 JD 还未分析，暂时无法用于训练。请先在岗位页完成分析。',
        running: '这份 JD 正在分析中，分析完成前无法用于训练。',
        succeeded:
          '本轮将使用这份 JD 当前的分析结果；训练开始后，后续修改不影响本轮。',
        failed: '这份 JD 最近一次分析失败，修正内容或重新分析后才能用于训练。',
        stale: '这份 JD 原文已变更，旧分析仅供回看；重新分析后才能用于训练。',
        unknown: '当前无法确认这份 JD 是否可用于训练，请在岗位页重新检查。',
      },
      trainFallback: {
        idle: '当前默认 JD「{name}」还未分析，系统不会自动应用；如需按岗位要求训练，请先在岗位页完成分析。',
        running:
          '当前默认 JD「{name}」正在分析中，系统暂不自动应用；请等待分析完成。',
        succeeded:
          '当前默认 JD「{name}」可直接用于训练；如本轮不需要岗位视角，可保持”通用训练”。',
        failed:
          '当前默认 JD「{name}」最近一次分析失败，系统不会自动应用；修正内容或重新分析后可恢复。',
        stale:
          '当前默认 JD「{name}」原文已变更，系统不会自动应用旧分析；重新分析后可恢复。',
        unknown:
          '当前默认 JD 暂时无法确认是否可用，系统不会自动应用；如需岗位视角，请先在岗位页确认状态。',
      },
      jobsReadiness: {
        idle: '这份 JD 还没有分析结果，暂时无法用于新训练。',
        running: '这份 JD 正在分析中，分析完成前暂时无法用于新训练。',
        succeeded: '这份 JD 原文与最新分析一致，可直接用于新训练。',
        failed: '这份 JD 当前分析失败。修正内容或重新分析后可恢复。',
        stale:
          '这份 JD 原文已变更，当前分析已过期。旧分析仅供回看，新训练不会使用。',
        unknown: '这份 JD 暂时无法确认是否可用于训练，请重新检查。',
      },
      jobsSnapshot: {
        idle: '当前还没有成功分析快照，所以这里暂时没有可回看的岗位要求。',
        running:
          '当前分析仍在进行中；如果下面出现旧快照，它只是历史成功结果，不代表当前原文已经重新可用。',
        succeeded: '下面展示的是当前可用于训练的最新分析快照。',
        failed:
          '下面展示的是最近一次成功分析的旧快照，仅供回看；当前原文仍处于分析失败状态。',
        stale:
          '下面展示的是最近一次成功分析的旧快照，仅供回看；当前原文需要重新分析后才能重新用于训练。',
        unknown: '下面的快照状态暂时无法确认，请回岗位页重新检查。',
      },
    },
    home: {
      onboarding: {
        kicker: '快速开始',
        title: '三步完成初始化',
        description: '填写基本信息 → 导入项目（可跳过）→ 开始第一轮训练',
        step1: '1. 填写画像',
        step2: '2. 导入项目',
        step3: '3. 开始训练',
        steps: {
          profile: {
            label: '填写画像',
            hint: '先把目标岗位、阶段和技术栈补齐。',
          },
          projects: {
            label: '导入项目',
            hint: '可导入真实仓库，也可以先跳过。',
          },
          train: {
            label: '开始训练',
            hint: '从第一轮基础题或项目题开始建立训练记录。',
          },
        },
        status: {
          done: '已完成',
          current: '当前',
          next: '下一步',
        },
      },
      hero: {
        kicker: '今日建议',
        title: '先练最影响当前面试表现的部分。',
        actionPrimary: '开始训练',
        actionSecondary: '管理项目',
      },
      deadline: {
        kicker: '投递节奏',
        title: '时间安排',
      },
      dueReviews: {
        kicker: '今日待复习',
        description: '你有 {count} 条待复习内容，建议直接开一轮针对性训练。',
        review: '复习',
        startAction: '开始补这一项',
        markDone: '已复习',
        topicHint: '建议直接从 {topic} 开始补。',
        genericHint: '当前还没归到具体主题，也可以先开一轮基础训练。',
      },
      currentSession: {
        kicker: '继续训练',
        title: '你有一轮训练还没结束',
        description: '{name} · {status}',
        jobTargetDescription: '本轮参考岗位：{name}',
        emptyTitle: '还没有进行中的训练',
        emptyDescription: '从基础训练或项目训练开始一轮新的练习。',
      },
      cards: {
        weaknessKicker: 'Top Issue',
        weaknessTitle: '当前短板',
        weaknessEmpty: '还没有薄弱点记录，完成第一轮训练后会在这里显示。',
        weaknessDescription:
          '当前最需要补的是 {kind} / {label}，建议优先围绕这个点练习。',
        sessionKicker: 'Recent',
        sessionTitle: '最近训练',
        sessionEmpty: '还没有历史训练记录，完成第一轮后这里会显示最近结果。',
        sessionDescription: '最近一轮是 {name}，当前状态为 {status}。',
        trackKicker: 'Focus',
        trackTitle: '推荐专项',
        jobTargetKicker: 'JD',
        jobTargetTitle: '当前岗位上下文',
        jobTargetEmpty: '当前没有默认 JD，推荐和训练按通用模式处理。',
        jobTargetScoped: '当前默认 JD 是 {name}，推荐和训练已按岗位要求调整。',
        jobTargetUnavailable:
          '当前默认 JD 是 {name}，但当前原文还没有可用分析结果；重新分析后才会恢复岗位视角。',
        profileKicker: 'Profile',
        profileTitle: '目标画像',
        profileEmpty: '画像还没初始化，系统暂时无法给出更准确的建议。',
        profileDescription: '{role} / {stage}，主讲项目：{projects}。',
      },
      sections: {
        weaknesses: '薄弱点',
        sessions: '训练记录',
        jobTargetDescription: '本轮参考岗位：{name}',
        weaknessesEmpty:
          '还没有历史薄弱点，完成第一轮训练后这里会显示重点问题。',
        weaknessTrends: '弱项趋势',
        sessionsEmpty: '还没有训练记录，开始第一轮后这里会看到最近结果。',
      },
    },
    profile: {
      hero: {
        kicker: '画像设置',
        newUserTitle: '先让我了解你的情况',
        newUserDescription:
          '后面的训练内容、追问方向和推荐都会以这份信息为基础。',
        returningTitle: '你的训练画像',
        returningDescription: '{role} · {company} · {stage}',
      },
      summaryStats: {
        deadline: '距投递 {days} 天',
        techCount: '技术栈 {count} 项',
        weaknessCount: '薄弱点 {count} 项',
        sessionCount: '已完成 {count} 轮训练',
      },
      sections: {
        directionTitle: '你想面什么方向？',
        directionHint: '这决定了训练时问题的行业语境和深度预期。',
        stageTitle: '你现在在什么阶段？',
        stageHint: '不同阶段的训练侧重不同，系统会据此调整难度。',
        techTitle: '你的技术准备',
        techHint: '技术栈和薄弱点直接影响出题方向和追问重点。',
      },
      fields: {
        targetRole: '目标岗位',
        targetCompanyType: '目标公司类型',
        currentStage: '当前阶段',
        applicationDeadline: '目标投递时间',
        techStacks: '技术栈',
        weaknesses: '自报薄弱点',
        systemWeaknesses: '系统追踪的薄弱点',
        linkedProjects: '已关联项目',
      },
      placeholders: {
        targetRole: '例如：Go 后端工程师 / Agent 工程师',
        techStacks: '输入后回车添加，例如 Go',
        weaknesses: '输入后回车添加，例如 Kafka 幂等',
      },
      presets: {
        companyType: {
          ai: 'AI 应用公司',
          bigTech: '互联网大厂',
          startup: '创业团队',
          midsize: '中小型公司',
          other: '其他',
        },
        stage: {
          campus: '校招准备期',
          newGrad: '应届求职',
          preIntern: '实习前准备',
          jobSwitch: '在职跳槽',
          other: '其他',
        },
      },
      deadlineHint: '设置后首页会显示倒计时',
      noProjects: '还没有导入项目',
      goImportProject: '去导入',
      noSystemWeaknesses: '完成训练后，系统会自动追踪你的薄弱点',
      validation: {
        targetRoleRequired: '请填写目标岗位',
      },
      saveSuccess: '画像已更新',
      saveAction: '保存画像',
      saveAndTrain: '保存并开始训练',
      emptyTitle: '还没有保存画像',
      emptyDescription:
        '先填写这份表单并保存，首页推荐和训练上下文才会更准确。',
      loadErrorTitle: '画像加载失败',
      saveErrorTitle: '画像保存失败',
    },
    projects: {
      hero: {
        kicker: '项目材料',
        title: '把项目整理成适合面试表达的材料。',
        description:
          '先导入仓库，再补齐项目摘要、亮点、难点、取舍和个人贡献，方便后续训练直接使用。',
      },
      onboarding: {
        kicker: '第二步',
        title: '导入项目，或者直接先开练',
        description: '如果你现在没有现成仓库，也可以先跳过这一步，后面再补。',
        readyDescription: '你已经有项目材料了，可以直接进入最后一步开始训练。',
        skipAction: '跳过项目，直接训练',
        continueAction: '继续去训练',
        backAction: '返回画像',
      },
      importPlaceholder: 'https://github.com/yourname/your-project',
      importAction: '导入仓库',
      importErrorTitle: '导入失败',
      retryAction: '重试导入',
      retryErrorTitle: '重试失败',
      jobsTitle: '后台导入任务',
      onboardingSkipHint:
        '项目导入这一步可以跳过，先开始第一轮训练也算完成首访链路。',
      skipToTrainAction: '跳过项目，直接训练',
      jobsEmpty: '还没有导入任务。提交仓库后，这里会显示后台进度。',
      jobRepo: '仓库地址',
      jobStage: '当前阶段',
      jobResult: '导入结果',
      openProject: '打开项目',
      listTitle: '项目列表',
      emptyList: '先导入一个公开的 GitHub 仓库。',
      editorTitle: '项目信息编辑',
      fields: {
        name: '项目名称',
        summary: '项目摘要',
        techStack: '技术栈',
        highlights: '亮点',
        challenges: '难点',
        tradeoffs: '取舍与权衡',
        ownership: '个人负责内容',
        followups: '可继续追问点',
      },
      saveAction: '保存项目信息',
      saveErrorTitle: '保存失败',
    },
    jobs: {
      hero: {
        kicker: '岗位材料',
        description:
          '保存目标岗位 JD 并生成结构化要求，后续训练即可按岗位要求进行。',
      },
      listTitle: 'JD 列表',
      emptyList: '还没有岗位 JD，先新建一份。',
      createTitle: '新建岗位 JD',
      editorTitle: '岗位详情',
      readinessTitle: '当前训练状态',
      latestAnalysisTitle: '最新成功分析',
      historyTitle: '分析历史',
      historyEmpty: '还没有分析记录。',
      noLatestAnalysis: '这份 JD 还没有成功分析结果。',
      noSelectionTitle: '先选择一份 JD',
      noSelectionDescription: '左侧选中一份 JD，或直接新建一份新的岗位目标。',
      createErrorTitle: '创建 JD 失败',
      saveErrorTitle: '保存 JD 失败',
      analyzeErrorTitle: '分析 JD 失败',
      activeErrorTitle: '设置默认 JD 失败',
      saveAction: '保存 JD',
      analyzeAction: '开始分析',
      reanalyzeAction: '重新分析',
      createAction: '新建 JD',
      activateAction: '设为默认 JD',
      clearActiveAction: '取消默认 JD',
      activeBadge: '当前默认 JD',
      activeReadyDescription:
        '这份 JD 已作为默认岗位，首页推荐和训练默认都会围绕它展开。',
      activeNotReadyDescription:
        '这份 JD 已设为默认岗位，但当前原文还没有可用分析；首页会继续显示它，但训练和推荐不会自动使用它。',
      analyzing: '分析中...',
      fields: {
        title: '岗位标题',
        companyName: '公司名',
        sourceText: 'JD 原文',
        summary: '岗位摘要',
        mustHaveSkills: '必备能力',
        bonusSkills: '加分项',
        responsibilities: '岗位职责',
        evaluationFocus: '面试重点',
      },
      placeholders: {
        title: '例如：后端工程师 - 某创业公司',
        companyName: '例如：某创业公司',
        sourceText: '直接粘贴岗位 JD 原文',
      },
      runStatus: '状态：{status}',
      runError: '失败原因：{message}',
    },
    train: {
      hero: {
        kicker: '训练设置',
        title: '先选对训练模式，再开始练习。',
        description:
          '基础训练适合补知识点，项目训练适合练项目讲述。每轮训练包含多次追问，轮次可在下方调整。',
      },
      onboarding: {
        kicker: '最后一步',
        title: '开始第一轮训练',
        description:
          '参数不用一次配很复杂，先开始一轮，首页和历史数据就会真正跑起来。',
      },
      fields: {
        mode: '训练模式',
        intensity: '训练强度',
        maxTurns: '训练轮次',
        topic: '主题',
        project: '选择项目',
        jobTarget: '参考岗位 JD',
        promptSet: 'Prompt 版本',
      },
      resumeTitle: '继续当前训练',
      resumeDescription: '{name} · {status}',
      chooseProject: '请选择项目',
      chooseJobTarget: '请选择岗位 JD',
      genericJobTargetOption: '不使用 JD（通用训练）',
      jobTargetUnavailable:
        '这份 JD 还没有可用分析结果，暂时不能开始本轮训练。',
      activeJobTargetUnavailable:
        '当前默认 JD「{name}」现在不能直接用于训练；如需按岗位要求训练，请先在岗位页修正或重新分析。',
      startAction: '开始这一轮训练',
      startErrorTitle: '启动失败',
    },
    history: {
      hero: {
        kicker: '训练记录',
        title: '查看所有训练历史和成绩变化。',
      },
      exportErrorTitle: '批量导出失败',
      filters: {
        allModes: '所有模式',
        allTopics: '所有主题',
        allStatuses: '所有状态',
      },
      batch: {
        kicker: '批量导出',
        description:
          '勾选需要导出的训练记录后，可以跨页累计，最后按 {format} 一次性打包导出。',
        selectedCount: '已选 {count} 条训练记录',
        selectPageAction: '全选当前页（{count} 条）',
        clearPageAction: '取消当前页（{count} 条）',
        clearAllAction: '清空全部选择',
        exportAction: '导出所选 {format}（{count} 条）',
        exportingAction: '正在打包导出...',
      },
      empty: '还没有训练记录。',
      noJobTarget: '无岗位关联',
      promptSetBadge: 'Prompt：{name}',
      openAction: '查看详情',
      prev: '上一页',
      next: '下一页',
    },
    session: {
      hero: {
        kicker: '训练过程',
        title: '先按真实面试场景作答。',
        description:
          '当前状态：{status}。系统会评估回答，按轮次决定是否继续追问。',
      },
      turnIndicator: '第 {current} / {total} 轮',
      currentQuestion: '当前问题',
      feedback: '过程反馈',
      mainScore: '本轮得分：{score}',
      strengths: '优点',
      gaps: '待补强点',
      feedbackEmpty: '先回答当前问题，反馈会显示在这里。',
      processingKicker: '处理中',
      reasoningTitle: '分析过程',
      contentTitle: '生成内容',
      processingGeneratingQuestionTitle: '正在准备本轮问题',
      processingGeneratingQuestionDescription:
        '系统正在整理上下文并生成问题，请稍候。',
      processingEvaluatingTitle: '正在评估你的回答',
      processingEvaluatingDescription: '系统正在分析本轮答案并组织下一步反馈。',
      processingReviewTitle: '正在生成复盘',
      processingReviewDescription: '系统正在汇总整轮表现并整理复盘卡。',
      placeholderInitial:
        '先说结论，再解释原因，最后给一个真实项目或场景例子。',
      placeholderFollowup: '直接回应追问，补充关键信息，不要重复上一次的回答。',
      submitShortcutHint: '支持 Ctrl + Enter 快速提交。',
      submitErrorTitle: '提交失败',
      conflictBusy: '上一轮提交还在处理中，请等待当前评估或复盘完成。',
      conflictReviewPending:
        '上一轮回答已保存，仅剩复盘未完成。请直接使用”重试生成复盘”。',
      conflictCompleted: '这轮训练已经完成，页面会跳转到复盘结果。',
      conflictInvalidStatus: '当前状态不能继续提交，请等待页面刷新到最新状态。',
      retryReviewNotRecoverable:
        '这轮训练已经不在可恢复状态，请刷新页面确认最新结果。',
      reviewGenerationRetry:
        '回答已保存，但复盘生成失败。页面会切换到可恢复状态，你可以直接重试。',
      answerLockedWhileProcessing:
        '系统正在处理中，当前输入区已锁定，避免重复提交。',
      submittedAnswerTitle: '已收到你的回答',
      submittedAnswerDescription:
        '系统正在处理这段内容，当前先锁定展示，避免重复提交。',
      followupIntentTitle: '这条追问在确认什么',
      suggestionTitle: '下一次怎么答会更好',
      suggestionFallback:
        '系统已经识别到你的主要缺口，下一轮会继续围绕这个点追问。',
      feedbackHeadlineFallback:
        '系统已经完成初步判断，下面是本轮最关键的反馈。',
      scoreBreakdownTitle: '分项得分',
      reviewPendingTitle: '复盘尚未完成',
      reviewPendingDescription:
        '上一轮回答已经保存，但复盘生成没有完成。你可以直接重试复盘，不需要重新回答。',
      retryReviewAction: '重试生成复盘',
      reviewWrapUpTitle: '本轮问答已完成',
      reviewWrapUpDescription: '系统正在整理复盘卡，马上带你进入总结页面。',
      jobTargetTitle: '本轮参考岗位',
      jobTargetDescription: '这轮问题、评分和复盘都会围绕这份 JD 的要求。',
      jobTargetEmpty: '本轮未关联岗位 JD，当前仍按通用训练处理。',
      streamPending: '内容生成中，正在整理结果，请稍候。',
      streamSectionCounter: '步骤 {index}',
      traceTitle: '运行轨迹',
      traceEmpty: '当前步骤还没有结构化轨迹。',
      streamKinds: {
        prepare: '正在准备上下文',
        drafting: '生成中',
        finalizing: '正在整理结果',
        processing: '处理中',
        question: '问题草稿',
        evaluation: '评估结果',
        review: '复盘草稿',
      },
      streamFields: {
        score: '当前得分：{score}',
        scoreBreakdown: '分项得分',
        followupQuestion: '后续追问',
      },
      tracePhases: {
        prepare_context: '准备上下文',
        tool_call: '调用工具',
        validate: '校验结果',
        fallback: '回退处理',
        finalize: '整理收口',
        error: '错误退出',
      },
      traceStatuses: {
        info: '进行中',
        success: '成功',
        retry: '重试',
        fallback: '已回退',
        error: '失败',
      },
      streamContexts: {
        read_question_templates: '正在参考题库',
        read_project_brief: '正在参考项目信息',
        read_context_chunks: '正在参考代码与文档',
        read_weakness_memory: '正在参考历史记录',
        read_evaluation_context: '正在分析回答内容',
        read_session_summary: '正在整理本轮训练',
        read_turn_history: '正在参考历史问答',
        read_repo_overview: '正在参考项目结构',
        read_repo_chunks: '正在参考关键代码',
        read_job_target_source: '正在参考岗位要求',
        read_job_target_analysis: '正在参考岗位分析',
      },
    },
    progress: {
      createSession: {
        title: '正在准备训练',
        description: '系统会先读取上下文，再生成当前问题。',
        steps: ['读取训练上下文', '生成问题', '整理训练页面'],
      },
      evaluateMain: {
        title: '正在评估回答',
        description: '系统会先分析回答，再生成追问。',
        steps: ['已收到回答', '正在评估回答', '追问与反馈已生成'],
      },
      evaluateFollowup: {
        title: '正在生成本轮复盘',
        description: '系统会先完成追问评估，再整理复盘卡。',
        steps: ['已收到追问回答', '已完成最终评估', '复盘卡已生成'],
      },
    },
    review: {
      hero: {
        kicker: '复盘',
        title: '复盘会总结这轮表现，并给出下一步重点。',
        loading: '复盘加载中...',
      },
      headerError: '复盘暂时无法加载。',
      loadingTitle: '正在读取复盘',
      loadErrorTitle: '复盘加载失败',
      exportErrorTitle: '导出失败',
      emptyTitle: '这轮复盘还没准备好',
      emptyDescription: '可能仍在生成中，或本轮训练未成功完成。',
      exportAction: '导出 {format}',
      exportingAction: '导出中...',
      promptExperimentAction: '查看 Prompt 实验',
      topFixTitle: '最该优先修正',
      topFixFallbackReason: '先修正这个问题，后续训练效果会明显提升。',
      scoreBreakdown: '分项得分',
      highlights: '回答亮点',
      gaps: '需要补强',
      nextFocus: '下一轮重点',
      recommendedNextTitle: '推荐下一轮',
      recommendedNextBasics: '建议先做一轮 {topic} 基础训练',
      recommendedNextMode: '建议直接开始一轮 {mode}',
      continueAction: '继续下一轮',
      startRecommendedAction: '立即开始推荐下一轮',
      jobTargetTitle: '本轮参考岗位',
      jobTargetDescription: '这轮复盘是按这份岗位 JD 的视角整理的。',
      promptSetTitle: '本轮 Prompt 版本',
      promptSetDescription: '这轮训练绑定的 Prompt 版本状态：{status}。',
      auditShowAction: '查看评估详情',
      auditHideAction: '收起评估详情',
      auditTitle: '评估审计详情',
      auditDescription:
        '这里会展开每个 flow 的模型、Prompt 哈希、耗时和原始输出。',
      auditErrorTitle: '评估详情加载失败',
      auditMeta: '模型 / Prompt 哈希 / 耗时',
      auditRawOutput: '原始输出',
      auditEmpty: '当前没有可展开的评估日志。',
      runtimeTraceTitle: '运行轨迹',
      retrievalTraceTitle: '检索轨迹',
      retrievalTraceDescription:
        '这里展示复盘生成时实际命中的长期记忆材料和 fallback 情况。',
      retrievalTraceObservations: 'Observations',
      retrievalTraceSummaries: 'Session Summaries',
      retrievalTraceEmpty: '当前没有记录到这组材料的命中结果。',
    },
    promptExperiments: {
      hero: {
        kicker: 'Prompt 实验',
        title: '按版本比较训练结果和耗时表现。',
        description:
          '这里按多次训练结果做跨 Session 聚合对比，不会在同一轮里双跑两套 Prompt。',
      },
      loadErrorTitle: '版本列表加载失败',
      compareErrorTitle: '实验结果加载失败',
      filters: {
        left: '左侧版本',
        right: '右侧版本',
        mode: '训练模式',
        topic: '主题',
        allModes: '全部模式',
        allTopics: '全部主题',
      },
      metrics: {
        sessionCount: '样本数',
        completedCount: '完成数',
        avgTotalScore: '平均总分',
        avgQuestionLatency: '问题生成平均耗时',
        avgAnswerLatency: '评估平均耗时',
        avgReviewLatency: '复盘平均耗时',
      },
      samples: {
        kicker: '样本',
        title: '最近训练样本',
        empty: '当前筛选条件下还没有可比较的样本。',
        showLogs: '展开审计明细',
        hideLogs: '收起审计明细',
      },
      logs: {
        errorTitle: '日志加载失败',
        empty: '这轮训练还没有审计明细。',
        modelName: '模型',
        promptHash: 'Prompt Hash',
        latency: '耗时',
      },
    },
  },
  en: {
    app: {
      name: 'PracticeHelper',
      title: 'Interview Practice Assistant',
      language: 'Language',
      theme: 'Theme',
      importNotice: 'Background import running: {stage}',
      locales: {
        zhCN: 'Chinese',
        en: 'English',
      },
      themes: {
        light: 'Light',
        dark: 'Dark',
      },
      nav: {
        home: 'Home',
        profile: 'Profile',
        jobs: 'Jobs',
        projects: 'Projects',
        promptExperiments: 'Prompt Experiments',
        train: 'Train',
        history: 'History',
      },
    },
    common: {
      loading: 'Loading...',
      requestFailed: 'Request failed. Please try again.',
      notProvided: 'Not provided',
      unnamedSession: 'Untitled session',
      unknownSession: 'Untitled item',
      save: 'Save',
      saving: 'Saving...',
      start: 'Start training',
      starting: 'Starting...',
      retry: 'Retry',
      submit: 'Submit answer',
      submitting: 'Submitting...',
      continue: 'Continue',
      resume: 'Resume',
      score: 'Score',
      status: 'Status',
      severity: 'Severity {value}',
      daysRemainingLabel: 'Days left before your target application date',
      lastUpdated: 'Last updated {value}',
      firstTrainingHint:
        'Complete your profile first, then start the first session.',
      nextRecommendationHint:
        'Complete your profile to get the next recommendation.',
      noRecommendation:
        'No recommendation yet. Finish the first session to generate one.',
      setDeadlineHint:
        'Set your target application date in the profile page to show the countdown here.',
      exportFormatLabel: 'Export format',
      exportFormats: {
        markdown: 'Markdown',
        json: 'JSON',
        pdf: 'PDF',
      },
      exportSuccess: 'Export complete',
      operationSuccess: 'Done',
      apiErrors: {
        toolContextMissing:
          'The run stopped before the model read the required context. Please retry.',
        toolCallFailed:
          'The system failed while calling an internal tool. Please retry shortly.',
        toolLoopExhausted:
          'The model kept looping without converging to a final answer. This run was stopped.',
        jsonParseFailed:
          'The system produced an invalid result shape that could not be parsed.',
        semanticValidationFailed:
          'The generated result did not pass business validation. Please retry.',
        singleShotFailed:
          'The fallback single-shot generation also failed. Please retry later.',
        backendCallbackFailed:
          'The system failed while fetching extra training material from the backend.',
        timeout: 'This request timed out. Please retry.',
        llmRequired:
          'No working model is configured right now, so generation is unavailable.',
        unknownError: 'This run failed. Please retry later.',
      },
    },
    enums: {
      mode: {
        basics: 'Fundamentals',
        project: 'Project',
      },
      intensity: {
        auto: 'Auto',
        light: 'Light',
        standard: 'Standard',
        pressure: 'Intensive',
      },
      topic: {
        mixed: 'Mixed fundamentals',
        go: 'Go',
        redis: 'Redis',
        kafka: 'Kafka',
        mysql: 'MySQL',
        system_design: 'System Design',
        distributed: 'Distributed Systems',
        network: 'Networking',
        microservice: 'Microservices',
        os: 'Operating Systems',
        docker_k8s: 'Docker & K8s',
      },
      status: {
        draft: 'Draft',
        active: 'In progress',
        generating_question: 'Generating question',
        waiting_answer: 'Waiting for answer',
        evaluating: 'Evaluating',
        followup: 'Follow-up',
        review_pending: 'Preparing review',
        completed: 'Completed',
      },
      importStatus: {
        ready: 'Ready',
      },
      importJobStatus: {
        queued: 'Queued',
        running: 'Running',
        completed: 'Completed',
        failed: 'Failed',
      },
      importJobStage: {
        queued: 'Waiting to start',
        analyzing_repository: 'Analyzing repository',
        persisting_project: 'Persisting project data',
        completed: 'Import completed',
        failed: 'Import failed',
      },
      jobTargetAnalysisStatus: {
        idle: 'Needs analysis',
        running: 'Analyzing',
        succeeded: 'Ready for training',
        failed: 'Analysis failed',
        stale: 'Source changed',
      },
      weaknessKind: {
        topic: 'Knowledge area',
        project: 'Project explanation',
        expression: 'Communication',
        followup_breakdown: 'Follow-up handling',
        depth: 'Depth',
        detail: 'Supporting detail',
      },
    },
    jobTargetStatus: {
      homeActive: {
        idle: 'The default JD is {name}, but it has not been analyzed yet, so recommendations and training stay in generic mode.',
        running:
          'The default JD is {name}, but analysis is still running. Training and recommendations will not use it automatically until that finishes.',
        succeeded:
          'The default JD is {name}, and both recommendations and training are already using job-target mode.',
        failed:
          'The default JD is {name}, but the latest analysis failed. Fix the JD or rerun analysis to restore job-target mode.',
        stale:
          'The default JD is {name}, but its source text changed. The old analysis remains visible for review only and new training will not use it.',
        unknown:
          'The default JD is temporarily unavailable, so recommendations and training stay in generic mode.',
      },
      trainSelection: {
        idle: 'This JD has not been analyzed yet, so it cannot be bound to a new session.',
        running:
          'This JD is still being analyzed, so it cannot be bound to a new session yet.',
        succeeded:
          'This session will bind to the current successful analysis snapshot for this JD, and later edits will not change the session.',
        failed:
          'The latest analysis for this JD failed. Fix the text or rerun analysis before binding it to training.',
        stale:
          'The JD source text has changed. The previous analysis is review-only until you rerun analysis.',
        unknown:
          'The current training readiness for this JD is unknown. Recheck it on the job target page.',
      },
      trainFallback: {
        idle: 'The default JD "{name}" has not been analyzed yet, so it will not be auto-applied. Go back to the job target page if you want job-target mode.',
        running:
          'The default JD "{name}" is still analyzing, so it will not be auto-applied yet.',
        succeeded:
          'The default JD "{name}" is ready for training. Keep the current generic option only if you want to skip job-target mode for this round.',
        failed:
          'The default JD "{name}" failed its latest analysis, so it will not be auto-applied until you fix or rerun it.',
        stale:
          'The default JD "{name}" has changed, so the old analysis will not be auto-applied. Rerun analysis to restore job-target mode.',
        unknown:
          'The default JD is temporarily unavailable, so it will not be auto-applied until you verify it again.',
      },
      jobsReadiness: {
        idle: 'This JD has no analysis result yet, so it cannot be used for new training.',
        running:
          'This JD is currently being analyzed. It cannot be used for new training until that finishes.',
        succeeded:
          'This JD is aligned with its latest successful analysis and can be used for new training now.',
        failed:
          'Analysis for the current JD text failed. Fix the text or rerun analysis before using it again.',
        stale:
          'The JD text changed, so the current analysis is stale. You can still review the old snapshot, but new training will not use it.',
        unknown:
          'The training readiness for this JD is currently unknown. Please recheck it.',
      },
      jobsSnapshot: {
        idle: 'There is no successful analysis snapshot yet, so there is nothing reviewable here.',
        running:
          'Analysis is still running. If you see a snapshot below, it is only an older successful result and not the current source text.',
        succeeded:
          'The snapshot below is the latest analysis currently used for training.',
        failed:
          'The snapshot below is the most recent successful result and is review-only while the current text is still in a failed state.',
        stale:
          'The snapshot below is the most recent successful result and is review-only until the current JD text is analyzed again.',
        unknown:
          'The status of the snapshot below is currently unknown. Please recheck it.',
      },
    },
    home: {
      onboarding: {
        kicker: 'Quick Start',
        title: 'Set up in 3 steps',
        description:
          'Fill in your profile → Import a project (optional) → Start your first session',
        step1: '1. Profile',
        step2: '2. Projects',
        step3: '3. Train',
        steps: {
          profile: {
            label: 'Profile',
            hint: 'Set your role, stage, and core tech stack first.',
          },
          projects: {
            label: 'Projects',
            hint: 'Import a real repo, or skip this step for now.',
          },
          train: {
            label: 'Training',
            hint: 'Start the first fundamentals or project round.',
          },
        },
        status: {
          done: 'Done',
          current: 'Current',
          next: 'Next',
        },
      },
      hero: {
        kicker: 'Today',
        title: 'Work on the area that affects your interviews the most.',
        actionPrimary: 'Start training',
        actionSecondary: 'Manage projects',
      },
      deadline: {
        kicker: 'Schedule',
        title: 'Timeline',
      },
      dueReviews: {
        kicker: 'Due for Review',
        description:
          'You have {count} due items. Start a targeted round directly from here.',
        review: 'Review',
        startAction: 'Start this fix',
        markDone: 'Done',
        topicHint: 'Best next round: start with {topic}.',
        genericHint:
          'This item is not mapped to a topic yet, so start with a fundamentals round.',
      },
      currentSession: {
        kicker: 'Resume',
        title: 'You have a training session in progress',
        description: '{name} · {status}',
        jobTargetDescription: 'This session is bound to JD: {name}',
        emptyTitle: 'No active session right now',
        emptyDescription:
          'Start a new fundamentals or project session when you are ready.',
      },
      cards: {
        weaknessKicker: 'Top Issue',
        weaknessTitle: 'Current weakness',
        weaknessEmpty:
          'No weakness data yet. Complete your first session to build a history.',
        weaknessDescription:
          'The top area to improve right now is {kind} / {label}. Start there first.',
        sessionKicker: 'Recent',
        sessionTitle: 'Recent session',
        sessionEmpty:
          'No training history yet. Your latest result will appear here after the first session.',
        sessionDescription:
          'Your latest session is {name}, and its current status is {status}.',
        trackKicker: 'Focus',
        trackTitle: 'Recommended focus',
        jobTargetKicker: 'JD',
        jobTargetTitle: 'Current job context',
        jobTargetEmpty:
          'There is no default JD right now, so recommendations and default training stay generic.',
        jobTargetScoped:
          'The current default JD is {name}, and recommendations now follow that role context.',
        jobTargetUnavailable:
          'The current default JD is {name}, but its current text has no usable analysis yet. Rerun analysis to restore job-target mode.',
        profileKicker: 'Profile',
        profileTitle: 'Target profile',
        profileEmpty:
          'Your profile is not set up yet, so recommendations are still generic.',
        profileDescription: '{role} / {stage}, primary projects: {projects}.',
      },
      sections: {
        weaknesses: 'Weaknesses',
        sessions: 'Training history',
        jobTargetDescription: 'Bound job target: {name}',
        weaknessesEmpty:
          'No historical weaknesses yet. They will appear after the first session.',
        weaknessTrends: 'Weakness Trends',
        sessionsEmpty:
          'No sessions yet. Start one to see your latest results here.',
      },
    },
    profile: {
      hero: {
        kicker: 'Profile',
        newUserTitle: 'Tell me about yourself first',
        newUserDescription:
          'Training content, follow-up directions, and recommendations will all be based on this.',
        returningTitle: 'Your training profile',
        returningDescription: '{role} · {company} · {stage}',
      },
      summaryStats: {
        deadline: '{days} days left',
        techCount: '{count} tech stacks',
        weaknessCount: '{count} weaknesses',
        sessionCount: '{count} sessions done',
      },
      sections: {
        directionTitle: 'What role are you targeting?',
        directionHint:
          'This determines the industry context and depth of training questions.',
        stageTitle: 'Where are you in your job search?',
        stageHint:
          'Training priorities differ by stage. The system adjusts difficulty accordingly.',
        techTitle: 'Your technical preparation',
        techHint:
          'Tech stacks and weaknesses directly influence question topics and follow-up focus.',
      },
      fields: {
        targetRole: 'Target role',
        targetCompanyType: 'Target company type',
        currentStage: 'Current stage',
        applicationDeadline: 'Target application date',
        techStacks: 'Tech stack',
        weaknesses: 'Self-reported weaknesses',
        systemWeaknesses: 'System-tracked weaknesses',
        linkedProjects: 'Linked projects',
      },
      placeholders: {
        targetRole: 'For example: Go Backend Engineer / Agent Engineer',
        techStacks: 'Type and press Enter to add, e.g. Go',
        weaknesses: 'Type and press Enter to add, e.g. Kafka idempotency',
      },
      presets: {
        companyType: {
          ai: 'AI product company',
          bigTech: 'Big tech',
          startup: 'Startup',
          midsize: 'Mid-size company',
          other: 'Other',
        },
        stage: {
          campus: 'Campus hiring prep',
          newGrad: 'New grad job search',
          preIntern: 'Before internship',
          jobSwitch: 'Job switch',
          other: 'Other',
        },
      },
      deadlineHint: 'A countdown will appear on the home page once set',
      noProjects: 'No imported projects yet',
      goImportProject: 'Import one',
      noSystemWeaknesses:
        'System-tracked weaknesses will appear after your first training session',
      validation: {
        targetRoleRequired: 'Please enter your target role',
      },
      saveSuccess: 'Profile updated',
      saveAction: 'Save profile',
      saveAndTrain: 'Save and start training',
      emptyTitle: 'No saved profile yet',
      emptyDescription:
        'Fill out this form and save it first so recommendations and training context become more accurate.',
      loadErrorTitle: 'Failed to load profile',
      saveErrorTitle: 'Failed to save profile',
    },
    projects: {
      hero: {
        kicker: 'Projects',
        title: 'Turn your projects into interview-ready material.',
        description:
          'Import a repository first, then refine the summary, highlights, challenges, trade-offs, and ownership for later training.',
      },
      onboarding: {
        kicker: 'Step 2',
        title: 'Import a project, or skip it for now',
        description:
          'If you do not have a ready repository yet, you can skip this step and still complete the first-run flow.',
        readyDescription:
          'Your project material is ready. Move to the last step and start training.',
        skipAction: 'Skip to training',
        continueAction: 'Continue to training',
        backAction: 'Back to profile',
      },
      importPlaceholder: 'https://github.com/yourname/your-project',
      importAction: 'Import repository',
      importErrorTitle: 'Import failed',
      retryAction: 'Retry import',
      retryErrorTitle: 'Retry failed',
      jobsTitle: 'Background import jobs',
      onboardingSkipHint:
        'Project import is optional for first-time setup. You can skip it and start training now.',
      skipToTrainAction: 'Skip projects and train now',
      jobsEmpty:
        'No import jobs yet. Submit a repository to see progress here.',
      jobRepo: 'Repository',
      jobStage: 'Current stage',
      jobResult: 'Import result',
      openProject: 'Open project',
      listTitle: 'Project list',
      emptyList: 'Import a public GitHub repository first.',
      editorTitle: 'Edit project details',
      fields: {
        name: 'Project name',
        summary: 'Summary',
        techStack: 'Tech stack',
        highlights: 'Highlights',
        challenges: 'Challenges',
        tradeoffs: 'Trade-offs',
        ownership: 'Your ownership',
        followups: 'Follow-up angles',
      },
      saveAction: 'Save project details',
      saveErrorTitle: 'Save failed',
    },
    jobs: {
      hero: {
        kicker: 'Job targets',
        description:
          'Save target JDs first, then analyze them into structured requirements so later training can stay role-specific.',
      },
      listTitle: 'JD list',
      emptyList: 'No JD yet. Create one first.',
      createTitle: 'Create JD',
      editorTitle: 'JD details',
      readinessTitle: 'Current training status',
      latestAnalysisTitle: 'Latest successful analysis',
      historyTitle: 'Analysis history',
      historyEmpty: 'No analysis history yet.',
      noLatestAnalysis: 'This JD has no successful analysis yet.',
      noSelectionTitle: 'Choose a JD first',
      noSelectionDescription:
        'Select a JD from the list on the left, or create a new one.',
      createErrorTitle: 'Failed to create JD',
      saveErrorTitle: 'Failed to save JD',
      analyzeErrorTitle: 'Failed to analyze JD',
      activeErrorTitle: 'Failed to update default JD',
      saveAction: 'Save JD',
      analyzeAction: 'Analyze',
      reanalyzeAction: 'Re-analyze',
      createAction: 'Create JD',
      activateAction: 'Set as default JD',
      clearActiveAction: 'Clear default JD',
      activeBadge: 'Current default JD',
      activeReadyDescription:
        'This JD is now the default role context, so home recommendations and default training will follow it.',
      activeNotReadyDescription:
        'This JD is still marked as default, but its current text is not usable yet. The product keeps showing it, but training and recommendations will not auto-use it.',
      analyzing: 'Analyzing...',
      fields: {
        title: 'Job title',
        companyName: 'Company',
        sourceText: 'Original JD',
        summary: 'Role summary',
        mustHaveSkills: 'Must-have skills',
        bonusSkills: 'Bonus skills',
        responsibilities: 'Responsibilities',
        evaluationFocus: 'Evaluation focus',
      },
      placeholders: {
        title: 'For example: Backend Engineer - Startup',
        companyName: 'For example: Startup Inc.',
        sourceText: 'Paste the original JD text here',
      },
      runStatus: 'Status: {status}',
      runError: 'Failure reason: {message}',
    },
    train: {
      hero: {
        kicker: 'Training',
        title: 'Choose the right mode, then start practicing.',
        description:
          'Fundamentals mode is for core knowledge. Project mode is for project storytelling. Each session includes multiple follow-up rounds — adjustable below.',
      },
      onboarding: {
        kicker: 'Final step',
        title: 'Start the first session',
        description:
          'Do not over-configure the first round. Start one session first so the dashboard and history can begin to accumulate real data.',
      },
      fields: {
        mode: 'Mode',
        intensity: 'Intensity',
        maxTurns: 'Turns',
        topic: 'Topic',
        project: 'Project',
        jobTarget: 'Job target JD',
        promptSet: 'Prompt set',
      },
      resumeTitle: 'Resume current session',
      resumeDescription: '{name} · {status}',
      chooseProject: 'Select a project',
      chooseJobTarget: 'Select a JD',
      genericJobTargetOption: 'No JD (generic training)',
      jobTargetUnavailable:
        'This JD does not have a usable analysis result yet, so this session cannot start.',
      activeJobTargetUnavailable:
        'The current default JD "{name}" cannot be used right now. Fix or rerun it from the job target page if you want job-target mode for this round.',
      startAction: 'Start this session',
      startErrorTitle: 'Start failed',
    },
    history: {
      hero: {
        kicker: 'History',
        title: 'View all training sessions and score trends.',
      },
      exportErrorTitle: 'Batch export failed',
      filters: {
        allModes: 'All modes',
        allTopics: 'All topics',
        allStatuses: 'All statuses',
      },
      batch: {
        kicker: 'Batch export',
        description:
          'Select sessions across pages, then export them together as a {format} archive.',
        selectedCount: '{count} sessions selected',
        selectPageAction: 'Select this page ({count})',
        clearPageAction: 'Clear this page ({count})',
        clearAllAction: 'Clear all',
        exportAction: 'Export selected {format} ({count})',
        exportingAction: 'Packaging export...',
      },
      empty: 'No training sessions yet.',
      noJobTarget: 'No job target',
      promptSetBadge: 'Prompt: {name}',
      openAction: 'Open details',
      prev: 'Previous',
      next: 'Next',
    },
    session: {
      hero: {
        kicker: 'Session',
        title: 'Answer as if you were in a real interview.',
        description:
          'Current status: {status}. The system evaluates each answer and decides whether to continue with follow-ups.',
      },
      turnIndicator: 'Turn {current} / {total}',
      currentQuestion: 'Current question',
      feedback: 'Feedback',
      mainScore: 'Score: {score}',
      strengths: 'Strengths',
      gaps: 'Gaps',
      feedbackEmpty:
        'Answer the current question first. Feedback will appear here.',
      processingKicker: 'Processing',
      reasoningTitle: 'Analysis Process',
      contentTitle: 'Generated Content',
      processingGeneratingQuestionTitle: 'Preparing the session question',
      processingGeneratingQuestionDescription:
        'The system is reading context and generating the question.',
      processingEvaluatingTitle: 'Evaluating your answer',
      processingEvaluatingDescription:
        'The system is analyzing the answer and preparing the next step.',
      processingReviewTitle: 'Generating the review',
      processingReviewDescription:
        'The system is summarizing the session and preparing the review card.',
      placeholderInitial:
        'State your conclusion first, explain why, then give a real example.',
      placeholderFollowup:
        'Answer the follow-up directly and add the missing detail without repeating yourself.',
      submitShortcutHint: 'Press Ctrl + Enter to submit quickly.',
      submitErrorTitle: 'Submit failed',
      conflictBusy:
        'A previous submission is still being processed. Wait for the current evaluation or review to finish.',
      conflictReviewPending:
        'Your last answer is already saved. Only review generation is left, so use "Retry review generation".',
      conflictCompleted:
        'This session is already completed. The page will move to the review result.',
      conflictInvalidStatus:
        'This session cannot accept another answer right now. Refresh to sync the latest state.',
      retryReviewNotRecoverable:
        'This session is no longer recoverable. Refresh to confirm the latest result.',
      reviewGenerationRetry:
        'Your answer is already saved, but review generation failed. The page will switch to a recoverable state so you can retry the review directly.',
      answerLockedWhileProcessing:
        'The system is still processing, so the answer box is locked to avoid duplicate submissions.',
      submittedAnswerTitle: 'Answer received',
      submittedAnswerDescription:
        'The system is processing this answer, so it is shown as read-only for now.',
      followupIntentTitle: 'What this follow-up is checking',
      suggestionTitle: 'How to answer better next time',
      suggestionFallback:
        'The system has identified the main gap and will keep pushing on it in the next round.',
      feedbackHeadlineFallback:
        'The system has finished the first-pass judgment. Here is the most important feedback.',
      scoreBreakdownTitle: 'Score breakdown',
      reviewPendingTitle: 'Review is not finished yet',
      reviewPendingDescription:
        'Your answer is already saved, but review generation did not finish. You can retry the review directly without answering again.',
      retryReviewAction: 'Retry review generation',
      reviewWrapUpTitle: 'This round is complete',
      reviewWrapUpDescription:
        'The system is organizing your review card and will take you there in a moment.',
      jobTargetTitle: 'Bound job target',
      jobTargetDescription:
        'This round uses the requirements from this JD for the question, scoring, and review.',
      jobTargetEmpty:
        'This round is not bound to a job target JD and is still running in generic mode.',
      streamPending:
        'Content is being generated. The result will appear shortly.',
      streamSectionCounter: 'Step {index}',
      traceTitle: 'Runtime trace',
      traceEmpty:
        'No structured runtime trace has been recorded for this step yet.',
      streamKinds: {
        prepare: 'Preparing context',
        drafting: 'Generating content',
        finalizing: 'Finalizing result',
        processing: 'Processing',
        question: 'Question draft',
        evaluation: 'Evaluation draft',
        review: 'Review draft',
      },
      streamFields: {
        score: 'Current score: {score}',
        scoreBreakdown: 'Score breakdown',
        followupQuestion: 'Next follow-up',
      },
      tracePhases: {
        prepare_context: 'Prepare context',
        tool_call: 'Call tool',
        validate: 'Validate output',
        fallback: 'Fallback',
        finalize: 'Finalize result',
        error: 'Exit with error',
      },
      traceStatuses: {
        info: 'Running',
        success: 'Success',
        retry: 'Retry',
        fallback: 'Fallback',
        error: 'Error',
      },
      streamContexts: {
        read_question_templates: 'Reviewing question bank',
        read_project_brief: 'Reviewing project details',
        read_context_chunks: 'Reviewing code and docs',
        read_weakness_memory: 'Reviewing past records',
        read_evaluation_context: 'Analyzing your answer',
        read_session_summary: 'Summarizing this session',
        read_turn_history: 'Reviewing past Q&A',
        read_repo_overview: 'Reviewing project structure',
        read_repo_chunks: 'Reviewing key code',
        read_job_target_source: 'Reviewing job requirements',
        read_job_target_analysis: 'Reviewing job analysis',
      },
    },
    progress: {
      createSession: {
        title: 'Preparing your session',
        description:
          'The system reads context first and then generates the question.',
        steps: [
          'Read training context',
          'Generate question',
          'Prepare the session view',
        ],
      },
      evaluateMain: {
        title: 'Evaluating the main answer',
        description:
          'The system evaluates the answer first and then prepares the follow-up.',
        steps: [
          'Answer received',
          'Evaluating answer',
          'Follow-up and feedback ready',
        ],
      },
      evaluateFollowup: {
        title: 'Generating the review',
        description:
          'The system completes the follow-up evaluation and then prepares the review card.',
        steps: [
          'Follow-up answer received',
          'Final evaluation completed',
          'Review card ready',
        ],
      },
    },
    review: {
      hero: {
        kicker: 'Review',
        title:
          'The review summarizes this session and highlights what to practice next.',
        loading: 'Loading review...',
      },
      headerError: 'The review is temporarily unavailable.',
      loadingTitle: 'Loading review',
      loadErrorTitle: 'Failed to load review',
      exportErrorTitle: 'Export failed',
      emptyTitle: 'This review is not ready yet',
      emptyDescription:
        'It may still be generating, or the session did not finish successfully.',
      exportAction: 'Export {format}',
      exportingAction: 'Exporting...',
      promptExperimentAction: 'View prompt experiments',
      topFixTitle: 'Top fix',
      topFixFallbackReason:
        'Fix this first. It will improve the payoff of the next training round the most.',
      scoreBreakdown: 'Score breakdown',
      highlights: 'Highlights',
      gaps: 'Needs improvement',
      nextFocus: 'Next focus',
      recommendedNextTitle: 'Recommended next round',
      recommendedNextBasics: 'Start with a {topic} fundamentals round',
      recommendedNextMode: 'Start a {mode} round next',
      continueAction: 'Continue training',
      startRecommendedAction: 'Start recommended round',
      jobTargetTitle: 'Bound job target',
      jobTargetDescription:
        'This review was written from the perspective of this JD.',
      promptSetTitle: 'Prompt set used',
      promptSetDescription:
        'This session is bound to a prompt set with status: {status}.',
      auditShowAction: 'Show evaluation details',
      auditHideAction: 'Hide evaluation details',
      auditTitle: 'Evaluation audit details',
      auditDescription:
        'Expand each flow to inspect model, prompt hash, latency, and raw output.',
      auditErrorTitle: 'Failed to load evaluation details',
      auditMeta: 'Model / prompt hash / latency',
      auditRawOutput: 'Raw output',
      auditEmpty: 'No evaluation logs are available for this review yet.',
      runtimeTraceTitle: 'Runtime trace',
      retrievalTraceTitle: 'Retrieval trace',
      retrievalTraceDescription:
        'This shows which long-term memory items were actually recalled during review generation.',
      retrievalTraceObservations: 'Observations',
      retrievalTraceSummaries: 'Session Summaries',
      retrievalTraceEmpty: 'No hit was recorded for this material group.',
    },
    promptExperiments: {
      hero: {
        kicker: 'Prompt Experiments',
        title: 'Compare prompt versions by result quality and latency.',
        description:
          'This page compares versions across multiple sessions instead of double-running the same session.',
      },
      loadErrorTitle: 'Failed to load prompt sets',
      compareErrorTitle: 'Failed to load experiment results',
      filters: {
        left: 'Left version',
        right: 'Right version',
        mode: 'Mode',
        topic: 'Topic',
        allModes: 'All modes',
        allTopics: 'All topics',
      },
      metrics: {
        sessionCount: 'Session count',
        completedCount: 'Completed count',
        avgTotalScore: 'Average total score',
        avgQuestionLatency: 'Avg question latency',
        avgAnswerLatency: 'Avg evaluation latency',
        avgReviewLatency: 'Avg review latency',
      },
      samples: {
        kicker: 'Samples',
        title: 'Recent samples',
        empty: 'No comparable samples under the current filters.',
        showLogs: 'Show audit logs',
        hideLogs: 'Hide audit logs',
      },
      logs: {
        errorTitle: 'Failed to load logs',
        empty: 'This session has no audit log rows yet.',
        modelName: 'Model',
        promptHash: 'Prompt hash',
        latency: 'Latency',
      },
    },
  },
} as const;
