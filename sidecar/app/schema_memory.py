from __future__ import annotations

from typing import Literal

from pydantic import BaseModel, Field

from app.schema_common import ProfileSnapshot, WeaknessTag


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


class EmbedMemoryItem(BaseModel):
    id: str
    text: str = Field(min_length=1, max_length=20_000)


class EmbedMemoryRequest(BaseModel):
    items: list[EmbedMemoryItem] = Field(default_factory=list, min_length=1, max_length=64)


class EmbeddedMemoryVector(BaseModel):
    id: str
    vector: list[float] = Field(default_factory=list)
    model_name: str = ""


class EmbedMemoryResponse(BaseModel):
    items: list[EmbeddedMemoryVector] = Field(default_factory=list)


class RerankMemoryCandidate(BaseModel):
    id: str
    text: str = Field(min_length=1, max_length=20_000)


class RerankMemoryRequest(BaseModel):
    query: str = Field(min_length=1, max_length=20_000)
    candidates: list[RerankMemoryCandidate] = Field(
        default_factory=list,
        min_length=1,
        max_length=64,
    )
    top_k: int = Field(default=5, ge=1, le=32)


class RerankMemoryResult(BaseModel):
    id: str
    score: float
    rank: int = Field(ge=1)


class RerankMemoryResponse(BaseModel):
    items: list[RerankMemoryResult] = Field(default_factory=list)


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
