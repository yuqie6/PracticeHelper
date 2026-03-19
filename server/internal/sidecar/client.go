package sidecar

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"practicehelper/server/internal/domain"
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

func (c *Client) EvaluateAnswer(ctx context.Context, request domain.EvaluateAnswerRequest) (*domain.EvaluationResult, error) {
	var response domain.EvaluationResult
	if err := c.postJSON(ctx, "/internal/evaluate_answer", request, &response); err != nil {
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

	response, err := c.httpClient.Do(request)
	if err != nil {
		return fmt.Errorf("call sidecar: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode >= http.StatusBadRequest {
		return fmt.Errorf("sidecar returned status %d", response.StatusCode)
	}

	if err := json.NewDecoder(response.Body).Decode(target); err != nil {
		return fmt.Errorf("decode sidecar response: %w", err)
	}

	return nil
}
