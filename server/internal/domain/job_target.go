package domain

import "time"

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
