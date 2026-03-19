package sidecar

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"practicehelper/server/internal/domain"
	"practicehelper/server/internal/observability"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
}

type streamEvent struct {
	Type    string          `json:"type"`
	Phase   string          `json:"phase,omitempty"`
	Name    string          `json:"name,omitempty"`
	Text    string          `json:"text,omitempty"`
	Message string          `json:"message,omitempty"`
	Data    json.RawMessage `json:"data,omitempty"`
}

func New(baseURL string, timeout time.Duration) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

func (c *Client) AnalyzeRepo(ctx context.Context, request domain.AnalyzeRepoRequest) (*domain.AnalyzeRepoResponse, error) {
	var response domain.AnalyzeRepoResponse
	if err := c.postJSON(ctx, "/internal/analyze_repo", request, &response); err != nil {
		return nil, err
	}

	return &response, nil
}

func (c *Client) GenerateQuestion(ctx context.Context, request domain.GenerateQuestionRequest) (*domain.GenerateQuestionResponse, error) {
	var response domain.GenerateQuestionResponse
	if err := c.postJSON(ctx, "/internal/generate_question", request, &response); err != nil {
		return nil, err
	}

	return &response, nil
}

func (c *Client) GenerateQuestionStream(
	ctx context.Context,
	request domain.GenerateQuestionRequest,
	emit func(domain.StreamEvent) error,
) (*domain.GenerateQuestionResponse, error) {
	var response domain.GenerateQuestionResponse
	if err := c.postJSONStream(ctx, "/internal/generate_question/stream", request, &response, emit); err != nil {
		return nil, err
	}

	return &response, nil
}

func (c *Client) EvaluateAnswer(ctx context.Context, request domain.EvaluateAnswerRequest) (*domain.EvaluationResult, error) {
	var response domain.EvaluationResult
	if err := c.postJSON(ctx, "/internal/evaluate_answer", request, &response); err != nil {
		return nil, err
	}

	return &response, nil
}

func (c *Client) EvaluateAnswerStream(
	ctx context.Context,
	request domain.EvaluateAnswerRequest,
	emit func(domain.StreamEvent) error,
) (*domain.EvaluationResult, error) {
	var response domain.EvaluationResult
	if err := c.postJSONStream(ctx, "/internal/evaluate_answer/stream", request, &response, emit); err != nil {
		return nil, err
	}

	return &response, nil
}

func (c *Client) GenerateReview(ctx context.Context, request domain.GenerateReviewRequest) (*domain.ReviewCard, error) {
	var response domain.ReviewCard
	if err := c.postJSON(ctx, "/internal/generate_review", request, &response); err != nil {
		return nil, err
	}

	return &response, nil
}

func (c *Client) GenerateReviewStream(
	ctx context.Context,
	request domain.GenerateReviewRequest,
	emit func(domain.StreamEvent) error,
) (*domain.ReviewCard, error) {
	var response domain.ReviewCard
	if err := c.postJSONStream(ctx, "/internal/generate_review/stream", request, &response, emit); err != nil {
		return nil, err
	}

	return &response, nil
}

func (c *Client) postJSON(ctx context.Context, path string, requestBody any, target any) error {
	body, err := json.Marshal(requestBody)
	if err != nil {
		return fmt.Errorf("marshal sidecar request: %w", err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build sidecar request: %w", err)
	}

	request.Header.Set("Content-Type", "application/json")
	if requestID := observability.RequestIDFromContext(ctx); requestID != "" {
		request.Header.Set(observability.RequestIDHeader, requestID)
	}

	startedAt := time.Now()
	response, err := c.httpClient.Do(request)
	if err != nil {
		slog.ErrorContext(ctx, "sidecar request failed", "path", path, "duration_ms", time.Since(startedAt).Milliseconds(), "error", err)
		return fmt.Errorf("call sidecar: %w", err)
	}
	defer func() { _ = response.Body.Close() }()

	if response.StatusCode >= http.StatusBadRequest {
		slog.ErrorContext(ctx, "sidecar request returned error status", "path", path, "status", response.StatusCode, "duration_ms", time.Since(startedAt).Milliseconds())
		return fmt.Errorf("sidecar returned status %d", response.StatusCode)
	}

	if err := json.NewDecoder(response.Body).Decode(target); err != nil {
		return fmt.Errorf("decode sidecar response: %w", err)
	}

	slog.InfoContext(ctx, "sidecar request completed", "path", path, "status", response.StatusCode, "duration_ms", time.Since(startedAt).Milliseconds())

	return nil
}

func (c *Client) postJSONStream(
	ctx context.Context,
	path string,
	requestBody any,
	target any,
	emit func(domain.StreamEvent) error,
) error {
	body, err := json.Marshal(requestBody)
	if err != nil {
		return fmt.Errorf("marshal sidecar request: %w", err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build sidecar request: %w", err)
	}

	request.Header.Set("Content-Type", "application/json")
	if requestID := observability.RequestIDFromContext(ctx); requestID != "" {
		request.Header.Set(observability.RequestIDHeader, requestID)
	}

	startedAt := time.Now()
	response, err := c.httpClient.Do(request)
	if err != nil {
		slog.ErrorContext(ctx, "sidecar stream request failed", "path", path, "duration_ms", time.Since(startedAt).Milliseconds(), "error", err)
		return fmt.Errorf("call sidecar stream: %w", err)
	}
	defer func() { _ = response.Body.Close() }()

	if response.StatusCode >= http.StatusBadRequest {
		slog.ErrorContext(ctx, "sidecar stream returned error status", "path", path, "status", response.StatusCode, "duration_ms", time.Since(startedAt).Milliseconds())
		return fmt.Errorf("sidecar returned status %d", response.StatusCode)
	}

	scanner := bufio.NewScanner(response.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}

		var event streamEvent
		if err := json.Unmarshal(line, &event); err != nil {
			return fmt.Errorf("decode sidecar stream event: %w", err)
		}

		if event.Type == "error" {
			return fmt.Errorf("sidecar stream error: %s", event.Message)
		}

		if event.Type == "result" {
			if err := json.Unmarshal(event.Data, target); err != nil {
				return fmt.Errorf("decode sidecar stream result: %w", err)
			}
			continue
		}

		if emit != nil {
			if err := emit(domain.StreamEvent{
				Type:    event.Type,
				Phase:   event.Phase,
				Name:    event.Name,
				Text:    event.Text,
				Message: event.Message,
			}); err != nil {
				return err
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read sidecar stream: %w", err)
	}

	slog.InfoContext(ctx, "sidecar stream completed", "path", path, "status", response.StatusCode, "duration_ms", time.Since(startedAt).Milliseconds())
	return nil
}
