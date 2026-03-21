package service

import (
	"context"
	"strings"

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
	if err := s.repo.EnsureKnowledgeSeeds(ctx); err != nil {
		return nil, err
	}

	profile, err := s.repo.GetUserProfile(ctx)
	if err != nil {
		return nil, err
	}
	weaknesses, err := s.repo.ListWeaknesses(ctx, defaultLimit(params.WeaknessLimit, 5))
	if err != nil {
		return nil, err
	}
	subgraph, err := s.repo.GetKnowledgeSubgraph(
		ctx,
		params.Topic,
		params.ProjectID,
		defaultLimit(params.KnowledgeNodeLimit, 8),
	)
	if err != nil {
		return nil, err
	}
	observations, err := s.loadObservationMemory(ctx, params)
	if err != nil {
		return nil, err
	}
	summaries, err := s.loadSessionSummaryMemory(ctx, params)
	if err != nil {
		return nil, err
	}

	return &domain.AgentContext{
		Profile:           buildProfileSnapshot(profile),
		KnowledgeSubgraph: subgraph,
		Observations:      observations,
		WeaknessProfile:   weaknesses,
		SessionSummaries:  summaries,
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
