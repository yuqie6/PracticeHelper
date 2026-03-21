package service

import (
	"context"
	"fmt"

	"practicehelper/server/internal/domain"
)

var fallbackPromptSet = domain.PromptSetSummary{
	ID:        "stable-v1",
	Label:     "Stable v1",
	Status:    "stable",
	IsDefault: true,
}

func (s *Service) ListPromptSets(ctx context.Context) ([]domain.PromptSetSummary, error) {
	if s.sidecar == nil {
		return []domain.PromptSetSummary{fallbackPromptSet}, nil
	}
	return s.sidecar.ListPromptSets(ctx)
}

func (s *Service) resolvePromptSet(
	ctx context.Context,
	requestedID string,
) (*domain.PromptSetSummary, error) {
	if requestedID == "" && s.sidecar == nil {
		promptSet := fallbackPromptSet
		return &promptSet, nil
	}

	promptSets, err := s.ListPromptSets(ctx)
	if err != nil {
		if requestedID == "" {
			promptSet := fallbackPromptSet
			return &promptSet, nil
		}
		return nil, err
	}

	var defaultPromptSet *domain.PromptSetSummary
	for _, item := range promptSets {
		current := item
		if current.IsDefault {
			defaultPromptSet = &current
		}
		if requestedID != "" && current.ID == requestedID {
			return &current, nil
		}
	}

	if requestedID == "" {
		if defaultPromptSet != nil {
			return defaultPromptSet, nil
		}
		if len(promptSets) > 0 {
			current := promptSets[0]
			return &current, nil
		}
	}

	return nil, ErrPromptSetNotFound
}

func (s *Service) ListSessionEvaluationLogs(
	ctx context.Context,
	sessionID string,
) ([]domain.EvaluationLogEntry, error) {
	session, err := s.repo.GetSession(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if session == nil {
		return nil, ErrSessionNotFound
	}
	return s.repo.ListEvaluationLogsBySession(ctx, sessionID)
}

func (s *Service) GetPromptExperiment(
	ctx context.Context,
	req domain.PromptExperimentRequest,
) (*domain.PromptExperimentReport, error) {
	if req.Left == "" || req.Right == "" || req.Left == req.Right {
		return nil, fmt.Errorf("left and right prompt sets must be different")
	}
	if req.Limit <= 0 || req.Limit > 50 {
		req.Limit = 12
	}

	promptSets, err := s.ListPromptSets(ctx)
	if err != nil {
		return nil, err
	}

	lookup := make(map[string]domain.PromptSetSummary, len(promptSets))
	for _, item := range promptSets {
		lookup[item.ID] = item
	}

	leftSummary, ok := lookup[req.Left]
	if !ok {
		return nil, ErrPromptSetNotFound
	}
	rightSummary, ok := lookup[req.Right]
	if !ok {
		return nil, ErrPromptSetNotFound
	}

	leftMetrics, err := s.repo.GetPromptExperimentMetrics(ctx, req.Left, req)
	if err != nil {
		return nil, err
	}
	rightMetrics, err := s.repo.GetPromptExperimentMetrics(ctx, req.Right, req)
	if err != nil {
		return nil, err
	}
	leftMetrics.PromptSet = leftSummary
	rightMetrics.PromptSet = rightSummary

	samples, err := s.repo.ListPromptExperimentSamples(ctx, req)
	if err != nil {
		return nil, err
	}

	return &domain.PromptExperimentReport{
		Left:          *leftMetrics,
		Right:         *rightMetrics,
		RecentSamples: samples,
		AppliedFilters: domain.PromptExperimentFilters{
			Left:  req.Left,
			Right: req.Right,
			Mode:  req.Mode,
			Topic: req.Topic,
			Limit: req.Limit,
		},
	}, nil
}
