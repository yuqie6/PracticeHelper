package service

import (
	"context"
	"fmt"
	"strings"

	"practicehelper/server/internal/domain"
)

func (s *Service) ExecuteAgentCommand(
	ctx context.Context,
	command domain.AgentCommandEnvelope,
) (*domain.AgentCommandResult, error) {
	if strings.TrimSpace(command.CommandID) == "" {
		return nil, ErrInvalidAgentCommand
	}
	if strings.TrimSpace(command.CommandType) == "" {
		return nil, ErrInvalidAgentCommand
	}
	if strings.TrimSpace(command.SessionID) == "" {
		return nil, ErrInvalidAgentCommand
	}
	if strings.TrimSpace(command.IdempotencyKey) == "" {
		return nil, ErrInvalidAgentCommand
	}

	session, err := s.repo.GetSession(ctx, command.SessionID)
	if err != nil {
		return nil, err
	}
	if session == nil {
		return nil, ErrSessionNotFound
	}

	switch command.CommandType {
	case domain.AgentCommandTypeTransitionSession:
		return s.resolveTransitionSessionCommand(session, command)
	case domain.AgentCommandTypeUpsertReviewPath:
		return s.resolveReviewPathCommand(ctx, session, command)
	default:
		return nil, ErrUnsupportedAgentCommand
	}
}

func (s *Service) resolveTransitionSessionCommand(
	session *domain.TrainingSession,
	command domain.AgentCommandEnvelope,
) (*domain.AgentCommandResult, error) {
	if session == nil {
		return nil, ErrSessionNotFound
	}
	decision, _ := command.Payload["decision"].(string)
	decision = strings.TrimSpace(decision)
	if decision != domain.DepthSignalSkipFollowup && decision != domain.DepthSignalExtend {
		return nil, ErrInvalidAgentCommand
	}

	result := &domain.AgentCommandResult{
		CommandID: command.CommandID,
		Status:    domain.AgentCommandStatusDeferred,
		Data: map[string]any{
			"resolved_depth_signal": domain.DepthSignalNormal,
			"resolved_max_turns":    session.MaxTurns,
			"session_status":        session.Status,
		},
	}

	if session.Status != domain.StatusEvaluating {
		result.Status = domain.AgentCommandStatusRejected
		result.ErrorCode = "session_status_not_allowed"
		result.ErrorMessage = fmt.Sprintf(
			"transition_session only supports evaluating sessions, got %s",
			session.Status,
		)
		return result, nil
	}

	if decision == domain.DepthSignalSkipFollowup {
		result.Data["resolved_depth_signal"] = domain.DepthSignalSkipFollowup
		return result, nil
	}

	if session.MaxTurns >= 6 {
		result.Status = domain.AgentCommandStatusRejected
		result.ErrorCode = "turn_limit_reached"
		result.ErrorMessage = "session already reached the max turn limit"
		return result, nil
	}

	result.Data["resolved_depth_signal"] = domain.DepthSignalExtend
	result.Data["resolved_max_turns"] = session.MaxTurns + 1
	return result, nil
}

func (s *Service) resolveReviewPathCommand(
	ctx context.Context,
	session *domain.TrainingSession,
	command domain.AgentCommandEnvelope,
) (*domain.AgentCommandResult, error) {
	if session == nil {
		return nil, ErrSessionNotFound
	}

	if session.Status != domain.StatusReviewPending && session.Status != domain.StatusEvaluating {
		return &domain.AgentCommandResult{
			CommandID: command.CommandID,
			Status:    domain.AgentCommandStatusRejected,
			ErrorCode: "session_status_not_allowed",
			ErrorMessage: fmt.Sprintf(
				"upsert_review_path only supports review_pending or evaluating sessions, got %s",
				session.Status,
			),
			Data: map[string]any{
				"session_status": session.Status,
			},
		}, nil
	}

	reviewDraft, sideEffects := reviewPathDraftFromCommand(command.Payload)
	prerequisiteTopic, err := s.resolveReviewPath(ctx, session, reviewDraft, sideEffects)
	if err != nil {
		return nil, err
	}

	return &domain.AgentCommandResult{
		CommandID: command.CommandID,
		Status:    domain.AgentCommandStatusApplied,
		Applied:   true,
		Data: map[string]any{
			"recommended_next":    reviewDraft.RecommendedNext,
			"suggested_topics":    reviewDraft.SuggestedTopics,
			"next_training_focus": reviewDraft.NextTrainingFocus,
			"prerequisite_topic":  prerequisiteTopic,
		},
	}, nil
}

func reviewPathDraftFromCommand(
	payload map[string]any,
) (*domain.ReviewCard, *domain.GenerateReviewSideEffects) {
	review := &domain.ReviewCard{
		TopFix:            stringValue(payload["top_fix"]),
		TopFixReason:      stringValue(payload["top_fix_reason"]),
		Gaps:              stringSliceValue(payload["gaps"]),
		SuggestedTopics:   stringSliceValue(payload["suggested_topics"]),
		NextTrainingFocus: stringSliceValue(payload["next_training_focus"]),
		ScoreBreakdown:    map[string]float64{},
	}

	var sideEffects *domain.GenerateReviewSideEffects
	if next, ok := nextSessionValue(payload["recommended_next"]); ok {
		review.RecommendedNext = next
		sideEffects = &domain.GenerateReviewSideEffects{RecommendedNext: next}
	}

	return review, sideEffects
}

func stringSliceValue(value any) []string {
	items, ok := value.([]any)
	if !ok {
		return nil
	}
	normalized := make([]string, 0, len(items))
	for _, item := range items {
		text := strings.TrimSpace(stringValue(item))
		if text == "" {
			continue
		}
		normalized = append(normalized, text)
	}
	return normalized
}

func stringValue(value any) string {
	text, _ := value.(string)
	return strings.TrimSpace(text)
}

func nextSessionValue(value any) (*domain.NextSession, bool) {
	if typed, ok := value.(*domain.NextSession); ok && typed != nil {
		copy := *typed
		return &copy, true
	}
	if typed, ok := value.(domain.NextSession); ok {
		copy := typed
		return &copy, true
	}
	raw, ok := value.(map[string]any)
	if !ok {
		return nil, false
	}
	mode := stringValue(raw["mode"])
	if mode == "" {
		return nil, false
	}
	return &domain.NextSession{
		Mode:      mode,
		Topic:     stringValue(raw["topic"]),
		ProjectID: stringValue(raw["project_id"]),
		Reason:    stringValue(raw["reason"]),
	}, true
}

func resolveEvaluateAnswerCommandDecision(
	commandResults []domain.AgentCommandResult,
	sideEffects *domain.EvaluateAnswerSideEffects,
	defaultMaxTurns int,
) (string, int) {
	result, ok := latestCommandResultByType(commandResults, domain.AgentCommandTypeTransitionSession)
	if ok && result.Status == domain.AgentCommandStatusDeferred {
		signal := stringValue(result.Data["resolved_depth_signal"])
		if signal == "" {
			signal = domain.DepthSignalNormal
		}
		maxTurns := intValue(result.Data["resolved_max_turns"], defaultMaxTurns)
		return signal, maxTurns
	}
	if sideEffects == nil || sideEffects.DepthSignal == "" {
		return domain.DepthSignalNormal, defaultMaxTurns
	}
	return sideEffects.DepthSignal, defaultMaxTurns
}

func intValue(value any, fallback int) int {
	switch typed := value.(type) {
	case int:
		return typed
	case int32:
		return int(typed)
	case int64:
		return int(typed)
	case float64:
		return int(typed)
	default:
		return fallback
	}
}

func (s *Service) applyReviewPathDecision(
	ctx context.Context,
	session *domain.TrainingSession,
	review *domain.ReviewCard,
	sideEffects *domain.GenerateReviewSideEffects,
	commandResults []domain.AgentCommandResult,
) error {
	result, ok := latestCommandResultByType(commandResults, domain.AgentCommandTypeUpsertReviewPath)
	if ok && result.Status == domain.AgentCommandStatusApplied {
		if next, ok := nextSessionValue(result.Data["recommended_next"]); ok {
			review.RecommendedNext = next
		}
		if topics := stringSliceValue(result.Data["suggested_topics"]); len(topics) > 0 {
			review.SuggestedTopics = topics
		}
		if focus := stringSliceValue(result.Data["next_training_focus"]); len(focus) > 0 {
			review.NextTrainingFocus = focus
		}
		if prerequisiteTopic := stringValue(result.Data["prerequisite_topic"]); prerequisiteTopic != "" {
			return s.repo.EnsureKnowledgePrerequisiteEdge(
				ctx,
				prerequisiteTopic,
				normalizeBasicsTopic(session.Topic),
			)
		}
		return nil
	}
	return s.normalizeRecommendedNext(ctx, session, review, sideEffects)
}

func latestCommandResultByType(
	commandResults []domain.AgentCommandResult,
	commandType string,
) (domain.AgentCommandResult, bool) {
	for index := len(commandResults) - 1; index >= 0; index-- {
		item := commandResults[index]
		if strings.TrimSpace(item.CommandType) == commandType {
			return item, true
		}
	}
	if len(commandResults) == 1 && strings.TrimSpace(commandResults[0].CommandType) == "" {
		return commandResults[0], true
	}
	return domain.AgentCommandResult{}, false
}
