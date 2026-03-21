package service

import (
	"context"
	"strings"
	"time"

	"practicehelper/server/internal/domain"
)

type agentContextParams struct {
	Topic               string
	ProjectID           string
	JobTargetID         string
	SessionID           string
	WeaknessLimit       int
	ObservationLimit    int
	SessionSummaryLimit int
	KnowledgeNodeLimit  int
}

func (s *Service) getAgentContext(
	ctx context.Context,
	params agentContextParams,
) (*domain.AgentContext, error) {
	agentContext, _, err := s.getAgentContextDetailed(ctx, params)
	return agentContext, err
}

func (s *Service) getAgentContextDetailed(
	ctx context.Context,
	params agentContextParams,
) (*domain.AgentContext, *domain.RetrievalTrace, error) {
	if err := s.repo.EnsureKnowledgeSeeds(ctx); err != nil {
		return nil, nil, err
	}

	profile, err := s.repo.GetUserProfile(ctx)
	if err != nil {
		return nil, nil, err
	}
	weaknesses, err := s.repo.ListWeaknesses(ctx, defaultLimit(params.WeaknessLimit, 5))
	if err != nil {
		return nil, nil, err
	}
	subgraph, err := s.repo.GetKnowledgeSubgraph(
		ctx,
		params.Topic,
		params.ProjectID,
		defaultLimit(params.KnowledgeNodeLimit, 8),
	)
	if err != nil {
		return nil, nil, err
	}
	observations, observationTrace, err := s.loadObservationMemoryWithTrace(ctx, params)
	if err != nil {
		return nil, nil, err
	}
	summaries, summaryTrace, err := s.loadSessionSummaryMemoryWithTrace(ctx, params)
	if err != nil {
		return nil, nil, err
	}

	return &domain.AgentContext{
			Profile:           buildProfileSnapshot(profile),
			KnowledgeSubgraph: subgraph,
			Observations:      observations,
			WeaknessProfile:   weaknesses,
			SessionSummaries:  summaries,
		}, &domain.RetrievalTrace{
			GeneratedAt:      time.Now().UTC(),
			Topic:            strings.TrimSpace(params.Topic),
			ProjectID:        strings.TrimSpace(params.ProjectID),
			JobTargetID:      strings.TrimSpace(params.JobTargetID),
			ObservationTrace: observationTrace,
			SummaryTrace:     summaryTrace,
		}, nil
}

func buildProfileSnapshot(profile *domain.UserProfile) *domain.ProfileSnapshot {
	if profile == nil {
		return nil
	}

	return &domain.ProfileSnapshot{
		TargetRole:           strings.TrimSpace(profile.TargetRole),
		TargetCompanyType:    strings.TrimSpace(profile.TargetCompanyType),
		CurrentStage:         strings.TrimSpace(profile.CurrentStage),
		ApplicationDeadline:  profile.ApplicationDeadline,
		TechStacks:           append([]string(nil), profile.TechStacks...),
		PrimaryProjects:      append([]string(nil), profile.PrimaryProjects...),
		SelfReportedWeakness: append([]string(nil), profile.SelfReportedWeakness...),
		ActiveJobTarget:      profile.ActiveJobTarget,
	}
}

func defaultLimit(value int, fallback int) int {
	if value > 0 {
		return value
	}
	return fallback
}
