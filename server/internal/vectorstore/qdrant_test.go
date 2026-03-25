package vectorstore

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestQdrantStoreUpsertEnsuresCollectionOnceAndReadsBackVectors(t *testing.T) {
	collectionCreates := 0
	pointsUpserts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/collections/practicehelper_memory":
			collectionCreates++
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"result":true}`))
		case "/collections/practicehelper_memory/points":
			if r.Method == http.MethodPut {
				pointsUpserts++
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"result":{"status":"acknowledged"}}`))
				return
			}
			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode(map[string]any{
				"result": map[string]any{
					"points": []map[string]any{
						{
							"id":      "memidx_1",
							"vector":  []float64{0.1, 0.2, 0.3},
							"payload": map[string]any{"memory_type": "observation"},
						},
					},
				},
			}); err != nil {
				t.Fatalf("encode qdrant response: %v", err)
			}
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	store := NewQdrantStore(server.URL, "secret", "practicehelper_memory", time.Second)
	if err := store.Upsert(context.Background(), []Point{{
		ID:      "memidx_1",
		Vector:  []float64{0.1, 0.2, 0.3},
		Payload: map[string]any{"memory_type": "observation"},
	}}, 3); err != nil {
		t.Fatalf("Upsert() error = %v", err)
	}
	if err := store.Upsert(context.Background(), []Point{{
		ID:      "memidx_2",
		Vector:  []float64{0.4, 0.5, 0.6},
		Payload: map[string]any{"memory_type": "session_summary"},
	}}, 3); err != nil {
		t.Fatalf("Upsert() second error = %v", err)
	}

	points, err := store.Get(context.Background(), []string{"memidx_1"})
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if collectionCreates != 1 {
		t.Fatalf("expected collection ensure once, got %d", collectionCreates)
	}
	if pointsUpserts != 2 {
		t.Fatalf("expected 2 point upserts, got %d", pointsUpserts)
	}
	if len(points["memidx_1"].Vector) != 3 {
		t.Fatalf("expected retrieved vector, got %#v", points)
	}
}

func TestQdrantStoreSearchUsesPayloadFilterAndReturnsScores(t *testing.T) {
	var capturedFilter map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/collections/practicehelper_memory/points/search":
			var requestBody map[string]any
			if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
				t.Fatalf("decode search request: %v", err)
			}
			filter, _ := requestBody["filter"].(map[string]any)
			capturedFilter = filter
			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode(map[string]any{
				"result": []map[string]any{
					{
						"id":      "chunk_1",
						"score":   0.97,
						"payload": map[string]any{"document_kind": "repo_chunk", "project_id": "proj_1"},
					},
					{
						"id":      "chunk_2",
						"score":   0.73,
						"payload": map[string]any{"document_kind": "repo_chunk", "project_id": "proj_1"},
					},
				},
			}); err != nil {
				t.Fatalf("encode search response: %v", err)
			}
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	store := NewQdrantStore(server.URL, "secret", "practicehelper_memory", time.Second)
	results, err := store.Search(context.Background(), []float64{0.2, 0.4}, SearchFilter{
		Equals: map[string]string{
			"document_kind": "repo_chunk",
			"project_id":    "proj_1",
		},
	}, 2)
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if len(results) != 2 || results[0].ID != "chunk_1" || results[0].Score != 0.97 {
		t.Fatalf("unexpected search results: %+v", results)
	}

	must, ok := capturedFilter["must"].([]any)
	if !ok || len(must) != 2 {
		t.Fatalf("expected filter.must with 2 conditions, got %+v", capturedFilter)
	}
}

func TestQdrantStoreDeletePostsPointIDs(t *testing.T) {
	var capturedIDs []string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/collections/practicehelper_memory/points/delete":
			if r.Method != http.MethodPost {
				t.Fatalf("unexpected method: %s", r.Method)
			}
			var payload struct {
				Points []string `json:"points"`
			}
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatalf("decode delete payload: %v", err)
			}
			capturedIDs = append(capturedIDs, payload.Points...)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"result":{"status":"acknowledged"}}`))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	store := NewQdrantStore(server.URL, "secret", "practicehelper_memory", time.Second)
	if err := store.Delete(context.Background(), []string{"memidx_1", "memidx_2"}); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	if len(capturedIDs) != 2 || capturedIDs[0] != "memidx_1" || capturedIDs[1] != "memidx_2" {
		t.Fatalf("unexpected deleted point ids: %+v", capturedIDs)
	}
}
