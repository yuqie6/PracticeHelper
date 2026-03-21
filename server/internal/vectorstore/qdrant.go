package vectorstore

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

type Point struct {
	ID      string         `json:"id"`
	Vector  []float64      `json:"vector"`
	Payload map[string]any `json:"payload,omitempty"`
}

type StoredPoint struct {
	ID      string
	Vector  []float64
	Payload map[string]any
}

type Store interface {
	Enabled() bool
	Upsert(ctx context.Context, points []Point, vectorSize int) error
	Get(ctx context.Context, ids []string) (map[string]StoredPoint, error)
}

type QdrantStore struct {
	baseURL    string
	apiKey     string
	collection string
	httpClient *http.Client

	mu         sync.Mutex
	ensuredDim int
}

func NewQdrantStore(
	baseURL string,
	apiKey string,
	collection string,
	timeout time.Duration,
) *QdrantStore {
	return &QdrantStore{
		baseURL:    strings.TrimRight(baseURL, "/"),
		apiKey:     strings.TrimSpace(apiKey),
		collection: strings.TrimSpace(collection),
		httpClient: &http.Client{Timeout: timeout},
	}
}

func (s *QdrantStore) Enabled() bool {
	return s != nil && s.baseURL != "" && s.collection != ""
}

func (s *QdrantStore) Upsert(ctx context.Context, points []Point, vectorSize int) error {
	if !s.Enabled() || len(points) == 0 {
		return nil
	}
	if vectorSize <= 0 {
		return fmt.Errorf("vector size must be positive")
	}
	if err := s.ensureCollection(ctx, vectorSize); err != nil {
		return err
	}

	payload := map[string]any{
		"points": points,
	}
	return s.callJSON(ctx, http.MethodPut, "/collections/"+s.collection+"/points?wait=true", payload, nil)
}

func (s *QdrantStore) Get(ctx context.Context, ids []string) (map[string]StoredPoint, error) {
	if !s.Enabled() || len(ids) == 0 {
		return map[string]StoredPoint{}, nil
	}

	var response struct {
		Result any `json:"result"`
	}
	if err := s.callJSON(ctx, http.MethodPost, "/collections/"+s.collection+"/points", map[string]any{
		"ids":          ids,
		"with_payload": true,
		"with_vector":  true,
	}, &response); err != nil {
		return nil, err
	}

	items := parseQdrantRetrieveResult(response.Result)
	byID := make(map[string]StoredPoint, len(items))
	for _, item := range items {
		byID[item.ID] = item
	}
	return byID, nil
}

func (s *QdrantStore) ensureCollection(ctx context.Context, vectorSize int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.ensuredDim == vectorSize {
		return nil
	}

	if err := s.callJSON(ctx, http.MethodPut, "/collections/"+s.collection, map[string]any{
		"vectors": map[string]any{
			"size":     vectorSize,
			"distance": "Cosine",
		},
	}, nil); err != nil {
		return fmt.Errorf("ensure qdrant collection: %w", err)
	}

	s.ensuredDim = vectorSize
	return nil
}

func (s *QdrantStore) callJSON(
	ctx context.Context,
	method string,
	path string,
	requestBody any,
	target any,
) error {
	body, err := json.Marshal(requestBody)
	if err != nil {
		return fmt.Errorf("marshal qdrant request: %w", err)
	}

	request, err := http.NewRequestWithContext(ctx, method, s.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build qdrant request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")
	if s.apiKey != "" {
		request.Header.Set("api-key", s.apiKey)
	}

	response, err := s.httpClient.Do(request)
	if err != nil {
		return fmt.Errorf("call qdrant: %w", err)
	}
	defer func() { _ = response.Body.Close() }()

	raw, err := io.ReadAll(response.Body)
	if err != nil {
		return fmt.Errorf("read qdrant response: %w", err)
	}
	if response.StatusCode >= http.StatusBadRequest {
		return fmt.Errorf("qdrant returned status %d: %s", response.StatusCode, strings.TrimSpace(string(raw)))
	}
	if target == nil || len(raw) == 0 {
		return nil
	}
	if err := json.Unmarshal(raw, target); err != nil {
		return fmt.Errorf("decode qdrant response: %w", err)
	}
	return nil
}

func parseQdrantRetrieveResult(raw any) []StoredPoint {
	switch value := raw.(type) {
	case []any:
		return parseQdrantPoints(value)
	case map[string]any:
		if points, ok := value["points"].([]any); ok {
			return parseQdrantPoints(points)
		}
	}
	return nil
}

func parseQdrantPoints(raw []any) []StoredPoint {
	items := make([]StoredPoint, 0, len(raw))
	for _, item := range raw {
		entry, ok := item.(map[string]any)
		if !ok {
			continue
		}
		id := fmt.Sprint(entry["id"])
		vector := parseFloat64Slice(entry["vector"])
		payload := map[string]any{}
		if rawPayload, ok := entry["payload"].(map[string]any); ok {
			payload = rawPayload
		}
		if id == "" || len(vector) == 0 {
			continue
		}
		items = append(items, StoredPoint{
			ID:      id,
			Vector:  vector,
			Payload: payload,
		})
	}
	return items
}

func parseFloat64Slice(raw any) []float64 {
	switch value := raw.(type) {
	case []any:
		items := make([]float64, 0, len(value))
		for _, item := range value {
			number, ok := item.(float64)
			if !ok {
				continue
			}
			items = append(items, number)
		}
		return items
	case []float64:
		return value
	default:
		return nil
	}
}
