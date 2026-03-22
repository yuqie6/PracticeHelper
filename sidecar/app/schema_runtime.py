from __future__ import annotations

from urllib.parse import urlparse

from pydantic import BaseModel, Field, field_validator

from app.schema_common import (
    DepthSignal,
    JobTargetAnalysisSnapshot,
    ProjectProfile,
    QuestionTemplate,
    RepoChunk,
    WeaknessHit,
    WeaknessTag,
)
from app.schema_memory import AgentContext, AgentObservation, KnowledgeUpdate


class AnalyzeRepoRequest(BaseModel):
    repo_url: str

    @field_validator("repo_url")
    @classmethod
    def validate_repo_url(cls, value: str) -> str:
        parsed = urlparse(value)
        if parsed.scheme not in ("https", "http") or not parsed.hostname:
            raise ValueError("repo_url must be a valid http(s) URL")
        return value


class AnalyzeRepoResponse(BaseModel):
    repo_url: str
    name: str
    default_branch: str = "main"
    import_commit: str = ""
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


class RuntimeTraceEntry(BaseModel):
    flow: str
    phase: str
    status: str = "info"
    code: str = ""
    message: str = ""
    attempt: int = 0
    tool_name: str = ""


class RuntimeTrace(BaseModel):
    entries: list[RuntimeTraceEntry] = Field(default_factory=list)


class AnalyzeRepoEnvelope(BaseModel):
    result: AnalyzeRepoResponse
    raw_output: str = ""
    trace: RuntimeTrace | None = None


class AnalyzeJobTargetEnvelope(BaseModel):
    result: AnalyzeJobTargetResponse
    raw_output: str = ""
    trace: RuntimeTrace | None = None


class GenerateQuestionRequest(BaseModel):
    mode: str
    topic: str = ""
    candidate_topics: list[str] = Field(default_factory=list)
    strategy: str = ""
    prompt_set_id: str = ""
    intensity: str
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
    trace: RuntimeTrace | None = None


class EvaluateAnswerRequest(BaseModel):
    mode: str
    topic: str = ""
    prompt_set_id: str = ""
    retry_feedback: str = ""
    project: ProjectProfile | None = None
    question: str
    expected_points: list[str] = Field(default_factory=list)
    answer: str
    context_chunks: list[RepoChunk] = Field(default_factory=list)
    turn_index: int = 1
    max_turns: int = 1
    score_weights: dict[str, float] = Field(default_factory=dict)
    job_target_analysis: JobTargetAnalysisSnapshot | None = None
    agent_context: AgentContext | None = None


class EvaluationResult(BaseModel):
    score: float
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
    trace: RuntimeTrace | None = None


class TrainingTurn(BaseModel):
    id: str = ""
    turn_index: int = 0
    question: str = ""
    expected_points: list[str] = Field(default_factory=list)
    answer: str = ""
    evaluation: EvaluationResult | None = None


class TrainingSession(BaseModel):
    id: str
    mode: str
    topic: str = ""
    project_id: str = ""
    job_target_id: str = ""
    job_target_analysis_id: str = ""
    prompt_set_id: str = ""
    intensity: str = "standard"
    status: str = ""
    max_turns: int = 1
    total_score: float = 0
    review_id: str = ""
    turns: list[TrainingTurn] = Field(default_factory=list)
    project: ProjectProfile | None = None


class GenerateReviewRequest(BaseModel):
    session: TrainingSession
    project: ProjectProfile | None = None
    turns: list[TrainingTurn] = Field(default_factory=list)
    prompt_set_id: str = ""
    retry_feedback: str = ""
    job_target_analysis: JobTargetAnalysisSnapshot | None = None
    agent_context: AgentContext | None = None


class NextSession(BaseModel):
    mode: str
    topic: str = ""
    project_id: str = ""
    reason: str = ""


class ReviewCard(BaseModel):
    id: str = ""
    session_id: str = ""
    job_target_id: str = ""
    job_target_analysis_id: str = ""
    prompt_set_id: str = ""
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
    side_effects: GenerateReviewSideEffects | None = None
    raw_output: str = ""
    trace: RuntimeTrace | None = None
