package repo

import (
	"context"
	"fmt"
)

func (s *Store) RecordEvaluationLog(
	ctx context.Context,
	sessionID string,
	turnID string,
	flowName string,
	modelName string,
	latencyMs float64,
) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO evaluation_logs (session_id, turn_id, flow_name, model_name, latency_ms, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, sessionID, turnID, flowName, modelName, latencyMs, nowUTC())
	if err != nil {
		return fmt.Errorf("record evaluation log: %w", err)
	}
	return nil
}
