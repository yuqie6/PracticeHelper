package service

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"practicehelper/server/internal/domain"
	"practicehelper/server/internal/repo"
	"practicehelper/server/internal/sidecar"
)

var (
	ErrInvalidMode     = errors.New("invalid mode")
	ErrProjectNotFound = errors.New("project not found")
	ErrSessionNotFound = errors.New("session not found")
)

type Service struct {
	repo    *repo.Store
	sidecar *sidecar.Client
}

func New(repository *repo.Store, sc *sidecar.Client) *Service {
	return &Service{
		repo:    repository,
		sidecar: sc,
	}
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

	dashboard := &domain.Dashboard{
		Profile:          profile,
		Weaknesses:       weaknesses,
		RecentSessions:   recent,
		TodayFocus:       buildTodayFocus(profile, weaknesses),
		RecommendedTrack: buildRecommendedTrack(profile, weaknesses),
	}

	if profile != nil && profile.ApplicationDeadline != nil {
		days := int(time.Until(profile.ApplicationDeadline.UTC()).Hours() / 24)
		if days < 0 {
			days = 0
		}
		dashboard.DaysUntilDeadline = &days
	}

	return dashboard, nil
}

func (s *Service) ImportProject(ctx context.Context, request domain.ProjectImportRequest) (*domain.ProjectProfile, error) {
	analysis, err := s.sidecar.AnalyzeRepo(ctx, domain.AnalyzeRepoRequest{RepoURL: request.RepoURL})
	if err != nil {
		return nil, err
	}

	return s.repo.CreateImportedProject(ctx, analysis)
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
		return fmt.Sprintf("今天优先补 %s：%s", weaknesses[0].Kind, weaknesses[0].Label)
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
			return "项目表达专项"
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

func newID(prefix string) string {
	return fmt.Sprintf("%s_%d", prefix, time.Now().UnixNano())
}
