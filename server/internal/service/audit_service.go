package service

import (
	"context"
	"time"

	"practicehelper/server/internal/domain"
)

func (s *Service) recordEvaluationLog(
	ctx context.Context,
	sessionID string,
	turnID string,
	flowName string,
	startedAt time.Time,
	meta *domain.PromptExecutionMeta,
) {
	if s.repo == nil || flowName == "" {
		return
	}

	latencyMs := float64(time.Since(startedAt).Microseconds()) / 1000
	_ = s.repo.CreateEvaluationLog(
		ctx,
		sessionID,
		turnID,
		flowName,
		metaValue(meta, func(item *domain.PromptExecutionMeta) string { return item.ModelName }),
		metaValue(meta, func(item *domain.PromptExecutionMeta) string { return item.PromptSetID }),
		metaValue(meta, func(item *domain.PromptExecutionMeta) string { return item.PromptHash }),
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
