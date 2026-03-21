from __future__ import annotations

import logging
from typing import TypedDict

from langgraph.graph import END, START, StateGraph

from app.agent_runtime import AgentRuntime, TaskExecutionResult
from app.config import Settings
from app.repo_context import RepoAnalysisBundle, rerank_repo_chunks
from app.runtime_prompts import resolve_question_strategy
from app.schemas import (
    AnalyzeJobTargetEnvelope,
    AnalyzeJobTargetRequest,
    AnalyzeRepoEnvelope,
    AnalyzeRepoRequest,
    EvaluateAnswerEnvelope,
    EvaluateAnswerRequest,
    GenerateQuestionEnvelope,
    GenerateQuestionRequest,
    GenerateReviewEnvelope,
    GenerateReviewRequest,
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
    result: EvaluateAnswerEnvelope


class GenerateReviewState(TypedDict):
    request: GenerateReviewRequest
    result: GenerateReviewEnvelope


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
    return _build_simple_task_graph(
        EvaluateAnswerState,
        runtime.evaluate_answer_task,
        EvaluateAnswerEnvelope,
    )


def _build_generate_review_graph(runtime: AgentRuntime):
    return _build_simple_task_graph(
        GenerateReviewState,
        runtime.generate_review_task,
        GenerateReviewEnvelope,
    )


def _to_envelope(task_result: TaskExecutionResult, envelope_cls):
    payload = {
        "result": task_result.result.model_dump(mode="json"),
        "raw_output": task_result.raw_output,
    }
    if task_result.side_effects:
        payload["side_effects"] = task_result.side_effects
    return envelope_cls.model_validate(payload)
