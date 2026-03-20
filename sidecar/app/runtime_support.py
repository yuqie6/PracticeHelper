from __future__ import annotations

import json
from collections.abc import Callable
from dataclasses import dataclass, field
from typing import Any

from pydantic import BaseModel

from app.repo_context import RepoAnalysisBundle
from app.schemas import RepoChunk


def _empty_parameters() -> dict[str, Any]:
    return {"type": "object", "properties": {}, "additionalProperties": False}


@dataclass
class RuntimeTool:
    name: str
    description: str
    handler: Callable[[dict[str, Any]], dict[str, Any]]
    parameters: dict[str, Any] = field(default_factory=_empty_parameters)

    def spec(self) -> dict[str, Any]:
        return {
            "type": "function",
            "function": {
                "name": self.name,
                "description": self.description,
                "parameters": self.parameters,
            },
        }


def compact_chunks(
    chunks: list[RepoChunk], *, limit: int = 6, max_chars: int = 420
) -> list[dict[str, Any]]:
    compact: list[dict[str, Any]] = []
    for chunk in chunks[:limit]:
        compact.append(
            {
                "file_path": chunk.file_path,
                "file_type": chunk.file_type,
                "importance": chunk.importance,
                "fts_key": chunk.fts_key,
                "content": chunk.content[:max_chars],
            }
        )
    return compact


def repo_overview_payload(bundle: RepoAnalysisBundle) -> dict[str, Any]:
    return {
        "repo_url": bundle.repo_url,
        "name": bundle.name,
        "default_branch": bundle.default_branch,
        "import_commit": bundle.import_commit,
        "tech_stack": bundle.tech_stack,
        "top_paths": bundle.top_paths[:8],
        "chunk_count": len(bundle.chunks),
    }


def validate_json_response[ResponseModelT: BaseModel](
    text: str, response_model: type[ResponseModelT]
) -> ResponseModelT:
    candidate = extract_json_block(text)
    data = json.loads(candidate)
    return response_model.model_validate(data)


def extract_json_block(text: str) -> str:
    stripped = text.strip()
    if stripped.startswith("```"):
        lines = stripped.splitlines()
        if len(lines) >= 3:
            stripped = "\n".join(lines[1:-1]).strip()

    try:
        json.loads(stripped)
        return stripped
    except json.JSONDecodeError:
        pass

    start = stripped.find("{")
    end = stripped.rfind("}")
    if start == -1 or end == -1 or end <= start:
        raise ValueError(f"model did not return JSON: {text}")

    candidate = stripped[start : end + 1]
    json.loads(candidate)
    return candidate


def parse_tool_arguments(tool_call: dict[str, Any]) -> dict[str, Any]:
    raw = tool_call.get("function", {}).get("arguments", "")
    if not raw:
        return {}
    try:
        parsed = json.loads(raw)
    except json.JSONDecodeError as exc:
        raise ValueError(f"tool arguments are not valid JSON: {raw}") from exc
    if not isinstance(parsed, dict):
        raise ValueError(f"tool arguments must be an object: {raw}")
    return parsed


def tool_summary(tool_name: str) -> str:
    summaries = {
        "read_question_templates": "已读取基础题模板，正在挑选更适合当前训练目标的问题。",
        "read_project_brief": "已读取项目摘要与亮点，正在组织更贴近真实项目的问题。",
        "read_context_chunks": "已读取源码与文档片段，正在结合具体上下文生成内容。",
        "read_weakness_memory": "已读取历史薄弱点，正在优先围绕弱项组织训练内容。",
        "read_evaluation_context": "已读取题目、答案与上下文，正在对照关键点评估回答。",
        "read_session_summary": "已读取整轮训练摘要，正在整理复盘主线。",
        "read_turn_history": "已读取所有问答记录，正在归纳亮点、漏洞和下一步建议。",
        "read_repo_overview": "已读取仓库概览，正在判断项目主线与技术栈。",
        "read_repo_chunks": "已读取关键源码片段，正在提炼项目亮点与难点。",
    }
    return summaries.get(tool_name, "")
