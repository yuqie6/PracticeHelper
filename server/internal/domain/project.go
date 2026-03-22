package domain

import "time"

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
