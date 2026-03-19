from __future__ import annotations

import json
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

    @property
    def _chat_completions_url(self) -> str:
        base = self._settings.openai_base_url.rstrip("/")
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
