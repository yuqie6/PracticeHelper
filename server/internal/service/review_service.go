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
	agentContext, retrievalTrace, err := s.getAgentContextDetailed(ctx, agentContextParams{
		Topic:               updatedSession.Topic,
		ProjectID:           updatedSession.ProjectID,
		JobTargetID:         updatedSession.JobTargetID,
		SessionID:           updatedSession.ID,
		WeaknessLimit:       5,
		ObservationLimit:    5,
		SessionSummaryLimit: 3,
		KnowledgeNodeLimit:  10,
	})
	if err != nil {
		return nil, err
	}

	startedAt := time.Now()
	review, sideEffects, commandResults, promptMeta, err := s.sidecar.GenerateReview(
		ctx,
		domain.GenerateReviewRequest{
			Session:           updatedSession,
			Project:           updatedSession.Project,
			Turns:             updatedSession.Turns,
			PromptSetID:       updatedSession.PromptSetID,
			PromptOverlay:     updatedSession.PromptOverlay,
			JobTargetAnalysis: jobTargetAnalysis,
			AgentContext:      agentContext,
		},
	)
	if err != nil {
		return nil, err
	}
	if sideEffects == nil {
		sideEffects = &domain.GenerateReviewSideEffects{}
	}
	review.RetrievalTrace = retrievalTrace
	reviewLatencyMs := latencyMsSince(startedAt)
	defer s.recordEvaluationLog(ctx, updatedSession.ID, "", "generate_review", reviewLatencyMs, promptMeta)
	appendPromptTrace(
		promptMeta,
		nil,
		"generate_review",
		"persist",
		"info",
		"persist_started",
		"Go 开始持久化 review 和相关副作用。",
		map[string]any{"section": "generate_review"},
	)

	return s.persistReview(
		ctx,
		updatedSession,
		review,
		sideEffects,
		commandResults,
		promptMeta,
		nil,
		"generate_review",
	)
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
	agentContext, retrievalTrace, err := s.getAgentContextDetailed(ctx, agentContextParams{
		Topic:               updatedSession.Topic,
		ProjectID:           updatedSession.ProjectID,
		JobTargetID:         updatedSession.JobTargetID,
		SessionID:           updatedSession.ID,
		WeaknessLimit:       5,
		ObservationLimit:    5,
		SessionSummaryLimit: 3,
		KnowledgeNodeLimit:  10,
	})
	if err != nil {
		return nil, err
	}

	startedAt := time.Now()
	review, sideEffects, commandResults, promptMeta, err := s.sidecar.GenerateReviewStream(
		ctx,
		domain.GenerateReviewRequest{
			Session:           updatedSession,
			Project:           updatedSession.Project,
			Turns:             updatedSession.Turns,
			PromptSetID:       updatedSession.PromptSetID,
			PromptOverlay:     updatedSession.PromptOverlay,
			JobTargetAnalysis: jobTargetAnalysis,
			AgentContext:      agentContext,
		},
		emit,
	)
	if err != nil {
		return nil, err
	}
	if sideEffects == nil {
		sideEffects = &domain.GenerateReviewSideEffects{}
	}
	review.RetrievalTrace = retrievalTrace
	reviewLatencyMs := latencyMsSince(startedAt)
	defer s.recordEvaluationLog(
		ctx,
		updatedSession.ID,
		"",
		"generate_review_stream",
		reviewLatencyMs,
		promptMeta,
	)
	appendPromptTrace(
		promptMeta,
		emit,
		"generate_review_stream",
		"persist",
		"info",
		"persist_started",
		"Go 开始持久化 review 和相关副作用。",
		map[string]any{"section": "generate_review"},
	)

	savedSession, err := s.persistReview(
		ctx,
		updatedSession,
		review,
		sideEffects,
		commandResults,
		promptMeta,
		emit,
		"generate_review_stream",
	)
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
	sideEffects *domain.GenerateReviewSideEffects,
	commandResults []domain.AgentCommandResult,
	promptMeta *domain.PromptExecutionMeta,
	emit func(domain.StreamEvent) error,
	flowName string,
) (*domain.TrainingSession, error) {
	if sideEffects != nil && review.RecommendedNext == nil && sideEffects.RecommendedNext != nil {
		review.RecommendedNext = sideEffects.RecommendedNext
	}
	if err := s.applyReviewPathDecision(ctx, session, review, sideEffects, commandResults); err != nil {
		appendPersistFailureTrace(promptMeta, emit, flowName, "recommended_next", err)
		return nil, fmt.Errorf("normalize recommended next: %w", err)
	}
	if review.RecommendedNext != nil {
		appendPromptTrace(
			promptMeta,
			emit,
			flowName,
			"persist",
			"success",
			"persist_recommended_next",
			"下一轮训练建议已归一化并准备落库。",
			map[string]any{
				"section": "recommended_next",
				"applied": 1,
			},
		)
	}
	review.ID = newID("review")
	review.SessionID = session.ID
	review.PromptSetID = session.PromptSetID
	review.PromptOverlay = session.PromptOverlay
	review.PromptOverlayHash = session.PromptOverlayHash
	review.PromptSet = session.PromptSet
	if err := s.repo.CreateReview(ctx, review); err != nil {
		appendPersistFailureTrace(promptMeta, emit, flowName, "review", err)
		return nil, fmt.Errorf("create review: %w", err)
	}
	appendPromptTrace(
		promptMeta,
		emit,
		flowName,
		"persist",
		"success",
		"persist_review",
		"review 已落库。",
		map[string]any{
			"section": "review",
			"applied": 1,
		},
	)

	endedAt := time.Now().UTC()
	session.ReviewID = review.ID
	session.Status = domain.StatusCompleted
	session.EndedAt = &endedAt
	if err := s.repo.SaveSession(ctx, session); err != nil {
		appendPersistFailureTrace(promptMeta, emit, flowName, "session", err)
		return nil, err
	}
	summary := buildSessionMemorySummary(session, review)
	if err := s.repo.UpsertSessionMemorySummary(ctx, summary); err != nil {
		appendPersistFailureTrace(promptMeta, emit, flowName, "session_memory_summary", err)
		return nil, fmt.Errorf("save session memory summary: %w", err)
	}
	if summary != nil && summary.ID != "" {
		s.syncOrQueueMemoryEmbeddings(ctx, []domain.MemoryRef{{
			RefTable: "session_memory_summaries",
			RefID:    summary.ID,
		}})
	}
	if err := s.applyGenerateReviewSideEffects(
		ctx,
		session,
		sideEffects,
		promptMeta,
		emit,
		flowName,
	); err != nil {
		return nil, err
	}

	_ = s.scheduleReview(ctx, session, review)

	return session, nil
}

func (s *Service) applyGenerateReviewSideEffects(
	ctx context.Context,
	session *domain.TrainingSession,
	sideEffects *domain.GenerateReviewSideEffects,
	promptMeta *domain.PromptExecutionMeta,
	emit func(domain.StreamEvent) error,
	flowName string,
) error {
	if sideEffects == nil {
		return nil
	}
	if len(sideEffects.Observations) > 0 {
		stats, err := s.repo.CreateObservationsWithStats(ctx, session.ID, sideEffects.Observations)
		if err != nil {
			appendPersistFailureTrace(promptMeta, emit, flowName, "observations", err)
			return fmt.Errorf("create review observations: %w", err)
		}
		appendPromptTrace(
			promptMeta,
			emit,
			flowName,
			"persist",
			"success",
			"persist_observations",
			"review 观察记录已写入长期记忆。",
			map[string]any{
				"section": "observations",
				"applied": stats.Applied,
				"skipped": stats.Skipped,
				"deduped": stats.Deduped,
			},
		)
		s.syncOrQueueMemoryEmbeddings(ctx, collectObservationMemoryRefs(sideEffects.Observations))
	}
	if len(sideEffects.KnowledgeUpdates) > 0 {
		applied, err := s.repo.UpsertKnowledgeNodesWithCount(ctx, session.ID, sideEffects.KnowledgeUpdates)
		if err != nil {
			appendPersistFailureTrace(promptMeta, emit, flowName, "knowledge_updates", err)
			return fmt.Errorf("upsert review knowledge updates: %w", err)
		}
		appendPromptTrace(
			promptMeta,
			emit,
			flowName,
			"persist",
			"success",
			"persist_knowledge_updates",
			"review 的知识图谱更新已回写。",
			map[string]any{
				"section": "knowledge_updates",
				"applied": applied,
			},
		)
	}
	return nil
}

func buildSessionMemorySummary(
	session *domain.TrainingSession,
	review *domain.ReviewCard,
) *domain.SessionMemorySummary {
	if session == nil || review == nil {
		return nil
	}

	growthSignals := make([]string, 0, 1)
	if session.TotalScore >= 75 {
		growthSignals = append(growthSignals, fmt.Sprintf("本轮总分 %.1f，说明主线回答已经能稳定站住。", session.TotalScore))
	}

	recommendedFocus := append([]string(nil), review.NextTrainingFocus...)
	if len(recommendedFocus) == 0 {
		recommendedFocus = append(recommendedFocus, review.SuggestedTopics...)
	}

	return &domain.SessionMemorySummary{
		SessionID:        session.ID,
		Mode:             session.Mode,
		Topic:            session.Topic,
		ProjectID:        session.ProjectID,
		JobTargetID:      session.JobTargetID,
		PromptSetID:      session.PromptSetID,
		Summary:          review.Overall,
		Strengths:        append([]string(nil), review.Highlights...),
		Gaps:             append([]string(nil), review.Gaps...),
		Misconceptions:   nil,
		GrowthSignals:    growthSignals,
		RecommendedFocus: recommendedFocus,
		Salience:         0.8,
	}
}

func (s *Service) scheduleReview(ctx context.Context, session *domain.TrainingSession, review *domain.ReviewCard) error {
	nextReview := time.Now().UTC().AddDate(0, 0, 1)
	items := s.buildReviewScheduleItems(ctx, session, review, nextReview)
	for _, item := range items {
		schedule := item
		if err := s.repo.CreateReviewSchedule(ctx, &schedule); err != nil {
			return fmt.Errorf("create review schedule: %w", err)
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
		return fmt.Errorf("load review schedule: %w", err)
	}
	if item == nil {
		return ErrReviewScheduleNotFound
	}

	session, err := s.repo.GetSession(ctx, item.SessionID)
	if err != nil {
		return fmt.Errorf("load session for due review: %w", err)
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
