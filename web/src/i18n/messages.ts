export const messages = {
  'zh-CN': {
    app: {
      name: 'PracticeHelper',
      title: '面试训练助手',
      language: '语言',
      importNotice: '后台正在导入项目：{stage}',
      locales: {
        zhCN: '中文',
        en: 'English',
      },
      nav: {
        home: '首页',
        profile: '画像',
        projects: '项目',
        train: '训练',
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
    },
    enums: {
      mode: {
        basics: '基础训练',
        project: '项目训练',
      },
      intensity: {
        light: '轻量',
        standard: '标准',
        pressure: '强化',
      },
      topic: {
        go: 'Go',
        redis: 'Redis',
        kafka: 'Kafka',
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
      weaknessKind: {
        topic: '知识点',
        project: '项目表达',
        expression: '表达方式',
        followup_breakdown: '追问应对',
        depth: '展开深度',
        detail: '细节支撑',
      },
    },
    home: {
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
      currentSession: {
        kicker: '继续训练',
        title: '你有一轮训练还没结束',
        description: '{name} · {status}',
        emptyTitle: '还没有进行中的训练',
        emptyDescription: '从基础训练或项目训练开始一轮新的练习。',
      },
      cards: {
        weaknessKicker: 'Top Issue',
        weaknessTitle: '当前短板',
        weaknessEmpty: '还没有薄弱点记录，完成第一轮训练后这里会开始沉淀。',
        weaknessDescription:
          '当前最需要补的是 {kind} / {label}，建议优先围绕这个点练习。',
        sessionKicker: 'Recent',
        sessionTitle: '最近训练',
        sessionEmpty: '还没有历史训练记录，完成第一轮后这里会显示最近结果。',
        sessionDescription: '最近一轮是 {name}，当前状态为 {status}。',
        trackKicker: 'Focus',
        trackTitle: '推荐专项',
        profileKicker: 'Profile',
        profileTitle: '目标画像',
        profileEmpty: '画像还没初始化，系统暂时无法给出更准确的建议。',
        profileDescription: '{role} / {stage}，主讲项目：{projects}。',
      },
      sections: {
        weaknesses: '薄弱点',
        sessions: '训练记录',
        weaknessesEmpty:
          '还没有历史薄弱点，完成第一轮训练后这里会显示重点问题。',
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
      importPlaceholder: 'https://github.com/yourname/your-project',
      importAction: '导入仓库',
      importErrorTitle: '导入失败',
      retryAction: '重试导入',
      retryErrorTitle: '重试失败',
      jobsTitle: '后台导入任务',
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
    train: {
      hero: {
        kicker: '训练设置',
        title: '先选对训练模式，再开始练习。',
        description:
          '基础训练适合补知识点，项目训练适合练项目讲述。当前版本每轮包含主问题和一次追问。',
      },
      fields: {
        mode: '训练模式',
        intensity: '训练强度',
        topic: '主题',
        project: '选择项目',
      },
      resumeTitle: '继续当前训练',
      resumeDescription: '{name} · {status}',
      chooseProject: '请选择项目',
      startAction: '开始这一轮训练',
      startErrorTitle: '启动失败',
    },
    session: {
      hero: {
        kicker: '训练过程',
        title: '先按真实面试场景作答。',
        description:
          '当前状态：{status}。系统会先评估主问题，再决定是否继续追问。',
      },
      currentQuestion: '当前问题',
      feedback: '过程反馈',
      mainScore: '主问题得分：{score}',
      strengths: '优点',
      gaps: '待补强点',
      feedbackEmpty: '先回答当前问题，反馈会显示在这里。',
      processingKicker: '处理中',
      reasoningTitle: '推理摘要',
      contentTitle: '模型输出',
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
      submitErrorTitle: '提交失败',
      conflictBusy: '上一轮提交还在处理中，请等待当前评估或复盘完成。',
      conflictReviewPending:
        '上一轮回答已经保存，当前只差复盘。请直接使用“重试生成复盘”。',
      conflictCompleted: '这轮训练已经完成，页面会跳转到复盘结果。',
      conflictInvalidStatus: '当前状态不能继续提交，请等待页面刷新到最新状态。',
      retryReviewNotRecoverable:
        '这轮训练已经不在可恢复状态，请刷新页面确认最新结果。',
      reviewGenerationRetry:
        '回答已经保存，但复盘生成失败了。页面会切到可恢复状态，你可以直接重试复盘。',
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
      reviewPendingTitle: '复盘还没收口',
      reviewPendingDescription:
        '上一轮回答已经保存，但复盘生成没有完成。你可以直接重试复盘，不需要重新回答。',
      retryReviewAction: '重试生成复盘',
      reviewWrapUpTitle: '本轮问答已完成',
      reviewWrapUpDescription: '系统正在整理复盘卡，马上带你进入总结页面。',
      streamPending: '模型已经开始返回内容，但还在整理成结构化结果，请稍候。',
      streamSectionCounter: '过程片段 {index}',
      streamKinds: {
        prepare: '正在准备上下文',
        drafting: '模型生成中',
        finalizing: '正在整理结果',
        processing: '处理中',
        question: '问题草稿',
        evaluation: '评估结果',
        review: '复盘草稿',
      },
      streamFields: {
        score: '当前得分：{score}',
        scoreBreakdown: '分项得分',
        followupQuestion: '下一刀追问',
      },
      streamContexts: {
        read_question_templates: '已读取基础题模板',
        read_project_brief: '已读取项目摘要与亮点',
        read_context_chunks: '已读取源码与文档片段',
        read_weakness_memory: '已读取历史薄弱点',
        read_evaluation_context: '已读取当前题目与回答',
        read_session_summary: '已读取整轮训练摘要',
        read_turn_history: '已读取历史问答记录',
        read_repo_overview: '已读取仓库概览',
        read_repo_chunks: '已读取关键源码片段',
      },
    },
    progress: {
      createSession: {
        title: '正在准备训练',
        description: '系统会先读取上下文，再生成当前问题。',
        steps: ['读取训练上下文', '调用模型生成问题', '整理训练页面'],
      },
      evaluateMain: {
        title: '正在评估主问题回答',
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
      emptyTitle: '这轮复盘还没准备好',
      emptyDescription: '可能还在生成中，或者这轮训练还没有成功收口。',
      topFixTitle: '最该优先修正',
      topFixFallbackReason: '先把这个问题修正掉，后面的训练收益会明显更高。',
      scoreBreakdown: '分项得分',
      highlights: '回答亮点',
      gaps: '需要补强',
      nextFocus: '下一轮重点',
      recommendedNextTitle: '推荐下一轮',
      recommendedNextBasics: '建议先做一轮 {topic} 基础训练',
      recommendedNextMode: '建议直接开始一轮 {mode}',
      continueAction: '继续下一轮',
      startRecommendedAction: '立即开始推荐下一轮',
    },
  },
  en: {
    app: {
      name: 'PracticeHelper',
      title: 'Interview Practice Assistant',
      language: 'Language',
      importNotice: 'Background import running: {stage}',
      locales: {
        zhCN: 'Chinese',
        en: 'English',
      },
      nav: {
        home: 'Home',
        profile: 'Profile',
        projects: 'Projects',
        train: 'Train',
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
    },
    enums: {
      mode: {
        basics: 'Fundamentals',
        project: 'Project',
      },
      intensity: {
        light: 'Light',
        standard: 'Standard',
        pressure: 'Intensive',
      },
      topic: {
        go: 'Go',
        redis: 'Redis',
        kafka: 'Kafka',
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
      weaknessKind: {
        topic: 'Knowledge area',
        project: 'Project explanation',
        expression: 'Communication',
        followup_breakdown: 'Follow-up handling',
        depth: 'Depth',
        detail: 'Supporting detail',
      },
    },
    home: {
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
      currentSession: {
        kicker: 'Resume',
        title: 'You have a training session in progress',
        description: '{name} · {status}',
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
        profileKicker: 'Profile',
        profileTitle: 'Target profile',
        profileEmpty:
          'Your profile is not set up yet, so recommendations are still generic.',
        profileDescription: '{role} / {stage}, primary projects: {projects}.',
      },
      sections: {
        weaknesses: 'Weaknesses',
        sessions: 'Training history',
        weaknessesEmpty:
          'No historical weaknesses yet. They will appear after the first session.',
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
      importPlaceholder: 'https://github.com/yourname/your-project',
      importAction: 'Import repository',
      importErrorTitle: 'Import failed',
      retryAction: 'Retry import',
      retryErrorTitle: 'Retry failed',
      jobsTitle: 'Background import jobs',
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
    train: {
      hero: {
        kicker: 'Training',
        title: 'Choose the right mode, then start practicing.',
        description:
          'Fundamentals mode is for core knowledge. Project mode is for project storytelling. The current version includes a main question and one follow-up.',
      },
      fields: {
        mode: 'Mode',
        intensity: 'Intensity',
        topic: 'Topic',
        project: 'Project',
      },
      resumeTitle: 'Resume current session',
      resumeDescription: '{name} · {status}',
      chooseProject: 'Select a project',
      startAction: 'Start this session',
      startErrorTitle: 'Start failed',
    },
    session: {
      hero: {
        kicker: 'Session',
        title: 'Answer as if you were in a real interview.',
        description:
          'Current status: {status}. The system will evaluate the main answer first and then decide whether a follow-up is needed.',
      },
      currentQuestion: 'Current question',
      feedback: 'Feedback',
      mainScore: 'Main question score: {score}',
      strengths: 'Strengths',
      gaps: 'Gaps',
      feedbackEmpty:
        'Answer the current question first. Feedback will appear here.',
      processingKicker: 'Processing',
      reasoningTitle: 'Reasoning Summary',
      contentTitle: 'Model Output',
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
      streamPending:
        'The model is already returning content, but the structured result is still being assembled.',
      streamSectionCounter: 'Stream block {index}',
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
      streamContexts: {
        read_question_templates: 'Question templates loaded',
        read_project_brief: 'Project brief loaded',
        read_context_chunks: 'Code and docs context loaded',
        read_weakness_memory: 'Weakness history loaded',
        read_evaluation_context: 'Current question and answer loaded',
        read_session_summary: 'Session summary loaded',
        read_turn_history: 'Turn history loaded',
        read_repo_overview: 'Repository overview loaded',
        read_repo_chunks: 'Key source chunks loaded',
      },
    },
    progress: {
      createSession: {
        title: 'Preparing your session',
        description:
          'The system reads context first and then generates the question.',
        steps: [
          'Read training context',
          'Call the model',
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
      emptyTitle: 'This review is not ready yet',
      emptyDescription:
        'It may still be generating, or the session did not finish successfully.',
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
    },
  },
} as const;
