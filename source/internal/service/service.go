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
	ErrLaneNotManned      = errors.New("lane is not manned")
	ErrAdminScoringNeeded = errors.New("admin scoring required")
	ErrStageLocked        = errors.New("stage is locked")
)

type Service struct {
	repo       *repository.Repository
	replicator ReplicationRequester
	scoreLog   *slog.Logger
	audit      *Audit
	emitMu     sync.Mutex
	emitSeq    int64
	bus        *Broadcaster
	snapMu     sync.Mutex
	snaps      map[int64]*snapSlot
}

func New(repo *repository.Repository, opts ...Option) *Service {
	s := &Service{repo: repo, bus: NewBroadcaster(defaultCoalesceWindow), snaps: map[int64]*snapSlot{}}
	for _, opt := range opts {
		opt(s)
	}
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
