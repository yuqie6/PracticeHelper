package sidecar

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math/rand"
	"net"
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
	promptSetHeader  = "X-PracticeHelper-Prompt-Set"
	promptHashHeader = "X-PracticeHelper-Prompt-Hash"
	modelNameHeader  = "X-PracticeHelper-Model-Name"
)

var (
	sidecarRetryBackoffs = []time.Duration{
		500 * time.Millisecond,
		1500 * time.Millisecond,
	}
	sidecarRetryJitter = 200 * time.Millisecond
)

type streamEvent struct {
	Type    string          `json:"type"`
	Phase   string          `json:"phase,omitempty"`
	Name    string          `json:"name,omitempty"`
	Text    string          `json:"text,omitempty"`
	Message string          `json:"message,omitempty"`
	Data    json.RawMessage `json:"data,omitempty"`
}

type responseEnvelope struct {
	Result    json.RawMessage `json:"result"`
	RawOutput string          `json:"raw_output,omitempty"`
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
	if _, _, err := c.postJSON(ctx, "/internal/analyze_repo", request, &response); err != nil {
		return nil, err
	}

	return &response, nil
}

func (c *Client) AnalyzeJobTarget(
	ctx context.Context,
	request domain.AnalyzeJobTargetRequest,
) (*domain.AnalyzeJobTargetResponse, error) {
	var response domain.AnalyzeJobTargetResponse
	if _, _, err := c.postJSON(ctx, "/internal/analyze_job_target", request, &response); err != nil {
		return nil, err
	}

	return &response, nil
}

func (c *Client) GenerateQuestion(
	ctx context.Context,
	request domain.GenerateQuestionRequest,
) (*domain.GenerateQuestionResponse, *domain.PromptExecutionMeta, error) {
	var response domain.GenerateQuestionResponse
	headers, rawOutput, err := c.postJSON(ctx, "/internal/generate_question", request, &response)
	if err != nil {
		return nil, nil, err
	}

	meta := promptMetaFromHeaders(headers)
	meta.RawOutput = rawOutput
	return &response, meta, nil
}

func (c *Client) GenerateQuestionStream(
	ctx context.Context,
	request domain.GenerateQuestionRequest,
	emit func(domain.StreamEvent) error,
) (*domain.GenerateQuestionResponse, *domain.PromptExecutionMeta, error) {
	var response domain.GenerateQuestionResponse
	headers, rawOutput, err := c.postJSONStream(ctx, "/internal/generate_question/stream", request, &response, emit)
	if err != nil {
		return nil, nil, err
	}

	meta := promptMetaFromHeaders(headers)
	meta.RawOutput = rawOutput
	return &response, meta, nil
}

func (c *Client) EvaluateAnswer(
	ctx context.Context,
	request domain.EvaluateAnswerRequest,
) (*domain.EvaluationResult, *domain.PromptExecutionMeta, error) {
	var response domain.EvaluationResult
	headers, rawOutput, err := c.postJSON(ctx, "/internal/evaluate_answer", request, &response)
	if err != nil {
		return nil, nil, err
	}

	meta := promptMetaFromHeaders(headers)
	meta.RawOutput = rawOutput
	return &response, meta, nil
}

func (c *Client) EvaluateAnswerStream(
	ctx context.Context,
	request domain.EvaluateAnswerRequest,
	emit func(domain.StreamEvent) error,
) (*domain.EvaluationResult, *domain.PromptExecutionMeta, error) {
	var response domain.EvaluationResult
	headers, rawOutput, err := c.postJSONStream(ctx, "/internal/evaluate_answer/stream", request, &response, emit)
	if err != nil {
		return nil, nil, err
	}

	meta := promptMetaFromHeaders(headers)
	meta.RawOutput = rawOutput
	return &response, meta, nil
}

func (c *Client) GenerateReview(
	ctx context.Context,
	request domain.GenerateReviewRequest,
) (*domain.ReviewCard, *domain.PromptExecutionMeta, error) {
	var response domain.ReviewCard
	headers, rawOutput, err := c.postJSON(ctx, "/internal/generate_review", request, &response)
	if err != nil {
		return nil, nil, err
	}

	meta := promptMetaFromHeaders(headers)
	meta.RawOutput = rawOutput
	return &response, meta, nil
}

func (c *Client) GenerateReviewStream(
	ctx context.Context,
	request domain.GenerateReviewRequest,
	emit func(domain.StreamEvent) error,
) (*domain.ReviewCard, *domain.PromptExecutionMeta, error) {
	var response domain.ReviewCard
	headers, rawOutput, err := c.postJSONStream(ctx, "/internal/generate_review/stream", request, &response, emit)
	if err != nil {
		return nil, nil, err
	}

	meta := promptMetaFromHeaders(headers)
	meta.RawOutput = rawOutput
	return &response, meta, nil
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

func (c *Client) postJSON(
	ctx context.Context,
	path string,
	requestBody any,
	target any,
) (http.Header, string, error) {
	body, err := json.Marshal(requestBody)
	if err != nil {
		return nil, "", fmt.Errorf("marshal sidecar request: %w", err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return nil, "", fmt.Errorf("build sidecar request: %w", err)
	}

	request.Header.Set("Content-Type", "application/json")
	if requestID := observability.RequestIDFromContext(ctx); requestID != "" {
		request.Header.Set(observability.RequestIDHeader, requestID)
	}

	startedAt := time.Now()
	response, err := c.doWithRetry(request, body)
	if err != nil {
		slog.ErrorContext(ctx, "sidecar request failed", "path", path, "duration_ms", time.Since(startedAt).Milliseconds(), "error", err)
		return nil, "", fmt.Errorf("call sidecar: %w", err)
	}
	defer func() { _ = response.Body.Close() }()

	if response.StatusCode >= http.StatusBadRequest {
		slog.ErrorContext(ctx, "sidecar request returned error status", "path", path, "status", response.StatusCode, "duration_ms", time.Since(startedAt).Milliseconds())
		return nil, "", fmt.Errorf("sidecar returned status %d", response.StatusCode)
	}

	payload, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, "", fmt.Errorf("read sidecar response: %w", err)
	}
	rawOutput, err := decodeResultPayload(payload, target)
	if err != nil {
		return nil, "", fmt.Errorf("decode sidecar response: %w", err)
	}

	slog.InfoContext(ctx, "sidecar request completed", "path", path, "status", response.StatusCode, "duration_ms", time.Since(startedAt).Milliseconds())

	return response.Header.Clone(), rawOutput, nil
}

func (c *Client) postJSONStream(
	ctx context.Context,
	path string,
	requestBody any,
	target any,
	emit func(domain.StreamEvent) error,
) (http.Header, string, error) {
	var rawOutput string
	body, err := json.Marshal(requestBody)
	if err != nil {
		return nil, "", fmt.Errorf("marshal sidecar request: %w", err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return nil, "", fmt.Errorf("build sidecar request: %w", err)
	}

	request.Header.Set("Content-Type", "application/json")
	if requestID := observability.RequestIDFromContext(ctx); requestID != "" {
		request.Header.Set(observability.RequestIDHeader, requestID)
	}

	startedAt := time.Now()
	response, err := c.doWithRetry(request, body)
	if err != nil {
		slog.ErrorContext(ctx, "sidecar stream request failed", "path", path, "duration_ms", time.Since(startedAt).Milliseconds(), "error", err)
		return nil, "", fmt.Errorf("call sidecar stream: %w", err)
	}
	defer func() { _ = response.Body.Close() }()

	if response.StatusCode >= http.StatusBadRequest {
		slog.ErrorContext(ctx, "sidecar stream returned error status", "path", path, "status", response.StatusCode, "duration_ms", time.Since(startedAt).Milliseconds())
		return nil, "", fmt.Errorf("sidecar returned status %d", response.StatusCode)
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
			return nil, "", fmt.Errorf("decode sidecar stream event: %w", err)
		}

		if event.Type == "error" {
			return nil, "", fmt.Errorf("sidecar stream error: %s", event.Message)
		}

		if event.Type == "result" {
			var err error
			rawOutput, err = decodeResultPayload(event.Data, target)
			if err != nil {
				return nil, "", fmt.Errorf("decode sidecar stream result: %w", err)
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
				return nil, "", err
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, "", fmt.Errorf("read sidecar stream: %w", err)
	}

	slog.InfoContext(ctx, "sidecar stream completed", "path", path, "status", response.StatusCode, "duration_ms", time.Since(startedAt).Milliseconds())
	return headers, rawOutput, nil
}

func (c *Client) doWithRetry(req *http.Request, body []byte) (*http.Response, error) {
	const maxAttempts = 3

	for attempt := 0; attempt < maxAttempts; attempt++ {
		attemptRequest := req.Clone(req.Context())
		attemptRequest.Body = io.NopCloser(bytes.NewReader(body))
		attemptRequest.ContentLength = int64(len(body))

		response, err := c.httpClient.Do(attemptRequest)
		if err == nil {
			if !isRetryableStatus(response.StatusCode) || attempt == maxAttempts-1 {
				return response, nil
			}
			_ = response.Body.Close()
		} else {
			if !isRetryableError(err) || attempt == maxAttempts-1 {
				return nil, err
			}
		}

		if attempt >= len(sidecarRetryBackoffs) {
			continue
		}

		wait := sidecarRetryBackoffs[attempt] + retryJitter()
		if wait < 0 {
			wait = 0
		}
		timer := time.NewTimer(wait)
		select {
		case <-req.Context().Done():
			timer.Stop()
			return nil, req.Context().Err()
		case <-timer.C:
		}
	}

	return nil, fmt.Errorf("sidecar request failed after retries")
}

func isRetryableError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}

	var netErr net.Error
	return errors.As(err, &netErr) && netErr.Timeout()
}

func isRetryableStatus(status int) bool {
	return status == http.StatusBadGateway ||
		status == http.StatusServiceUnavailable ||
		status == http.StatusGatewayTimeout
}

func retryJitter() time.Duration {
	if sidecarRetryJitter <= 0 {
		return 0
	}
	maxJitterMs := int(sidecarRetryJitter / time.Millisecond)
	return time.Duration(rand.Intn(maxJitterMs*2+1)-maxJitterMs) * time.Millisecond
}

func decodeResultPayload(payload []byte, target any) (string, error) {
	var envelope responseEnvelope
	if err := json.Unmarshal(payload, &envelope); err == nil && len(envelope.Result) > 0 {
		if err := json.Unmarshal(envelope.Result, target); err != nil {
			return "", err
		}
		return envelope.RawOutput, nil
	}

	if err := json.Unmarshal(payload, target); err != nil {
		return "", err
	}
	return "", nil
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
