package service

import (
	"context"
	"fmt"
	"strings"

	"practicehelper/server/internal/domain"
)

func (s *Service) ListJobTargets(ctx context.Context) ([]domain.JobTarget, error) {
	return s.repo.ListJobTargets(ctx)
}

func (s *Service) GetJobTarget(ctx context.Context, jobTargetID string) (*domain.JobTarget, error) {
	target, err := s.repo.GetJobTarget(ctx, jobTargetID)
	if err != nil {
		return nil, err
	}
	if target == nil {
		return nil, ErrJobTargetNotFound
	}
	return target, nil
}

func (s *Service) CreateJobTarget(ctx context.Context, input domain.JobTargetInput) (*domain.JobTarget, error) {
	return s.repo.CreateJobTarget(ctx, normalizeJobTargetInput(input))
}

func (s *Service) UpdateJobTarget(
	ctx context.Context,
	jobTargetID string,
	input domain.JobTargetInput,
) (*domain.JobTarget, error) {
	target, err := s.repo.UpdateJobTarget(ctx, jobTargetID, normalizeJobTargetInput(input))
	if err != nil {
		return nil, err
	}
	if target == nil {
		return nil, ErrJobTargetNotFound
	}
	return target, nil
}

func (s *Service) AnalyzeJobTarget(
	ctx context.Context,
	jobTargetID string,
) (*domain.JobTargetAnalysisRun, error) {
	target, err := s.repo.GetJobTarget(ctx, jobTargetID)
	if err != nil {
		return nil, err
	}
	if target == nil {
		return nil, ErrJobTargetNotFound
	}

	run, err := s.repo.StartJobTargetAnalysis(ctx, target.ID, target.SourceText)
	if err != nil {
		return nil, err
	}

	analysis, err := s.sidecar.AnalyzeJobTarget(ctx, domain.AnalyzeJobTargetRequest{
		Title:       target.Title,
		CompanyName: target.CompanyName,
		SourceText:  target.SourceText,
	})
	if err != nil {
		_ = s.repo.FailJobTargetAnalysis(ctx, target.ID, run.ID, err.Error())
		return nil, err
	}

	if err := s.repo.CompleteJobTargetAnalysis(ctx, target.ID, run.ID, analysis); err != nil {
		return nil, err
	}

	updatedRun, err := s.repo.GetJobTargetAnalysisRun(ctx, run.ID)
	if err != nil {
		return nil, err
	}
	if updatedRun == nil {
		return nil, fmt.Errorf("job target analysis %s missing after completion", run.ID)
	}
	return updatedRun, nil
}

func (s *Service) ListJobTargetAnalysisRuns(
	ctx context.Context,
	jobTargetID string,
) ([]domain.JobTargetAnalysisRun, error) {
	target, err := s.repo.GetJobTarget(ctx, jobTargetID)
	if err != nil {
		return nil, err
	}
	if target == nil {
		return nil, ErrJobTargetNotFound
	}
	return s.repo.ListJobTargetAnalysisRuns(ctx, jobTargetID)
}

func (s *Service) GetJobTargetAnalysisRun(
	ctx context.Context,
	runID string,
) (*domain.JobTargetAnalysisRun, error) {
	run, err := s.repo.GetJobTargetAnalysisRun(ctx, runID)
	if err != nil {
		return nil, err
	}
	if run == nil {
		return nil, ErrJobTargetAnalysisNotFound
	}
	return run, nil
}

func (s *Service) resolveJobTargetBinding(
	ctx context.Context,
	jobTargetID string,
) (*domain.JobTarget, *domain.JobTargetAnalysisRun, error) {
	jobTargetID = strings.TrimSpace(jobTargetID)
	if jobTargetID == "" {
		return nil, nil, nil
	}

	target, err := s.repo.GetJobTarget(ctx, jobTargetID)
	if err != nil {
		return nil, nil, err
	}
	if target == nil {
		return nil, nil, ErrJobTargetNotFound
	}
	if target.LatestAnalysisStatus != domain.JobTargetAnalysisSucceeded || target.LatestAnalysisID == "" {
		return nil, nil, ErrJobTargetNotReady
	}

	analysis, err := s.repo.GetJobTargetAnalysisRun(ctx, target.LatestAnalysisID)
	if err != nil {
		return nil, nil, err
	}
	if analysis == nil || analysis.Status != domain.JobTargetAnalysisSucceeded {
		return nil, nil, ErrJobTargetNotReady
	}

	if err := s.repo.MarkJobTargetUsed(ctx, target.ID); err != nil {
		return nil, nil, err
	}

	return target, analysis, nil
}

func (s *Service) getJobTargetAnalysisSnapshotForSession(
	ctx context.Context,
	session *domain.TrainingSession,
) (*domain.AnalyzeJobTargetResponse, error) {
	if session == nil || session.JobTargetAnalysisID == "" {
		return nil, nil
	}

	run, err := s.repo.GetJobTargetAnalysisRun(ctx, session.JobTargetAnalysisID)
	if err != nil {
		return nil, err
	}
	if run == nil {
		return nil, ErrJobTargetAnalysisNotFound
	}
	return buildJobTargetAnalysisSnapshot(run), nil
}

func buildJobTargetAnalysisSnapshot(
	run *domain.JobTargetAnalysisRun,
) *domain.AnalyzeJobTargetResponse {
	if run == nil {
		return nil
	}
	return &domain.AnalyzeJobTargetResponse{
		Summary:          run.Summary,
		MustHaveSkills:   run.MustHaveSkills,
		BonusSkills:      run.BonusSkills,
		Responsibilities: run.Responsibilities,
		EvaluationFocus:  run.EvaluationFocus,
	}
}

func normalizeJobTargetInput(input domain.JobTargetInput) domain.JobTargetInput {
	input.Title = strings.TrimSpace(input.Title)
	input.CompanyName = strings.TrimSpace(input.CompanyName)
	input.SourceText = strings.TrimSpace(input.SourceText)
	return input
}
