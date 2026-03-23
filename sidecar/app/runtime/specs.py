from __future__ import annotations

from dataclasses import dataclass, field
from typing import Any

from pydantic import BaseModel

from app.adapters.go_client import GoBackendClient
from app.prompts.bundles import (
    AnalyzeJobTargetDraft,
    AnalyzeRepoDraft,
    analyze_job_target_prompt_bundle,
    analyze_repo_prompt_bundle,
    evaluate_prompt_bundle,
    question_prompt_bundle,
    review_prompt_bundle,
)
from app.runtime.validation import validate_evaluation_result, validate_review_result
from app.runtime_tools import (
    build_evaluate_answer_agent_tools,
    build_generate_question_agent_tools,
    build_generate_review_agent_tools,
    prepare_evaluate_answer_agent_tooling,
    prepare_generate_question_agent_tooling,
    prepare_generate_review_agent_tooling,
)
from app.schemas import (
    EvaluateAnswerRequest,
    EvaluationResult,
    GenerateQuestionRequest,
    GenerateQuestionResponse,
    GenerateReviewRequest,
    ReviewCard,
)


@dataclass(frozen=True)
class TaskSpec:
    response_model: type[BaseModel]
    system_prompt: str
    user_prompt: str
    tools: list[Any]
    fallback_tools: list[Any] | None = None
    result_validator: Any = None
    context_trace_details: list[dict[str, Any]] = field(default_factory=list)


def build_analyze_repo_spec(bundle) -> TaskSpec:
    system_prompt, user_prompt, tools = analyze_repo_prompt_bundle(bundle)
    return TaskSpec(
        response_model=AnalyzeRepoDraft,
        system_prompt=system_prompt,
        user_prompt=user_prompt,
        tools=tools,
    )


def build_analyze_job_target_spec(request) -> TaskSpec:
    system_prompt, user_prompt, tools = analyze_job_target_prompt_bundle(request)
    return TaskSpec(
        response_model=AnalyzeJobTargetDraft,
        system_prompt=system_prompt,
        user_prompt=user_prompt,
        tools=tools,
    )


def build_generate_question_spec(
    request: GenerateQuestionRequest,
    go_client: GoBackendClient,
) -> TaskSpec:
    system_prompt, user_prompt, prompt_tools = question_prompt_bundle(request)
    prepared = prepare_generate_question_agent_tooling(request)
    tools = prompt_tools + build_generate_question_agent_tools(
        request,
        backend_client=go_client,
        prepared=prepared,
    )
    return TaskSpec(
        response_model=GenerateQuestionResponse,
        system_prompt=system_prompt,
        user_prompt=user_prompt,
        tools=tools,
        context_trace_details=prepared.trace_details,
    )


def build_evaluate_answer_spec(
    request: EvaluateAnswerRequest,
    go_client: GoBackendClient,
) -> TaskSpec:
    system_prompt, user_prompt, prompt_tools = evaluate_prompt_bundle(request)
    prepared = prepare_evaluate_answer_agent_tooling(request)
    tools = prompt_tools + build_evaluate_answer_agent_tools(
        request,
        {},
        backend_client=go_client,
        prepared=prepared,
    )
    return TaskSpec(
        response_model=EvaluationResult,
        system_prompt=system_prompt,
        user_prompt=user_prompt,
        tools=tools,
        result_validator=lambda result, side_effects, command_results: validate_evaluation_result(
            request,
            result,
            side_effects,
            command_results,
        ),
        context_trace_details=prepared.trace_details,
    )


def build_generate_review_spec(
    request: GenerateReviewRequest,
    go_client: GoBackendClient,
) -> TaskSpec:
    system_prompt, user_prompt, prompt_tools = review_prompt_bundle(request)
    prepared = prepare_generate_review_agent_tooling(request)
    tools = prompt_tools + build_generate_review_agent_tools(
        request,
        {},
        backend_client=go_client,
        prepared=prepared,
    )
    return TaskSpec(
        response_model=ReviewCard,
        system_prompt=system_prompt,
        user_prompt=user_prompt,
        tools=tools,
        result_validator=validate_review_result,
        context_trace_details=prepared.trace_details,
    )
