# ruff: noqa: F403,F405
from runtime_test_support import *


def test_generate_question_uses_tool_loop_before_returning_json() -> None:
    runtime = AgentRuntime(
        Settings(
            github_token="",
            model="test-model",
            openai_base_url="http://example.com/v1",
            openai_api_key="test-key",
            llm_timeout_seconds=10,
        ),
        model_client=FakeModelClient(
            [
                ChatCompletionResult(
                    content="",
                    tool_calls=[
                        {
                            "id": "call_templates",
                            "function": {
                                "name": "read_question_templates",
                                "arguments": "{}",
                            },
                        }
                    ],
                ),
                ChatCompletionResult(
                    content=(
                        '{"question":"请讲讲 Mirror 的 trade-off。","expected_points":'
                        '["问题背景","技术选型理由","trade-off","真实结果"]}'
                    ),
                    tool_calls=[],
                ),
            ]
        ),
    )

    response = runtime.generate_question(
        GenerateQuestionRequest(
            mode="project",
            intensity="standard",
            project=ProjectProfile(
                name="Mirror",
                summary="Agent workflow",
                followup_points=["trade-off"],
            ),
        )
    )

    assert "Mirror" in response.question
    assert "trade-off" in response.expected_points


def test_generate_question_falls_back_to_single_shot_when_model_skips_tools() -> None:
    client = FakeModelClient(
        [
            ChatCompletionResult(
                content='{"question":"请讲讲项目里的 ownership。","expected_points":["ownership"]}',
                tool_calls=[],
            ),
            ChatCompletionResult(
                content=(
                    '{"question":"请讲讲 Mirror 里你负责的 ownership。","expected_points":'
                    '["模块边界","关键取舍","真实结果","后续改进"]}'
                ),
                tool_calls=[],
            ),
        ]
    )
    runtime = AgentRuntime(
        Settings(
            github_token="",
            model="test-model",
            openai_base_url="http://example.com/v1",
            openai_api_key="test-key",
            llm_timeout_seconds=10,
        ),
        model_client=client,
    )

    response = runtime.generate_question(
        GenerateQuestionRequest(
            mode="project",
            intensity="standard",
            project=ProjectProfile(
                name="Mirror",
                summary="Agent workflow",
                followup_points=["ownership"],
            ),
        )
    )

    assert response.question == "请讲讲 Mirror 里你负责的 ownership。"
    assert len(client.calls) == 2
    assert "下面是你已经可以直接使用的上下文" in client.calls[1]["messages"][1]["content"]
    assert "read_project_brief" in client.calls[1]["messages"][1]["content"]


def test_generate_question_can_use_search_repo_chunks_callback_tool() -> None:
    backend_client = FakeBackendClient()
    runtime = AgentRuntime(
        Settings(
            github_token="",
            model="test-model",
            openai_base_url="http://example.com/v1",
            openai_api_key="test-key",
            llm_timeout_seconds=10,
            server_base_url="http://127.0.0.1:8090",
            internal_token="secret-token",
        ),
        model_client=FakeModelClient(
            [
                ChatCompletionResult(
                    content="",
                    tool_calls=[
                        {
                            "id": "call_search",
                            "function": {
                                "name": "search_repo_chunks",
                                "arguments": '{"query":"retry path","limit":2}',
                            },
                        }
                    ],
                ),
                ChatCompletionResult(
                    content=(
                        '{"question":"讲讲项目里的重试链路设计。",'
                        '"expected_points":["触发条件","兜底策略","一致性"]}'
                    ),
                    tool_calls=[],
                ),
            ]
        ),
        go_client=backend_client,
    )

    response = runtime.generate_question(
        GenerateQuestionRequest(
            mode="project",
            intensity="standard",
            project=ProjectProfile(
                id="proj_1",
                name="Mirror",
                summary="Agent workflow",
                followup_points=["retry"],
            ),
        )
    )

    assert response.question == "讲讲项目里的重试链路设计。"
    assert backend_client.search_queries == [("proj_1", "retry path", 2)]


def test_analyze_job_target_returns_structured_snapshot() -> None:
    runtime = AgentRuntime(
        Settings(
            github_token="",
            model="test-model",
            openai_base_url="http://example.com/v1",
            openai_api_key="test-key",
            llm_timeout_seconds=10,
        ),
        model_client=FakeModelClient(
            [
                ChatCompletionResult(
                    content=(
                        '{"summary":"核心是在招能独立推进高并发后端系统的人。",'
                        '"must_have_skills":["Go","Redis","Kafka"],'
                        '"bonus_skills":["Kubernetes"],'
                        '"responsibilities":["负责核心服务设计"],'
                        '"evaluation_focus":["并发设计取舍"]}'
                    ),
                    tool_calls=[],
                ),
                ChatCompletionResult(
                    content=(
                        '{"summary":"核心是在招能独立推进高并发后端系统的人。",'
                        '"must_have_skills":["Go","Redis","Kafka"],'
                        '"bonus_skills":["Kubernetes"],'
                        '"responsibilities":["负责核心服务设计"],'
                        '"evaluation_focus":["并发设计取舍"]}'
                    ),
                    tool_calls=[],
                ),
            ]
        ),
    )

    response = runtime.analyze_job_target(
        AnalyzeJobTargetRequest(
            title="后端工程师",
            company_name="Example",
            source_text="负责高并发后端服务开发，要求 Go、Redis、Kafka 经验。",
        )
    )

    assert response.summary
    assert response.must_have_skills == ["Go", "Redis", "Kafka"]
    assert response.evaluation_focus == ["并发设计取舍"]


def test_question_prompt_bundle_includes_all_basics_templates() -> None:
    system_prompt, user_prompt, tools = question_prompt_bundle(
        GenerateQuestionRequest(
            mode="basics",
            topic="go",
            intensity="standard",
            templates=[
                QuestionTemplate(mode="basics", topic="go", prompt="问题1"),
                QuestionTemplate(mode="basics", topic="go", prompt="问题2"),
                QuestionTemplate(mode="basics", topic="go", prompt="问题3"),
                QuestionTemplate(mode="basics", topic="go", prompt="问题4"),
                QuestionTemplate(mode="basics", topic="go", prompt="问题5"),
            ],
        )
    )

    assert "主问题" in user_prompt
    templates_payload = tools[0].handler({})
    assert len(templates_payload["templates"]) == 5
    assert templates_payload["templates"][0]["prompt"] == "问题1"


def test_question_prompt_bundle_includes_job_target_analysis_when_present() -> None:
    _, user_prompt, tools = question_prompt_bundle(
        GenerateQuestionRequest(
            mode="basics",
            topic="redis",
            intensity="standard",
            job_target_analysis=JobTargetAnalysisSnapshot(
                summary="偏高并发后端",
                must_have_skills=["Redis", "缓存一致性"],
                evaluation_focus=["高并发缓存设计"],
            ),
        )
    )

    assert "是否绑定岗位 JD：有" in user_prompt
    payload = tools[-1].handler({})
    assert payload["job_target_analysis"]["must_have_skills"] == ["Redis", "缓存一致性"]


def test_question_prompt_bundle_marks_mixed_mode_candidate_topics() -> None:
    _, user_prompt, tools = question_prompt_bundle(
        GenerateQuestionRequest(
            mode="basics",
            topic="mixed",
            candidate_topics=["redis", "mysql", "os"],
            intensity="standard",
            templates=[
                QuestionTemplate(mode="basics", topic="redis", prompt="问题1"),
                QuestionTemplate(mode="basics", topic="mysql", prompt="问题2"),
            ],
        )
    )

    assert "这是基础混合模式" in user_prompt
    assert "redis, mysql, os" in user_prompt
    payload = tools[0].handler({})
    assert payload["candidate_topics"] == ["redis", "mysql", "os"]


def test_question_prompt_bundle_resolves_strategy_from_weaknesses() -> None:
    _, user_prompt, _ = question_prompt_bundle(
        GenerateQuestionRequest(
            mode="basics",
            topic="redis",
            intensity="standard",
            weaknesses=[
                WeaknessTag(
                    kind="topic",
                    label="缓存一致性",
                    severity=0.95,
                    frequency=3,
                )
            ],
        )
    )

    assert "优先围绕用户历史弱项出题" in user_prompt


def test_generate_question_graph_passes_selected_strategy_to_runtime() -> None:
    runtime = FakeQuestionGraphRuntime()
    graph = _build_generate_question_graph(runtime)

    graph.invoke(
        {
            "request": GenerateQuestionRequest(
                mode="project",
                intensity="standard",
                project=ProjectProfile(name="Mirror", summary="Agent workflow"),
            )
        }
    )

    assert len(runtime.requests) == 1
    assert runtime.requests[0].strategy == "project_deep_dive"


def test_analyze_repo_graph_reranks_chunks_before_summarizing() -> None:
    runtime = FakeAnalyzeRepoGraphRuntime()
    graph = _build_analyze_repo_graph(runtime)

    result = graph.invoke(
        {"request": AnalyzeRepoRequest(repo_url="https://example.com/mirror.git")}
    )

    assert isinstance(result["result"], AnalyzeRepoEnvelope)
    assert runtime.summarized_bundle is not None
    assert runtime.summarized_bundle.chunks[0].file_path == "internal/agent/runtime.go"


def test_evaluate_prompt_bundle_requires_conservative_followup_when_evidence_is_thin() -> None:
    system_prompt, user_prompt, tools = evaluate_prompt_bundle(
        EvaluateAnswerRequest(
            mode="basics",
            topic="redis",
            question="Redis 为什么快？",
            expected_points=["内存访问", "事件循环"],
            answer="因为它在内存里。",
            turn_index=1,
            max_turns=2,
        )
    )

    assert "证据不足，要用保守表达追问" in system_prompt
    assert "不要把未证实的经历、做法、线上事故或项目事实写成既定前提" in system_prompt
    payload = tools[0].handler({})
    assert payload["turn_index"] == 1
    assert "是否为追问回答：否" in user_prompt


def test_evaluate_prompt_bundle_includes_job_target_analysis_context() -> None:
    _, user_prompt, tools = evaluate_prompt_bundle(
        EvaluateAnswerRequest(
            mode="project",
            question="你怎么处理缓存一致性？",
            expected_points=["先定义一致性目标"],
            answer="我会先定策略，再看写路径。",
            job_target_analysis=JobTargetAnalysisSnapshot(
                summary="看重高并发缓存架构",
                must_have_skills=["缓存一致性"],
                responsibilities=["负责核心链路稳定性"],
                evaluation_focus=["故障排查闭环"],
            ),
        )
    )

    assert "是否绑定岗位 JD：有" in user_prompt
    payload = tools[0].handler({})
    assert payload["job_target_analysis"]["evaluation_focus"] == ["故障排查闭环"]


def test_evaluate_prompt_bundle_marks_followup_requests() -> None:
    _, user_prompt, tools = evaluate_prompt_bundle(
        EvaluateAnswerRequest(
            mode="project",
            topic="",
            question="如果线上报警频繁，你会怎么止血？",
            expected_points=["先止血", "再排查"],
            answer="我会先降级，再看指标。",
            turn_index=2,
            max_turns=2,
        )
    )

    payload = tools[0].handler({})
    assert payload["turn_index"] == 2
    assert "是否为追问回答：是" in user_prompt
    assert "最后一轮" in user_prompt


def test_evaluate_prompt_bundle_includes_retry_feedback_when_present() -> None:
    _, user_prompt, _ = evaluate_prompt_bundle(
        EvaluateAnswerRequest(
            mode="basics",
            topic="redis",
            question="Redis 为什么快？",
            expected_points=["内存访问", "事件循环"],
            answer="因为它在内存里。",
            turn_index=1,
            max_turns=2,
            retry_feedback="missing strengths/gaps",
        )
    )

    assert "上一次输出没有过校验" in user_prompt
    assert "missing strengths/gaps" in user_prompt


def test_prepare_generate_review_agent_tooling_compacts_turns_and_memory_context() -> None:
    request = GenerateReviewRequest(
        session=TrainingSession(
            id="sess_compact",
            mode="basics",
            topic="redis",
            status="review_pending",
            max_turns=2,
            total_score=81,
        ),
        turns=[
            TrainingTurn(
                turn_index=1,
                question="Redis 为什么快？",
                answer="A" * 320,
                evaluation=EvaluationResult(
                    score=81,
                    score_breakdown={"准确性": 81},
                    headline="主线基本完整，但案例深度还不够。",
                    strengths=["主线清楚", "先讲结论", "能对上场景"],
                    gaps=["案例不够具体", "trade-off 不够展开", "监控闭环没说"],
                    weakness_hits=[],
                ),
            )
        ],
        agent_context={
            "knowledge_subgraph": {
                "nodes": [
                    {
                        "id": "node_1",
                        "label": "redis",
                        "node_type": "topic",
                        "proficiency": 3.0,
                        "confidence": 0.8,
                        "scope_type": "global",
                        "scope_id": "",
                        "parent_id": "",
                    }
                ],
                "edges": [
                    {
                        "source_id": "node_1",
                        "target_id": "node_2",
                        "edge_type": "related",
                    }
                ],
            },
            "observations": [
                {
                    "id": "obs_1",
                    "category": "pattern",
                    "content": "B" * 260,
                    "tags": ["redis", "tradeoff", "ops", "detail", "extra"],
                    "topic": "redis",
                    "scope_type": "global",
                    "scope_id": "",
                    "relevance": 0.9,
                }
            ],
            "session_summaries": [
                {
                    "id": "sum_1",
                    "session_id": "sess_prev",
                    "mode": "basics",
                    "topic": "redis",
                    "summary": "C" * 320,
                    "strengths": ["主线清楚", "会讲场景", "表达稳定"],
                    "gaps": ["细节不足", "闭环不够", "观测面弱"],
                    "recommended_focus": ["缓存一致性", "热点 key", "监控"],
                    "salience": 0.7,
                }
            ],
            "weakness_profile": [
                {
                    "id": "weak_1",
                    "kind": "topic",
                    "label": "缓存一致性",
                    "severity": 0.9,
                    "frequency": 3,
                    "last_seen_at": "",
                    "evidence_session_id": "sess_prev",
                }
            ],
        },
    )

    prepared = prepare_generate_review_agent_tooling(request)

    turn = prepared.training_context_payload["turns"][0]
    assert "answer" not in turn
    assert len(turn["answer_excerpt"]) <= 240
    assert len(turn["top_strengths"]) == 2
    assert len(turn["top_gaps"]) == 2

    observation = prepared.observations_payload["observations"][0]
    assert "created_at" not in observation
    assert len(observation["content"]) <= 180
    assert len(observation["tags"]) == 4

    summary = prepared.session_summaries_payload["session_summaries"][0]
    assert len(summary["summary"]) <= 240
    assert len(summary["strengths"]) == 2
    assert len(summary["gaps"]) == 2
    assert len(summary["recommended_focus"]) == 2

    sections = {detail["section"] for detail in prepared.trace_details}
    assert {"turns", "knowledge_subgraph", "observations", "session_summaries"} <= sections


def test_review_prompt_bundle_includes_job_target_analysis_context() -> None:
    _, user_prompt, tools = review_prompt_bundle(
        GenerateReviewRequest(
            session=TrainingSession(
                id="sess_1",
                mode="project",
                project_id="proj_1",
                job_target_id="jt_1",
                job_target_analysis_id="jta_1",
            ),
            turns=[
                TrainingTurn(
                    question="你怎么处理缓存一致性？",
                    expected_points=["一致性目标", "失败兜底"],
                    answer="我会先定目标，再看写路径。",
                )
            ],
            job_target_analysis=JobTargetAnalysisSnapshot(
                summary="看重高并发缓存架构",
                must_have_skills=["缓存一致性"],
                evaluation_focus=["故障排查闭环"],
            ),
        )
    )

    assert "是否绑定岗位 JD：有" in user_prompt
    payload = tools[0].handler({})
    assert payload["job_target_analysis"]["must_have_skills"] == ["缓存一致性"]


def test_review_prompt_bundle_includes_retry_feedback_when_present() -> None:
    _, user_prompt, _ = review_prompt_bundle(
        GenerateReviewRequest(
            session=TrainingSession(id="sess_1", mode="basics", topic="redis"),
            turns=[],
            retry_feedback="missing top_fix",
        )
    )

    assert "上一次输出没有过校验" in user_prompt
    assert "missing top_fix" in user_prompt
