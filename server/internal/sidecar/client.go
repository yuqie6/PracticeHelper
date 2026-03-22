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

type APIError struct {
	Code    string
	Message string
	Status  int
}

func (e *APIError) Error() string {
	if e == nil {
		return ""
	}
	return e.Message
}

func (e *APIError) ErrorCode() string {
	if e == nil {
		return ""
	}
	return e.Code
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
	Code    string          `json:"code,omitempty"`
	Phase   string          `json:"phase,omitempty"`
	Name    string          `json:"name,omitempty"`
	Text    string          `json:"text,omitempty"`
	Message string          `json:"message,omitempty"`
	Data    json.RawMessage `json:"data,omitempty"`
}

type responseEnvelope struct {
	Result      json.RawMessage `json:"result"`
	SideEffects json.RawMessage `json:"side_effects,omitempty"`
	RawOutput   string          `json:"raw_output,omitempty"`
	Trace       json.RawMessage `json:"trace,omitempty"`
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
	if _, _, _, err := c.postJSON(ctx, "/internal/analyze_repo", request, &response, nil); err != nil {
		return nil, err
	}

	return &response, nil
}

func (c *Client) AnalyzeJobTarget(
	ctx context.Context,
	request domain.AnalyzeJobTargetRequest,
) (*domain.AnalyzeJobTargetResponse, error) {
	var response domain.AnalyzeJobTargetResponse
	if _, _, _, err := c.postJSON(ctx, "/internal/analyze_job_target", request, &response, nil); err != nil {
		return nil, err
	}

	return &response, nil
}

func (c *Client) EmbedMemory(
	ctx context.Context,
	request domain.EmbedMemoryRequest,
) (*domain.EmbedMemoryResponse, error) {
	var response domain.EmbedMemoryResponse
	if _, _, _, err := c.postJSON(ctx, "/internal/embed_memory", request, &response, nil); err != nil {
		return nil, err
	}
	return &response, nil
}

func (c *Client) RerankMemory(
	ctx context.Context,
	request domain.RerankMemoryRequest,
) (*domain.RerankMemoryResponse, error) {
	var response domain.RerankMemoryResponse
	if _, _, _, err := c.postJSON(ctx, "/internal/rerank_memory", request, &response, nil); err != nil {
		return nil, err
	}
	return &response, nil
}

func (c *Client) GenerateQuestion(
	ctx context.Context,
	request domain.GenerateQuestionRequest,
) (*domain.GenerateQuestionResponse, *domain.PromptExecutionMeta, error) {
	var response domain.GenerateQuestionResponse
	headers, rawOutput, runtimeTrace, err := c.postJSON(ctx, "/internal/generate_question", request, &response, nil)
	if err != nil {
		return nil, nil, err
	}

	meta := promptMetaFromHeaders(headers)
	meta.RawOutput = rawOutput
	meta.RuntimeTrace = runtimeTrace
	return &response, meta, nil
}

func (c *Client) GenerateQuestionStream(
	ctx context.Context,
	request domain.GenerateQuestionRequest,
	emit func(domain.StreamEvent) error,
) (*domain.GenerateQuestionResponse, *domain.PromptExecutionMeta, error) {
	var response domain.GenerateQuestionResponse
	headers, rawOutput, runtimeTrace, err := c.postJSONStream(ctx, "/internal/generate_question/stream", request, &response, nil, emit)
	if err != nil {
		return nil, nil, err
	}

	meta := promptMetaFromHeaders(headers)
	meta.RawOutput = rawOutput
	meta.RuntimeTrace = runtimeTrace
	return &response, meta, nil
}

func (c *Client) EvaluateAnswer(
	ctx context.Context,
	request domain.EvaluateAnswerRequest,
) (*domain.EvaluationResult, *domain.EvaluateAnswerSideEffects, *domain.PromptExecutionMeta, error) {
	var response domain.EvaluationResult
	var sideEffects domain.EvaluateAnswerSideEffects
	headers, rawOutput, runtimeTrace, err := c.postJSON(ctx, "/internal/evaluate_answer", request, &response, &sideEffects)
	if err != nil {
		return nil, nil, nil, err
	}

	meta := promptMetaFromHeaders(headers)
	meta.RawOutput = rawOutput
	meta.RuntimeTrace = runtimeTrace
	return &response, &sideEffects, meta, nil
}

func (c *Client) EvaluateAnswerStream(
	ctx context.Context,
	request domain.EvaluateAnswerRequest,
	emit func(domain.StreamEvent) error,
) (*domain.EvaluationResult, *domain.EvaluateAnswerSideEffects, *domain.PromptExecutionMeta, error) {
	var response domain.EvaluationResult
	var sideEffects domain.EvaluateAnswerSideEffects
	headers, rawOutput, runtimeTrace, err := c.postJSONStream(
		ctx,
		"/internal/evaluate_answer/stream",
		request,
		&response,
		&sideEffects,
		emit,
	)
	if err != nil {
		return nil, nil, nil, err
	}

	meta := promptMetaFromHeaders(headers)
	meta.RawOutput = rawOutput
	meta.RuntimeTrace = runtimeTrace
	return &response, &sideEffects, meta, nil
}

func (c *Client) GenerateReview(
	ctx context.Context,
	request domain.GenerateReviewRequest,
) (*domain.ReviewCard, *domain.GenerateReviewSideEffects, *domain.PromptExecutionMeta, error) {
	var response domain.ReviewCard
	var sideEffects domain.GenerateReviewSideEffects
	headers, rawOutput, runtimeTrace, err := c.postJSON(ctx, "/internal/generate_review", request, &response, &sideEffects)
	if err != nil {
		return nil, nil, nil, err
	}

	meta := promptMetaFromHeaders(headers)
	meta.RawOutput = rawOutput
	meta.RuntimeTrace = runtimeTrace
	return &response, &sideEffects, meta, nil
}

func (c *Client) GenerateReviewStream(
	ctx context.Context,
	request domain.GenerateReviewRequest,
	emit func(domain.StreamEvent) error,
) (*domain.ReviewCard, *domain.GenerateReviewSideEffects, *domain.PromptExecutionMeta, error) {
	var response domain.ReviewCard
	var sideEffects domain.GenerateReviewSideEffects
	headers, rawOutput, runtimeTrace, err := c.postJSONStream(
		ctx,
		"/internal/generate_review/stream",
		request,
		&response,
		&sideEffects,
		emit,
	)
	if err != nil {
		return nil, nil, nil, err
	}

	meta := promptMetaFromHeaders(headers)
	meta.RawOutput = rawOutput
	meta.RuntimeTrace = runtimeTrace
	return &response, &sideEffects, meta, nil
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
	sideEffectsTarget any,
) (http.Header, string, *domain.RuntimeTrace, error) {
	body, err := json.Marshal(requestBody)
	if err != nil {
		return nil, "", nil, fmt.Errorf("marshal sidecar request: %w", err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return nil, "", nil, fmt.Errorf("build sidecar request: %w", err)
	}

	request.Header.Set("Content-Type", "application/json")
	if requestID := observability.RequestIDFromContext(ctx); requestID != "" {
		request.Header.Set(observability.RequestIDHeader, requestID)
	}

	startedAt := time.Now()
	response, err := c.doWithRetry(request, body)
	if err != nil {
		slog.ErrorContext(ctx, "sidecar request failed", "path", path, "duration_ms", time.Since(startedAt).Milliseconds(), "error", err)
		return nil, "", nil, fmt.Errorf("call sidecar: %w", err)
	}
	defer func() { _ = response.Body.Close() }()

	if response.StatusCode >= http.StatusBadRequest {
		slog.ErrorContext(ctx, "sidecar request returned error status", "path", path, "status", response.StatusCode, "duration_ms", time.Since(startedAt).Milliseconds())
		payload, _ := io.ReadAll(response.Body)
		return nil, "", nil, decodeAPIError(response.StatusCode, payload)
	}

	payload, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, "", nil, fmt.Errorf("read sidecar response: %w", err)
	}
	rawOutput, runtimeTrace, err := decodeResultPayload(payload, target, sideEffectsTarget)
	if err != nil {
		return nil, "", nil, fmt.Errorf("decode sidecar response: %w", err)
	}

	slog.InfoContext(ctx, "sidecar request completed", "path", path, "status", response.StatusCode, "duration_ms", time.Since(startedAt).Milliseconds())

	return response.Header.Clone(), rawOutput, runtimeTrace, nil
}

func (c *Client) postJSONStream(
	ctx context.Context,
	path string,
	requestBody any,
	target any,
	sideEffectsTarget any,
	emit func(domain.StreamEvent) error,
) (http.Header, string, *domain.RuntimeTrace, error) {
	var rawOutput string
	var runtimeTrace *domain.RuntimeTrace
	body, err := json.Marshal(requestBody)
	if err != nil {
		return nil, "", nil, fmt.Errorf("marshal sidecar request: %w", err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return nil, "", nil, fmt.Errorf("build sidecar request: %w", err)
	}

	request.Header.Set("Content-Type", "application/json")
	if requestID := observability.RequestIDFromContext(ctx); requestID != "" {
		request.Header.Set(observability.RequestIDHeader, requestID)
	}

	startedAt := time.Now()
	response, err := c.doWithRetry(request, body)
	if err != nil {
		slog.ErrorContext(ctx, "sidecar stream request failed", "path", path, "duration_ms", time.Since(startedAt).Milliseconds(), "error", err)
		return nil, "", nil, fmt.Errorf("call sidecar stream: %w", err)
	}
	defer func() { _ = response.Body.Close() }()

	if response.StatusCode >= http.StatusBadRequest {
		slog.ErrorContext(ctx, "sidecar stream returned error status", "path", path, "status", response.StatusCode, "duration_ms", time.Since(startedAt).Milliseconds())
		payload, _ := io.ReadAll(response.Body)
		return nil, "", nil, decodeAPIError(response.StatusCode, payload)
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
			return nil, "", nil, fmt.Errorf("decode sidecar stream event: %w", err)
		}

		if event.Type == "error" {
			return nil, "", nil, &APIError{
				Code:    event.Code,
				Message: event.Message,
				Status:  response.StatusCode,
			}
		}

		if event.Type == "result" {
			var err error
			rawOutput, runtimeTrace, err = decodeResultPayload(event.Data, target, sideEffectsTarget)
			if err != nil {
				return nil, "", nil, fmt.Errorf("decode sidecar stream result: %w", err)
			}
			continue
		}

		if emit != nil {
			data, err := decodeStreamEventData(event.Data)
			if err != nil {
				return nil, "", nil, fmt.Errorf("decode sidecar stream event data: %w", err)
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
				return nil, "", nil, err
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, "", nil, fmt.Errorf("read sidecar stream: %w", err)
	}

	slog.InfoContext(ctx, "sidecar stream completed", "path", path, "status", response.StatusCode, "duration_ms", time.Since(startedAt).Milliseconds())
	return headers, rawOutput, runtimeTrace, nil
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

func decodeResultPayload(payload []byte, target any, sideEffectsTarget any) (string, *domain.RuntimeTrace, error) {
	var envelope responseEnvelope
	if err := json.Unmarshal(payload, &envelope); err == nil && len(envelope.Result) > 0 {
		if err := json.Unmarshal(envelope.Result, target); err != nil {
			return "", nil, err
		}
		if sideEffectsTarget != nil && len(envelope.SideEffects) > 0 {
			if err := json.Unmarshal(envelope.SideEffects, sideEffectsTarget); err != nil {
				return "", nil, err
			}
		}
		runtimeTrace, err := decodeRuntimeTrace(envelope.Trace)
		if err != nil {
			return "", nil, err
		}
		return envelope.RawOutput, runtimeTrace, nil
	}

	if err := json.Unmarshal(payload, target); err != nil {
		return "", nil, err
	}
	return "", nil, nil
}

func decodeRuntimeTrace(raw json.RawMessage) (*domain.RuntimeTrace, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, nil
	}
	var trace domain.RuntimeTrace
	if err := json.Unmarshal(raw, &trace); err != nil {
		return nil, err
	}
	if len(trace.Entries) == 0 {
		return nil, nil
	}
	return &trace, nil
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

func decodeAPIError(status int, payload []byte) error {
	var body struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(payload, &body); err == nil && body.Error.Message != "" {
		return &APIError{
			Code:    body.Error.Code,
			Message: body.Error.Message,
			Status:  status,
		}
	}
	return &APIError{
		Code:    "unknown_error",
		Message: fmt.Sprintf("sidecar returned status %d", status),
		Status:  status,
	}
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
