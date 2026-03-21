from __future__ import annotations

import json
import sys
from pathlib import Path

from fastapi.testclient import TestClient

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))

from app import main
from app.llm_client import ModelClientError


def test_evaluate_answer_stream_endpoint_sets_prompt_headers_and_streams_ndjson(
    monkeypatch,
) -> None:
    def fake_stream(_request):
        yield {"type": "phase", "phase": "prepare_context"}
        yield {
            "type": "result",
            "data": {
                "result": {
                    "score": 88,
                    "score_breakdown": {"准确性": 88},
                    "headline": "结论到位",
                    "strengths": ["主线清楚"],
                    "gaps": ["例子不够具体"],
                    "suggestion": "补真实案例",
                    "followup_intent": "",
                    "followup_question": "",
                    "followup_expected_points": [],
                    "weakness_hits": [],
                },
                "raw_output": '{"score":88}',
            },
        }

    monkeypatch.setattr(main.runtime, "stream_evaluate_answer", fake_stream)

    client = TestClient(main.app)
    response = client.post(
        "/internal/evaluate_answer/stream",
        json={
            "mode": "basics",
            "topic": "go",
            "question": "Go interface 有什么坑？",
            "expected_points": ["nil 语义"],
            "answer": "注意 nil。",
            "turn_index": 1,
            "max_turns": 2,
            "prompt_set_id": "candidate-v1",
        },
    )

    assert response.status_code == 200
    assert response.headers[main.PROMPT_SET_HEADER] == "candidate-v1"
    assert response.headers[main.PROMPT_HASH_HEADER]
    assert response.headers[main.MODEL_NAME_HEADER] == main.settings.model

    lines = [json.loads(line) for line in response.text.splitlines() if line.strip()]
    assert lines[0] == {"type": "phase", "phase": "prepare_context"}
    assert lines[-1]["data"]["result"]["score"] == 88
    assert lines[-1]["data"]["raw_output"] == '{"score":88}'


def test_generate_review_stream_endpoint_sets_prompt_headers_and_streams_ndjson(
    monkeypatch,
) -> None:
    def fake_stream(_request):
        yield {"type": "phase", "phase": "call_model"}
        yield {
            "type": "result",
            "data": {
                "result": {
                    "overall": "总结",
                    "top_fix": "先补例子",
                    "top_fix_reason": "说服力更强",
                    "highlights": ["主线清楚"],
                    "gaps": ["案例偏少"],
                    "suggested_topics": ["redis"],
                    "next_training_focus": ["补细节"],
                    "recommended_next": {
                        "mode": "basics",
                        "topic": "redis",
                        "reason": "继续补短板",
                    },
                    "score_breakdown": {"准确性": 80},
                },
                "raw_output": '{"overall":"总结"}',
            },
        }

    monkeypatch.setattr(main.runtime, "stream_generate_review", fake_stream)

    client = TestClient(main.app)
    response = client.post(
        "/internal/generate_review/stream",
        json={
            "session": {"id": "sess_1", "mode": "basics", "topic": "redis"},
            "prompt_set_id": "candidate-v1",
        },
    )

    assert response.status_code == 200
    assert response.headers[main.PROMPT_SET_HEADER] == "candidate-v1"
    assert response.headers[main.PROMPT_HASH_HEADER]
    assert response.headers[main.MODEL_NAME_HEADER] == main.settings.model

    lines = [json.loads(line) for line in response.text.splitlines() if line.strip()]
    assert lines[0] == {"type": "phase", "phase": "call_model"}
    assert lines[-1]["data"]["result"]["top_fix"] == "先补例子"


def test_model_client_errors_map_to_expected_status_codes(monkeypatch) -> None:
    class RequiredFlow:
        def invoke(self, payload):
            raise ModelClientError("LLM is required for sidecar core flows.")

    class GatewayFlow:
        def invoke(self, payload):
            raise ModelClientError("provider temporarily unavailable")

    client = TestClient(main.app)

    monkeypatch.setitem(main.flows, "evaluate_answer", RequiredFlow())
    required_response = client.post(
        "/internal/evaluate_answer",
        json={
            "mode": "basics",
            "topic": "go",
            "question": "Go interface 有什么坑？",
            "expected_points": ["nil 语义"],
            "answer": "注意 nil。",
            "turn_index": 1,
            "max_turns": 2,
        },
    )
    assert required_response.status_code == 503
    assert "LLM is required" in required_response.json()["error"]["message"]

    monkeypatch.setitem(main.flows, "generate_review", GatewayFlow())
    gateway_response = client.post(
        "/internal/generate_review",
        json={"session": {"id": "sess_1", "mode": "basics", "topic": "redis"}},
    )
    assert gateway_response.status_code == 502
    assert "provider temporarily unavailable" in gateway_response.json()["error"]["message"]


def test_embed_memory_endpoint_returns_vectors(monkeypatch) -> None:
    monkeypatch.setattr(
        main.model_client,
        "create_embeddings",
        lambda texts: ([[0.1, 0.2, 0.3]], "embed-test"),
    )

    client = TestClient(main.app)
    response = client.post(
        "/internal/embed_memory",
        json={"items": [{"id": "memidx_1", "text": "redis cache consistency"}]},
    )

    assert response.status_code == 200
    assert response.json()["items"][0]["id"] == "memidx_1"
    assert response.json()["items"][0]["model_name"] == "embed-test"


def test_rerank_memory_endpoint_returns_ranked_items(monkeypatch) -> None:
    monkeypatch.setattr(
        main.model_client,
        "rerank_documents",
        lambda **_: [{"index": 1, "score": 0.8}, {"index": 0, "score": 0.4}],
    )

    client = TestClient(main.app)
    response = client.post(
        "/internal/rerank_memory",
        json={
            "query": "redis tradeoff",
            "candidates": [
                {"id": "memidx_1", "text": "redis basic"},
                {"id": "memidx_2", "text": "redis tradeoff"},
            ],
            "top_k": 2,
        },
    )

    assert response.status_code == 200
    assert response.json()["items"][0] == {"id": "memidx_2", "score": 0.8, "rank": 1}
