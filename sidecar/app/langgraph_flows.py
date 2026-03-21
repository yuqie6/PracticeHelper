from __future__ import annotations

import logging
from typing import TypedDict

from langgraph.graph import END, START, StateGraph

from app.agent_runtime import AgentRuntime
from app.config import Settings
from app.schemas import (
    AnalyzeJobTargetRequest,
    AnalyzeJobTargetResponse,
    AnalyzeRepoRequest,
    AnalyzeRepoResponse,
    EvaluateAnswerRequest,
    EvaluationResult,
    GenerateQuestionRequest,
    GenerateQuestionResponse,
    GenerateReviewRequest,
    ReviewCard,
)

logger = logging.getLogger(__name__)


class AnalyzeRepoState(TypedDict):
    request: AnalyzeRepoRequest
    result: AnalyzeRepoResponse


class AnalyzeJobTargetState(TypedDict):
    request: AnalyzeJobTargetRequest
    result: AnalyzeJobTargetResponse


class GenerateQuestionState(TypedDict):
    request: GenerateQuestionRequest
    strategy: str
    result: GenerateQuestionResponse


class EvaluateAnswerState(TypedDict):
    request: EvaluateAnswerRequest
    result: EvaluationResult
    attempts: int


class GenerateReviewState(TypedDict):
    request: GenerateReviewRequest
    result: ReviewCard


_MAX_EVALUATE_ATTEMPTS = 2


def build_flows(settings: Settings) -> dict[str, object]:
    runtime = AgentRuntime(settings)
    return {
        "analyze_repo": _build_simple_graph(
            AnalyzeRepoState, runtime.analyze_repo, AnalyzeRepoResponse
        ),
        "analyze_job_target": _build_simple_graph(
            AnalyzeJobTargetState, runtime.analyze_job_target, AnalyzeJobTargetResponse
        ),
        "generate_question": _build_generate_question_graph(runtime),
        "evaluate_answer": _build_evaluate_answer_graph(runtime),
        "generate_review": _build_simple_graph(
            GenerateReviewState, runtime.generate_review, ReviewCard
        ),
    }


def _build_simple_graph(state_type, run_fn, result_model):
    def run(state):
        result = run_fn(state["request"])
        return {"result": result_model.model_validate(result)}

    graph = StateGraph(state_type)
    graph.add_node("run", run)
    graph.add_edge(START, "run")
    graph.add_edge("run", END)
    return graph.compile()


def _build_generate_question_graph(runtime: AgentRuntime):
    def select_strategy(state: GenerateQuestionState) -> dict:
        req = state["request"]
        if req.weaknesses and any(w.severity >= 0.8 for w in req.weaknesses):
            strategy = "weakness_first"
        elif req.mode == "project" and req.project:
            strategy = "project_deep_dive"
        else:
            strategy = "template_based"
        logger.info("generate_question strategy=%s mode=%s topic=%s", strategy, req.mode, req.topic)
        return {"strategy": strategy}

    def generate(state: GenerateQuestionState) -> dict:
        result = runtime.generate_question(state["request"])
        return {"result": GenerateQuestionResponse.model_validate(result)}

    graph = StateGraph(GenerateQuestionState)
    graph.add_node("select_strategy", select_strategy)
    graph.add_node("generate", generate)
    graph.add_edge(START, "select_strategy")
    graph.add_edge("select_strategy", "generate")
    graph.add_edge("generate", END)
    return graph.compile()


def _build_evaluate_answer_graph(runtime: AgentRuntime):
    def evaluate(state: EvaluateAnswerState) -> dict:
        result = runtime.evaluate_answer(state["request"])
        validated = EvaluationResult.model_validate(result)
        return {"result": validated, "attempts": state.get("attempts", 0) + 1}

    def should_retry(state: EvaluateAnswerState) -> str:
        result = state.get("result")
        attempts = state.get("attempts", 0)
        if attempts >= _MAX_EVALUATE_ATTEMPTS:
            return "accept"

        if result is None:
            return "retry"

        if not isinstance(result, EvaluationResult):
            return "retry"

        if not result.strengths and not result.gaps:
            logger.warning("evaluate_answer output missing strengths/gaps, retrying")
            return "retry"

        is_last_turn = state["request"].turn_index >= state["request"].max_turns
        if not is_last_turn and not result.followup_question:
            logger.warning("evaluate_answer missing followup_question on non-last turn, retrying")
            return "retry"

        return "accept"

    graph = StateGraph(EvaluateAnswerState)
    graph.add_node("evaluate", evaluate)
    graph.add_edge(START, "evaluate")
    graph.add_conditional_edges("evaluate", should_retry, {"accept": END, "retry": "evaluate"})
    return graph.compile()
