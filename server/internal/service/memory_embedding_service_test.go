package service

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sort"
	"testing"
	"time"

	"practicehelper/server/internal/domain"
	"practicehelper/server/internal/sidecar"
	"practicehelper/server/internal/vectorstore"
)

func TestSyncOrQueueMemoryEmbeddingsIndexesObservations(t *testing.T) {
	store, err := openTestStore(t)
	if err != nil {
		t.Fatalf("openTestStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	ctx := context.Background()
	observations := []domain.AgentObservation{{
		ID:        "obs_embed_ready",
		SessionID: "sess_embed_ready",
		ScopeType: domain.MemoryScopeGlobal,
		Topic:     domain.BasicsTopicRedis,
		Category:  domain.ObservationCategoryPattern,
		Content:   "用户总能说清 Redis 主线，但 trade-off 细节还是偏虚。",
		Tags:      []string{"redis", "tradeoff"},
		Relevance: 0.88,
	}}
	if err := store.CreateObservations(ctx, "sess_embed_ready", observations); err != nil {
		t.Fatalf("CreateObservations() error = %v", err)
	}

	sidecarServer := newMemoryEmbeddingTestSidecarServer(t, func(request domain.EmbedMemoryRequest) domain.EmbedMemoryResponse {
		return domain.EmbedMemoryResponse{
			Items: []domain.EmbeddedMemoryVector{{
				ID:        request.Items[0].ID,
				Vector:    []float64{0.9, 0.1, 0.2},
				ModelName: "embed-test",
			}},
		}
	}, nil)
	defer sidecarServer.Close()

	vectorStore := newFakeVectorStore()
	svc := New(
		store,
		sidecar.New(sidecarServer.URL, time.Second),
		WithVectorStore(vectorStore),
		WithVectorRetrievalConfig(true, false, false, time.Second, 10*time.Second, time.Minute),
	)

	svc.syncOrQueueMemoryEmbeddings(ctx, []domain.MemoryRef{{
		RefTable: "agent_observations",
		RefID:    "obs_embed_ready",
	}})

	entries, err := store.GetMemoryIndexEntriesByRefs(ctx, []domain.MemoryRef{{
		RefTable: "agent_observations",
		RefID:    "obs_embed_ready",
	}})
	if err != nil {
		t.Fatalf("GetMemoryIndexEntriesByRefs() error = %v", err)
	}
	records, err := store.GetMemoryEmbeddingRecordsByMemoryIndexIDs(ctx, []string{entries[0].ID})
	if err != nil {
		t.Fatalf("GetMemoryEmbeddingRecordsByMemoryIndexIDs() error = %v", err)
	}
	if len(records) != 1 || records[0].Status != domain.MemoryEmbeddingStatusIndexed {
		t.Fatalf("expected indexed embedding record, got %+v", records)
	}
	if _, ok := vectorStore.points[entries[0].ID]; !ok {
		t.Fatalf("expected vector point for %s", entries[0].ID)
	}
}

func TestSyncOrQueueMemoryEmbeddingsQueuesJobOnFailure(t *testing.T) {
	store, err := openTestStore(t)
	if err != nil {
		t.Fatalf("openTestStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	ctx := context.Background()
	observations := []domain.AgentObservation{{
		ID:        "obs_embed_queue",
		SessionID: "sess_embed_queue",
		ScopeType: domain.MemoryScopeGlobal,
		Topic:     domain.BasicsTopicRedis,
		Category:  domain.ObservationCategoryPattern,
		Content:   "这条 observation 会触发入队。",
		Tags:      []string{"redis"},
		Relevance: 0.6,
	}}
	if err := store.CreateObservations(ctx, "sess_embed_queue", observations); err != nil {
		t.Fatalf("CreateObservations() error = %v", err)
	}

	sidecarServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/internal/embed_memory" {
			http.Error(w, "provider temporarily unavailable", http.StatusServiceUnavailable)
			return
		}
		t.Fatalf("unexpected path: %s", r.URL.Path)
	}))
	defer sidecarServer.Close()

	svc := New(
		store,
		sidecar.New(sidecarServer.URL, time.Second),
		WithVectorStore(newFakeVectorStore()),
		WithVectorRetrievalConfig(true, false, false, 100*time.Millisecond, 10*time.Second, time.Minute),
	)

	svc.syncOrQueueMemoryEmbeddings(ctx, []domain.MemoryRef{{
		RefTable: "agent_observations",
		RefID:    "obs_embed_queue",
	}})

	job, err := store.ClaimNextMemoryEmbeddingJob(ctx, "claim_test", time.Now().UTC().Add(10*time.Second))
	if err != nil {
		t.Fatalf("ClaimNextMemoryEmbeddingJob() error = %v", err)
	}
	if job == nil || job.RefID != "obs_embed_queue" {
		t.Fatalf("expected queued embedding job, got %+v", job)
	}
}

func TestGetAgentContextUsesVectorSimilarityForSessionSummaries(t *testing.T) {
	store, err := openTestStore(t)
	if err != nil {
		t.Fatalf("openTestStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	ctx := context.Background()
	if err := store.UpsertSessionMemorySummary(ctx, &domain.SessionMemorySummary{
		ID:               "sm_semantic_match",
		SessionID:        "sess_semantic_match",
		Mode:             domain.ModeBasics,
		Topic:            domain.BasicsTopicRedis,
		Summary:          "用户对 Redis 缓存一致性 trade-off 的解释已经接近到位。",
		RecommendedFocus: []string{"缓存一致性 trade-off"},
		Salience:         0.55,
	}); err != nil {
		t.Fatalf("UpsertSessionMemorySummary() semantic error = %v", err)
	}
	if err := store.UpsertSessionMemorySummary(ctx, &domain.SessionMemorySummary{
		ID:               "sm_rule_only",
		SessionID:        "sess_rule_only",
		Mode:             domain.ModeBasics,
		Topic:            domain.BasicsTopicRedis,
		Summary:          "高热度但语义更泛的 Redis 总结。",
		RecommendedFocus: []string{"Redis 基础"},
		Salience:         0.95,
	}); err != nil {
		t.Fatalf("UpsertSessionMemorySummary() rule error = %v", err)
	}

	entries, err := store.GetMemoryIndexEntriesByRefs(ctx, []domain.MemoryRef{
		{RefTable: "session_memory_summaries", RefID: "sm_semantic_match"},
		{RefTable: "session_memory_summaries", RefID: "sm_rule_only"},
	})
	if err != nil {
		t.Fatalf("GetMemoryIndexEntriesByRefs() error = %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 memory index entries, got %d", len(entries))
	}
	semanticEntry := entries[0]
	ruleEntry := entries[1]

	sidecarServer := newMemoryEmbeddingTestSidecarServer(
		t,
		func(request domain.EmbedMemoryRequest) domain.EmbedMemoryResponse {
			items := make([]domain.EmbeddedMemoryVector, 0, len(request.Items))
			for _, item := range request.Items {
				vector := []float64{0.1, 0.1, 0.1}
				switch item.ID {
				case "query":
					vector = []float64{1, 0, 0}
				case semanticEntry.ID:
					vector = []float64{0.98, 0.02, 0}
				case ruleEntry.ID:
					vector = []float64{0.15, 0.85, 0}
				}
				items = append(items, domain.EmbeddedMemoryVector{
					ID:        item.ID,
					Vector:    vector,
					ModelName: "embed-test",
				})
			}
			return domain.EmbedMemoryResponse{Items: items}
		},
		nil,
	)
	defer sidecarServer.Close()

	vectorStore := newFakeVectorStore()
	svc := New(
		store,
		sidecar.New(sidecarServer.URL, time.Second),
		WithVectorStore(vectorStore),
		WithVectorRetrievalConfig(true, true, false, time.Second, 10*time.Second, time.Minute),
	)

	if err := svc.indexMemoryIndexEntries(ctx, entries); err != nil {
		t.Fatalf("indexMemoryIndexEntries() error = %v", err)
	}

	agentContext, err := svc.getAgentContext(ctx, agentContextParams{
		Topic:               domain.BasicsTopicRedis,
		SessionID:           "sess_current",
		SessionSummaryLimit: 1,
	})
	if err != nil {
		t.Fatalf("getAgentContext() error = %v", err)
	}

	if len(agentContext.SessionSummaries) != 1 {
		t.Fatalf("expected one session summary, got %d", len(agentContext.SessionSummaries))
	}
	if agentContext.SessionSummaries[0].ID != "sm_semantic_match" {
		t.Fatalf("expected semantic summary first, got %q", agentContext.SessionSummaries[0].ID)
	}
}

type fakeVectorStore struct {
	points    map[string]vectorstore.StoredPoint
	deleted   []string
	searchErr error
}

func newFakeVectorStore() *fakeVectorStore {
	return &fakeVectorStore{points: make(map[string]vectorstore.StoredPoint)}
}

func (s *fakeVectorStore) Enabled() bool {
	return true
}

func (s *fakeVectorStore) Upsert(_ context.Context, points []vectorstore.Point, _ int) error {
	for _, point := range points {
		s.points[point.ID] = vectorstore.StoredPoint{
			ID:      point.ID,
			Vector:  append([]float64(nil), point.Vector...),
			Payload: point.Payload,
		}
	}
	return nil
}

func (s *fakeVectorStore) Delete(_ context.Context, ids []string) error {
	s.deleted = append(s.deleted, ids...)
	for _, id := range ids {
		delete(s.points, id)
	}
	return nil
}

func (s *fakeVectorStore) Get(_ context.Context, ids []string) (map[string]vectorstore.StoredPoint, error) {
	items := make(map[string]vectorstore.StoredPoint, len(ids))
	for _, id := range ids {
		point, ok := s.points[id]
		if !ok {
			continue
		}
		items[id] = point
	}
	return items, nil
}

func (s *fakeVectorStore) Search(
	_ context.Context,
	vector []float64,
	filter vectorstore.SearchFilter,
	limit int,
) ([]vectorstore.SearchResult, error) {
	if s.searchErr != nil {
		return nil, s.searchErr
	}

	results := make([]vectorstore.SearchResult, 0, len(s.points))
	for _, point := range s.points {
		if !matchesVectorFilter(point.Payload, filter) {
			continue
		}
		results = append(results, vectorstore.SearchResult{
			ID:      point.ID,
			Score:   cosineSimilarity(vector, point.Vector),
			Payload: point.Payload,
		})
	}

	sort.SliceStable(results, func(i, j int) bool {
		if results[i].Score != results[j].Score {
			return results[i].Score > results[j].Score
		}
		return results[i].ID < results[j].ID
	})
	if limit > 0 && len(results) > limit {
		results = results[:limit]
	}
	return results, nil
}

func matchesVectorFilter(payload map[string]any, filter vectorstore.SearchFilter) bool {
	if len(filter.Equals) == 0 {
		return true
	}

	for key, expected := range filter.Equals {
		if expected == "" {
			continue
		}
		actual, ok := payload[key]
		if !ok || actual != expected {
			return false
		}
	}
	return true
}

func newMemoryEmbeddingTestSidecarServer(
	t *testing.T,
	embed func(domain.EmbedMemoryRequest) domain.EmbedMemoryResponse,
	rerank func(domain.RerankMemoryRequest) domain.RerankMemoryResponse,
) *httptest.Server {
	t.Helper()

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if handlePromptSetRequest(t, w, r) {
			return
		}
		switch r.URL.Path {
		case "/internal/embed_memory":
			var request domain.EmbedMemoryRequest
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				t.Fatalf("decode embed request: %v", err)
			}
			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode(embed(request)); err != nil {
				t.Fatalf("encode embed response: %v", err)
			}
		case "/internal/rerank_memory":
			var request domain.RerankMemoryRequest
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				t.Fatalf("decode rerank request: %v", err)
			}
			w.Header().Set("Content-Type", "application/json")
			response := domain.RerankMemoryResponse{}
			if rerank != nil {
				response = rerank(request)
			}
			if err := json.NewEncoder(w).Encode(response); err != nil {
				t.Fatalf("encode rerank response: %v", err)
			}
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
}
