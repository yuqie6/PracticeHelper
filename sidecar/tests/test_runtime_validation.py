# ruff: noqa: F403,F405
from runtime_test_support import *


def test_evaluate_answer_graph_wraps_runtime_result_envelope() -> None:
    runtime = SimpleEvaluateGraphRuntime()
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

    assert runtime.calls == 1
    assert isinstance(result["result"], EvaluateAnswerEnvelope)
    assert result["result"].result.followup_question == "如果线上抖动，你会先看什么？"
    assert result["result"].side_effects.depth_signal == "normal"


def test_generate_review_graph_wraps_runtime_result_envelope() -> None:
    runtime = SimpleGenerateReviewGraphRuntime()
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

    assert runtime.calls == 1
    assert isinstance(result["result"], GenerateReviewEnvelope)
    assert result["result"].result.top_fix == "先补最关键缺口"


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
