package service

import (
	"context"
	"sort"
	"strings"

	"practicehelper/server/internal/domain"
)

func (s *Service) loadObservationMemory(
	ctx context.Context,
	params agentContextParams,
) ([]domain.AgentObservation, error) {
	limit := defaultLimit(params.ObservationLimit, 4)
	candidateLimit := max(limit*4, limit+4)
	if s.vectorReadAvailable() {
		candidateLimit = max(limit*8, limit+8)
	}
	indexEntries, err := s.repo.SearchMemoryIndexEntries(
		ctx,
		domain.MemoryTypeObservation,
		params.Topic,
		params.ProjectID,
		params.JobTargetID,
		params.SessionID,
		candidateLimit,
	)
	if err != nil {
		return nil, err
	}
	indexEntries = rerankMemoryIndexEntries(params, indexEntries)
	indexEntries = s.rerankMemoryIndexEntriesByVector(
		ctx,
		domain.MemoryTypeObservation,
		params,
		indexEntries,
		limit,
	)

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
	candidateLimit := max(limit*4, limit+4)
	if s.vectorReadAvailable() {
		candidateLimit = max(limit*8, limit+8)
	}
	indexEntries, err := s.repo.SearchMemoryIndexEntries(
		ctx,
		domain.MemoryTypeSessionSummary,
		params.Topic,
		params.ProjectID,
		params.JobTargetID,
		params.SessionID,
		candidateLimit,
	)
	if err != nil {
		return nil, err
	}
	indexEntries = rerankMemoryIndexEntries(params, indexEntries)
	indexEntries = s.rerankMemoryIndexEntriesByVector(
		ctx,
		domain.MemoryTypeSessionSummary,
		params,
		indexEntries,
		limit,
	)

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

func rerankMemoryIndexEntries(
	params agentContextParams,
	entries []domain.MemoryIndexEntry,
) []domain.MemoryIndexEntry {
	if len(entries) < 2 {
		return entries
	}

	ranked := append([]domain.MemoryIndexEntry(nil), entries...)
	sort.SliceStable(ranked, func(i, j int) bool {
		left := scoreMemoryIndexEntry(params, ranked[i])
		right := scoreMemoryIndexEntry(params, ranked[j])
		if left != right {
			return left > right
		}
		return ranked[i].UpdatedAt.After(ranked[j].UpdatedAt)
	})
	return ranked
}

func scoreMemoryIndexEntry(params agentContextParams, entry domain.MemoryIndexEntry) float64 {
	score := 0.0
	projectID := strings.TrimSpace(params.ProjectID)
	jobTargetID := strings.TrimSpace(params.JobTargetID)
	sessionID := strings.TrimSpace(params.SessionID)
	topic := normalizeBasicsTopic(params.Topic)
	if topic == "" {
		topic = strings.TrimSpace(strings.ToLower(params.Topic))
	}

	if sessionID != "" && entry.SessionID == sessionID {
		score += 4.0
	}
	if projectID != "" && (entry.ProjectID == projectID || (entry.ScopeType == domain.MemoryScopeProject && entry.ScopeID == projectID)) {
		score += 3.0
	}
	if jobTargetID != "" && (entry.JobTargetID == jobTargetID || (entry.ScopeType == domain.MemoryScopeJobTarget && entry.ScopeID == jobTargetID)) {
		score += 2.5
	}
	if topic != "" && entry.Topic == topic {
		score += 2.0
	}

	score += entry.Salience * 0.6
	score += entry.Confidence * 0.4
	score += entry.Freshness * 0.2

	focusText := strings.ToLower(entry.Summary + " " + strings.Join(entry.Tags, " ") + " " + strings.Join(entry.Entities, " "))
	if topic != "" && strings.Contains(focusText, topic) {
		score += 0.35
	}
	for _, matched := range matchBasicsTopics(focusText) {
		if matched == topic {
			score += 0.2
			break
		}
	}

	switch entry.ScopeType {
	case domain.MemoryScopeProject:
		score += 0.25
	case domain.MemoryScopeJobTarget:
		score += 0.2
	case domain.MemoryScopeGlobal:
		score += 0.05
	}

	return score
}
