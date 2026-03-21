from __future__ import annotations

from collections.abc import Callable
from typing import Any

from app.go_client import GoBackendClient
from app.runtime_support import RuntimeTool, compact_chunks
from app.schemas import (
    AgentContext,
    AgentObservation,
    DepthSignal,
    EvaluateAnswerRequest,
    GenerateQuestionRequest,
    GenerateReviewRequest,
    KnowledgeUpdate,
    NextSession,
)

ACTION_TOOL_NAMES = {
    "record_observation",
    "update_knowledge",
    "suggest_next_session",
    "set_depth_signal",
}


def build_generate_question_agent_tools(
    request: GenerateQuestionRequest,
    backend_client: GoBackendClient | None = None,
) -> list[RuntimeTool]:
    context = request.agent_context or AgentContext()
    tools = [
        _recall_training_context_tool(
            lambda: {
                "mode": request.mode,
                "topic": request.topic,
                "candidate_topics": request.candidate_topics,
                "intensity": request.intensity,
                "project": request.project.model_dump(mode="json") if request.project else None,
                "templates": [item.model_dump(mode="json") for item in request.templates],
                "context_chunks": compact_chunks(request.context_chunks),
                "weaknesses": [item.model_dump(mode="json") for item in request.weaknesses],
                "job_target_analysis": (
                    request.job_target_analysis.model_dump(mode="json")
                    if request.job_target_analysis
                    else None
                ),
            }
        ),
        _recall_knowledge_graph_tool(context),
        _recall_observations_tool(context),
    ]
    if backend_client is not None and backend_client.enabled and request.project:
        tools.append(_search_repo_chunks_tool(request.project.id, backend_client))
    return tools


def build_evaluate_answer_agent_tools(
    request: EvaluateAnswerRequest,
    side_effects: dict[str, Any],
    backend_client: GoBackendClient | None = None,
) -> list[RuntimeTool]:
    context = request.agent_context or AgentContext()
    tools = [
        _recall_training_context_tool(
            lambda: {
                "mode": request.mode,
                "topic": request.topic,
                "question": request.question,
                "expected_points": request.expected_points,
                "answer": request.answer,
                "project": request.project.model_dump(mode="json") if request.project else None,
                "context_chunks": compact_chunks(request.context_chunks),
                "turn_index": request.turn_index,
                "max_turns": request.max_turns,
                "job_target_analysis": (
                    request.job_target_analysis.model_dump(mode="json")
                    if request.job_target_analysis
                    else None
                ),
            }
        ),
        _recall_knowledge_graph_tool(context),
        _recall_observations_tool(context),
        make_record_observation_tool(side_effects),
        make_update_knowledge_tool(side_effects),
        make_set_depth_signal_tool(side_effects),
    ]
    if backend_client is not None and backend_client.enabled and request.project:
        tools.append(_search_repo_chunks_tool(request.project.id, backend_client))
    return tools


def build_generate_review_agent_tools(
    request: GenerateReviewRequest,
    side_effects: dict[str, Any],
    backend_client: GoBackendClient | None = None,
) -> list[RuntimeTool]:
    context = request.agent_context or AgentContext()
    tools = [
        _recall_training_context_tool(
            lambda: {
                "session": request.session.model_dump(mode="json"),
                "project": request.project.model_dump(mode="json") if request.project else None,
                "turns": [turn.model_dump(mode="json") for turn in request.turns],
                "job_target_analysis": (
                    request.job_target_analysis.model_dump(mode="json")
                    if request.job_target_analysis
                    else None
                ),
            }
        ),
        _recall_weakness_profile_tool(context),
        _recall_knowledge_graph_tool(context),
        _recall_observations_tool(context),
        _recall_session_summaries_tool(context),
        make_record_observation_tool(side_effects),
        make_update_knowledge_tool(side_effects),
        make_suggest_next_session_tool(side_effects),
    ]
    if backend_client is not None and backend_client.enabled and request.session.id:
        tools.append(_get_session_detail_tool(request.session.id, backend_client))
    return tools


def is_action_tool(tool_name: str) -> bool:
    return tool_name in ACTION_TOOL_NAMES


def _recall_training_context_tool(loader: Callable[[], dict[str, Any]]) -> RuntimeTool:
    return RuntimeTool(
        name="recall_training_context",
        description=(
            "Read the current task context, including question, answer, "
            "project, and template details."
        ),
        handler=lambda _: loader(),
    )


def _recall_weakness_profile_tool(context: AgentContext) -> RuntimeTool:
    return RuntimeTool(
        name="recall_weakness_profile",
        description="Read the current weakness profile accumulated from previous sessions.",
        handler=lambda _: {
            "weakness_profile": [item.model_dump(mode="json") for item in context.weakness_profile]
        },
    )


def _recall_knowledge_graph_tool(context: AgentContext) -> RuntimeTool:
    return RuntimeTool(
        name="recall_knowledge_graph",
        description="Read the relevant knowledge graph subgraph for the current topic.",
        handler=lambda _: {
            "knowledge_subgraph": (
                context.knowledge_subgraph.model_dump(mode="json")
                if context.knowledge_subgraph
                else {"nodes": [], "edges": []}
            )
        },
    )


def _recall_observations_tool(context: AgentContext) -> RuntimeTool:
    return RuntimeTool(
        name="recall_observations",
        description="Read prior observations and strategy notes relevant to this user and topic.",
        handler=lambda _: {
            "observations": [item.model_dump(mode="json") for item in context.observations]
        },
    )


def _recall_session_summaries_tool(context: AgentContext) -> RuntimeTool:
    return RuntimeTool(
        name="recall_session_summaries",
        description=(
            "Read compact summaries from recent similar sessions instead of "
            "replaying full turn history."
        ),
        handler=lambda _: {
            "session_summaries": [
                item.model_dump(mode="json") for item in context.session_summaries
            ]
        },
    )


def _search_repo_chunks_tool(project_id: str, backend_client: GoBackendClient) -> RuntimeTool:
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


def _get_session_detail_tool(session_id: str, backend_client: GoBackendClient) -> RuntimeTool:
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
            "Recommend the next training session configuration to continue the "
            "learning path."
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
