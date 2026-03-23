from __future__ import annotations

from typing import Any

from app.runtime_support import RuntimeTool
from app.schemas import AgentObservation, DepthSignal, KnowledgeUpdate, NextSession


def make_record_observation_tool(side_effects: dict[str, Any]) -> RuntimeTool:
    return RuntimeTool(
        name="record_observation",
        description=(
            "Record a reusable observation about the user's pattern, "
            "misconception, growth, or strategy note."
        ),
        parameters={
            "type": "object",
            "properties": {
                "category": {"type": "string"},
                "content": {"type": "string"},
                "tags": {"type": "array", "items": {"type": "string"}},
                "relevance": {"type": "number"},
                "topic": {"type": "string"},
                "scope_type": {"type": "string"},
                "scope_id": {"type": "string"},
            },
            "required": ["category", "content"],
            "additionalProperties": False,
        },
        handler=lambda arguments: _record_observation(side_effects, arguments),
    )


def make_update_knowledge_tool(side_effects: dict[str, Any]) -> RuntimeTool:
    return RuntimeTool(
        name="update_knowledge",
        description="Update or create a knowledge node based on the current evaluation or review.",
        parameters={
            "type": "object",
            "properties": {
                "node_id": {"type": "string"},
                "scope_type": {"type": "string"},
                "scope_id": {"type": "string"},
                "parent_id": {"type": "string"},
                "label": {"type": "string"},
                "node_type": {"type": "string"},
                "proficiency": {"type": "number"},
                "confidence": {"type": "number"},
                "evidence": {"type": "string"},
                "observed_label": {"type": "string"},
            },
            "required": ["proficiency"],
            "additionalProperties": False,
        },
        handler=lambda arguments: _update_knowledge(side_effects, arguments),
    )


def make_set_depth_signal_tool(side_effects: dict[str, Any]) -> RuntimeTool:
    return RuntimeTool(
        name="set_depth_signal",
        description=(
            "Set whether the Go FSM should skip follow-up, extend by one turn, "
            "or keep the normal depth."
        ),
        parameters={
            "type": "object",
            "properties": {
                "depth_signal": {
                    "type": "string",
                    "enum": ["skip_followup", "extend", "normal"],
                }
            },
            "required": ["depth_signal"],
            "additionalProperties": False,
        },
        handler=lambda arguments: _set_depth_signal(side_effects, arguments),
    )


def make_suggest_next_session_tool(side_effects: dict[str, Any]) -> RuntimeTool:
    return RuntimeTool(
        name="suggest_next_session",
        description=(
            "Recommend the next training session configuration to continue the learning path."
        ),
        parameters={
            "type": "object",
            "properties": {
                "mode": {"type": "string"},
                "topic": {"type": "string"},
                "project_id": {"type": "string"},
                "reason": {"type": "string"},
            },
            "required": ["mode"],
            "additionalProperties": False,
        },
        handler=lambda arguments: _suggest_next_session(side_effects, arguments),
    )


def _record_observation(side_effects: dict[str, Any], arguments: dict[str, Any]) -> dict[str, Any]:
    observation = AgentObservation.model_validate(arguments)
    observations = side_effects.setdefault("observations", [])
    observations.append(observation.model_dump(mode="json"))
    return {"status": "recorded", "count": len(observations)}


def _update_knowledge(side_effects: dict[str, Any], arguments: dict[str, Any]) -> dict[str, Any]:
    update = KnowledgeUpdate.model_validate(arguments)
    updates = side_effects.setdefault("knowledge_updates", [])
    updates.append(update.model_dump(mode="json"))
    return {"status": "queued", "count": len(updates)}


def _set_depth_signal(side_effects: dict[str, Any], arguments: dict[str, Any]) -> dict[str, Any]:
    signal = arguments.get("depth_signal", "normal")
    if signal not in {"skip_followup", "extend", "normal"}:
        raise ValueError(f"unsupported depth_signal: {signal}")
    validated: DepthSignal = signal  # type: ignore[assignment]
    side_effects["depth_signal"] = validated
    return {"status": "set", "depth_signal": validated}


def _suggest_next_session(
    side_effects: dict[str, Any], arguments: dict[str, Any]
) -> dict[str, Any]:
    suggestion = NextSession.model_validate(arguments)
    side_effects["recommended_next"] = suggestion.model_dump(mode="json")
    return {"status": "prepared", "recommended_next": suggestion.model_dump(mode="json")}
