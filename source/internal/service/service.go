package service

import (
	"errors"
	"log/slog"
	"sync"

	"github.com/sollie/matchmark/internal/repository"
)

var (
	ErrNotFound           = errors.New("not found")
	ErrConflict           = errors.New("conflict")
	ErrInvalid            = errors.New("invalid input")
	ErrShooterHasEntries  = errors.New("shooter has match entries and cannot be deleted")
	ErrShooterArchived    = errors.New("shooter_archived")
	ErrGone               = errors.New("gone")
	ErrLaneNotManned      = errors.New("lane is not manned")
	ErrAdminScoringNeeded = errors.New("admin scoring required")
	ErrStageLocked        = errors.New("stage is locked")

	ErrStageNotAwaitingAcceptance = errors.New("stage is not awaiting acceptance")
	ErrStageNotLocked             = errors.New("stage is not locked")
	ErrNotCurrentStage            = errors.New("not the current stage")
	ErrLanesOpen                  = errors.New("a lane is still open")
	ErrStageGateNotMet            = errors.New("every manned lane must accept the current stage first")
)

type Service struct {
	repo           *repository.Repository
	replicator     ReplicationRequester
	scoreLogger    *slog.Logger
	audit          *Audit
	bus            *Broadcaster
	changes        *ChangeRecorder
	presence       presence
	screenThrottle *LoginThrottle
	snapMu         sync.Mutex
	snaps          map[int64]*snapSlot
}

func New(repo *repository.Repository, opts ...Option) *Service {
	s := &Service{repo: repo, bus: NewBroadcaster(defaultCoalesceWindow), snaps: map[int64]*snapSlot{}, screenThrottle: NewLoginThrottle()}
	for _, opt := range opts {
		opt(s)
	}
	s.changes = NewChangeRecorder(s.audit, NewScoreLog(repo, s.scoreLogger), s.bus)
	return s
}

func mapRepoErr(err error) error {
	switch {
	case errors.Is(err, repository.ErrNotFound):
		return ErrNotFound
	case errors.Is(err, repository.ErrConflict):
		return ErrConflict
	}
	return err
}
