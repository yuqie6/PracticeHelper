from __future__ import annotations

import json
from typing import Any

from pydantic import BaseModel, Field

from app.prompt_loader import load_prompt, render_prompt
from app.repo_context import RepoAnalysisBundle
from app.runtime_support import RuntimeTool, compact_chunks, repo_overview_payload
from app.schemas import (
    AnalyzeJobTargetRequest,
    EvaluateAnswerRequest,
    GenerateQuestionRequest,
    GenerateReviewRequest,
)


class AnalyzeRepoDraft(BaseModel):
    summary: str
    highlights: list[str] = Field(default_factory=list)
    challenges: list[str] = Field(default_factory=list)
    tradeoffs: list[str] = Field(default_factory=list)
    ownership_points: list[str] = Field(default_factory=list)
    followup_points: list[str] = Field(default_factory=list)


class AnalyzeJobTargetDraft(BaseModel):
    summary: str
    must_have_skills: list[str] = Field(default_factory=list)
    bonus_skills: list[str] = Field(default_factory=list)
    responsibilities: list[str] = Field(default_factory=list)
    evaluation_focus: list[str] = Field(default_factory=list)


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
            handler=lambda _: {"chunks": compact_chunks(bundle.chunks, limit=8, max_chars=520)},
        ),
    ]
    return system_prompt, user_prompt, tools


def analyze_job_target_prompt_bundle(
    request: AnalyzeJobTargetRequest,
) -> tuple[str, str, list[RuntimeTool]]:
    system_prompt = load_prompt("analyze_job_target_system.md")
    user_prompt = "请根据当前岗位 JD 生成结构化岗位要求。"
    tools = [
        RuntimeTool(
            name="read_job_target_source",
            description="Read the current job target metadata and original JD text.",
            handler=lambda _: {
                "title": request.title,
                "company_name": request.company_name,
                "source_text": request.source_text,
            },
        ),
    ]
    return system_prompt, user_prompt, tools


def question_prompt_bundle(
    request: GenerateQuestionRequest,
) -> tuple[str, str, list[RuntimeTool]]:
    system_prompt = load_prompt("generate_question_system.md")
    jd_label = "有" if request.job_target_analysis else "无"
    strategy_hint = {
        "weakness_first": "出题策略：优先围绕用户历史弱项出题，确保题目直击薄弱环节。",
        "project_deep_dive": "出题策略：优先围绕项目的取舍、挑战和追问点深挖，而非泛问基础概念。",
        "template_based": "出题策略：从模板库选取合适题目，结合弱项和岗位要求微调。",
    }.get(request.strategy, "")
    user_prompt = (
        f"请生成本轮训练的主问题。\n"
        f"当前模式：{request.mode}，主题：{request.topic}，是否绑定岗位 JD：{jd_label}\n"
    )
    if strategy_hint:
        user_prompt += f"{strategy_hint}\n"
    user_prompt += '\n最终答案必须匹配：{{"question": "string", "expected_points": ["string"]}}'
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
        RuntimeTool(
            name="read_job_target_analysis",
            description="Read the bound job target analysis snapshot for this training session.",
            handler=lambda _: {
                "job_target_analysis": (
                    request.job_target_analysis.model_dump(mode="json")
                    if request.job_target_analysis
                    else None
                ),
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
    followup_label = "是" if request.turn_index > 1 else "否"
    jd_label = "有" if request.job_target_analysis else "无"
    is_last_turn = request.turn_index >= request.max_turns
    turn_label = f"{request.turn_index}/{request.max_turns}"
    user_prompt = (
        f"请评估这次回答，并决定下一刀追问。\n"
        f"当前模式：{request.mode}，主题：{request.topic}，"
        f"是否为追问回答：{followup_label}，当前轮次：{turn_label}，是否绑定岗位 JD：{jd_label}"
    )
    if is_last_turn:
        user_prompt += "\n注意：这是最后一轮，不需要生成追问，followup_question 和 followup_expected_points 置空。"
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
                "turn_index": request.turn_index,
                "max_turns": request.max_turns,
                "job_target_analysis": (
                    request.job_target_analysis.model_dump(mode="json")
                    if request.job_target_analysis
                    else None
                ),
            },
        ),
    ]
    return system_prompt, user_prompt, tools


def review_prompt_bundle(
    request: GenerateReviewRequest,
) -> tuple[str, str, list[RuntimeTool]]:
    system_prompt = load_prompt("generate_review_system.md")
    jd_label = "有" if request.job_target_analysis else "无"
    user_prompt = f"请根据整轮训练历史生成最终复盘卡。是否绑定岗位 JD：{jd_label}"
    tools = [
        RuntimeTool(
            name="read_session_summary",
            description="Read the current session summary and project summary if available.",
            handler=lambda _: {
                "session": request.session.model_dump(mode="json"),
                "project": request.project.model_dump(mode="json") if request.project else None,
                "job_target_analysis": (
                    request.job_target_analysis.model_dump(mode="json")
                    if request.job_target_analysis
                    else None
                ),
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
