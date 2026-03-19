package domain

import "time"

const (
	ModeBasics  = "basics"
	ModeProject = "project"

	StatusDraft         = "draft"
	StatusActive        = "active"
	StatusWaitingAnswer = "waiting_answer"
	StatusEvaluating    = "evaluating"
	StatusFollowup      = "followup"
	StatusReviewPending = "review_pending"
	StatusCompleted     = "completed"
)

type UserProfile struct {
	ID                   int64      `json:"id"`
	TargetRole           string     `json:"target_role"`
	TargetCompanyType    string     `json:"target_company_type"`
	CurrentStage         string     `json:"current_stage"`
	ApplicationDeadline  *time.Time `json:"application_deadline,omitempty"`
	TechStacks           []string   `json:"tech_stacks"`
	PrimaryProjects      []string   `json:"primary_projects"`
	SelfReportedWeakness []string   `json:"self_reported_weaknesses"`
	CreatedAt            time.Time  `json:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at"`
}

type UserProfileInput struct {
	TargetRole           string   `json:"target_role" binding:"required"`
	TargetCompanyType    string   `json:"target_company_type" binding:"required"`
	CurrentStage         string   `json:"current_stage" binding:"required"`
	ApplicationDeadline  *string  `json:"application_deadline"`
	TechStacks           []string `json:"tech_stacks"`
	PrimaryProjects      []string `json:"primary_projects"`
	SelfReportedWeakness []string `json:"self_reported_weaknesses"`
}

type ProjectProfile struct {
	ID              string    `json:"id"`
	Name            string    `json:"name"`
	RepoURL         string    `json:"repo_url"`
	DefaultBranch   string    `json:"default_branch"`
	ImportCommit    string    `json:"import_commit"`
	Summary         string    `json:"summary"`
	TechStack       []string  `json:"tech_stack"`
	Highlights      []string  `json:"highlights"`
	Challenges      []string  `json:"challenges"`
	Tradeoffs       []string  `json:"tradeoffs"`
	OwnershipPoints []string  `json:"ownership_points"`
	FollowupPoints  []string  `json:"followup_points"`
	ImportStatus    string    `json:"import_status"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type ProjectImportRequest struct {
	RepoURL string `json:"repo_url" binding:"required"`
}

type ProjectProfileInput struct {
	Name            string   `json:"name" binding:"required"`
	Summary         string   `json:"summary" binding:"required"`
	TechStack       []string `json:"tech_stack"`
	Highlights      []string `json:"highlights"`
	Challenges      []string `json:"challenges"`
	Tradeoffs       []string `json:"tradeoffs"`
	OwnershipPoints []string `json:"ownership_points"`
	FollowupPoints  []string `json:"followup_points"`
}

type RepoChunk struct {
	ID         string    `json:"id"`
	ProjectID  string    `json:"project_id"`
	FilePath   string    `json:"file_path"`
	FileType   string    `json:"file_type"`
	Content    string    `json:"content"`
	Importance float64   `json:"importance"`
	FTSKey     string    `json:"fts_key"`
	CreatedAt  time.Time `json:"created_at"`
}

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

type TrainingSession struct {
	ID         string          `json:"id"`
	Mode       string          `json:"mode"`
	Topic      string          `json:"topic,omitempty"`
	ProjectID  string          `json:"project_id,omitempty"`
	Intensity  string          `json:"intensity"`
	Status     string          `json:"status"`
	TotalScore float64         `json:"total_score"`
	StartedAt  *time.Time      `json:"started_at,omitempty"`
	EndedAt    *time.Time      `json:"ended_at,omitempty"`
	CreatedAt  time.Time       `json:"created_at"`
	UpdatedAt  time.Time       `json:"updated_at"`
	ReviewID   string          `json:"review_id,omitempty"`
	Turns      []TrainingTurn  `json:"turns,omitempty"`
	Project    *ProjectProfile `json:"project,omitempty"`
}

type TrainingTurn struct {
	ID                    string            `json:"id"`
	SessionID             string            `json:"session_id"`
	TurnIndex             int               `json:"turn_index"`
	Stage                 string            `json:"stage"`
	Question              string            `json:"question"`
	ExpectedPoints        []string          `json:"expected_points"`
	Answer                string            `json:"answer"`
	Evaluation            *EvaluationResult `json:"evaluation,omitempty"`
	FollowupQuestion      string            `json:"followup_question,omitempty"`
	FollowupExpectedPoint []string          `json:"followup_expected_points,omitempty"`
	FollowupAnswer        string            `json:"followup_answer,omitempty"`
	FollowupEvaluation    *EvaluationResult `json:"followup_evaluation,omitempty"`
	WeaknessHits          []WeaknessHit     `json:"weakness_hits,omitempty"`
	CreatedAt             time.Time         `json:"created_at"`
	UpdatedAt             time.Time         `json:"updated_at"`
}

type EvaluationResult struct {
	Score            float64            `json:"score"`
	ScoreBreakdown   map[string]float64 `json:"score_breakdown"`
	Strengths        []string           `json:"strengths"`
	Gaps             []string           `json:"gaps"`
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

type ReviewCard struct {
	ID                string             `json:"id"`
	SessionID         string             `json:"session_id"`
	Overall           string             `json:"overall"`
	Highlights        []string           `json:"highlights"`
	Gaps              []string           `json:"gaps"`
	SuggestedTopics   []string           `json:"suggested_topics"`
	NextTrainingFocus []string           `json:"next_training_focus"`
	ScoreBreakdown    map[string]float64 `json:"score_breakdown"`
	CreatedAt         time.Time          `json:"created_at"`
}

type TrainingSessionSummary struct {
	ID          string    `json:"id"`
	Mode        string    `json:"mode"`
	Topic       string    `json:"topic,omitempty"`
	ProjectName string    `json:"project_name,omitempty"`
	Status      string    `json:"status"`
	TotalScore  float64   `json:"total_score"`
	ReviewID    string    `json:"review_id,omitempty"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type Dashboard struct {
	Profile           *UserProfile             `json:"profile"`
	Weaknesses        []WeaknessTag            `json:"weaknesses"`
	RecentSessions    []TrainingSessionSummary `json:"recent_sessions"`
	CurrentSession    *TrainingSessionSummary  `json:"current_session,omitempty"`
	TodayFocus        string                   `json:"today_focus"`
	RecommendedTrack  string                   `json:"recommended_track"`
	DaysUntilDeadline *int                     `json:"days_until_deadline,omitempty"`
}

type CreateSessionRequest struct {
	Mode      string `json:"mode" binding:"required"`
	Topic     string `json:"topic"`
	ProjectID string `json:"project_id"`
	Intensity string `json:"intensity" binding:"required"`
}

type SubmitAnswerRequest struct {
	Answer string `json:"answer" binding:"required"`
}

type StreamEvent struct {
	Type    string `json:"type"`
	Phase   string `json:"phase,omitempty"`
	Name    string `json:"name,omitempty"`
	Text    string `json:"text,omitempty"`
	Message string `json:"message,omitempty"`
	Data    any    `json:"data,omitempty"`
}

type AnalyzeRepoRequest struct {
	RepoURL string `json:"repo_url"`
}

type AnalyzeRepoResponse struct {
	RepoURL         string      `json:"repo_url"`
	Name            string      `json:"name"`
	DefaultBranch   string      `json:"default_branch"`
	ImportCommit    string      `json:"import_commit"`
	Summary         string      `json:"summary"`
	TechStack       []string    `json:"tech_stack"`
	Highlights      []string    `json:"highlights"`
	Challenges      []string    `json:"challenges"`
	Tradeoffs       []string    `json:"tradeoffs"`
	OwnershipPoints []string    `json:"ownership_points"`
	FollowupPoints  []string    `json:"followup_points"`
	Chunks          []RepoChunk `json:"chunks"`
}

type GenerateQuestionRequest struct {
	Mode          string             `json:"mode"`
	Topic         string             `json:"topic,omitempty"`
	Intensity     string             `json:"intensity"`
	Project       *ProjectProfile    `json:"project,omitempty"`
	Templates     []QuestionTemplate `json:"templates,omitempty"`
	ContextChunks []RepoChunk        `json:"context_chunks,omitempty"`
	Weaknesses    []WeaknessTag      `json:"weaknesses,omitempty"`
}

type GenerateQuestionResponse struct {
	Question       string   `json:"question"`
	ExpectedPoints []string `json:"expected_points"`
}

type EvaluateAnswerRequest struct {
	Mode           string          `json:"mode"`
	Topic          string          `json:"topic,omitempty"`
	Project        *ProjectProfile `json:"project,omitempty"`
	Question       string          `json:"question"`
	ExpectedPoints []string        `json:"expected_points"`
	Answer         string          `json:"answer"`
	ContextChunks  []RepoChunk     `json:"context_chunks,omitempty"`
	IsFollowup     bool            `json:"is_followup"`
}

type GenerateReviewRequest struct {
	Session *TrainingSession `json:"session"`
	Project *ProjectProfile  `json:"project,omitempty"`
	Turns   []TrainingTurn   `json:"turns"`
}
