package service

import (
	"context"
	"sync"

	"github.com/sollie/matchmark/internal/repository"
	"github.com/sollie/matchmark/internal/rulebook"
)

func (s *Service) SetVisibility(ctx context.Context, matchID int64, v repository.Visibility) error {
	if err := s.repo.SetVisibility(ctx, matchID, v); err != nil {
		return mapRepoErr(err)
	}
	s.changes.Record(ctx, Change{Audit: AuditEvent{Action: "match.visibility", Target: idTarget("match", matchID)}, MatchID: matchID, Live: LiveMatch})
	return nil
}

func (s *Service) publicMatch(ctx context.Context, id int64) (repository.Match, error) {
	c, err := s.repo.GetMatch(ctx, id)
	if err != nil {
		return repository.Match{}, mapRepoErr(err)
	}
	if MatchStatus(c.Status) == StatusDraft {
		return repository.Match{}, ErrNotFound
	}
	return c, nil
}

func (s *Service) Standings(ctx context.Context, matchID int64) ([]StandingView, error) {
	comp, err := s.publicMatch(ctx, matchID)
	if err != nil {
		return nil, err
	}
	return s.standings(ctx, comp)
}

func (s *Service) standings(ctx context.Context, comp repository.Match) ([]StandingView, error) {
	vis, err := s.repo.GetVisibility(ctx, comp.ID)
	if err != nil {
		return nil, mapRepoErr(err)
	}
	entries, err := s.repo.ListSpectatorEntries(ctx, comp.ID)
	if err != nil {
		return nil, err
	}
	return projectStandings(comp.CurrentStage, vis, entries), nil
}

type SpectatorSnapshot struct {
	CurrentStage int            `json:"current_stage"`
	TotalStages  int            `json:"total_stages"`
	Status       string         `json:"status"`
	Standings    []StandingView `json:"standings"`
}

type cachedSnapshot struct {
	version  uint64
	snapshot SpectatorSnapshot
}

type snapSlot struct {
	mu     sync.Mutex
	cached *cachedSnapshot
}

func (s *Service) snapSlot(matchID int64) *snapSlot {
	s.snapMu.Lock()
	defer s.snapMu.Unlock()
	slot, ok := s.snaps[matchID]
	if !ok {
		slot = &snapSlot{}
		s.snaps[matchID] = slot
	}
	return slot
}

func (s *Service) Subscribe(matchID int64) *Subscription {
	return s.bus.Subscribe(matchID)
}

func (s *Service) CheckPublicStream(ctx context.Context, matchID int64) error {
	_, err := s.publicMatch(ctx, matchID)
	return err
}

func (s *Service) CheckAdminStream(ctx context.Context, matchID int64) error {
	_, err := s.repo.GetMatch(ctx, matchID)
	return mapRepoErr(err)
}

func (s *Service) SpectatorSnapshot(ctx context.Context, matchID int64) (SpectatorSnapshot, error) {
	slot := s.snapSlot(matchID)
	slot.mu.Lock()
	defer slot.mu.Unlock()
	version := s.bus.Version(matchID)
	if c := slot.cached; c != nil && c.version == version {
		return c.snapshot, nil
	}
	comp, err := s.publicMatch(ctx, matchID)
	if err != nil {
		slot.cached = nil
		return SpectatorSnapshot{}, err
	}
	standings, err := s.standings(ctx, comp)
	if err != nil {
		slot.cached = nil
		return SpectatorSnapshot{}, err
	}
	course, err := rulebook.CourseByKey(comp.Course)
	if err != nil {
		slot.cached = nil
		return SpectatorSnapshot{}, err
	}
	snap := SpectatorSnapshot{CurrentStage: comp.CurrentStage, TotalStages: len(course.Stages), Status: comp.Status, Standings: standings}
	slot.cached = &cachedSnapshot{version: version, snapshot: snap}
	return snap, nil
}
