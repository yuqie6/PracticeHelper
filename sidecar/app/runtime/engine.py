from __future__ import annotations

import json
import logging
from typing import Any

from pydantic import BaseModel

from app.llm_client import ModelClientError, OpenAICompatibleModelClient
from app.runtime.single_shot import run_single_shot
from app.runtime.state import TaskExecutionResult, ToolRuntimeState
from app.runtime.trace import (
    append_trace,
    command_trace_code,
    command_trace_details,
    command_trace_message,
    command_trace_status,
    context_compaction_message,
    is_command_tool,
    runtime_error_code,
    tool_failure_code,
)
from app.runtime.validation import bind_runtime_tool, read_only_tools
from app.runtime_support import RuntimeTool, parse_tool_arguments, validate_json_response
from app.schemas import RuntimeTrace

logger = logging.getLogger(__name__)


def run_task(
    *,
    model_client: OpenAICompatibleModelClient,
    task_name: str,
    response_model: type[BaseModel],
    system_prompt: str,
    user_prompt: str,
    tools: list[RuntimeTool],
    fallback_tools: list[RuntimeTool] | None = None,
    result_validator: Any = None,
    context_trace_details: list[dict[str, Any]] | None = None,
) -> TaskExecutionResult[BaseModel]:
    fallback_tools = fallback_tools or read_only_tools(tools)
    trace = RuntimeTrace()

    try:
        return run_agent_loop(
            model_client=model_client,
            task_name=task_name,
            response_model=response_model,
            system_prompt=system_prompt,
            user_prompt=user_prompt,
            tools=tools,
            result_validator=result_validator,
            trace=trace,
            context_trace_details=context_trace_details,
        )
    except (ModelClientError, ValueError, json.JSONDecodeError) as exc:
        logger.warning("agent tool loop failed, retrying single-shot: %s", exc)
        append_trace(
            trace,
            flow=task_name,
            phase="fallback",
            status="fallback",
            code=runtime_error_code(exc, fallback="single_shot_failed"),
            message="工具循环没有稳定收口，退回单轮生成。",
        )

    try:
        return run_single_shot(
            model_client=model_client,
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
        append_trace(
            trace,
            flow=task_name,
            phase="error",
            status="error",
            code=runtime_error_code(exc, fallback="single_shot_failed"),
            message=str(exc),
        )
        raise


def run_agent_loop(
    *,
    model_client: OpenAICompatibleModelClient,
    task_name: str,
    response_model: type[BaseModel],
    system_prompt: str,
    user_prompt: str,
    tools: list[RuntimeTool],
    result_validator: Any = None,
    trace: RuntimeTrace,
    context_trace_details: list[dict[str, Any]] | None = None,
) -> TaskExecutionResult[BaseModel]:
    messages: list[dict[str, Any]] = [
        {"role": "system", "content": system_prompt},
        {"role": "user", "content": user_prompt},
    ]
    tool_map = {tool.name: tool for tool in tools}
    used_any_tool = False
    runtime_state = ToolRuntimeState()
    validation_attempts = 0
    append_trace(
        trace,
        flow=task_name,
        phase="prepare_context",
        status="info",
        code="runtime_started",
        message="开始执行 agent runtime。",
    )
    for detail in context_trace_details or []:
        append_trace(
            trace,
            flow=task_name,
            phase="prepare_context",
            status="info",
            code="context_compacted",
            message=context_compaction_message(detail),
            details=detail,
        )

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
                    error_entry = append_trace(
                        trace,
                        flow=task_name,
                        phase="tool_call",
                        status="error",
                        code="tool_call_failed",
                        message=f"模型请求了未知工具：{tool_name}",
                        tool_name=tool_name,
                    )
                    raise ModelClientError(error_entry.message, code=error_entry.code)
                tool = bind_runtime_tool(tool, runtime_state)
                request_entry = None
                if is_command_tool(tool_name):
                    request_entry = append_trace(
                        trace,
                        flow=task_name,
                        phase="command",
                        status="info",
                        code="command_requested",
                        message=f"命令 {tool_name} 已发起请求。",
                        tool_name=tool_name,
                        details=command_trace_details(tool_name, arguments),
                    )
                try:
                    tool_result = tool.handler(arguments)
                except Exception as exc:  # noqa: BLE001
                    code = tool_failure_code(tool_name, exc)
                    error_entry = append_trace(
                        trace,
                        flow=task_name,
                        phase="command" if is_command_tool(tool_name) else "tool_call",
                        status="error",
                        code=code,
                        message=f"工具 {tool_name} 执行失败：{exc}",
                        tool_name=tool_name,
                    )
                    raise ModelClientError(error_entry.message, code=error_entry.code) from exc
                if is_command_tool(tool_name):
                    if request_entry is not None:
                        request_entry.details.update(command_trace_details(tool_name, tool_result))
                    append_trace(
                        trace,
                        flow=task_name,
                        phase="command",
                        status=command_trace_status(tool_result),
                        code=command_trace_code(tool_result),
                        message=command_trace_message(tool_name, tool_result),
                        tool_name=tool_name,
                        details=command_trace_details(tool_name, tool_result),
                    )
                else:
                    append_trace(
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
                error_entry = append_trace(
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
                    validation_error = result_validator(
                        result_model,
                        runtime_state.side_effects,
                        runtime_state.command_results,
                    )
                    if validation_error:
                        validation_code = "semantic_validation_failed"

            if validation_error:
                validation_attempts += 1
                status = "retry" if validation_attempts < 2 else "error"
                append_trace(
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
                            f"上一次输出没有过校验，请修正这些问题后重新生成：{validation_error}"
                        ),
                    }
                )
                continue

            append_trace(
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
                side_effects=runtime_state.side_effects,
                command_results=runtime_state.command_results,
                trace=trace,
            )

        error_entry = append_trace(
            trace,
            flow=task_name,
            phase="error",
            status="error",
            code="tool_loop_exhausted",
            message="模型既没有返回内容，也没有发起工具调用。",
        )
        raise ModelClientError(error_entry.message, code=error_entry.code)

    error_entry = append_trace(
        trace,
        flow=task_name,
        phase="error",
        status="error",
        code="tool_loop_exhausted",
        message="工具循环已到上限，仍未得到最终答案。",
    )
    raise ModelClientError(error_entry.message, code=error_entry.code)
