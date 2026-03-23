package sidecar

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"time"

	"practicehelper/server/internal/domain"
)

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
