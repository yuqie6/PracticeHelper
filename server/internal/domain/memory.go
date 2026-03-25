package domain

import "time"

type KnowledgeNode struct {
	ID             string     `json:"id"`
	ScopeType      string     `json:"scope_type,omitempty"`
	ScopeID        string     `json:"scope_id,omitempty"`
	ParentID       string     `json:"parent_id,omitempty"`
	Label          string     `json:"label"`
	NodeType       string     `json:"node_type"`
	Proficiency    float64    `json:"proficiency"`
	Confidence     float64    `json:"confidence"`
	HitCount       int        `json:"hit_count"`
	LastAssessedAt *time.Time `json:"last_assessed_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

type KnowledgeEdge struct {
	SourceID  string    `json:"source_id"`
	TargetID  string    `json:"target_id"`
	EdgeType  string    `json:"edge_type"`
	CreatedAt time.Time `json:"created_at"`
}

type KnowledgeSubgraph struct {
	Nodes []KnowledgeNode `json:"nodes"`
	Edges []KnowledgeEdge `json:"edges"`
}

type KnowledgeSnapshot struct {
	ID          string    `json:"id"`
	NodeID      string    `json:"node_id"`
	SessionID   string    `json:"session_id,omitempty"`
	Proficiency float64   `json:"proficiency"`
	Confidence  float64   `json:"confidence"`
	Evidence    string    `json:"evidence,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

type KnowledgeUpdate struct {
	NodeID        string  `json:"node_id,omitempty"`
	ScopeType     string  `json:"scope_type,omitempty"`
	ScopeID       string  `json:"scope_id,omitempty"`
	ParentID      string  `json:"parent_id,omitempty"`
	Label         string  `json:"label,omitempty"`
	NodeType      string  `json:"node_type,omitempty"`
	Proficiency   float64 `json:"proficiency"`
	Confidence    float64 `json:"confidence,omitempty"`
	Evidence      string  `json:"evidence,omitempty"`
	ObservedLabel string  `json:"observed_label,omitempty"`
}

type AgentObservation struct {
	ID         string     `json:"id"`
	SessionID  string     `json:"session_id,omitempty"`
	ScopeType  string     `json:"scope_type,omitempty"`
	ScopeID    string     `json:"scope_id,omitempty"`
	Topic      string     `json:"topic,omitempty"`
	Category   string     `json:"category"`
	Content    string     `json:"content"`
	Tags       []string   `json:"tags,omitempty"`
	Relevance  float64    `json:"relevance"`
	CreatedAt  time.Time  `json:"created_at"`
	ArchivedAt *time.Time `json:"archived_at,omitempty"`
}

type SessionMemorySummary struct {
	ID               string    `json:"id"`
	SessionID        string    `json:"session_id"`
	Mode             string    `json:"mode"`
	Topic            string    `json:"topic,omitempty"`
	ProjectID        string    `json:"project_id,omitempty"`
	JobTargetID      string    `json:"job_target_id,omitempty"`
	PromptSetID      string    `json:"prompt_set_id,omitempty"`
	Summary          string    `json:"summary"`
	Strengths        []string  `json:"strengths,omitempty"`
	Gaps             []string  `json:"gaps,omitempty"`
	Misconceptions   []string  `json:"misconceptions,omitempty"`
	GrowthSignals    []string  `json:"growth,omitempty"`
	RecommendedFocus []string  `json:"recommended_focus,omitempty"`
	Salience         float64   `json:"salience"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type MemoryIndexEntry struct {
	ID          string    `json:"id"`
	MemoryType  string    `json:"memory_type"`
	ScopeType   string    `json:"scope_type"`
	ScopeID     string    `json:"scope_id,omitempty"`
	Topic       string    `json:"topic,omitempty"`
	ProjectID   string    `json:"project_id,omitempty"`
	SessionID   string    `json:"session_id,omitempty"`
	JobTargetID string    `json:"job_target_id,omitempty"`
	Tags        []string  `json:"tags,omitempty"`
	Entities    []string  `json:"entities,omitempty"`
	Summary     string    `json:"summary,omitempty"`
	Salience    float64   `json:"salience"`
	Confidence  float64   `json:"confidence"`
	Freshness   float64   `json:"freshness"`
	RefTable    string    `json:"ref_table"`
	RefID       string    `json:"ref_id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type MemoryRef struct {
	RefTable string `json:"ref_table"`
	RefID    string `json:"ref_id"`
}

type MemoryEmbeddingRecord struct {
	ID            string     `json:"id"`
	MemoryIndexID string     `json:"memory_index_id"`
	MemoryType    string     `json:"memory_type"`
	RefTable      string     `json:"ref_table"`
	RefID         string     `json:"ref_id"`
	ContentHash   string     `json:"content_hash"`
	ModelName     string     `json:"model_name"`
	VectorStoreID string     `json:"vector_store_id"`
	VectorDim     int        `json:"vector_dim"`
	Status        string     `json:"status"`
	LastError     string     `json:"last_error,omitempty"`
	LastIndexedAt *time.Time `json:"last_indexed_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

type MemoryEmbeddingJob struct {
	ID             string     `json:"id"`
	MemoryIndexID  string     `json:"memory_index_id"`
	MemoryType     string     `json:"memory_type"`
	RefTable       string     `json:"ref_table"`
	RefID          string     `json:"ref_id"`
	Status         string     `json:"status"`
	AttemptCount   int        `json:"attempt_count"`
	ErrorMessage   string     `json:"error_message,omitempty"`
	ClaimToken     string     `json:"claim_token,omitempty"`
	ClaimExpiresAt *time.Time `json:"claim_expires_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	StartedAt      *time.Time `json:"started_at,omitempty"`
	FinishedAt     *time.Time `json:"finished_at,omitempty"`
}

type EmbedMemoryItem struct {
	ID   string `json:"id"`
	Text string `json:"text"`
}

type EmbedMemoryRequest struct {
	Items []EmbedMemoryItem `json:"items"`
}

type EmbeddedMemoryVector struct {
	ID        string    `json:"id"`
	Vector    []float64 `json:"vector"`
	ModelName string    `json:"model_name,omitempty"`
}

type EmbedMemoryResponse struct {
	Items []EmbeddedMemoryVector `json:"items"`
}

type RerankMemoryCandidate struct {
	ID   string `json:"id"`
	Text string `json:"text"`
}

type RerankMemoryRequest struct {
	Query      string                  `json:"query"`
	Candidates []RerankMemoryCandidate `json:"candidates"`
	TopK       int                     `json:"top_k,omitempty"`
}

type RerankMemoryResult struct {
	ID    string  `json:"id"`
	Score float64 `json:"score"`
	Rank  int     `json:"rank"`
}

type RerankMemoryResponse struct {
	Items []RerankMemoryResult `json:"items"`
}

type AgentContext struct {
	Profile           *ProfileSnapshot       `json:"profile,omitempty"`
	KnowledgeSubgraph *KnowledgeSubgraph     `json:"knowledge_subgraph,omitempty"`
	Observations      []AgentObservation     `json:"observations,omitempty"`
	WeaknessProfile   []WeaknessTag          `json:"weakness_profile,omitempty"`
	SessionSummaries  []SessionMemorySummary `json:"session_summaries,omitempty"`
}

type RetrievalTrace struct {
	GeneratedAt      time.Time             `json:"generated_at"`
	Topic            string                `json:"topic,omitempty"`
	ProjectID        string                `json:"project_id,omitempty"`
	JobTargetID      string                `json:"job_target_id,omitempty"`
	ObservationTrace *MemoryRetrievalTrace `json:"observations,omitempty"`
	SummaryTrace     *MemoryRetrievalTrace `json:"session_summaries,omitempty"`
}

type MemoryRetrievalTrace struct {
	MemoryType     string               `json:"memory_type"`
	Query          string               `json:"query,omitempty"`
	Strategy       string               `json:"strategy"`
	CandidateCount int                  `json:"candidate_count"`
	SelectedCount  int                  `json:"selected_count"`
	FallbackUsed   bool                 `json:"fallback_used,omitempty"`
	FallbackReason string               `json:"fallback_reason,omitempty"`
	Hits           []MemoryRetrievalHit `json:"hits,omitempty"`
}

type MemoryRetrievalHit struct {
	Source        string  `json:"source"`
	MemoryIndexID string  `json:"memory_index_id,omitempty"`
	RefTable      string  `json:"ref_table,omitempty"`
	RefID         string  `json:"ref_id,omitempty"`
	ScopeType     string  `json:"scope_type,omitempty"`
	ScopeID       string  `json:"scope_id,omitempty"`
	Topic         string  `json:"topic,omitempty"`
	Summary       string  `json:"summary,omitempty"`
	RuleScore     float64 `json:"rule_score,omitempty"`
	VectorScore   float64 `json:"vector_score,omitempty"`
	RerankScore   float64 `json:"rerank_score,omitempty"`
	FinalScore    float64 `json:"final_score,omitempty"`
	Reason        string  `json:"reason,omitempty"`
}
