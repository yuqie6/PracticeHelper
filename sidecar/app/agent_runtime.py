from __future__ import annotations

import json
import logging
from collections.abc import Callable
from dataclasses import dataclass, field
from typing import Any

from pydantic import BaseModel, Field

from app.config import Settings
from app.llm_client import ModelClientError, OpenAICompatibleModelClient
from app.repo_context import RepoAnalysisBundle, collect_repo_analysis_bundle
from app.schemas import (
    AnalyzeRepoRequest,
    AnalyzeRepoResponse,
    EvaluateAnswerRequest,
    EvaluationResult,
    GenerateQuestionRequest,
    GenerateQuestionResponse,
    GenerateReviewRequest,
    RepoChunk,
    ReviewCard,
)

logger = logging.getLogger(__name__)


class AnalyzeRepoDraft(BaseModel):
    summary: str
    highlights: list[str] = Field(default_factory=list)
    challenges: list[str] = Field(default_factory=list)
    tradeoffs: list[str] = Field(default_factory=list)
    ownership_points: list[str] = Field(default_factory=list)
    followup_points: list[str] = Field(default_factory=list)


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
        self._require_model_client()
        bundle = collect_repo_analysis_bundle(request, self._settings)
        tools = [
            RuntimeTool(
                name="read_repo_overview",
                description="Read the repository overview collected from the imported repository.",
                handler=lambda _: _repo_overview_payload(bundle),
            ),
            RuntimeTool(
                name="read_repo_chunks",
                description="Read the top repo chunks ranked by importance.",
                handler=lambda _: {
                    "chunks": _compact_chunks(bundle.chunks, limit=8, max_chars=520)
                },
            ),
        ]
        system_prompt = """
你是 PracticeHelper 的项目导入分析 agent。

任务：
1. 根据仓库概览和源码/文档片段，生成一份可用于项目面试训练的项目画像。
2. 你的输出必须真实、克制、可追问，不能只写漂亮话。
3. 输出必须是严格 JSON，字段只能是：
   - summary: string
   - highlights: string[]
   - challenges: string[]
   - tradeoffs: string[]
   - ownership_points: string[]
   - followup_points: string[]

要求：
- highlights / challenges / tradeoffs / ownership_points / followup_points 每项给 3 到 6 条。
- 尽量从工具返回的具体文件和内容里提炼，不要泛化成空话。
- 如果证据不够，就保守表达，不要脑补。
- 不要输出 Markdown，不要解释，只输出 JSON。
""".strip()
        user_prompt = _build_user_prompt(
            "请根据当前仓库材料生成项目画像。",
            request,
            response_shape="""
{
  "summary": "string",
  "highlights": ["string"],
  "challenges": ["string"],
  "tradeoffs": ["string"],
  "ownership_points": ["string"],
  "followup_points": ["string"]
}
""".strip(),
        )
        draft = self._run_task(
            response_model=AnalyzeRepoDraft,
            system_prompt=system_prompt,
            user_prompt=user_prompt,
            tools=tools,
        )
        return AnalyzeRepoResponse(
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

    def generate_question(self, request: GenerateQuestionRequest) -> GenerateQuestionResponse:
        tools = [
            RuntimeTool(
                name="read_question_templates",
                description="Read curated question templates for basics training.",
                handler=lambda _: {
                    "templates": [item.model_dump(mode="json") for item in request.templates],
                },
            ),
            RuntimeTool(
                name="read_project_brief",
                description="Read the current project profile for project interview mode.",
                handler=lambda _: {
                    "project": request.project.model_dump(mode="json") if request.project else None,
                },
            ),
            RuntimeTool(
                name="read_context_chunks",
                description="Read the retrieved repo chunks that can ground follow-up questions.",
                handler=lambda _: {"chunks": _compact_chunks(request.context_chunks)},
            ),
            RuntimeTool(
                name="read_weakness_memory",
                description="Read the current weakness memory accumulated from previous sessions.",
                handler=lambda _: {
                    "weaknesses": [item.model_dump(mode="json") for item in request.weaknesses],
                },
            ),
        ]
        system_prompt = """
你是 PracticeHelper 的真实面试训练 agent。

目标：
1. 先利用可用工具理解用户当前训练上下文。
2. 再生成一条有训练价值的主问题，而不是泛泛而谈。
3. 输出必须是严格 JSON，字段只能是：
   - question: string
   - expected_points: string[]

要求：
- basics 模式优先围绕主题、历史弱项和模板做一条可追问的问题。
- project 模式必须围绕项目背景、trade-off、ownership 和真实结果。
- expected_points 控制在 4 到 6 个，必须具体、可判定。
- 不要输出 Markdown，不要解释，只输出 JSON。
""".strip()
        user_prompt = _build_user_prompt(
            "请生成本轮训练的主问题。",
            request,
            response_shape="""
{
  "question": "string",
  "expected_points": ["string"]
}
""".strip(),
        )
        return self._run_task(
            response_model=GenerateQuestionResponse,
            system_prompt=system_prompt,
            user_prompt=user_prompt,
            tools=tools,
        )

    def evaluate_answer(self, request: EvaluateAnswerRequest) -> EvaluationResult:
        tools = [
            RuntimeTool(
                name="read_evaluation_context",
                description=(
                    "Read the question, expected points, answer, and project context for scoring."
                ),
                handler=lambda _: {
                    "mode": request.mode,
                    "topic": request.topic,
                    "question": request.question,
                    "expected_points": request.expected_points,
                    "answer": request.answer,
                    "project": request.project.model_dump(mode="json") if request.project else None,
                    "context_chunks": _compact_chunks(request.context_chunks),
                    "is_followup": request.is_followup,
                },
            ),
        ]
        system_prompt = """
你是 PracticeHelper 的追问型面试官 agent。

任务：
1. 结合工具返回的上下文，对回答做结构化评估。
2. 分清楚回答覆盖了什么、缺了什么、哪里虚、下一刀该追问哪里。
3. 输出必须是严格 JSON，字段只能是：
   - score: number (0-100)
   - score_breakdown: object<string, number>
   - strengths: string[]
   - gaps: string[]
   - followup_question: string
   - followup_expected_points: string[]
   - weakness_hits: [{"kind": string, "label": string, "severity": number}]

要求：
- strengths 和 gaps 要具体，不要写空话。
- 非 followup 回答时必须给一条追问问题；followup 回答时 followup_question 置空。
- weakness_hits 最多 3 条，severity 在 0 到 1.5 之间。
- 不要输出 Markdown，不要解释，只输出 JSON。
""".strip()
        user_prompt = _build_user_prompt(
            "请评估这次回答，并决定下一刀追问。",
            request,
            response_shape="""
{
  "score": 0,
  "score_breakdown": {"维度": 0},
  "strengths": ["string"],
  "gaps": ["string"],
  "followup_question": "string",
  "followup_expected_points": ["string"],
  "weakness_hits": [{"kind": "topic", "label": "redis", "severity": 0.6}]
}
""".strip(),
        )
        return self._run_task(
            response_model=EvaluationResult,
            system_prompt=system_prompt,
            user_prompt=user_prompt,
            tools=tools,
        )

    def generate_review(self, request: GenerateReviewRequest) -> ReviewCard:
        tools = [
            RuntimeTool(
                name="read_session_summary",
                description="Read the current session summary and project summary if available.",
                handler=lambda _: {
                    "session": request.session.model_dump(mode="json"),
                    "project": request.project.model_dump(mode="json") if request.project else None,
                },
            ),
            RuntimeTool(
                name="read_turn_history",
                description="Read all turns, including evaluations and follow-up evaluations.",
                handler=lambda _: {
                    "turns": [turn.model_dump(mode="json") for turn in request.turns],
                },
            ),
        ]
        system_prompt = """
你是 PracticeHelper 的复盘 agent。

任务：
1. 阅读整轮训练历史，输出一张真正可执行的 review card。
2. 重点总结：整体判断、亮点、漏洞、建议主题、下一轮重点。
3. 输出必须是严格 JSON，字段只能是：
   - overall: string
   - highlights: string[]
   - gaps: string[]
   - suggested_topics: string[]
   - next_training_focus: string[]
   - score_breakdown: object<string, number>

要求：
- overall 要像一个严厉但有帮助的教练总结。
- highlights 和 gaps 都要尽量去重且具体。
- next_training_focus 要能直接拿去开始下一轮训练。
- 不要输出 Markdown，不要解释，只输出 JSON。
""".strip()
        user_prompt = _build_user_prompt(
            "请根据整轮训练历史生成最终复盘卡。",
            request,
            response_shape="""
{
  "overall": "string",
  "highlights": ["string"],
  "gaps": ["string"],
  "suggested_topics": ["string"],
  "next_training_focus": ["string"],
  "score_breakdown": {"维度": 0}
}
""".strip(),
        )
        return self._run_task(
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
            # 主路径优先保留 tool-calling：上下文更小，模型也会被限制在“先取证再回答”的节奏里。
            # 只有当 provider 的工具调用或 JSON 输出不稳定时，才退化到 single-shot 兼容模式。
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

        for _ in range(4):
            result = model_client.create_completion(
                messages=messages,
                tools=[tool.spec() for tool in tools],
            )
            if result.tool_calls:
                # 这里显式回灌 assistant/tool 消息，是为了维持标准的
                # assistant -> tool -> assistant 对话协议，让下一轮模型继续基于已取回的证据推理。
                # 循环次数固定为 4，是为了防止不稳定 provider 陷入无限工具循环。
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
                    arguments = _parse_tool_arguments(tool_call)
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
                continue

            if result.content.strip():
                return _validate_json_response(result.content, response_model)

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

        # single-shot 是兼容兜底，不是首选路径。
        # 它把所有工具结果一次性灌进 prompt，成功率通常更高，但会失去逐步取证能力并放大上下文体积。
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
        return _validate_json_response(result.content, response_model)

    def _require_model_client(self) -> OpenAICompatibleModelClient:
        if self._model_client is None:
            raise ModelClientError(
                "LLM is required for sidecar core flows. Configure PRACTICEHELPER_SIDECAR_MODEL, "
                "PRACTICEHELPER_SIDECAR_OPENAI_BASE_URL, and PRACTICEHELPER_SIDECAR_OPENAI_API_KEY."
            )
        return self._model_client


def _compact_chunks(
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


def _repo_overview_payload(bundle: RepoAnalysisBundle) -> dict[str, Any]:
    return {
        "repo_url": bundle.repo_url,
        "name": bundle.name,
        "default_branch": bundle.default_branch,
        "import_commit": bundle.import_commit,
        "tech_stack": bundle.tech_stack,
        "top_paths": bundle.top_paths[:8],
        "chunk_count": len(bundle.chunks),
    }


def _build_user_prompt(instruction: str, request: BaseModel, *, response_shape: str) -> str:
    return (
        f"{instruction}\n\n"
        "这是当前请求的结构化载荷：\n"
        f"{json.dumps(request.model_dump(mode='json'), ensure_ascii=False, indent=2)}\n\n"
        "最终答案必须匹配下面这个 JSON 形状：\n"
        f"{response_shape}"
    )


def _validate_json_response[ResponseModelT: BaseModel](
    text: str, response_model: type[ResponseModelT]
) -> ResponseModelT:
    candidate = _extract_json_block(text)
    data = json.loads(candidate)
    return response_model.model_validate(data)


def _extract_json_block(text: str) -> str:
    stripped = text.strip()
    if stripped.startswith("```"):
        lines = stripped.splitlines()
        if len(lines) >= 3:
            stripped = "\n".join(lines[1:-1]).strip()

    # 这里容忍模型输出 fenced code block 或在 JSON 外包一层解释文本，
    # 但只接受单个 object，避免在多段内容里做模糊猜测式解析。
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


def _parse_tool_arguments(tool_call: dict[str, Any]) -> dict[str, Any]:
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
