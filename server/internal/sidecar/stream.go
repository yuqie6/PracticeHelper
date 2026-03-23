package sidecar

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"practicehelper/server/internal/domain"
	"practicehelper/server/internal/observability"
)

type streamEvent struct {
	Type    string          `json:"type"`
	Code    string          `json:"code,omitempty"`
	Phase   string          `json:"phase,omitempty"`
	Name    string          `json:"name,omitempty"`
	Text    string          `json:"text,omitempty"`
	Message string          `json:"message,omitempty"`
	Data    json.RawMessage `json:"data,omitempty"`
}

func (c *Client) postJSONStream(
	ctx context.Context,
	path string,
	requestBody any,
	target any,
	sideEffectsTarget any,
	emit func(domain.StreamEvent) error,
) (http.Header, string, *domain.RuntimeTrace, []domain.AgentCommandResult, error) {
	var rawOutput string
	var runtimeTrace *domain.RuntimeTrace
	var commandResults []domain.AgentCommandResult
	body, err := json.Marshal(requestBody)
	if err != nil {
		return nil, "", nil, nil, fmt.Errorf("marshal sidecar request: %w", err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return nil, "", nil, nil, fmt.Errorf("build sidecar request: %w", err)
	}

	request.Header.Set("Content-Type", "application/json")
	if requestID := observability.RequestIDFromContext(ctx); requestID != "" {
		request.Header.Set(observability.RequestIDHeader, requestID)
	}

	startedAt := time.Now()
	response, err := c.doWithRetry(request, body)
	if err != nil {
		slog.ErrorContext(ctx, "sidecar stream request failed", "path", path, "duration_ms", time.Since(startedAt).Milliseconds(), "error", err)
		return nil, "", nil, nil, fmt.Errorf("call sidecar stream: %w", err)
	}
	defer func() { _ = response.Body.Close() }()

	if response.StatusCode >= http.StatusBadRequest {
		slog.ErrorContext(ctx, "sidecar stream returned error status", "path", path, "status", response.StatusCode, "duration_ms", time.Since(startedAt).Milliseconds())
		payload, _ := io.ReadAll(response.Body)
		return nil, "", nil, nil, decodeAPIError(response.StatusCode, payload)
	}

	headers := response.Header.Clone()

	scanner := bufio.NewScanner(response.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}

		var event streamEvent
		if err := json.Unmarshal(line, &event); err != nil {
			return nil, "", nil, nil, fmt.Errorf("decode sidecar stream event: %w", err)
		}

		if event.Type == "error" {
			return nil, "", nil, nil, &APIError{
				Code:    event.Code,
				Message: event.Message,
				Status:  response.StatusCode,
			}
		}

		if event.Type == "result" {
			var err error
			rawOutput, runtimeTrace, commandResults, err = decodeResultPayload(
				event.Data,
				target,
				sideEffectsTarget,
			)
			if err != nil {
				return nil, "", nil, nil, fmt.Errorf("decode sidecar stream result: %w", err)
			}
			continue
		}

		if emit != nil {
			data, err := decodeStreamEventData(event.Data)
			if err != nil {
				return nil, "", nil, nil, fmt.Errorf("decode sidecar stream event data: %w", err)
			}
			if err := emit(domain.StreamEvent{
				Type:    event.Type,
				Code:    event.Code,
				Phase:   event.Phase,
				Name:    event.Name,
				Text:    event.Text,
				Message: event.Message,
				Data:    data,
			}); err != nil {
				return nil, "", nil, nil, err
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, "", nil, nil, fmt.Errorf("read sidecar stream: %w", err)
	}

	slog.InfoContext(ctx, "sidecar stream completed", "path", path, "status", response.StatusCode, "duration_ms", time.Since(startedAt).Milliseconds())
	return headers, rawOutput, runtimeTrace, commandResults, nil
}

func decodeStreamEventData(raw json.RawMessage) (any, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	var data any
	if err := json.Unmarshal(raw, &data); err != nil {
		return nil, err
	}
	return data, nil
}
