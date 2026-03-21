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
	jobTargetAnalysis, err := s.getJobTargetAnalysisSnapshotForSession(ctx, updatedSession)
	if err != nil {
		return nil, err
	}

	review, err := s.sidecar.GenerateReview(ctx, domain.GenerateReviewRequest{
		Session:           updatedSession,
		Project:           updatedSession.Project,
		Turns:             updatedSession.Turns,
		JobTargetAnalysis: jobTargetAnalysis,
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
	jobTargetAnalysis, err := s.getJobTargetAnalysisSnapshotForSession(ctx, updatedSession)
	if err != nil {
		return nil, err
	}

	review, err := s.sidecar.GenerateReviewStream(ctx, domain.GenerateReviewRequest{
		Session:           updatedSession,
		Project:           updatedSession.Project,
		Turns:             updatedSession.Turns,
		JobTargetAnalysis: jobTargetAnalysis,
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

	_ = s.scheduleReview(ctx, session, review)

	return session, nil
}

func (s *Service) scheduleReview(ctx context.Context, session *domain.TrainingSession, review *domain.ReviewCard) error {
	nextReview := time.Now().UTC().AddDate(0, 0, 1)
	return s.repo.CreateReviewSchedule(ctx, &domain.ReviewScheduleItem{
		SessionID:    session.ID,
		ReviewCardID: review.ID,
		Topic:        session.Topic,
		NextReviewAt: nextReview,
		IntervalDays: 1,
		EaseFactor:   2.5,
	})
}

func (s *Service) ListDueReviews(ctx context.Context) ([]domain.ReviewScheduleItem, error) {
	return s.repo.ListDueReviews(ctx, time.Now().UTC())
}

func (s *Service) CompleteDueReview(ctx context.Context, id int64, score float64) error {
	return s.repo.CompleteReviewSchedule(ctx, id, score)
}

func (s *Service) ListSessions(ctx context.Context, req domain.ListSessionsRequest) (*domain.PaginatedList[domain.TrainingSessionSummary], error) {
	return s.repo.ListSessions(ctx, req)
}

func (s *Service) GetWeaknessTrends(ctx context.Context) ([]domain.WeaknessTrend, error) {
	return s.repo.GetWeaknessTrends(ctx, 5)
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
