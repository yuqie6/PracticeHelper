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
    RuntimeTrace,
    RuntimeTraceEntry,
)

logger = logging.getLogger(__name__)


@dataclass(frozen=True)
class TaskExecutionResult[ResultModelT: BaseModel]:
    result: ResultModelT
    raw_output: str = ""
    side_effects: dict[str, Any] = field(default_factory=dict)
    trace: RuntimeTrace = field(default_factory=RuntimeTrace)


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
            task_name="analyze_repo",
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
        return TaskExecutionResult(
            result=response,
            raw_output=draft_result.raw_output,
            trace=draft_result.trace,
        )

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
            task_name="analyze_job_target",
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
        return TaskExecutionResult(
            result=response,
            raw_output=draft_result.raw_output,
            trace=draft_result.trace,
        )

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
            task_name="generate_question",
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
        system_prompt, user_prompt, prompt_tools = question_prompt_bundle(request)
        tools = prompt_tools + build_generate_question_agent_tools(
            request, backend_client=self._go_client
        )
        yield from self._stream_task(
            task_name="generate_question",
            response_model=GenerateQuestionResponse,
            system_prompt=system_prompt,
            user_prompt=user_prompt,
            tools=tools,
            fallback_tools=_read_only_tools(tools),
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
            task_name="evaluate_answer",
            response_model=EvaluationResult,
            system_prompt=system_prompt,
            user_prompt=user_prompt,
            tools=tools,
            fallback_tools=_read_only_tools(tools),
            result_validator=lambda result, side_effects: _validate_evaluation_result(
                request, result, side_effects
            ),
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
        system_prompt, user_prompt, prompt_tools = evaluate_prompt_bundle(request)
        tools = prompt_tools + build_evaluate_answer_agent_tools(
            request, {}, backend_client=self._go_client
        )
        yield from self._stream_task(
            task_name="evaluate_answer",
            response_model=EvaluationResult,
            system_prompt=system_prompt,
            user_prompt=user_prompt,
            tools=tools,
            fallback_tools=_read_only_tools(tools),
            result_validator=lambda result, side_effects: _validate_evaluation_result(
                request, result, side_effects
            ),
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
            task_name="generate_review",
            response_model=ReviewCard,
            system_prompt=system_prompt,
            user_prompt=user_prompt,
            tools=tools,
            fallback_tools=_read_only_tools(tools),
            result_validator=_validate_review_result,
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
        system_prompt, user_prompt, prompt_tools = review_prompt_bundle(request)
        tools = prompt_tools + build_generate_review_agent_tools(
            request, {}, backend_client=self._go_client
        )
        yield from self._stream_task(
            task_name="generate_review",
            response_model=ReviewCard,
            system_prompt=system_prompt,
            user_prompt=user_prompt,
            tools=tools,
            fallback_tools=_read_only_tools(tools),
            result_validator=_validate_review_result,
        )

    def _run_task(
        self,
        *,
        task_name: str,
        response_model: type[BaseModel],
        system_prompt: str,
        user_prompt: str,
        tools: list[RuntimeTool],
        fallback_tools: list[RuntimeTool] | None = None,
        result_validator: Any = None,
    ) -> TaskExecutionResult[BaseModel]:
        self._require_model_client()
        fallback_tools = fallback_tools or _read_only_tools(tools)
        trace = RuntimeTrace()

        try:
            # 先走非流式 agent loop；只有 tool call 或 JSON 收口不稳定时才退回单轮生成。
            return self._run_agent_loop(
                task_name=task_name,
                response_model=response_model,
                system_prompt=system_prompt,
                user_prompt=user_prompt,
                tools=tools,
                result_validator=result_validator,
                trace=trace,
            )
        except (ModelClientError, ValueError, json.JSONDecodeError) as exc:
            logger.warning("agent tool loop failed, retrying single-shot: %s", exc)
            _append_trace(
                trace,
                flow=task_name,
                phase="fallback",
                status="fallback",
                code=_runtime_error_code(exc, fallback="single_shot_failed"),
                message="工具循环没有稳定收口，退回单轮生成。",
            )

        try:
            return self._run_single_shot(
                task_name=task_name,
                response_model=response_model,
                system_prompt=system_prompt,
                user_prompt=user_prompt,
                tools=fallback_tools,
                result_validator=result_validator,
                trace=trace,
            )
        except (ModelClientError, ValueError, json.JSONDecodeError) as exc:
            logger.warning("agent single-shot failed with no heuristic fallback: %s", exc)
            _append_trace(
                trace,
                flow=task_name,
                phase="error",
                status="error",
                code=_runtime_error_code(exc, fallback="single_shot_failed"),
                message=str(exc),
            )
            raise

    def _stream_task(
        self,
        *,
        task_name: str,
        response_model: type[BaseModel],
        system_prompt: str,
        user_prompt: str,
        tools: list[RuntimeTool],
        fallback_tools: list[RuntimeTool] | None = None,
        result_validator: Any = None,
    ) -> Iterator[dict[str, Any]]:
        self._require_model_client()
        fallback_tools = fallback_tools or _read_only_tools(tools)
        trace = RuntimeTrace()

        try:
            yield from self._stream_agent_loop(
                task_name=task_name,
                response_model=response_model,
                system_prompt=system_prompt,
                user_prompt=user_prompt,
                tools=tools,
                result_validator=result_validator,
                trace=trace,
            )
            return
        except (ModelClientError, ValueError, json.JSONDecodeError) as exc:
            logger.warning("streaming agent tool loop failed, retrying single-shot: %s", exc)
            fallback_entry = _append_trace(
                trace,
                flow=task_name,
                phase="fallback",
                status="fallback",
                code=_runtime_error_code(exc, fallback="single_shot_failed"),
                message="工具循环没有稳定收口，正在退回单轮生成。",
            )
            yield {"type": "trace", "data": fallback_entry.model_dump(mode="json")}
            yield {
                "type": "reasoning",
                "text": "工具循环没有稳定收口，正在退回单轮生成以保证本次结果可用。",
            }

        try:
            yield from self._stream_single_shot_task(
                task_name=task_name,
                response_model=response_model,
                system_prompt=system_prompt,
                user_prompt=user_prompt,
                tools=fallback_tools,
                result_validator=result_validator,
                trace=trace,
            )
        except (ModelClientError, ValueError, json.JSONDecodeError) as exc:
            logger.warning("streaming agent single-shot failed with no heuristic fallback: %s", exc)
            error_entry = _append_trace(
                trace,
                flow=task_name,
                phase="error",
                status="error",
                code=_runtime_error_code(exc, fallback="single_shot_failed"),
                message=str(exc),
            )
            yield {"type": "trace", "data": error_entry.model_dump(mode="json")}
            raise

    def _run_agent_loop(
        self,
        *,
        task_name: str,
        response_model: type[BaseModel],
        system_prompt: str,
        user_prompt: str,
        tools: list[RuntimeTool],
        result_validator: Any = None,
        trace: RuntimeTrace,
    ) -> TaskExecutionResult[BaseModel]:
        model_client = self._require_model_client()

        messages: list[dict[str, Any]] = [
            {"role": "system", "content": system_prompt},
            {"role": "user", "content": user_prompt},
        ]
        tool_map = {tool.name: tool for tool in tools}
        used_any_tool = False
        side_effects: dict[str, Any] = {}
        validation_attempts = 0
        _append_trace(
            trace,
            flow=task_name,
            phase="prepare_context",
            status="info",
            code="runtime_started",
            message="开始执行 agent runtime。",
        )

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
                        error_entry = _append_trace(
                            trace,
                            flow=task_name,
                            phase="tool_call",
                            status="error",
                            code="tool_call_failed",
                            message=f"模型请求了未知工具：{tool_name}",
                            tool_name=tool_name,
                        )
                        raise ModelClientError(error_entry.message, code=error_entry.code)
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
                    try:
                        tool_result = tool.handler(arguments)
                    except Exception as exc:  # noqa: BLE001
                        code = (
                            "backend_callback_failed"
                            if tool_name in {"search_repo_chunks", "get_session_detail"}
                            else "tool_call_failed"
                        )
                        error_entry = _append_trace(
                            trace,
                            flow=task_name,
                            phase="tool_call",
                            status="error",
                            code=code,
                            message=f"工具 {tool_name} 执行失败：{exc}",
                            tool_name=tool_name,
                        )
                        raise ModelClientError(error_entry.message, code=error_entry.code) from exc
                    _append_trace(
                        trace,
                        flow=task_name,
                        phase="tool_call",
                        status="success",
                        code="tool_call_succeeded",
                        message=f"工具 {tool_name} 调用成功。",
                        tool_name=tool_name,
                    )
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
                    error_entry = _append_trace(
                        trace,
                        flow=task_name,
                        phase="validate",
                        status="error",
                        code="tool_context_missing",
                        message="模型在读取必要上下文前直接返回了最终答案。",
                    )
                    raise ModelClientError(error_entry.message, code=error_entry.code)

                raw_output = completion.content.strip()
                validation_error = ""
                validation_code = ""
                try:
                    result_model = validate_json_response(raw_output, response_model)
                except (ValueError, json.JSONDecodeError) as exc:
                    result_model = None
                    validation_error = str(exc)
                    validation_code = "json_parse_failed"
                else:
                    if result_validator is not None:
                        validation_error = result_validator(result_model, side_effects)
                        if validation_error:
                            validation_code = "semantic_validation_failed"

                if validation_error:
                    validation_attempts += 1
                    status = "retry" if validation_attempts < 2 else "error"
                    _append_trace(
                        trace,
                        flow=task_name,
                        phase="validate",
                        status=status,
                        code=validation_code or "semantic_validation_failed",
                        message=validation_error,
                        attempt=validation_attempts,
                    )
                    if validation_attempts >= 2:
                        raise ModelClientError(
                            f"{task_name} failed after validation retries: {validation_error}",
                            code=validation_code or "semantic_validation_failed",
                        )
                    messages.append({"role": "assistant", "content": raw_output})
                    messages.append(
                        {
                            "role": "user",
                            "content": (
                                "上一次输出没有过校验，请修正这些问题后重新生成："
                                f"{validation_error}"
                            ),
                        }
                    )
                    continue

                _append_trace(
                    trace,
                    flow=task_name,
                    phase="finalize",
                    status="success",
                    code="runtime_completed",
                    message="agent runtime 已稳定收口。",
                )
                return TaskExecutionResult(
                    result=result_model,
                    raw_output=raw_output,
                    side_effects=side_effects,
                    trace=trace,
                )

            error_entry = _append_trace(
                trace,
                flow=task_name,
                phase="error",
                status="error",
                code="tool_loop_exhausted",
                message="模型既没有返回内容，也没有发起工具调用。",
            )
            raise ModelClientError(error_entry.message, code=error_entry.code)

        error_entry = _append_trace(
            trace,
            flow=task_name,
            phase="error",
            status="error",
            code="tool_loop_exhausted",
            message="工具循环已到上限，仍未得到最终答案。",
        )
        raise ModelClientError(error_entry.message, code=error_entry.code)

    def _stream_agent_loop(
        self,
        *,
        task_name: str,
        response_model: type[BaseModel],
        system_prompt: str,
        user_prompt: str,
        tools: list[RuntimeTool],
        result_validator: Any = None,
        trace: RuntimeTrace,
    ) -> Iterator[dict[str, Any]]:
        model_client = self._require_model_client()

        messages: list[dict[str, Any]] = [
            {"role": "system", "content": system_prompt},
            {"role": "user", "content": user_prompt},
        ]
        tool_map = {tool.name: tool for tool in tools}
        used_any_tool = False
        side_effects: dict[str, Any] = {}
        validation_attempts = 0

        yield {"type": "phase", "phase": "prepare_context"}
        entry = _append_trace(
            trace,
            flow=task_name,
            phase="prepare_context",
            status="info",
            code="runtime_started",
            message="开始执行 agent runtime。",
        )
        yield {"type": "trace", "data": entry.model_dump(mode="json")}

        for _ in range(8):
            yield {"type": "phase", "phase": "call_model"}
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
                        error_entry = _append_trace(
                            trace,
                            flow=task_name,
                            phase="tool_call",
                            status="error",
                            code="tool_call_failed",
                            message=f"模型请求了未知工具：{tool_name}",
                            tool_name=tool_name,
                        )
                        yield {"type": "trace", "data": error_entry.model_dump(mode="json")}
                        raise ModelClientError(error_entry.message, code=error_entry.code)
                    if is_action_tool(tool_name):
                        arguments = dict(arguments)
                    if tool_name in {
                        "record_observation",
                        "update_knowledge",
                        "suggest_next_session",
                        "set_depth_signal",
                    }:
                        tool = _rebind_action_tool(tool, side_effects)
                    try:
                        tool_result = tool.handler(arguments)
                    except Exception as exc:  # noqa: BLE001
                        code = (
                            "backend_callback_failed"
                            if tool_name in {"search_repo_chunks", "get_session_detail"}
                            else "tool_call_failed"
                        )
                        error_entry = _append_trace(
                            trace,
                            flow=task_name,
                            phase="tool_call",
                            status="error",
                            code=code,
                            message=f"工具 {tool_name} 执行失败：{exc}",
                            tool_name=tool_name,
                        )
                        yield {"type": "trace", "data": error_entry.model_dump(mode="json")}
                        raise ModelClientError(error_entry.message, code=error_entry.code) from exc
                    trace_entry = _append_trace(
                        trace,
                        flow=task_name,
                        phase="tool_call",
                        status="success",
                        code="tool_call_succeeded",
                        message=f"工具 {tool_name} 调用成功。",
                        tool_name=tool_name,
                    )
                    messages.append(
                        {
                            "role": "tool",
                            "tool_call_id": tool_call_id,
                            "name": tool_name,
                            "content": json.dumps(tool_result, ensure_ascii=False),
                        }
                    )
                    used_any_tool = True
                    yield {"type": "context", "name": tool_name}
                    yield {"type": "trace", "data": trace_entry.model_dump(mode="json")}
                    summary = tool_summary(tool_name)
                    if summary:
                        yield {"type": "reasoning", "text": summary}
                continue

            if completion.content.strip():
                if not used_any_tool:
                    error_entry = _append_trace(
                        trace,
                        flow=task_name,
                        phase="validate",
                        status="error",
                        code="tool_context_missing",
                        message="模型在读取必要上下文前直接返回了最终答案。",
                    )
                    yield {"type": "trace", "data": error_entry.model_dump(mode="json")}
                    raise ModelClientError(error_entry.message, code=error_entry.code)

                raw_output = completion.content.strip()
                validation_error = ""
                validation_code = ""
                yield {"type": "phase", "phase": "parse_result"}
                try:
                    result_model = validate_json_response(raw_output, response_model)
                except (ValueError, json.JSONDecodeError) as exc:
                    result_model = None
                    validation_error = str(exc)
                    validation_code = "json_parse_failed"
                else:
                    if result_validator is not None:
                        validation_error = result_validator(result_model, side_effects)
                        if validation_error:
                            validation_code = "semantic_validation_failed"

                if validation_error:
                    validation_attempts += 1
                    status = "retry" if validation_attempts < 2 else "error"
                    entry = _append_trace(
                        trace,
                        flow=task_name,
                        phase="validate",
                        status=status,
                        code=validation_code or "semantic_validation_failed",
                        message=validation_error,
                        attempt=validation_attempts,
                    )
                    yield {"type": "trace", "data": entry.model_dump(mode="json")}
                    if validation_attempts >= 2:
                        raise ModelClientError(
                            f"{task_name} failed after validation retries: {validation_error}",
                            code=validation_code or "semantic_validation_failed",
                        )
                    messages.append({"role": "assistant", "content": raw_output})
                    messages.append(
                        {
                            "role": "user",
                            "content": (
                                "上一次输出没有过校验，请修正这些问题后重新生成："
                                f"{validation_error}"
                            ),
                        }
                    )
                    yield {
                        "type": "reasoning",
                        "text": f"输出没有过校验，正在按约束重试：{validation_error}",
                    }
                    continue

                yield {"type": "content", "text": raw_output}
                entry = _append_trace(
                    trace,
                    flow=task_name,
                    phase="finalize",
                    status="success",
                    code="runtime_completed",
                    message="agent runtime 已稳定收口。",
                )
                yield {"type": "trace", "data": entry.model_dump(mode="json")}
                payload = {
                    "result": result_model.model_dump(mode="json"),
                    "raw_output": raw_output,
                    "trace": trace.model_dump(mode="json"),
                }
                if side_effects:
                    payload["side_effects"] = side_effects
                yield {"type": "result", "data": payload}
                return

            error_entry = _append_trace(
                trace,
                flow=task_name,
                phase="error",
                status="error",
                code="tool_loop_exhausted",
                message="模型既没有返回内容，也没有发起工具调用。",
            )
            yield {"type": "trace", "data": error_entry.model_dump(mode="json")}
            raise ModelClientError(error_entry.message, code=error_entry.code)

        error_entry = _append_trace(
            trace,
            flow=task_name,
            phase="error",
            status="error",
            code="tool_loop_exhausted",
            message="工具循环已到上限，仍未得到最终答案。",
        )
        yield {"type": "trace", "data": error_entry.model_dump(mode="json")}
        raise ModelClientError(error_entry.message, code=error_entry.code)

    def _run_single_shot(
        self,
        *,
        task_name: str,
        response_model: type[BaseModel],
        system_prompt: str,
        user_prompt: str,
        tools: list[RuntimeTool],
        result_validator: Any = None,
        trace: RuntimeTrace,
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
            raise ModelClientError(
                "model returned empty content in single-shot mode",
                code="single_shot_failed",
            )
        try:
            result_model = validate_json_response(completion.content, response_model)
        except (ValueError, json.JSONDecodeError) as exc:
            raise ModelClientError(str(exc), code="json_parse_failed") from exc
        if result_validator is not None:
            validation_error = result_validator(result_model, {})
            if validation_error:
                _append_trace(
                    trace,
                    flow=task_name,
                    phase="validate",
                    status="error",
                    code="semantic_validation_failed",
                    message=validation_error,
                    attempt=1,
                )
                raise ModelClientError(
                    f"{task_name} single-shot validation failed: {validation_error}",
                    code="semantic_validation_failed",
                )
        _append_trace(
            trace,
            flow=task_name,
            phase="finalize",
            status="success",
            code="runtime_completed",
            message="single-shot fallback 已成功收口。",
        )
        return TaskExecutionResult(
            result=result_model,
            raw_output=completion.content.strip(),
            trace=trace,
        )

    def _stream_single_shot_task(
        self,
        *,
        task_name: str,
        response_model: type[BaseModel],
        system_prompt: str,
        user_prompt: str,
        tools: list[RuntimeTool],
        result_validator: Any = None,
        trace: RuntimeTrace,
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
            error_entry = _append_trace(
                trace,
                flow=task_name,
                phase="error",
                status="error",
                code="single_shot_failed",
                message="流式 single-shot 没有返回可用内容。",
            )
            yield {"type": "trace", "data": error_entry.model_dump(mode="json")}
            raise ModelClientError(error_entry.message, code=error_entry.code)

        try:
            result_model = validate_json_response(raw_output, response_model)
        except (ValueError, json.JSONDecodeError) as exc:
            error_entry = _append_trace(
                trace,
                flow=task_name,
                phase="validate",
                status="error",
                code="json_parse_failed",
                message=str(exc),
                attempt=1,
            )
            yield {"type": "trace", "data": error_entry.model_dump(mode="json")}
            raise ModelClientError(str(exc), code="json_parse_failed") from exc
        if result_validator is not None:
            validation_error = result_validator(result_model, {})
            if validation_error:
                error_entry = _append_trace(
                    trace,
                    flow=task_name,
                    phase="validate",
                    status="error",
                    code="semantic_validation_failed",
                    message=validation_error,
                    attempt=1,
                )
                yield {"type": "trace", "data": error_entry.model_dump(mode="json")}
                raise ModelClientError(
                    f"{task_name} streaming single-shot validation failed: {validation_error}",
                    code="semantic_validation_failed",
                )
        entry = _append_trace(
            trace,
            flow=task_name,
            phase="finalize",
            status="success",
            code="runtime_completed",
            message="single-shot fallback 已成功收口。",
        )
        yield {"type": "trace", "data": entry.model_dump(mode="json")}
        yield {
            "type": "result",
            "data": {
                "result": result_model.model_dump(mode="json"),
                "raw_output": raw_output,
                "trace": trace.model_dump(mode="json"),
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


def _validate_evaluation_result(
    request: EvaluateAnswerRequest,
    result: EvaluationResult,
    side_effects: dict[str, Any],
) -> str:
    score_keys = list(result.score_breakdown.keys())
    if not score_keys:
        return "missing score_breakdown"
    if not result.strengths and not result.gaps:
        return "missing strengths/gaps"

    depth_signal = side_effects.get("depth_signal", "normal")
    if depth_signal == "skip_followup":
        if result.followup_question or result.followup_expected_points:
            return "skip_followup must not include followup output"
        return ""

    is_last_turn = request.turn_index >= request.max_turns and depth_signal != "extend"
    if is_last_turn:
        if result.followup_question or result.followup_expected_points:
            return "last turn must not include followup output"
        return ""

    if not result.followup_question:
        return "missing followup_question on non-last turn"
    if not result.followup_expected_points:
        return "missing followup_expected_points on non-last turn"
    return ""


def _validate_review_result(result: ReviewCard, side_effects: dict[str, Any]) -> str:
    if not result.overall:
        return "missing overall"
    if not result.top_fix:
        return "missing top_fix"
    if not result.top_fix_reason:
        return "missing top_fix_reason"
    if not result.score_breakdown:
        return "missing score_breakdown"
    if result.recommended_next is None and not side_effects.get("recommended_next"):
        return "missing recommended_next"
    return ""


def _append_trace(
    trace: RuntimeTrace,
    *,
    flow: str,
    phase: str,
    status: str,
    code: str = "",
    message: str = "",
    attempt: int = 0,
    tool_name: str = "",
) -> RuntimeTraceEntry:
    entry = RuntimeTraceEntry(
        flow=flow,
        phase=phase,
        status=status,
        code=code,
        message=message,
        attempt=attempt,
        tool_name=tool_name,
    )
    trace.entries.append(entry)
    return entry


def _runtime_error_code(exc: Exception, *, fallback: str) -> str:
    if isinstance(exc, ModelClientError) and exc.code:
        return exc.code

    message = str(exc).lower()
    if "required tool context" in message:
        return "tool_context_missing"
    if "validation failed" in message:
        return "semantic_validation_failed"
    if "json" in message:
        return "json_parse_failed"
    if "go backend" in message:
        return "backend_callback_failed"
    if "timeout" in message or "timed out" in message:
        return "timeout"
    if "tool loop" in message:
        return "tool_loop_exhausted"
    return fallback
