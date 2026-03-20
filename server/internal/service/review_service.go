package service

import (
	"context"
	"fmt"
	"time"

	"practicehelper/server/internal/domain"
)

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

	savedSession, err := s.persistReview(ctx, updatedSession, review)
	if err != nil {
		return nil, err
	}

	emitStatus(emit, "review_saved")
	return savedSession, nil
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
		return nil, wrapReviewGenerationRetry(err)
	}

	return updatedSession, nil
}

func wrapReviewGenerationRetry(err error) error {
	return fmt.Errorf("%w: %v", ErrReviewGenerationRetry, err)
}
