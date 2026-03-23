# ruff: noqa: F401

import sys
from pathlib import Path

import pytest

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))

from app.agent_runtime import AgentRuntime, TaskExecutionResult
from app.agent_tools import (
    build_evaluate_answer_agent_tools,
    build_generate_review_agent_tools,
    prepare_generate_review_agent_tooling,
)
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
    AgentCommandEnvelope,
    AgentCommandResult,
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

    def create_completion(self, *, messages, tools=None, temperature=0.2):
        self.calls.append({"messages": messages, "tools": tools, "temperature": temperature})
        raise ModelClientError("stream-only fake")

    def create_completion_stream(self, *, messages, temperature=0.2):
        self.calls.append({"messages": messages, "temperature": temperature, "stream": True})
        yield ChatCompletionStreamChunk(
            content='{"question":"请讲讲 Redis 一致性。","expected_points":["主线","取舍"]}'
        )


class FakeEvaluateStreamModelClient(FakeModelClient):
    def __init__(self, responses: list[ChatCompletionResult]) -> None:
        super().__init__(responses)

    def create_completion(self, *, messages, tools=None, temperature=0.2):
        self.calls.append({"messages": messages, "tools": tools, "temperature": temperature})
        raise ModelClientError("stream-only fake")

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

    def create_completion(self, *, messages, tools=None, temperature=0.2):
        self.calls.append({"messages": messages, "tools": tools, "temperature": temperature})
        raise ModelClientError("stream-only fake")

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
        self.commands: list[AgentCommandEnvelope] = []
        self.command_results: dict[str, AgentCommandResult] = {}

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

    def run_agent_command(self, command: AgentCommandEnvelope) -> AgentCommandResult:
        self.commands.append(command)
        result = self.command_results.get(command.command_type)
        if result is None:
            raise AssertionError(f"missing fake command result for {command.command_type}")
        return result


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


class SimpleEvaluateGraphRuntime:
    def __init__(self) -> None:
        self.calls = 0

    def evaluate_answer_task(
        self, request: EvaluateAnswerRequest
    ) -> TaskExecutionResult[EvaluationResult]:
        self.calls += 1
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


class SimpleGenerateReviewGraphRuntime:
    def __init__(self) -> None:
        self.calls = 0

    def generate_review_task(
        self, request: GenerateReviewRequest
    ) -> TaskExecutionResult[ReviewCard]:
        self.calls += 1
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


__all__ = [name for name in globals() if not name.startswith("__")]
__all__.extend(
    [
        "_build_analyze_repo_graph",
        "_build_evaluate_answer_graph",
        "_build_generate_question_graph",
        "_build_generate_review_graph",
    ]
)
