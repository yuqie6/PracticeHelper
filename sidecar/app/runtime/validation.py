from __future__ import annotations

from typing import Any

from app.agent_tools import (
    is_action_tool,
    make_record_observation_tool,
    make_set_depth_signal_tool,
    make_suggest_next_session_tool,
    make_update_knowledge_tool,
)
from app.runtime_support import RuntimeTool
from app.schemas import EvaluateAnswerRequest, EvaluationResult, NextSession, ReviewCard


def read_only_tools(tools: list[RuntimeTool]) -> list[RuntimeTool]:
    return [tool for tool in tools if not is_action_tool(tool.name)]


def bind_runtime_tool(tool: RuntimeTool, runtime_state: Any) -> RuntimeTool:
    if tool.runtime_bind is not None:
        return tool.runtime_bind(runtime_state)
    return rebind_action_tool(tool, runtime_state.side_effects)


def rebind_action_tool(tool: RuntimeTool, side_effects: dict[str, Any]) -> RuntimeTool:
    if tool.name == "record_observation":
        rebound = make_record_observation_tool(side_effects)
    elif tool.name == "update_knowledge":
        rebound = make_update_knowledge_tool(side_effects)
    elif tool.name == "suggest_next_session":
        rebound = make_suggest_next_session_tool(side_effects)
    elif tool.name == "set_depth_signal":
        rebound = make_set_depth_signal_tool(side_effects)
    else:
        return tool
    return rebound


def validate_evaluation_result(
    request: EvaluateAnswerRequest,
    result: EvaluationResult,
    side_effects: dict[str, Any],
    command_results: list[dict[str, Any]],
) -> str:
    score_keys = list(result.score_breakdown.keys())
    if not score_keys:
        return "missing score_breakdown"
    if not result.strengths and not result.gaps:
        return "missing strengths/gaps"

    transition_result = latest_command_result(command_results)
    depth_signal = resolved_depth_signal(transition_result, side_effects)
    if depth_signal == "skip_followup":
        if result.followup_question or result.followup_expected_points:
            return "skip_followup must not include followup output"
        return ""

    max_turns = resolved_max_turns(transition_result, request.max_turns)
    is_last_turn = request.turn_index >= max_turns and depth_signal != "extend"
    if is_last_turn:
        if result.followup_question or result.followup_expected_points:
            return "last turn must not include followup output"
        return ""

    if not result.followup_question:
        return "missing followup_question on non-last turn"
    if not result.followup_expected_points:
        return "missing followup_expected_points on non-last turn"
    return ""


def validate_review_result(
    result: ReviewCard,
    side_effects: dict[str, Any],
    command_results: list[dict[str, Any]],
) -> str:
    if not result.overall:
        return "missing overall"
    if not result.top_fix:
        return "missing top_fix"
    if not result.top_fix_reason:
        return "missing top_fix_reason"
    if not result.score_breakdown:
        return "missing score_breakdown"

    review_path_result = latest_command_result(command_results)
    if command_result_status(review_path_result) == "applied":
        payload = command_result_data(review_path_result)
        if payload:
            expected_next = payload.get("recommended_next")
            if expected_next and result.recommended_next is not None:
                expected_model = NextSession.model_validate(expected_next)
                if result.recommended_next.model_dump(mode="json") != expected_model.model_dump(
                    mode="json"
                ):
                    return "recommended_next must match upsert_review_path result"
            expected_topics = payload.get("suggested_topics")
            if isinstance(expected_topics, list) and result.suggested_topics != expected_topics:
                return "suggested_topics must match upsert_review_path result"
            expected_focus = payload.get("next_training_focus")
            if isinstance(expected_focus, list) and result.next_training_focus != expected_focus:
                return "next_training_focus must match upsert_review_path result"

    if result.recommended_next is None and not side_effects.get("recommended_next"):
        return "missing recommended_next"
    return ""


def latest_command_result(command_results: list[dict[str, Any]]) -> dict[str, Any] | None:
    if not command_results:
        return None
    candidate = command_results[-1]
    if not isinstance(candidate, dict):
        return None
    return candidate


def command_result_status(command_result: dict[str, Any] | None) -> str:
    if not command_result:
        return ""
    status = command_result.get("status", "")
    if not isinstance(status, str):
        return ""
    return status.strip().lower()


def command_result_data(command_result: dict[str, Any] | None) -> dict[str, Any]:
    if not command_result:
        return {}
    payload = command_result.get("data")
    if not isinstance(payload, dict):
        return {}
    return payload


def resolved_depth_signal(
    command_result: dict[str, Any] | None,
    side_effects: dict[str, Any],
) -> str:
    if command_result_status(command_result) == "deferred":
        resolved = command_result_data(command_result).get("resolved_depth_signal")
        if isinstance(resolved, str) and resolved.strip():
            return resolved.strip()
    return str(side_effects.get("depth_signal", "normal")).strip() or "normal"


def resolved_max_turns(command_result: dict[str, Any] | None, default: int) -> int:
    if command_result_status(command_result) == "deferred":
        resolved = command_result_data(command_result).get("resolved_max_turns")
        if isinstance(resolved, int):
            return resolved
        if isinstance(resolved, float):
            return int(resolved)
    return default
