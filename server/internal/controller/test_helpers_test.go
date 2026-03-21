package controller

import (
	"bufio"
	"bytes"
	"encoding/json"
	"testing"
)

type dataEnvelope[T any] struct {
	Data T `json:"data"`
}

type streamPayload struct {
	Type    string          `json:"type"`
	Code    string          `json:"code,omitempty"`
	Phase   string          `json:"phase,omitempty"`
	Name    string          `json:"name,omitempty"`
	Text    string          `json:"text,omitempty"`
	Message string          `json:"message,omitempty"`
	Data    json.RawMessage `json:"data,omitempty"`
}

func decodeDataEnvelope[T any](t *testing.T, body []byte) T {
	t.Helper()

	var payload dataEnvelope[T]
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	return payload.Data
}

func decodeStreamEvents(t *testing.T, body []byte) []streamPayload {
	t.Helper()

	scanner := bufio.NewScanner(bytes.NewReader(body))
	events := make([]streamPayload, 0)
	for scanner.Scan() {
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}

		var event streamPayload
		if err := json.Unmarshal(line, &event); err != nil {
			t.Fatalf("json.Unmarshal(stream line) error = %v line=%s", err, line)
		}
		events = append(events, event)
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("scanner.Err() error = %v", err)
	}
	return events
}

func decodeStreamResult[T any](t *testing.T, events []streamPayload) T {
	t.Helper()

	for i := len(events) - 1; i >= 0; i-- {
		if events[i].Type != "result" {
			continue
		}

		var result T
		if err := json.Unmarshal(events[i].Data, &result); err != nil {
			t.Fatalf("json.Unmarshal(result data) error = %v", err)
		}
		return result
	}

	t.Fatal("expected result event")
	var zero T
	return zero
}
