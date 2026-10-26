package service

import (
	"context"
	"sync"

	"github.com/sollie/matchmark/internal/repository"
	"github.com/sollie/matchmark/internal/rulebook"
)

func (s *Service) SetVisibility(ctx context.Context, competitionID int64, v repository.Visibility) error {
	if err := s.repo.SetVisibility(ctx, competitionID, v); err != nil {
		return mapRepoErr(err)
	}
	s.changes.Record(ctx, Change{Audit: AuditEvent{Action: "competition.visibility", Target: idTarget("competition", competitionID)}, CompetitionID: competitionID, Live: LiveCompetition})
	return nil
}

func (s *Service) publicCompetition(ctx context.Context, id int64) (repository.Competition, error) {
	c, err := s.repo.GetCompetition(ctx, id)
	if err != nil {
		return repository.Competition{}, mapRepoErr(err)
	}
	if CompetitionStatus(c.Status) == StatusDraft {
		return repository.Competition{}, ErrNotFound
	}
	return c, nil
}

func (s *Service) Standings(ctx context.Context, competitionID int64) ([]StandingView, error) {
	comp, err := s.publicCompetition(ctx, competitionID)
	if err != nil {
		return nil, err
	}
	return s.standings(ctx, comp)
}

func (s *Service) standings(ctx context.Context, comp repository.Competition) ([]StandingView, error) {
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

func (s *Service) snapSlot(competitionID int64) *snapSlot {
	s.snapMu.Lock()
	defer s.snapMu.Unlock()
	slot, ok := s.snaps[competitionID]
	if !ok {
		slot = &snapSlot{}
		s.snaps[competitionID] = slot
	}
	return slot
}

func (s *Service) Subscribe(competitionID int64) *Subscription {
	return s.bus.Subscribe(competitionID)
}

func (s *Service) CheckPublicStream(ctx context.Context, competitionID int64) error {
	_, err := s.publicCompetition(ctx, competitionID)
	return err
}

func (s *Service) CheckAdminStream(ctx context.Context, competitionID int64) error {
	_, err := s.repo.GetCompetition(ctx, competitionID)
	return mapRepoErr(err)
}

func (s *Service) SpectatorSnapshot(ctx context.Context, competitionID int64) (SpectatorSnapshot, error) {
	slot := s.snapSlot(competitionID)
	slot.mu.Lock()
	defer slot.mu.Unlock()
	version := s.bus.Version(competitionID)
	if c := slot.cached; c != nil && c.version == version {
		return c.snapshot, nil
	}
	comp, err := s.publicCompetition(ctx, competitionID)
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
