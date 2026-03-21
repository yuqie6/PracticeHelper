package service

import (
	"context"

	"practicehelper/server/internal/domain"
)

func (s *Service) loadObservationMemory(
	ctx context.Context,
	params agentContextParams,
) ([]domain.AgentObservation, error) {
	limit := defaultLimit(params.ObservationLimit, 4)
	indexEntries, err := s.repo.SearchMemoryIndexEntries(
		ctx,
		"observation",
		params.Topic,
		params.ProjectID,
		params.JobTargetID,
		params.SessionID,
		limit,
	)
	if err != nil {
		return nil, err
	}

	items, err := s.repo.GetObservationsByIDs(ctx, collectMemoryRefIDs(indexEntries, "agent_observations"))
	if err != nil {
		return nil, err
	}
	items = items[:min(limit, len(items))]
	if len(items) >= limit {
		return items, nil
	}

	fallback, err := s.repo.ListRelevantObservations(
		ctx,
		params.SessionID,
		params.ProjectID,
		params.JobTargetID,
		params.Topic,
		limit,
	)
	if err != nil {
		return nil, err
	}

	return appendMissingObservations(items, fallback, limit), nil
}

func (s *Service) loadSessionSummaryMemory(
	ctx context.Context,
	params agentContextParams,
) ([]domain.SessionMemorySummary, error) {
	limit := defaultLimit(params.SessionSummaryLimit, 3)
	indexEntries, err := s.repo.SearchMemoryIndexEntries(
		ctx,
		"session_summary",
		params.Topic,
		params.ProjectID,
		params.JobTargetID,
		params.SessionID,
		limit,
	)
	if err != nil {
		return nil, err
	}

	items, err := s.repo.GetSessionMemorySummariesByIDs(
		ctx,
		collectMemoryRefIDs(indexEntries, "session_memory_summaries"),
	)
	if err != nil {
		return nil, err
	}
	items = items[:min(limit, len(items))]
	if len(items) >= limit {
		return items, nil
	}

	fallback, err := s.repo.ListRelevantSessionMemorySummaries(
		ctx,
		params.Topic,
		params.ProjectID,
		params.JobTargetID,
		params.SessionID,
		limit,
	)
	if err != nil {
		return nil, err
	}

	return appendMissingSessionSummaries(items, fallback, limit), nil
}

func collectMemoryRefIDs(entries []domain.MemoryIndexEntry, refTable string) []string {
	if len(entries) == 0 {
		return nil
	}

	ids := make([]string, 0, len(entries))
	seen := make(map[string]struct{}, len(entries))
	for _, entry := range entries {
		if entry.RefTable != refTable || entry.RefID == "" {
			continue
		}
		if _, ok := seen[entry.RefID]; ok {
			continue
		}
		seen[entry.RefID] = struct{}{}
		ids = append(ids, entry.RefID)
	}
	return ids
}

func appendMissingObservations(
	primary []domain.AgentObservation,
	fallback []domain.AgentObservation,
	limit int,
) []domain.AgentObservation {
	if len(primary) >= limit {
		return primary[:limit]
	}

	items := append([]domain.AgentObservation(nil), primary...)
	seen := make(map[string]struct{}, len(items))
	for _, item := range items {
		seen[item.ID] = struct{}{}
	}
	for _, item := range fallback {
		if len(items) >= limit {
			break
		}
		if _, ok := seen[item.ID]; ok {
			continue
		}
		seen[item.ID] = struct{}{}
		items = append(items, item)
	}
	return items
}

func appendMissingSessionSummaries(
	primary []domain.SessionMemorySummary,
	fallback []domain.SessionMemorySummary,
	limit int,
) []domain.SessionMemorySummary {
	if len(primary) >= limit {
		return primary[:limit]
	}

	items := append([]domain.SessionMemorySummary(nil), primary...)
	seen := make(map[string]struct{}, len(items))
	for _, item := range items {
		seen[item.ID] = struct{}{}
	}
	for _, item := range fallback {
		if len(items) >= limit {
			break
		}
		if _, ok := seen[item.ID]; ok {
			continue
		}
		seen[item.ID] = struct{}{}
		items = append(items, item)
	}
	return items
}
