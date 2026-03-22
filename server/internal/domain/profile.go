package domain

import "time"

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

type ProfileSnapshot struct {
	TargetRole           string        `json:"target_role,omitempty"`
	TargetCompanyType    string        `json:"target_company_type,omitempty"`
	CurrentStage         string        `json:"current_stage,omitempty"`
	ApplicationDeadline  *time.Time    `json:"application_deadline,omitempty"`
	TechStacks           []string      `json:"tech_stacks,omitempty"`
	PrimaryProjects      []string      `json:"primary_projects,omitempty"`
	SelfReportedWeakness []string      `json:"self_reported_weaknesses,omitempty"`
	ActiveJobTarget      *JobTargetRef `json:"active_job_target,omitempty"`
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
