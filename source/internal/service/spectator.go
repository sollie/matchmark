package service

import (
	"context"
	"sort"
	"sync"

	"github.com/sollie/matchmark/internal/repository"
	"github.com/sollie/matchmark/internal/rulebook"
)

type RingCountsView struct {
	X     int `json:"x"`
	Ten   int `json:"10"`
	Nine  int `json:"9"`
	Eight int `json:"8"`
	Seven int `json:"7"`
	Zero  int `json:"0"`
	Miss  int `json:"miss"`
}

type StageScoreView struct {
	Stage    int             `json:"stage"`
	Score    *int            `json:"score,omitempty"`
	Accepted bool            `json:"accepted"`
	State    StageState      `json:"state"`
	Counts   *RingCountsView `json:"ring_counts,omitempty"`
}

type StageState string

const (
	StageNotStarted         StageState = "not_started"
	StageInProgress         StageState = "in_progress"
	StageAwaitingAcceptance StageState = "awaiting_acceptance"
	StageAccepted           StageState = "accepted"
)

func stageState(st repository.StageStatus, number, currentStage int) StageState {
	switch st {
	case repository.StatusAccepted:
		return StageAccepted
	case repository.StatusAwaitingAcceptance:
		return StageAwaitingAcceptance
	}
	if number > currentStage {
		return StageNotStarted
	}
	return StageInProgress
}

type StandingView struct {
	Lane       int              `json:"lane"`
	Category   string           `json:"category"`
	Withdrawn  bool             `json:"withdrawn"`
	Total      int              `json:"total"`
	LastName   *string          `json:"last_name,omitempty"`
	FirstName  *string          `json:"first_name,omitempty"`
	Club       *string          `json:"club,omitempty"`
	Country    *string          `json:"country,omitempty"`
	XCount     *int             `json:"x_count,omitempty"`
	RingCounts *RingCountsView  `json:"ring_counts,omitempty"`
	Stages     []StageScoreView `json:"stages,omitempty"`
}

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

func ringCountsView(c repository.RingCounts) RingCountsView {
	return RingCountsView{X: c.X, Ten: c.Ten, Nine: c.Nine, Eight: c.Eight, Seven: c.Seven, Zero: c.Zero, Miss: c.Miss}
}

func (s *Service) Standings(ctx context.Context, competitionID int64) ([]StandingView, error) {
	comp, err := s.publicCompetition(ctx, competitionID)
	if err != nil {
		return nil, err
	}
	return s.standings(ctx, comp)
}

func (s *Service) standings(ctx context.Context, comp repository.Competition) ([]StandingView, error) {
	competitionID := comp.ID
	vis, err := s.repo.GetVisibility(ctx, competitionID)
	if err != nil {
		return nil, mapRepoErr(err)
	}
	entries, err := s.repo.ListSpectatorEntries(ctx, competitionID)
	if err != nil {
		return nil, err
	}
	type ranked struct {
		view StandingView
		x    int
	}
	rows := make([]ranked, 0, len(entries))
	for _, e := range entries {
		var total repository.RingCounts
		v := StandingView{Lane: e.Lane, Category: e.Category, Withdrawn: e.Withdrawn}
		for _, st := range e.Stages {
			accepted := st.Status == repository.StatusAccepted
			if !accepted {
				st.Counts = repository.RingCounts{}
			}
			v.Total += st.Counts.Score()
			total.X += st.Counts.X
			total.Ten += st.Counts.Ten
			total.Nine += st.Counts.Nine
			total.Eight += st.Counts.Eight
			total.Seven += st.Counts.Seven
			total.Zero += st.Counts.Zero
			total.Miss += st.Counts.Miss
			sv := StageScoreView{Stage: st.Number, Accepted: accepted, State: stageState(st.Status, st.Number, comp.CurrentStage)}
			if vis.StageScores && accepted {
				score := st.Counts.Score()
				sv.Score = &score
				if vis.RingCounts {
					c := ringCountsView(st.Counts)
					sv.Counts = &c
				}
			}
			v.Stages = append(v.Stages, sv)
		}
		if vis.Names {
			v.LastName, v.FirstName = &e.LastName, &e.FirstName
		}
		if vis.Clubs {
			v.Club = &e.Club
		}
		if vis.Countries {
			v.Country = &e.Country
		}
		if vis.RingCounts {
			c := ringCountsView(total)
			x := total.X
			v.RingCounts, v.XCount = &c, &x
		}
		rows = append(rows, ranked{view: v, x: total.X})
	}
	sort.SliceStable(rows, func(i, j int) bool {
		if rows[i].view.Total != rows[j].view.Total {
			return rows[i].view.Total > rows[j].view.Total
		}
		return rows[i].x > rows[j].x
	})
	out := make([]StandingView, len(rows))
	for i, r := range rows {
		out[i] = r.view
	}
	return out, nil
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
