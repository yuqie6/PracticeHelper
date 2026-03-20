package service

import (
	"errors"

	"practicehelper/server/internal/repo"
	"practicehelper/server/internal/sidecar"
)

var (
	ErrInvalidMode           = errors.New("invalid mode")
	ErrProjectNotFound       = errors.New("project not found")
	ErrSessionNotFound       = errors.New("session not found")
	ErrImportJobNotFound     = errors.New("import job not found")
	ErrSessionNotRecoverable = errors.New("session is not in a recoverable state")
	ErrSessionBusy           = errors.New("session is already processing a previous submission")
	ErrSessionReviewPending  = errors.New("session review is pending; retry review instead")
	ErrSessionCompleted      = errors.New("session is already completed")
	ErrSessionAnswerConflict = errors.New("session cannot accept answers in its current status")
)

type Service struct {
	repo    *repo.Store
	sidecar *sidecar.Client
}

func New(repository *repo.Store, sc *sidecar.Client) *Service {
	svc := &Service{
		repo:    repository,
		sidecar: sc,
	}
	svc.resumePendingImportJobs()
	return svc
}
