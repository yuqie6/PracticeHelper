package service

import (
	"context"
	"fmt"
	"math"
	"time"

	"practicehelper/server/internal/domain"
)

type evaluateFunc func(
	ctx context.Context,
	request domain.EvaluateAnswerRequest,
) (*domain.EvaluationResult, *domain.EvaluateAnswerSideEffects, *domain.PromptExecutionMeta, error)

type finalizeFunc func(ctx context.Context, sessionID string) (*domain.TrainingSession, error)

func (s *Service) SubmitAnswer(ctx context.Context, sessionID string, request domain.SubmitAnswerRequest) (*domain.TrainingSession, error) {
	return s.submitAnswerCore(
		ctx,
		sessionID,
		request,
		func(ctx context.Context, request domain.EvaluateAnswerRequest) (*domain.EvaluationResult, *domain.EvaluateAnswerSideEffects, *domain.PromptExecutionMeta, error) {
			return s.sidecar.EvaluateAnswer(ctx, request)
		},
		func(ctx context.Context, sessionID string) (*domain.TrainingSession, error) {
			return s.finalizeReview(ctx, sessionID)
		},
		nil,
		"evaluate_answer",
	)
}

func (s *Service) SubmitAnswerStream(
	ctx context.Context,
	sessionID string,
	request domain.SubmitAnswerRequest,
	emit func(domain.StreamEvent) error,
) (*domain.TrainingSession, error) {
	return s.submitAnswerCore(
		ctx,
		sessionID,
		request,
		func(ctx context.Context, request domain.EvaluateAnswerRequest) (*domain.EvaluationResult, *domain.EvaluateAnswerSideEffects, *domain.PromptExecutionMeta, error) {
			return s.sidecar.EvaluateAnswerStream(ctx, request, emit)
		},
		func(ctx context.Context, sessionID string) (*domain.TrainingSession, error) {
			return s.finalizeReviewStream(ctx, sessionID, emit)
		},
		emit,
		"evaluate_answer_stream",
	)
}

func (s *Service) submitAnswerCore(
	ctx context.Context,
	sessionID string,
	request domain.SubmitAnswerRequest,
	evaluate evaluateFunc,
	finalize finalizeFunc,
	emit func(domain.StreamEvent) error,
	flowName string,
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
	agentContext, err := s.getAgentContext(ctx, agentContextParams{
		Topic:               session.Topic,
		ProjectID:           session.ProjectID,
		JobTargetID:         session.JobTargetID,
		SessionID:           session.ID,
		WeaknessLimit:       5,
		ObservationLimit:    4,
		SessionSummaryLimit: 3,
		KnowledgeNodeLimit:  8,
	})
	if err != nil {
		s.restoreSessionStatus(ctx, session.ID, previousStatus)
		return nil, err
	}

	isLastTurn := turn.TurnIndex >= session.MaxTurns
	emitStatus(emit, "evaluation_started")

	evalStart := time.Now()
	evaluation, sideEffects, promptMeta, err := evaluate(ctx, domain.EvaluateAnswerRequest{
		Mode:              session.Mode,
		Topic:             session.Topic,
		PromptSetID:       session.PromptSetID,
		Project:           session.Project,
		Question:          turn.Question,
		ExpectedPoints:    turn.ExpectedPoints,
		Answer:            request.Answer,
		ContextChunks:     contextChunks,
		TurnIndex:         turn.TurnIndex,
		MaxTurns:          session.MaxTurns,
		ScoreWeights:      scoreWeights,
		JobTargetAnalysis: jobTargetAnalysis,
		AgentContext:      agentContext,
	})
	if err != nil {
		s.restoreSessionStatus(ctx, session.ID, previousStatus)
		return nil, err
	}
	s.recordEvaluationLog(ctx, session.ID, turn.ID, flowName, evalStart, promptMeta)
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
	if err := s.applyEvaluateAnswerSideEffects(ctx, session, sideEffects); err != nil {
		s.restoreSessionStatus(ctx, session.ID, previousStatus)
		return nil, err
	}

	session.TotalScore = s.computeSessionScore(session)

	if evaluation.Score >= 75 {
		_ = s.coolDownSessionWeakness(ctx, session, turn.Question, evaluation.WeaknessHits)
	}

	switch sideEffects.DepthSignal {
	case domain.DepthSignalSkipFollowup:
		isLastTurn = true
	case domain.DepthSignalExtend:
		if turn.TurnIndex >= session.MaxTurns && session.MaxTurns < 6 {
			session.MaxTurns++
			isLastTurn = false
		}
	}

	if isLastTurn {
		session.Status = domain.StatusReviewPending
		if err := s.repo.SaveSession(ctx, session); err != nil {
			return nil, err
		}
		emitStatus(emit, "review_started")
		if _, err := finalize(ctx, session.ID); err != nil {
			return nil, wrapReviewGenerationRetry(err)
		}
	} else {
		nextTurn := buildFollowupTrainingTurn(session.ID, turn.TurnIndex+1, evaluation)
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

func (s *Service) applyEvaluateAnswerSideEffects(
	ctx context.Context,
	session *domain.TrainingSession,
	sideEffects *domain.EvaluateAnswerSideEffects,
) error {
	if sideEffects == nil {
		return nil
	}
	if len(sideEffects.Observations) > 0 {
		if err := s.repo.CreateObservations(ctx, session.ID, sideEffects.Observations); err != nil {
			return fmt.Errorf("create observations: %w", err)
		}
		s.syncOrQueueMemoryEmbeddings(ctx, collectObservationMemoryRefs(sideEffects.Observations))
	}
	if len(sideEffects.KnowledgeUpdates) > 0 {
		if err := s.repo.UpsertKnowledgeNodes(ctx, session.ID, sideEffects.KnowledgeUpdates); err != nil {
			return fmt.Errorf("upsert knowledge updates: %w", err)
		}
	}
	if sideEffects.DepthSignal == "" {
		sideEffects.DepthSignal = domain.DepthSignalNormal
	}
	return nil
}

func buildFollowupTrainingTurn(
	sessionID string,
	turnIndex int,
	evaluation *domain.EvaluationResult,
) *domain.TrainingTurn {
	return &domain.TrainingTurn{
		ID:             newID("turn"),
		SessionID:      sessionID,
		TurnIndex:      turnIndex,
		Stage:          "question",
		Question:       evaluation.FollowupQuestion,
		ExpectedPoints: evaluation.FollowupPoints,
	}
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
