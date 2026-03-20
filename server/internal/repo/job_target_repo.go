package repo

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"practicehelper/server/internal/domain"
)

func (s *Store) ListJobTargets(ctx context.Context) ([]domain.JobTarget, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, title, company_name, source_text, latest_analysis_id, latest_analysis_status, last_used_at, created_at, updated_at
		FROM job_targets
		ORDER BY updated_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("list job targets: %w", err)
	}
	defer func() { _ = rows.Close() }()

	items := make([]domain.JobTarget, 0)
	for rows.Next() {
		target, err := scanJobTarget(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *target)
	}

	return items, nil
}

func (s *Store) CreateJobTarget(ctx context.Context, input domain.JobTargetInput) (*domain.JobTarget, error) {
	targetID := newID("jt")
	now := nowUTC()

	if _, err := s.db.ExecContext(ctx, `
		INSERT INTO job_targets (
			id, title, company_name, source_text, latest_analysis_id, latest_analysis_status, last_used_at, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		targetID,
		input.Title,
		input.CompanyName,
		input.SourceText,
		"",
		domain.JobTargetAnalysisIdle,
		"",
		now,
		now,
	); err != nil {
		return nil, fmt.Errorf("create job target: %w", err)
	}

	return s.GetJobTarget(ctx, targetID)
}

func (s *Store) GetJobTarget(ctx context.Context, jobTargetID string) (*domain.JobTarget, error) {
	target, err := s.getJobTargetRow(ctx, jobTargetID)
	if err != nil || target == nil {
		return target, err
	}

	latestSuccessful, err := s.GetLatestSuccessfulJobTargetAnalysis(ctx, jobTargetID)
	if err != nil {
		return nil, err
	}
	target.LatestSuccessfulAnalysis = latestSuccessful
	return target, nil
}

func (s *Store) UpdateJobTarget(ctx context.Context, jobTargetID string, input domain.JobTargetInput) (*domain.JobTarget, error) {
	existing, err := s.getJobTargetRow(ctx, jobTargetID)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, nil
	}

	nextStatus := existing.LatestAnalysisStatus
	if input.SourceText != existing.SourceText {
		if existing.LatestAnalysisID == "" {
			nextStatus = domain.JobTargetAnalysisIdle
		} else {
			nextStatus = domain.JobTargetAnalysisStale
		}
	}

	if _, err := s.db.ExecContext(ctx, `
		UPDATE job_targets
		SET title = ?, company_name = ?, source_text = ?, latest_analysis_status = ?, updated_at = ?
		WHERE id = ?
	`,
		input.Title,
		input.CompanyName,
		input.SourceText,
		nextStatus,
		nowUTC(),
		jobTargetID,
	); err != nil {
		return nil, fmt.Errorf("update job target: %w", err)
	}

	return s.GetJobTarget(ctx, jobTargetID)
}

func (s *Store) StartJobTargetAnalysis(
	ctx context.Context,
	jobTargetID string,
	sourceTextSnapshot string,
) (*domain.JobTargetAnalysisRun, error) {
	runID := newID("jta")
	now := nowUTC()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin job target analysis: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO job_target_analysis_runs (
			id, job_target_id, source_text_snapshot, status, error_message, summary,
			must_have_skills_json, bonus_skills_json, responsibilities_json, evaluation_focus_json,
			created_at, finished_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		runID,
		jobTargetID,
		sourceTextSnapshot,
		domain.JobTargetAnalysisRunning,
		"",
		"",
		"[]",
		"[]",
		"[]",
		"[]",
		now,
		"",
	); err != nil {
		return nil, fmt.Errorf("insert job target analysis run: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `
		UPDATE job_targets
		SET latest_analysis_id = ?, latest_analysis_status = ?, updated_at = ?
		WHERE id = ?
	`,
		runID,
		domain.JobTargetAnalysisRunning,
		now,
		jobTargetID,
	); err != nil {
		return nil, fmt.Errorf("mark job target analysis running: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit job target analysis start: %w", err)
	}

	return s.GetJobTargetAnalysisRun(ctx, runID)
}

func (s *Store) CompleteJobTargetAnalysis(
	ctx context.Context,
	jobTargetID string,
	runID string,
	analysis *domain.AnalyzeJobTargetResponse,
) error {
	finishedAt := nowUTC()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin complete job target analysis: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, `
		UPDATE job_target_analysis_runs
		SET status = ?, error_message = ?, summary = ?, must_have_skills_json = ?, bonus_skills_json = ?,
			responsibilities_json = ?, evaluation_focus_json = ?, finished_at = ?
		WHERE id = ?
	`,
		domain.JobTargetAnalysisSucceeded,
		"",
		analysis.Summary,
		mustJSON(analysis.MustHaveSkills),
		mustJSON(analysis.BonusSkills),
		mustJSON(analysis.Responsibilities),
		mustJSON(analysis.EvaluationFocus),
		finishedAt,
		runID,
	); err != nil {
		return fmt.Errorf("complete job target analysis run: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `
		UPDATE job_targets
		SET latest_analysis_id = ?, latest_analysis_status = ?, updated_at = ?
		WHERE id = ?
			AND latest_analysis_id = ?
			AND source_text = (
				SELECT source_text_snapshot
				FROM job_target_analysis_runs
				WHERE id = ?
			)
	`,
		runID,
		domain.JobTargetAnalysisSucceeded,
		finishedAt,
		jobTargetID,
		runID,
		runID,
	); err != nil {
		return fmt.Errorf("mark job target analysis succeeded: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit complete job target analysis: %w", err)
	}

	return nil
}

func (s *Store) FailJobTargetAnalysis(
	ctx context.Context,
	jobTargetID string,
	runID string,
	errorMessage string,
) error {
	finishedAt := nowUTC()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin fail job target analysis: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, `
		UPDATE job_target_analysis_runs
		SET status = ?, error_message = ?, finished_at = ?
		WHERE id = ?
	`,
		domain.JobTargetAnalysisFailed,
		errorMessage,
		finishedAt,
		runID,
	); err != nil {
		return fmt.Errorf("fail job target analysis run: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `
		UPDATE job_targets
		SET latest_analysis_id = ?, latest_analysis_status = ?, updated_at = ?
		WHERE id = ?
			AND latest_analysis_id = ?
			AND source_text = (
				SELECT source_text_snapshot
				FROM job_target_analysis_runs
				WHERE id = ?
			)
	`,
		runID,
		domain.JobTargetAnalysisFailed,
		finishedAt,
		jobTargetID,
		runID,
		runID,
	); err != nil {
		return fmt.Errorf("mark job target analysis failed: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit fail job target analysis: %w", err)
	}

	return nil
}

func (s *Store) ListJobTargetAnalysisRuns(
	ctx context.Context,
	jobTargetID string,
) ([]domain.JobTargetAnalysisRun, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, job_target_id, source_text_snapshot, status, error_message, summary,
			must_have_skills_json, bonus_skills_json, responsibilities_json, evaluation_focus_json,
			created_at, finished_at
		FROM job_target_analysis_runs
		WHERE job_target_id = ?
		ORDER BY created_at DESC
	`, jobTargetID)
	if err != nil {
		return nil, fmt.Errorf("list job target analysis runs: %w", err)
	}
	defer func() { _ = rows.Close() }()

	items := make([]domain.JobTargetAnalysisRun, 0)
	for rows.Next() {
		run, err := scanJobTargetAnalysisRun(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *run)
	}

	return items, nil
}

func (s *Store) GetJobTargetAnalysisRun(ctx context.Context, runID string) (*domain.JobTargetAnalysisRun, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, job_target_id, source_text_snapshot, status, error_message, summary,
			must_have_skills_json, bonus_skills_json, responsibilities_json, evaluation_focus_json,
			created_at, finished_at
		FROM job_target_analysis_runs
		WHERE id = ?
	`, runID)
	run, err := scanJobTargetAnalysisRun(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return run, nil
}

func (s *Store) GetLatestSuccessfulJobTargetAnalysis(
	ctx context.Context,
	jobTargetID string,
) (*domain.JobTargetAnalysisRun, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, job_target_id, source_text_snapshot, status, error_message, summary,
			must_have_skills_json, bonus_skills_json, responsibilities_json, evaluation_focus_json,
			created_at, finished_at
		FROM job_target_analysis_runs
		WHERE job_target_id = ? AND status = ?
		ORDER BY created_at DESC
		LIMIT 1
	`, jobTargetID, domain.JobTargetAnalysisSucceeded)
	run, err := scanJobTargetAnalysisRun(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return run, nil
}

func (s *Store) MarkJobTargetUsed(ctx context.Context, jobTargetID string) error {
	if _, err := s.db.ExecContext(ctx, `
		UPDATE job_targets
		SET last_used_at = ?
		WHERE id = ?
	`, nowUTC(), jobTargetID); err != nil {
		return fmt.Errorf("mark job target used: %w", err)
	}

	return nil
}

func (s *Store) getJobTargetRow(ctx context.Context, jobTargetID string) (*domain.JobTarget, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, title, company_name, source_text, latest_analysis_id, latest_analysis_status, last_used_at, created_at, updated_at
		FROM job_targets
		WHERE id = ?
	`, jobTargetID)
	target, err := scanJobTarget(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get job target: %w", err)
	}

	return target, nil
}
