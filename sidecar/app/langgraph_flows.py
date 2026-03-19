from __future__ import annotations

from typing import TypedDict

from langgraph.graph import END, START, StateGraph

from app.config import Settings
from app.heuristics import analyze_repo, evaluate_answer, generate_question, generate_review
from app.schemas import (
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
    return {
        "analyze_repo": build_analyze_repo_graph(settings),
        "generate_question": build_generate_question_graph(),
        "evaluate_answer": build_evaluate_answer_graph(),
        "generate_review": build_generate_review_graph(),
    }


def build_analyze_repo_graph(settings: Settings):
    def run(state: AnalyzeRepoState) -> AnalyzeRepoState:
        result = analyze_repo(state["request"], settings)
        return {"request": state["request"], "result": AnalyzeRepoResponse.model_validate(result)}

    graph = StateGraph(AnalyzeRepoState)
    graph.add_node("run", run)
    graph.add_edge(START, "run")
    graph.add_edge("run", END)
    return graph.compile()


def build_generate_question_graph():
    def run(state: GenerateQuestionState) -> GenerateQuestionState:
        result = generate_question(state["request"])
        return {
            "request": state["request"],
            "result": GenerateQuestionResponse.model_validate(result),
        }

    graph = StateGraph(GenerateQuestionState)
    graph.add_node("run", run)
    graph.add_edge(START, "run")
    graph.add_edge("run", END)
    return graph.compile()


def build_evaluate_answer_graph():
    def run(state: EvaluateAnswerState) -> EvaluateAnswerState:
        result = evaluate_answer(state["request"])
        return {"request": state["request"], "result": EvaluationResult.model_validate(result)}

    graph = StateGraph(EvaluateAnswerState)
    graph.add_node("run", run)
    graph.add_edge(START, "run")
    graph.add_edge("run", END)
    return graph.compile()


def build_generate_review_graph():
    def run(state: GenerateReviewState) -> GenerateReviewState:
        result = generate_review(state["request"])
        return {"request": state["request"], "result": ReviewCard.model_validate(result)}

    graph = StateGraph(GenerateReviewState)
    graph.add_node("run", run)
    graph.add_edge(START, "run")
    graph.add_edge("run", END)
    return graph.compile()
