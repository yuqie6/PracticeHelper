from __future__ import annotations

import logging
from typing import TypedDict

from langgraph.graph import END, START, StateGraph
from app.llm_client import ModelClientError
from app.runtime_prompts import resolve_question_strategy

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
    result: EvaluationResult | None
    attempts: int
    error: str


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
        strategy = resolve_question_strategy(req)
        logger.info("generate_question strategy=%s mode=%s topic=%s", strategy, req.mode, req.topic)
        req_with_strategy = req.model_copy(update={"strategy": strategy})
        return {"strategy": strategy, "request": req_with_strategy}

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
        attempts = state.get("attempts", 0) + 1
        try:
            result = runtime.evaluate_answer(state["request"])
            validated = EvaluationResult.model_validate(result)
            return {"result": validated, "attempts": attempts, "error": ""}
        except Exception as exc:
            logger.warning("evaluate_answer attempt %d failed: %s", attempts, exc)
            return {"result": None, "attempts": attempts, "error": str(exc)}

    def should_retry(state: EvaluateAnswerState) -> str:
        attempts = state.get("attempts", 0)
        error = state.get("error", "")
        result = state.get("result")

        if error or result is None:
            if attempts >= _MAX_EVALUATE_ATTEMPTS:
                raise ModelClientError(f"evaluate_answer failed after {attempts} attempts: {error or 'no result'}")
            logger.warning("evaluate_answer retrying due to: %s", error or "no result")
            return "retry"

        if not result.strengths and not result.gaps:
            if attempts >= _MAX_EVALUATE_ATTEMPTS:
                raise ModelClientError("evaluate_answer failed after retries: missing strengths/gaps")
            logger.warning("evaluate_answer output missing strengths/gaps, retrying")
            return "retry"

        is_last_turn = state["request"].turn_index >= state["request"].max_turns
        if not is_last_turn and not result.followup_question:
            if attempts >= _MAX_EVALUATE_ATTEMPTS:
                raise ModelClientError(
                    "evaluate_answer failed after retries: missing followup_question on non-last turn"
                )
            logger.warning("evaluate_answer missing followup_question on non-last turn, retrying")
            return "retry"

        return "accept"

    graph = StateGraph(EvaluateAnswerState)
    graph.add_node("evaluate", evaluate)
    graph.add_edge(START, "evaluate")
    graph.add_conditional_edges("evaluate", should_retry, {"accept": END, "retry": "evaluate"})
    return graph.compile()
