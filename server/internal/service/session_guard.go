package service

import (
	"context"
	"fmt"

	"practicehelper/server/internal/domain"
)

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
