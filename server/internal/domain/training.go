package domain

import "time"

type TrainingSession struct {
	ID                  string            `json:"id"`
	Mode                string            `json:"mode"`
	Topic               string            `json:"topic,omitempty"`
	ProjectID           string            `json:"project_id,omitempty"`
	JobTargetID         string            `json:"job_target_id,omitempty"`
	JobTargetAnalysisID string            `json:"job_target_analysis_id,omitempty"`
	PromptSetID         string            `json:"prompt_set_id,omitempty"`
	Intensity           string            `json:"intensity"`
	Status              string            `json:"status"`
	MaxTurns            int               `json:"max_turns"`
	TotalScore          float64           `json:"total_score"`
	StartedAt           *time.Time        `json:"started_at,omitempty"`
	EndedAt             *time.Time        `json:"ended_at,omitempty"`
	CreatedAt           time.Time         `json:"created_at"`
	UpdatedAt           time.Time         `json:"updated_at"`
	ReviewID            string            `json:"review_id,omitempty"`
	Turns               []TrainingTurn    `json:"turns,omitempty"`
	Project             *ProjectProfile   `json:"project,omitempty"`
	JobTarget           *JobTargetRef     `json:"job_target,omitempty"`
	PromptSet           *PromptSetSummary `json:"prompt_set,omitempty"`
}

type TrainingTurn struct {
	ID             string            `json:"id"`
	SessionID      string            `json:"session_id"`
	TurnIndex      int               `json:"turn_index"`
	Stage          string            `json:"stage"`
	Question       string            `json:"question"`
	ExpectedPoints []string          `json:"expected_points"`
	Answer         string            `json:"answer"`
	Evaluation     *EvaluationResult `json:"evaluation,omitempty"`
	WeaknessHits   []WeaknessHit     `json:"weakness_hits,omitempty"`
	CreatedAt      time.Time         `json:"created_at"`
	UpdatedAt      time.Time         `json:"updated_at"`
}

type EvaluationResult struct {
	Score            float64            `json:"score"`
	ScoreBreakdown   map[string]float64 `json:"score_breakdown"`
	Headline         string             `json:"headline,omitempty"`
	Strengths        []string           `json:"strengths"`
	Gaps             []string           `json:"gaps"`
	Suggestion       string             `json:"suggestion,omitempty"`
	FollowupIntent   string             `json:"followup_intent,omitempty"`
	FollowupQuestion string             `json:"followup_question,omitempty"`
	FollowupPoints   []string           `json:"followup_expected_points,omitempty"`
	WeaknessHits     []WeaknessHit      `json:"weakness_hits"`
}

type WeaknessHit struct {
	Kind     string  `json:"kind"`
	Label    string  `json:"label"`
	Severity float64 `json:"severity"`
}

type WeaknessTag struct {
	ID                string    `json:"id"`
	Kind              string    `json:"kind"`
	Label             string    `json:"label"`
	Severity          float64   `json:"severity"`
	Frequency         int       `json:"frequency"`
	LastSeenAt        time.Time `json:"last_seen_at"`
	EvidenceSessionID string    `json:"evidence_session_id"`
}

type WeaknessTrendPoint struct {
	SessionID string  `json:"session_id"`
	Severity  float64 `json:"severity"`
	CreatedAt string  `json:"created_at"`
}

type WeaknessTrend struct {
	ID     string               `json:"id"`
	Kind   string               `json:"kind"`
	Label  string               `json:"label"`
	Points []WeaknessTrendPoint `json:"points"`
}

type ReviewScheduleItem struct {
	ID            int64     `json:"id"`
	SessionID     string    `json:"session_id"`
	ReviewCardID  string    `json:"review_card_id,omitempty"`
	WeaknessTagID string    `json:"weakness_tag_id,omitempty"`
	WeaknessKind  string    `json:"weakness_kind,omitempty"`
	WeaknessLabel string    `json:"weakness_label,omitempty"`
	Topic         string    `json:"topic,omitempty"`
	NextReviewAt  time.Time `json:"next_review_at"`
	IntervalDays  int       `json:"interval_days"`
	EaseFactor    float64   `json:"ease_factor"`
	CreatedAt     time.Time `json:"created_at"`
}

type ReviewCard struct {
	ID                  string             `json:"id"`
	SessionID           string             `json:"session_id"`
	JobTargetID         string             `json:"job_target_id,omitempty"`
	JobTargetAnalysisID string             `json:"job_target_analysis_id,omitempty"`
	PromptSetID         string             `json:"prompt_set_id,omitempty"`
	Overall             string             `json:"overall"`
	TopFix              string             `json:"top_fix,omitempty"`
	TopFixReason        string             `json:"top_fix_reason,omitempty"`
	Highlights          []string           `json:"highlights"`
	Gaps                []string           `json:"gaps"`
	SuggestedTopics     []string           `json:"suggested_topics"`
	NextTrainingFocus   []string           `json:"next_training_focus"`
	RecommendedNext     *NextSession       `json:"recommended_next,omitempty"`
	RetrievalTrace      *RetrievalTrace    `json:"retrieval_trace,omitempty"`
	ScoreBreakdown      map[string]float64 `json:"score_breakdown"`
	CreatedAt           time.Time          `json:"created_at"`
	JobTarget           *JobTargetRef      `json:"job_target,omitempty"`
	PromptSet           *PromptSetSummary  `json:"prompt_set,omitempty"`
}

type NextSession struct {
	Mode      string `json:"mode"`
	Topic     string `json:"topic,omitempty"`
	ProjectID string `json:"project_id,omitempty"`
	Reason    string `json:"reason,omitempty"`
}

type TrainingSessionSummary struct {
	ID          string            `json:"id"`
	Mode        string            `json:"mode"`
	Topic       string            `json:"topic,omitempty"`
	ProjectName string            `json:"project_name,omitempty"`
	Status      string            `json:"status"`
	TotalScore  float64           `json:"total_score"`
	ReviewID    string            `json:"review_id,omitempty"`
	UpdatedAt   time.Time         `json:"updated_at"`
	JobTarget   *JobTargetRef     `json:"job_target,omitempty"`
	PromptSetID string            `json:"prompt_set_id,omitempty"`
	PromptSet   *PromptSetSummary `json:"prompt_set,omitempty"`
}

type ListSessionsRequest struct {
	Page    int    `form:"page"`
	PerPage int    `form:"per_page"`
	Mode    string `form:"mode"`
	Topic   string `form:"topic"`
	Status  string `form:"status"`
}

type PaginatedList[T any] struct {
	Items      []T `json:"items"`
	Total      int `json:"total"`
	Page       int `json:"page"`
	PerPage    int `json:"per_page"`
	TotalPages int `json:"total_pages"`
}

type CreateSessionRequest struct {
	Mode                  string `json:"mode" binding:"required"`
	Topic                 string `json:"topic"`
	ProjectID             string `json:"project_id"`
	JobTargetID           string `json:"job_target_id"`
	PromptSetID           string `json:"prompt_set_id"`
	IgnoreActiveJobTarget bool   `json:"ignore_active_job_target,omitempty"`
	Intensity             string `json:"intensity" binding:"required"`
	MaxTurns              int    `json:"max_turns,omitempty"`
}

type ExportSessionsRequest struct {
	SessionIDs []string `json:"session_ids" binding:"required"`
	Format     string   `json:"format" binding:"required"`
}

type SubmitAnswerRequest struct {
	Answer string `json:"answer" binding:"required"`
}

type StreamEvent struct {
	Type    string `json:"type"`
	Code    string `json:"code,omitempty"`
	Phase   string `json:"phase,omitempty"`
	Name    string `json:"name,omitempty"`
	Text    string `json:"text,omitempty"`
	Message string `json:"message,omitempty"`
	Data    any    `json:"data,omitempty"`
}
