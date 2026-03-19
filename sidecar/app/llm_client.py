from __future__ import annotations

import json
from collections.abc import Iterator
from dataclasses import dataclass
from typing import Any
from urllib import error as urllib_error
from urllib import request as urllib_request

from app.config import Settings


class ModelClientError(RuntimeError):
    """Raised when the upstream LLM provider cannot satisfy a request."""


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

        try:
            with urllib_request.urlopen(
                request,
                timeout=self._settings.llm_timeout_seconds,
            ) as response:
                raw = response.read().decode("utf-8")
        except urllib_error.HTTPError as exc:
            detail = exc.read().decode("utf-8", errors="replace")
            raise ModelClientError(f"LLM provider returned HTTP {exc.code}: {detail}") from exc
        except urllib_error.URLError as exc:
            raise ModelClientError(f"LLM provider is unreachable: {exc.reason}") from exc

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

        try:
            with urllib_request.urlopen(
                request,
                timeout=self._settings.llm_timeout_seconds,
            ) as response:
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
        except urllib_error.HTTPError as exc:
            detail = exc.read().decode("utf-8", errors="replace")
            raise ModelClientError(f"LLM provider returned HTTP {exc.code}: {detail}") from exc
        except urllib_error.URLError as exc:
            raise ModelClientError(f"LLM provider is unreachable: {exc.reason}") from exc

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
