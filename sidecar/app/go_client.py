from __future__ import annotations

import json
from urllib import error as urllib_error
from urllib import parse as urllib_parse
from urllib import request as urllib_request

from app.config import Settings
from app.schemas import AgentCommandEnvelope, AgentCommandResult, AgentSessionDetail, RepoChunk


class GoBackendClientError(RuntimeError):
    """Raised when the Go backend callback endpoints are unavailable or misconfigured."""


class GoBackendClient:
    def __init__(self, settings: Settings) -> None:
        self._settings = settings

    @property
    def enabled(self) -> bool:
        return self._settings.backend_enabled

    def search_repo_chunks(self, project_id: str, query: str, limit: int = 6) -> list[RepoChunk]:
        payload = self._get_json(
            "/internal/search-chunks",
            {"project_id": project_id, "query": query, "limit": str(limit)},
        )
        return [RepoChunk.model_validate(item) for item in payload.get("data", [])]

    def get_session_detail(self, session_id: str) -> AgentSessionDetail:
        payload = self._get_json(f"/internal/session-detail/{session_id}")
        return AgentSessionDetail.model_validate(payload["data"])

    def run_agent_command(self, command: AgentCommandEnvelope) -> AgentCommandResult:
        payload = self._post_json("/internal/agent-commands", command.model_dump(mode="json"))
        return AgentCommandResult.model_validate(payload["data"])

    def _get_json(self, path: str, query: dict[str, str] | None = None) -> dict:
        return self._request_json("GET", path, query=query)

    def _post_json(self, path: str, payload: dict[str, object]) -> dict:
        return self._request_json("POST", path, payload=payload)

    def _request_json(
        self,
        method: str,
        path: str,
        *,
        query: dict[str, str] | None = None,
        payload: dict[str, object] | None = None,
    ) -> dict:
        if not self.enabled:
            raise GoBackendClientError(
                "Go backend callback is not configured. "
                "Set PRACTICEHELPER_SIDECAR_SERVER_BASE_URL and PRACTICEHELPER_INTERNAL_TOKEN."
            )

        url = self._settings.server_base_url.rstrip("/") + path
        if query:
            url = f"{url}?{urllib_parse.urlencode(query)}"

        raw_payload: bytes | None = None
        headers = {self._internal_token_header: self._settings.internal_token}
        if payload is not None:
            raw_payload = json.dumps(payload, ensure_ascii=False).encode("utf-8")
            headers["Content-Type"] = "application/json"

        request = urllib_request.Request(
            url,
            data=raw_payload,
            headers=headers,
            method=method,
        )
        try:
            with urllib_request.urlopen(
                request,
                timeout=self._settings.backend_timeout_seconds,
            ) as response:
                raw = response.read().decode("utf-8")
        except urllib_error.HTTPError as exc:
            detail = exc.read().decode("utf-8", errors="replace")
            raise GoBackendClientError(f"Go backend returned HTTP {exc.code}: {detail}") from exc
        except urllib_error.URLError as exc:
            raise GoBackendClientError(f"Go backend is unreachable: {exc.reason}") from exc

        try:
            payload = json.loads(raw)
        except json.JSONDecodeError as exc:
            raise GoBackendClientError(f"Go backend returned invalid JSON: {raw}") from exc
        if not isinstance(payload, dict) or "data" not in payload:
            raise GoBackendClientError(f"Go backend returned unexpected payload: {payload}")
        return payload

    @property
    def _internal_token_header(self) -> str:
        return "X-PracticeHelper-Internal-Token"
