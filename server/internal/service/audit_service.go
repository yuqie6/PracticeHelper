package service

import (
	"context"
	"time"
)

func (s *Service) recordEvaluationLog(
	ctx context.Context,
	sessionID string,
	turnID string,
	flowName string,
	startedAt time.Time,
) {
	if s.repo == nil || flowName == "" {
		return
	}

	latencyMs := float64(time.Since(startedAt).Microseconds()) / 1000
	_ = s.repo.CreateEvaluationLog(ctx, sessionID, turnID, flowName, "", latencyMs)
}
