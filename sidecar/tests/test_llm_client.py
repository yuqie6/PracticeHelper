from __future__ import annotations

import io
import json
import sys
from pathlib import Path
from urllib import error as urllib_error

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))

from app.adapters import llm_client
from app.adapters.llm_client import ModelClientError, OpenAICompatibleModelClient
from app.config import Settings


class FakeHeaders:
    def __init__(self, content_type: str) -> None:
        self._content_type = content_type

    def get_content_type(self) -> str:
        return self._content_type


class FakeHTTPResponse:
    def __init__(self, chunks: list[bytes], *, content_type: str = "application/json") -> None:
        self._chunks = chunks
        self.headers = FakeHeaders(content_type)

    def read(self) -> bytes:
        return b"".join(self._chunks)

    def __iter__(self):
        return iter(self._chunks)

    def __enter__(self) -> FakeHTTPResponse:
        return self

    def __exit__(self, exc_type, exc, tb) -> bool:
        return False


def build_settings(
    base_url: str = "https://example.com/v1",
    *,
    openai_api_key: str = "test-key",
    embedding_api_key: str = "embed-key",
    rerank_api_key: str = "rerank-key",
) -> Settings:
    return Settings(
        github_token="",
        model="test-model",
        openai_base_url=base_url,
        openai_api_key=openai_api_key,
        llm_timeout_seconds=5,
        embedding_model="embed-model",
        embedding_base_url="https://embed.example.com/v1",
        embedding_api_key=embedding_api_key,
        rerank_model="rerank-model",
        rerank_base_url="https://rerank.example.com",
        rerank_api_key=rerank_api_key,
    )


def test_chat_completion_url_accepts_multiple_base_url_shapes() -> None:
    bare = OpenAICompatibleModelClient(build_settings("https://example.com"))
    versioned = OpenAICompatibleModelClient(build_settings("https://example.com/v1"))
    full = OpenAICompatibleModelClient(build_settings("https://example.com/chat/completions"))

    assert bare._chat_completions_url == "https://example.com/v1/chat/completions"
    assert versioned._chat_completions_url == "https://example.com/v1/chat/completions"
    assert full._chat_completions_url == "https://example.com/chat/completions"


def test_settings_treat_local_endpoints_without_api_keys_as_enabled() -> None:
    settings = build_settings(openai_api_key="", embedding_api_key="", rerank_api_key="")

    assert settings.llm_enabled is True
    assert settings.embedding_enabled is True
    assert settings.rerank_enabled is True


def test_create_completion_normalizes_segmented_content_and_tool_calls(monkeypatch) -> None:
    payload = {
        "choices": [
            {
                "message": {
                    "content": [
                        {"type": "text", "text": "第一段"},
                        {"type": "output_text", "content": "第二段"},
                    ],
                    "tool_calls": [{"id": "call_1", "type": "function"}],
                }
            }
        ]
    }

    monkeypatch.setattr(
        llm_client.urllib_request,
        "urlopen",
        lambda request, timeout: FakeHTTPResponse([json.dumps(payload).encode("utf-8")]),
    )

    client = OpenAICompatibleModelClient(build_settings())
    result = client.create_completion(messages=[{"role": "user", "content": "hello"}])

    assert result.content == "第一段\n第二段"
    assert result.tool_calls == [{"id": "call_1", "type": "function"}]


def test_create_completion_raises_model_client_error_on_http_error(monkeypatch) -> None:
    def raise_http_error(request, timeout):
        raise urllib_error.HTTPError(
            request.full_url,
            429,
            "Too Many Requests",
            hdrs=None,
            fp=io.BytesIO(b'{"error":"rate_limited"}'),
        )

    monkeypatch.setattr(llm_client.urllib_request, "urlopen", raise_http_error)

    client = OpenAICompatibleModelClient(build_settings())

    try:
        client.create_completion(messages=[{"role": "user", "content": "hello"}])
    except ModelClientError as exc:
        assert "HTTP 429" in str(exc)
        assert "rate_limited" in str(exc)
    else:
        raise AssertionError("expected ModelClientError")


def test_create_completion_omits_authorization_header_when_api_key_is_empty(monkeypatch) -> None:
    captured: dict[str, str | None] = {}

    def fake_urlopen(request, timeout):
        captured["authorization"] = request.headers.get("Authorization")
        return FakeHTTPResponse(
            [
                json.dumps(
                    {
                        "choices": [
                            {
                                "message": {
                                    "content": "ok",
                                    "tool_calls": [],
                                }
                            }
                        ]
                    }
                ).encode("utf-8")
            ]
        )

    monkeypatch.setattr(llm_client.urllib_request, "urlopen", fake_urlopen)

    client = OpenAICompatibleModelClient(build_settings(openai_api_key=""))
    result = client.create_completion(messages=[{"role": "user", "content": "hello"}])

    assert result.content == "ok"
    assert captured["authorization"] is None


def test_create_completion_stream_falls_back_to_non_sse_json(monkeypatch) -> None:
    payload = {
        "choices": [
            {
                "message": {
                    "content": "直接返回完整 JSON",
                    "tool_calls": [],
                }
            }
        ]
    }

    monkeypatch.setattr(
        llm_client.urllib_request,
        "urlopen",
        lambda request, timeout: FakeHTTPResponse(
            [json.dumps(payload).encode("utf-8")],
            content_type="application/json",
        ),
    )

    client = OpenAICompatibleModelClient(build_settings())
    chunks = list(client.create_completion_stream(messages=[{"role": "user", "content": "hello"}]))

    assert chunks[0].content == "直接返回完整 JSON"
    assert chunks[-1].done is True


def test_create_completion_stream_parses_reasoning_and_done_from_sse(monkeypatch) -> None:
    reasoning_event = 'data: {"choices":[{"delta":{"reasoning":"先整理思路"}}]}\n'.encode()
    monkeypatch.setattr(
        llm_client.urllib_request,
        "urlopen",
        lambda request, timeout: FakeHTTPResponse(
            [
                reasoning_event,
                b"\n",
                b'data: {"choices":[{"delta":{"content":"done"},"finish_reason":"stop"}]}\n',
                b"\n",
            ],
            content_type="text/event-stream",
        ),
    )

    client = OpenAICompatibleModelClient(build_settings())
    chunks = list(client.create_completion_stream(messages=[{"role": "user", "content": "hello"}]))

    assert chunks[0].reasoning == "先整理思路"
    assert chunks[1].content == "done"
    assert chunks[-1].done is True


def test_create_embeddings_uses_embedding_endpoint_and_parses_vectors(monkeypatch) -> None:
    captured: dict[str, str] = {}

    def fake_urlopen(request, timeout):
        captured["url"] = request.full_url
        captured["timeout"] = str(timeout)
        return FakeHTTPResponse(
            [
                json.dumps(
                    {
                        "model": "embed-model",
                        "data": [
                            {"index": 1, "embedding": [0.2, 0.3]},
                            {"index": 0, "embedding": [0.9, 0.1]},
                        ],
                    }
                ).encode("utf-8")
            ]
        )

    monkeypatch.setattr(llm_client.urllib_request, "urlopen", fake_urlopen)

    client = OpenAICompatibleModelClient(build_settings())
    vectors, model_name = client.create_embeddings(["query", "doc"])

    assert captured["url"] == "https://embed.example.com/v1/embeddings"
    assert model_name == "embed-model"
    assert vectors == [[0.9, 0.1], [0.2, 0.3]]


def test_create_embeddings_omits_authorization_header_when_api_key_is_empty(monkeypatch) -> None:
    captured: dict[str, str | None] = {}

    def fake_urlopen(request, timeout):
        captured["authorization"] = request.headers.get("Authorization")
        return FakeHTTPResponse(
            [
                json.dumps(
                    {
                        "model": "embed-model",
                        "data": [{"index": 0, "embedding": [0.9, 0.1]}],
                    }
                ).encode("utf-8")
            ]
        )

    monkeypatch.setattr(llm_client.urllib_request, "urlopen", fake_urlopen)

    client = OpenAICompatibleModelClient(build_settings(embedding_api_key=""))
    vectors, model_name = client.create_embeddings(["query"])

    assert model_name == "embed-model"
    assert vectors == [[0.9, 0.1]]
    assert captured["authorization"] is None


def test_rerank_documents_uses_rerank_endpoint_and_sorts_scores(monkeypatch) -> None:
    captured: dict[str, str] = {}

    def fake_urlopen(request, timeout):
        captured["url"] = request.full_url
        return FakeHTTPResponse(
            [
                json.dumps(
                    {
                        "results": [
                            {"index": 1, "relevance_score": 0.4},
                            {"index": 0, "relevance_score": 0.9},
                        ]
                    }
                ).encode("utf-8")
            ]
        )

    monkeypatch.setattr(llm_client.urllib_request, "urlopen", fake_urlopen)

    client = OpenAICompatibleModelClient(build_settings())
    items = client.rerank_documents(query="redis", documents=["a", "b"], top_k=2)

    assert captured["url"] == "https://rerank.example.com/rerank"
    assert items == [{"index": 0, "score": 0.9}, {"index": 1, "score": 0.4}]
