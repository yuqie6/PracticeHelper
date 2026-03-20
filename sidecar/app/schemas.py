from __future__ import annotations

from typing import Literal

from pydantic import BaseModel, Field, field_validator

WeaknessKind = Literal["topic", "project", "expression", "followup_breakdown", "depth", "detail"]


def _normalize_weakness_kind(value: str) -> str:
    normalized = value.strip().lower().replace("-", "_").replace(" ", "_")
    aliases = {
        "communication": "expression",
        "followup": "followup_breakdown",
        "follow_up": "followup_breakdown",
        "followupbreakdown": "followup_breakdown",
    }
    return aliases.get(normalized, normalized)


class RepoChunk(BaseModel):
    id: str = ""
    project_id: str = ""
    file_path: str
    file_type: str
    content: str
    importance: float = Field(default=0.5, ge=0.0, le=1.5)
    fts_key: str


class ProjectProfile(BaseModel):
    id: str = ""
    name: str
    repo_url: str = ""
    default_branch: str = "main"
    import_commit: str = ""
    summary: str
    tech_stack: list[str] = Field(default_factory=list)
    highlights: list[str] = Field(default_factory=list)
    challenges: list[str] = Field(default_factory=list)
    tradeoffs: list[str] = Field(default_factory=list)
    ownership_points: list[str] = Field(default_factory=list)
    followup_points: list[str] = Field(default_factory=list)
    import_status: str = "ready"


class WeaknessHit(BaseModel):
    kind: WeaknessKind
    label: str
    severity: float = Field(default=0.4, ge=0.0, le=1.5)

    @field_validator("kind", mode="before")
    @classmethod
    def normalize_kind(cls, value: str) -> str:
        return _normalize_weakness_kind(value)


class WeaknessTag(BaseModel):
    id: str = ""
    kind: WeaknessKind
    label: str
    severity: float = Field(default=0.4, ge=0.0, le=1.5)
    frequency: int = 1
    last_seen_at: str = ""
    evidence_session_id: str = ""

    @field_validator("kind", mode="before")
    @classmethod
    def normalize_kind(cls, value: str) -> str:
        return _normalize_weakness_kind(value)


class QuestionTemplate(BaseModel):
    id: str = ""
    mode: str
    topic: str
    prompt: str
    focus_points: list[str] = Field(default_factory=list)
    bad_answers: list[str] = Field(default_factory=list)
    followup_templates: list[str] = Field(default_factory=list)
    score_weights: dict[str, float] = Field(default_factory=dict)


class AnalyzeRepoRequest(BaseModel):
    repo_url: str


class AnalyzeRepoResponse(BaseModel):
    repo_url: str
    name: str
    default_branch: str
    import_commit: str
    summary: str
    tech_stack: list[str] = Field(default_factory=list)
    highlights: list[str] = Field(default_factory=list)
    challenges: list[str] = Field(default_factory=list)
    tradeoffs: list[str] = Field(default_factory=list)
    ownership_points: list[str] = Field(default_factory=list)
    followup_points: list[str] = Field(default_factory=list)
    chunks: list[RepoChunk] = Field(default_factory=list)


class GenerateQuestionRequest(BaseModel):
    mode: Literal["basics", "project"]
    topic: str = ""
    intensity: str = "standard"
    project: ProjectProfile | None = None
    templates: list[QuestionTemplate] = Field(default_factory=list)
    context_chunks: list[RepoChunk] = Field(default_factory=list)
    weaknesses: list[WeaknessTag] = Field(default_factory=list)


class GenerateQuestionResponse(BaseModel):
    question: str
    expected_points: list[str] = Field(default_factory=list)


class EvaluateAnswerRequest(BaseModel):
    mode: Literal["basics", "project"]
    topic: str = ""
    project: ProjectProfile | None = None
    question: str
    expected_points: list[str] = Field(default_factory=list)
    answer: str
    context_chunks: list[RepoChunk] = Field(default_factory=list)
    is_followup: bool = False
    score_weights: dict[str, float] = Field(default_factory=dict)


class EvaluationResult(BaseModel):
    score: float = Field(ge=0.0, le=100.0)
    score_breakdown: dict[str, float] = Field(default_factory=dict)
    strengths: list[str] = Field(default_factory=list)
    gaps: list[str] = Field(default_factory=list)
    followup_question: str = ""
    followup_expected_points: list[str] = Field(default_factory=list)
    weakness_hits: list[WeaknessHit] = Field(default_factory=list)


class TrainingTurn(BaseModel):
    id: str = ""
    question: str
    expected_points: list[str] = Field(default_factory=list)
    answer: str = ""
    evaluation: EvaluationResult | None = None
    followup_question: str = ""
    followup_expected_points: list[str] = Field(default_factory=list)
    followup_answer: str = ""
    followup_evaluation: EvaluationResult | None = None


class TrainingSession(BaseModel):
    id: str = ""
    mode: Literal["basics", "project"]
    topic: str = ""
    project_id: str = ""
    intensity: str = "standard"
    status: str = ""
    total_score: float = 0.0


class GenerateReviewRequest(BaseModel):
    session: TrainingSession
    project: ProjectProfile | None = None
    turns: list[TrainingTurn] = Field(default_factory=list)


class ReviewCard(BaseModel):
    id: str = ""
    session_id: str = ""
    overall: str
    highlights: list[str] = Field(default_factory=list)
    gaps: list[str] = Field(default_factory=list)
    suggested_topics: list[str] = Field(default_factory=list)
    next_training_focus: list[str] = Field(default_factory=list)
    score_breakdown: dict[str, float] = Field(default_factory=dict)
