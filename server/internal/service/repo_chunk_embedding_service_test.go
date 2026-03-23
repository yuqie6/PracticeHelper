package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"practicehelper/server/internal/domain"
	"practicehelper/server/internal/sidecar"
	"practicehelper/server/internal/vectorstore"
)

func TestSearchProjectChunksUsesVectorSearchAndRerank(t *testing.T) {
	store, err := openTestStore(t)
	if err != nil {
		t.Fatalf("openTestStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	project, chunks := createTestProjectWithChunks(t, store, []domain.RepoChunk{
		{
			FilePath:   "internal/cache.go",
			FileType:   "go",
			Content:    "redis consistency tradeoff details and cache invalidation flow",
			Importance: 0.82,
		},
		{
			FilePath:   "internal/worker.go",
			FileType:   "go",
			Content:    "background worker handles queue retries and graceful fallback",
			Importance: 0.61,
		},
		{
			FilePath:   "cmd/api/main.go",
			FileType:   "go",
			Content:    "http server bootstrap and wiring",
			Importance: 0.44,
		},
	})

	vectorStore := newFakeVectorStore()
	for _, chunk := range chunks {
		vector := []float64{0.1, 0.1}
		switch chunk.FilePath {
		case "internal/cache.go":
			vector = []float64{0.99, 0.01}
		case "internal/worker.go":
			vector = []float64{0.76, 0.24}
		case "cmd/api/main.go":
			vector = []float64{0.22, 0.78}
		}
		vectorStore.points[chunk.ID] = repoChunkPoint(chunk, vector)
	}

	var rerankCalled bool
	sidecarServer := newMemoryEmbeddingTestSidecarServer(
		t,
		func(request domain.EmbedMemoryRequest) domain.EmbedMemoryResponse {
			return domain.EmbedMemoryResponse{
				Items: []domain.EmbeddedMemoryVector{{
					ID:        request.Items[0].ID,
					Vector:    []float64{1, 0},
					ModelName: "embed-test",
				}},
			}
		},
		func(request domain.RerankMemoryRequest) domain.RerankMemoryResponse {
			rerankCalled = true
			return domain.RerankMemoryResponse{
				Items: []domain.RerankMemoryResult{
					{ID: findChunkIDByPath(chunks, "internal/worker.go"), Score: 0.98, Rank: 1},
					{ID: findChunkIDByPath(chunks, "internal/cache.go"), Score: 0.71, Rank: 2},
				},
			}
		},
	)
	defer sidecarServer.Close()

	svc := &Service{
		repo:                store,
		sidecar:             sidecar.New(sidecarServer.URL, time.Second),
		vectorStore:         vectorStore,
		vectorReadEnabled:   true,
		vectorRerankEnabled: true,
	}

	results, err := svc.SearchProjectChunks(context.Background(), project.ID, "redis fallback worker", 2)
	if err != nil {
		t.Fatalf("SearchProjectChunks() error = %v", err)
	}
	if !rerankCalled {
		t.Fatal("expected rerank to be called")
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if results[0].FilePath != "internal/worker.go" {
		t.Fatalf("expected reranked worker chunk first, got %q", results[0].FilePath)
	}
}

func TestSearchProjectChunksFallsBackToFTSWhenVectorSearchFails(t *testing.T) {
	store, err := openTestStore(t)
	if err != nil {
		t.Fatalf("openTestStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	project, _ := createTestProjectWithChunks(t, store, []domain.RepoChunk{
		{
			FilePath:   "internal/consumer.go",
			FileType:   "go",
			Content:    "redis stream consumer group retry handling",
			Importance: 0.32,
		},
		{
			FilePath:   "internal/http.go",
			FileType:   "go",
			Content:    "http handler and validation",
			Importance: 0.91,
		},
	})

	sidecarServer := newMemoryEmbeddingTestSidecarServer(
		t,
		func(request domain.EmbedMemoryRequest) domain.EmbedMemoryResponse {
			return domain.EmbedMemoryResponse{
				Items: []domain.EmbeddedMemoryVector{{
					ID:        request.Items[0].ID,
					Vector:    []float64{1, 0},
					ModelName: "embed-test",
				}},
			}
		},
		nil,
	)
	defer sidecarServer.Close()

	vectorStore := newFakeVectorStore()
	vectorStore.searchErr = errors.New("qdrant unavailable")
	svc := &Service{
		repo:              store,
		sidecar:           sidecar.New(sidecarServer.URL, time.Second),
		vectorStore:       vectorStore,
		vectorReadEnabled: true,
	}

	results, err := svc.SearchProjectChunks(context.Background(), project.ID, "consumer group", 1)
	if err != nil {
		t.Fatalf("SearchProjectChunks() error = %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].FilePath != "internal/consumer.go" {
		t.Fatalf("expected FTS fallback chunk, got %q", results[0].FilePath)
	}
}

func TestEnqueueProjectRepoChunkEmbeddingsQueuesOnlyMissingOrStale(t *testing.T) {
	store, err := openTestStore(t)
	if err != nil {
		t.Fatalf("openTestStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	_, chunks := createTestProjectWithChunks(t, store, []domain.RepoChunk{
		{
			FilePath:   "internal/cache.go",
			FileType:   "go",
			Content:    "redis cache flow",
			Importance: 0.8,
		},
		{
			FilePath:   "internal/queue.go",
			FileType:   "go",
			Content:    "queue retry and worker flow",
			Importance: 0.6,
		},
	})

	if err := store.UpsertRepoChunkEmbeddingRecords(context.Background(), []domain.RepoChunkEmbeddingRecord{{
		RepoChunkID: chunks[0].ID,
		ProjectID:   chunks[0].ProjectID,
		ContentHash: hashMemoryText(buildRepoChunkDocumentText(chunks[0])),
		Status:      domain.RepoChunkEmbeddingStatusIndexed,
	}}); err != nil {
		t.Fatalf("UpsertRepoChunkEmbeddingRecords() indexed error = %v", err)
	}
	if err := store.UpsertRepoChunkEmbeddingRecords(context.Background(), []domain.RepoChunkEmbeddingRecord{{
		RepoChunkID: chunks[1].ID,
		ProjectID:   chunks[1].ProjectID,
		ContentHash: "stale-hash",
		Status:      domain.RepoChunkEmbeddingStatusIndexed,
	}}); err != nil {
		t.Fatalf("UpsertRepoChunkEmbeddingRecords() stale error = %v", err)
	}

	svc := &Service{
		repo:               store,
		sidecar:            sidecar.New("http://127.0.0.1:8000", time.Second),
		vectorStore:        newFakeVectorStore(),
		vectorWriteEnabled: true,
	}

	svc.enqueueProjectRepoChunkEmbeddings(context.Background(), chunks[0].ProjectID)

	job, err := store.ClaimNextRepoChunkEmbeddingJob(context.Background(), "claim_repo_chunk", time.Now().UTC().Add(10*time.Second))
	if err != nil {
		t.Fatalf("ClaimNextRepoChunkEmbeddingJob() error = %v", err)
	}
	if job == nil {
		t.Fatal("expected one queued repo chunk embedding job")
	}
	if job.RepoChunkID != chunks[1].ID {
		t.Fatalf("expected stale chunk %s to be queued, got %s", chunks[1].ID, job.RepoChunkID)
	}
}

func TestProcessNextRepoChunkEmbeddingJobIndexesChunk(t *testing.T) {
	store, err := openTestStore(t)
	if err != nil {
		t.Fatalf("openTestStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	_, chunks := createTestProjectWithChunks(t, store, []domain.RepoChunk{{
		FilePath:   "internal/cache.go",
		FileType:   "go",
		Content:    "cache invalidation and redis consistency",
		Importance: 0.73,
	}})

	if err := store.EnqueueRepoChunkEmbeddingJobs(context.Background(), chunks); err != nil {
		t.Fatalf("EnqueueRepoChunkEmbeddingJobs() error = %v", err)
	}

	sidecarServer := newMemoryEmbeddingTestSidecarServer(
		t,
		func(request domain.EmbedMemoryRequest) domain.EmbedMemoryResponse {
			return domain.EmbedMemoryResponse{
				Items: []domain.EmbeddedMemoryVector{{
					ID:        request.Items[0].ID,
					Vector:    []float64{0.9, 0.1, 0.2},
					ModelName: "embed-test",
				}},
			}
		},
		nil,
	)
	defer sidecarServer.Close()

	vectorStore := newFakeVectorStore()
	svc := &Service{
		repo:               store,
		sidecar:            sidecar.New(sidecarServer.URL, time.Second),
		vectorStore:        vectorStore,
		vectorWriteEnabled: true,
	}

	svc.processNextRepoChunkEmbeddingJob(context.Background())

	records, err := store.GetRepoChunkEmbeddingRecordsByChunkIDs(context.Background(), []string{chunks[0].ID})
	if err != nil {
		t.Fatalf("GetRepoChunkEmbeddingRecordsByChunkIDs() error = %v", err)
	}
	if len(records) != 1 || records[0].Status != domain.RepoChunkEmbeddingStatusIndexed {
		t.Fatalf("expected indexed repo chunk record, got %+v", records)
	}
	point, ok := vectorStore.points[chunks[0].ID]
	if !ok {
		t.Fatalf("expected vector point for %s", chunks[0].ID)
	}
	if point.Payload["document_kind"] != domain.VectorDocumentKindRepoChunk {
		t.Fatalf("expected repo_chunk payload, got %+v", point.Payload)
	}
	if point.Payload["project_id"] != chunks[0].ProjectID {
		t.Fatalf("expected project_id payload, got %+v", point.Payload)
	}
}

func createTestProjectWithChunks(
	t *testing.T,
	store interface {
		CreateImportedProject(context.Context, *domain.AnalyzeRepoResponse) (*domain.ProjectProfile, error)
		ListRepoChunksByProject(context.Context, string) ([]domain.RepoChunk, error)
	},
	chunks []domain.RepoChunk,
) (*domain.ProjectProfile, []domain.RepoChunk) {
	t.Helper()

	project, err := store.CreateImportedProject(context.Background(), &domain.AnalyzeRepoResponse{
		RepoURL:       "https://github.com/octocat/repo-vector-test",
		Name:          "repo-vector-test",
		DefaultBranch: "main",
		ImportCommit:  "abc123",
		Summary:       "project summary",
		Chunks:        chunks,
	})
	if err != nil {
		t.Fatalf("CreateImportedProject() error = %v", err)
	}

	savedChunks, err := store.ListRepoChunksByProject(context.Background(), project.ID)
	if err != nil {
		t.Fatalf("ListRepoChunksByProject() error = %v", err)
	}
	return project, savedChunks
}

func repoChunkPoint(chunk domain.RepoChunk, vector []float64) vectorstore.StoredPoint {
	return vectorstore.StoredPoint{
		ID:     chunk.ID,
		Vector: vector,
		Payload: map[string]any{
			"document_kind": domain.VectorDocumentKindRepoChunk,
			"project_id":    chunk.ProjectID,
			"file_path":     chunk.FilePath,
		},
	}
}

func findChunkIDByPath(chunks []domain.RepoChunk, path string) string {
	for _, chunk := range chunks {
		if chunk.FilePath == path {
			return chunk.ID
		}
	}
	return ""
}
