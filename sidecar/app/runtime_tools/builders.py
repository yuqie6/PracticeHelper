from __future__ import annotations

from typing import Any

from app.go_client import GoBackendClient
from app.runtime_support import RuntimeTool
from app.runtime_tools.actions import (
    make_record_observation_tool,
    make_set_depth_signal_tool,
    make_suggest_next_session_tool,
    make_update_knowledge_tool,
)
from app.runtime_tools.commands import (
    make_transition_session_tool,
    make_upsert_review_path_tool,
)
from app.runtime_tools.prepared import (
    PreparedAgentTooling,
    prepare_evaluate_answer_agent_tooling,
    prepare_generate_question_agent_tooling,
    prepare_generate_review_agent_tooling,
)
from app.runtime_tools.readers import (
    get_session_detail_tool,
    payload_tool,
    recall_training_context_tool,
    search_repo_chunks_tool,
)
from app.schemas import EvaluateAnswerRequest, GenerateQuestionRequest, GenerateReviewRequest

ACTION_TOOL_NAMES = {
    "record_observation",
    "update_knowledge",
    "suggest_next_session",
    "set_depth_signal",
    "transition_session",
    "upsert_review_path",
}


def build_generate_question_agent_tools(
    request: GenerateQuestionRequest,
    backend_client: GoBackendClient | None = None,
    *,
    prepared: PreparedAgentTooling | None = None,
) -> list[RuntimeTool]:
    prepared = prepared or prepare_generate_question_agent_tooling(request)
    tools = [
        recall_training_context_tool(lambda: prepared.training_context_payload),
        payload_tool(
            "recall_knowledge_graph",
            "Read the relevant knowledge graph subgraph for the current topic.",
            lambda: prepared.knowledge_graph_payload,
        ),
        payload_tool(
            "recall_observations",
            "Read prior observations and strategy notes relevant to this user and topic.",
            lambda: prepared.observations_payload,
        ),
    ]
    if backend_client is not None and backend_client.enabled and request.project:
        tools.append(search_repo_chunks_tool(request.project.id, backend_client))
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
        recall_training_context_tool(lambda: prepared.training_context_payload),
        payload_tool(
            "recall_knowledge_graph",
            "Read the relevant knowledge graph subgraph for the current topic.",
            lambda: prepared.knowledge_graph_payload,
        ),
        payload_tool(
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
        tools.append(search_repo_chunks_tool(request.project.id, backend_client))
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
        recall_training_context_tool(lambda: prepared.training_context_payload),
        payload_tool(
            "recall_weakness_profile",
            "Read the current weakness profile accumulated from previous sessions.",
            lambda: prepared.weakness_profile_payload,
        ),
        payload_tool(
            "recall_knowledge_graph",
            "Read the relevant knowledge graph subgraph for the current topic.",
            lambda: prepared.knowledge_graph_payload,
        ),
        payload_tool(
            "recall_observations",
            "Read prior observations and strategy notes relevant to this user and topic.",
            lambda: prepared.observations_payload,
        ),
        payload_tool(
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
        tools.append(get_session_detail_tool(request.session.id, backend_client))
    return tools


def is_action_tool(tool_name: str) -> bool:
    return tool_name in ACTION_TOOL_NAMES
