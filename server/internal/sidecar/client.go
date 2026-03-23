package sidecar

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
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
	if _, _, _, _, err := c.postJSON(ctx, "/internal/analyze_repo", request, &response, nil); err != nil {
		return nil, err
	}

	return &response, nil
}

func (c *Client) AnalyzeJobTarget(
	ctx context.Context,
	request domain.AnalyzeJobTargetRequest,
) (*domain.AnalyzeJobTargetResponse, error) {
	var response domain.AnalyzeJobTargetResponse
	if _, _, _, _, err := c.postJSON(ctx, "/internal/analyze_job_target", request, &response, nil); err != nil {
		return nil, err
	}

	return &response, nil
}

func (c *Client) EmbedMemory(
	ctx context.Context,
	request domain.EmbedMemoryRequest,
) (*domain.EmbedMemoryResponse, error) {
	var response domain.EmbedMemoryResponse
	if _, _, _, _, err := c.postJSON(ctx, "/internal/embed_memory", request, &response, nil); err != nil {
		return nil, err
	}
	return &response, nil
}

func (c *Client) RerankMemory(
	ctx context.Context,
	request domain.RerankMemoryRequest,
) (*domain.RerankMemoryResponse, error) {
	var response domain.RerankMemoryResponse
	if _, _, _, _, err := c.postJSON(ctx, "/internal/rerank_memory", request, &response, nil); err != nil {
		return nil, err
	}
	return &response, nil
}

func (c *Client) GenerateQuestion(
	ctx context.Context,
	request domain.GenerateQuestionRequest,
) (*domain.GenerateQuestionResponse, *domain.PromptExecutionMeta, error) {
	var response domain.GenerateQuestionResponse
	headers, rawOutput, runtimeTrace, _, err := c.postJSON(
		ctx,
		"/internal/generate_question",
		request,
		&response,
		nil,
	)
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
	headers, rawOutput, runtimeTrace, _, err := c.postJSONStream(
		ctx,
		"/internal/generate_question/stream",
		request,
		&response,
		nil,
		emit,
	)
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
) (
	*domain.EvaluationResult,
	*domain.EvaluateAnswerSideEffects,
	[]domain.AgentCommandResult,
	*domain.PromptExecutionMeta,
	error,
) {
	var response domain.EvaluationResult
	var sideEffects domain.EvaluateAnswerSideEffects
	headers, rawOutput, runtimeTrace, commandResults, err := c.postJSON(
		ctx,
		"/internal/evaluate_answer",
		request,
		&response,
		&sideEffects,
	)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	meta := promptMetaFromHeaders(headers)
	meta.RawOutput = rawOutput
	meta.RuntimeTrace = runtimeTrace
	return &response, &sideEffects, commandResults, meta, nil
}

func (c *Client) EvaluateAnswerStream(
	ctx context.Context,
	request domain.EvaluateAnswerRequest,
	emit func(domain.StreamEvent) error,
) (
	*domain.EvaluationResult,
	*domain.EvaluateAnswerSideEffects,
	[]domain.AgentCommandResult,
	*domain.PromptExecutionMeta,
	error,
) {
	var response domain.EvaluationResult
	var sideEffects domain.EvaluateAnswerSideEffects
	headers, rawOutput, runtimeTrace, commandResults, err := c.postJSONStream(
		ctx,
		"/internal/evaluate_answer/stream",
		request,
		&response,
		&sideEffects,
		emit,
	)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	meta := promptMetaFromHeaders(headers)
	meta.RawOutput = rawOutput
	meta.RuntimeTrace = runtimeTrace
	return &response, &sideEffects, commandResults, meta, nil
}

func (c *Client) GenerateReview(
	ctx context.Context,
	request domain.GenerateReviewRequest,
) (
	*domain.ReviewCard,
	*domain.GenerateReviewSideEffects,
	[]domain.AgentCommandResult,
	*domain.PromptExecutionMeta,
	error,
) {
	var response domain.ReviewCard
	var sideEffects domain.GenerateReviewSideEffects
	headers, rawOutput, runtimeTrace, commandResults, err := c.postJSON(
		ctx,
		"/internal/generate_review",
		request,
		&response,
		&sideEffects,
	)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	meta := promptMetaFromHeaders(headers)
	meta.RawOutput = rawOutput
	meta.RuntimeTrace = runtimeTrace
	return &response, &sideEffects, commandResults, meta, nil
}

func (c *Client) GenerateReviewStream(
	ctx context.Context,
	request domain.GenerateReviewRequest,
	emit func(domain.StreamEvent) error,
) (
	*domain.ReviewCard,
	*domain.GenerateReviewSideEffects,
	[]domain.AgentCommandResult,
	*domain.PromptExecutionMeta,
	error,
) {
	var response domain.ReviewCard
	var sideEffects domain.GenerateReviewSideEffects
	headers, rawOutput, runtimeTrace, commandResults, err := c.postJSONStream(
		ctx,
		"/internal/generate_review/stream",
		request,
		&response,
		&sideEffects,
		emit,
	)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	meta := promptMetaFromHeaders(headers)
	meta.RawOutput = rawOutput
	meta.RuntimeTrace = runtimeTrace
	return &response, &sideEffects, commandResults, meta, nil
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
) (http.Header, string, *domain.RuntimeTrace, []domain.AgentCommandResult, error) {
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
		slog.ErrorContext(ctx, "sidecar request failed", "path", path, "duration_ms", time.Since(startedAt).Milliseconds(), "error", err)
		return nil, "", nil, nil, fmt.Errorf("call sidecar: %w", err)
	}
	defer func() { _ = response.Body.Close() }()

	if response.StatusCode >= http.StatusBadRequest {
		slog.ErrorContext(ctx, "sidecar request returned error status", "path", path, "status", response.StatusCode, "duration_ms", time.Since(startedAt).Milliseconds())
		payload, _ := io.ReadAll(response.Body)
		return nil, "", nil, nil, decodeAPIError(response.StatusCode, payload)
	}

	payload, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, "", nil, nil, fmt.Errorf("read sidecar response: %w", err)
	}
	rawOutput, runtimeTrace, commandResults, err := decodeResultPayload(payload, target, sideEffectsTarget)
	if err != nil {
		return nil, "", nil, nil, fmt.Errorf("decode sidecar response: %w", err)
	}

	slog.InfoContext(ctx, "sidecar request completed", "path", path, "status", response.StatusCode, "duration_ms", time.Since(startedAt).Milliseconds())

	return response.Header.Clone(), rawOutput, runtimeTrace, commandResults, nil
}
