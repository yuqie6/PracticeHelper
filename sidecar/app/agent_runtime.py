from __future__ import annotations

import json
import logging
import time
from collections.abc import Iterator
from typing import Any

from pydantic import BaseModel

from app.config import Settings
from app.llm_client import ModelClientError, OpenAICompatibleModelClient
from app.repo_context import collect_repo_analysis_bundle
from app.runtime_prompts import (
    AnalyzeJobTargetDraft,
    AnalyzeRepoDraft,
    analyze_job_target_prompt_bundle,
    analyze_repo_prompt_bundle,
    evaluate_prompt_bundle,
    question_prompt_bundle,
    review_prompt_bundle,
)
from app.runtime_support import (
    RuntimeTool,
    parse_tool_arguments,
    tool_summary,
    validate_json_response,
)
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


class AgentRuntime:
    def __init__(
        self,
        settings: Settings,
        *,
        model_client: OpenAICompatibleModelClient | None = None,
    ) -> None:
        self._settings = settings
        self._model_client = model_client
        if self._model_client is None and settings.llm_enabled:
            self._model_client = OpenAICompatibleModelClient(settings)

    def analyze_repo(self, request: AnalyzeRepoRequest) -> AnalyzeRepoResponse:
        started_at = time.perf_counter()
        logger.info("analyze_repo started repo_url=%s", request.repo_url)
        self._require_model_client()
        bundle = collect_repo_analysis_bundle(request, self._settings)
        system_prompt, user_prompt, tools = analyze_repo_prompt_bundle(bundle)
        draft = self._run_task(
            response_model=AnalyzeRepoDraft,
            system_prompt=system_prompt,
            user_prompt=user_prompt,
            tools=tools,
        )
        response = AnalyzeRepoResponse(
            repo_url=bundle.repo_url,
            name=bundle.name,
            default_branch=bundle.default_branch,
            import_commit=bundle.import_commit,
            summary=draft.summary,
            tech_stack=bundle.tech_stack,
            highlights=draft.highlights,
            challenges=draft.challenges,
            tradeoffs=draft.tradeoffs,
            ownership_points=draft.ownership_points,
            followup_points=draft.followup_points,
            chunks=bundle.chunks,
        )
        logger.info(
            "analyze_repo completed repo_url=%s duration_ms=%.2f",
            request.repo_url,
            (time.perf_counter() - started_at) * 1000,
        )
        return response

    def analyze_job_target(self, request: AnalyzeJobTargetRequest) -> AnalyzeJobTargetResponse:
        started_at = time.perf_counter()
        logger.info("analyze_job_target started title=%s", request.title)
        self._require_model_client()
        system_prompt, user_prompt, tools = analyze_job_target_prompt_bundle(request)
        draft = self._run_task(
            response_model=AnalyzeJobTargetDraft,
            system_prompt=system_prompt,
            user_prompt=user_prompt,
            tools=tools,
        )
        response = AnalyzeJobTargetResponse(
            summary=draft.summary,
            must_have_skills=draft.must_have_skills,
            bonus_skills=draft.bonus_skills,
            responsibilities=draft.responsibilities,
            evaluation_focus=draft.evaluation_focus,
        )
        logger.info(
            "analyze_job_target completed title=%s duration_ms=%.2f",
            request.title,
            (time.perf_counter() - started_at) * 1000,
        )
        return response

    def generate_question(self, request: GenerateQuestionRequest) -> GenerateQuestionResponse:
        started_at = time.perf_counter()
        logger.info("generate_question started mode=%s topic=%s", request.mode, request.topic)
        system_prompt, user_prompt, tools = question_prompt_bundle(request)
        response = self._run_task(
            response_model=GenerateQuestionResponse,
            system_prompt=system_prompt,
            user_prompt=user_prompt,
            tools=tools,
        )
        logger.info(
            "generate_question completed mode=%s topic=%s duration_ms=%.2f",
            request.mode,
            request.topic,
            (time.perf_counter() - started_at) * 1000,
        )
        return response

    def stream_generate_question(
        self, request: GenerateQuestionRequest
    ) -> Iterator[dict[str, Any]]:
        system_prompt, user_prompt, tools = question_prompt_bundle(request)
        yield from self._stream_single_shot_task(
            response_model=GenerateQuestionResponse,
            system_prompt=system_prompt,
            user_prompt=user_prompt,
            tools=tools,
        )

    def evaluate_answer(self, request: EvaluateAnswerRequest) -> EvaluationResult:
        started_at = time.perf_counter()
        logger.info(
            "evaluate_answer started mode=%s topic=%s turn=%d/%d",
            request.mode,
            request.topic,
            request.turn_index,
            request.max_turns,
        )
        system_prompt, user_prompt, tools = evaluate_prompt_bundle(request)
        response = self._run_task(
            response_model=EvaluationResult,
            system_prompt=system_prompt,
            user_prompt=user_prompt,
            tools=tools,
        )
        logger.info(
            "evaluate_answer completed mode=%s topic=%s turn=%d/%d duration_ms=%.2f",
            request.mode,
            request.topic,
            request.turn_index,
            request.max_turns,
            (time.perf_counter() - started_at) * 1000,
        )
        return response

    def stream_evaluate_answer(self, request: EvaluateAnswerRequest) -> Iterator[dict[str, Any]]:
        system_prompt, user_prompt, tools = evaluate_prompt_bundle(request)
        yield from self._stream_single_shot_task(
            response_model=EvaluationResult,
            system_prompt=system_prompt,
            user_prompt=user_prompt,
            tools=tools,
        )

    def generate_review(self, request: GenerateReviewRequest) -> ReviewCard:
        started_at = time.perf_counter()
        logger.info("generate_review started session_id=%s", request.session.id)
        system_prompt, user_prompt, tools = review_prompt_bundle(request)
        response = self._run_task(
            response_model=ReviewCard,
            system_prompt=system_prompt,
            user_prompt=user_prompt,
            tools=tools,
        )
        logger.info(
            "generate_review completed session_id=%s duration_ms=%.2f",
            request.session.id,
            (time.perf_counter() - started_at) * 1000,
        )
        return response

    def stream_generate_review(self, request: GenerateReviewRequest) -> Iterator[dict[str, Any]]:
        system_prompt, user_prompt, tools = review_prompt_bundle(request)
        yield from self._stream_single_shot_task(
            response_model=ReviewCard,
            system_prompt=system_prompt,
            user_prompt=user_prompt,
            tools=tools,
        )

    def _run_task(
        self,
        *,
        response_model: type[BaseModel],
        system_prompt: str,
        user_prompt: str,
        tools: list[RuntimeTool],
    ) -> BaseModel:
        self._require_model_client()

        try:
            return self._run_tool_loop(
                response_model=response_model,
                system_prompt=system_prompt,
                user_prompt=user_prompt,
                tools=tools,
            )
        except (ModelClientError, ValueError, json.JSONDecodeError) as exc:
            logger.warning("agent tool loop failed, retrying single-shot: %s", exc)

        try:
            return self._run_single_shot(
                response_model=response_model,
                system_prompt=system_prompt,
                user_prompt=user_prompt,
                tools=tools,
            )
        except (ModelClientError, ValueError, json.JSONDecodeError) as exc:
            logger.warning("agent single-shot failed with no heuristic fallback: %s", exc)
            raise

    def _run_tool_loop(
        self,
        *,
        response_model: type[BaseModel],
        system_prompt: str,
        user_prompt: str,
        tools: list[RuntimeTool],
    ) -> BaseModel:
        model_client = self._require_model_client()

        messages: list[dict[str, Any]] = [
            {"role": "system", "content": system_prompt},
            {"role": "user", "content": user_prompt},
        ]
        tool_map = {tool.name: tool for tool in tools}
        used_any_tool = False

        for _ in range(4):
            result = model_client.create_completion(
                messages=messages,
                tools=[tool.spec() for tool in tools],
            )
            if result.tool_calls:
                messages.append(
                    {
                        "role": "assistant",
                        "content": result.content,
                        "tool_calls": result.tool_calls,
                    }
                )
                for tool_call in result.tool_calls:
                    tool_name = tool_call.get("function", {}).get("name", "")
                    tool_call_id = tool_call.get("id", tool_name)
                    arguments = parse_tool_arguments(tool_call)
                    tool = tool_map.get(tool_name)
                    if tool is None:
                        tool_result = {"error": f"unknown tool: {tool_name}"}
                    else:
                        tool_result = tool.handler(arguments)
                    messages.append(
                        {
                            "role": "tool",
                            "tool_call_id": tool_call_id,
                            "name": tool_name,
                            "content": json.dumps(tool_result, ensure_ascii=False),
                        }
                    )
                    used_any_tool = True
                continue

            if result.content.strip():
                if tools and not used_any_tool:
                    raise ModelClientError(
                        "model returned a final answer before reading any required tool context"
                    )
                return validate_json_response(result.content, response_model)

            raise ModelClientError("model returned neither content nor tool calls")

        raise ModelClientError("model exhausted tool loop without producing a final answer")

    def _run_single_shot(
        self,
        *,
        response_model: type[BaseModel],
        system_prompt: str,
        user_prompt: str,
        tools: list[RuntimeTool],
    ) -> BaseModel:
        model_client = self._require_model_client()

        context_dump = {tool.name: tool.handler({}) for tool in tools}
        messages = [
            {"role": "system", "content": system_prompt},
            {
                "role": "user",
                "content": (
                    f"{user_prompt}\n\n"
                    "下面是你已经可以直接使用的上下文，请在此基础上直接输出最终 JSON：\n"
                    f"{json.dumps(context_dump, ensure_ascii=False, indent=2)}"
                ),
            },
        ]
        result = model_client.create_completion(messages=messages)
        if not result.content.strip():
            raise ModelClientError("model returned empty content in single-shot mode")
        return validate_json_response(result.content, response_model)

    def _stream_single_shot_task(
        self,
        *,
        response_model: type[BaseModel],
        system_prompt: str,
        user_prompt: str,
        tools: list[RuntimeTool],
    ) -> Iterator[dict[str, Any]]:
        model_client = self._require_model_client()

        yield {"type": "phase", "phase": "prepare_context"}
        context_dump: dict[str, Any] = {}
        for tool in tools:
            context_dump[tool.name] = tool.handler({})
            yield {"type": "context", "name": tool.name}
            summary = tool_summary(tool.name)
            if summary:
                yield {"type": "reasoning", "text": summary}

        messages = [
            {"role": "system", "content": system_prompt},
            {
                "role": "user",
                "content": (
                    f"{user_prompt}\n\n"
                    "下面是你已经可以直接使用的上下文，请在此基础上直接输出最终 JSON：\n"
                    f"{json.dumps(context_dump, ensure_ascii=False, indent=2)}"
                ),
            },
        ]

        yield {"type": "phase", "phase": "call_model"}
        chunks: list[str] = []
        try:
            for chunk in model_client.create_completion_stream(messages=messages):
                if chunk.reasoning:
                    yield {"type": "reasoning", "text": chunk.reasoning}
                if chunk.content:
                    chunks.append(chunk.content)
                    yield {"type": "content", "text": chunk.content}
        except ModelClientError:
            result = model_client.create_completion(messages=messages)
            if result.content:
                chunks.append(result.content)
                yield {"type": "content", "text": result.content}

        yield {"type": "phase", "phase": "parse_result"}
        text = "".join(chunks).strip()
        if not text:
            raise ModelClientError("model returned empty content in streaming mode")

        result_model = validate_json_response(text, response_model)
        yield {"type": "result", "data": result_model.model_dump(mode="json")}

    def _require_model_client(self) -> OpenAICompatibleModelClient:
        if self._model_client is None:
            raise ModelClientError(
                "LLM is required for sidecar core flows. Configure PRACTICEHELPER_SIDECAR_MODEL, "
                "PRACTICEHELPER_SIDECAR_OPENAI_BASE_URL, and PRACTICEHELPER_SIDECAR_OPENAI_API_KEY."
            )
        return self._model_client
