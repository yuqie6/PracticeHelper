from __future__ import annotations

import json
from typing import Any

from pydantic import BaseModel, Field

from app.prompt_loader import load_prompt, render_prompt
from app.repo_context import RepoAnalysisBundle
from app.runtime_support import RuntimeTool, compact_chunks, repo_overview_payload
from app.schemas import EvaluateAnswerRequest, GenerateQuestionRequest, GenerateReviewRequest


class AnalyzeRepoDraft(BaseModel):
    summary: str
    highlights: list[str] = Field(default_factory=list)
    challenges: list[str] = Field(default_factory=list)
    tradeoffs: list[str] = Field(default_factory=list)
    ownership_points: list[str] = Field(default_factory=list)
    followup_points: list[str] = Field(default_factory=list)


_DEFAULT_SCORE_WEIGHTS: dict[str, float] = {
    "准确性": 30,
    "完整性": 25,
    "落地感": 15,
    "表达清晰度": 15,
    "抗追问能力": 15,
}


def analyze_repo_prompt_bundle(
    bundle: RepoAnalysisBundle,
) -> tuple[str, str, list[RuntimeTool]]:
    system_prompt = load_prompt("analyze_repo_system.md")
    user_prompt = "请根据当前仓库材料生成项目画像。"
    tools = [
        RuntimeTool(
            name="read_repo_overview",
            description="Read the repository overview collected from the imported repository.",
            handler=lambda _: repo_overview_payload(bundle),
        ),
        RuntimeTool(
            name="read_repo_chunks",
            description="Read the top repo chunks ranked by importance.",
            handler=lambda _: {
                "chunks": compact_chunks(bundle.chunks, limit=8, max_chars=520)
            },
        ),
    ]
    return system_prompt, user_prompt, tools


def question_prompt_bundle(
    request: GenerateQuestionRequest,
) -> tuple[str, str, list[RuntimeTool]]:
    system_prompt = load_prompt("generate_question_system.md")
    user_prompt = (
        f"请生成本轮训练的主问题。\n"
        f"当前模式：{request.mode}，主题：{request.topic}\n\n"
        '最终答案必须匹配：{{"question": "string", "expected_points": ["string"]}}'
    )
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
            handler=lambda _: {"chunks": compact_chunks(request.context_chunks)},
        ),
        RuntimeTool(
            name="read_weakness_memory",
            description="Read the current weakness memory accumulated from previous sessions.",
            handler=lambda _: {
                "weaknesses": [item.model_dump(mode="json") for item in request.weaknesses],
            },
        ),
    ]
    return system_prompt, user_prompt, tools


def evaluate_prompt_bundle(
    request: EvaluateAnswerRequest,
) -> tuple[str, str, list[RuntimeTool]]:
    weights = request.score_weights or _DEFAULT_SCORE_WEIGHTS
    rubric_lines = "\n".join(f"- {key} ({int(value)}%)" for key, value in weights.items())
    dimensions_example = json.dumps({key: 0 for key in weights}, ensure_ascii=False)
    system_prompt = render_prompt(
        "evaluate_answer_system.md",
        {
            "RUBRIC_LINES": rubric_lines,
            "DIMENSIONS_EXAMPLE": dimensions_example,
        },
    )
    followup_label = "是" if request.is_followup else "否"
    user_prompt = (
        f"请评估这次回答，并决定下一刀追问。\n"
        f"当前模式：{request.mode}，主题：{request.topic}，"
        f"是否为追问回答：{followup_label}"
    )
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
                "context_chunks": compact_chunks(request.context_chunks),
                "is_followup": request.is_followup,
            },
        ),
    ]
    return system_prompt, user_prompt, tools


def review_prompt_bundle(
    request: GenerateReviewRequest,
) -> tuple[str, str, list[RuntimeTool]]:
    system_prompt = load_prompt("generate_review_system.md")
    user_prompt = "请根据整轮训练历史生成最终复盘卡。"
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
    return system_prompt, user_prompt, tools
