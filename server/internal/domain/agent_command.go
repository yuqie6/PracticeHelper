package domain

const (
	AgentCommandTypeTransitionSession = "transition_session"
	AgentCommandTypeUpsertReviewPath  = "upsert_review_path"

	AgentCommandStatusAccepted = "accepted"
	AgentCommandStatusRejected = "rejected"
	AgentCommandStatusApplied  = "applied"
	AgentCommandStatusDeferred = "deferred"
)

type AgentCommandEnvelope struct {
	CommandID      string         `json:"command_id"`
	CommandType    string         `json:"command_type"`
	SessionID      string         `json:"session_id,omitempty"`
	IdempotencyKey string         `json:"idempotency_key"`
	Reason         string         `json:"reason,omitempty"`
	Payload        map[string]any `json:"payload,omitempty"`
}

type AgentCommandResult struct {
	CommandID    string         `json:"command_id"`
	CommandType  string         `json:"command_type,omitempty"`
	Status       string         `json:"status"`
	Applied      bool           `json:"applied,omitempty"`
	Data         map[string]any `json:"data,omitempty"`
	ErrorCode    string         `json:"error_code,omitempty"`
	ErrorMessage string         `json:"error_message,omitempty"`
}
