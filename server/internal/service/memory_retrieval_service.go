package service

import (
	"context"
	"sort"
	"strconv"
	"strings"

	"practicehelper/server/internal/domain"
)

func (s *Service) loadObservationMemory(
	ctx context.Context,
	params agentContextParams,
) ([]domain.AgentObservation, error) {
	items, _, err := s.loadObservationMemoryWithTrace(ctx, params)
	return items, err
}

func (s *Service) loadObservationMemoryWithTrace(
	ctx context.Context,
	params agentContextParams,
) ([]domain.AgentObservation, *domain.MemoryRetrievalTrace, error) {
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
		return nil, nil, err
	}
	candidateCount := len(indexEntries)
	indexEntries = rerankMemoryIndexEntries(params, indexEntries)
	rankedEntries, scoreMap, strategy, queryText := s.rankMemoryIndexEntriesByVector(
		ctx,
		domain.MemoryTypeObservation,
		params,
		indexEntries,
		limit,
	)
	indexEntries = rankedEntries

	items, err := s.repo.GetObservationsByIDs(ctx, collectMemoryRefIDs(indexEntries, "agent_observations"))
	if err != nil {
		return nil, nil, err
	}
	items = items[:min(limit, len(items))]
	if len(items) >= limit {
		return items, buildObservationTrace(params, candidateCount, strategy, queryText, items, scoreMap, false, ""), nil
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
		return nil, nil, err
	}

	items = appendMissingObservations(items, fallback, limit)
	return items, buildObservationTrace(
		params,
		candidateCount,
		strategy,
		queryText,
		items,
		scoreMap,
		true,
		"memory_index 候选不足，使用 observation 直接查询补齐。",
	), nil
}

func (s *Service) loadSessionSummaryMemory(
	ctx context.Context,
	params agentContextParams,
) ([]domain.SessionMemorySummary, error) {
	items, _, err := s.loadSessionSummaryMemoryWithTrace(ctx, params)
	return items, err
}

func (s *Service) loadSessionSummaryMemoryWithTrace(
	ctx context.Context,
	params agentContextParams,
) ([]domain.SessionMemorySummary, *domain.MemoryRetrievalTrace, error) {
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
		return nil, nil, err
	}
	candidateCount := len(indexEntries)
	indexEntries = rerankMemoryIndexEntries(params, indexEntries)
	rankedEntries, scoreMap, strategy, queryText := s.rankMemoryIndexEntriesByVector(
		ctx,
		domain.MemoryTypeSessionSummary,
		params,
		indexEntries,
		limit,
	)
	indexEntries = rankedEntries

	items, err := s.repo.GetSessionMemorySummariesByIDs(
		ctx,
		collectMemoryRefIDs(indexEntries, "session_memory_summaries"),
	)
	if err != nil {
		return nil, nil, err
	}
	items = items[:min(limit, len(items))]
	if len(items) >= limit {
		return items, buildSessionSummaryTrace(params, candidateCount, strategy, queryText, items, scoreMap, false, ""), nil
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
		return nil, nil, err
	}

	items = appendMissingSessionSummaries(items, fallback, limit)
	return items, buildSessionSummaryTrace(
		params,
		candidateCount,
		strategy,
		queryText,
		items,
		scoreMap,
		true,
		"memory_index 候选不足，使用 session summary 直接查询补齐。",
	), nil
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

func buildObservationTrace(
	params agentContextParams,
	candidateCount int,
	strategy string,
	queryText string,
	items []domain.AgentObservation,
	scoreMap map[string]memoryScoredEntry,
	fallbackUsed bool,
	fallbackReason string,
) *domain.MemoryRetrievalTrace {
	hits := make([]domain.MemoryRetrievalHit, 0, len(items))
	for _, item := range items {
		hits = append(hits, buildMemoryRetrievalHit(
			scoreMap,
			"agent_observations",
			item.ID,
			item.ScopeType,
			item.ScopeID,
			item.Topic,
			item.Content,
			item.Tags,
		))
	}
	return &domain.MemoryRetrievalTrace{
		MemoryType:     domain.MemoryTypeObservation,
		Query:          resolveTraceQuery(queryText, domain.MemoryTypeObservation, params),
		Strategy:       strategy,
		CandidateCount: candidateCount,
		SelectedCount:  len(items),
		FallbackUsed:   fallbackUsed,
		FallbackReason: fallbackReason,
		Hits:           hits,
	}
}

func buildSessionSummaryTrace(
	params agentContextParams,
	candidateCount int,
	strategy string,
	queryText string,
	items []domain.SessionMemorySummary,
	scoreMap map[string]memoryScoredEntry,
	fallbackUsed bool,
	fallbackReason string,
) *domain.MemoryRetrievalTrace {
	hits := make([]domain.MemoryRetrievalHit, 0, len(items))
	for _, item := range items {
		hits = append(hits, buildMemoryRetrievalHit(
			scoreMap,
			"session_memory_summaries",
			item.ID,
			domain.MemoryScopeSession,
			item.SessionID,
			item.Topic,
			item.Summary,
			append(append([]string{}, item.Strengths...), item.RecommendedFocus...),
		))
	}
	return &domain.MemoryRetrievalTrace{
		MemoryType:     domain.MemoryTypeSessionSummary,
		Query:          resolveTraceQuery(queryText, domain.MemoryTypeSessionSummary, params),
		Strategy:       strategy,
		CandidateCount: candidateCount,
		SelectedCount:  len(items),
		FallbackUsed:   fallbackUsed,
		FallbackReason: fallbackReason,
		Hits:           hits,
	}
}

func buildMemoryRetrievalHit(
	scoreMap map[string]memoryScoredEntry,
	refTable string,
	refID string,
	scopeType string,
	scopeID string,
	topic string,
	summary string,
	tags []string,
) domain.MemoryRetrievalHit {
	for _, item := range scoreMap {
		if item.entry.RefTable == refTable && item.entry.RefID == refID {
			return domain.MemoryRetrievalHit{
				Source:        "memory_index",
				MemoryIndexID: item.entry.ID,
				RefTable:      item.entry.RefTable,
				RefID:         item.entry.RefID,
				ScopeType:     item.entry.ScopeType,
				ScopeID:       item.entry.ScopeID,
				Topic:         item.entry.Topic,
				Summary:       item.entry.Summary,
				RuleScore:     item.ruleScore,
				VectorScore:   item.vectorScore,
				RerankScore:   item.rerankScore,
				FinalScore:    item.combined,
				Reason:        buildMemoryHitReason(item),
			}
		}
	}

	return domain.MemoryRetrievalHit{
		Source:    "direct_fallback",
		RefTable:  refTable,
		RefID:     refID,
		ScopeType: scopeType,
		ScopeID:   scopeID,
		Topic:     topic,
		Summary:   summary,
		Reason:    buildFallbackHitReason(scopeType, topic, tags),
	}
}

func buildMemoryHitReason(item memoryScoredEntry) string {
	reasons := make([]string, 0, 5)
	if item.entry.ScopeType == domain.MemoryScopeProject {
		reasons = append(reasons, "项目 scope 命中")
	} else if item.entry.ScopeType == domain.MemoryScopeJobTarget {
		reasons = append(reasons, "岗位 scope 命中")
	} else if item.entry.ScopeType == domain.MemoryScopeSession {
		reasons = append(reasons, "历史 session 命中")
	} else {
		reasons = append(reasons, "全局记忆兜底")
	}
	if item.entry.Topic != "" {
		reasons = append(reasons, "topic="+item.entry.Topic)
	}
	if item.vectorScore > 0 {
		reasons = append(reasons, "semantic 相似度高")
	}
	if item.rerankScore > 0 {
		reasons = append(reasons, "rerank 提升排序")
	}
	reasons = append(reasons, "final="+formatTraceScore(item.combined))
	return strings.Join(reasons, "；")
}

func buildFallbackHitReason(scopeType string, topic string, tags []string) string {
	reasons := []string{"直接查询补齐"}
	if scopeType != "" {
		reasons = append(reasons, "scope="+scopeType)
	}
	if topic != "" {
		reasons = append(reasons, "topic="+topic)
	}
	if len(tags) > 0 {
		reasons = append(reasons, "tags="+strings.Join(tags, ","))
	}
	return strings.Join(reasons, "；")
}

func resolveTraceQuery(queryText string, memoryType string, params agentContextParams) string {
	if strings.TrimSpace(queryText) != "" {
		return queryText
	}
	return buildMemoryRetrievalQuery(memoryType, params)
}

func formatTraceScore(value float64) string {
	return strings.TrimRight(strings.TrimRight(strconv.FormatFloat(value, 'f', 3, 64), "0"), ".")
}
