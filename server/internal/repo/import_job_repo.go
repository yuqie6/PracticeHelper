package repo

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"practicehelper/server/internal/domain"
)

func (s *Store) CreateProjectImportJob(ctx context.Context, repoURL string) (*domain.ProjectImportJob, error) {
	jobID := newID("import")
	now := nowUTC()
	if _, err := s.db.ExecContext(ctx, `
		INSERT INTO project_import_jobs (
			id, repo_url, status, stage, message, error_message, project_id, claim_token, claim_expires_at, created_at, updated_at, started_at, finished_at
		) VALUES (?, ?, ?, ?, ?, '', '', '', '', ?, ?, '', '')
	`,
		jobID,
		repoURL,
		domain.ProjectImportStatusQueued,
		domain.ProjectImportStageQueued,
		"任务已创建，等待后台开始导入。",
		now,
		now,
	); err != nil {
		return nil, fmt.Errorf("create import job: %w", err)
	}

	return s.GetProjectImportJob(ctx, jobID)
}

func (s *Store) ClaimProjectImportJob(
	ctx context.Context,
	jobID string,
	claimToken string,
	stage string,
	message string,
	startedAt time.Time,
	claimExpiresAt time.Time,
) (bool, error) {
	now := nowUTC()
	result, err := s.db.ExecContext(ctx, `
		UPDATE project_import_jobs
		SET status = ?, stage = ?, message = ?, error_message = '', project_id = '',
			claim_token = ?, claim_expires_at = ?, started_at = ?, finished_at = '',
			updated_at = ?
		WHERE id = ?
			AND (
				status = ?
				OR (status = ? AND (claim_token = '' OR claim_expires_at = '' OR claim_expires_at <= ?))
			)
	`,
		domain.ProjectImportStatusRunning,
		stage,
		message,
		claimToken,
		claimExpiresAt.UTC().Format(time.RFC3339Nano),
		startedAt.UTC().Format(time.RFC3339Nano),
		now,
		jobID,
		domain.ProjectImportStatusQueued,
		domain.ProjectImportStatusRunning,
		now,
	)
	if err != nil {
		return false, fmt.Errorf("claim import job: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("claim import job rows affected: %w", err)
	}
	return rowsAffected > 0, nil
}

func (s *Store) TouchProjectImportJobClaim(
	ctx context.Context,
	jobID string,
	claimToken string,
	claimExpiresAt time.Time,
) (bool, error) {
	result, err := s.db.ExecContext(ctx, `
		UPDATE project_import_jobs
		SET claim_expires_at = ?, updated_at = ?
		WHERE id = ? AND status = ? AND claim_token = ?
	`,
		claimExpiresAt.UTC().Format(time.RFC3339Nano),
		nowUTC(),
		jobID,
		domain.ProjectImportStatusRunning,
		claimToken,
	)
	if err != nil {
		return false, fmt.Errorf("touch import job claim: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("touch import job claim rows affected: %w", err)
	}
	return rowsAffected > 0, nil
}

func (s *Store) AdvanceClaimedProjectImportJob(
	ctx context.Context,
	jobID string,
	claimToken string,
	stage string,
	message string,
	claimExpiresAt time.Time,
) (bool, error) {
	result, err := s.db.ExecContext(ctx, `
		UPDATE project_import_jobs
		SET stage = ?, message = ?, claim_expires_at = ?, updated_at = ?
		WHERE id = ? AND status = ? AND claim_token = ?
	`,
		stage,
		message,
		claimExpiresAt.UTC().Format(time.RFC3339Nano),
		nowUTC(),
		jobID,
		domain.ProjectImportStatusRunning,
		claimToken,
	)
	if err != nil {
		return false, fmt.Errorf("advance claimed import job: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("advance claimed import job rows affected: %w", err)
	}
	return rowsAffected > 0, nil
}

func (s *Store) FinishClaimedProjectImportJob(
	ctx context.Context,
	jobID string,
	claimToken string,
	status string,
	stage string,
	message string,
	errorMessage string,
	projectID string,
	finishedAt time.Time,
) (bool, error) {
	result, err := s.db.ExecContext(ctx, `
		UPDATE project_import_jobs
		SET status = ?, stage = ?, message = ?, error_message = ?, project_id = ?,
			finished_at = ?, claim_token = '', claim_expires_at = '', updated_at = ?
		WHERE id = ? AND status = ? AND claim_token = ?
	`,
		status,
		stage,
		message,
		errorMessage,
		projectID,
		finishedAt.UTC().Format(time.RFC3339Nano),
		nowUTC(),
		jobID,
		domain.ProjectImportStatusRunning,
		claimToken,
	)
	if err != nil {
		return false, fmt.Errorf("finish claimed import job: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("finish claimed import job rows affected: %w", err)
	}
	return rowsAffected > 0, nil
}

func (s *Store) UpdateProjectImportJobStatus(
	ctx context.Context,
	jobID string,
	status string,
	stage string,
	message string,
	errorMessage string,
	projectID string,
	startedAt *time.Time,
	finishedAt *time.Time,
) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE project_import_jobs
		SET status = ?, stage = ?, message = ?, error_message = ?, project_id = ?,
			started_at = CASE WHEN ? != '' THEN ? ELSE started_at END,
			finished_at = CASE WHEN ? != '' THEN ? ELSE finished_at END,
			updated_at = ?
		WHERE id = ?
	`,
		status,
		stage,
		message,
		errorMessage,
		projectID,
		toNullableTimeString(startedAt),
		toNullableTimeString(startedAt),
		toNullableTimeString(finishedAt),
		toNullableTimeString(finishedAt),
		nowUTC(),
		jobID,
	)
	if err != nil {
		return fmt.Errorf("update import job: %w", err)
	}

	return nil
}

func (s *Store) RetryProjectImportJob(ctx context.Context, jobID string, message string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE project_import_jobs
		SET status = ?, stage = ?, message = ?, error_message = '', project_id = '',
			claim_token = '', claim_expires_at = '',
			started_at = '', finished_at = '', updated_at = ?
		WHERE id = ?
	`,
		domain.ProjectImportStatusQueued,
		domain.ProjectImportStageQueued,
		message,
		nowUTC(),
		jobID,
	)
	if err != nil {
		return fmt.Errorf("retry import job: %w", err)
	}

	return nil
}

func (s *Store) ListProjectImportJobs(ctx context.Context, limit int) ([]domain.ProjectImportJob, error) {
	if limit <= 0 {
		limit = 20
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT j.id, j.repo_url, j.status, j.stage, j.message, j.error_message, j.project_id,
			COALESCE(p.name, ''), j.created_at, j.updated_at, j.started_at, j.finished_at
		FROM project_import_jobs j
		LEFT JOIN project_profiles p ON p.id = j.project_id
		ORDER BY j.updated_at DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("list import jobs: %w", err)
	}
	defer func() { _ = rows.Close() }()

	jobs := make([]domain.ProjectImportJob, 0)
	for rows.Next() {
		job, err := scanProjectImportJob(rows)
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, *job)
	}

	return jobs, nil
}

func (s *Store) GetProjectImportJob(ctx context.Context, jobID string) (*domain.ProjectImportJob, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT j.id, j.repo_url, j.status, j.stage, j.message, j.error_message, j.project_id,
			COALESCE(p.name, ''), j.created_at, j.updated_at, j.started_at, j.finished_at
		FROM project_import_jobs j
		LEFT JOIN project_profiles p ON p.id = j.project_id
		WHERE j.id = ?
	`, jobID)
	job, err := scanProjectImportJob(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return job, nil
}

func (s *Store) FindActiveProjectImportJobByRepoURL(ctx context.Context, repoURL string) (*domain.ProjectImportJob, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT j.id, j.repo_url, j.status, j.stage, j.message, j.error_message, j.project_id,
			COALESCE(p.name, ''), j.created_at, j.updated_at, j.started_at, j.finished_at
		FROM project_import_jobs j
		LEFT JOIN project_profiles p ON p.id = j.project_id
		WHERE j.repo_url = ? AND j.status IN (?, ?)
		ORDER BY j.updated_at DESC
		LIMIT 1
	`, repoURL, domain.ProjectImportStatusQueued, domain.ProjectImportStatusRunning)
	job, err := scanProjectImportJob(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return job, nil
}

func (s *Store) ListPendingProjectImportJobs(ctx context.Context) ([]domain.ProjectImportJob, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT j.id, j.repo_url, j.status, j.stage, j.message, j.error_message, j.project_id,
			COALESCE(p.name, ''), j.created_at, j.updated_at, j.started_at, j.finished_at
		FROM project_import_jobs j
		LEFT JOIN project_profiles p ON p.id = j.project_id
		WHERE j.status IN (?, ?)
		ORDER BY j.created_at ASC
	`, domain.ProjectImportStatusQueued, domain.ProjectImportStatusRunning)
	if err != nil {
		return nil, fmt.Errorf("list pending import jobs: %w", err)
	}
	defer func() { _ = rows.Close() }()

	jobs := make([]domain.ProjectImportJob, 0)
	for rows.Next() {
		job, err := scanProjectImportJob(rows)
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, *job)
	}

	return jobs, nil
}
