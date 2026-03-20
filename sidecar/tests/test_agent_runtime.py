import sys
from pathlib import Path

import pytest

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))

from app.agent_runtime import AgentRuntime
from app.config import Settings
from app.llm_client import ChatCompletionResult, ModelClientError
from app.runtime_prompts import (
    evaluate_prompt_bundle,
    question_prompt_bundle,
    review_prompt_bundle,
)
from app.schemas import (
    AnalyzeJobTargetRequest,
    AnalyzeRepoRequest,
    EvaluateAnswerRequest,
    GenerateReviewRequest,
    GenerateQuestionRequest,
    JobTargetAnalysisSnapshot,
    QuestionTemplate,
    ProjectProfile,
    TrainingSession,
    TrainingTurn,
    WeaknessHit,
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
