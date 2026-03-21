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

const (
	promptSetHeader = "X-PracticeHelper-Prompt-Set"
	promptHashHeader = "X-PracticeHelper-Prompt-Hash"
	modelNameHeader = "X-PracticeHelper-Model-Name"
)

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
	if _, err := c.postJSON(ctx, "/internal/analyze_repo", request, &response); err != nil {
		return nil, err
	}

	return &response, nil
}

func (c *Client) AnalyzeJobTarget(
	ctx context.Context,
	request domain.AnalyzeJobTargetRequest,
) (*domain.AnalyzeJobTargetResponse, error) {
	var response domain.AnalyzeJobTargetResponse
	if _, err := c.postJSON(ctx, "/internal/analyze_job_target", request, &response); err != nil {
		return nil, err
	}

	return &response, nil
}

func (c *Client) GenerateQuestion(
	ctx context.Context,
	request domain.GenerateQuestionRequest,
) (*domain.GenerateQuestionResponse, *domain.PromptExecutionMeta, error) {
	var response domain.GenerateQuestionResponse
	headers, err := c.postJSON(ctx, "/internal/generate_question", request, &response)
	if err != nil {
		return nil, nil, err
	}

	return &response, promptMetaFromHeaders(headers), nil
}

func (c *Client) GenerateQuestionStream(
	ctx context.Context,
	request domain.GenerateQuestionRequest,
	emit func(domain.StreamEvent) error,
) (*domain.GenerateQuestionResponse, *domain.PromptExecutionMeta, error) {
	var response domain.GenerateQuestionResponse
	headers, err := c.postJSONStream(ctx, "/internal/generate_question/stream", request, &response, emit)
	if err != nil {
		return nil, nil, err
	}

	return &response, promptMetaFromHeaders(headers), nil
}

func (c *Client) EvaluateAnswer(
	ctx context.Context,
	request domain.EvaluateAnswerRequest,
) (*domain.EvaluationResult, *domain.PromptExecutionMeta, error) {
	var response domain.EvaluationResult
	headers, err := c.postJSON(ctx, "/internal/evaluate_answer", request, &response)
	if err != nil {
		return nil, nil, err
	}

	return &response, promptMetaFromHeaders(headers), nil
}

func (c *Client) EvaluateAnswerStream(
	ctx context.Context,
	request domain.EvaluateAnswerRequest,
	emit func(domain.StreamEvent) error,
) (*domain.EvaluationResult, *domain.PromptExecutionMeta, error) {
	var response domain.EvaluationResult
	headers, err := c.postJSONStream(ctx, "/internal/evaluate_answer/stream", request, &response, emit)
	if err != nil {
		return nil, nil, err
	}

	return &response, promptMetaFromHeaders(headers), nil
}

func (c *Client) GenerateReview(
	ctx context.Context,
	request domain.GenerateReviewRequest,
) (*domain.ReviewCard, *domain.PromptExecutionMeta, error) {
	var response domain.ReviewCard
	headers, err := c.postJSON(ctx, "/internal/generate_review", request, &response)
	if err != nil {
		return nil, nil, err
	}

	return &response, promptMetaFromHeaders(headers), nil
}

func (c *Client) GenerateReviewStream(
	ctx context.Context,
	request domain.GenerateReviewRequest,
	emit func(domain.StreamEvent) error,
) (*domain.ReviewCard, *domain.PromptExecutionMeta, error) {
	var response domain.ReviewCard
	headers, err := c.postJSONStream(ctx, "/internal/generate_review/stream", request, &response, emit)
	if err != nil {
		return nil, nil, err
	}

	return &response, promptMetaFromHeaders(headers), nil
}

func (c *Client) ListPromptSets(ctx context.Context) ([]domain.PromptSetSummary, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/internal/prompt-sets", nil)
	if err != nil {
		return nil, fmt.Errorf("build sidecar request: %w", err)
	}

	request.Header.Set("Accept", "application/json")
	if requestID := observability.RequestIDFromContext(ctx); requestID != "" {
		request.Header.Set(observability.RequestIDHeader, requestID)
	}

	startedAt := time.Now()
	response, err := c.httpClient.Do(request)
	if err != nil {
		slog.ErrorContext(ctx, "sidecar request failed", "path", "/internal/prompt-sets", "duration_ms", time.Since(startedAt).Milliseconds(), "error", err)
		return nil, fmt.Errorf("call sidecar: %w", err)
	}
	defer func() { _ = response.Body.Close() }()

	if response.StatusCode >= http.StatusBadRequest {
		return nil, fmt.Errorf("sidecar returned status %d", response.StatusCode)
	}

	var items []domain.PromptSetSummary
	if err := json.NewDecoder(response.Body).Decode(&items); err != nil {
		return nil, fmt.Errorf("decode sidecar response: %w", err)
	}
	return items, nil
}

func (c *Client) postJSON(ctx context.Context, path string, requestBody any, target any) (http.Header, error) {
	body, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("marshal sidecar request: %w", err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("build sidecar request: %w", err)
	}

	request.Header.Set("Content-Type", "application/json")
	if requestID := observability.RequestIDFromContext(ctx); requestID != "" {
		request.Header.Set(observability.RequestIDHeader, requestID)
	}

	startedAt := time.Now()
	response, err := c.httpClient.Do(request)
	if err != nil {
		slog.ErrorContext(ctx, "sidecar request failed", "path", path, "duration_ms", time.Since(startedAt).Milliseconds(), "error", err)
		return nil, fmt.Errorf("call sidecar: %w", err)
	}
	defer func() { _ = response.Body.Close() }()

	if response.StatusCode >= http.StatusBadRequest {
		slog.ErrorContext(ctx, "sidecar request returned error status", "path", path, "status", response.StatusCode, "duration_ms", time.Since(startedAt).Milliseconds())
		return nil, fmt.Errorf("sidecar returned status %d", response.StatusCode)
	}

	if err := json.NewDecoder(response.Body).Decode(target); err != nil {
		return nil, fmt.Errorf("decode sidecar response: %w", err)
	}

	slog.InfoContext(ctx, "sidecar request completed", "path", path, "status", response.StatusCode, "duration_ms", time.Since(startedAt).Milliseconds())

	return response.Header.Clone(), nil
}

func (c *Client) postJSONStream(
	ctx context.Context,
	path string,
	requestBody any,
	target any,
	emit func(domain.StreamEvent) error,
) (http.Header, error) {
	body, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("marshal sidecar request: %w", err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("build sidecar request: %w", err)
	}

	request.Header.Set("Content-Type", "application/json")
	if requestID := observability.RequestIDFromContext(ctx); requestID != "" {
		request.Header.Set(observability.RequestIDHeader, requestID)
	}

	startedAt := time.Now()
	response, err := c.httpClient.Do(request)
	if err != nil {
		slog.ErrorContext(ctx, "sidecar stream request failed", "path", path, "duration_ms", time.Since(startedAt).Milliseconds(), "error", err)
		return nil, fmt.Errorf("call sidecar stream: %w", err)
	}
	defer func() { _ = response.Body.Close() }()

	if response.StatusCode >= http.StatusBadRequest {
		slog.ErrorContext(ctx, "sidecar stream returned error status", "path", path, "status", response.StatusCode, "duration_ms", time.Since(startedAt).Milliseconds())
		return nil, fmt.Errorf("sidecar returned status %d", response.StatusCode)
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
			return nil, fmt.Errorf("decode sidecar stream event: %w", err)
		}

		if event.Type == "error" {
			return nil, fmt.Errorf("sidecar stream error: %s", event.Message)
		}

		if event.Type == "result" {
			if err := json.Unmarshal(event.Data, target); err != nil {
				return nil, fmt.Errorf("decode sidecar stream result: %w", err)
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
				return nil, err
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read sidecar stream: %w", err)
	}

	slog.InfoContext(ctx, "sidecar stream completed", "path", path, "status", response.StatusCode, "duration_ms", time.Since(startedAt).Milliseconds())
	return headers, nil
}

func promptMetaFromHeaders(headers http.Header) *domain.PromptExecutionMeta {
	if headers == nil {
		return &domain.PromptExecutionMeta{}
	}

	return &domain.PromptExecutionMeta{
		ModelName:   headers.Get(modelNameHeader),
		PromptSetID: headers.Get(promptSetHeader),
		PromptHash:  headers.Get(promptHashHeader),
	}
}
