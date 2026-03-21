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
