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
from app.schemas import GenerateQuestionEnvelope, GenerateQuestionRequest, GenerateQuestionResponse


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
