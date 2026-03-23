from __future__ import annotations

from collections.abc import Callable
from typing import Any

from app.adapters.go_client import GoBackendClient
from app.shared import RuntimeTool


def recall_training_context_tool(loader: Callable[[], dict[str, Any]]) -> RuntimeTool:
    return RuntimeTool(
        name="recall_training_context",
        description=(
            "Read the current task context, including question, answer, "
            "project, and template details."
        ),
        handler=lambda _: loader(),
    )


def payload_tool(
    name: str,
    description: str,
    loader: Callable[[], dict[str, Any]],
) -> RuntimeTool:
    return RuntimeTool(
        name=name,
        description=description,
        handler=lambda _: loader(),
    )


def search_repo_chunks_tool(project_id: str, backend_client: GoBackendClient) -> RuntimeTool:
    return RuntimeTool(
        name="search_repo_chunks",
        description=(
            "Search additional repository chunks from the Go backend when "
            "preloaded context is not enough."
        ),
        parameters={
            "type": "object",
            "properties": {
                "query": {"type": "string"},
                "limit": {"type": "integer"},
            },
            "required": ["query"],
            "additionalProperties": False,
        },
        handler=lambda arguments: {
            "chunks": [
                item.model_dump(mode="json")
                for item in backend_client.search_repo_chunks(
                    project_id=project_id,
                    query=str(arguments.get("query", "")),
                    limit=int(arguments.get("limit", 6)),
                )
            ]
        },
    )


def get_session_detail_tool(session_id: str, backend_client: GoBackendClient) -> RuntimeTool:
    return RuntimeTool(
        name="get_session_detail",
        description=(
            "Load a fuller historical session detail from the Go backend when "
            "review needs more evidence."
        ),
        parameters={
            "type": "object",
            "properties": {},
            "additionalProperties": False,
        },
        handler=lambda _: backend_client.get_session_detail(session_id).model_dump(mode="json"),
    )
