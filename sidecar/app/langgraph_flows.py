from __future__ import annotations

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


class AnalyzeRepoState(TypedDict):
    request: AnalyzeRepoRequest
    result: AnalyzeRepoResponse


class AnalyzeJobTargetState(TypedDict):
    request: AnalyzeJobTargetRequest
    result: AnalyzeJobTargetResponse


class GenerateQuestionState(TypedDict):
    request: GenerateQuestionRequest
    result: GenerateQuestionResponse


class EvaluateAnswerState(TypedDict):
    request: EvaluateAnswerRequest
    result: EvaluationResult


class GenerateReviewState(TypedDict):
    request: GenerateReviewRequest
    result: ReviewCard


def build_flows(settings: Settings) -> dict[str, object]:
    runtime = AgentRuntime(settings)
    # 当前每条 flow 只有一个 run 节点，但这里仍统一包成 LangGraph，
    # 是为了先把 FastAPI 入口固定成 graph.invoke 边界，后续再按需插入检索、审查或缓存节点。
    return {
        "analyze_repo": build_analyze_repo_graph(runtime),
        "analyze_job_target": build_analyze_job_target_graph(runtime),
        "generate_question": build_generate_question_graph(runtime),
        "evaluate_answer": build_evaluate_answer_graph(runtime),
        "generate_review": build_generate_review_graph(runtime),
    }


def build_analyze_repo_graph(runtime: AgentRuntime):
    def run(state: AnalyzeRepoState) -> AnalyzeRepoState:
        result = runtime.analyze_repo(state["request"])
        return {"request": state["request"], "result": AnalyzeRepoResponse.model_validate(result)}

    graph = StateGraph(AnalyzeRepoState)
    graph.add_node("run", run)
    graph.add_edge(START, "run")
    graph.add_edge("run", END)
    return graph.compile()


def build_analyze_job_target_graph(runtime: AgentRuntime):
    def run(state: AnalyzeJobTargetState) -> AnalyzeJobTargetState:
        result = runtime.analyze_job_target(state["request"])
        return {
            "request": state["request"],
            "result": AnalyzeJobTargetResponse.model_validate(result),
        }

    graph = StateGraph(AnalyzeJobTargetState)
    graph.add_node("run", run)
    graph.add_edge(START, "run")
    graph.add_edge("run", END)
    return graph.compile()


def build_generate_question_graph(runtime: AgentRuntime):
    def run(state: GenerateQuestionState) -> GenerateQuestionState:
        result = runtime.generate_question(state["request"])
        return {
            "request": state["request"],
            "result": GenerateQuestionResponse.model_validate(result),
        }

    graph = StateGraph(GenerateQuestionState)
    graph.add_node("run", run)
    graph.add_edge(START, "run")
    graph.add_edge("run", END)
    return graph.compile()


def build_evaluate_answer_graph(runtime: AgentRuntime):
    def run(state: EvaluateAnswerState) -> EvaluateAnswerState:
        result = runtime.evaluate_answer(state["request"])
        return {"request": state["request"], "result": EvaluationResult.model_validate(result)}

    graph = StateGraph(EvaluateAnswerState)
    graph.add_node("run", run)
    graph.add_edge(START, "run")
    graph.add_edge("run", END)
    return graph.compile()


def build_generate_review_graph(runtime: AgentRuntime):
    def run(state: GenerateReviewState) -> GenerateReviewState:
        result = runtime.generate_review(state["request"])
        return {"request": state["request"], "result": ReviewCard.model_validate(result)}

    graph = StateGraph(GenerateReviewState)
    graph.add_node("run", run)
    graph.add_edge(START, "run")
    graph.add_edge("run", END)
    return graph.compile()
