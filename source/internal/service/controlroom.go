package service

import (
	"context"
	"errors"
	"slices"
	"strconv"
	"strings"

	"github.com/sollie/matchmark/internal/repository"
)

type LaneStatus struct {
	Lane        int
	EntryID     int64
	ShooterName string
	ScorerName  string
	Status      repository.StageStatus
	Disputed    bool
	Disputes    int
	Paired      bool
	Connected   bool
}

type ControlRoom struct {
	CurrentStage int
	CanStartNext bool
	CanAcceptAll bool
	Lanes        []LaneStatus
}

func (s *Service) ControlRoom(ctx context.Context, matchID int64) (ControlRoom, error) {
	comp, err := s.repo.GetMatch(ctx, matchID)
	if err != nil {
		return ControlRoom{}, mapRepoErr(err)
	}
	if MatchStatus(comp.Status) == StatusDraft {
		return ControlRoom{}, ErrConflict
	}
	entries, err := s.repo.ListEntries(ctx, matchID)
	if err != nil {
		return ControlRoom{}, err
	}
	pairings, err := s.repo.ListPairings(ctx, matchID)
	if err != nil {
		return ControlRoom{}, err
	}
	events, err := s.repo.ListStageEvents(ctx, matchID)
	if err != nil {
		return ControlRoom{}, err
	}
	disputes := map[int64]int{}
	for _, ev := range events {
		if ev.Event == repository.EventDisputed {
			disputes[ev.EntryID]++
		}
	}
	paired := map[int]bool{}
	for _, p := range pairings {
		if !p.Revoked {
			paired[p.Lane] = true
		}
	}

	manned := make([]repository.EntryWithStages, 0, len(entries))
	for _, e := range entries {
		if !e.Withdrawn {
			manned = append(manned, e)
		}
	}
	slices.SortFunc(manned, func(a, b repository.EntryWithStages) int { return a.Lane - b.Lane })
	names := make(map[int]string, len(manned))
	for _, e := range manned {
		sh, err := s.repo.GetShooter(ctx, e.ShooterID)
		if err != nil {
			return ControlRoom{}, mapRepoErr(err)
		}
		names[e.Lane] = shooterName(sh)
	}

	room := ControlRoom{CurrentStage: comp.CurrentStage, Lanes: make([]LaneStatus, 0, len(manned))}
	open, awaiting := false, false
	for i, e := range manned {
		st, _ := stageOf(e, comp.CurrentStage)
		scorer := ""
		switch {
		case st.ScorerName != nil:
			scorer = *st.ScorerName
		case len(manned) > 1:
			scorer = names[manned[(i+len(manned)-1)%len(manned)].Lane]
		}
		open = open || st.Status == repository.StatusOpen
		awaiting = awaiting || st.Status == repository.StatusAwaitingAcceptance
		room.Lanes = append(room.Lanes, LaneStatus{
			Lane: e.Lane, EntryID: e.ID, ShooterName: names[e.Lane], ScorerName: scorer,
			Status: st.Status, Disputed: st.DisputedAt != "", Disputes: disputes[e.ID],
			Paired: paired[e.Lane], Connected: s.presence.connected(laneKey{matchID, e.Lane}),
		})
	}
	active := MatchStatus(comp.Status) == StatusActive
	last, err := lastStage(comp.Course)
	if err != nil {
		return ControlRoom{}, err
	}
	room.CanStartNext = active && comp.CurrentStage < last && stageAccepted(entries, comp.CurrentStage)
	room.CanAcceptAll = active && !open && awaiting
	return room, nil
}

func (s *Service) AcceptAll(ctx context.Context, matchID int64) (int, error) {
	comp, err := s.repo.GetMatch(ctx, matchID)
	if err != nil {
		return 0, mapRepoErr(err)
	}
	if MatchStatus(comp.Status) != StatusActive {
		return 0, ErrConflict
	}
	entries, err := s.repo.ListEntries(ctx, matchID)
	if err != nil {
		return 0, err
	}
	username := ActorFrom(ctx).Username
	var items []repository.StageAcceptance
	for _, e := range entries {
		if e.Withdrawn {
			continue
		}
		st, ok := stageOf(e, comp.CurrentStage)
		if !ok {
			continue
		}
		if st.Status == repository.StatusOpen {
			return 0, ErrLanesOpen
		}
		if st.Status != repository.StatusAwaitingAcceptance {
			continue
		}
		stage, err := s.repo.GetStage(ctx, matchID, e.ID, comp.CurrentStage)
		if err != nil {
			return 0, mapRepoErr(err)
		}
		ev, err := s.userStageEvent(ctx, repository.EventAcceptedAll, username, matchID, e.ID, stage)
		if err != nil {
			return 0, err
		}
		items = append(items, repository.StageAcceptance{StageID: stage.ID, Event: ev})
	}
	if len(items) == 0 {
		return 0, nil
	}
	if err := s.repo.AcceptStages(ctx, username, items); err != nil {
		if errors.Is(err, repository.ErrConflict) {
			return 0, ErrStageNotAwaitingAcceptance
		}
		return 0, mapRepoErr(err)
	}
	for i, it := range items {
		c := Change{
			MatchID: matchID,
			Live:    LiveMatch,
			Stage:   &StageEvent{MatchID: matchID, EntryID: it.Event.EntryID, StageNumber: comp.CurrentStage},
		}
		if i == 0 {
			c.Audit = AuditEvent{
				Action:  "match.accept_all",
				Target:  idTarget("match", matchID),
				Details: "stage " + strconv.Itoa(comp.CurrentStage) + ", " + strconv.Itoa(len(items)) + " lanes",
			}
		}
		s.changes.Record(ctx, c)
	}
	s.finishIfComplete(ctx, matchID, nil)
	return len(items), nil
}

type StageHistoryFilter struct {
	Scorer  string
	Shooter string
	Stage   int
}

type ScorerDisputes struct {
	ScorerName string
	Locks      int
	Disputes   int
}

type StageHistory struct {
	Events  []repository.StageEvent
	Scorers []ScorerDisputes
}

func (s *Service) StageHistory(ctx context.Context, matchID int64, f StageHistoryFilter) (StageHistory, error) {
	if _, err := s.repo.GetMatch(ctx, matchID); err != nil {
		return StageHistory{}, mapRepoErr(err)
	}
	all, err := s.repo.ListStageEvents(ctx, matchID)
	if err != nil {
		return StageHistory{}, err
	}
	history := StageHistory{Events: []repository.StageEvent{}, Scorers: []ScorerDisputes{}}
	counts := map[string]*ScorerDisputes{}
	for _, ev := range all {
		if (f.Scorer != "" && ev.ScorerName != f.Scorer) ||
			(f.Shooter != "" && ev.ShooterName != f.Shooter) ||
			(f.Stage != 0 && ev.StageNumber != f.Stage) {
			continue
		}
		history.Events = append(history.Events, ev)
		if ev.ScorerName == "" {
			continue
		}
		c := counts[ev.ScorerName]
		if c == nil {
			c = &ScorerDisputes{ScorerName: ev.ScorerName}
			counts[ev.ScorerName] = c
		}
		switch ev.Event {
		case repository.EventLocked:
			c.Locks++
		case repository.EventDisputed:
			c.Disputes++
		}
	}
	for _, c := range counts {
		history.Scorers = append(history.Scorers, *c)
	}
	slices.SortFunc(history.Scorers, func(a, b ScorerDisputes) int { return strings.Compare(a.ScorerName, b.ScorerName) })
	return history, nil
}
