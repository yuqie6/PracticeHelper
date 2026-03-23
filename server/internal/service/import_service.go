package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"practicehelper/server/internal/domain"
	"practicehelper/server/internal/repo"
)

const (
	projectImportClaimTTL             = 15 * time.Second
	projectImportClaimHeartbeatPeriod = 5 * time.Second
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
		s.startImportJob(*job)
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
	s.startImportJobAttempt(job, job.Status == domain.ProjectImportStatusRunning)
}

func (s *Service) startImportJobAttempt(job domain.ProjectImportJob, allowRetry bool) {
	if s.sidecar == nil {
		return
	}

	go func() {
		backgroundCtx := context.Background()
		claimToken := newImportClaimToken()
		startedAt := time.Now().UTC()
		claimed, err := s.repo.ClaimProjectImportJob(
			backgroundCtx,
			job.ID,
			claimToken,
			domain.ProjectImportStageAnalyzing,
			"正在克隆仓库、提取关键文件并生成项目画像。",
			startedAt,
			startedAt.Add(projectImportClaimTTL),
		)
		if err != nil {
			slog.Error("claim import job failed", "job_id", job.ID, "error", err)
			return
		}
		if !claimed {
			if allowRetry && job.Status == domain.ProjectImportStatusRunning {
				s.scheduleImportJobRetry(job)
			}
			return
		}

		stopHeartbeat := s.startImportJobHeartbeat(job.ID, claimToken)
		defer stopHeartbeat()

		analysis, err := s.sidecar.AnalyzeRepo(backgroundCtx, domain.AnalyzeRepoRequest{RepoURL: job.RepoURL})
		if err != nil {
			s.failImportJob(backgroundCtx, job.ID, claimToken, fmt.Sprintf("项目导入失败：%v", err))
			return
		}

		advanced, err := s.repo.AdvanceClaimedProjectImportJob(
			backgroundCtx,
			job.ID,
			claimToken,
			domain.ProjectImportStagePersisting,
			"正在写入项目画像、源码片段和检索索引。",
			time.Now().UTC().Add(projectImportClaimTTL),
		)
		if err != nil {
			slog.Error("advance import job to persisting failed", "job_id", job.ID, "error", err)
			return
		}
		if !advanced {
			slog.Warn("lost import job claim before persisting", "job_id", job.ID)
			return
		}

		project, err := s.repo.CreateImportedProject(backgroundCtx, analysis)
		if errors.Is(err, repo.ErrAlreadyImported) {
			project, err = s.repo.GetProjectByRepoURL(backgroundCtx, job.RepoURL)
			if err == nil && project != nil {
				if updateErr := s.completeImportJob(
					backgroundCtx,
					job.ID,
					claimToken,
					"仓库已存在，已复用已有项目材料。",
					project.ID,
				); updateErr != nil {
					slog.Error("complete import job with existing project failed", "job_id", job.ID, "error", updateErr)
				}
				return
			}
		}
		if err != nil {
			s.failImportJob(backgroundCtx, job.ID, claimToken, fmt.Sprintf("项目画像落库失败：%v", err))
			return
		}

		if err := s.completeImportJob(
			backgroundCtx,
			job.ID,
			claimToken,
			"项目材料已准备好，可以开始编辑和训练。",
			project.ID,
		); err != nil {
			slog.Error("complete import job failed", "job_id", job.ID, "project_id", project.ID, "error", err)
			return
		}

		go s.enqueueProjectRepoChunkEmbeddings(context.Background(), project.ID)
	}()
}

func (s *Service) failImportJob(
	ctx context.Context,
	jobID string,
	claimToken string,
	message string,
) {
	finishedAt := time.Now().UTC()
	finished, err := s.repo.FinishClaimedProjectImportJob(
		ctx,
		jobID,
		claimToken,
		domain.ProjectImportStatusFailed,
		domain.ProjectImportStageFailed,
		"导入失败，请检查仓库地址、LLM 配置或稍后重试。",
		message,
		"",
		finishedAt,
	)
	if err != nil {
		slog.Error("mark import job failed error", "job_id", jobID, "error", err)
		return
	}
	if !finished {
		slog.Warn("skip failing import job because claim was lost", "job_id", jobID)
	}
}

func (s *Service) resumePendingImportJobs() {
	if s.sidecar == nil {
		return
	}

	jobs, err := s.repo.ListPendingProjectImportJobs(context.Background())
	if err != nil {
		slog.Error("list pending import jobs failed", "error", err)
		return
	}

	for _, job := range jobs {
		s.startImportJob(job)
	}
}

func (s *Service) startImportJobHeartbeat(jobID string, claimToken string) func() {
	stop := make(chan struct{})

	go func() {
		ticker := time.NewTicker(projectImportClaimHeartbeatPeriod)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				touched, err := s.repo.TouchProjectImportJobClaim(
					context.Background(),
					jobID,
					claimToken,
					time.Now().UTC().Add(projectImportClaimTTL),
				)
				if err != nil {
					slog.Error("touch import job claim failed", "job_id", jobID, "error", err)
					continue
				}
				if !touched {
					slog.Warn("stop import job heartbeat because claim was lost", "job_id", jobID)
					return
				}
			case <-stop:
				return
			}
		}
	}()

	return func() {
		close(stop)
	}
}

func (s *Service) scheduleImportJobRetry(job domain.ProjectImportJob) {
	time.AfterFunc(projectImportClaimTTL, func() {
		s.startImportJobAttempt(job, false)
	})
}

func (s *Service) completeImportJob(
	ctx context.Context,
	jobID string,
	claimToken string,
	message string,
	projectID string,
) error {
	finished, err := s.repo.FinishClaimedProjectImportJob(
		ctx,
		jobID,
		claimToken,
		domain.ProjectImportStatusCompleted,
		domain.ProjectImportStageCompleted,
		message,
		"",
		projectID,
		time.Now().UTC(),
	)
	if err != nil {
		return err
	}
	if !finished {
		return fmt.Errorf("import job %s claim lost before completion", jobID)
	}
	return nil
}

func newImportClaimToken() string {
	buffer := make([]byte, 8)
	if _, err := rand.Read(buffer); err != nil {
		panic(err)
	}
	return "import_claim_" + hex.EncodeToString(buffer)
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
