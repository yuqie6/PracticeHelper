package domain

const (
	ModeBasics  = "basics"
	ModeProject = "project"

	BasicsTopicMixed        = "mixed"
	BasicsTopicGo           = "go"
	BasicsTopicRedis        = "redis"
	BasicsTopicKafka        = "kafka"
	BasicsTopicMySQL        = "mysql"
	BasicsTopicSystemDesign = "system_design"
	BasicsTopicDistributed  = "distributed"
	BasicsTopicNetwork      = "network"
	BasicsTopicMicroservice = "microservice"
	BasicsTopicOS           = "os"
	BasicsTopicDockerK8s    = "docker_k8s"

	StatusDraft         = "draft"
	StatusActive        = "active"
	StatusWaitingAnswer = "waiting_answer"
	StatusEvaluating    = "evaluating"
	StatusReviewPending = "review_pending"
	StatusCompleted     = "completed"

	ProjectImportStatusQueued    = "queued"
	ProjectImportStatusRunning   = "running"
	ProjectImportStatusCompleted = "completed"
	ProjectImportStatusFailed    = "failed"

	JobTargetAnalysisIdle      = "idle"
	JobTargetAnalysisRunning   = "running"
	JobTargetAnalysisSucceeded = "succeeded"
	JobTargetAnalysisFailed    = "failed"
	JobTargetAnalysisStale     = "stale"

	ProjectImportStageQueued     = "queued"
	ProjectImportStageAnalyzing  = "analyzing_repository"
	ProjectImportStagePersisting = "persisting_project"
	ProjectImportStageCompleted  = "completed"
	ProjectImportStageFailed     = "failed"

	MemoryScopeGlobal    = "global"
	MemoryScopeProject   = "project"
	MemoryScopeSession   = "session"
	MemoryScopeJobTarget = "job_target"

	KnowledgeNodeTypeTopic   = "topic"
	KnowledgeNodeTypeConcept = "concept"
	KnowledgeNodeTypeSkill   = "skill"

	KnowledgeEdgeContains     = "contains"
	KnowledgeEdgePrerequisite = "prerequisite"
	KnowledgeEdgeRelated      = "related"

	ObservationCategoryPattern       = "pattern"
	ObservationCategoryMisconception = "misconception"
	ObservationCategoryGrowth        = "growth"
	ObservationCategoryStrategyNote  = "strategy_note"

	MemoryTypeObservation    = "observation"
	MemoryTypeSessionSummary = "session_summary"

	MemoryEmbeddingStatusPending = "pending"
	MemoryEmbeddingStatusIndexed = "indexed"
	MemoryEmbeddingStatusFailed  = "failed"

	MemoryEmbeddingJobStatusQueued  = "queued"
	MemoryEmbeddingJobStatusRunning = "running"
	MemoryEmbeddingJobStatusFailed  = "failed"

	DepthSignalNormal       = "normal"
	DepthSignalSkipFollowup = "skip_followup"
	DepthSignalExtend       = "extend"
)
