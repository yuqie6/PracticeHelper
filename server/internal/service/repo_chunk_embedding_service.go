package service

import (
	"context"
	"log/slog"
	"sort"
	"strings"
	"time"

	"practicehelper/server/internal/domain"
	"practicehelper/server/internal/vectorstore"
)

type repoChunkDocument struct {
	Chunk       domain.RepoChunk
	Text        string
	ContentHash string
}

type repoChunkSearchHit struct {
	chunk       domain.RepoChunk
	vectorScore float64
	rerankScore float64
}

func (s *Service) startRepoChunkEmbeddingWorker() {
	if s.repoChunkEmbeddingWorkerAlive || !s.vectorWriteAvailable() || s.repo == nil || s.sidecar == nil {
		return
	}
	s.repoChunkEmbeddingWorkerAlive = true

	go func() {
		s.enqueueMissingRepoChunkEmbeddings(context.Background())
		s.processNextRepoChunkEmbeddingJob(context.Background())

		ticker := time.NewTicker(s.memoryEmbeddingPollEvery)
		defer ticker.Stop()
		for range ticker.C {
			s.processNextRepoChunkEmbeddingJob(context.Background())
		}
	}()
}

func (s *Service) enqueueMissingRepoChunkEmbeddings(ctx context.Context) {
	chunks, err := s.repo.ListAllRepoChunks(ctx)
	if err != nil {
		slog.Warn("list repo chunks for embedding backfill failed", "error", err)
		return
	}
	if err := s.enqueueRepoChunkEmbeddingJobs(ctx, chunks); err != nil {
		slog.Warn("enqueue repo chunk embedding backfill failed", "error", err)
	}
}

func (s *Service) enqueueProjectRepoChunkEmbeddings(ctx context.Context, projectID string) {
	if !s.vectorWriteAvailable() || strings.TrimSpace(projectID) == "" {
		return
	}

	chunks, err := s.repo.ListRepoChunksByProject(ctx, projectID)
	if err != nil {
		slog.Warn("list project repo chunks for embedding failed", "project_id", projectID, "error", err)
		return
	}
	if err := s.enqueueRepoChunkEmbeddingJobs(ctx, chunks); err != nil {
		slog.Warn("enqueue project repo chunk embeddings failed", "project_id", projectID, "error", err)
	}
}

func (s *Service) enqueueRepoChunkEmbeddingJobs(
	ctx context.Context,
	chunks []domain.RepoChunk,
) error {
	if !s.vectorWriteAvailable() || len(chunks) == 0 {
		return nil
	}

	documents := buildRepoChunkDocuments(chunks)
	if len(documents) == 0 {
		return nil
	}

	records, err := s.repo.GetRepoChunkEmbeddingRecordsByChunkIDs(ctx, collectRepoChunkIDs(documents))
	if err != nil {
		return err
	}
	recordByChunkID := make(map[string]domain.RepoChunkEmbeddingRecord, len(records))
	for _, record := range records {
		recordByChunkID[record.RepoChunkID] = record
	}

	pendingChunks := make([]domain.RepoChunk, 0, len(documents))
	for _, document := range documents {
		record, ok := recordByChunkID[document.Chunk.ID]
		if ok && record.Status == domain.RepoChunkEmbeddingStatusIndexed && record.ContentHash == document.ContentHash {
			continue
		}
		pendingChunks = append(pendingChunks, document.Chunk)
	}
	if len(pendingChunks) == 0 {
		return nil
	}

	return s.repo.EnqueueRepoChunkEmbeddingJobs(ctx, pendingChunks)
}

func (s *Service) processNextRepoChunkEmbeddingJob(ctx context.Context) {
	job, err := s.repo.ClaimNextRepoChunkEmbeddingJob(
		ctx,
		newMemoryEmbeddingClaimToken(),
		time.Now().UTC().Add(s.memoryEmbeddingClaimTTL),
	)
	if err != nil {
		slog.Error("claim repo chunk embedding job failed", "error", err)
		return
	}
	if job == nil {
		return
	}

	chunks, err := s.repo.GetRepoChunksByIDs(ctx, []string{job.RepoChunkID})
	if err != nil || len(chunks) == 0 {
		failErr := err
		if failErr == nil {
			failErr = modelShapeError("repo chunk missing for embedding job")
		}
		slog.Error("load repo chunk for embedding job failed", "job_id", job.ID, "error", failErr)
		_ = s.repo.FailRepoChunkEmbeddingJob(ctx, job.ID, job.ClaimToken, failErr.Error(), true)
		return
	}

	if err := s.indexRepoChunks(ctx, chunks); err != nil {
		slog.Warn("process repo chunk embedding job failed", "job_id", job.ID, "error", err)
		permanent := strings.Contains(err.Error(), "status 4")
		_ = s.repo.FailRepoChunkEmbeddingJob(ctx, job.ID, job.ClaimToken, err.Error(), permanent)
		return
	}

	if err := s.repo.CompleteRepoChunkEmbeddingJob(ctx, job.ID, job.ClaimToken); err != nil {
		slog.Error("complete repo chunk embedding job failed", "job_id", job.ID, "error", err)
	}
}

func (s *Service) indexRepoChunks(ctx context.Context, chunks []domain.RepoChunk) error {
	if !s.vectorWriteAvailable() || len(chunks) == 0 {
		return nil
	}

	documents := buildRepoChunkDocuments(chunks)
	if len(documents) == 0 {
		return nil
	}

	records, err := s.repo.GetRepoChunkEmbeddingRecordsByChunkIDs(ctx, collectRepoChunkIDs(documents))
	if err != nil {
		return err
	}
	recordByChunkID := make(map[string]domain.RepoChunkEmbeddingRecord, len(records))
	for _, record := range records {
		recordByChunkID[record.RepoChunkID] = record
	}

	pendingDocs := make([]repoChunkDocument, 0, len(documents))
	for _, document := range documents {
		record, ok := recordByChunkID[document.Chunk.ID]
		if ok && record.Status == domain.RepoChunkEmbeddingStatusIndexed && record.ContentHash == document.ContentHash {
			continue
		}
		pendingDocs = append(pendingDocs, document)
	}
	if len(pendingDocs) == 0 {
		return nil
	}

	embedResponse, err := s.sidecar.EmbedMemory(ctx, domain.EmbedMemoryRequest{
		Items: buildEmbedRepoChunkItems(pendingDocs),
	})
	if err != nil {
		return err
	}
	if embedResponse == nil || len(embedResponse.Items) != len(pendingDocs) {
		return ModelShapeError("unexpected repo chunk embed response size")
	}

	vectorPoints := make([]vectorstore.Point, 0, len(embedResponse.Items))
	updatedRecords := make([]domain.RepoChunkEmbeddingRecord, 0, len(embedResponse.Items))
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
			ID:     document.Chunk.ID,
			Vector: item.Vector,
			Payload: map[string]any{
				"document_kind": domain.VectorDocumentKindRepoChunk,
				"project_id":    document.Chunk.ProjectID,
				"file_path":     document.Chunk.FilePath,
				"file_type":     document.Chunk.FileType,
				"importance":    document.Chunk.Importance,
				"content_hash":  document.ContentHash,
			},
		})

		indexedAt := time.Now().UTC()
		updatedRecords = append(updatedRecords, domain.RepoChunkEmbeddingRecord{
			ID:            recordByChunkID[document.Chunk.ID].ID,
			RepoChunkID:   document.Chunk.ID,
			ProjectID:     document.Chunk.ProjectID,
			ContentHash:   document.ContentHash,
			ModelName:     item.ModelName,
			VectorStoreID: document.Chunk.ID,
			VectorDim:     len(item.Vector),
			Status:        domain.RepoChunkEmbeddingStatusIndexed,
			LastIndexedAt: &indexedAt,
		})
	}
	if len(vectorPoints) == 0 {
		return ModelShapeError("embed repo chunk returned no vectors")
	}

	if err := s.vectorStore.Upsert(ctx, vectorPoints, vectorSize); err != nil {
		failedRecords := make([]domain.RepoChunkEmbeddingRecord, 0, len(pendingDocs))
		for _, document := range pendingDocs {
			failedRecords = append(failedRecords, domain.RepoChunkEmbeddingRecord{
				ID:          recordByChunkID[document.Chunk.ID].ID,
				RepoChunkID: document.Chunk.ID,
				ProjectID:   document.Chunk.ProjectID,
				ContentHash: document.ContentHash,
				Status:      domain.RepoChunkEmbeddingStatusFailed,
				LastError:   err.Error(),
			})
		}
		_ = s.repo.UpsertRepoChunkEmbeddingRecords(context.Background(), failedRecords)
		return err
	}

	return s.repo.UpsertRepoChunkEmbeddingRecords(ctx, updatedRecords)
}

func (s *Service) SearchProjectChunks(
	ctx context.Context,
	projectID string,
	query string,
	limit int,
) ([]domain.RepoChunk, error) {
	if limit <= 0 {
		limit = 5
	}

	chunks, ok := s.searchProjectChunksByVector(ctx, projectID, query, limit)
	if ok {
		return chunks, nil
	}
	return s.repo.SearchProjectChunks(ctx, projectID, query, limit)
}

func (s *Service) searchProjectChunksByVector(
	ctx context.Context,
	projectID string,
	query string,
	limit int,
) ([]domain.RepoChunk, bool) {
	if !s.vectorReadAvailable() || strings.TrimSpace(projectID) == "" || strings.TrimSpace(query) == "" {
		return nil, false
	}

	embedResponse, err := s.sidecar.EmbedMemory(ctx, domain.EmbedMemoryRequest{
		Items: []domain.EmbedMemoryItem{{
			ID:   "repo_query",
			Text: strings.TrimSpace(query),
		}},
	})
	if err != nil || embedResponse == nil || len(embedResponse.Items) == 0 || len(embedResponse.Items[0].Vector) == 0 {
		return nil, false
	}

	candidateLimit := max(limit*4, limit+4)
	results, err := s.vectorStore.Search(ctx, embedResponse.Items[0].Vector, vectorstore.SearchFilter{
		Equals: map[string]string{
			"document_kind": domain.VectorDocumentKindRepoChunk,
			"project_id":    projectID,
		},
	}, candidateLimit)
	if err != nil || len(results) < limit {
		return nil, false
	}

	ids := collectRepoChunkSearchResultIDs(results)
	chunks, err := s.repo.GetRepoChunksByIDs(ctx, ids)
	if err != nil || len(chunks) < limit {
		return nil, false
	}

	byID := make(map[string]domain.RepoChunk, len(chunks))
	for _, chunk := range chunks {
		byID[chunk.ID] = chunk
	}

	scored := make([]repoChunkSearchHit, 0, len(results))
	for _, item := range results {
		chunk, ok := byID[item.ID]
		if !ok {
			return nil, false
		}
		scored = append(scored, repoChunkSearchHit{
			chunk:       chunk,
			vectorScore: item.Score,
		})
	}
	if len(scored) < limit {
		return nil, false
	}

	if s.vectorRerankEnabled {
		topK := min(max(limit*3, 6), len(scored))
		candidates := make([]domain.RerankMemoryCandidate, 0, topK)
		for _, item := range scored[:topK] {
			candidates = append(candidates, domain.RerankMemoryCandidate{
				ID:   item.chunk.ID,
				Text: buildRepoChunkDocumentText(item.chunk),
			})
		}
		if rerankResponse, err := s.sidecar.RerankMemory(ctx, domain.RerankMemoryRequest{
			Query:      strings.TrimSpace(query),
			Candidates: candidates,
			TopK:       topK,
		}); err == nil && rerankResponse != nil {
			rerankScores := make(map[string]float64, len(rerankResponse.Items))
			for _, item := range rerankResponse.Items {
				rerankScores[item.ID] = item.Score
			}
			for index := range scored {
				scored[index].rerankScore = rerankScores[scored[index].chunk.ID]
			}
		}
	}

	sortRepoChunkSearchHits(scored)
	selected := make([]domain.RepoChunk, 0, limit)
	for _, item := range scored {
		selected = append(selected, item.chunk)
		if len(selected) >= limit {
			break
		}
	}
	if len(selected) < limit {
		return nil, false
	}
	return selected, true
}

func buildRepoChunkDocuments(chunks []domain.RepoChunk) []repoChunkDocument {
	documents := make([]repoChunkDocument, 0, len(chunks))
	for _, chunk := range chunks {
		if chunk.ID == "" {
			continue
		}
		text := buildRepoChunkDocumentText(chunk)
		if text == "" {
			continue
		}
		documents = append(documents, repoChunkDocument{
			Chunk:       chunk,
			Text:        text,
			ContentHash: hashMemoryText(text),
		})
	}
	return documents
}

func buildRepoChunkDocumentText(chunk domain.RepoChunk) string {
	parts := make([]string, 0, 3)
	if chunk.FilePath != "" {
		parts = append(parts, "file_path: "+chunk.FilePath)
	}
	if chunk.FileType != "" {
		parts = append(parts, "file_type: "+chunk.FileType)
	}
	if chunk.Content != "" {
		parts = append(parts, chunk.Content)
	}
	return strings.TrimSpace(strings.Join(parts, "\n"))
}

func buildEmbedRepoChunkItems(documents []repoChunkDocument) []domain.EmbedMemoryItem {
	items := make([]domain.EmbedMemoryItem, 0, len(documents))
	for _, document := range documents {
		items = append(items, domain.EmbedMemoryItem{
			ID:   document.Chunk.ID,
			Text: document.Text,
		})
	}
	return items
}

func collectRepoChunkIDs(documents []repoChunkDocument) []string {
	ids := make([]string, 0, len(documents))
	for _, document := range documents {
		ids = append(ids, document.Chunk.ID)
	}
	return ids
}

func collectRepoChunkSearchResultIDs(items []vectorstore.SearchResult) []string {
	ids := make([]string, 0, len(items))
	seen := make(map[string]struct{}, len(items))
	for _, item := range items {
		if item.ID == "" {
			continue
		}
		if _, ok := seen[item.ID]; ok {
			continue
		}
		seen[item.ID] = struct{}{}
		ids = append(ids, item.ID)
	}
	return ids
}

func sortRepoChunkSearchHits(items []repoChunkSearchHit) {
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].rerankScore != items[j].rerankScore {
			return items[i].rerankScore > items[j].rerankScore
		}
		if items[i].vectorScore != items[j].vectorScore {
			return items[i].vectorScore > items[j].vectorScore
		}
		if items[i].chunk.Importance != items[j].chunk.Importance {
			return items[i].chunk.Importance > items[j].chunk.Importance
		}
		return items[i].chunk.FilePath < items[j].chunk.FilePath
	})
}
