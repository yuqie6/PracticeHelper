package domain

import "time"

type QuestionTemplate struct {
	ID                string             `json:"id"`
	Mode              string             `json:"mode"`
	Topic             string             `json:"topic"`
	Prompt            string             `json:"prompt"`
	FocusPoints       []string           `json:"focus_points"`
	BadAnswers        []string           `json:"bad_answers"`
	FollowupTemplates []string           `json:"followup_templates"`
	ScoreWeights      map[string]float64 `json:"score_weights"`
}

type PromptSetSummary struct {
	ID          string `json:"id"`
	Label       string `json:"label"`
	Description string `json:"description,omitempty"`
	Status      string `json:"status"`
	IsDefault   bool   `json:"is_default,omitempty"`
}

type PromptExecutionMeta struct {
	ModelName   string `json:"model_name,omitempty"`
	PromptSetID string `json:"prompt_set_id,omitempty"`
	PromptHash  string `json:"prompt_hash,omitempty"`
	RawOutput   string `json:"raw_output,omitempty"`
}

type EvaluationLogEntry struct {
	ID          int64     `json:"id"`
	SessionID   string    `json:"session_id"`
	TurnID      string    `json:"turn_id,omitempty"`
	FlowName    string    `json:"flow_name"`
	ModelName   string    `json:"model_name,omitempty"`
	PromptSetID string    `json:"prompt_set_id,omitempty"`
	PromptHash  string    `json:"prompt_hash,omitempty"`
	RawOutput   string    `json:"raw_output,omitempty"`
	LatencyMs   float64   `json:"latency_ms"`
	CreatedAt   time.Time `json:"created_at"`
}

type PromptExperimentRequest struct {
	Left  string `form:"left"`
	Right string `form:"right"`
	Mode  string `form:"mode"`
	Topic string `form:"topic"`
	Limit int    `form:"limit"`
}

type PromptExperimentFilters struct {
	Left  string `json:"left"`
	Right string `json:"right"`
	Mode  string `json:"mode,omitempty"`
	Topic string `json:"topic,omitempty"`
	Limit int    `json:"limit"`
}

type PromptExperimentMetrics struct {
	PromptSet                    PromptSetSummary `json:"prompt_set"`
	SessionCount                 int              `json:"session_count"`
	CompletedCount               int              `json:"completed_count"`
	AvgTotalScore                float64          `json:"avg_total_score"`
	AvgGenerateQuestionLatencyMs float64          `json:"avg_generate_question_latency_ms"`
	AvgEvaluateAnswerLatencyMs   float64          `json:"avg_evaluate_answer_latency_ms"`
	AvgGenerateReviewLatencyMs   float64          `json:"avg_generate_review_latency_ms"`
}

type PromptExperimentSample struct {
	SessionID  string           `json:"session_id"`
	ReviewID   string           `json:"review_id,omitempty"`
	Mode       string           `json:"mode"`
	Topic      string           `json:"topic,omitempty"`
	Status     string           `json:"status"`
	TotalScore float64          `json:"total_score"`
	UpdatedAt  time.Time        `json:"updated_at"`
	PromptSet  PromptSetSummary `json:"prompt_set"`
}

type PromptExperimentReport struct {
	Left           PromptExperimentMetrics  `json:"left"`
	Right          PromptExperimentMetrics  `json:"right"`
	RecentSamples  []PromptExperimentSample `json:"recent_samples"`
	AppliedFilters PromptExperimentFilters  `json:"applied_filters"`
}
