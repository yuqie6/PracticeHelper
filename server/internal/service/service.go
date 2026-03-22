package service

import (
	"errors"
	"time"

	"practicehelper/server/internal/domain"
	"practicehelper/server/internal/repo"
	"practicehelper/server/internal/sidecar"
	"practicehelper/server/internal/vectorstore"
)

var (
	ErrInvalidMode               = errors.New("invalid mode")
	ErrProjectNotFound           = errors.New("project not found")
	ErrJobTargetNotFound         = errors.New("job target not found")
	ErrJobTargetNotReady         = errors.New("job target is not ready")
	ErrJobTargetAnalysisNotFound = errors.New("job target analysis not found")
	ErrReviewScheduleNotFound    = errors.New("review schedule not found")
	ErrSessionNotFound           = errors.New("session not found")
	ErrPromptSetNotFound         = errors.New("prompt set not found")
	ErrEmptyExportSelection      = errors.New("session ids are required")
	ErrUnsupportedExportFormat   = errors.New("unsupported export format")
	ErrImportJobNotFound         = errors.New("import job not found")
	ErrSessionNotRecoverable     = errors.New("session is not in a recoverable state")
	ErrSessionBusy               = errors.New("session is already processing a previous submission")
	ErrSessionReviewPending      = errors.New("session review is pending; retry review instead")
	ErrSessionCompleted          = errors.New("session is already completed")
	ErrSessionAnswerConflict     = errors.New("session cannot accept answers in its current status")
	ErrReviewGenerationRetry     = errors.New("review generation failed but the session can be recovered by retrying review")
	ErrInvalidAgentCommand       = errors.New("invalid agent command")
	ErrUnsupportedAgentCommand   = errors.New("unsupported agent command")
)

type Service struct {
	repo    *repo.Store
	sidecar *sidecar.Client

	vectorStore                vectorstore.Store
	vectorWriteEnabled         bool
	vectorReadEnabled          bool
	vectorRerankEnabled        bool
	memoryHotIndexTimeout      time.Duration
	memoryEmbeddingClaimTTL    time.Duration
	memoryEmbeddingPollEvery   time.Duration
	memoryEmbeddingWorkerAlive bool
}

type Option func(*Service)

func WithVectorStore(store vectorstore.Store) Option {
	return func(s *Service) {
		s.vectorStore = store
	}
}

func WithVectorRetrievalConfig(
	writeEnabled bool,
	readEnabled bool,
	rerankEnabled bool,
	hotIndexTimeout time.Duration,
	claimTTL time.Duration,
	pollEvery time.Duration,
) Option {
	return func(s *Service) {
		s.vectorWriteEnabled = writeEnabled
		s.vectorReadEnabled = readEnabled
		s.vectorRerankEnabled = rerankEnabled
		if hotIndexTimeout > 0 {
			s.memoryHotIndexTimeout = hotIndexTimeout
		}
		if claimTTL > 0 {
			s.memoryEmbeddingClaimTTL = claimTTL
		}
		if pollEvery > 0 {
			s.memoryEmbeddingPollEvery = pollEvery
		}
	}
}

func New(repository *repo.Store, sc *sidecar.Client, options ...Option) *Service {
	svc := &Service{
		repo:                     repository,
		sidecar:                  sc,
		memoryHotIndexTimeout:    2 * time.Second,
		memoryEmbeddingClaimTTL:  20 * time.Second,
		memoryEmbeddingPollEvery: 4 * time.Second,
	}
	for _, option := range options {
		if option != nil {
			option(svc)
		}
	}
	svc.resumePendingImportJobs()
	svc.startMemoryEmbeddingWorker()
	return svc
}

func emitStatus(emit func(domain.StreamEvent) error, name string) {
	if emit == nil || name == "" {
		return
	}
	_ = emit(domain.StreamEvent{Type: "status", Name: name})
}
