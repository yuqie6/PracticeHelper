package service

import (
	"context"
	"fmt"
	"math"
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
		previousStatus := session.Status
		if err := s.claimSessionForAnswer(ctx, session, previousStatus); err != nil {
			return nil, err
		}

		// 当前实现固定走“一道主问题 + 一道追问”的有限状态机：
		// 首答先落主评估和弱点，再把 session 切到 followup，保证中间态可恢复。
		contextChunks, err := s.lookupSessionContext(ctx, session)
		if err != nil {
			s.restoreSessionStatus(ctx, session.ID, previousStatus)
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
			s.restoreSessionStatus(ctx, session.ID, previousStatus)
			return nil, err
		}

		turn.Answer = request.Answer
		turn.Evaluation = evaluation
		turn.FollowupQuestion = evaluation.FollowupQuestion
		turn.FollowupExpectedPoint = evaluation.FollowupPoints
		turn.WeaknessHits = mergeWeaknessHits(turn.WeaknessHits, evaluation.WeaknessHits)

		if err := s.repo.SaveTurn(ctx, &turn); err != nil {
			s.restoreSessionStatus(ctx, session.ID, previousStatus)
			return nil, err
		}
		if err := s.repo.UpsertWeaknesses(ctx, session.ID, evaluation.WeaknessHits); err != nil {
			s.restoreSessionStatus(ctx, session.ID, previousStatus)
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
		previousStatus := session.Status
		if err := s.claimSessionForAnswer(ctx, session, previousStatus); err != nil {
			return nil, err
		}

		// 追问答完后先固化 turn 与弱点，再进入 review 生成流程；
		// 这些步骤故意不包成一个大事务，这样 sidecar 调用失败时可以保留 review_pending 中间态供后续恢复。
		contextChunks, err := s.lookupSessionContext(ctx, session)
		if err != nil {
			s.restoreSessionStatus(ctx, session.ID, previousStatus)
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
			s.restoreSessionStatus(ctx, session.ID, previousStatus)
			return nil, err
		}

		turn.FollowupAnswer = request.Answer
		turn.FollowupEvaluation = evaluation
		turn.WeaknessHits = mergeWeaknessHits(turn.WeaknessHits, evaluation.WeaknessHits)

		if err := s.repo.SaveTurn(ctx, &turn); err != nil {
			s.restoreSessionStatus(ctx, session.ID, previousStatus)
			return nil, err
		}
		if err := s.repo.UpsertWeaknesses(ctx, session.ID, evaluation.WeaknessHits); err != nil {
			s.restoreSessionStatus(ctx, session.ID, previousStatus)
			return nil, err
		}

		session.Status = domain.StatusReviewPending
		session.TotalScore = math.Round(((session.TotalScore+evaluation.Score)/2)*10) / 10
		if err := s.repo.SaveSession(ctx, session); err != nil {
			return nil, err
		}

		updatedSession, err := s.finalizeReview(ctx, session.ID)
		if err != nil {
			return nil, err
		}

		if evaluation.Score >= 75 {
			_ = s.coolDownSessionWeakness(ctx, updatedSession, evaluation.WeaknessHits)
		}
	default:
		return nil, classifySubmitAnswerStatus(session.Status)
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

	switch session.Status {
	case domain.StatusWaitingAnswer, domain.StatusActive:
		previousStatus := session.Status
		if err := s.claimSessionForAnswer(ctx, session, previousStatus); err != nil {
			return nil, err
		}

		contextChunks, err := s.lookupSessionContext(ctx, session)
		if err != nil {
			s.restoreSessionStatus(ctx, session.ID, previousStatus)
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
			s.restoreSessionStatus(ctx, session.ID, previousStatus)
			return nil, err
		}

		turn.Answer = request.Answer
		turn.Evaluation = evaluation
		turn.FollowupQuestion = evaluation.FollowupQuestion
		turn.FollowupExpectedPoint = evaluation.FollowupPoints
		turn.WeaknessHits = mergeWeaknessHits(turn.WeaknessHits, evaluation.WeaknessHits)

		if err := s.repo.SaveTurn(ctx, &turn); err != nil {
			s.restoreSessionStatus(ctx, session.ID, previousStatus)
			return nil, err
		}
		if err := s.repo.UpsertWeaknesses(ctx, session.ID, evaluation.WeaknessHits); err != nil {
			s.restoreSessionStatus(ctx, session.ID, previousStatus)
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
		previousStatus := session.Status
		if err := s.claimSessionForAnswer(ctx, session, previousStatus); err != nil {
			return nil, err
		}

		contextChunks, err := s.lookupSessionContext(ctx, session)
		if err != nil {
			s.restoreSessionStatus(ctx, session.ID, previousStatus)
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
			s.restoreSessionStatus(ctx, session.ID, previousStatus)
			return nil, err
		}

		turn.FollowupAnswer = request.Answer
		turn.FollowupEvaluation = evaluation
		turn.WeaknessHits = mergeWeaknessHits(turn.WeaknessHits, evaluation.WeaknessHits)

		if err := s.repo.SaveTurn(ctx, &turn); err != nil {
			s.restoreSessionStatus(ctx, session.ID, previousStatus)
			return nil, err
		}
		if err := s.repo.UpsertWeaknesses(ctx, session.ID, evaluation.WeaknessHits); err != nil {
			s.restoreSessionStatus(ctx, session.ID, previousStatus)
			return nil, err
		}

		session.Status = domain.StatusReviewPending
		session.TotalScore = math.Round(((session.TotalScore+evaluation.Score)/2)*10) / 10
		if err := s.repo.SaveSession(ctx, session); err != nil {
			return nil, err
		}

		updatedSession, err := s.finalizeReviewStream(ctx, session.ID, emit)
		if err != nil {
			return nil, err
		}

		if evaluation.Score >= 75 {
			_ = s.coolDownSessionWeakness(ctx, updatedSession, evaluation.WeaknessHits)
		}
	default:
		return nil, classifySubmitAnswerStatus(session.Status)
	}

	return s.repo.GetSession(ctx, session.ID)
}

func (s *Service) finalizeReview(ctx context.Context, sessionID string) (*domain.TrainingSession, error) {
	updatedSession, err := s.repo.GetSession(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if updatedSession == nil {
		return nil, ErrSessionNotFound
	}

	review, err := s.sidecar.GenerateReview(ctx, domain.GenerateReviewRequest{
		Session: updatedSession,
		Project: updatedSession.Project,
		Turns:   updatedSession.Turns,
	})
	if err != nil {
		return nil, err
	}

	return s.persistReview(ctx, updatedSession, review)
}

func (s *Service) finalizeReviewStream(
	ctx context.Context,
	sessionID string,
	emit func(domain.StreamEvent) error,
) (*domain.TrainingSession, error) {
	updatedSession, err := s.repo.GetSession(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if updatedSession == nil {
		return nil, ErrSessionNotFound
	}

	review, err := s.sidecar.GenerateReviewStream(ctx, domain.GenerateReviewRequest{
		Session: updatedSession,
		Project: updatedSession.Project,
		Turns:   updatedSession.Turns,
	}, emit)
	if err != nil {
		return nil, err
	}

	return s.persistReview(ctx, updatedSession, review)
}

func (s *Service) persistReview(
	ctx context.Context,
	session *domain.TrainingSession,
	review *domain.ReviewCard,
) (*domain.TrainingSession, error) {
	review.ID = newID("review")
	review.SessionID = session.ID
	if err := s.repo.CreateReview(ctx, review); err != nil {
		return nil, err
	}

	endedAt := time.Now().UTC()
	session.ReviewID = review.ID
	session.Status = domain.StatusCompleted
	session.EndedAt = &endedAt
	if err := s.repo.SaveSession(ctx, session); err != nil {
		return nil, err
	}

	return session, nil
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

func (s *Service) RetrySessionReview(ctx context.Context, sessionID string) (*domain.TrainingSession, error) {
	session, err := s.repo.GetSession(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if session == nil {
		return nil, ErrSessionNotFound
	}
	switch session.Status {
	case domain.StatusReviewPending:
	default:
		return nil, classifyRetryReviewStatus(session.Status)
	}

	claimed, err := s.repo.TransitionSessionStatus(
		ctx,
		session.ID,
		[]string{domain.StatusReviewPending},
		domain.StatusEvaluating,
	)
	if err != nil {
		return nil, err
	}
	if !claimed {
		latest, err := s.repo.GetSession(ctx, sessionID)
		if err != nil {
			return nil, err
		}
		if latest == nil {
			return nil, ErrSessionNotFound
		}
		return nil, classifyRetryReviewStatus(latest.Status)
	}

	updatedSession, err := s.finalizeReview(ctx, sessionID)
	if err != nil {
		s.restoreSessionStatus(ctx, session.ID, domain.StatusReviewPending)
		return nil, err
	}

	return updatedSession, nil
}

func (s *Service) claimSessionForAnswer(
	ctx context.Context,
	session *domain.TrainingSession,
	from string,
) error {
	claimed, err := s.repo.TransitionSessionStatus(ctx, session.ID, []string{from}, domain.StatusEvaluating)
	if err != nil {
		return err
	}
	if claimed {
		session.Status = domain.StatusEvaluating
		return nil
	}

	latest, err := s.repo.GetSession(ctx, session.ID)
	if err != nil {
		return err
	}
	if latest == nil {
		return ErrSessionNotFound
	}

	return classifySubmitAnswerStatus(latest.Status)
}

func (s *Service) restoreSessionStatus(ctx context.Context, sessionID string, target string) {
	_, _ = s.repo.TransitionSessionStatus(ctx, sessionID, []string{domain.StatusEvaluating}, target)
}

func classifySubmitAnswerStatus(status string) error {
	switch status {
	case domain.StatusEvaluating:
		return ErrSessionBusy
	case domain.StatusReviewPending:
		return ErrSessionReviewPending
	case domain.StatusCompleted:
		return ErrSessionCompleted
	default:
		return fmt.Errorf("%w: %s", ErrSessionAnswerConflict, status)
	}
}

func classifyRetryReviewStatus(status string) error {
	switch status {
	case domain.StatusEvaluating:
		return ErrSessionBusy
	case domain.StatusCompleted:
		return ErrSessionCompleted
	default:
		return ErrSessionNotRecoverable
	}
}
