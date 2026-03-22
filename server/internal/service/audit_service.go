package service

import (
	"context"

	"practicehelper/server/internal/domain"
)

func (s *Service) recordEvaluationLog(
	ctx context.Context,
	sessionID string,
	turnID string,
	flowName string,
	latencyMs float64,
	meta *domain.PromptExecutionMeta,
) {
	if s.repo == nil || flowName == "" {
		return
	}

	_ = s.repo.CreateEvaluationLog(
		ctx,
		sessionID,
		turnID,
		flowName,
		metaValue(meta, func(item *domain.PromptExecutionMeta) string { return item.ModelName }),
		metaValue(meta, func(item *domain.PromptExecutionMeta) string { return item.PromptSetID }),
		metaValue(meta, func(item *domain.PromptExecutionMeta) string { return item.PromptHash }),
		metaValue(meta, func(item *domain.PromptExecutionMeta) string { return item.RawOutput }),
		metaTrace(meta),
		latencyMs,
	)
}

func metaValue(
	meta *domain.PromptExecutionMeta,
	pick func(*domain.PromptExecutionMeta) string,
) string {
	if meta == nil {
		return ""
	}
	return pick(meta)
}

func metaTrace(meta *domain.PromptExecutionMeta) *domain.RuntimeTrace {
	if meta == nil {
		return nil
	}
	return meta.RuntimeTrace
}
