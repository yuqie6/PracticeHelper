from __future__ import annotations

from collections.abc import Callable
from dataclasses import dataclass, field
from typing import Any

from app.go_client import GoBackendClient
from app.runtime_support import (
    RuntimeTool,
    build_compaction_details,
    compact_chunks,
    compact_string_list,
    trim_text,
)
from app.schemas import (
    AgentCommandEnvelope,
    AgentCommandResult,
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
    "transition_session",
    "upsert_review_path",
}


class CommandBudgetExceededError(RuntimeError):
    """Raised when a command tool exceeds the per-run command budget."""


@dataclass(frozen=True)
class PreparedAgentTooling:
    training_context_payload: dict[str, Any]
    weakness_profile_payload: dict[str, Any] = field(default_factory=dict)
    knowledge_graph_payload: dict[str, Any] = field(default_factory=dict)
    observations_payload: dict[str, Any] = field(default_factory=dict)
    session_summaries_payload: dict[str, Any] = field(default_factory=dict)
    trace_details: list[dict[str, Any]] = field(default_factory=list)


def build_generate_question_agent_tools(
    request: GenerateQuestionRequest,
    backend_client: GoBackendClient | None = None,
    *,
    prepared: PreparedAgentTooling | None = None,
) -> list[RuntimeTool]:
    prepared = prepared or prepare_generate_question_agent_tooling(request)
    tools = [
        _recall_training_context_tool(lambda: prepared.training_context_payload),
        _payload_tool(
            "recall_knowledge_graph",
            "Read the relevant knowledge graph subgraph for the current topic.",
            lambda: prepared.knowledge_graph_payload,
        ),
        _payload_tool(
            "recall_observations",
            "Read prior observations and strategy notes relevant to this user and topic.",
            lambda: prepared.observations_payload,
        ),
    ]
    if backend_client is not None and backend_client.enabled and request.project:
        tools.append(_search_repo_chunks_tool(request.project.id, backend_client))
    return tools


def build_evaluate_answer_agent_tools(
    request: EvaluateAnswerRequest,
    side_effects: dict[str, Any],
    backend_client: GoBackendClient | None = None,
    *,
    prepared: PreparedAgentTooling | None = None,
) -> list[RuntimeTool]:
    prepared = prepared or prepare_evaluate_answer_agent_tooling(request)
    tools = [
        _recall_training_context_tool(lambda: prepared.training_context_payload),
        _payload_tool(
            "recall_knowledge_graph",
            "Read the relevant knowledge graph subgraph for the current topic.",
            lambda: prepared.knowledge_graph_payload,
        ),
        _payload_tool(
            "recall_observations",
            "Read prior observations and strategy notes relevant to this user and topic.",
            lambda: prepared.observations_payload,
        ),
        make_record_observation_tool(side_effects),
        make_update_knowledge_tool(side_effects),
        make_set_depth_signal_tool(side_effects),
        make_transition_session_tool(request, backend_client),
    ]
    if backend_client is not None and backend_client.enabled and request.project:
        tools.append(_search_repo_chunks_tool(request.project.id, backend_client))
    return tools


def build_generate_review_agent_tools(
    request: GenerateReviewRequest,
    side_effects: dict[str, Any],
    backend_client: GoBackendClient | None = None,
    *,
    prepared: PreparedAgentTooling | None = None,
) -> list[RuntimeTool]:
    prepared = prepared or prepare_generate_review_agent_tooling(request)
    tools = [
        _recall_training_context_tool(lambda: prepared.training_context_payload),
        _payload_tool(
            "recall_weakness_profile",
            "Read the current weakness profile accumulated from previous sessions.",
            lambda: prepared.weakness_profile_payload,
        ),
        _payload_tool(
            "recall_knowledge_graph",
            "Read the relevant knowledge graph subgraph for the current topic.",
            lambda: prepared.knowledge_graph_payload,
        ),
        _payload_tool(
            "recall_observations",
            "Read prior observations and strategy notes relevant to this user and topic.",
            lambda: prepared.observations_payload,
        ),
        _payload_tool(
            "recall_session_summaries",
            (
                "Read compact summaries from recent similar sessions instead of replaying "
                "full turn history."
            ),
            lambda: prepared.session_summaries_payload,
        ),
        make_record_observation_tool(side_effects),
        make_update_knowledge_tool(side_effects),
        make_suggest_next_session_tool(side_effects),
        make_upsert_review_path_tool(request, backend_client),
    ]
    if backend_client is not None and backend_client.enabled and request.session.id:
        tools.append(_get_session_detail_tool(request.session.id, backend_client))
    return tools


def is_action_tool(tool_name: str) -> bool:
    return tool_name in ACTION_TOOL_NAMES


def prepare_generate_question_agent_tooling(
    request: GenerateQuestionRequest,
) -> PreparedAgentTooling:
    context = request.agent_context or AgentContext()
    trace_details: list[dict[str, Any]] = []

    templates_raw = [item.model_dump(mode="json") for item in request.templates]
    templates = [_compact_question_template(item) for item in request.templates[:6]]
    if templates_raw:
        trace_details.append(
            build_compaction_details(
                section="templates",
                before_count=len(request.templates),
                after_count=len(templates),
                before_value=templates_raw,
                after_value=templates,
                budget="limit<=6 prompt<=180 followups<=2 bad_answers<=2",
            )
        )

    weakness_raw = [item.model_dump(mode="json") for item in request.weaknesses]
    weaknesses = [_compact_weakness(item) for item in request.weaknesses[:5]]
    if weakness_raw:
        trace_details.append(
            build_compaction_details(
                section="weakness_profile",
                before_count=len(request.weaknesses),
                after_count=len(weaknesses),
                before_value=weakness_raw,
                after_value=weaknesses,
                budget="limit<=5 fields=kind,label,severity,frequency",
            )
        )

    knowledge_payload, knowledge_detail = _compact_knowledge_payload(context)
    if knowledge_detail:
        trace_details.append(knowledge_detail)

    observations_payload, observations_detail = _compact_observations_payload(context)
    if observations_detail:
        trace_details.append(observations_detail)

    return PreparedAgentTooling(
        training_context_payload={
            "mode": request.mode,
            "topic": request.topic,
            "candidate_topics": compact_string_list(
                request.candidate_topics, limit=8, max_chars=48
            ),
            "intensity": request.intensity,
            "project": request.project.model_dump(mode="json") if request.project else None,
            "templates": templates,
            "context_chunks": compact_chunks(request.context_chunks),
            "weaknesses": weaknesses,
            "job_target_analysis": (
                request.job_target_analysis.model_dump(mode="json")
                if request.job_target_analysis
                else None
            ),
        },
        knowledge_graph_payload=knowledge_payload,
        observations_payload=observations_payload,
        trace_details=trace_details,
    )


def prepare_evaluate_answer_agent_tooling(
    request: EvaluateAnswerRequest,
) -> PreparedAgentTooling:
    context = request.agent_context or AgentContext()
    trace_details: list[dict[str, Any]] = []

    knowledge_payload, knowledge_detail = _compact_knowledge_payload(context)
    if knowledge_detail:
        trace_details.append(knowledge_detail)

    observations_payload, observations_detail = _compact_observations_payload(context)
    if observations_detail:
        trace_details.append(observations_detail)

    return PreparedAgentTooling(
        training_context_payload={
            "mode": request.mode,
            "topic": request.topic,
            "question": request.question,
            "expected_points": compact_string_list(request.expected_points, limit=6, max_chars=80),
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
        },
        knowledge_graph_payload=knowledge_payload,
        observations_payload=observations_payload,
        trace_details=trace_details,
    )


def prepare_generate_review_agent_tooling(
    request: GenerateReviewRequest,
) -> PreparedAgentTooling:
    context = request.agent_context or AgentContext()
    trace_details: list[dict[str, Any]] = []

    turns_raw = [turn.model_dump(mode="json") for turn in request.turns]
    compact_turns = [_compact_review_turn(turn) for turn in request.turns]
    if turns_raw:
        trace_details.append(
            build_compaction_details(
                section="turns",
                before_count=len(request.turns),
                after_count=len(compact_turns),
                before_value=turns_raw,
                after_value=compact_turns,
                budget="answer<=240 strengths<=2 gaps<=2",
            )
        )

    weakness_raw = [item.model_dump(mode="json") for item in context.weakness_profile]
    weaknesses = [_compact_weakness(item) for item in context.weakness_profile[:5]]
    if weakness_raw:
        trace_details.append(
            build_compaction_details(
                section="weakness_profile",
                before_count=len(context.weakness_profile),
                after_count=len(weaknesses),
                before_value=weakness_raw,
                after_value=weaknesses,
                budget="limit<=5 fields=kind,label,severity,frequency",
            )
        )

    knowledge_payload, knowledge_detail = _compact_knowledge_payload(context)
    if knowledge_detail:
        trace_details.append(knowledge_detail)

    observations_payload, observations_detail = _compact_observations_payload(context)
    if observations_detail:
        trace_details.append(observations_detail)

    summaries_payload, summaries_detail = _compact_session_summaries_payload(context)
    if summaries_detail:
        trace_details.append(summaries_detail)

    return PreparedAgentTooling(
        training_context_payload={
            "session": _compact_review_session(request),
            "project": request.project.model_dump(mode="json") if request.project else None,
            "turns": compact_turns,
            "job_target_analysis": (
                request.job_target_analysis.model_dump(mode="json")
                if request.job_target_analysis
                else None
            ),
        },
        weakness_profile_payload={"weakness_profile": weaknesses},
        knowledge_graph_payload=knowledge_payload,
        observations_payload=observations_payload,
        session_summaries_payload=summaries_payload,
        trace_details=trace_details,
    )


def _recall_training_context_tool(loader: Callable[[], dict[str, Any]]) -> RuntimeTool:
    return RuntimeTool(
        name="recall_training_context",
        description=(
            "Read the current task context, including question, answer, "
            "project, and template details."
        ),
        handler=lambda _: loader(),
    )


def _payload_tool(
    name: str,
    description: str,
    loader: Callable[[], dict[str, Any]],
) -> RuntimeTool:
    return RuntimeTool(
        name=name,
        description=description,
        handler=lambda _: loader(),
    )


def _compact_question_template(template: Any) -> dict[str, Any]:
    return {
        "id": template.id,
        "topic": template.topic,
        "prompt": trim_text(template.prompt, 180),
        "focus_points": compact_string_list(template.focus_points, limit=3, max_chars=72),
        "bad_answers": compact_string_list(template.bad_answers, limit=2, max_chars=90),
        "followup_templates": compact_string_list(
            template.followup_templates, limit=2, max_chars=90
        ),
        "score_weights": dict(template.score_weights),
    }


def _compact_weakness(item: Any) -> dict[str, Any]:
    return {
        "kind": item.kind,
        "label": item.label,
        "severity": item.severity,
        "frequency": item.frequency,
    }


def _compact_review_session(request: GenerateReviewRequest) -> dict[str, Any]:
    session = request.session
    return {
        "id": session.id,
        "mode": session.mode,
        "topic": session.topic,
        "project_id": session.project_id,
        "job_target_id": session.job_target_id,
        "job_target_analysis_id": session.job_target_analysis_id,
        "intensity": session.intensity,
        "status": session.status,
        "max_turns": session.max_turns,
        "total_score": session.total_score,
    }


def _compact_review_turn(turn: Any) -> dict[str, Any]:
    evaluation = turn.evaluation
    return {
        "turn_index": turn.turn_index,
        "question": trim_text(turn.question, 180),
        "answer_excerpt": trim_text(turn.answer, 240),
        "score": evaluation.score if evaluation else None,
        "headline": trim_text(evaluation.headline, 120) if evaluation else "",
        "top_gaps": compact_string_list(
            evaluation.gaps if evaluation else [], limit=2, max_chars=80
        ),
        "top_strengths": compact_string_list(
            evaluation.strengths if evaluation else [], limit=2, max_chars=80
        ),
    }


def _compact_knowledge_payload(
    context: AgentContext,
) -> tuple[dict[str, Any], dict[str, Any] | None]:
    if context.knowledge_subgraph is None:
        return {"knowledge_subgraph": {"nodes": [], "edges": []}}, None

    before_value = context.knowledge_subgraph.model_dump(mode="json")
    compact_nodes = [
        {
            "id": node.id,
            "label": node.label,
            "node_type": node.node_type,
            "proficiency": node.proficiency,
            "confidence": node.confidence,
            "parent_id": node.parent_id,
        }
        for node in context.knowledge_subgraph.nodes
    ]
    node_ids = {node["id"] for node in compact_nodes}
    compact_edges = [
        {
            "source_id": edge.source_id,
            "target_id": edge.target_id,
            "edge_type": edge.edge_type,
        }
        for edge in context.knowledge_subgraph.edges
        if edge.source_id in node_ids and edge.target_id in node_ids
    ][: max(len(compact_nodes) * 2, 0)]
    after_value = {"nodes": compact_nodes, "edges": compact_edges}
    return {"knowledge_subgraph": after_value}, build_compaction_details(
        section="knowledge_subgraph",
        before_count=len(context.knowledge_subgraph.nodes) + len(context.knowledge_subgraph.edges),
        after_count=len(compact_nodes) + len(compact_edges),
        before_value=before_value,
        after_value=after_value,
        budget="node_fields=6 edge_limit<=nodes*2",
    )


def _compact_observations_payload(
    context: AgentContext,
) -> tuple[dict[str, Any], dict[str, Any] | None]:
    raw = [item.model_dump(mode="json") for item in context.observations]
    compact = [_compact_observation(item) for item in context.observations]
    details = None
    if raw:
        details = build_compaction_details(
            section="observations",
            before_count=len(context.observations),
            after_count=len(compact),
            before_value=raw,
            after_value=compact,
            budget="content<=180 tags<=4",
        )
    return {"observations": compact}, details


def _compact_observation(item: AgentObservation) -> dict[str, Any]:
    return {
        "id": item.id,
        "category": item.category,
        "topic": item.topic,
        "scope_type": item.scope_type,
        "scope_id": item.scope_id,
        "content": trim_text(item.content, 180),
        "tags": compact_string_list(item.tags, limit=4, max_chars=32),
        "relevance": item.relevance,
    }


def _compact_session_summaries_payload(
    context: AgentContext,
) -> tuple[dict[str, Any], dict[str, Any] | None]:
    raw = [item.model_dump(mode="json") for item in context.session_summaries]
    compact = [
        {
            "id": item.id,
            "session_id": item.session_id,
            "mode": item.mode,
            "topic": item.topic,
            "project_id": item.project_id,
            "job_target_id": item.job_target_id,
            "summary": trim_text(item.summary, 240),
            "strengths": compact_string_list(item.strengths, limit=2, max_chars=80),
            "gaps": compact_string_list(item.gaps, limit=2, max_chars=80),
            "recommended_focus": compact_string_list(item.recommended_focus, limit=2, max_chars=80),
            "salience": item.salience,
        }
        for item in context.session_summaries
    ]
    details = None
    if raw:
        details = build_compaction_details(
            section="session_summaries",
            before_count=len(context.session_summaries),
            after_count=len(compact),
            before_value=raw,
            after_value=compact,
            budget="summary<=240 strengths/gaps/focus<=2",
        )
    return {"session_summaries": compact}, details


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
    return _execute_command(backend_client, state, command)


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
            "suggested_topics": _coerce_string_list(arguments.get("suggested_topics")),
            "next_training_focus": _coerce_string_list(arguments.get("next_training_focus")),
            "gaps": _coerce_string_list(arguments.get("gaps")),
            "top_fix": str(arguments.get("top_fix", "")).strip(),
            "top_fix_reason": str(arguments.get("top_fix_reason", "")).strip(),
        },
    )
    return _execute_command(backend_client, state, command)


def _execute_command(
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
    serialized = _serialize_command_result(command, result, deduped=False)
    state.command_counts[command.command_type] = used + 1
    state.command_cache[command.idempotency_key] = serialized
    state.command_results.append(result.model_dump(mode="json"))
    return serialized


def _serialize_command_result(
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


def _coerce_string_list(value: Any) -> list[str]:
    if not isinstance(value, list):
        return []
    items: list[str] = []
    for item in value:
        text = str(item).strip()
        if text:
            items.append(text)
    return items
