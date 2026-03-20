package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"practicehelper/server/internal/domain"
	"practicehelper/server/internal/repo"
)

func (s *Service) ImportProject(ctx context.Context, request domain.ProjectImportRequest) (*domain.ProjectImportJob, error) {
	repoURL := strings.TrimSpace(request.RepoURL)
	if repoURL == "" {
		return nil, errors.New("repo_url is required")
	}

	project, err := s.repo.GetProjectByRepoURL(ctx, repoURL)
	if err != nil {
		return nil, err
	}
	if project != nil {
		return nil, repo.ErrAlreadyImported
	}

	job, err := s.repo.FindActiveProjectImportJobByRepoURL(ctx, repoURL)
	if err != nil {
		return nil, err
	}
	if job != nil {
		return job, nil
	}

	job, err = s.repo.CreateProjectImportJob(ctx, repoURL)
	if err != nil {
		return nil, err
	}

	s.startImportJob(*job)
	return job, nil
}

func (s *Service) ListProjectImportJobs(ctx context.Context) ([]domain.ProjectImportJob, error) {
	return s.repo.ListProjectImportJobs(ctx, 20)
}

func (s *Service) GetProjectImportJob(ctx context.Context, jobID string) (*domain.ProjectImportJob, error) {
	return s.repo.GetProjectImportJob(ctx, jobID)
}

func (s *Service) RetryProjectImportJob(ctx context.Context, jobID string) (*domain.ProjectImportJob, error) {
	job, err := s.repo.GetProjectImportJob(ctx, jobID)
	if err != nil {
		return nil, err
	}
	if job == nil {
		return nil, ErrImportJobNotFound
	}

	if job.Status == domain.ProjectImportStatusCompleted || job.Status == domain.ProjectImportStatusRunning || job.Status == domain.ProjectImportStatusQueued {
		return job, nil
	}

	if err := s.repo.RetryProjectImportJob(ctx, jobID, "任务已重新排队，等待后台再次导入。"); err != nil {
		return nil, err
	}

	updatedJob, err := s.repo.GetProjectImportJob(ctx, jobID)
	if err != nil {
		return nil, err
	}
	if updatedJob == nil {
		return nil, ErrImportJobNotFound
	}

	s.startImportJob(*updatedJob)
	return updatedJob, nil
}

func (s *Service) startImportJob(job domain.ProjectImportJob) {
	go func() {
		backgroundCtx := context.Background()
		startedAt := time.Now().UTC()

		if err := s.repo.UpdateProjectImportJobStatus(
			backgroundCtx,
			job.ID,
			domain.ProjectImportStatusRunning,
			domain.ProjectImportStageAnalyzing,
			"正在克隆仓库、提取关键文件并生成项目画像。",
			"",
			"",
			&startedAt,
			nil,
		); err != nil {
			slog.Error("update import job to running failed", "job_id", job.ID, "error", err)
			return
		}

		analysis, err := s.sidecar.AnalyzeRepo(backgroundCtx, domain.AnalyzeRepoRequest{RepoURL: job.RepoURL})
		if err != nil {
			s.failImportJob(backgroundCtx, job.ID, fmt.Sprintf("项目导入失败：%v", err))
			return
		}

		if err := s.repo.UpdateProjectImportJobStatus(
			backgroundCtx,
			job.ID,
			domain.ProjectImportStatusRunning,
			domain.ProjectImportStagePersisting,
			"正在写入项目画像、源码片段和检索索引。",
			"",
			"",
			nil,
			nil,
		); err != nil {
			slog.Error("update import job to persisting failed", "job_id", job.ID, "error", err)
			return
		}

		project, err := s.repo.CreateImportedProject(backgroundCtx, analysis)
		if errors.Is(err, repo.ErrAlreadyImported) {
			project, err = s.repo.GetProjectByRepoURL(backgroundCtx, job.RepoURL)
			if err == nil && project != nil {
				finishedAt := time.Now().UTC()
				if updateErr := s.repo.UpdateProjectImportJobStatus(
					backgroundCtx,
					job.ID,
					domain.ProjectImportStatusCompleted,
					domain.ProjectImportStageCompleted,
					"仓库已存在，已复用已有项目材料。",
					"",
					project.ID,
					nil,
					&finishedAt,
				); updateErr != nil {
					slog.Error("complete import job with existing project failed", "job_id", job.ID, "error", updateErr)
				}
				return
			}
		}
		if err != nil {
			s.failImportJob(backgroundCtx, job.ID, fmt.Sprintf("项目画像落库失败：%v", err))
			return
		}

		finishedAt := time.Now().UTC()
		if err := s.repo.UpdateProjectImportJobStatus(
			backgroundCtx,
			job.ID,
			domain.ProjectImportStatusCompleted,
			domain.ProjectImportStageCompleted,
			"项目材料已准备好，可以开始编辑和训练。",
			"",
			project.ID,
			nil,
			&finishedAt,
		); err != nil {
			slog.Error("complete import job failed", "job_id", job.ID, "project_id", project.ID, "error", err)
		}
	}()
}

func (s *Service) failImportJob(ctx context.Context, jobID string, message string) {
	finishedAt := time.Now().UTC()
	if err := s.repo.UpdateProjectImportJobStatus(
		ctx,
		jobID,
		domain.ProjectImportStatusFailed,
		domain.ProjectImportStageFailed,
		"导入失败，请检查仓库地址、LLM 配置或稍后重试。",
		message,
		"",
		nil,
		&finishedAt,
	); err != nil {
		slog.Error("mark import job failed error", "job_id", jobID, "error", err)
	}
}

func (s *Service) resumePendingImportJobs() {
	jobs, err := s.repo.ListPendingProjectImportJobs(context.Background())
	if err != nil {
		slog.Error("list pending import jobs failed", "error", err)
		return
	}

	for _, job := range jobs {
		s.startImportJob(job)
	}
}

func (s *Service) ListProjects(ctx context.Context) ([]domain.ProjectProfile, error) {
	return s.repo.ListProjects(ctx)
}

func (s *Service) GetProject(ctx context.Context, projectID string) (*domain.ProjectProfile, error) {
	return s.repo.GetProject(ctx, projectID)
}

func (s *Service) UpdateProject(ctx context.Context, projectID string, input domain.ProjectProfileInput) (*domain.ProjectProfile, error) {
	return s.repo.UpdateProject(ctx, projectID, input)
}
