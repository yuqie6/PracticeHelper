import sys
from pathlib import Path

import pytest

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))

from app.agent_runtime import AgentRuntime, TaskExecutionResult
from app.agent_tools import build_generate_review_agent_tools
from app.config import Settings
from app.langgraph_flows import (
    _build_analyze_repo_graph,
    _build_evaluate_answer_graph,
    _build_generate_question_graph,
    _build_generate_review_graph,
)
from app.llm_client import ChatCompletionResult, ChatCompletionStreamChunk, ModelClientError
from app.repo_context import RepoAnalysisBundle
from app.runtime_prompts import (
    evaluate_prompt_bundle,
    question_prompt_bundle,
    review_prompt_bundle,
)
from app.schemas import (
    AgentSessionDetail,
    AnalyzeJobTargetRequest,
    AnalyzeRepoEnvelope,
    AnalyzeRepoRequest,
    AnalyzeRepoResponse,
    EvaluateAnswerEnvelope,
    EvaluateAnswerRequest,
    EvaluationResult,
    GenerateQuestionRequest,
    GenerateQuestionResponse,
    GenerateReviewEnvelope,
    GenerateReviewRequest,
    JobTargetAnalysisSnapshot,
    ProjectProfile,
    QuestionTemplate,
    RepoChunk,
    ReviewCard,
    TrainingSession,
    TrainingTurn,
    WeaknessHit,
    WeaknessTag,
)


class FakeModelClient:
    def __init__(self, responses: list[ChatCompletionResult]) -> None:
        self._responses = responses
        self.calls: list[dict] = []

    def create_completion(self, *, messages, tools=None, temperature=0.2):
        self.calls.append({"messages": messages, "tools": tools, "temperature": temperature})
        if not self._responses:
            raise AssertionError("fake client has no more responses")
        return self._responses.pop(0)


class FakeStreamModelClient(FakeModelClient):
    def __init__(self, responses: list[ChatCompletionResult]) -> None:
        super().__init__(responses)

    def create_completion_stream(self, *, messages, temperature=0.2):
        self.calls.append({"messages": messages, "temperature": temperature, "stream": True})
        yield ChatCompletionStreamChunk(
            content='{"question":"请讲讲 Redis 一致性。","expected_points":["主线","取舍"]}'
        )


class FakeEvaluateStreamModelClient(FakeModelClient):
    def __init__(self, responses: list[ChatCompletionResult]) -> None:
        super().__init__(responses)

    def create_completion_stream(self, *, messages, temperature=0.2):
        self.calls.append({"messages": messages, "temperature": temperature, "stream": True})
        yield ChatCompletionStreamChunk(
            content=(
                '{"score":86,"score_breakdown":{"准确性":86},"headline":"主线基本清楚",'
                '"strengths":["回答主线清楚"],"gaps":["例子不够具体"],"suggestion":"补真实案例",'
                '"weakness_hits":[]}'
            )
        )


class FakeReviewStreamModelClient(FakeModelClient):
    def __init__(self, responses: list[ChatCompletionResult]) -> None:
        super().__init__(responses)

    def create_completion_stream(self, *, messages, temperature=0.2):
        self.calls.append({"messages": messages, "temperature": temperature, "stream": True})
        yield ChatCompletionStreamChunk(
            content=(
                '{"overall":"总结","top_fix":"先补关键缺口","top_fix_reason":"这是最大短板",'
                '"highlights":["主线清楚"],"gaps":["案例不够具体"],'
                '"suggested_topics":["redis"],"next_training_focus":["补细节"],'
                '"recommended_next":{"mode":"basics","topic":"redis","reason":"补短板"},'
                '"score_breakdown":{"准确性":72}}'
            )
        )


class FakeQuestionGraphRuntime:
    def __init__(self) -> None:
        self.requests: list[GenerateQuestionRequest] = []

    def generate_question_task(
        self, request: GenerateQuestionRequest
    ) -> TaskExecutionResult[GenerateQuestionResponse]:
        self.requests.append(request)
        return TaskExecutionResult(
            result=GenerateQuestionResponse(question="问题", expected_points=["点1"]),
            raw_output='{"question":"问题","expected_points":["点1"]}',
        )


class FakeBackendClient:
    def __init__(self) -> None:
        self.enabled = True
        self.search_queries: list[tuple[str, str, int]] = []
        self.session_ids: list[str] = []

    def search_repo_chunks(self, project_id: str, query: str, limit: int = 6) -> list[RepoChunk]:
        self.search_queries.append((project_id, query, limit))
        return [
            RepoChunk(
                file_path="internal/cache.go",
                file_type=".go",
                content="retry and cache consistency",
                importance=1.0,
                fts_key="internal/cache.go#0",
            )
        ]

    def get_session_detail(self, session_id: str):
        self.session_ids.append(session_id)
        return AgentSessionDetail(
            session=TrainingSession(id=session_id, mode="basics", topic="redis", turns=[]),
            review=None,
        )


class FakeAnalyzeRepoGraphRuntime:
    def __init__(self) -> None:
        self.summarized_bundle: RepoAnalysisBundle | None = None

    def collect_repo_bundle(self, request: AnalyzeRepoRequest) -> RepoAnalysisBundle:
        return RepoAnalysisBundle(
            repo_url=request.repo_url,
            name="mirror",
            default_branch="main",
            import_commit="abc123",
            tech_stack=["Go", "LangGraph"],
            top_paths=["internal/agent/runtime.go"],
            chunks=[
                RepoChunk(
                    file_path="docs/notes.md",
                    file_type=".md",
                    content="misc misc misc",
                    importance=1.4,
                    fts_key="docs/notes.md#0",
                ),
                RepoChunk(
                    file_path="internal/agent/runtime.go",
                    file_type=".go",
                    content="agent runtime orchestrates LangGraph nodes",
                    importance=0.8,
                    fts_key="internal/agent/runtime.go#0",
                ),
            ],
        )

    def summarize_repo_bundle(
        self, bundle: RepoAnalysisBundle
    ) -> TaskExecutionResult[AnalyzeRepoResponse]:
        self.summarized_bundle = bundle
        return TaskExecutionResult(
            result=AnalyzeRepoResponse(
                repo_url=bundle.repo_url,
                name=bundle.name,
                default_branch=bundle.default_branch,
                import_commit=bundle.import_commit,
                summary="仓库主线清楚",
                tech_stack=bundle.tech_stack,
                highlights=["亮点"],
                challenges=["挑战"],
                tradeoffs=["取舍"],
                ownership_points=["owner"],
                followup_points=["follow"],
                chunks=bundle.chunks,
            ),
            raw_output='{"summary":"仓库主线清楚"}',
        )


class FlakyEvaluateGraphRuntime:
    def __init__(self) -> None:
        self.calls = 0
        self.retry_feedback: list[str] = []

    def evaluate_answer_task(
        self, request: EvaluateAnswerRequest
    ) -> TaskExecutionResult[EvaluationResult]:
        self.calls += 1
        self.retry_feedback.append(request.retry_feedback)
        if self.calls == 1:
            return TaskExecutionResult(
                result=EvaluationResult(score=80, score_breakdown={"准确性": 80}),
                raw_output='{"score":80}',
            )
        return TaskExecutionResult(
            result=EvaluationResult(
                score=82,
                score_breakdown={"准确性": 82},
                strengths=["主线清楚"],
                gaps=["例子不够具体"],
                followup_question="如果线上抖动，你会先看什么？",
                followup_expected_points=["先止血", "再定位"],
                weakness_hits=[],
            ),
            raw_output='{"score":82,"followup_question":"如果线上抖动，你会先看什么？"}',
        )


class AlwaysInvalidEvaluateGraphRuntime:
    def __init__(self) -> None:
        self.calls = 0

    def evaluate_answer_task(
        self, request: EvaluateAnswerRequest
    ) -> TaskExecutionResult[EvaluationResult]:
        self.calls += 1
        return TaskExecutionResult(
            result=EvaluationResult(score=80, score_breakdown={"准确性": 80}),
            raw_output='{"score":80}',
        )


class FlakyGenerateReviewGraphRuntime:
    def __init__(self) -> None:
        self.calls = 0
        self.retry_feedback: list[str] = []

    def generate_review_task(
        self, request: GenerateReviewRequest
    ) -> TaskExecutionResult[ReviewCard]:
        self.calls += 1
        self.retry_feedback.append(request.retry_feedback)
        if self.calls == 1:
            return TaskExecutionResult(
                result=ReviewCard(overall="总结", score_breakdown={"准确性": 70}),
                raw_output='{"overall":"总结"}',
            )
        return TaskExecutionResult(
            result=ReviewCard(
                overall="总结",
                top_fix="先补最关键缺口",
                top_fix_reason="这是当前最影响说服力的部分",
                highlights=["主线清楚"],
                gaps=["案例不够具体"],
                suggested_topics=["redis"],
                next_training_focus=["补细节"],
                recommended_next={"mode": "basics", "topic": "redis", "reason": "补短板"},
                score_breakdown={"准确性": 70},
            ),
            raw_output='{"overall":"总结","top_fix":"先补最关键缺口"}',
        )


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

    result = graph.invoke({"request": AnalyzeRepoRequest(repo_url="https://example.com/mirror.git")})

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


def test_evaluate_answer_graph_retries_after_invalid_output() -> None:
    runtime = FlakyEvaluateGraphRuntime()
    graph = _build_evaluate_answer_graph(runtime)

    result = graph.invoke(
        {
            "request": EvaluateAnswerRequest(
                mode="basics",
                topic="redis",
                question="Redis 为什么快？",
                expected_points=["内存访问", "事件循环"],
                answer="因为它在内存里。",
                turn_index=1,
                max_turns=2,
            )
        }
    )

    assert runtime.calls == 2
    assert isinstance(result["result"], EvaluateAnswerEnvelope)
    assert result["result"].result.followup_question == "如果线上抖动，你会先看什么？"
    assert runtime.retry_feedback[1] == "missing strengths/gaps"
    assert result["result"].side_effects.depth_signal == "normal"


def test_evaluate_answer_graph_raises_after_retry_budget_is_exhausted() -> None:
    runtime = AlwaysInvalidEvaluateGraphRuntime()
    graph = _build_evaluate_answer_graph(runtime)

    with pytest.raises(ModelClientError, match="missing strengths/gaps"):
        graph.invoke(
            {
                "request": EvaluateAnswerRequest(
                    mode="basics",
                    topic="redis",
                    question="Redis 为什么快？",
                    expected_points=["内存访问", "事件循环"],
                    answer="因为它在内存里。",
                    turn_index=1,
                    max_turns=2,
                )
            }
        )

    assert runtime.calls == 2


def test_generate_review_graph_retries_after_invalid_output() -> None:
    runtime = FlakyGenerateReviewGraphRuntime()
    graph = _build_generate_review_graph(runtime)

    result = graph.invoke(
        {
            "request": GenerateReviewRequest(
                session=TrainingSession(id="sess_1", mode="basics", topic="redis"),
                turns=[
                    TrainingTurn(
                        question="Redis 为什么快？",
                        expected_points=["内存访问", "事件循环"],
                        answer="因为它在内存里。",
                    )
                ],
            )
        }
    )

    assert runtime.calls == 2
    assert isinstance(result["result"], GenerateReviewEnvelope)
    assert result["result"].result.top_fix == "先补最关键缺口"
    assert runtime.retry_feedback[1] == "missing top_fix"


def test_evaluate_answer_returns_side_effects_from_action_tools() -> None:
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
                            "id": "call_ctx",
                            "function": {
                                "name": "recall_training_context",
                                "arguments": "{}",
                            },
                        }
                    ],
                ),
                ChatCompletionResult(
                    content="",
                    tool_calls=[
                        {
                            "id": "call_obs",
                            "function": {
                                "name": "record_observation",
                                "arguments": (
                                    '{"category":"pattern","content":"用户会讲主线，但案例不够具体。",'
                                    '"tags":["表达","案例"],"relevance":0.9,"topic":"redis"}'
                                ),
                            },
                        },
                        {
                            "id": "call_kn",
                            "function": {
                                "name": "update_knowledge",
                                "arguments": (
                                    '{"label":"redis","node_type":"topic","proficiency":2.5,'
                                    '"confidence":0.8,"evidence":"能讲清主线，但案例不足"}'
                                ),
                            },
                        },
                        {
                            "id": "call_depth",
                            "function": {
                                "name": "set_depth_signal",
                                "arguments": '{"depth_signal":"extend"}',
                            },
                        },
                    ],
                ),
                ChatCompletionResult(
                    content=(
                        '{"score":78,"score_breakdown":{"准确性":78},"headline":"主线清楚",'
                        '"strengths":["主线完整"],"gaps":["案例不够具体"],"suggestion":"补真实案例",'
                        '"followup_question":"如果线上抖动，你先看什么？",'
                        '"followup_expected_points":["先止血","再定位"],'
                        '"weakness_hits":[{"kind":"detail","label":"案例不够具体","severity":0.7}]}'
                    ),
                    tool_calls=[],
                ),
            ]
        ),
    )

    graph = _build_evaluate_answer_graph(runtime)
    result = graph.invoke(
        {
            "request": EvaluateAnswerRequest(
                mode="basics",
                topic="redis",
                question="Redis 为什么快？",
                expected_points=["内存访问", "事件循环"],
                answer="因为主要在内存里，线程模型也比较简单。",
                turn_index=1,
                max_turns=2,
            )
        }
    )

    envelope = result["result"]
    assert isinstance(envelope, EvaluateAnswerEnvelope)
    assert envelope.side_effects.depth_signal == "extend"
    assert len(envelope.side_effects.observations) == 1
    assert envelope.side_effects.observations[0].content == "用户会讲主线，但案例不够具体。"
    assert len(envelope.side_effects.knowledge_updates) == 1
    assert envelope.side_effects.knowledge_updates[0].label == "redis"


def test_generate_review_returns_recommended_next_from_action_tool() -> None:
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
                            "id": "call_session",
                            "function": {
                                "name": "recall_training_context",
                                "arguments": "{}",
                            },
                        }
                    ],
                ),
                ChatCompletionResult(
                    content="",
                    tool_calls=[
                        {
                            "id": "call_next",
                            "function": {
                                "name": "suggest_next_session",
                                "arguments": (
                                    '{"mode":"basics","topic":"redis","reason":"先补缓存击穿止血顺序"}'
                                ),
                            },
                        }
                    ],
                ),
                ChatCompletionResult(
                    content=(
                        '{"overall":"总结","top_fix":"先补关键缺口","top_fix_reason":"这是最大短板",'
                        '"highlights":["主线清楚"],"gaps":["案例不够具体"],'
                        '"suggested_topics":["redis"],"next_training_focus":["补细节"],'
                        '"recommended_next":{"mode":"basics","topic":"redis","reason":"补短板"},'
                        '"score_breakdown":{"准确性":72}}'
                    ),
                    tool_calls=[],
                ),
            ]
        ),
    )

    graph = _build_generate_review_graph(runtime)
    result = graph.invoke(
        {
            "request": GenerateReviewRequest(
                session=TrainingSession(id="sess_1", mode="basics", topic="redis"),
                turns=[
                    TrainingTurn(
                        question="Redis 为什么快？",
                        expected_points=["内存访问", "事件循环"],
                        answer="因为它主要在内存里。",
                    )
                ],
            )
        }
    )

    envelope = result["result"]
    assert isinstance(envelope, GenerateReviewEnvelope)
    assert envelope.side_effects.recommended_next is not None
    assert envelope.side_effects.recommended_next.topic == "redis"
    assert envelope.side_effects.recommended_next.reason == "先补缓存击穿止血顺序"


def test_generate_review_tools_include_session_detail_callback_when_backend_enabled() -> None:
    backend_client = FakeBackendClient()
    tools = build_generate_review_agent_tools(
        GenerateReviewRequest(
            session=TrainingSession(id="sess_1", mode="basics", topic="redis"),
            turns=[],
        ),
        {},
        backend_client=backend_client,
    )

    by_name = {tool.name: tool for tool in tools}
    assert "get_session_detail" in by_name
    payload = by_name["get_session_detail"].handler({})
    assert payload["session"]["id"] == "sess_1"
    assert backend_client.session_ids == ["sess_1"]


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


def test_stream_generate_question_result_event_wraps_raw_output() -> None:
    runtime = AgentRuntime(
        Settings(
            github_token="",
            model="test-model",
            openai_base_url="http://example.com/v1",
            openai_api_key="test-key",
            llm_timeout_seconds=10,
        ),
        model_client=FakeStreamModelClient([]),
    )

    events = list(
        runtime.stream_generate_question(
            GenerateQuestionRequest(mode="basics", topic="redis", intensity="standard")
        )
    )

    result_event = events[-1]
    assert result_event["type"] == "result"
    assert result_event["data"]["result"]["question"] == "请讲讲 Redis 一致性。"
    assert '"question":"请讲讲 Redis 一致性。"' in result_event["data"]["raw_output"]


def test_stream_evaluate_answer_result_event_wraps_raw_output() -> None:
    runtime = AgentRuntime(
        Settings(
            github_token="",
            model="test-model",
            openai_base_url="http://example.com/v1",
            openai_api_key="test-key",
            llm_timeout_seconds=10,
        ),
        model_client=FakeEvaluateStreamModelClient([]),
    )

    events = list(
        runtime.stream_evaluate_answer(
            EvaluateAnswerRequest(
                mode="basics",
                topic="redis",
                question="Redis 为什么快？",
                expected_points=["内存访问", "事件循环"],
                answer="因为它在内存里。",
                turn_index=1,
                max_turns=1,
            )
        )
    )

    result_event = events[-1]
    assert result_event["type"] == "result"
    assert result_event["data"]["result"]["score"] == 86
    assert '"score":86' in result_event["data"]["raw_output"]


def test_stream_generate_review_result_event_wraps_raw_output() -> None:
    runtime = AgentRuntime(
        Settings(
            github_token="",
            model="test-model",
            openai_base_url="http://example.com/v1",
            openai_api_key="test-key",
            llm_timeout_seconds=10,
        ),
        model_client=FakeReviewStreamModelClient([]),
    )

    events = list(
        runtime.stream_generate_review(
            GenerateReviewRequest(
                session=TrainingSession(id="sess_1", mode="basics", topic="redis"),
                turns=[
                    TrainingTurn(
                        question="Redis 为什么快？",
                        expected_points=["内存访问", "事件循环"],
                        answer="因为它在内存里。",
                    )
                ],
            )
        )
    )

    result_event = events[-1]
    assert result_event["type"] == "result"
    assert result_event["data"]["result"]["top_fix"] == "先补关键缺口"
    assert '"top_fix":"先补关键缺口"' in result_event["data"]["raw_output"]


def test_runtime_raises_when_llm_is_disabled() -> None:
    runtime = AgentRuntime(
        Settings(
            github_token="",
            model="",
            openai_base_url="",
            openai_api_key="",
            llm_timeout_seconds=10,
        )
    )

    with pytest.raises(ModelClientError):
        runtime.evaluate_answer(
            EvaluateAnswerRequest(
                mode="basics",
                topic="redis",
                question="Redis 为什么快？",
                expected_points=["内存访问", "事件循环", "高效数据结构"],
                answer="因为它在内存里。",
            )
        )


def test_analyze_repo_requires_llm_when_no_model_client_is_configured() -> None:
    runtime = AgentRuntime(
        Settings(
            github_token="",
            model="",
            openai_base_url="",
            openai_api_key="",
            llm_timeout_seconds=10,
        )
    )

    with pytest.raises(ModelClientError):
        runtime.analyze_repo(AnalyzeRepoRequest(repo_url="https://github.com/example/repo"))


def test_analyze_job_target_requires_llm_when_no_model_client_is_configured() -> None:
    runtime = AgentRuntime(
        Settings(
            github_token="",
            model="",
            openai_base_url="",
            openai_api_key="",
            llm_timeout_seconds=10,
        )
    )

    with pytest.raises(ModelClientError):
        runtime.analyze_job_target(
            AnalyzeJobTargetRequest(
                title="后端工程师",
                source_text="要求 Go、Redis、Kafka 经验。",
            )
        )


@pytest.mark.parametrize(
    ("raw_kind", "expected_kind"),
    [
        ("topic", "topic"),
        ("accuracy", "detail"),
        ("correctness", "detail"),
        ("completeness", "depth"),
        ("coverage", "depth"),
        ("clarity", "expression"),
        ("followup-breakdown", "followup_breakdown"),
        ("communication", "expression"),
        ("depth", "depth"),
        ("detail", "detail"),
    ],
)
def test_weakness_hit_normalizes_supported_kinds(raw_kind: str, expected_kind: str) -> None:
    hit = WeaknessHit(kind=raw_kind, label="coverage", severity=0.6)

    assert hit.kind == expected_kind
