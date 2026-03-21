package service

import (
	"context"
	"fmt"
	"sort"
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

	startedAt := time.Now()
	review, promptMeta, err := s.sidecar.GenerateReview(ctx, domain.GenerateReviewRequest{
		Session:           updatedSession,
		Project:           updatedSession.Project,
		Turns:             updatedSession.Turns,
		PromptSetID:       updatedSession.PromptSetID,
		JobTargetAnalysis: jobTargetAnalysis,
	})
	if err != nil {
		return nil, err
	}
	s.recordEvaluationLog(ctx, updatedSession.ID, "", "generate_review", startedAt, promptMeta)

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

	startedAt := time.Now()
	review, promptMeta, err := s.sidecar.GenerateReviewStream(ctx, domain.GenerateReviewRequest{
		Session:           updatedSession,
		Project:           updatedSession.Project,
		Turns:             updatedSession.Turns,
		PromptSetID:       updatedSession.PromptSetID,
		JobTargetAnalysis: jobTargetAnalysis,
	}, emit)
	if err != nil {
		return nil, err
	}
	s.recordEvaluationLog(ctx, updatedSession.ID, "", "generate_review_stream", startedAt, promptMeta)

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
	review.PromptSetID = session.PromptSetID
	review.PromptSet = session.PromptSet
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
	items := s.buildReviewScheduleItems(ctx, session, review, nextReview)
	for _, item := range items {
		schedule := item
		if err := s.repo.CreateReviewSchedule(ctx, &schedule); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) ListDueReviews(ctx context.Context) ([]domain.ReviewScheduleItem, error) {
	return s.repo.ListDueReviews(ctx, time.Now().UTC())
}

func (s *Service) CompleteDueReview(ctx context.Context, id int64) error {
	item, err := s.repo.GetReviewSchedule(ctx, id)
	if err != nil {
		return err
	}
	if item == nil {
		return ErrReviewScheduleNotFound
	}

	session, err := s.repo.GetSession(ctx, item.SessionID)
	if err != nil {
		return err
	}
	if session == nil {
		return ErrSessionNotFound
	}

	score := session.TotalScore
	if score <= 0 {
		score = 60
	}
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

func (s *Service) buildReviewScheduleItems(
	ctx context.Context,
	session *domain.TrainingSession,
	review *domain.ReviewCard,
	nextReview time.Time,
) []domain.ReviewScheduleItem {
	hits := collectReviewWeaknessHits(session)
	items := make([]domain.ReviewScheduleItem, 0, len(hits))
	for _, hit := range hits {
		tag, err := s.repo.GetWeaknessTag(ctx, hit.Kind, hit.Label)
		if err != nil || tag == nil {
			continue
		}
		items = append(items, domain.ReviewScheduleItem{
			SessionID:     session.ID,
			ReviewCardID:  review.ID,
			WeaknessTagID: tag.ID,
			WeaknessKind:  tag.Kind,
			WeaknessLabel: tag.Label,
			Topic:         resolveReviewScheduleTopic(session, review, hit),
			NextReviewAt:  nextReview,
			IntervalDays:  1,
			EaseFactor:    2.5,
		})
	}

	if len(items) > 0 {
		return items
	}

	return []domain.ReviewScheduleItem{{
		SessionID:    session.ID,
		ReviewCardID: review.ID,
		Topic:        fallbackReviewScheduleTopic(session, review),
		NextReviewAt: nextReview,
		IntervalDays: 1,
		EaseFactor:   2.5,
	}}
}

func collectReviewWeaknessHits(session *domain.TrainingSession) []domain.WeaknessHit {
	if session == nil {
		return nil
	}

	keyed := make(map[string]domain.WeaknessHit)
	for _, turn := range session.Turns {
		for _, hit := range turn.WeaknessHits {
			if hit.Kind == "" || hit.Label == "" {
				continue
			}
			key := hit.Kind + "\x00" + hit.Label
			current, exists := keyed[key]
			if !exists || hit.Severity > current.Severity {
				keyed[key] = hit
			}
		}
	}

	items := make([]domain.WeaknessHit, 0, len(keyed))
	for _, item := range keyed {
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Severity != items[j].Severity {
			return items[i].Severity > items[j].Severity
		}
		if items[i].Kind != items[j].Kind {
			return items[i].Kind < items[j].Kind
		}
		return items[i].Label < items[j].Label
	})
	return items
}

func resolveReviewScheduleTopic(
	session *domain.TrainingSession,
	review *domain.ReviewCard,
	hit domain.WeaknessHit,
) string {
	candidates := make([]string, 0, 6)
	if hit.Kind == "topic" {
		candidates = append(candidates, hit.Label)
	}
	candidates = append(candidates, matchBasicsTopics(hit.Label)...)
	if review != nil {
		if review.RecommendedNext != nil &&
			review.RecommendedNext.Mode == domain.ModeBasics &&
			review.RecommendedNext.Topic != "" {
			candidates = append(candidates, review.RecommendedNext.Topic)
		}
		candidates = append(candidates, review.SuggestedTopics...)
	}
	if session != nil && session.Topic != "" {
		candidates = append(candidates, session.Topic)
	}

	for _, candidate := range candidates {
		normalized := normalizeBasicsTopic(candidate)
		if normalized != "" && normalized != domain.BasicsTopicMixed {
			return normalized
		}
	}

	return domain.BasicsTopicGo
}

func fallbackReviewScheduleTopic(
	session *domain.TrainingSession,
	review *domain.ReviewCard,
) string {
	if review != nil {
		if review.RecommendedNext != nil &&
			review.RecommendedNext.Mode == domain.ModeBasics &&
			review.RecommendedNext.Topic != "" {
			if normalized := normalizeBasicsTopic(review.RecommendedNext.Topic); normalized != "" && normalized != domain.BasicsTopicMixed {
				return normalized
			}
		}
		for _, topic := range review.SuggestedTopics {
			if normalized := normalizeBasicsTopic(topic); normalized != "" && normalized != domain.BasicsTopicMixed {
				return normalized
			}
		}
	}
	if session != nil {
		if normalized := normalizeBasicsTopic(session.Topic); normalized != "" && normalized != domain.BasicsTopicMixed {
			return normalized
		}
	}
	return domain.BasicsTopicGo
}
