from __future__ import annotations

from typing import Any

from app.adapters.go_client import GoBackendClient
from app.schemas import (
    AgentCommandEnvelope,
    AgentCommandResult,
    EvaluateAnswerRequest,
    GenerateReviewRequest,
    NextSession,
)
from app.shared import RuntimeTool


class CommandBudgetExceededError(RuntimeError):
    """Raised when a command tool exceeds the per-run command budget."""


def make_transition_session_tool(
    request: EvaluateAnswerRequest,
    backend_client: GoBackendClient | None,
) -> RuntimeTool:
    return RuntimeTool(
        name="transition_session",
        description=(
            "Request the Go FSM to resolve whether this answer should skip the next follow-up "
            "or extend by one turn, then return the structured decision."
        ),
        parameters={
            "type": "object",
            "properties": {
                "decision": {
                    "type": "string",
                    "enum": ["skip_followup", "extend"],
                },
                "reason": {"type": "string"},
            },
            "required": ["decision"],
            "additionalProperties": False,
        },
        handler=lambda _: {
            "status": "disabled",
            "error_code": "command_runtime_bind_missing",
            "error_message": "transition_session requires runtime binding.",
        },
        runtime_bind=lambda state: _bind_transition_session_tool(
            request,
            backend_client,
            state,
        ),
    )


def make_upsert_review_path_tool(
    request: GenerateReviewRequest,
    backend_client: GoBackendClient | None,
) -> RuntimeTool:
    return RuntimeTool(
        name="upsert_review_path",
        description=(
            "Ask the Go backend to normalize the next training path and return the canonical "
            "recommended_next, suggested_topics, and next_training_focus."
        ),
        parameters={
            "type": "object",
            "properties": {
                "recommended_next": {
                    "type": "object",
                    "properties": {
                        "mode": {"type": "string"},
                        "topic": {"type": "string"},
                        "project_id": {"type": "string"},
                        "reason": {"type": "string"},
                    },
                    "additionalProperties": False,
                },
                "suggested_topics": {"type": "array", "items": {"type": "string"}},
                "next_training_focus": {"type": "array", "items": {"type": "string"}},
                "gaps": {"type": "array", "items": {"type": "string"}},
                "top_fix": {"type": "string"},
                "top_fix_reason": {"type": "string"},
            },
            "additionalProperties": False,
        },
        handler=lambda _: {
            "status": "disabled",
            "error_code": "command_runtime_bind_missing",
            "error_message": "upsert_review_path requires runtime binding.",
        },
        runtime_bind=lambda state: _bind_upsert_review_path_tool(
            request,
            backend_client,
            state,
        ),
    )


def _bind_transition_session_tool(
    request: EvaluateAnswerRequest,
    backend_client: GoBackendClient | None,
    state: Any,
) -> RuntimeTool:
    return RuntimeTool(
        name="transition_session",
        description=(
            "Request the Go FSM to resolve whether this answer should skip the next follow-up "
            "or extend by one turn, then return the structured decision."
        ),
        parameters={
            "type": "object",
            "properties": {
                "decision": {
                    "type": "string",
                    "enum": ["skip_followup", "extend"],
                },
                "reason": {"type": "string"},
            },
            "required": ["decision"],
            "additionalProperties": False,
        },
        handler=lambda arguments: _run_transition_session_command(
            request,
            backend_client,
            state,
            arguments,
        ),
    )


def _bind_upsert_review_path_tool(
    request: GenerateReviewRequest,
    backend_client: GoBackendClient | None,
    state: Any,
) -> RuntimeTool:
    return RuntimeTool(
        name="upsert_review_path",
        description=(
            "Ask the Go backend to normalize the next training path and return the canonical "
            "recommended_next, suggested_topics, and next_training_focus."
        ),
        parameters={
            "type": "object",
            "properties": {
                "recommended_next": {
                    "type": "object",
                    "properties": {
                        "mode": {"type": "string"},
                        "topic": {"type": "string"},
                        "project_id": {"type": "string"},
                        "reason": {"type": "string"},
                    },
                    "additionalProperties": False,
                },
                "suggested_topics": {"type": "array", "items": {"type": "string"}},
                "next_training_focus": {"type": "array", "items": {"type": "string"}},
                "gaps": {"type": "array", "items": {"type": "string"}},
                "top_fix": {"type": "string"},
                "top_fix_reason": {"type": "string"},
            },
            "additionalProperties": False,
        },
        handler=lambda arguments: _run_upsert_review_path_command(
            request,
            backend_client,
            state,
            arguments,
        ),
    )


def _run_transition_session_command(
    request: EvaluateAnswerRequest,
    backend_client: GoBackendClient | None,
    state: Any,
    arguments: dict[str, Any],
) -> dict[str, Any]:
    decision = str(arguments.get("decision", "")).strip()
    if decision not in {"skip_followup", "extend"}:
        raise ValueError(f"unsupported transition decision: {decision}")

    command = AgentCommandEnvelope(
        command_id=f"cmd_transition_session_turn_{request.turn_index}_{decision}",
        command_type="transition_session",
        session_id=request.session_id,
        idempotency_key=(
            f"{request.session_id}:evaluate_answer:transition_session:{request.turn_index}:{decision}"
        ),
        reason=str(arguments.get("reason", "")).strip(),
        payload={
            "decision": decision,
            "turn_index": request.turn_index,
            "current_max_turns": request.max_turns,
        },
    )
    return execute_command(backend_client, state, command)


def _run_upsert_review_path_command(
    request: GenerateReviewRequest,
    backend_client: GoBackendClient | None,
    state: Any,
    arguments: dict[str, Any],
) -> dict[str, Any]:
    draft = arguments.get("recommended_next")
    recommended_next: NextSession | None = None
    if isinstance(draft, dict):
        draft_payload = dict(draft)
        draft_payload.setdefault("mode", request.session.mode)
        recommended_next = NextSession.model_validate(draft_payload)

    command = AgentCommandEnvelope(
        command_id="cmd_upsert_review_path",
        command_type="upsert_review_path",
        session_id=request.session.id,
        idempotency_key=f"{request.session.id}:generate_review:upsert_review_path",
        reason=str(arguments.get("top_fix_reason", "")).strip(),
        payload={
            "recommended_next": (
                recommended_next.model_dump(mode="json") if recommended_next is not None else None
            ),
            "suggested_topics": coerce_string_list(arguments.get("suggested_topics")),
            "next_training_focus": coerce_string_list(arguments.get("next_training_focus")),
            "gaps": coerce_string_list(arguments.get("gaps")),
            "top_fix": str(arguments.get("top_fix", "")).strip(),
            "top_fix_reason": str(arguments.get("top_fix_reason", "")).strip(),
        },
    )
    return execute_command(backend_client, state, command)


def execute_command(
    backend_client: GoBackendClient | None,
    state: Any,
    command: AgentCommandEnvelope,
) -> dict[str, Any]:
    if backend_client is None:
        raise ValueError(f"{command.command_type} requires Go backend callback support")

    cached = state.command_cache.get(command.idempotency_key)
    if cached is not None:
        deduped = dict(cached)
        deduped["deduped"] = True
        deduped["idempotency_key"] = command.idempotency_key
        deduped["command_type"] = command.command_type
        return deduped

    used = state.command_counts.get(command.command_type, 0)
    budget = state.command_budget.get(command.command_type, 0)
    if budget > 0 and used >= budget:
        raise CommandBudgetExceededError(f"command budget exhausted: {command.command_type}")

    result = backend_client.run_agent_command(command)
    serialized = serialize_command_result(command, result, deduped=False)
    state.command_counts[command.command_type] = used + 1
    state.command_cache[command.idempotency_key] = serialized
    state.command_results.append(dict(serialized))
    return serialized


def serialize_command_result(
    command: AgentCommandEnvelope,
    result: AgentCommandResult,
    *,
    deduped: bool,
) -> dict[str, Any]:
    return {
        "command_id": result.command_id,
        "command_type": command.command_type,
        "idempotency_key": command.idempotency_key,
        "status": result.status,
        "applied": result.applied,
        "data": result.data,
        "error_code": result.error_code,
        "error_message": result.error_message,
        "deduped": deduped,
    }


def coerce_string_list(value: Any) -> list[str]:
    if not isinstance(value, list):
        return []
    items: list[str] = []
    for item in value:
        text = str(item).strip()
        if text:
            items.append(text)
    return items
