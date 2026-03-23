from __future__ import annotations

import json
from typing import Any

from pydantic import BaseModel

from app.llm_client import ModelClientError, OpenAICompatibleModelClient
from app.runtime.state import TaskExecutionResult
from app.runtime.trace import append_trace
from app.runtime_support import RuntimeTool, tool_summary, validate_json_response
from app.schemas import RuntimeTrace


def run_single_shot(
    *,
    model_client: OpenAICompatibleModelClient,
    task_name: str,
    response_model: type[BaseModel],
    system_prompt: str,
    user_prompt: str,
    tools: list[RuntimeTool],
    result_validator: Any = None,
    trace: RuntimeTrace,
) -> TaskExecutionResult[BaseModel]:
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
        validation_error = result_validator(result_model, {}, [])
        if validation_error:
            append_trace(
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
    append_trace(
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


def stream_single_shot_task(
    *,
    model_client: OpenAICompatibleModelClient,
    task_name: str,
    response_model: type[BaseModel],
    system_prompt: str,
    user_prompt: str,
    tools: list[RuntimeTool],
    result_validator: Any = None,
    trace: RuntimeTrace,
):
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
        error_entry = append_trace(
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
        error_entry = append_trace(
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
        validation_error = result_validator(result_model, {}, [])
        if validation_error:
            error_entry = append_trace(
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
    entry = append_trace(
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
