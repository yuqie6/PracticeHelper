from __future__ import annotations

from typing import Literal
from urllib.parse import urlparse

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


class KnowledgeNode(BaseModel):
    id: str
    scope_type: str = "global"
    scope_id: str = ""
    parent_id: str = ""
    label: str
    node_type: Literal["topic", "concept", "skill"]
    proficiency: float = Field(default=0, ge=0.0, le=5.0)
    confidence: float = Field(default=0.5, ge=0.0, le=1.0)
    hit_count: int = Field(default=0, ge=0)
    last_assessed_at: str | None = None
    created_at: str = ""
    updated_at: str = ""


class KnowledgeEdge(BaseModel):
    source_id: str
    target_id: str
    edge_type: Literal["contains", "prerequisite", "related"]
    created_at: str = ""


class KnowledgeSubgraph(BaseModel):
    nodes: list[KnowledgeNode] = Field(default_factory=list)
    edges: list[KnowledgeEdge] = Field(default_factory=list)


class AgentObservation(BaseModel):
    id: str = ""
    session_id: str = ""
    scope_type: str = "global"
    scope_id: str = ""
    topic: str = ""
    category: Literal["pattern", "misconception", "growth", "strategy_note"]
    content: str
    tags: list[str] = Field(default_factory=list)
    relevance: float = Field(default=1.0, ge=0.0, le=10.0)
    created_at: str = ""
    archived_at: str | None = None


class SessionMemorySummary(BaseModel):
    id: str = ""
    session_id: str
    mode: Literal["basics", "project"]
    topic: str = ""
    project_id: str = ""
    job_target_id: str = ""
    prompt_set_id: str = ""
    summary: str
    strengths: list[str] = Field(default_factory=list)
    gaps: list[str] = Field(default_factory=list)
    misconceptions: list[str] = Field(default_factory=list)
    growth: list[str] = Field(default_factory=list)
    recommended_focus: list[str] = Field(default_factory=list)
    salience: float = Field(default=0.5, ge=0.0, le=1.0)
    created_at: str = ""
    updated_at: str = ""


class AgentContext(BaseModel):
    profile: ProfileSnapshot | None = None
    knowledge_subgraph: KnowledgeSubgraph | None = None
    observations: list[AgentObservation] = Field(default_factory=list)
    weakness_profile: list[WeaknessTag] = Field(default_factory=list)
    session_summaries: list[SessionMemorySummary] = Field(default_factory=list)


class KnowledgeUpdate(BaseModel):
    node_id: str = ""
    scope_type: str = "global"
    scope_id: str = ""
    parent_id: str = ""
    label: str = ""
    node_type: str = ""
    proficiency: float = Field(default=0.0, ge=0.0, le=5.0)
    confidence: float = Field(default=0.0, ge=0.0, le=1.0)
    evidence: str = ""
    observed_label: str = ""


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


class AnalyzeRepoRequest(BaseModel):
    repo_url: str

    @field_validator("repo_url")
    @classmethod
    def validate_repo_url(cls, value: str) -> str:
        parsed = urlparse(value)
        if parsed.scheme not in ("https", "http") or not parsed.hostname:
            raise ValueError("repo_url must be a valid HTTP(S) URL")
        return value


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


class AnalyzeJobTargetRequest(BaseModel):
    title: str = ""
    company_name: str = ""
    source_text: str


class AnalyzeJobTargetResponse(JobTargetAnalysisSnapshot):
    pass


class AnalyzeRepoEnvelope(BaseModel):
    result: AnalyzeRepoResponse
    raw_output: str = ""


class AnalyzeJobTargetEnvelope(BaseModel):
    result: AnalyzeJobTargetResponse
    raw_output: str = ""


class GenerateQuestionRequest(BaseModel):
    mode: Literal["basics", "project"]
    topic: str = Field(default="", max_length=200)
    candidate_topics: list[str] = Field(default_factory=list)
    prompt_set_id: str = ""
    intensity: str = "standard"
    strategy: str = ""
    project: ProjectProfile | None = None
    templates: list[QuestionTemplate] = Field(default_factory=list)
    context_chunks: list[RepoChunk] = Field(default_factory=list)
    weaknesses: list[WeaknessTag] = Field(default_factory=list)
    job_target_analysis: JobTargetAnalysisSnapshot | None = None
    agent_context: AgentContext | None = None


class GenerateQuestionResponse(BaseModel):
    question: str
    expected_points: list[str] = Field(default_factory=list)


class GenerateQuestionEnvelope(BaseModel):
    result: GenerateQuestionResponse
    raw_output: str = ""


class EvaluateAnswerRequest(BaseModel):
    mode: Literal["basics", "project"]
    topic: str = ""
    prompt_set_id: str = ""
    project: ProjectProfile | None = None
    question: str
    expected_points: list[str] = Field(default_factory=list)
    answer: str = Field(max_length=50_000)
    context_chunks: list[RepoChunk] = Field(default_factory=list)
    turn_index: int = Field(default=1, ge=1, le=10)
    max_turns: int = Field(default=2, ge=1, le=10)
    score_weights: dict[str, float] = Field(default_factory=dict)
    job_target_analysis: JobTargetAnalysisSnapshot | None = None
    retry_feedback: str = ""
    agent_context: AgentContext | None = None


class EvaluationResult(BaseModel):
    score: float = Field(ge=0.0, le=100.0)
    score_breakdown: dict[str, float] = Field(default_factory=dict)
    headline: str = ""
    strengths: list[str] = Field(default_factory=list)
    gaps: list[str] = Field(default_factory=list)
    suggestion: str = ""
    followup_intent: str = ""
    followup_question: str = ""
    followup_expected_points: list[str] = Field(default_factory=list)
    weakness_hits: list[WeaknessHit] = Field(default_factory=list)


class EvaluateAnswerSideEffects(BaseModel):
    observations: list[AgentObservation] = Field(default_factory=list)
    knowledge_updates: list[KnowledgeUpdate] = Field(default_factory=list)
    depth_signal: DepthSignal = "normal"


class EvaluateAnswerEnvelope(BaseModel):
    result: EvaluationResult
    side_effects: EvaluateAnswerSideEffects = Field(default_factory=EvaluateAnswerSideEffects)
    raw_output: str = ""


class TrainingTurn(BaseModel):
    id: str = ""
    turn_index: int = 1
    question: str
    expected_points: list[str] = Field(default_factory=list)
    answer: str = ""
    evaluation: EvaluationResult | None = None


class TrainingSession(BaseModel):
    id: str = ""
    mode: Literal["basics", "project"]
    topic: str = ""
    project_id: str = ""
    job_target_id: str = ""
    job_target_analysis_id: str = ""
    prompt_set_id: str = ""
    intensity: str = "standard"
    status: str = ""
    max_turns: int = 0
    total_score: float = 0.0
    project: ProjectProfile | None = None
    turns: list[TrainingTurn] = Field(default_factory=list)


class GenerateReviewRequest(BaseModel):
    session: TrainingSession
    project: ProjectProfile | None = None
    turns: list[TrainingTurn] = Field(default_factory=list)
    prompt_set_id: str = ""
    job_target_analysis: JobTargetAnalysisSnapshot | None = None
    retry_feedback: str = ""
    agent_context: AgentContext | None = None


class NextSession(BaseModel):
    mode: Literal["basics", "project"]
    topic: str = ""
    project_id: str = ""
    reason: str = ""


class ReviewCard(BaseModel):
    id: str = ""
    session_id: str = ""
    overall: str
    top_fix: str = ""
    top_fix_reason: str = ""
    highlights: list[str] = Field(default_factory=list)
    gaps: list[str] = Field(default_factory=list)
    suggested_topics: list[str] = Field(default_factory=list)
    next_training_focus: list[str] = Field(default_factory=list)
    recommended_next: NextSession | None = None
    score_breakdown: dict[str, float] = Field(default_factory=dict)


class AgentSessionDetail(BaseModel):
    session: TrainingSession
    review: ReviewCard | None = None


class GenerateReviewSideEffects(BaseModel):
    observations: list[AgentObservation] = Field(default_factory=list)
    knowledge_updates: list[KnowledgeUpdate] = Field(default_factory=list)
    recommended_next: NextSession | None = None


class GenerateReviewEnvelope(BaseModel):
    result: ReviewCard
    side_effects: GenerateReviewSideEffects = Field(default_factory=GenerateReviewSideEffects)
    raw_output: str = ""


EvaluateAnswerEnvelope.model_rebuild()
