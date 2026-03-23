from __future__ import annotations

from typing import Any

from app.agent_tools import CommandBudgetExceededError
from app.llm_client import ModelClientError
from app.schemas import RuntimeTrace, RuntimeTraceEntry


def append_trace(
    trace: RuntimeTrace,
    *,
    flow: str,
    phase: str,
    status: str,
    code: str = "",
    message: str = "",
    attempt: int = 0,
    tool_name: str = "",
    details: dict[str, Any] | None = None,
) -> RuntimeTraceEntry:
    entry = RuntimeTraceEntry(
        flow=flow,
        phase=phase,
        status=status,
        code=code,
        message=message,
        attempt=attempt,
        tool_name=tool_name,
        details=details or {},
    )
    trace.entries.append(entry)
    return entry


def context_compaction_message(details: dict[str, Any]) -> str:
    section = str(details.get("section", "")).replace("_", " ").strip()
    if not section:
        return "上下文已按预算压缩。"
    return f"上下文 {section} 已按预算压缩。"


def runtime_error_code(exc: Exception, *, fallback: str) -> str:
    if isinstance(exc, ModelClientError) and exc.code:
        return exc.code

    message = str(exc).lower()
    if "required tool context" in message:
        return "tool_context_missing"
    if "validation failed" in message:
        return "semantic_validation_failed"
    if "json" in message:
        return "json_parse_failed"
    if "command budget exhausted" in message:
        return "command_budget_exhausted"
    if "go backend" in message:
        return "backend_callback_failed"
    if "timeout" in message or "timed out" in message:
        return "timeout"
    if "tool loop" in message:
        return "tool_loop_exhausted"
    return fallback


def is_command_tool(tool_name: str) -> bool:
    return tool_name in {"transition_session", "upsert_review_path", "enqueue_long_job"}


def tool_failure_code(tool_name: str, exc: Exception) -> str:
    if isinstance(exc, CommandBudgetExceededError):
        return "command_budget_exhausted"
    if tool_name in {"search_repo_chunks", "get_session_detail"}:
        return "backend_callback_failed"
    if is_command_tool(tool_name) and "go backend" in str(exc).lower():
        return "backend_callback_failed"
    return "tool_call_failed"


def command_trace_code(tool_result: dict[str, Any]) -> str:
    if tool_result.get("deduped"):
        return "command_deduped"
    status = str(tool_result.get("status", "")).strip().lower()
    return {
        "applied": "command_applied",
        "deferred": "command_deferred",
        "rejected": "command_rejected",
        "accepted": "command_applied",
    }.get(status, "command_applied")


def command_trace_status(tool_result: dict[str, Any]) -> str:
    if tool_result.get("deduped"):
        return "info"
    status = str(tool_result.get("status", "")).strip().lower()
    if status == "rejected":
        return "error"
    return "success"


def command_trace_message(tool_name: str, tool_result: dict[str, Any]) -> str:
    status = str(tool_result.get("status", "")).strip().lower()
    if tool_result.get("deduped"):
        return f"命令 {tool_name} 命中本轮幂等缓存，复用已有结果。"
    if status == "rejected":
        error_message = str(tool_result.get("error_message", "")).strip()
        if error_message:
            return f"命令 {tool_name} 被 Go 拒绝：{error_message}"
        return f"命令 {tool_name} 被 Go 拒绝。"
    if status == "deferred":
        return f"命令 {tool_name} 已由 Go 裁决，将在主请求收口阶段统一应用。"
    return f"命令 {tool_name} 已返回结构化结果。"


def command_trace_details(tool_name: str, source: dict[str, Any]) -> dict[str, Any]:
    details = {
        "command_type": tool_name,
    }
    for key in ("command_id", "idempotency_key", "status", "error_code"):
        value = source.get(key)
        if isinstance(value, str) and value.strip():
            details[key] = value.strip()
    if isinstance(source.get("deduped"), bool):
        details["deduped"] = source["deduped"]
    return details


def command_status_event_name(tool_result: dict[str, Any]) -> str:
    status = str(tool_result.get("status", "")).strip().lower()
    return {
        "applied": "command_applied",
        "deferred": "command_deferred",
        "rejected": "command_rejected",
        "accepted": "command_applied",
    }.get(status, "command_applied")
