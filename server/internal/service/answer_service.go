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
		scoreWeights, err := s.resolveScoreWeights(ctx, session)
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
			ScoreWeights:   scoreWeights,
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
			_ = s.coolDownSessionWeakness(ctx, session, turn.Question, evaluation.WeaknessHits)
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
		scoreWeights, err := s.resolveScoreWeights(ctx, session)
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
			ScoreWeights:   scoreWeights,
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
			return nil, wrapReviewGenerationRetry(err)
		}

		if evaluation.Score >= 75 {
			_ = s.coolDownSessionWeakness(
				ctx,
				updatedSession,
				turn.Question+" "+turn.FollowupQuestion,
				evaluation.WeaknessHits,
			)
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
		emitStatus(emit, "answer_received")

		contextChunks, err := s.lookupSessionContext(ctx, session)
		if err != nil {
			s.restoreSessionStatus(ctx, session.ID, previousStatus)
			return nil, err
		}
		scoreWeights, err := s.resolveScoreWeights(ctx, session)
		if err != nil {
			s.restoreSessionStatus(ctx, session.ID, previousStatus)
			return nil, err
		}

		emitStatus(emit, "evaluation_started")

		evaluation, err := s.sidecar.EvaluateAnswerStream(ctx, domain.EvaluateAnswerRequest{
			Mode:           session.Mode,
			Topic:          session.Topic,
			Project:        session.Project,
			Question:       turn.Question,
			ExpectedPoints: turn.ExpectedPoints,
			Answer:         request.Answer,
			ContextChunks:  contextChunks,
			IsFollowup:     false,
			ScoreWeights:   scoreWeights,
		}, emit)
		if err != nil {
			s.restoreSessionStatus(ctx, session.ID, previousStatus)
			return nil, err
		}
		emitStatus(emit, "feedback_ready")

		turn.Answer = request.Answer
		turn.Evaluation = evaluation
		turn.FollowupQuestion = evaluation.FollowupQuestion
		turn.FollowupExpectedPoint = evaluation.FollowupPoints
		turn.WeaknessHits = mergeWeaknessHits(turn.WeaknessHits, evaluation.WeaknessHits)

		if err := s.repo.SaveTurn(ctx, &turn); err != nil {
			s.restoreSessionStatus(ctx, session.ID, previousStatus)
			return nil, err
		}
		emitStatus(emit, "answer_saved")
		if err := s.repo.UpsertWeaknesses(ctx, session.ID, evaluation.WeaknessHits); err != nil {
			s.restoreSessionStatus(ctx, session.ID, previousStatus)
			return nil, err
		}

		session.Status = domain.StatusFollowup
		session.TotalScore = evaluation.Score
		if err := s.repo.SaveSession(ctx, session); err != nil {
			return nil, err
		}
		emitStatus(emit, "followup_ready")

		if evaluation.Score >= 75 {
			_ = s.coolDownSessionWeakness(ctx, session, turn.Question, evaluation.WeaknessHits)
		}
	case domain.StatusFollowup:
		previousStatus := session.Status
		if err := s.claimSessionForAnswer(ctx, session, previousStatus); err != nil {
			return nil, err
		}
		emitStatus(emit, "answer_received")

		contextChunks, err := s.lookupSessionContext(ctx, session)
		if err != nil {
			s.restoreSessionStatus(ctx, session.ID, previousStatus)
			return nil, err
		}
		scoreWeights, err := s.resolveScoreWeights(ctx, session)
		if err != nil {
			s.restoreSessionStatus(ctx, session.ID, previousStatus)
			return nil, err
		}
		emitStatus(emit, "evaluation_started")

		evaluation, err := s.sidecar.EvaluateAnswerStream(ctx, domain.EvaluateAnswerRequest{
			Mode:           session.Mode,
			Topic:          session.Topic,
			Project:        session.Project,
			Question:       turn.FollowupQuestion,
			ExpectedPoints: turn.FollowupExpectedPoint,
			Answer:         request.Answer,
			ContextChunks:  contextChunks,
			IsFollowup:     true,
			ScoreWeights:   scoreWeights,
		}, emit)
		if err != nil {
			s.restoreSessionStatus(ctx, session.ID, previousStatus)
			return nil, err
		}
		emitStatus(emit, "feedback_ready")

		turn.FollowupAnswer = request.Answer
		turn.FollowupEvaluation = evaluation
		turn.WeaknessHits = mergeWeaknessHits(turn.WeaknessHits, evaluation.WeaknessHits)

		if err := s.repo.SaveTurn(ctx, &turn); err != nil {
			s.restoreSessionStatus(ctx, session.ID, previousStatus)
			return nil, err
		}
		emitStatus(emit, "answer_saved")
		if err := s.repo.UpsertWeaknesses(ctx, session.ID, evaluation.WeaknessHits); err != nil {
			s.restoreSessionStatus(ctx, session.ID, previousStatus)
			return nil, err
		}

		session.Status = domain.StatusReviewPending
		session.TotalScore = math.Round(((session.TotalScore+evaluation.Score)/2)*10) / 10
		if err := s.repo.SaveSession(ctx, session); err != nil {
			return nil, err
		}
		emitStatus(emit, "review_started")

		updatedSession, err := s.finalizeReviewStream(ctx, session.ID, emit)
		if err != nil {
			return nil, wrapReviewGenerationRetry(err)
		}

		if evaluation.Score >= 75 {
			_ = s.coolDownSessionWeakness(
				ctx,
				updatedSession,
				turn.Question+" "+turn.FollowupQuestion,
				evaluation.WeaknessHits,
			)
		}
	default:
		return nil, classifySubmitAnswerStatus(session.Status)
	}

	return s.repo.GetSession(ctx, session.ID)
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
		// 当前 basics 题库按 topic 只维护一套评分权重；如果未来同 topic 出现多套 rubric，
		// 就需要把选中的 template ID 一起持久化到 turn，而不是继续在答题时反查 topic。
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
