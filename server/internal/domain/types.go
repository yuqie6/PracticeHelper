package domain

import "time"

const (
	ModeBasics  = "basics"
	ModeProject = "project"

	StatusDraft         = "draft"
	StatusActive        = "active"
	StatusWaitingAnswer = "waiting_answer"
	StatusEvaluating    = "evaluating"
	StatusReviewPending = "review_pending"
	StatusCompleted     = "completed"

	ProjectImportStatusQueued    = "queued"
	ProjectImportStatusRunning   = "running"
	ProjectImportStatusCompleted = "completed"
	ProjectImportStatusFailed    = "failed"

	JobTargetAnalysisIdle      = "idle"
	JobTargetAnalysisRunning   = "running"
	JobTargetAnalysisSucceeded = "succeeded"
	JobTargetAnalysisFailed    = "failed"
	JobTargetAnalysisStale     = "stale"

	ProjectImportStageQueued     = "queued"
	ProjectImportStageAnalyzing  = "analyzing_repository"
	ProjectImportStagePersisting = "persisting_project"
	ProjectImportStageCompleted  = "completed"
	ProjectImportStageFailed     = "failed"
)

type UserProfile struct {
	ID                   int64         `json:"id"`
	TargetRole           string        `json:"target_role"`
	TargetCompanyType    string        `json:"target_company_type"`
	CurrentStage         string        `json:"current_stage"`
	ApplicationDeadline  *time.Time    `json:"application_deadline,omitempty"`
	TechStacks           []string      `json:"tech_stacks"`
	PrimaryProjects      []string      `json:"primary_projects"`
	SelfReportedWeakness []string      `json:"self_reported_weaknesses"`
	ActiveJobTargetID    string        `json:"active_job_target_id,omitempty"`
	CreatedAt            time.Time     `json:"created_at"`
	UpdatedAt            time.Time     `json:"updated_at"`
	ActiveJobTarget      *JobTargetRef `json:"active_job_target,omitempty"`
}

type UserProfileInput struct {
	TargetRole           string   `json:"target_role" binding:"required"`
	TargetCompanyType    string   `json:"target_company_type" binding:"required"`
	CurrentStage         string   `json:"current_stage" binding:"required"`
	ApplicationDeadline  *string  `json:"application_deadline"`
	TechStacks           []string `json:"tech_stacks"`
	PrimaryProjects      []string `json:"primary_projects"`
	SelfReportedWeakness []string `json:"self_reported_weaknesses"`
	ActiveJobTargetID    string   `json:"active_job_target_id,omitempty"`
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

type JobTargetRef struct {
	ID                   string `json:"id"`
	Title                string `json:"title"`
	CompanyName          string `json:"company_name,omitempty"`
	LatestAnalysisStatus string `json:"latest_analysis_status,omitempty"`
}

type JobTarget struct {
	ID                       string                `json:"id"`
	Title                    string                `json:"title"`
	CompanyName              string                `json:"company_name,omitempty"`
	SourceText               string                `json:"source_text"`
	LatestAnalysisID         string                `json:"latest_analysis_id,omitempty"`
	LatestAnalysisStatus     string                `json:"latest_analysis_status"`
	LastUsedAt               *time.Time            `json:"last_used_at,omitempty"`
	CreatedAt                time.Time             `json:"created_at"`
	UpdatedAt                time.Time             `json:"updated_at"`
	LatestSuccessfulAnalysis *JobTargetAnalysisRun `json:"latest_successful_analysis,omitempty"`
}

type JobTargetInput struct {
	Title       string `json:"title" binding:"required"`
	CompanyName string `json:"company_name"`
	SourceText  string `json:"source_text" binding:"required"`
}

type JobTargetAnalysisRun struct {
	ID                 string     `json:"id"`
	JobTargetID        string     `json:"job_target_id"`
	SourceTextSnapshot string     `json:"source_text_snapshot"`
	Status             string     `json:"status"`
	ErrorMessage       string     `json:"error_message,omitempty"`
	Summary            string     `json:"summary,omitempty"`
	MustHaveSkills     []string   `json:"must_have_skills,omitempty"`
	BonusSkills        []string   `json:"bonus_skills,omitempty"`
	Responsibilities   []string   `json:"responsibilities,omitempty"`
	EvaluationFocus    []string   `json:"evaluation_focus,omitempty"`
	CreatedAt          time.Time  `json:"created_at"`
	FinishedAt         *time.Time `json:"finished_at,omitempty"`
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

type ProjectImportJob struct {
	ID           string     `json:"id"`
	RepoURL      string     `json:"repo_url"`
	Status       string     `json:"status"`
	Stage        string     `json:"stage"`
	Message      string     `json:"message"`
	ErrorMessage string     `json:"error_message,omitempty"`
	ProjectID    string     `json:"project_id,omitempty"`
	ProjectName  string     `json:"project_name,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	StartedAt    *time.Time `json:"started_at,omitempty"`
	FinishedAt   *time.Time `json:"finished_at,omitempty"`
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
	ID                  string          `json:"id"`
	Mode                string          `json:"mode"`
	Topic               string          `json:"topic,omitempty"`
	ProjectID           string          `json:"project_id,omitempty"`
	JobTargetID         string          `json:"job_target_id,omitempty"`
	JobTargetAnalysisID string          `json:"job_target_analysis_id,omitempty"`
	Intensity           string          `json:"intensity"`
	Status              string          `json:"status"`
	MaxTurns            int             `json:"max_turns"`
	TotalScore          float64         `json:"total_score"`
	StartedAt           *time.Time      `json:"started_at,omitempty"`
	EndedAt             *time.Time      `json:"ended_at,omitempty"`
	CreatedAt           time.Time       `json:"created_at"`
	UpdatedAt           time.Time       `json:"updated_at"`
	ReviewID            string          `json:"review_id,omitempty"`
	Turns               []TrainingTurn  `json:"turns,omitempty"`
	Project             *ProjectProfile `json:"project,omitempty"`
	JobTarget           *JobTargetRef   `json:"job_target,omitempty"`
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
	Overall             string             `json:"overall"`
	TopFix              string             `json:"top_fix,omitempty"`
	TopFixReason        string             `json:"top_fix_reason,omitempty"`
	Highlights          []string           `json:"highlights"`
	Gaps                []string           `json:"gaps"`
	SuggestedTopics     []string           `json:"suggested_topics"`
	NextTrainingFocus   []string           `json:"next_training_focus"`
	RecommendedNext     *NextSession       `json:"recommended_next,omitempty"`
	ScoreBreakdown      map[string]float64 `json:"score_breakdown"`
	CreatedAt           time.Time          `json:"created_at"`
	JobTarget           *JobTargetRef      `json:"job_target,omitempty"`
}

type NextSession struct {
	Mode      string `json:"mode"`
	Topic     string `json:"topic,omitempty"`
	ProjectID string `json:"project_id,omitempty"`
	Reason    string `json:"reason,omitempty"`
}

type TrainingSessionSummary struct {
	ID          string        `json:"id"`
	Mode        string        `json:"mode"`
	Topic       string        `json:"topic,omitempty"`
	ProjectName string        `json:"project_name,omitempty"`
	Status      string        `json:"status"`
	TotalScore  float64       `json:"total_score"`
	ReviewID    string        `json:"review_id,omitempty"`
	UpdatedAt   time.Time     `json:"updated_at"`
	JobTarget   *JobTargetRef `json:"job_target,omitempty"`
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

type Dashboard struct {
	Profile             *UserProfile             `json:"profile"`
	Weaknesses          []WeaknessTag            `json:"weaknesses"`
	RecentSessions      []TrainingSessionSummary `json:"recent_sessions"`
	CurrentSession      *TrainingSessionSummary  `json:"current_session,omitempty"`
	TodayFocus          string                   `json:"today_focus"`
	RecommendedTrack    string                   `json:"recommended_track"`
	ActiveJobTarget     *JobTargetRef            `json:"active_job_target,omitempty"`
	RecommendationScope string                   `json:"recommendation_scope"`
	DaysUntilDeadline   *int                     `json:"days_until_deadline,omitempty"`
}

type CreateSessionRequest struct {
	Mode                  string `json:"mode" binding:"required"`
	Topic                 string `json:"topic"`
	ProjectID             string `json:"project_id"`
	JobTargetID           string `json:"job_target_id"`
	IgnoreActiveJobTarget bool   `json:"ignore_active_job_target,omitempty"`
	Intensity             string `json:"intensity" binding:"required"`
	MaxTurns              int    `json:"max_turns,omitempty"`
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

type AnalyzeRepoRequest struct {
	RepoURL string `json:"repo_url"`
}

type AnalyzeJobTargetRequest struct {
	Title       string `json:"title,omitempty"`
	CompanyName string `json:"company_name,omitempty"`
	SourceText  string `json:"source_text"`
}

type AnalyzeJobTargetResponse struct {
	Summary          string   `json:"summary"`
	MustHaveSkills   []string `json:"must_have_skills"`
	BonusSkills      []string `json:"bonus_skills"`
	Responsibilities []string `json:"responsibilities"`
	EvaluationFocus  []string `json:"evaluation_focus"`
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
	Mode              string                    `json:"mode"`
	Topic             string                    `json:"topic,omitempty"`
	Intensity         string                    `json:"intensity"`
	Project           *ProjectProfile           `json:"project,omitempty"`
	Templates         []QuestionTemplate        `json:"templates,omitempty"`
	ContextChunks     []RepoChunk               `json:"context_chunks,omitempty"`
	Weaknesses        []WeaknessTag             `json:"weaknesses,omitempty"`
	JobTargetAnalysis *AnalyzeJobTargetResponse `json:"job_target_analysis,omitempty"`
}

type GenerateQuestionResponse struct {
	Question       string   `json:"question"`
	ExpectedPoints []string `json:"expected_points"`
}

type EvaluateAnswerRequest struct {
	Mode              string                    `json:"mode"`
	Topic             string                    `json:"topic,omitempty"`
	Project           *ProjectProfile           `json:"project,omitempty"`
	Question          string                    `json:"question"`
	ExpectedPoints    []string                  `json:"expected_points"`
	Answer            string                    `json:"answer"`
	ContextChunks     []RepoChunk               `json:"context_chunks,omitempty"`
	TurnIndex         int                       `json:"turn_index"`
	MaxTurns          int                       `json:"max_turns"`
	ScoreWeights      map[string]float64        `json:"score_weights,omitempty"`
	JobTargetAnalysis *AnalyzeJobTargetResponse `json:"job_target_analysis,omitempty"`
}

type GenerateReviewRequest struct {
	Session           *TrainingSession          `json:"session"`
	Project           *ProjectProfile           `json:"project,omitempty"`
	Turns             []TrainingTurn            `json:"turns"`
	JobTargetAnalysis *AnalyzeJobTargetResponse `json:"job_target_analysis,omitempty"`
}
