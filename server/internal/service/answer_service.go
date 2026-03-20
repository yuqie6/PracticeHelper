package service

import (
	"context"
	"fmt"
	"math"

	"practicehelper/server/internal/domain"
)

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
	if session.Status != domain.StatusWaitingAnswer && session.Status != domain.StatusActive {
		return nil, classifySubmitAnswerStatus(session.Status)
	}

	turn := &session.Turns[len(session.Turns)-1]
	previousStatus := session.Status
	if err := s.claimSessionForAnswer(ctx, session, previousStatus); err != nil {
		return nil, err
	}

	contextChunks, jobTargetAnalysis, scoreWeights, err := s.loadAnswerContext(ctx, session)
	if err != nil {
		s.restoreSessionStatus(ctx, session.ID, previousStatus)
		return nil, err
	}

	isLastTurn := turn.TurnIndex >= session.MaxTurns
	evaluation, err := s.sidecar.EvaluateAnswer(ctx, domain.EvaluateAnswerRequest{
		Mode:              session.Mode,
		Topic:             session.Topic,
		Project:           session.Project,
		Question:          turn.Question,
		ExpectedPoints:    turn.ExpectedPoints,
		Answer:            request.Answer,
		ContextChunks:     contextChunks,
		TurnIndex:         turn.TurnIndex,
		MaxTurns:          session.MaxTurns,
		ScoreWeights:      scoreWeights,
		JobTargetAnalysis: jobTargetAnalysis,
	})
	if err != nil {
		s.restoreSessionStatus(ctx, session.ID, previousStatus)
		return nil, err
	}

	turn.Answer = request.Answer
	turn.Evaluation = evaluation
	turn.WeaknessHits = mergeWeaknessHits(turn.WeaknessHits, evaluation.WeaknessHits)

	if err := s.repo.SaveTurn(ctx, turn); err != nil {
		s.restoreSessionStatus(ctx, session.ID, previousStatus)
		return nil, err
	}
	if err := s.repo.UpsertWeaknesses(ctx, session.ID, evaluation.WeaknessHits); err != nil {
		s.restoreSessionStatus(ctx, session.ID, previousStatus)
		return nil, err
	}

	session.TotalScore = s.computeSessionScore(session)

	if evaluation.Score >= 75 {
		_ = s.coolDownSessionWeakness(ctx, session, turn.Question, evaluation.WeaknessHits)
	}

	if isLastTurn {
		session.Status = domain.StatusReviewPending
		if err := s.repo.SaveSession(ctx, session); err != nil {
			return nil, err
		}
		if _, err := s.finalizeReview(ctx, session.ID); err != nil {
			return nil, wrapReviewGenerationRetry(err)
		}
	} else {
		nextTurn := &domain.TrainingTurn{
			ID:             newID("turn"),
			SessionID:      session.ID,
			TurnIndex:      turn.TurnIndex + 1,
			Stage:          "question",
			Question:       evaluation.FollowupQuestion,
			ExpectedPoints: evaluation.FollowupPoints,
		}
		session.Status = domain.StatusWaitingAnswer
		if err := s.repo.SaveSession(ctx, session); err != nil {
			return nil, err
		}
		if err := s.repo.InsertTurn(ctx, nextTurn); err != nil {
			return nil, err
		}
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
	if session.Status != domain.StatusWaitingAnswer && session.Status != domain.StatusActive {
		return nil, classifySubmitAnswerStatus(session.Status)
	}

	turn := &session.Turns[len(session.Turns)-1]
	previousStatus := session.Status
	if err := s.claimSessionForAnswer(ctx, session, previousStatus); err != nil {
		return nil, err
	}
	emitStatus(emit, "answer_received")

	contextChunks, jobTargetAnalysis, scoreWeights, err := s.loadAnswerContext(ctx, session)
	if err != nil {
		s.restoreSessionStatus(ctx, session.ID, previousStatus)
		return nil, err
	}

	isLastTurn := turn.TurnIndex >= session.MaxTurns
	emitStatus(emit, "evaluation_started")

	evaluation, err := s.sidecar.EvaluateAnswerStream(ctx, domain.EvaluateAnswerRequest{
		Mode:              session.Mode,
		Topic:             session.Topic,
		Project:           session.Project,
		Question:          turn.Question,
		ExpectedPoints:    turn.ExpectedPoints,
		Answer:            request.Answer,
		ContextChunks:     contextChunks,
		TurnIndex:         turn.TurnIndex,
		MaxTurns:          session.MaxTurns,
		ScoreWeights:      scoreWeights,
		JobTargetAnalysis: jobTargetAnalysis,
	}, emit)
	if err != nil {
		s.restoreSessionStatus(ctx, session.ID, previousStatus)
		return nil, err
	}
	emitStatus(emit, "feedback_ready")

	turn.Answer = request.Answer
	turn.Evaluation = evaluation
	turn.WeaknessHits = mergeWeaknessHits(turn.WeaknessHits, evaluation.WeaknessHits)

	if err := s.repo.SaveTurn(ctx, turn); err != nil {
		s.restoreSessionStatus(ctx, session.ID, previousStatus)
		return nil, err
	}
	emitStatus(emit, "answer_saved")
	if err := s.repo.UpsertWeaknesses(ctx, session.ID, evaluation.WeaknessHits); err != nil {
		s.restoreSessionStatus(ctx, session.ID, previousStatus)
		return nil, err
	}

	session.TotalScore = s.computeSessionScore(session)

	if evaluation.Score >= 75 {
		_ = s.coolDownSessionWeakness(ctx, session, turn.Question, evaluation.WeaknessHits)
	}

	if isLastTurn {
		session.Status = domain.StatusReviewPending
		if err := s.repo.SaveSession(ctx, session); err != nil {
			return nil, err
		}
		emitStatus(emit, "review_started")
		if _, err := s.finalizeReviewStream(ctx, session.ID, emit); err != nil {
			return nil, wrapReviewGenerationRetry(err)
		}
	} else {
		nextTurn := &domain.TrainingTurn{
			ID:             newID("turn"),
			SessionID:      session.ID,
			TurnIndex:      turn.TurnIndex + 1,
			Stage:          "question",
			Question:       evaluation.FollowupQuestion,
			ExpectedPoints: evaluation.FollowupPoints,
		}
		session.Status = domain.StatusWaitingAnswer
		if err := s.repo.SaveSession(ctx, session); err != nil {
			return nil, err
		}
		if err := s.repo.InsertTurn(ctx, nextTurn); err != nil {
			return nil, err
		}
		emitStatus(emit, "followup_ready")
	}

	return s.repo.GetSession(ctx, session.ID)
}

func (s *Service) loadAnswerContext(ctx context.Context, session *domain.TrainingSession) (
	[]domain.RepoChunk, *domain.AnalyzeJobTargetResponse, map[string]float64, error,
) {
	contextChunks, err := s.lookupSessionContext(ctx, session)
	if err != nil {
		return nil, nil, nil, err
	}
	jobTargetAnalysis, err := s.getJobTargetAnalysisSnapshotForSession(ctx, session)
	if err != nil {
		return nil, nil, nil, err
	}
	scoreWeights, err := s.resolveScoreWeights(ctx, session)
	if err != nil {
		return nil, nil, nil, err
	}
	return contextChunks, jobTargetAnalysis, scoreWeights, nil
}

func (s *Service) computeSessionScore(session *domain.TrainingSession) float64 {
	var total float64
	var count int
	for _, t := range session.Turns {
		if t.Evaluation != nil {
			total += t.Evaluation.Score
			count++
		}
	}
	if count == 0 {
		return 0
	}
	return math.Round((total/float64(count))*10) / 10
}

func (s *Service) resolveScoreWeights(
	ctx context.Context,
	session *domain.TrainingSession,
) (map[string]float64, error) {
	if session.Mode != domain.ModeBasics || session.Topic == "" {
		return nil, nil
	}

	templates, err := s.repo.ListQuestionTemplatesByTopic(ctx, session.Topic)
	if err != nil {
		return nil, err
	}
	for _, template := range templates {
		if len(template.ScoreWeights) == 0 {
			continue
		}

		weights := make(map[string]float64, len(template.ScoreWeights))
		for key, value := range template.ScoreWeights {
			weights[key] = value
		}
		return weights, nil
	}

	return nil, nil
}
