import sys
from pathlib import Path

from fastapi import Response

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))

from app import main
from app.prompt_loader import (
    get_default_prompt_set,
    list_prompt_sets,
    load_prompt_with_meta,
    render_prompt_with_meta,
)
from app.schemas import (
    EvaluateAnswerEnvelope,
    EvaluateAnswerRequest,
    EvaluationResult,
    GenerateQuestionEnvelope,
    GenerateQuestionRequest,
    GenerateQuestionResponse,
    GenerateReviewEnvelope,
    GenerateReviewRequest,
    ReviewCard,
    TrainingSession,
)


def test_prompt_registry_exposes_default_and_candidate_sets() -> None:
    prompt_sets = list_prompt_sets()

    assert [item.id for item in prompt_sets] == ["stable-v1", "candidate-v1"]
    assert get_default_prompt_set().id == "stable-v1"


def test_load_prompt_with_meta_uses_selected_prompt_set() -> None:
    stable_prompt = load_prompt_with_meta("generate_question_system.md", "stable-v1")
    candidate_prompt = load_prompt_with_meta("generate_question_system.md", "candidate-v1")

    assert stable_prompt.prompt_set_id == "stable-v1"
    assert candidate_prompt.prompt_set_id == "candidate-v1"
    assert stable_prompt.prompt_hash != candidate_prompt.prompt_hash


def test_render_prompt_with_meta_updates_hash_after_replacements() -> None:
    rendered = render_prompt_with_meta(
        "evaluate_answer_system.md",
        {
            "RUBRIC_LINES": "- 维度一：准确性",
            "DIMENSIONS_EXAMPLE": '{"准确性": 40}',
        },
        "stable-v1",
    )
    raw_prompt = load_prompt_with_meta("evaluate_answer_system.md", "stable-v1")

    assert rendered.prompt_set_id == "stable-v1"
    assert "__RUBRIC_LINES__" not in rendered.content
    assert rendered.prompt_hash != raw_prompt.prompt_hash


def test_list_prompt_sets_endpoint_returns_registry_items() -> None:
    response = main.list_prompt_sets_endpoint()

    assert [item.id for item in response] == ["stable-v1", "candidate-v1"]
    assert response[0].is_default is True


def test_generate_question_endpoint_sets_prompt_headers(monkeypatch) -> None:
    class FakeFlow:
        def invoke(self, payload):
            request = payload["request"]
            return {
                "result": GenerateQuestionEnvelope(
                    result=GenerateQuestionResponse(
                        question=f"{request.prompt_set_id}-question",
                        expected_points=["主线清楚"],
                    ),
                    raw_output='{"question":"candidate-v1-question"}',
                )
            }

    monkeypatch.setitem(main.flows, "generate_question", FakeFlow())

    response = Response()
    result = main.generate_question_endpoint(
        GenerateQuestionRequest(
            mode="basics",
            topic="go",
            intensity="standard",
            prompt_set_id="candidate-v1",
        ),
        response,
    )

    assert result.result.question == "candidate-v1-question"
    assert result.raw_output == '{"question":"candidate-v1-question"}'
    assert response.headers[main.PROMPT_SET_HEADER] == "candidate-v1"
    assert response.headers[main.PROMPT_HASH_HEADER]
    assert response.headers[main.MODEL_NAME_HEADER] == main.settings.model


def test_evaluate_answer_endpoint_sets_prompt_headers_and_returns_raw_output(monkeypatch) -> None:
    class FakeFlow:
        def invoke(self, payload):
            return {
                "result": EvaluateAnswerEnvelope(
                    result=EvaluationResult(
                        score=84,
                        score_breakdown={"准确性": 84},
                        strengths=["主线清楚"],
                        gaps=["例子不够具体"],
                    ),
                    raw_output=(
                        '{"score":84,"score_breakdown":{"准确性":84},'
                        '"strengths":["主线清楚"],"gaps":["例子不够具体"]}'
                    ),
                )
            }

    monkeypatch.setitem(main.flows, "evaluate_answer", FakeFlow())

    response = Response()
    result = main.evaluate_answer_endpoint(
        EvaluateAnswerRequest(
            mode="basics",
            topic="go",
            question="Go channel 有什么风险？",
            expected_points=["阻塞", "关闭语义"],
            answer="容易阻塞。",
            turn_index=1,
            max_turns=1,
            prompt_set_id="candidate-v1",
        ),
        response,
    )

    assert result.result.score == 84
    assert '"score":84' in result.raw_output
    assert response.headers[main.PROMPT_SET_HEADER] == "candidate-v1"
    assert response.headers[main.PROMPT_HASH_HEADER]
    assert response.headers[main.MODEL_NAME_HEADER] == main.settings.model


def test_generate_review_endpoint_sets_prompt_headers_and_returns_raw_output(monkeypatch) -> None:
    class FakeFlow:
        def invoke(self, payload):
            request = payload["request"]
            return {
                "result": GenerateReviewEnvelope(
                    result=ReviewCard(
                        overall="总结",
                        top_fix="先补关键缺口",
                        top_fix_reason="这是当前最大短板",
                        score_breakdown={"准确性": 75},
                        recommended_next={
                            "mode": request.session.mode,
                            "topic": request.session.topic,
                            "reason": "继续补短板",
                        },
                    ),
                    raw_output='{"overall":"总结","top_fix":"先补关键缺口"}',
                )
            }

    monkeypatch.setitem(main.flows, "generate_review", FakeFlow())

    response = Response()
    result = main.generate_review_endpoint(
        GenerateReviewRequest(
            session=TrainingSession(id="sess_1", mode="basics", topic="redis"),
            prompt_set_id="candidate-v1",
        ),
        response,
    )

    assert result.result.top_fix == "先补关键缺口"
    assert result.raw_output == '{"overall":"总结","top_fix":"先补关键缺口"}'
    assert response.headers[main.PROMPT_SET_HEADER] == "candidate-v1"
    assert response.headers[main.PROMPT_HASH_HEADER]
    assert response.headers[main.MODEL_NAME_HEADER] == main.settings.model
