from __future__ import annotations

import json
import logging
import time
from collections.abc import Iterator
from dataclasses import dataclass, field
from typing import Any

from pydantic import BaseModel

from app.agent_tools import (
    build_evaluate_answer_agent_tools,
    build_generate_question_agent_tools,
    build_generate_review_agent_tools,
    is_action_tool,
    make_record_observation_tool,
    make_set_depth_signal_tool,
    make_suggest_next_session_tool,
    make_update_knowledge_tool,
)
from app.config import Settings
from app.go_client import GoBackendClient
from app.llm_client import ModelClientError, OpenAICompatibleModelClient
from app.repo_context import RepoAnalysisBundle, collect_repo_analysis_bundle
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


@dataclass(frozen=True)
class TaskExecutionResult[ResultModelT: BaseModel]:
    result: ResultModelT
    raw_output: str = ""
    side_effects: dict[str, Any] = field(default_factory=dict)


class AgentRuntime:
    def __init__(
        self,
        settings: Settings,
        *,
        model_client: OpenAICompatibleModelClient | None = None,
        go_client: GoBackendClient | None = None,
    ) -> None:
        self._settings = settings
        self._model_client = model_client
        self._go_client = go_client or GoBackendClient(settings)
        if self._model_client is None and settings.llm_enabled:
            self._model_client = OpenAICompatibleModelClient(settings)

    def collect_repo_bundle(self, request: AnalyzeRepoRequest) -> RepoAnalysisBundle:
        self._require_model_client()
        return collect_repo_analysis_bundle(request, self._settings)

    def summarize_repo_bundle(
        self,
        bundle: RepoAnalysisBundle,
    ) -> TaskExecutionResult[AnalyzeRepoResponse]:
        started_at = time.perf_counter()
        logger.info("analyze_repo summarize started repo_url=%s", bundle.repo_url)
        system_prompt, user_prompt, tools = analyze_repo_prompt_bundle(bundle)
        draft_result = self._run_task(
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
            summary=draft_result.result.summary,
            tech_stack=bundle.tech_stack,
            highlights=draft_result.result.highlights,
            challenges=draft_result.result.challenges,
            tradeoffs=draft_result.result.tradeoffs,
            ownership_points=draft_result.result.ownership_points,
            followup_points=draft_result.result.followup_points,
            chunks=bundle.chunks,
        )
        logger.info(
            "analyze_repo summarize completed repo_url=%s duration_ms=%.2f",
            bundle.repo_url,
            (time.perf_counter() - started_at) * 1000,
        )
        return TaskExecutionResult(result=response, raw_output=draft_result.raw_output)

    def analyze_repo(self, request: AnalyzeRepoRequest) -> AnalyzeRepoResponse:
        bundle = self.collect_repo_bundle(request)
        return self.summarize_repo_bundle(bundle).result

    def analyze_job_target_task(
        self,
        request: AnalyzeJobTargetRequest,
    ) -> TaskExecutionResult[AnalyzeJobTargetResponse]:
        started_at = time.perf_counter()
        logger.info("analyze_job_target started title=%s", request.title)
        self._require_model_client()
        system_prompt, user_prompt, tools = analyze_job_target_prompt_bundle(request)
        draft_result = self._run_task(
            response_model=AnalyzeJobTargetDraft,
            system_prompt=system_prompt,
            user_prompt=user_prompt,
            tools=tools,
        )
        response = AnalyzeJobTargetResponse(
            summary=draft_result.result.summary,
            must_have_skills=draft_result.result.must_have_skills,
            bonus_skills=draft_result.result.bonus_skills,
            responsibilities=draft_result.result.responsibilities,
            evaluation_focus=draft_result.result.evaluation_focus,
        )
        logger.info(
            "analyze_job_target completed title=%s duration_ms=%.2f",
            request.title,
            (time.perf_counter() - started_at) * 1000,
        )
        return TaskExecutionResult(result=response, raw_output=draft_result.raw_output)

    def analyze_job_target(self, request: AnalyzeJobTargetRequest) -> AnalyzeJobTargetResponse:
        return self.analyze_job_target_task(request).result

    def generate_question_task(
        self,
        request: GenerateQuestionRequest,
    ) -> TaskExecutionResult[GenerateQuestionResponse]:
        started_at = time.perf_counter()
        logger.info("generate_question started mode=%s topic=%s", request.mode, request.topic)
        system_prompt, user_prompt, prompt_tools = question_prompt_bundle(request)
        tools = prompt_tools + build_generate_question_agent_tools(
            request, backend_client=self._go_client
        )
        response = self._run_task(
            response_model=GenerateQuestionResponse,
            system_prompt=system_prompt,
            user_prompt=user_prompt,
            tools=tools,
            fallback_tools=_read_only_tools(tools),
        )
        logger.info(
            "generate_question completed mode=%s topic=%s duration_ms=%.2f",
            request.mode,
            request.topic,
            (time.perf_counter() - started_at) * 1000,
        )
        return response

    def generate_question(self, request: GenerateQuestionRequest) -> GenerateQuestionResponse:
        return self.generate_question_task(request).result

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

    def evaluate_answer_task(
        self,
        request: EvaluateAnswerRequest,
    ) -> TaskExecutionResult[EvaluationResult]:
        started_at = time.perf_counter()
        logger.info(
            "evaluate_answer started mode=%s topic=%s turn=%d/%d",
            request.mode,
            request.topic,
            request.turn_index,
            request.max_turns,
        )
        system_prompt, user_prompt, prompt_tools = evaluate_prompt_bundle(request)
        tools = prompt_tools + build_evaluate_answer_agent_tools(
            request, {}, backend_client=self._go_client
        )
        response = self._run_task(
            response_model=EvaluationResult,
            system_prompt=system_prompt,
            user_prompt=user_prompt,
            tools=tools,
            fallback_tools=_read_only_tools(tools),
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

    def evaluate_answer(self, request: EvaluateAnswerRequest) -> EvaluationResult:
        return self.evaluate_answer_task(request).result

    def stream_evaluate_answer(self, request: EvaluateAnswerRequest) -> Iterator[dict[str, Any]]:
        system_prompt, user_prompt, tools = evaluate_prompt_bundle(request)
        yield from self._stream_single_shot_task(
            response_model=EvaluationResult,
            system_prompt=system_prompt,
            user_prompt=user_prompt,
            tools=tools,
        )

    def generate_review_task(
        self,
        request: GenerateReviewRequest,
    ) -> TaskExecutionResult[ReviewCard]:
        started_at = time.perf_counter()
        logger.info("generate_review started session_id=%s", request.session.id)
        system_prompt, user_prompt, prompt_tools = review_prompt_bundle(request)
        tools = prompt_tools + build_generate_review_agent_tools(
            request, {}, backend_client=self._go_client
        )
        response = self._run_task(
            response_model=ReviewCard,
            system_prompt=system_prompt,
            user_prompt=user_prompt,
            tools=tools,
            fallback_tools=_read_only_tools(tools),
        )
        logger.info(
            "generate_review completed session_id=%s duration_ms=%.2f",
            request.session.id,
            (time.perf_counter() - started_at) * 1000,
        )
        return response

    def generate_review(self, request: GenerateReviewRequest) -> ReviewCard:
        return self.generate_review_task(request).result

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
        fallback_tools: list[RuntimeTool] | None = None,
    ) -> TaskExecutionResult[BaseModel]:
        self._require_model_client()
        fallback_tools = fallback_tools or _read_only_tools(tools)

        try:
            # 先走非流式 agent loop；只有 tool call 或 JSON 收口不稳定时才退回单轮生成。
            return self._run_agent_loop(
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
                tools=fallback_tools,
            )
        except (ModelClientError, ValueError, json.JSONDecodeError) as exc:
            logger.warning("agent single-shot failed with no heuristic fallback: %s", exc)
            raise

    def _run_agent_loop(
        self,
        *,
        response_model: type[BaseModel],
        system_prompt: str,
        user_prompt: str,
        tools: list[RuntimeTool],
    ) -> TaskExecutionResult[BaseModel]:
        model_client = self._require_model_client()

        messages: list[dict[str, Any]] = [
            {"role": "system", "content": system_prompt},
            {"role": "user", "content": user_prompt},
        ]
        tool_map = {tool.name: tool for tool in tools}
        used_any_tool = False
        side_effects: dict[str, Any] = {}

        # 轮数上限用于兜住 prompt/tool 契约失配，避免 sidecar 卡在无限工具往返里。
        for _ in range(8):
            completion = model_client.create_completion(
                messages=messages,
                tools=[tool.spec() for tool in tools],
            )
            if completion.tool_calls:
                messages.append(
                    {
                        "role": "assistant",
                        "content": completion.content,
                        "tool_calls": completion.tool_calls,
                    }
                )
                for tool_call in completion.tool_calls:
                    tool_name = tool_call.get("function", {}).get("name", "")
                    tool_call_id = tool_call.get("id", tool_name)
                    arguments = parse_tool_arguments(tool_call)
                    tool = tool_map.get(tool_name)
                    if tool is None:
                        tool_result = {"error": f"unknown tool: {tool_name}"}
                    else:
                        if is_action_tool(tool_name):
                            arguments = dict(arguments)
                        if tool_name in {
                            "record_observation",
                            "update_knowledge",
                            "suggest_next_session",
                            "set_depth_signal",
                        }:
                            # 行动工具通过闭包把副作用暂存在 side_effects，不直接写 DB。
                            tool = _rebind_action_tool(tool, side_effects)
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

            if completion.content.strip():
                if tools and not used_any_tool:
                    raise ModelClientError(
                        "model returned a final answer before reading any required tool context"
                    )
                result_model = validate_json_response(completion.content, response_model)
                return TaskExecutionResult(
                    result=result_model,
                    raw_output=completion.content.strip(),
                    side_effects=side_effects,
                )

            raise ModelClientError("model returned neither content nor tool calls")

        raise ModelClientError("model exhausted tool loop without producing a final answer")

    def _run_single_shot(
        self,
        *,
        response_model: type[BaseModel],
        system_prompt: str,
        user_prompt: str,
        tools: list[RuntimeTool],
    ) -> TaskExecutionResult[BaseModel]:
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
        completion = model_client.create_completion(messages=messages)
        if not completion.content.strip():
            raise ModelClientError("model returned empty content in single-shot mode")
        result_model = validate_json_response(completion.content, response_model)
        return TaskExecutionResult(
            result=result_model,
            raw_output=completion.content.strip(),
        )

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
            completion = model_client.create_completion(messages=messages)
            if completion.content:
                chunks.append(completion.content)
                yield {"type": "content", "text": completion.content}

        yield {"type": "phase", "phase": "parse_result"}
        raw_output = "".join(chunks).strip()
        if not raw_output:
            raise ModelClientError("model returned empty content in streaming mode")

        result_model = validate_json_response(raw_output, response_model)
        yield {
            "type": "result",
            "data": {
                "result": result_model.model_dump(mode="json"),
                "raw_output": raw_output,
            },
        }

    def _require_model_client(self) -> OpenAICompatibleModelClient:
        if self._model_client is None:
            raise ModelClientError(
                "LLM is required for sidecar core flows. Configure PRACTICEHELPER_SIDECAR_MODEL, "
                "PRACTICEHELPER_SIDECAR_OPENAI_BASE_URL, and PRACTICEHELPER_SIDECAR_OPENAI_API_KEY."
            )
        return self._model_client


def _read_only_tools(tools: list[RuntimeTool]) -> list[RuntimeTool]:
    return [tool for tool in tools if not is_action_tool(tool.name)]


def _rebind_action_tool(tool: RuntimeTool, side_effects: dict[str, Any]) -> RuntimeTool:
    if tool.name == "record_observation":
        rebound = make_record_observation_tool(side_effects)
    elif tool.name == "update_knowledge":
        rebound = make_update_knowledge_tool(side_effects)
    elif tool.name == "suggest_next_session":
        rebound = make_suggest_next_session_tool(side_effects)
    elif tool.name == "set_depth_signal":
        rebound = make_set_depth_signal_tool(side_effects)
    else:
        return tool
    return rebound
