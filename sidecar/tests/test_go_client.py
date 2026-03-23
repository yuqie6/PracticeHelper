from __future__ import annotations

import io
import json
import sys
from pathlib import Path
from urllib import request as urllib_request

import pytest

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))

from app.adapters.go_client import GoBackendClient, GoBackendClientError
from app.config import Settings


class FakeHTTPResponse:
    def __init__(self, payload: dict) -> None:
        self._body = io.BytesIO(json.dumps(payload).encode("utf-8"))

    def read(self) -> bytes:
        return self._body.read()

    def __enter__(self):
        return self

    def __exit__(self, exc_type, exc, tb):
        return False


def test_go_backend_client_requires_callback_configuration() -> None:
    client = GoBackendClient(
        Settings(
            github_token="",
            model="test-model",
            openai_base_url="http://example.com/v1",
            openai_api_key="test-key",
            llm_timeout_seconds=10,
        )
    )

    with pytest.raises(GoBackendClientError, match="not configured"):
        client.search_repo_chunks("proj_1", "redis")


def test_go_backend_client_search_repo_chunks_parses_envelope(monkeypatch) -> None:
    captured: dict[str, str] = {}

    def fake_urlopen(request: urllib_request.Request, timeout: float):
        captured["url"] = request.full_url
        captured["token"] = dict(request.header_items()).get("X-practicehelper-internal-token")
        captured["timeout"] = str(timeout)
        return FakeHTTPResponse(
            {
                "data": [
                    {
                        "id": "chunk_1",
                        "project_id": "proj_1",
                        "file_path": "internal/cache.go",
                        "file_type": ".go",
                        "content": "redis cache consistency",
                        "importance": 1.0,
                        "fts_key": "internal/cache.go#0",
                    }
                ]
            }
        )

    monkeypatch.setattr(urllib_request, "urlopen", fake_urlopen)

    client = GoBackendClient(
        Settings(
            github_token="",
            model="test-model",
            openai_base_url="http://example.com/v1",
            openai_api_key="test-key",
            llm_timeout_seconds=10,
            server_base_url="http://127.0.0.1:8090",
            internal_token="secret-token",
            backend_timeout_seconds=7,
        )
    )

    chunks = client.search_repo_chunks("proj_1", "redis", limit=3)

    assert len(chunks) == 1
    assert chunks[0].file_path == "internal/cache.go"
    assert "project_id=proj_1" in captured["url"]
    assert "query=redis" in captured["url"]
    assert "limit=3" in captured["url"]
    assert captured["token"] == "secret-token"
    assert captured["timeout"] == "7"


def test_go_backend_client_get_session_detail_parses_payload(monkeypatch) -> None:
    def fake_urlopen(request: urllib_request.Request, timeout: float):
        return FakeHTTPResponse(
            {
                "data": {
                    "session": {
                        "id": "sess_1",
                        "mode": "basics",
                        "topic": "redis",
                        "status": "completed",
                        "total_score": 82,
                        "turns": [
                            {
                                "id": "turn_1",
                                "turn_index": 1,
                                "question": "Redis 为什么快？",
                                "expected_points": ["内存访问", "事件循环"],
                                "answer": "因为主要在内存里。",
                            }
                        ],
                    },
                    "review": {
                        "overall": "整体过线",
                        "top_fix": "补细节",
                        "top_fix_reason": "案例不够具体",
                        "highlights": ["主线清楚"],
                        "gaps": ["案例偏少"],
                        "suggested_topics": ["redis"],
                        "next_training_focus": ["补细节"],
                        "score_breakdown": {"准确性": 82},
                    },
                }
            }
        )

    monkeypatch.setattr(urllib_request, "urlopen", fake_urlopen)

    client = GoBackendClient(
        Settings(
            github_token="",
            model="test-model",
            openai_base_url="http://example.com/v1",
            openai_api_key="test-key",
            llm_timeout_seconds=10,
            server_base_url="http://127.0.0.1:8090",
            internal_token="secret-token",
        )
    )

    detail = client.get_session_detail("sess_1")

    assert detail.session.id == "sess_1"
    assert detail.session.turns[0].question == "Redis 为什么快？"
    assert detail.review is not None
    assert detail.review.top_fix == "补细节"
