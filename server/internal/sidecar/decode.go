package sidecar

import (
	"encoding/json"

	"practicehelper/server/internal/domain"
)

type responseEnvelope struct {
	Result         json.RawMessage `json:"result"`
	SideEffects    json.RawMessage `json:"side_effects,omitempty"`
	CommandResults json.RawMessage `json:"command_results,omitempty"`
	RawOutput      string          `json:"raw_output,omitempty"`
	Trace          json.RawMessage `json:"trace,omitempty"`
}

func decodeResultPayload(
	payload []byte,
	target any,
	sideEffectsTarget any,
) (string, *domain.RuntimeTrace, []domain.AgentCommandResult, error) {
	var envelope responseEnvelope
	if err := json.Unmarshal(payload, &envelope); err == nil && len(envelope.Result) > 0 {
		if err := json.Unmarshal(envelope.Result, target); err != nil {
			return "", nil, nil, err
		}
		if sideEffectsTarget != nil && len(envelope.SideEffects) > 0 {
			if err := json.Unmarshal(envelope.SideEffects, sideEffectsTarget); err != nil {
				return "", nil, nil, err
			}
		}
		runtimeTrace, err := decodeRuntimeTrace(envelope.Trace)
		if err != nil {
			return "", nil, nil, err
		}
		commandResults, err := decodeCommandResults(envelope.CommandResults)
		if err != nil {
			return "", nil, nil, err
		}
		return envelope.RawOutput, runtimeTrace, commandResults, nil
	}

	if err := json.Unmarshal(payload, target); err != nil {
		return "", nil, nil, err
	}
	return "", nil, nil, nil
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

func decodeCommandResults(raw json.RawMessage) ([]domain.AgentCommandResult, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, nil
	}
	var items []domain.AgentCommandResult
	if err := json.Unmarshal(raw, &items); err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, nil
	}
	return items, nil
}
