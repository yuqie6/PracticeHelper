from __future__ import annotations

from dataclasses import dataclass, field
from typing import Any

from app.schemas import (
    AgentContext,
    AgentObservation,
    EvaluateAnswerRequest,
    GenerateQuestionRequest,
    GenerateReviewRequest,
)
from app.shared import (
    build_compaction_details,
    compact_chunks,
    compact_string_list,
    trim_text,
)


@dataclass(frozen=True)
class PreparedAgentTooling:
    # 这是给 runtime tools 用的“压缩后上下文快照”，不是请求体的原样镜像。
    # 单独收口成 dataclass，是为了把 token 预算和 trace 细节绑在同一个准备阶段。
    training_context_payload: dict[str, Any]
    weakness_profile_payload: dict[str, Any] = field(default_factory=dict)
    knowledge_graph_payload: dict[str, Any] = field(default_factory=dict)
    observations_payload: dict[str, Any] = field(default_factory=dict)
    session_summaries_payload: dict[str, Any] = field(default_factory=dict)
    trace_details: list[dict[str, Any]] = field(default_factory=list)


def prepare_generate_question_agent_tooling(
    request: GenerateQuestionRequest,
) -> PreparedAgentTooling:
    context = request.agent_context or AgentContext()
    trace_details: list[dict[str, Any]] = []

    templates_raw = [item.model_dump(mode="json") for item in request.templates]
    # 出题阶段只需要少量高信号模板来定风格，不值得把整套题库都塞进上下文。
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
    # 弱项只保留最值得本轮参考的前几条，避免历史包袱压过当前题目上下文。
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
    # 只保留仍指向保留节点的边，并把边数压到节点数的线性级别，
    # 否则图一大就会迅速吞掉 prompt 预算。
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
    # 历史 session 摘要主要用于帮助模型把握近期趋势，不需要把整段复盘原文全部搬进去。
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
