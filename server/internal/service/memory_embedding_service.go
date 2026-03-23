package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"log/slog"
	"math"
	"sort"
	"strings"
	"time"

	"practicehelper/server/internal/domain"
	"practicehelper/server/internal/vectorstore"
)

type memoryDocument struct {
	Entry       domain.MemoryIndexEntry
	Text        string
	ContentHash string
}

type memoryScoredEntry struct {
	entry       domain.MemoryIndexEntry
	ruleScore   float64
	vectorScore float64
	rerankScore float64
	combined    float64
}

func (s *Service) startMemoryEmbeddingWorker() {
	if s.memoryEmbeddingWorkerAlive || !s.vectorWriteAvailable() || s.repo == nil || s.sidecar == nil {
		return
	}
	s.memoryEmbeddingWorkerAlive = true

	go func() {
		s.processNextMemoryEmbeddingJob(context.Background())

		ticker := time.NewTicker(s.memoryEmbeddingPollEvery)
		defer ticker.Stop()
		for range ticker.C {
			s.processNextMemoryEmbeddingJob(context.Background())
		}
	}()
}

func (s *Service) processNextMemoryEmbeddingJob(ctx context.Context) {
	job, err := s.repo.ClaimNextMemoryEmbeddingJob(ctx, newMemoryEmbeddingClaimToken(), time.Now().UTC().Add(s.memoryEmbeddingClaimTTL))
	if err != nil {
		slog.Error("claim memory embedding job failed", "error", err)
		return
	}
	if job == nil {
		return
	}

	entries, err := s.repo.GetMemoryIndexEntriesByIDs(ctx, []string{job.MemoryIndexID})
	if err != nil || len(entries) == 0 {
		failErr := err
		if failErr == nil {
			failErr = modelShapeError("memory index entry missing for embedding job")
		}
		slog.Error("load memory index entry for embedding job failed", "job_id", job.ID, "error", failErr)
		_ = s.repo.FailMemoryEmbeddingJob(ctx, job.ID, job.ClaimToken, failErr.Error(), true)
		return
	}

	if err := s.indexMemoryIndexEntries(ctx, entries); err != nil {
		slog.Warn("process memory embedding job failed", "job_id", job.ID, "error", err)
		permanent := strings.Contains(err.Error(), "status 4")
		_ = s.repo.FailMemoryEmbeddingJob(ctx, job.ID, job.ClaimToken, err.Error(), permanent)
		return
	}

	if err := s.repo.CompleteMemoryEmbeddingJob(ctx, job.ID, job.ClaimToken); err != nil {
		slog.Error("complete memory embedding job failed", "job_id", job.ID, "error", err)
	}
}

func (s *Service) syncOrQueueMemoryEmbeddings(ctx context.Context, refs []domain.MemoryRef) {
	if !s.vectorWriteAvailable() || len(refs) == 0 {
		return
	}

	entries, err := s.repo.GetMemoryIndexEntriesByRefs(ctx, refs)
	if err != nil {
		slog.Warn("load memory index entries for embedding failed", "error", err)
		return
	}
	if len(entries) == 0 {
		return
	}

	indexCtx, cancel := context.WithTimeout(context.Background(), s.memoryHotIndexTimeout)
	defer cancel()

	if err := s.indexMemoryIndexEntries(indexCtx, entries); err != nil {
		slog.Warn("sync hot memory indexing failed; queueing background retry", "error", err)
		if enqueueErr := s.repo.EnqueueMemoryEmbeddingJobs(context.Background(), entries); enqueueErr != nil {
			slog.Error("enqueue memory embedding jobs failed", "error", enqueueErr)
		}
	}
}

func (s *Service) vectorWriteAvailable() bool {
	return s.vectorStore != nil && s.vectorStore.Enabled() && s.vectorWriteEnabled && s.sidecar != nil
}

func (s *Service) vectorReadAvailable() bool {
	return s.vectorStore != nil && s.vectorStore.Enabled() && s.vectorReadEnabled && s.sidecar != nil
}

func (s *Service) indexMemoryIndexEntries(
	ctx context.Context,
	entries []domain.MemoryIndexEntry,
) error {
	if !s.vectorWriteAvailable() || len(entries) == 0 {
		return nil
	}

	documents := make([]memoryDocument, 0, len(entries))
	for _, entry := range entries {
		text := buildMemoryDocumentText(entry)
		if text == "" {
			continue
		}
		documents = append(documents, memoryDocument{
			Entry:       entry,
			Text:        text,
			ContentHash: hashMemoryText(text),
		})
	}
	if len(documents) == 0 {
		return nil
	}

	existingRecords, err := s.repo.GetMemoryEmbeddingRecordsByMemoryIndexIDs(ctx, collectMemoryIndexIDs(documents))
	if err != nil {
		return err
	}
	recordByMemoryIndexID := make(map[string]domain.MemoryEmbeddingRecord, len(existingRecords))
	for _, record := range existingRecords {
		recordByMemoryIndexID[record.MemoryIndexID] = record
	}

	pendingDocs := make([]memoryDocument, 0, len(documents))
	for _, document := range documents {
		record, ok := recordByMemoryIndexID[document.Entry.ID]
		if ok && record.Status == domain.MemoryEmbeddingStatusIndexed && record.ContentHash == document.ContentHash {
			continue
		}
		pendingDocs = append(pendingDocs, document)
	}
	if len(pendingDocs) == 0 {
		return nil
	}

	embedResponse, err := s.sidecar.EmbedMemory(ctx, domain.EmbedMemoryRequest{
		Items: buildEmbedMemoryItems(pendingDocs),
	})
	if err != nil {
		return err
	}
	if embedResponse == nil || len(embedResponse.Items) != len(pendingDocs) {
		return ModelShapeError("unexpected embed_memory response size")
	}

	vectorPoints := make([]vectorstore.Point, 0, len(embedResponse.Items))
	records := make([]domain.MemoryEmbeddingRecord, 0, len(embedResponse.Items))
	vectorSize := 0
	for index, item := range embedResponse.Items {
		if len(item.Vector) == 0 {
			continue
		}
		if vectorSize == 0 {
			vectorSize = len(item.Vector)
		}
		document := pendingDocs[index]
		vectorPoints = append(vectorPoints, vectorstore.Point{
			ID:     document.Entry.ID,
			Vector: item.Vector,
			Payload: map[string]any{
				"document_kind": domain.VectorDocumentKindMemory,
				"memory_type":   document.Entry.MemoryType,
				"scope_type":    document.Entry.ScopeType,
				"scope_id":      document.Entry.ScopeID,
				"topic":         document.Entry.Topic,
				"project_id":    document.Entry.ProjectID,
				"session_id":    document.Entry.SessionID,
				"job_target_id": document.Entry.JobTargetID,
				"content_hash":  document.ContentHash,
			},
		})
		indexedAt := time.Now().UTC()
		records = append(records, domain.MemoryEmbeddingRecord{
			ID:            recordByMemoryIndexID[document.Entry.ID].ID,
			MemoryIndexID: document.Entry.ID,
			MemoryType:    document.Entry.MemoryType,
			RefTable:      document.Entry.RefTable,
			RefID:         document.Entry.RefID,
			ContentHash:   document.ContentHash,
			ModelName:     item.ModelName,
			VectorStoreID: document.Entry.ID,
			VectorDim:     len(item.Vector),
			Status:        domain.MemoryEmbeddingStatusIndexed,
			LastIndexedAt: &indexedAt,
		})
	}
	if len(vectorPoints) == 0 {
		return ModelShapeError("embed_memory returned no vectors")
	}

	if err := s.vectorStore.Upsert(ctx, vectorPoints, vectorSize); err != nil {
		failedRecords := make([]domain.MemoryEmbeddingRecord, 0, len(pendingDocs))
		for _, document := range pendingDocs {
			failedRecords = append(failedRecords, domain.MemoryEmbeddingRecord{
				ID:            recordByMemoryIndexID[document.Entry.ID].ID,
				MemoryIndexID: document.Entry.ID,
				MemoryType:    document.Entry.MemoryType,
				RefTable:      document.Entry.RefTable,
				RefID:         document.Entry.RefID,
				ContentHash:   document.ContentHash,
				Status:        domain.MemoryEmbeddingStatusFailed,
				LastError:     err.Error(),
			})
		}
		_ = s.repo.UpsertMemoryEmbeddingRecords(ctx, failedRecords)
		return err
	}

	return s.repo.UpsertMemoryEmbeddingRecords(ctx, records)
}

func (s *Service) rankMemoryIndexEntriesByVector(
	ctx context.Context,
	memoryType string,
	params agentContextParams,
	entries []domain.MemoryIndexEntry,
	limit int,
) ([]domain.MemoryIndexEntry, map[string]memoryScoredEntry, string, string) {
	if !s.vectorReadAvailable() || len(entries) < 2 {
		return entries, buildRuleOnlyScores(params, entries), "memory_index_rule", ""
	}

	queryText := buildMemoryRetrievalQuery(memoryType, params)
	if queryText == "" {
		return entries, buildRuleOnlyScores(params, entries), "memory_index_rule", ""
	}

	embedResponse, err := s.sidecar.EmbedMemory(ctx, domain.EmbedMemoryRequest{
		Items: []domain.EmbedMemoryItem{{ID: "query", Text: queryText}},
	})
	if err != nil || embedResponse == nil || len(embedResponse.Items) == 0 || len(embedResponse.Items[0].Vector) == 0 {
		return entries, buildRuleOnlyScores(params, entries), "memory_index_rule", queryText
	}

	points, err := s.vectorStore.Get(ctx, collectMemoryIndexEntryIDs(entries))
	if err != nil || len(points) == 0 {
		return entries, buildRuleOnlyScores(params, entries), "memory_index_rule", queryText
	}

	scored := make([]memoryScoredEntry, 0, len(entries))
	for _, entry := range entries {
		point, ok := points[entry.ID]
		if !ok || len(point.Vector) == 0 {
			scored = append(scored, memoryScoredEntry{
				entry:     entry,
				ruleScore: scoreMemoryIndexEntry(params, entry),
				combined:  scoreMemoryIndexEntry(params, entry),
			})
			continue
		}
		ruleScore := scoreMemoryIndexEntry(params, entry)
		vectorScore := cosineSimilarity(embedResponse.Items[0].Vector, point.Vector)
		scored = append(scored, memoryScoredEntry{
			entry:       entry,
			ruleScore:   ruleScore,
			vectorScore: vectorScore,
			combined:    ruleScore + vectorScore*2.2,
		})
	}

	if s.vectorRerankEnabled {
		topK := min(max(limit*3, 6), len(scored))
		candidates := make([]domain.RerankMemoryCandidate, 0, topK)
		sortScoredEntries(scored)
		for _, item := range scored[:topK] {
			candidates = append(candidates, domain.RerankMemoryCandidate{
				ID:   item.entry.ID,
				Text: buildMemoryDocumentText(item.entry),
			})
		}
		if rerankResponse, err := s.sidecar.RerankMemory(ctx, domain.RerankMemoryRequest{
			Query:      queryText,
			Candidates: candidates,
			TopK:       topK,
		}); err == nil && rerankResponse != nil {
			rerankScores := make(map[string]float64, len(rerankResponse.Items))
			for _, item := range rerankResponse.Items {
				rerankScores[item.ID] = item.Score
			}
			for index := range scored {
				if score, ok := rerankScores[scored[index].entry.ID]; ok {
					scored[index].rerankScore = score
					scored[index].combined += score * 3.0
				}
			}
		}
	}

	sortScoredEntries(scored)
	ranked := make([]domain.MemoryIndexEntry, 0, len(scored))
	scoreMap := make(map[string]memoryScoredEntry, len(scored))
	for _, item := range scored {
		ranked = append(ranked, item.entry)
		scoreMap[item.entry.ID] = item
	}
	strategy := "memory_index_vector"
	if s.vectorRerankEnabled {
		strategy = "memory_index_vector_rerank"
	}
	return ranked, scoreMap, strategy, queryText
}

func buildEmbedMemoryItems(documents []memoryDocument) []domain.EmbedMemoryItem {
	items := make([]domain.EmbedMemoryItem, 0, len(documents))
	for _, document := range documents {
		items = append(items, domain.EmbedMemoryItem{
			ID:   document.Entry.ID,
			Text: document.Text,
		})
	}
	return items
}

func collectMemoryIndexIDs(documents []memoryDocument) []string {
	ids := make([]string, 0, len(documents))
	for _, document := range documents {
		ids = append(ids, document.Entry.ID)
	}
	return ids
}

func collectMemoryIndexEntryIDs(entries []domain.MemoryIndexEntry) []string {
	ids := make([]string, 0, len(entries))
	for _, entry := range entries {
		ids = append(ids, entry.ID)
	}
	return ids
}

func collectObservationMemoryRefs(items []domain.AgentObservation) []domain.MemoryRef {
	refs := make([]domain.MemoryRef, 0, len(items))
	for _, item := range items {
		if item.ID == "" {
			continue
		}
		refs = append(refs, domain.MemoryRef{
			RefTable: "agent_observations",
			RefID:    item.ID,
		})
	}
	return refs
}

func buildMemoryDocumentText(entry domain.MemoryIndexEntry) string {
	parts := make([]string, 0, 6)
	if entry.MemoryType != "" {
		parts = append(parts, "memory_type: "+entry.MemoryType)
	}
	if entry.Topic != "" {
		parts = append(parts, "topic: "+entry.Topic)
	}
	if entry.Summary != "" {
		parts = append(parts, entry.Summary)
	}
	if len(entry.Tags) > 0 {
		parts = append(parts, "tags: "+strings.Join(entry.Tags, " | "))
	}
	if len(entry.Entities) > 0 {
		parts = append(parts, "entities: "+strings.Join(entry.Entities, " | "))
	}
	return strings.TrimSpace(strings.Join(parts, "\n"))
}

func buildMemoryRetrievalQuery(memoryType string, params agentContextParams) string {
	parts := make([]string, 0, 5)
	if memoryType != "" {
		parts = append(parts, "memory_type="+memoryType)
	}
	if topic := normalizeBasicsTopic(params.Topic); topic != "" {
		parts = append(parts, "topic="+topic)
	}
	if params.ProjectID != "" {
		parts = append(parts, "project_id="+params.ProjectID)
	}
	if params.JobTargetID != "" {
		parts = append(parts, "job_target_id="+params.JobTargetID)
	}
	if params.SessionID != "" {
		parts = append(parts, "exclude_session="+params.SessionID)
	}
	return strings.Join(parts, "\n")
}

func hashMemoryText(text string) string {
	sum := sha256.Sum256([]byte(text))
	return hex.EncodeToString(sum[:])
}

func cosineSimilarity(left []float64, right []float64) float64 {
	if len(left) == 0 || len(right) == 0 {
		return 0
	}
	size := min(len(left), len(right))
	var dot float64
	var leftNorm float64
	var rightNorm float64
	for i := 0; i < size; i++ {
		dot += left[i] * right[i]
		leftNorm += left[i] * left[i]
		rightNorm += right[i] * right[i]
	}
	if leftNorm == 0 || rightNorm == 0 {
		return 0
	}
	return dot / (math.Sqrt(leftNorm) * math.Sqrt(rightNorm))
}

func sortScoredEntries(items []memoryScoredEntry) {
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].combined != items[j].combined {
			return items[i].combined > items[j].combined
		}
		return items[i].entry.UpdatedAt.After(items[j].entry.UpdatedAt)
	})
}

func buildRuleOnlyScores(
	params agentContextParams,
	entries []domain.MemoryIndexEntry,
) map[string]memoryScoredEntry {
	items := make(map[string]memoryScoredEntry, len(entries))
	for _, entry := range entries {
		score := scoreMemoryIndexEntry(params, entry)
		items[entry.ID] = memoryScoredEntry{
			entry:     entry,
			ruleScore: score,
			combined:  score,
		}
	}
	return items
}

func newMemoryEmbeddingClaimToken() string {
	return newImportClaimToken()
}

type modelShapeError string

func (e modelShapeError) Error() string {
	return string(e)
}

func ModelShapeError(message string) error {
	return modelShapeError(message)
}
