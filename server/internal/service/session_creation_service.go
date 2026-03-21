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

	maxTurns := request.MaxTurns
	if maxTurns < 2 {
		maxTurns = 2
	}
	if maxTurns > 5 {
		maxTurns = 5
	}

	intensity := request.Intensity
	if intensity == "auto" {
		intensity = s.resolveAutoIntensity(ctx)
	}

	startedAt := time.Now().UTC()
	session := &domain.TrainingSession{
		ID:         newID("sess"),
		Mode:       request.Mode,
		Topic:      strings.TrimSpace(strings.ToLower(request.Topic)),
		ProjectID:  request.ProjectID,
		Intensity:  intensity,
		Status:     domain.StatusWaitingAnswer,
		MaxTurns:   maxTurns,
		TotalScore: 0,
		StartedAt:  &startedAt,
	}

	jobTarget, jobTargetAnalysis, err := s.resolveJobTargetBinding(ctx, request.JobTargetID, request.IgnoreActiveJobTarget)
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
		Intensity:         intensity,
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
		query := strings.Join(append(project.FollowupPoints, project.Summary), " ")
		chunks, err := s.repo.SearchProjectChunks(ctx, project.ID, query, 6)
		if err != nil {
			return nil, err
		}
		generateRequest.ContextChunks = chunks
	default:
		return nil, ErrInvalidMode
	}

	questionStartedAt := time.Now()
	question, err := s.sidecar.GenerateQuestion(ctx, generateRequest)
	if err != nil {
		return nil, err
	}
	s.recordEvaluationLog(ctx, session.ID, "", "generate_question", questionStartedAt)

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

	maxTurns := request.MaxTurns
	if maxTurns < 2 {
		maxTurns = 2
	}
	if maxTurns > 5 {
		maxTurns = 5
	}

	intensity := request.Intensity
	if intensity == "auto" {
		intensity = s.resolveAutoIntensity(ctx)
	}

	startedAt := time.Now().UTC()
	session := &domain.TrainingSession{
		ID:         newID("sess"),
		Mode:       request.Mode,
		Topic:      strings.TrimSpace(strings.ToLower(request.Topic)),
		ProjectID:  request.ProjectID,
		Intensity:  intensity,
		Status:     domain.StatusWaitingAnswer,
		MaxTurns:   maxTurns,
		TotalScore: 0,
		StartedAt:  &startedAt,
	}

	jobTarget, jobTargetAnalysis, err := s.resolveJobTargetBinding(ctx, request.JobTargetID, request.IgnoreActiveJobTarget)
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
		Intensity:         intensity,
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

	questionStartedAt := time.Now()
	question, err := s.sidecar.GenerateQuestionStream(ctx, generateRequest, emit)
	if err != nil {
		return nil, err
	}
	s.recordEvaluationLog(ctx, session.ID, "", "generate_question_stream", questionStartedAt)

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

func (s *Service) resolveAutoIntensity(ctx context.Context) string {
	recent, err := s.repo.ListRecentSessions(ctx, 5)
	if err != nil || len(recent) == 0 {
		return "standard"
	}

	var total float64
	var count int
	for _, sess := range recent {
		if sess.Status == domain.StatusCompleted && sess.TotalScore > 0 {
			total += sess.TotalScore
			count++
		}
	}
	if count == 0 {
		return "standard"
	}

	avg := total / float64(count)
	switch {
	case avg >= 80:
		return "pressure"
	case avg < 60:
		return "light"
	default:
		return "standard"
	}
}
