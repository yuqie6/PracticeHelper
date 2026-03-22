package domain

type EvaluateAnswerSideEffects struct {
	Observations     []AgentObservation `json:"observations,omitempty"`
	KnowledgeUpdates []KnowledgeUpdate  `json:"knowledge_updates,omitempty"`
	DepthSignal      string             `json:"depth_signal,omitempty"`
}

type GenerateReviewSideEffects struct {
	Observations     []AgentObservation `json:"observations,omitempty"`
	KnowledgeUpdates []KnowledgeUpdate  `json:"knowledge_updates,omitempty"`
	RecommendedNext  *NextSession       `json:"recommended_next,omitempty"`
}

type AgentSessionDetail struct {
	Session *TrainingSession `json:"session"`
	Review  *ReviewCard      `json:"review,omitempty"`
}

type GenerateQuestionRequest struct {
	Mode              string                    `json:"mode"`
	Topic             string                    `json:"topic,omitempty"`
	CandidateTopics   []string                  `json:"candidate_topics,omitempty"`
	PromptSetID       string                    `json:"prompt_set_id,omitempty"`
	Intensity         string                    `json:"intensity"`
	Project           *ProjectProfile           `json:"project,omitempty"`
	Templates         []QuestionTemplate        `json:"templates,omitempty"`
	ContextChunks     []RepoChunk               `json:"context_chunks,omitempty"`
	Weaknesses        []WeaknessTag             `json:"weaknesses,omitempty"`
	JobTargetAnalysis *AnalyzeJobTargetResponse `json:"job_target_analysis,omitempty"`
	AgentContext      *AgentContext             `json:"agent_context,omitempty"`
}

type GenerateQuestionResponse struct {
	Question       string   `json:"question"`
	ExpectedPoints []string `json:"expected_points"`
}

type EvaluateAnswerRequest struct {
	SessionID         string                    `json:"session_id,omitempty"`
	Mode              string                    `json:"mode"`
	Topic             string                    `json:"topic,omitempty"`
	PromptSetID       string                    `json:"prompt_set_id,omitempty"`
	Project           *ProjectProfile           `json:"project,omitempty"`
	Question          string                    `json:"question"`
	ExpectedPoints    []string                  `json:"expected_points"`
	Answer            string                    `json:"answer"`
	ContextChunks     []RepoChunk               `json:"context_chunks,omitempty"`
	TurnIndex         int                       `json:"turn_index"`
	MaxTurns          int                       `json:"max_turns"`
	ScoreWeights      map[string]float64        `json:"score_weights,omitempty"`
	JobTargetAnalysis *AnalyzeJobTargetResponse `json:"job_target_analysis,omitempty"`
	AgentContext      *AgentContext             `json:"agent_context,omitempty"`
}

type GenerateReviewRequest struct {
	Session           *TrainingSession          `json:"session"`
	Project           *ProjectProfile           `json:"project,omitempty"`
	Turns             []TrainingTurn            `json:"turns"`
	PromptSetID       string                    `json:"prompt_set_id,omitempty"`
	JobTargetAnalysis *AnalyzeJobTargetResponse `json:"job_target_analysis,omitempty"`
	AgentContext      *AgentContext             `json:"agent_context,omitempty"`
}
