from __future__ import annotations

import http.client
import json
import random
import time
from collections.abc import Iterator
from dataclasses import dataclass
from typing import Any
from urllib import error as urllib_error
from urllib import request as urllib_request

from app.config import Settings


class ModelClientError(RuntimeError):
    """Raised when the upstream LLM provider cannot satisfy a request."""

    def __init__(self, message: str, *, code: str = "") -> None:
        super().__init__(message)
        self.code = code


@dataclass(frozen=True)
class ChatCompletionResult:
    content: str
    tool_calls: list[dict[str, Any]]


@dataclass(frozen=True)
class ChatCompletionStreamChunk:
    content: str = ""
    reasoning: str = ""
    done: bool = False


class OpenAICompatibleModelClient:
    def __init__(self, settings: Settings) -> None:
        self._settings = settings

    def create_completion(
        self,
        *,
        messages: list[dict[str, Any]],
        tools: list[dict[str, Any]] | None = None,
        temperature: float = 0.2,
    ) -> ChatCompletionResult:
        payload: dict[str, Any] = {
            "model": self._settings.model,
            "messages": messages,
            "temperature": temperature,
        }
        if tools:
            payload["tools"] = tools
            payload["tool_choice"] = "auto"

        request = urllib_request.Request(
            self._chat_completions_url,
            data=json.dumps(payload).encode("utf-8"),
            headers={
                "Authorization": f"Bearer {self._settings.openai_api_key}",
                "Content-Type": "application/json",
            },
            method="POST",
        )

        with self._call_with_retry(request, timeout=self._settings.llm_timeout_seconds) as response:
            raw = response.read().decode("utf-8")

        try:
            data = json.loads(raw)
        except json.JSONDecodeError as exc:
            raise ModelClientError(f"LLM provider returned invalid JSON: {raw}") from exc

        choices = data.get("choices")
        if not isinstance(choices, list) or not choices:
            raise ModelClientError(f"LLM provider returned no choices: {data}")

        message = choices[0].get("message")
        if not isinstance(message, dict):
            raise ModelClientError(f"LLM provider returned invalid message payload: {data}")

        content = self._normalize_content(message.get("content"))
        tool_calls = message.get("tool_calls") or []
        if not isinstance(tool_calls, list):
            raise ModelClientError(f"LLM provider returned invalid tool_calls payload: {data}")

        return ChatCompletionResult(content=content, tool_calls=tool_calls)

    def create_completion_stream(
        self,
        *,
        messages: list[dict[str, Any]],
        temperature: float = 0.2,
    ) -> Iterator[ChatCompletionStreamChunk]:
        payload: dict[str, Any] = {
            "model": self._settings.model,
            "messages": messages,
            "temperature": temperature,
            "stream": True,
        }

        request = urllib_request.Request(
            self._chat_completions_url,
            data=json.dumps(payload).encode("utf-8"),
            headers={
                "Authorization": f"Bearer {self._settings.openai_api_key}",
                "Content-Type": "application/json",
            },
            method="POST",
        )

        with self._call_with_retry(request, timeout=self._settings.llm_timeout_seconds) as response:
            content_type = response.headers.get_content_type()
            if content_type != "text/event-stream":
                raw = response.read().decode("utf-8")
                result = self._parse_completion_response(raw)
                if result.content:
                    yield ChatCompletionStreamChunk(content=result.content)
                yield ChatCompletionStreamChunk(done=True)
                return

            event_lines: list[str] = []
            for raw_line in response:
                line = raw_line.decode("utf-8", errors="replace").rstrip("\r\n")
                if not line:
                    yield from self._parse_stream_event(event_lines)
                    event_lines = []
                    continue

                if line.startswith("data:"):
                    event_lines.append(line[5:].strip())

            if event_lines:
                yield from self._parse_stream_event(event_lines)

    def create_embeddings(self, texts: list[str]) -> tuple[list[list[float]], str]:
        if not self._settings.embedding_enabled:
            raise ModelClientError(
                "Embedding model is required. Configure PRACTICEHELPER_SIDECAR_EMBEDDING_MODEL, "
                "PRACTICEHELPER_SIDECAR_EMBEDDING_BASE_URL, "
                "and PRACTICEHELPER_SIDECAR_EMBEDDING_API_KEY."
            )
        if not texts:
            return [], ""

        request = urllib_request.Request(
            self._embeddings_url,
            data=json.dumps(
                {
                    "model": self._settings.embedding_model,
                    "input": texts,
                }
            ).encode("utf-8"),
            headers={
                "Authorization": f"Bearer {self._settings.embedding_api_key}",
                "Content-Type": "application/json",
            },
            method="POST",
        )

        with self._call_with_retry(
            request,
            timeout=self._settings.embedding_timeout_seconds,
        ) as response:
            raw = response.read().decode("utf-8")

        try:
            payload = json.loads(raw)
        except json.JSONDecodeError as exc:
            raise ModelClientError(f"Embedding provider returned invalid JSON: {raw}") from exc

        items = payload.get("data")
        if not isinstance(items, list):
            raise ModelClientError(f"Embedding provider returned invalid payload: {payload}")

        vectors: list[tuple[int, list[float]]] = []
        for item in items:
            if not isinstance(item, dict):
                continue
            index = item.get("index", len(vectors))
            embedding = item.get("embedding")
            if not isinstance(index, int) or not isinstance(embedding, list):
                continue
            vectors.append((index, [float(value) for value in embedding]))
        vectors.sort(key=lambda item: item[0])
        model_name = str(payload.get("model") or self._settings.embedding_model)
        return [vector for _, vector in vectors], model_name

    def rerank_documents(
        self,
        *,
        query: str,
        documents: list[str],
        top_k: int,
    ) -> list[dict[str, float | int]]:
        if not self._settings.rerank_enabled:
            raise ModelClientError(
                "Rerank model is required. Configure PRACTICEHELPER_SIDECAR_RERANK_MODEL, "
                "PRACTICEHELPER_SIDECAR_RERANK_BASE_URL, and PRACTICEHELPER_SIDECAR_RERANK_API_KEY."
            )
        if not documents:
            return []

        request = urllib_request.Request(
            self._rerank_url,
            data=json.dumps(
                {
                    "model": self._settings.rerank_model,
                    "query": query,
                    "documents": documents,
                    "top_n": top_k,
                    "return_documents": False,
                }
            ).encode("utf-8"),
            headers={
                "Authorization": f"Bearer {self._settings.rerank_api_key}",
                "Content-Type": "application/json",
            },
            method="POST",
        )

        with self._call_with_retry(
            request,
            timeout=self._settings.rerank_timeout_seconds,
        ) as response:
            raw = response.read().decode("utf-8")

        try:
            payload = json.loads(raw)
        except json.JSONDecodeError as exc:
            raise ModelClientError(f"Rerank provider returned invalid JSON: {raw}") from exc

        raw_items = payload.get("results")
        if raw_items is None:
            raw_items = payload.get("data")
        if not isinstance(raw_items, list):
            raise ModelClientError(f"Rerank provider returned invalid payload: {payload}")

        items: list[dict[str, float | int]] = []
        for item in raw_items:
            if not isinstance(item, dict):
                continue
            index = item.get("index")
            score = item.get("relevance_score", item.get("score"))
            if not isinstance(index, int):
                continue
            try:
                normalized_score = float(score)
            except (TypeError, ValueError):
                continue
            items.append({"index": index, "score": normalized_score})
        items.sort(key=lambda item: float(item["score"]), reverse=True)
        return items

    @property
    def _chat_completions_url(self) -> str:
        base = self._settings.openai_base_url.rstrip("/")
        # 兼容三种常见 base URL 形态：完整 /chat/completions、/v1，以及裸 base URL。
        # 这样 provider 配置可以尽量保持最小要求，而不用强制用户记住某一种固定写法。
        if base.endswith("/chat/completions"):
            return base
        if base.endswith("/v1"):
            return f"{base}/chat/completions"
        return f"{base}/v1/chat/completions"

    @property
    def _embeddings_url(self) -> str:
        base = self._settings.embedding_base_url.rstrip("/")
        if base.endswith("/embeddings"):
            return base
        if base.endswith("/v1"):
            return f"{base}/embeddings"
        return f"{base}/v1/embeddings"

    @property
    def _rerank_url(self) -> str:
        base = self._settings.rerank_base_url.rstrip("/")
        if base.endswith("/rerank"):
            return base
        return f"{base}/rerank"

    @staticmethod
    def _normalize_content(content: Any) -> str:
        if content is None:
            return ""
        if isinstance(content, str):
            return content
        if isinstance(content, list):
            # 一些 OpenAI-compatible provider 返回传统字符串 content，
            # 另一些会返回分段的 content list；这里只抽取 text/output_text，保持最小兼容面。
            parts: list[str] = []
            for item in content:
                if isinstance(item, str):
                    parts.append(item)
                    continue
                if not isinstance(item, dict):
                    continue
                if item.get("type") in {"text", "output_text"}:
                    text = item.get("text") or item.get("content") or ""
                    if text:
                        parts.append(str(text))
            return "\n".join(part for part in parts if part).strip()
        return str(content)

    def _parse_completion_response(self, raw: str) -> ChatCompletionResult:
        try:
            data = json.loads(raw)
        except json.JSONDecodeError as exc:
            raise ModelClientError(f"LLM provider returned invalid JSON: {raw}") from exc

        choices = data.get("choices")
        if not isinstance(choices, list) or not choices:
            raise ModelClientError(f"LLM provider returned no choices: {data}")

        message = choices[0].get("message")
        if not isinstance(message, dict):
            raise ModelClientError(f"LLM provider returned invalid message payload: {data}")

        content = self._normalize_content(message.get("content"))
        tool_calls = message.get("tool_calls") or []
        if not isinstance(tool_calls, list):
            raise ModelClientError(f"LLM provider returned invalid tool_calls payload: {data}")

        return ChatCompletionResult(content=content, tool_calls=tool_calls)

    def _parse_stream_event(self, event_lines: list[str]) -> Iterator[ChatCompletionStreamChunk]:
        if not event_lines:
            return

        payload = "\n".join(event_lines).strip()
        if not payload:
            return
        if payload == "[DONE]":
            yield ChatCompletionStreamChunk(done=True)
            return

        try:
            data = json.loads(payload)
        except json.JSONDecodeError:
            return

        choices = data.get("choices")
        if not isinstance(choices, list) or not choices:
            return

        delta = choices[0].get("delta")
        if not isinstance(delta, dict):
            return

        content = self._normalize_content(delta.get("content"))
        reasoning = self._normalize_reasoning(delta)
        finish_reason = choices[0].get("finish_reason")

        if content or reasoning:
            yield ChatCompletionStreamChunk(content=content, reasoning=reasoning)
        if finish_reason is not None:
            yield ChatCompletionStreamChunk(done=True)

    @classmethod
    def _normalize_reasoning(cls, delta: dict[str, Any]) -> str:
        parts: list[str] = []
        for key in ("reasoning", "reasoning_content", "thinking", "summary"):
            value = delta.get(key)
            normalized = cls._normalize_content(value)
            if normalized:
                parts.append(normalized)
        return "\n".join(part for part in parts if part).strip()

    def _call_with_retry(
        self,
        request: urllib_request.Request,
        *,
        timeout: float,
        max_attempts: int = 3,
        base_delay: float = 0.5,
    ) -> http.client.HTTPResponse:
        for attempt in range(max_attempts):
            try:
                return urllib_request.urlopen(
                    request,
                    timeout=timeout,
                )
            except urllib_error.HTTPError as exc:
                if not self._is_retryable_http_status(exc.code) or attempt == max_attempts - 1:
                    detail = exc.read().decode("utf-8", errors="replace")
                    raise ModelClientError(
                        f"LLM provider returned HTTP {exc.code}: {detail}"
                    ) from exc
                exc.close()
            except urllib_error.URLError as exc:
                if attempt == max_attempts - 1:
                    raise ModelClientError(f"LLM provider is unreachable: {exc.reason}") from exc

            time.sleep(base_delay * (2**attempt) + random.uniform(0, 0.2))

        raise ModelClientError("LLM provider retry budget exhausted")

    @staticmethod
    def _is_retryable_http_status(status_code: int) -> bool:
        return status_code in {502, 503, 504}
