package service

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/sollie/matchmark/internal/repository"
	"github.com/sollie/matchmark/internal/wa1500"
)

const (
	shooterSyncLayout       = "2006-01-02T15:04:05Z"
	addEntryRefreshAfter    = 24 * time.Hour
	addEntryRefreshTimeout  = 3 * time.Second
	backgroundRefreshAfter  = 7 * 24 * time.Hour
	backgroundSyncInterval  = time.Hour
	backgroundSyncBatch     = 20
	backgroundSyncPace      = 2 * time.Second
	shooterRefreshBackoff   = 15 * time.Minute
	backgroundRefreshTimout = 15 * time.Second
)

var errNotInRegister = errors.New("shooter not found in the wa1500 register")

type shooterSync struct {
	mu      sync.Mutex
	backoff map[int64]time.Time
}

func WithShooterRegister(r wa1500.Searcher) Option {
	return func(s *Service) { s.register = r }
}

func WithClock(now func() time.Time) Option {
	return func(s *Service) { s.now = now }
}

func WithShooterSyncPace(d time.Duration) Option {
	return func(s *Service) { s.syncPace = d }
}

func (s *Service) RefreshShooter(ctx context.Context, sh repository.Shooter) error {
	if s.register == nil || sh.WAID == "" {
		return nil
	}
	if s.inBackoff(sh.ID) {
		return errors.New("shooter refresh is backing off")
	}
	err := s.refreshShooter(ctx, sh)
	if err != nil {
		s.setBackoff(sh.ID)
		slog.WarnContext(ctx, "wa1500 shooter refresh failed", "shooter_id", sh.ID, "waid", sh.WAID, "error", err)
	}
	return err
}

func (s *Service) refreshShooter(ctx context.Context, sh repository.Shooter) error {
	rows, err := s.register.Search(ctx, sh.WAID)
	if err != nil {
		return err
	}
	var found *wa1500.Shooter
	for i := range rows {
		if rows[i].WAID == sh.WAID {
			found = &rows[i]
			break
		}
	}
	if found == nil {
		return errNotInRegister
	}
	in := repository.ShooterRefresh{
		LastName: NormalizeLastName(found.LastName), FirstName: NormalizeFirstName(found.FirstName),
		Country: strings.TrimSpace(found.Country), Club: strings.TrimSpace(found.Club),
		Classes:  map[string]int{},
		SyncedAt: s.now().UTC().Format(shooterSyncLayout),
	}
	if in.LastName == "" {
		in.LastName = sh.LastName
	}
	if in.FirstName == "" {
		in.FirstName = sh.FirstName
	}
	if in.Country == "" {
		in.Country = sh.Country
	}
	if in.Club == "" {
		in.Club = sh.Club
	}
	for category, class := range found.Classes {
		if class.Valid() {
			in.Classes[category] = int(class)
		}
	}
	return s.repo.RefreshShooter(ctx, sh.ID, in)
}

func (s *Service) refreshStaleShooter(ctx context.Context, sh repository.Shooter) {
	if s.register == nil || sh.WAID == "" || !s.syncedBefore(sh, addEntryRefreshAfter) {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, addEntryRefreshTimeout)
	defer cancel()
	_ = s.RefreshShooter(ctx, sh)
}

func (s *Service) syncedBefore(sh repository.Shooter, age time.Duration) bool {
	if sh.SyncedAt == "" {
		return true
	}
	at, err := time.Parse(shooterSyncLayout, sh.SyncedAt)
	return err != nil || s.now().Sub(at) > age
}

func (s *Service) SyncDueShooters(ctx context.Context) {
	if s.register == nil {
		return
	}
	cutoff := s.now().Add(-backgroundRefreshAfter).UTC().Format(shooterSyncLayout)
	due, err := s.repo.ListShootersDueForSync(ctx, cutoff)
	if err != nil {
		slog.WarnContext(ctx, "wa1500 background sync list failed", "error", err)
		return
	}
	attempts := 0
	for _, sh := range due {
		if ctx.Err() != nil || attempts == backgroundSyncBatch {
			return
		}
		if s.inBackoff(sh.ID) {
			continue
		}
		if attempts > 0 && s.syncPace > 0 {
			select {
			case <-ctx.Done():
				return
			case <-time.After(s.syncPace):
			}
		}
		attempts++
		reqCtx, cancel := context.WithTimeout(ctx, backgroundRefreshTimout)
		_ = s.RefreshShooter(reqCtx, sh)
		cancel()
	}
}

func (s *Service) RunShooterSync(ctx context.Context) {
	if s.register == nil {
		return
	}
	ticker := time.NewTicker(backgroundSyncInterval)
	defer ticker.Stop()
	for {
		s.SyncDueShooters(ctx)
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (s *Service) inBackoff(id int64) bool {
	s.sync.mu.Lock()
	defer s.sync.mu.Unlock()
	until, ok := s.sync.backoff[id]
	if !ok {
		return false
	}
	if !s.now().Before(until) {
		delete(s.sync.backoff, id)
		return false
	}
	return true
}

func (s *Service) setBackoff(id int64) {
	s.sync.mu.Lock()
	defer s.sync.mu.Unlock()
	if s.sync.backoff == nil {
		s.sync.backoff = map[int64]time.Time{}
	}
	s.sync.backoff[id] = s.now().Add(shooterRefreshBackoff)
}
