package service

import (
	"context"
	"strings"
	"time"

	"practicehelper/server/internal/domain"
)

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

	jobTarget, jobTargetAnalysis, err := s.resolveJobTargetBinding(ctx, request.JobTargetID)
	if err != nil {
		return nil, err
	}
	if jobTarget != nil {
		session.JobTargetID = jobTarget.ID
		session.JobTargetAnalysisID = jobTargetAnalysis.ID
		session.JobTarget = &domain.JobTargetRef{
			ID:          jobTarget.ID,
			Title:       jobTarget.Title,
			CompanyName: jobTarget.CompanyName,
		}
	}

	weaknesses, err := s.repo.ListWeaknesses(ctx, 5)
	if err != nil {
		return nil, err
	}

	generateRequest := domain.GenerateQuestionRequest{
		Mode:              request.Mode,
		Topic:             session.Topic,
		Intensity:         request.Intensity,
		Weaknesses:        weaknesses,
		JobTargetAnalysis: buildJobTargetAnalysisSnapshot(jobTargetAnalysis),
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

	jobTarget, jobTargetAnalysis, err := s.resolveJobTargetBinding(ctx, request.JobTargetID)
	if err != nil {
		return nil, err
	}
	if jobTarget != nil {
		session.JobTargetID = jobTarget.ID
		session.JobTargetAnalysisID = jobTargetAnalysis.ID
		session.JobTarget = &domain.JobTargetRef{
			ID:          jobTarget.ID,
			Title:       jobTarget.Title,
			CompanyName: jobTarget.CompanyName,
		}
	}

	weaknesses, err := s.repo.ListWeaknesses(ctx, 5)
	if err != nil {
		return nil, err
	}

	generateRequest := domain.GenerateQuestionRequest{
		Mode:              request.Mode,
		Topic:             session.Topic,
		Intensity:         request.Intensity,
		Weaknesses:        weaknesses,
		JobTargetAnalysis: buildJobTargetAnalysisSnapshot(jobTargetAnalysis),
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
