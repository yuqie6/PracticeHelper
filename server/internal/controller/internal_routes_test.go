package controller

import (
	"context"
	"encoding/json"
	"math"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"practicehelper/server/internal/domain"
	"practicehelper/server/internal/service"
	"practicehelper/server/internal/sidecar"
	"practicehelper/server/internal/vectorstore"
)

func TestInternalSearchChunksRequiresValidToken(t *testing.T) {
	store, err := openTestStore(t)
	if err != nil {
		t.Fatalf("openTestStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	project, err := store.CreateImportedProject(context.Background(), &domain.AnalyzeRepoResponse{
		RepoURL:       "https://example.com/mirror.git",
		Name:          "Mirror",
		DefaultBranch: "main",
		ImportCommit:  "abc123",
		Summary:       "Agent workflow",
		Chunks: []domain.RepoChunk{
			{
				FilePath:   "internal/runtime.go",
				FileType:   ".go",
				Content:    "redis cache consistency and retries",
				Importance: 1.0,
				FTSKey:     "internal/runtime.go#0",
			},
		},
	})
	if err != nil {
		t.Fatalf("CreateImportedProject() error = %v", err)
	}

	router := NewRouterWithInternalToken(service.New(store, nil), "secret-token")

	unauthorized := httptest.NewRecorder()
	router.ServeHTTP(
		unauthorized,
		httptest.NewRequest(
			http.MethodGet,
			"/internal/search-chunks?project_id="+project.ID+"&query=redis&limit=2",
			nil,
		),
	)
	if unauthorized.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d body=%s", unauthorized.Code, unauthorized.Body.String())
	}

	request := httptest.NewRequest(
		http.MethodGet,
		"/internal/search-chunks?project_id="+project.ID+"&query=redis&limit=2",
		nil,
	)
	request.Header.Set(internalTokenHeader, "secret-token")

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", recorder.Code, recorder.Body.String())
	}

	chunks := decodeDataEnvelope[[]domain.RepoChunk](t, recorder.Body.Bytes())
	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(chunks))
	}
	if chunks[0].FilePath != "internal/runtime.go" {
		t.Fatalf("unexpected file path: %q", chunks[0].FilePath)
	}
}

func TestInternalSessionDetailReturnsSessionAndReview(t *testing.T) {
	store, err := openTestStore(t)
	if err != nil {
		t.Fatalf("openTestStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	session := &domain.TrainingSession{
		ID:        "sess_internal_detail",
		Mode:      domain.ModeBasics,
		Topic:     domain.BasicsTopicRedis,
		Intensity: "standard",
		Status:    domain.StatusCompleted,
		MaxTurns:  2,
		ReviewID:  "review_internal_detail",
	}
	turn := &domain.TrainingTurn{
		ID:             "turn_internal_detail",
		SessionID:      session.ID,
		TurnIndex:      1,
		Stage:          "question",
		Question:       "Redis 为什么快？",
		ExpectedPoints: []string{"内存访问", "事件循环"},
		Answer:         "因为主要在内存里。",
	}
	if err := store.CreateSession(context.Background(), session, turn); err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}
	if err := store.CreateReview(context.Background(), &domain.ReviewCard{
		ID:                session.ReviewID,
		SessionID:         session.ID,
		Overall:           "整体过线。",
		TopFix:            "补细节",
		TopFixReason:      "案例不够具体",
		Highlights:        []string{"主线清楚"},
		Gaps:              []string{"案例不足"},
		SuggestedTopics:   []string{"redis"},
		NextTrainingFocus: []string{"补真实案例"},
		ScoreBreakdown:    map[string]float64{"准确性": 80},
	}); err != nil {
		t.Fatalf("CreateReview() error = %v", err)
	}

	router := NewRouterWithInternalToken(service.New(store, nil), "secret-token")
	request := httptest.NewRequest(
		http.MethodGet,
		"/internal/session-detail/"+session.ID,
		nil,
	)
	request.Header.Set(internalTokenHeader, "secret-token")

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", recorder.Code, recorder.Body.String())
	}

	detail := decodeDataEnvelope[domain.AgentSessionDetail](t, recorder.Body.Bytes())
	if detail.Session == nil || detail.Session.ID != session.ID {
		t.Fatalf("unexpected session detail: %+v", detail.Session)
	}
	if len(detail.Session.Turns) != 1 {
		t.Fatalf("expected 1 turn, got %+v", detail.Session.Turns)
	}
	if detail.Review == nil || detail.Review.Overall != "整体过线。" {
		t.Fatalf("unexpected review detail: %+v", detail.Review)
	}
}

func TestInternalSearchChunksUsesVectorRecallAndRerank(t *testing.T) {
	store, err := openTestStore(t)
	if err != nil {
		t.Fatalf("openTestStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	project, err := store.CreateImportedProject(context.Background(), &domain.AnalyzeRepoResponse{
		RepoURL:       "https://example.com/rag-vector.git",
		Name:          "rag-vector",
		DefaultBranch: "main",
		ImportCommit:  "abc123",
		Summary:       "RAG vector route smoke",
		Chunks: []domain.RepoChunk{
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
		},
	})
	if err != nil {
		t.Fatalf("CreateImportedProject() error = %v", err)
	}

	chunks, err := store.ListRepoChunksByProject(context.Background(), project.ID)
	if err != nil {
		t.Fatalf("ListRepoChunksByProject() error = %v", err)
	}
	if len(chunks) != 2 {
		t.Fatalf("expected 2 chunks, got %d", len(chunks))
	}

	qdrantServer := newTestQdrantServer(t)
	defer qdrantServer.Close()

	vectorStore := vectorstore.NewQdrantStore(
		qdrantServer.URL(),
		"",
		"practicehelper_memory",
		time.Second,
	)
	if err := vectorStore.Upsert(context.Background(), []vectorstore.Point{
		{
			ID:     chunks[0].ID,
			Vector: []float64{0.99, 0.01},
			Payload: map[string]any{
				"document_kind": domain.VectorDocumentKindRepoChunk,
				"project_id":    project.ID,
				"file_path":     chunks[0].FilePath,
			},
		},
		{
			ID:     chunks[1].ID,
			Vector: []float64{0.76, 0.24},
			Payload: map[string]any{
				"document_kind": domain.VectorDocumentKindRepoChunk,
				"project_id":    project.ID,
				"file_path":     chunks[1].FilePath,
			},
		},
	}, 2); err != nil {
		t.Fatalf("Upsert() error = %v", err)
	}

	sidecarServer := newInternalSearchSidecarServer(
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
			return domain.RerankMemoryResponse{
				Items: []domain.RerankMemoryResult{
					{ID: chunks[1].ID, Score: 0.98, Rank: 1},
					{ID: chunks[0].ID, Score: 0.71, Rank: 2},
				},
			}
		},
	)
	defer sidecarServer.Close()

	router := NewRouterWithInternalToken(
		service.New(
			store,
			sidecar.New(sidecarServer.URL, time.Second),
			service.WithVectorStore(vectorStore),
			service.WithVectorRetrievalConfig(false, true, true, time.Second, 10*time.Second, time.Minute),
		),
		"secret-token",
	)

	request := httptest.NewRequest(
		http.MethodGet,
		"/internal/search-chunks?project_id="+project.ID+"&query=redis%20fallback%20worker&limit=2",
		nil,
	)
	request.Header.Set(internalTokenHeader, "secret-token")

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", recorder.Code, recorder.Body.String())
	}

	results := decodeDataEnvelope[[]domain.RepoChunk](t, recorder.Body.Bytes())
	if len(results) != 2 {
		t.Fatalf("expected 2 chunks, got %d", len(results))
	}
	if results[0].FilePath != "internal/worker.go" {
		t.Fatalf("expected reranked worker chunk first, got %q", results[0].FilePath)
	}
}

func TestInternalSearchChunksFallsBackToFTSWhenQdrantUnavailable(t *testing.T) {
	store, err := openTestStore(t)
	if err != nil {
		t.Fatalf("openTestStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	project, err := store.CreateImportedProject(context.Background(), &domain.AnalyzeRepoResponse{
		RepoURL:       "https://example.com/rag-fallback.git",
		Name:          "rag-fallback",
		DefaultBranch: "main",
		ImportCommit:  "abc123",
		Summary:       "RAG fallback route smoke",
		Chunks: []domain.RepoChunk{
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
		},
	})
	if err != nil {
		t.Fatalf("CreateImportedProject() error = %v", err)
	}

	sidecarServer := newInternalSearchSidecarServer(
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

	router := NewRouterWithInternalToken(
		service.New(
			store,
			sidecar.New(sidecarServer.URL, time.Second),
			service.WithVectorStore(
				vectorstore.NewQdrantStore("http://127.0.0.1:1", "", "practicehelper_memory", 50*time.Millisecond),
			),
			service.WithVectorRetrievalConfig(false, true, false, time.Second, 10*time.Second, time.Minute),
		),
		"secret-token",
	)

	request := httptest.NewRequest(
		http.MethodGet,
		"/internal/search-chunks?project_id="+project.ID+"&query=consumer%20group&limit=1",
		nil,
	)
	request.Header.Set(internalTokenHeader, "secret-token")

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", recorder.Code, recorder.Body.String())
	}

	results := decodeDataEnvelope[[]domain.RepoChunk](t, recorder.Body.Bytes())
	if len(results) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(results))
	}
	if results[0].FilePath != "internal/consumer.go" {
		t.Fatalf("expected FTS fallback chunk, got %q", results[0].FilePath)
	}
}

type testQdrantServer struct {
	server *httptest.Server
	points map[string]vectorstore.StoredPoint
}

func newTestQdrantServer(t *testing.T) *testQdrantServer {
	t.Helper()

	state := &testQdrantServer{
		points: make(map[string]vectorstore.StoredPoint),
	}
	state.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/collections/practicehelper_memory":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"result":true}`))
		case "/collections/practicehelper_memory/points":
			if r.Method != http.MethodPut {
				t.Fatalf("unexpected method for points: %s", r.Method)
			}
			var payload struct {
				Points []vectorstore.Point `json:"points"`
			}
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatalf("decode upsert payload: %v", err)
			}
			for _, point := range payload.Points {
				state.points[point.ID] = vectorstore.StoredPoint{
					ID:      point.ID,
					Vector:  append([]float64(nil), point.Vector...),
					Payload: point.Payload,
				}
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"result":{"status":"acknowledged"}}`))
		case "/collections/practicehelper_memory/points/search":
			var requestBody struct {
				Vector []float64 `json:"vector"`
				Limit  int       `json:"limit"`
				Filter struct {
					Must []struct {
						Key   string `json:"key"`
						Match struct {
							Value string `json:"value"`
						} `json:"match"`
					} `json:"must"`
				} `json:"filter"`
			}
			if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
				t.Fatalf("decode search payload: %v", err)
			}

			filters := make(map[string]string, len(requestBody.Filter.Must))
			for _, item := range requestBody.Filter.Must {
				filters[item.Key] = item.Match.Value
			}

			results := make([]map[string]any, 0, len(state.points))
			for _, point := range state.points {
				if !matchesQdrantFilters(point.Payload, filters) {
					continue
				}
				results = append(results, map[string]any{
					"id":      point.ID,
					"score":   repoChunkCosineSimilarity(requestBody.Vector, point.Vector),
					"payload": point.Payload,
				})
			}
			sortSearchResults(results)
			if requestBody.Limit > 0 && len(results) > requestBody.Limit {
				results = results[:requestBody.Limit]
			}
			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode(map[string]any{"result": results}); err != nil {
				t.Fatalf("encode search response: %v", err)
			}
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	return state
}

func (s *testQdrantServer) URL() string {
	return s.server.URL
}

func (s *testQdrantServer) Close() {
	s.server.Close()
}

func matchesQdrantFilters(payload map[string]any, filters map[string]string) bool {
	for key, expected := range filters {
		if expected == "" {
			continue
		}
		if payload[key] != expected {
			return false
		}
	}
	return true
}

func sortSearchResults(results []map[string]any) {
	for i := 0; i < len(results); i++ {
		for j := i + 1; j < len(results); j++ {
			left, _ := results[i]["score"].(float64)
			right, _ := results[j]["score"].(float64)
			if right > left {
				results[i], results[j] = results[j], results[i]
			}
		}
	}
}

func repoChunkCosineSimilarity(left []float64, right []float64) float64 {
	if len(left) == 0 || len(right) == 0 {
		return 0
	}
	size := len(left)
	if len(right) < size {
		size = len(right)
	}

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
	return dot / (math.Sqrt(leftNorm)*math.Sqrt(rightNorm) + 1e-12)
}

func newInternalSearchSidecarServer(
	t *testing.T,
	embed func(domain.EmbedMemoryRequest) domain.EmbedMemoryResponse,
	rerank func(domain.RerankMemoryRequest) domain.RerankMemoryResponse,
) *httptest.Server {
	t.Helper()

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
			response := domain.RerankMemoryResponse{}
			if rerank != nil {
				response = rerank(request)
			}
			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode(response); err != nil {
				t.Fatalf("encode rerank response: %v", err)
			}
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
}
