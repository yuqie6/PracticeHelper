from __future__ import annotations

import logging
from typing import TypedDict

from langgraph.graph import END, START, StateGraph

from app.agent_runtime import AgentRuntime, TaskExecutionResult
from app.config import Settings
from app.llm_client import ModelClientError
from app.repo_context import RepoAnalysisBundle, rerank_repo_chunks
from app.runtime_prompts import resolve_question_strategy
from app.schemas import (
    AnalyzeJobTargetEnvelope,
    AnalyzeJobTargetRequest,
    AnalyzeRepoEnvelope,
    AnalyzeRepoRequest,
    EvaluateAnswerEnvelope,
    EvaluateAnswerRequest,
    EvaluationResult,
    GenerateQuestionEnvelope,
    GenerateQuestionRequest,
    GenerateReviewEnvelope,
    GenerateReviewRequest,
    ReviewCard,
)

logger = logging.getLogger(__name__)


class AnalyzeRepoState(TypedDict):
    request: AnalyzeRepoRequest
    bundle: RepoAnalysisBundle
    result: AnalyzeRepoEnvelope


class AnalyzeJobTargetState(TypedDict):
    request: AnalyzeJobTargetRequest
    result: AnalyzeJobTargetEnvelope


class GenerateQuestionState(TypedDict):
    request: GenerateQuestionRequest
    strategy: str
    result: GenerateQuestionEnvelope


class EvaluateAnswerState(TypedDict):
    request: EvaluateAnswerRequest
    prepared_request: EvaluateAnswerRequest
    result: EvaluateAnswerEnvelope | None
    attempts: int
    error: str
    validation_error: str


class GenerateReviewState(TypedDict):
    request: GenerateReviewRequest
    prepared_request: GenerateReviewRequest
    result: GenerateReviewEnvelope | None
    attempts: int
    error: str
    validation_error: str


_MAX_EVALUATE_ATTEMPTS = 2
_MAX_REVIEW_ATTEMPTS = 2


def build_flows(settings: Settings) -> dict[str, object]:
    runtime = AgentRuntime(settings)
    return {
        "analyze_repo": _build_analyze_repo_graph(runtime),
        "analyze_job_target": _build_simple_task_graph(
            AnalyzeJobTargetState,
            runtime.analyze_job_target_task,
            AnalyzeJobTargetEnvelope,
        ),
        "generate_question": _build_generate_question_graph(runtime),
        "evaluate_answer": _build_evaluate_answer_graph(runtime),
        "generate_review": _build_generate_review_graph(runtime),
    }


def _build_simple_task_graph(state_type, run_fn, envelope_cls):
    def run(state):
        return {"result": _to_envelope(run_fn(state["request"]), envelope_cls)}

    graph = StateGraph(state_type)
    graph.add_node("run", run)
    graph.add_edge(START, "run")
    graph.add_edge("run", END)
    return graph.compile()


def _build_analyze_repo_graph(runtime: AgentRuntime):
    def collect_bundle(state: AnalyzeRepoState) -> dict:
        bundle = runtime.collect_repo_bundle(state["request"])
        return {"bundle": bundle}

    def rank_chunks(state: AnalyzeRepoState) -> dict:
        bundle = state["bundle"]
        reranked = rerank_repo_chunks(bundle)
        reranked_bundle = RepoAnalysisBundle(
            repo_url=bundle.repo_url,
            name=bundle.name,
            default_branch=bundle.default_branch,
            import_commit=bundle.import_commit,
            tech_stack=bundle.tech_stack,
            top_paths=bundle.top_paths,
            chunks=reranked,
        )
        return {"bundle": reranked_bundle}

    def summarize(state: AnalyzeRepoState) -> dict:
        task_result = runtime.summarize_repo_bundle(state["bundle"])
        return {"result": _to_envelope(task_result, AnalyzeRepoEnvelope)}

    graph = StateGraph(AnalyzeRepoState)
    graph.add_node("collect_bundle", collect_bundle)
    graph.add_node("rank_chunks", rank_chunks)
    graph.add_node("summarize", summarize)
    graph.add_edge(START, "collect_bundle")
    graph.add_edge("collect_bundle", "rank_chunks")
    graph.add_edge("rank_chunks", "summarize")
    graph.add_edge("summarize", END)
    return graph.compile()


def _build_generate_question_graph(runtime: AgentRuntime):
    def select_strategy(state: GenerateQuestionState) -> dict:
        req = state["request"]
        strategy = resolve_question_strategy(req)
        logger.info("generate_question strategy=%s mode=%s topic=%s", strategy, req.mode, req.topic)
        req_with_strategy = req.model_copy(update={"strategy": strategy})
        return {"strategy": strategy, "request": req_with_strategy}

    def generate(state: GenerateQuestionState) -> dict:
        task_result = runtime.generate_question_task(state["request"])
        return {"result": _to_envelope(task_result, GenerateQuestionEnvelope)}

    graph = StateGraph(GenerateQuestionState)
    graph.add_node("select_strategy", select_strategy)
    graph.add_node("generate", generate)
    graph.add_edge(START, "select_strategy")
    graph.add_edge("select_strategy", "generate")
    graph.add_edge("generate", END)
    return graph.compile()


def _build_evaluate_answer_graph(runtime: AgentRuntime):
    def retrieve_context(state: EvaluateAnswerState) -> dict:
        return {"prepared_request": state["request"]}

    def evaluate(state: EvaluateAnswerState) -> dict:
        attempts = state.get("attempts", 0) + 1
        request = state.get("prepared_request", state["request"])
        try:
            task_result = runtime.evaluate_answer_task(request)
            return {
                "result": _to_envelope(task_result, EvaluateAnswerEnvelope),
                "attempts": attempts,
                "error": "",
                "validation_error": "",
            }
        except Exception as exc:
            logger.warning("evaluate_answer attempt %d failed: %s", attempts, exc)
            return {
                "result": None,
                "attempts": attempts,
                "error": str(exc),
                "validation_error": "",
            }

    def validate_output(state: EvaluateAnswerState) -> str:
        attempts = state.get("attempts", 0)
        error = state.get("error", "")
        envelope = state.get("result")

        if error or envelope is None:
            if attempts >= _MAX_EVALUATE_ATTEMPTS:
                raise ModelClientError(
                    f"evaluate_answer failed after {attempts} attempts: {error or 'no result'}"
                )
            logger.warning("evaluate_answer retrying due to: %s", error or "no result")
            return "re_evaluate"

        validation_error = _validate_evaluation_result(
            state.get("prepared_request", state["request"]),
            envelope.result,
        )
        if not validation_error:
            return "accept"

        if attempts >= _MAX_EVALUATE_ATTEMPTS:
            raise ModelClientError(f"evaluate_answer failed after retries: {validation_error}")

        logger.warning("evaluate_answer output invalid, retrying: %s", validation_error)
        return "re_evaluate"

    def re_evaluate(state: EvaluateAnswerState) -> dict:
        envelope = state.get("result")
        validation_error = ""
        if envelope is not None:
            validation_error = _validate_evaluation_result(
                state.get("prepared_request", state["request"]),
                envelope.result,
            )
        feedback = validation_error or state.get("error", "") or "输出不符合要求"
        prepared = state.get("prepared_request", state["request"]).model_copy(
            update={"retry_feedback": feedback}
        )
        return {"prepared_request": prepared, "validation_error": feedback}

    graph = StateGraph(EvaluateAnswerState)
    graph.add_node("retrieve_context", retrieve_context)
    graph.add_node("evaluate", evaluate)
    graph.add_node("re_evaluate", re_evaluate)
    graph.add_edge(START, "retrieve_context")
    graph.add_edge("retrieve_context", "evaluate")
    graph.add_conditional_edges(
        "evaluate",
        validate_output,
        {"accept": END, "re_evaluate": "re_evaluate"},
    )
    graph.add_edge("re_evaluate", "evaluate")
    return graph.compile()


def _build_generate_review_graph(runtime: AgentRuntime):
    def prepare_review_context(state: GenerateReviewState) -> dict:
        return {"prepared_request": state["request"]}

    def generate_review(state: GenerateReviewState) -> dict:
        attempts = state.get("attempts", 0) + 1
        request = state.get("prepared_request", state["request"])
        try:
            task_result = runtime.generate_review_task(request)
            return {
                "result": _to_envelope(task_result, GenerateReviewEnvelope),
                "attempts": attempts,
                "error": "",
                "validation_error": "",
            }
        except Exception as exc:
            logger.warning("generate_review attempt %d failed: %s", attempts, exc)
            return {
                "result": None,
                "attempts": attempts,
                "error": str(exc),
                "validation_error": "",
            }

    def validate_review(state: GenerateReviewState) -> str:
        attempts = state.get("attempts", 0)
        error = state.get("error", "")
        envelope = state.get("result")

        if error or envelope is None:
            if attempts >= _MAX_REVIEW_ATTEMPTS:
                raise ModelClientError(
                    f"generate_review failed after {attempts} attempts: {error or 'no result'}"
                )
            logger.warning("generate_review retrying due to: %s", error or "no result")
            return "re_generate"

        validation_error = _validate_review_result(envelope.result)
        if not validation_error:
            return "accept"

        if attempts >= _MAX_REVIEW_ATTEMPTS:
            raise ModelClientError(f"generate_review failed after retries: {validation_error}")

        logger.warning("generate_review output invalid, retrying: %s", validation_error)
        return "re_generate"

    def re_generate(state: GenerateReviewState) -> dict:
        envelope = state.get("result")
        validation_error = _validate_review_result(envelope.result) if envelope else ""
        feedback = validation_error or state.get("error", "") or "输出不符合要求"
        prepared = state.get("prepared_request", state["request"]).model_copy(
            update={"retry_feedback": feedback}
        )
        return {"prepared_request": prepared, "validation_error": feedback}

    graph = StateGraph(GenerateReviewState)
    graph.add_node("prepare_review_context", prepare_review_context)
    graph.add_node("generate_review", generate_review)
    graph.add_node("re_generate", re_generate)
    graph.add_edge(START, "prepare_review_context")
    graph.add_edge("prepare_review_context", "generate_review")
    graph.add_conditional_edges(
        "generate_review",
        validate_review,
        {"accept": END, "re_generate": "re_generate"},
    )
    graph.add_edge("re_generate", "generate_review")
    return graph.compile()


def _validate_evaluation_result(request: EvaluateAnswerRequest, result: EvaluationResult) -> str:
    score_keys = list(result.score_breakdown.keys())
    if not score_keys:
        return "missing score_breakdown"
    if not result.strengths and not result.gaps:
        return "missing strengths/gaps"

    is_last_turn = request.turn_index >= request.max_turns
    if is_last_turn:
        if result.followup_question or result.followup_expected_points:
            return "last turn must not include followup output"
        return ""

    if not result.followup_question:
        return "missing followup_question on non-last turn"
    if not result.followup_expected_points:
        return "missing followup_expected_points on non-last turn"
    return ""


def _validate_review_result(result: ReviewCard) -> str:
    if not result.overall:
        return "missing overall"
    if not result.top_fix:
        return "missing top_fix"
    if not result.top_fix_reason:
        return "missing top_fix_reason"
    if not result.score_breakdown:
        return "missing score_breakdown"
    if result.recommended_next is None:
        return "missing recommended_next"
    return ""


def _to_envelope(task_result: TaskExecutionResult, envelope_cls):
    return envelope_cls.model_validate(
        {
            "result": task_result.result.model_dump(mode="json"),
            "raw_output": task_result.raw_output,
        }
    )
