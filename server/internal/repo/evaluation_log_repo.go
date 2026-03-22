package repo

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"practicehelper/server/internal/domain"
)

func (s *Store) CreateEvaluationLog(
	ctx context.Context,
	sessionID string,
	turnID string,
	flowName string,
	modelName string,
	promptSetID string,
	promptHash string,
	rawOutput string,
	runtimeTrace *domain.RuntimeTrace,
	latencyMs float64,
) error {
	runtimeTraceJSON, err := marshalRuntimeTrace(runtimeTrace)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `
        INSERT INTO evaluation_logs (
            session_id, turn_id, flow_name, model_name, prompt_set_id,
            prompt_hash, raw_output, runtime_trace_json, latency_ms, created_at
        )
        VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
    `, sessionID, turnID, flowName, modelName, promptSetID, promptHash, rawOutput, runtimeTraceJSON, latencyMs, nowUTC())
	if err != nil {
		return fmt.Errorf("create evaluation log: %w", err)
	}
	return nil
}

func (s *Store) ListEvaluationLogsBySession(
	ctx context.Context,
	sessionID string,
) ([]domain.EvaluationLogEntry, error) {
	rows, err := s.db.QueryContext(ctx, `
        SELECT id, session_id, turn_id, flow_name, model_name, prompt_set_id, prompt_hash,
            raw_output, runtime_trace_json, latency_ms, created_at
        FROM evaluation_logs
        WHERE session_id = ?
        ORDER BY created_at ASC, id ASC
	`, sessionID)
	if err != nil {
		return nil, fmt.Errorf("list evaluation logs: %w", err)
	}
	defer func() { _ = rows.Close() }()

	items := make([]domain.EvaluationLogEntry, 0)
	for rows.Next() {
		item, err := scanEvaluationLogEntry(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *item)
	}
	return items, nil
}

func (s *Store) GetPromptExperimentMetrics(
	ctx context.Context,
	promptSetID string,
	req domain.PromptExperimentRequest,
) (*domain.PromptExperimentMetrics, error) {
	where, args := promptExperimentSessionWhere(promptSetID, req)

	var sessionCount, completedCount int
	var avgTotalScore sql.NullFloat64
	sessionQuery := fmt.Sprintf(`
		SELECT COUNT(*),
			SUM(CASE WHEN status = ? THEN 1 ELSE 0 END),
			AVG(CASE WHEN status = ? THEN total_score END)
		FROM training_sessions
		WHERE %s
	`, where)
	sessionArgs := append([]any{domain.StatusCompleted, domain.StatusCompleted}, args...)
	if err := s.db.QueryRowContext(ctx, sessionQuery, sessionArgs...).Scan(
		&sessionCount,
		&completedCount,
		&avgTotalScore,
	); err != nil {
		return nil, fmt.Errorf("query prompt experiment sessions: %w", err)
	}

	metrics := &domain.PromptExperimentMetrics{
		SessionCount:   sessionCount,
		CompletedCount: completedCount,
	}
	if avgTotalScore.Valid {
		metrics.AvgTotalScore = avgTotalScore.Float64
	}

	latencyWhere, latencyArgs := promptExperimentLatencyWhere(promptSetID, req)
	var (
		avgGenerateQuestion sql.NullFloat64
		avgEvaluateAnswer   sql.NullFloat64
		avgGenerateReview   sql.NullFloat64
	)
	if err := s.db.QueryRowContext(ctx, fmt.Sprintf(`
		SELECT
			AVG(CASE WHEN el.flow_name IN ('generate_question', 'generate_question_stream') THEN el.latency_ms END),
			AVG(CASE WHEN el.flow_name IN ('evaluate_answer', 'evaluate_answer_stream') THEN el.latency_ms END),
			AVG(CASE WHEN el.flow_name IN ('generate_review', 'generate_review_stream') THEN el.latency_ms END)
		FROM evaluation_logs el
		JOIN training_sessions ts ON ts.id = el.session_id
		WHERE %s
	`, latencyWhere), latencyArgs...).Scan(&avgGenerateQuestion, &avgEvaluateAnswer, &avgGenerateReview); err != nil {
		return nil, fmt.Errorf("query prompt experiment latencies: %w", err)
	}

	if avgGenerateQuestion.Valid {
		metrics.AvgGenerateQuestionLatencyMs = avgGenerateQuestion.Float64
	}
	if avgEvaluateAnswer.Valid {
		metrics.AvgEvaluateAnswerLatencyMs = avgEvaluateAnswer.Float64
	}
	if avgGenerateReview.Valid {
		metrics.AvgGenerateReviewLatencyMs = avgGenerateReview.Float64
	}

	return metrics, nil
}

func (s *Store) ListPromptExperimentSamples(
	ctx context.Context,
	req domain.PromptExperimentRequest,
) ([]domain.PromptExperimentSample, error) {
	limit := req.Limit
	if limit <= 0 || limit > 50 {
		limit = 12
	}

	where, args := promptExperimentSamplesWhere(req)
	query := fmt.Sprintf(`
		SELECT id, review_id, mode, topic, status, total_score, updated_at,
			COALESCE(prompt_set_id, ''), COALESCE(prompt_set_label, ''), COALESCE(prompt_set_status, '')
		FROM training_sessions
		WHERE %s
		ORDER BY updated_at DESC
		LIMIT ?
	`, where)
	args = append(args, limit)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list prompt experiment samples: %w", err)
	}
	defer func() { _ = rows.Close() }()

	items := make([]domain.PromptExperimentSample, 0)
	for rows.Next() {
		var (
			sessionID, reviewID, mode, topic, status, updatedAt string
			promptSetID, promptSetLabel, promptSetStatus        string
			totalScore                                          float64
		)
		if err := rows.Scan(
			&sessionID,
			&reviewID,
			&mode,
			&topic,
			&status,
			&totalScore,
			&updatedAt,
			&promptSetID,
			&promptSetLabel,
			&promptSetStatus,
		); err != nil {
			return nil, fmt.Errorf("scan prompt experiment sample: %w", err)
		}

		items = append(items, domain.PromptExperimentSample{
			SessionID:  sessionID,
			ReviewID:   reviewID,
			Mode:       mode,
			Topic:      topic,
			Status:     status,
			TotalScore: totalScore,
			UpdatedAt:  parseTime(updatedAt),
			PromptSet: domain.PromptSetSummary{
				ID:     promptSetID,
				Label:  promptSetLabel,
				Status: promptSetStatus,
			},
		})
	}
	return items, nil
}

func scanEvaluationLogEntry(
	scanner interface{ Scan(dest ...any) error },
) (*domain.EvaluationLogEntry, error) {
	var (
		id                                                  int64
		sessionID, turnID, flowName, modelName, promptSetID string
		promptHash, rawOutput, runtimeTraceJSON, createdAt  string
		latencyMs                                           float64
	)
	if err := scanner.Scan(
		&id,
		&sessionID,
		&turnID,
		&flowName,
		&modelName,
		&promptSetID,
		&promptHash,
		&rawOutput,
		&runtimeTraceJSON,
		&latencyMs,
		&createdAt,
	); err != nil {
		return nil, err
	}

	runtimeTrace, err := unmarshalRuntimeTrace(runtimeTraceJSON)
	if err != nil {
		return nil, err
	}

	return &domain.EvaluationLogEntry{
		ID:           id,
		SessionID:    sessionID,
		TurnID:       turnID,
		FlowName:     flowName,
		ModelName:    modelName,
		PromptSetID:  promptSetID,
		PromptHash:   promptHash,
		RawOutput:    rawOutput,
		RuntimeTrace: runtimeTrace,
		LatencyMs:    latencyMs,
		CreatedAt:    parseTime(createdAt),
	}, nil
}

func marshalRuntimeTrace(trace *domain.RuntimeTrace) (string, error) {
	if trace == nil || len(trace.Entries) == 0 {
		return "null", nil
	}
	payload, err := json.Marshal(trace)
	if err != nil {
		return "", fmt.Errorf("marshal runtime trace: %w", err)
	}
	return string(payload), nil
}

func unmarshalRuntimeTrace(raw string) (*domain.RuntimeTrace, error) {
	if raw == "" || raw == "null" {
		return nil, nil
	}

	var trace domain.RuntimeTrace
	if err := json.Unmarshal([]byte(raw), &trace); err != nil {
		return nil, fmt.Errorf("unmarshal runtime trace: %w", err)
	}
	if len(trace.Entries) == 0 {
		return nil, nil
	}
	return &trace, nil
}

func promptExperimentSessionWhere(
	promptSetID string,
	req domain.PromptExperimentRequest,
) (string, []any) {
	where := "prompt_set_id = ?"
	args := []any{promptSetID}

	if req.Mode != "" {
		where += " AND mode = ?"
		args = append(args, req.Mode)
	}
	if req.Topic != "" {
		where += " AND topic = ?"
		args = append(args, req.Topic)
	}

	return where, args
}

func promptExperimentLatencyWhere(
	promptSetID string,
	req domain.PromptExperimentRequest,
) (string, []any) {
	where := "ts.prompt_set_id = ? AND el.flow_name IN (?, ?, ?, ?, ?, ?)"
	args := []any{
		promptSetID,
		"generate_question",
		"generate_question_stream",
		"evaluate_answer",
		"evaluate_answer_stream",
		"generate_review",
		"generate_review_stream",
	}

	if req.Mode != "" {
		where += " AND ts.mode = ?"
		args = append(args, req.Mode)
	}
	if req.Topic != "" {
		where += " AND ts.topic = ?"
		args = append(args, req.Topic)
	}
	return where, args
}

func promptExperimentSamplesWhere(req domain.PromptExperimentRequest) (string, []any) {
	where := "prompt_set_id IN (?, ?)"
	args := []any{req.Left, req.Right}

	if req.Mode != "" {
		where += " AND mode = ?"
		args = append(args, req.Mode)
	}
	if req.Topic != "" {
		where += " AND topic = ?"
		args = append(args, req.Topic)
	}
	return where, args
}
