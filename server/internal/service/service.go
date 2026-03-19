package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"strings"
	"time"

	"practicehelper/server/internal/domain"
	"practicehelper/server/internal/repo"
	"practicehelper/server/internal/sidecar"
)

var (
	ErrInvalidMode       = errors.New("invalid mode")
	ErrProjectNotFound   = errors.New("project not found")
	ErrSessionNotFound   = errors.New("session not found")
	ErrImportJobNotFound = errors.New("import job not found")
)

type Service struct {
	repo    *repo.Store
	sidecar *sidecar.Client
}

func New(repository *repo.Store, sc *sidecar.Client) *Service {
	svc := &Service{
		repo:    repository,
		sidecar: sc,
	}
	svc.resumePendingImportJobs()
	return svc
}

func (s *Service) GetProfile(ctx context.Context) (*domain.UserProfile, error) {
	return s.repo.GetUserProfile(ctx)
}

func (s *Service) SaveProfile(ctx context.Context, input domain.UserProfileInput) (*domain.UserProfile, error) {
	return s.repo.SaveUserProfile(ctx, input)
}

func (s *Service) GetDashboard(ctx context.Context) (*domain.Dashboard, error) {
	profile, err := s.repo.GetUserProfile(ctx)
	if err != nil {
		return nil, err
	}

	weaknesses, err := s.repo.ListWeaknesses(ctx, 5)
	if err != nil {
		return nil, err
	}

	recent, err := s.repo.ListRecentSessions(ctx, 5)
	if err != nil {
		return nil, err
	}

	currentSession, err := s.repo.GetLatestResumableSession(ctx)
	if err != nil {
		return nil, err
	}

	dashboard := &domain.Dashboard{
		Profile:          profile,
		Weaknesses:       weaknesses,
		RecentSessions:   recent,
		CurrentSession:   currentSession,
		TodayFocus:       buildTodayFocus(profile, weaknesses),
		RecommendedTrack: buildRecommendedTrack(profile, weaknesses),
	}

	if profile != nil && profile.ApplicationDeadline != nil {
		days := daysUntilDeadline(*profile.ApplicationDeadline)
		if days < 0 {
			days = 0
		}
		dashboard.DaysUntilDeadline = &days
	}

	return dashboard, nil
}

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

func (s *Service) CreateSession(ctx context.Context, request domain.CreateSessionRequest) (*domain.TrainingSession, error) {
	if request.Mode != domain.ModeBasics && request.Mode != domain.ModeProject {
		return nil, ErrInvalidMode
	}

	// 训练会话在创建时就要把首题持久化下来，而不是等用户答题后再补写，
	// 这样页面刷新后仍能恢复到同一轮训练上下文。topic 也在这里统一 trim/lower，
	// 避免 basics 模式因为大小写和空白差异导致模板命中不稳定。
	startedAt := time.Now().UTC()
	session := &domain.TrainingSession{
		ID:         newID("sess"),
		Mode:       request.Mode,
		Topic:      strings.TrimSpace(strings.ToLower(request.Topic)),
		ProjectID:  request.ProjectID,
		Intensity:  request.Intensity,
		Status:     domain.StatusWaitingAnswer,
		TotalScore: 0,
		StartedAt:  &startedAt,
	}

	weaknesses, err := s.repo.ListWeaknesses(ctx, 5)
	if err != nil {
		return nil, err
	}

	generateRequest := domain.GenerateQuestionRequest{
		Mode:       request.Mode,
		Topic:      session.Topic,
		Intensity:  request.Intensity,
		Weaknesses: weaknesses,
	}

	var project *domain.ProjectProfile
	switch request.Mode {
	case domain.ModeBasics:
		templates, err := s.repo.ListQuestionTemplatesByTopic(ctx, session.Topic)
		if err != nil {
			return nil, err
		}
		generateRequest.Templates = templates
	case domain.ModeProject:
		project, err = s.repo.GetProject(ctx, request.ProjectID)
		if err != nil {
			return nil, err
		}
		if project == nil {
			return nil, ErrProjectNotFound
		}
		session.Project = project
		generateRequest.Project = project
		// project 模式优先用 followup_points + summary 组装检索词，
		// 目的是把 sidecar 的首题约束在“最值得追问的项目点”上，而不是退化成全仓库泛扫。
		query := strings.Join(append(project.FollowupPoints, project.Summary), " ")
		chunks, err := s.repo.SearchProjectChunks(ctx, project.ID, query, 6)
		if err != nil {
			return nil, err
		}
		generateRequest.ContextChunks = chunks
	default:
		return nil, ErrInvalidMode
	}

	question, err := s.sidecar.GenerateQuestion(ctx, generateRequest)
	if err != nil {
		return nil, err
	}

	turn := &domain.TrainingTurn{
		ID:             newID("turn"),
		SessionID:      session.ID,
		TurnIndex:      1,
		Stage:          "question",
		Question:       question.Question,
		ExpectedPoints: question.ExpectedPoints,
	}

	if err := s.repo.CreateSession(ctx, session, turn); err != nil {
		return nil, err
	}

	return s.repo.GetSession(ctx, session.ID)
}

func (s *Service) CreateSessionStream(
	ctx context.Context,
	request domain.CreateSessionRequest,
	emit func(domain.StreamEvent) error,
) (*domain.TrainingSession, error) {
	if request.Mode != domain.ModeBasics && request.Mode != domain.ModeProject {
		return nil, ErrInvalidMode
	}

	startedAt := time.Now().UTC()
	session := &domain.TrainingSession{
		ID:         newID("sess"),
		Mode:       request.Mode,
		Topic:      strings.TrimSpace(strings.ToLower(request.Topic)),
		ProjectID:  request.ProjectID,
		Intensity:  request.Intensity,
		Status:     domain.StatusWaitingAnswer,
		TotalScore: 0,
		StartedAt:  &startedAt,
	}

	weaknesses, err := s.repo.ListWeaknesses(ctx, 5)
	if err != nil {
		return nil, err
	}

	generateRequest := domain.GenerateQuestionRequest{
		Mode:       request.Mode,
		Topic:      session.Topic,
		Intensity:  request.Intensity,
		Weaknesses: weaknesses,
	}

	switch request.Mode {
	case domain.ModeBasics:
		templates, err := s.repo.ListQuestionTemplatesByTopic(ctx, session.Topic)
		if err != nil {
			return nil, err
		}
		generateRequest.Templates = templates
	case domain.ModeProject:
		project, err := s.repo.GetProject(ctx, request.ProjectID)
		if err != nil {
			return nil, err
		}
		if project == nil {
			return nil, ErrProjectNotFound
		}
		session.Project = project
		generateRequest.Project = project
		query := strings.Join(append(project.FollowupPoints, project.Summary), " ")
		chunks, err := s.repo.SearchProjectChunks(ctx, project.ID, query, 6)
		if err != nil {
			return nil, err
		}
		generateRequest.ContextChunks = chunks
	default:
		return nil, ErrInvalidMode
	}

	question, err := s.sidecar.GenerateQuestionStream(ctx, generateRequest, emit)
	if err != nil {
		return nil, err
	}

	turn := &domain.TrainingTurn{
		ID:             newID("turn"),
		SessionID:      session.ID,
		TurnIndex:      1,
		Stage:          "question",
		Question:       question.Question,
		ExpectedPoints: question.ExpectedPoints,
	}

	if err := s.repo.CreateSession(ctx, session, turn); err != nil {
		return nil, err
	}

	return s.repo.GetSession(ctx, session.ID)
}

func (s *Service) SubmitAnswer(ctx context.Context, sessionID string, request domain.SubmitAnswerRequest) (*domain.TrainingSession, error) {
	session, err := s.repo.GetSession(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if session == nil {
		return nil, ErrSessionNotFound
	}
	if len(session.Turns) == 0 {
		return nil, fmt.Errorf("session %s has no turns", sessionID)
	}

	turn := session.Turns[len(session.Turns)-1]

	switch session.Status {
	case domain.StatusWaitingAnswer, domain.StatusActive:
		// 当前实现固定走“一道主问题 + 一道追问”的有限状态机：
		// 首答先落主评估和弱点，再把 session 切到 followup，保证中间态可恢复。
		contextChunks, err := s.lookupSessionContext(ctx, session)
		if err != nil {
			return nil, err
		}
		evaluation, err := s.sidecar.EvaluateAnswer(ctx, domain.EvaluateAnswerRequest{
			Mode:           session.Mode,
			Topic:          session.Topic,
			Project:        session.Project,
			Question:       turn.Question,
			ExpectedPoints: turn.ExpectedPoints,
			Answer:         request.Answer,
			ContextChunks:  contextChunks,
			IsFollowup:     false,
		})
		if err != nil {
			return nil, err
		}

		turn.Answer = request.Answer
		turn.Evaluation = evaluation
		turn.FollowupQuestion = evaluation.FollowupQuestion
		turn.FollowupExpectedPoint = evaluation.FollowupPoints
		turn.WeaknessHits = mergeWeaknessHits(turn.WeaknessHits, evaluation.WeaknessHits)

		if err := s.repo.SaveTurn(ctx, &turn); err != nil {
			return nil, err
		}
		if err := s.repo.UpsertWeaknesses(ctx, session.ID, evaluation.WeaknessHits); err != nil {
			return nil, err
		}

		session.Status = domain.StatusFollowup
		session.TotalScore = evaluation.Score
		if err := s.repo.SaveSession(ctx, session); err != nil {
			return nil, err
		}

		if evaluation.Score >= 75 {
			_ = s.coolDownSessionWeakness(ctx, session, evaluation.WeaknessHits)
		}
	case domain.StatusFollowup:
		// 追问答完后先固化 turn 与弱点，再进入 review 生成流程；
		// 这些步骤故意不包成一个大事务，这样 sidecar 调用失败时可以保留 review_pending 中间态供后续恢复。
		contextChunks, err := s.lookupSessionContext(ctx, session)
		if err != nil {
			return nil, err
		}
		evaluation, err := s.sidecar.EvaluateAnswer(ctx, domain.EvaluateAnswerRequest{
			Mode:           session.Mode,
			Topic:          session.Topic,
			Project:        session.Project,
			Question:       turn.FollowupQuestion,
			ExpectedPoints: turn.FollowupExpectedPoint,
			Answer:         request.Answer,
			ContextChunks:  contextChunks,
			IsFollowup:     true,
		})
		if err != nil {
			return nil, err
		}

		turn.FollowupAnswer = request.Answer
		turn.FollowupEvaluation = evaluation
		turn.WeaknessHits = mergeWeaknessHits(turn.WeaknessHits, evaluation.WeaknessHits)

		if err := s.repo.SaveTurn(ctx, &turn); err != nil {
			return nil, err
		}
		if err := s.repo.UpsertWeaknesses(ctx, session.ID, evaluation.WeaknessHits); err != nil {
			return nil, err
		}

		session.Status = domain.StatusReviewPending
		session.TotalScore = math.Round(((session.TotalScore+evaluation.Score)/2)*10) / 10
		if err := s.repo.SaveSession(ctx, session); err != nil {
			return nil, err
		}

		updatedSession, err := s.repo.GetSession(ctx, session.ID)
		if err != nil {
			return nil, err
		}
		review, err := s.sidecar.GenerateReview(ctx, domain.GenerateReviewRequest{
			Session: updatedSession,
			Project: updatedSession.Project,
			Turns:   updatedSession.Turns,
		})
		if err != nil {
			return nil, err
		}

		review.ID = newID("review")
		review.SessionID = updatedSession.ID
		if err := s.repo.CreateReview(ctx, review); err != nil {
			return nil, err
		}

		endedAt := time.Now().UTC()
		updatedSession.ReviewID = review.ID
		updatedSession.Status = domain.StatusCompleted
		updatedSession.EndedAt = &endedAt
		if err := s.repo.SaveSession(ctx, updatedSession); err != nil {
			return nil, err
		}

		if evaluation.Score >= 75 {
			_ = s.coolDownSessionWeakness(ctx, updatedSession, evaluation.WeaknessHits)
		}
	default:
		return nil, fmt.Errorf("session in status %s cannot accept answers", session.Status)
	}

	return s.repo.GetSession(ctx, session.ID)
}

func (s *Service) SubmitAnswerStream(
	ctx context.Context,
	sessionID string,
	request domain.SubmitAnswerRequest,
	emit func(domain.StreamEvent) error,
) (*domain.TrainingSession, error) {
	session, err := s.repo.GetSession(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if session == nil {
		return nil, ErrSessionNotFound
	}
	if len(session.Turns) == 0 {
		return nil, fmt.Errorf("session %s has no turns", sessionID)
	}

	turn := session.Turns[len(session.Turns)-1]
	previousStatus := session.Status

	switch session.Status {
	case domain.StatusWaitingAnswer, domain.StatusActive:
		contextChunks, err := s.lookupSessionContext(ctx, session)
		if err != nil {
			return nil, err
		}

		session.Status = domain.StatusEvaluating
		if err := s.repo.SaveSession(ctx, session); err != nil {
			return nil, err
		}

		evaluation, err := s.sidecar.EvaluateAnswerStream(ctx, domain.EvaluateAnswerRequest{
			Mode:           session.Mode,
			Topic:          session.Topic,
			Project:        session.Project,
			Question:       turn.Question,
			ExpectedPoints: turn.ExpectedPoints,
			Answer:         request.Answer,
			ContextChunks:  contextChunks,
			IsFollowup:     false,
		}, emit)
		if err != nil {
			session.Status = previousStatus
			_ = s.repo.SaveSession(ctx, session)
			return nil, err
		}

		turn.Answer = request.Answer
		turn.Evaluation = evaluation
		turn.FollowupQuestion = evaluation.FollowupQuestion
		turn.FollowupExpectedPoint = evaluation.FollowupPoints
		turn.WeaknessHits = mergeWeaknessHits(turn.WeaknessHits, evaluation.WeaknessHits)

		if err := s.repo.SaveTurn(ctx, &turn); err != nil {
			session.Status = previousStatus
			_ = s.repo.SaveSession(ctx, session)
			return nil, err
		}
		if err := s.repo.UpsertWeaknesses(ctx, session.ID, evaluation.WeaknessHits); err != nil {
			session.Status = previousStatus
			_ = s.repo.SaveSession(ctx, session)
			return nil, err
		}

		session.Status = domain.StatusFollowup
		session.TotalScore = evaluation.Score
		if err := s.repo.SaveSession(ctx, session); err != nil {
			return nil, err
		}

		if evaluation.Score >= 75 {
			_ = s.coolDownSessionWeakness(ctx, session, evaluation.WeaknessHits)
		}
	case domain.StatusFollowup:
		contextChunks, err := s.lookupSessionContext(ctx, session)
		if err != nil {
			return nil, err
		}

		session.Status = domain.StatusEvaluating
		if err := s.repo.SaveSession(ctx, session); err != nil {
			return nil, err
		}

		evaluation, err := s.sidecar.EvaluateAnswerStream(ctx, domain.EvaluateAnswerRequest{
			Mode:           session.Mode,
			Topic:          session.Topic,
			Project:        session.Project,
			Question:       turn.FollowupQuestion,
			ExpectedPoints: turn.FollowupExpectedPoint,
			Answer:         request.Answer,
			ContextChunks:  contextChunks,
			IsFollowup:     true,
		}, emit)
		if err != nil {
			session.Status = previousStatus
			_ = s.repo.SaveSession(ctx, session)
			return nil, err
		}

		turn.FollowupAnswer = request.Answer
		turn.FollowupEvaluation = evaluation
		turn.WeaknessHits = mergeWeaknessHits(turn.WeaknessHits, evaluation.WeaknessHits)

		if err := s.repo.SaveTurn(ctx, &turn); err != nil {
			session.Status = previousStatus
			_ = s.repo.SaveSession(ctx, session)
			return nil, err
		}
		if err := s.repo.UpsertWeaknesses(ctx, session.ID, evaluation.WeaknessHits); err != nil {
			session.Status = previousStatus
			_ = s.repo.SaveSession(ctx, session)
			return nil, err
		}

		session.Status = domain.StatusReviewPending
		session.TotalScore = math.Round(((session.TotalScore+evaluation.Score)/2)*10) / 10
		if err := s.repo.SaveSession(ctx, session); err != nil {
			return nil, err
		}

		updatedSession, err := s.repo.GetSession(ctx, session.ID)
		if err != nil {
			return nil, err
		}
		review, err := s.sidecar.GenerateReviewStream(ctx, domain.GenerateReviewRequest{
			Session: updatedSession,
			Project: updatedSession.Project,
			Turns:   updatedSession.Turns,
		}, emit)
		if err != nil {
			return nil, err
		}

		review.ID = newID("review")
		review.SessionID = updatedSession.ID
		if err := s.repo.CreateReview(ctx, review); err != nil {
			return nil, err
		}

		endedAt := time.Now().UTC()
		updatedSession.ReviewID = review.ID
		updatedSession.Status = domain.StatusCompleted
		updatedSession.EndedAt = &endedAt
		if err := s.repo.SaveSession(ctx, updatedSession); err != nil {
			return nil, err
		}

		if evaluation.Score >= 75 {
			_ = s.coolDownSessionWeakness(ctx, updatedSession, evaluation.WeaknessHits)
		}
	default:
		return nil, fmt.Errorf("session in status %s cannot accept answers", session.Status)
	}

	return s.repo.GetSession(ctx, session.ID)
}

func (s *Service) GetSession(ctx context.Context, sessionID string) (*domain.TrainingSession, error) {
	session, err := s.repo.GetSession(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if session == nil {
		return nil, ErrSessionNotFound
	}
	return session, nil
}

func (s *Service) GetReview(ctx context.Context, reviewID string) (*domain.ReviewCard, error) {
	return s.repo.GetReview(ctx, reviewID)
}

func (s *Service) ListWeaknesses(ctx context.Context) ([]domain.WeaknessTag, error) {
	return s.repo.ListWeaknesses(ctx, 20)
}

func (s *Service) lookupSessionContext(ctx context.Context, session *domain.TrainingSession) ([]domain.RepoChunk, error) {
	if session.Mode != domain.ModeProject || session.Project == nil {
		return nil, nil
	}

	query := strings.Join(append(session.Project.FollowupPoints, session.Project.Summary), " ")
	return s.repo.SearchProjectChunks(ctx, session.Project.ID, query, 6)
}

func (s *Service) coolDownSessionWeakness(ctx context.Context, session *domain.TrainingSession, hits []domain.WeaknessHit) error {
	// 这里是“答得好就给弱点降温”的启发式修正，不追求精确评分模型。
	// 命中的具体弱点和本轮主维度（topic/project）都会被轻微衰减；
	// 即便降温失败，也不能反向影响训练主流程，所以这里统一忽略 repo 写入错误。
	for _, hit := range hits {
		_ = s.repo.RelieveWeakness(ctx, hit.Kind, hit.Label, 0.18)
	}

	if session.Mode == domain.ModeBasics && session.Topic != "" {
		_ = s.repo.RelieveWeakness(ctx, "topic", session.Topic, 0.15)
	}
	if session.Mode == domain.ModeProject && session.Project != nil {
		_ = s.repo.RelieveWeakness(ctx, "project", session.Project.Name, 0.15)
	}

	return nil
}

func buildTodayFocus(profile *domain.UserProfile, weaknesses []domain.WeaknessTag) string {
	if len(weaknesses) > 0 {
		return fmt.Sprintf("今天优先补 %s：%s", formatWeaknessKindLabel(weaknesses[0].Kind), weaknesses[0].Label)
	}

	if profile != nil && len(profile.PrimaryProjects) > 0 {
		return fmt.Sprintf("今天先做一轮项目表达训练，主讲 %s", profile.PrimaryProjects[0])
	}

	return "先完成画像初始化，然后做一轮 Redis 或 Go 基础训练。"
}

func buildRecommendedTrack(profile *domain.UserProfile, weaknesses []domain.WeaknessTag) string {
	if len(weaknesses) > 0 {
		switch weaknesses[0].Kind {
		case "topic":
			return fmt.Sprintf("%s 专项训练", strings.ToUpper(weaknesses[0].Label))
		case "project":
			return fmt.Sprintf("%s 项目专项", weaknesses[0].Label)
		case "expression":
			return "表达方式专项"
		case "followup_breakdown":
			return "追问抗压专项"
		case "depth":
			return "展开深挖专项"
		case "detail":
			return "细节补强专项"
		default:
			return "追问抗压专项"
		}
	}

	if profile != nil && profile.TargetRole != "" {
		return fmt.Sprintf("%s 目标岗位基础训练", profile.TargetRole)
	}

	return "Go 后端 + Agent 方向基础训练"
}

func mergeWeaknessHits(base []domain.WeaknessHit, extra []domain.WeaknessHit) []domain.WeaknessHit {
	result := append([]domain.WeaknessHit{}, base...)
	result = append(result, extra...)
	return result
}

func formatWeaknessKindLabel(kind string) string {
	switch kind {
	case "topic":
		return "知识点"
	case "project":
		return "项目表达"
	case "expression":
		return "表达方式"
	case "followup_breakdown":
		return "追问应对"
	case "depth":
		return "展开深度"
	case "detail":
		return "细节支撑"
	default:
		return kind
	}
}

func daysUntilDeadline(deadline time.Time) int {
	today := time.Now().UTC()
	todayStart := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, time.UTC)
	target := deadline.UTC()
	targetStart := time.Date(target.Year(), target.Month(), target.Day(), 0, 0, 0, 0, time.UTC)

	return int(targetStart.Sub(todayStart).Hours() / 24)
}

func newID(prefix string) string {
	return fmt.Sprintf("%s_%d", prefix, time.Now().UnixNano())
}
