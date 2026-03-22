from __future__ import annotations

from typing import Literal

from pydantic import BaseModel, Field, field_validator

WeaknessKind = Literal["topic", "project", "expression", "followup_breakdown", "depth", "detail"]
DepthSignal = Literal["skip_followup", "extend", "normal"]


def _normalize_weakness_kind(value: str) -> str:
    normalized = value.strip().lower().replace("-", "_").replace(" ", "_")
    aliases = {
        "accuracy": "detail",
        "correctness": "detail",
        "precision": "detail",
        "completeness": "depth",
        "coverage": "depth",
        "breadth": "depth",
        "clarity": "expression",
        "structure": "expression",
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


class JobTargetAnalysisSnapshot(BaseModel):
    summary: str = ""
    must_have_skills: list[str] = Field(default_factory=list)
    bonus_skills: list[str] = Field(default_factory=list)
    responsibilities: list[str] = Field(default_factory=list)
    evaluation_focus: list[str] = Field(default_factory=list)


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


class JobTargetRef(BaseModel):
    id: str
    title: str
    company_name: str = ""
    latest_analysis_status: str = ""


class ProfileSnapshot(BaseModel):
    target_role: str = ""
    target_company_type: str = ""
    current_stage: str = ""
    application_deadline: str | None = None
    tech_stacks: list[str] = Field(default_factory=list)
    primary_projects: list[str] = Field(default_factory=list)
    self_reported_weaknesses: list[str] = Field(default_factory=list)
    active_job_target: JobTargetRef | None = None


class QuestionTemplate(BaseModel):
    id: str = ""
    mode: str
    topic: str
    prompt: str
    focus_points: list[str] = Field(default_factory=list)
    bad_answers: list[str] = Field(default_factory=list)
    followup_templates: list[str] = Field(default_factory=list)
    score_weights: dict[str, float] = Field(default_factory=dict)


class PromptSetSummary(BaseModel):
    id: str
    label: str
    description: str = ""
    status: str
    is_default: bool = False
